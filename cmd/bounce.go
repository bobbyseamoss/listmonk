package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// GetBounce handles retrieval of a specific bounce record by ID.
func (a *App) GetBounce(c echo.Context) error {
	// Fetch one bounce from the DB.
	id := getID(c)
	out, err := a.core.GetBounce(id)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GetBounces handles retrieval of bounce records.
func (a *App) GetBounces(c echo.Context) error {
	var (
		campID, _ = strconv.Atoi(c.QueryParam("campaign_id"))
		source    = c.FormValue("source")
		orderBy   = c.FormValue("order_by")
		order     = c.FormValue("order")

		pg = a.pg.NewFromURL(c.Request().URL.Query())
	)

	// Query and fetch bounces from the DB.
	res, total, err := a.core.QueryBounces(campID, 0, source, orderBy, order, pg.Offset, pg.Limit)
	if err != nil {
		return err
	}

	// No results.
	if len(res) == 0 {
		return c.JSON(http.StatusOK, okResp{models.PageResults{Results: []models.Bounce{}}})
	}

	out := models.PageResults{
		Results: res,
		Total:   total,
		Page:    pg.Page,
		PerPage: pg.PerPage,
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GetSubscriberBounces retrieves a subscriber's bounce records.
func (a *App) GetSubscriberBounces(c echo.Context) error {
	// Query and fetch bounces from the DB.
	subID := getID(c)
	out, _, err := a.core.QueryBounces(0, subID, "", "", "", 0, 1000)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// DeleteBounces handles bounce deletion of a list.
func (a *App) DeleteBounces(c echo.Context) error {
	all, _ := strconv.ParseBool(c.QueryParam("all"))

	var ids []int
	if !all {
		// There are multiple IDs in the query string.
		res, err := parseStringIDs(c.Request().URL.Query()["id"])
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidID", "error", err.Error()))
		}
		if len(res) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidID"))
		}

		ids = res
	}

	// Delete bounces from the DB.
	if err := a.core.DeleteBounces(ids, all); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// DeleteBounce handles bounce deletion of a single bounce record.
func (a *App) DeleteBounce(c echo.Context) error {
	// Delete bounces from the DB.
	id := getID(c)
	if err := a.core.DeleteBounces([]int{id}, false); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// BlocklistBouncedSubscribers handles blocklisting of all bounced subscribers.
func (a *App) BlocklistBouncedSubscribers(c echo.Context) error {
	if err := a.core.BlocklistBouncedSubscribers(); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// BounceWebhook renders the HTML preview of a template.
func (a *App) BounceWebhook(c echo.Context) error {
	// Read the request body instead of using c.Bind() to read to save the entire raw request as meta.
	rawReq, err := io.ReadAll(c.Request().Body)
	if err != nil {
		a.log.Printf("error reading ses notification body: %v", err)
		a.logWebhook("unknown", "", c.Request().Header, []byte{}, http.StatusBadRequest, "", false, err.Error())
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.internalError"))
	}

	var (
		service       = c.Param("service")
		webhookType   = service
		eventType     = ""
		processed     = false
		errorMsg      = ""
		responseStatus = http.StatusOK

		bounces []models.Bounce
	)

	// Default webhook type for native webhooks
	if webhookType == "" {
		webhookType = "native"
	}

	// Defer logging the webhook after processing
	defer func() {
		a.logWebhook(webhookType, eventType, c.Request().Header, rawReq, responseStatus, "", processed, errorMsg)
	}()
	switch true {
	// Native internal webhook.
	case service == "":
		var b models.Bounce
		if err := json.Unmarshal(rawReq, &b); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidData")+":"+err.Error())
		}

		if bv, err := a.validateBounceFields(b); err != nil {
			return err
		} else {
			b = bv
		}

		if len(b.Meta) == 0 {
			b.Meta = json.RawMessage("{}")
		}

		if b.CreatedAt.Year() == 0 {
			b.CreatedAt = time.Now()
		}

		bounces = append(bounces, b)

	// Amazon SES.
	case service == "ses" && a.cfg.BounceSESEnabled:
		switch c.Request().Header.Get("X-Amz-Sns-Message-Type") {
		// SNS webhook registration confirmation. Only after these are processed will the endpoint
		// start getting bounce notifications.
		case "SubscriptionConfirmation", "UnsubscribeConfirmation":
			if err := a.bounce.SES.ProcessSubscription(rawReq); err != nil {
				a.log.Printf("error processing SNS (SES) subscription: %v", err)
				return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
			}

		// Bounce notification.
		case "Notification":
			b, err := a.bounce.SES.ProcessBounce(rawReq)
			if err != nil {
				a.log.Printf("error processing SES notification: %v", err)
				return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
			}
			bounces = append(bounces, b)

		default:
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
		}

	// SendGrid.
	case service == "sendgrid" && a.cfg.BounceSendgridEnabled:
		var (
			sig = c.Request().Header.Get("X-Twilio-Email-Event-Webhook-Signature")
			ts  = c.Request().Header.Get("X-Twilio-Email-Event-Webhook-Timestamp")
		)

		// Sendgrid sends multiple bounces.
		bs, err := a.bounce.Sendgrid.ProcessBounce(sig, ts, rawReq)
		if err != nil {
			a.log.Printf("error processing sendgrid notification: %v", err)
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
		}
		bounces = append(bounces, bs...)

	// Postmark.
	case service == "postmark" && a.cfg.BouncePostmarkEnabled:
		bs, err := a.bounce.Postmark.ProcessBounce(rawReq, c)
		if err != nil {
			a.log.Printf("error processing postmark notification: %v", err)
			if _, ok := err.(*echo.HTTPError); ok {
				return err
			}

			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
		}
		bounces = append(bounces, bs...)

	// ForwardEmail.
	case service == "forwardemail" && a.cfg.BounceForwardemailEnabled:
		var (
			sig = c.Request().Header.Get("X-Webhook-Signature")
		)

		bs, err := a.bounce.Forwardemail.ProcessBounce(sig, rawReq)
		if err != nil {
			a.log.Printf("error processing forwardemail notification: %v", err)
			if _, ok := err.(*echo.HTTPError); ok {
				return err
			}

			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
		}
		bounces = append(bounces, bs...)

	// Azure Event Grid.
	case service == "azure" && a.cfg.BounceAzureEnabled:
		var events []struct {
			EventType string                 `json:"eventType"`
			Data      map[string]interface{} `json:"data"`
		}

		if err := json.Unmarshal(rawReq, &events); err != nil {
			a.log.Printf("error unmarshalling Azure Event Grid notification: %v", err)
			errorMsg = err.Error()
			responseStatus = http.StatusBadRequest
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
		}

		for _, event := range events {
			// Set event type for logging (use first event's type)
			if eventType == "" {
				eventType = event.EventType
			}

			switch event.EventType {
			case "Microsoft.EventGrid.SubscriptionValidationEvent":
				// Handle subscription validation
				resp, err := a.bounce.Azure.ProcessValidation(rawReq)
				if err != nil {
					a.log.Printf("error processing Azure validation: %v", err)
					return echo.NewHTTPError(http.StatusBadRequest, a.i18n.T("globals.messages.invalidData"))
				}
				return c.JSON(http.StatusOK, resp)

			case "Microsoft.Communication.EmailDeliveryReportReceived":
				// Handle delivery status events
				b, shouldRecord, err := a.bounce.Azure.ProcessDeliveryEvent(event.Data)
				if err != nil {
					a.log.Printf("error processing Azure delivery event: %v", err)
					continue
				}

				// Debug: Log the entire event data to understand what Azure sends
				if eventDataJSON, err := json.Marshal(event.Data); err == nil {
					a.log.Printf("DEBUG: Azure delivery event data: %s", string(eventDataJSON))
				}

				// Extract delivery event data for storage
				messageID, _ := event.Data["messageId"].(string)
				internetMessageID, _ := event.Data["internetMessageId"].(string)
				status, _ := event.Data["status"].(string)
				deliveryAttemptTimeStamp, _ := event.Data["deliveryAttemptTimeStamp"].(string)

				// Parse timestamp
				eventTime := time.Now()
				if deliveryAttemptTimeStamp != "" {
					if t, err := time.Parse(time.RFC3339, deliveryAttemptTimeStamp); err == nil {
						eventTime = t
					}
				}

				// Try to correlate event back to campaign/subscriber
				// Strategy 1: Look up by Message-ID in tracking table
				// Strategy 2: Look up by recipient email if Message-ID fails
				var campaignID, subscriberID int
				recipient, _ := event.Data["recipient"].(string)

				// Try Message-ID lookup first
				err = a.db.QueryRow(`
					SELECT campaign_id, subscriber_id
					FROM azure_message_tracking
					WHERE azure_message_id = $1
				`, messageID).Scan(&campaignID, &subscriberID)

				// If Message-ID lookup fails, try recipient email
				if err != nil && recipient != "" {
					a.log.Printf("Message-ID lookup failed for %s, trying recipient email %s", messageID, recipient)
					err = a.db.QueryRow(`
						SELECT c.id, s.id
						FROM campaigns c
						CROSS JOIN subscribers s
						WHERE s.email = $1
						AND c.status IN ('running', 'finished', 'paused', 'cancelled')
						ORDER BY c.updated_at DESC
						LIMIT 1
					`, recipient).Scan(&campaignID, &subscriberID)

					if err == nil {
						a.log.Printf("Found campaign %d, subscriber %d via recipient email fallback", campaignID, subscriberID)
					}
				}

				if err == nil && messageID != "" {
					// Extract status reason and delivery details
					statusReason := ""
					deliveryDetailsJSON := ""

					if deliveryDetails, ok := event.Data["deliveryStatusDetails"].(map[string]interface{}); ok {
						if statusMessage, ok := deliveryDetails["statusMessage"].(string); ok {
							statusReason = statusMessage
						}
						// Store full delivery details as JSON
						if detailsBytes, err := json.Marshal(deliveryDetails); err == nil {
							deliveryDetailsJSON = string(detailsBytes)
						}
					}

					// Store delivery event in azure_delivery_events table
					_, err := a.db.Exec(`
						INSERT INTO azure_delivery_events
							(azure_message_id, campaign_id, subscriber_id, status, status_reason, delivery_status_details, event_timestamp)
						VALUES ($1, $2, $3, $4, $5, $6, $7)
					`, messageID, campaignID, subscriberID, status, statusReason, deliveryDetailsJSON, eventTime)

					if err != nil {
						a.log.Printf("error storing Azure delivery event: %v", err)
					}

					// Update azure_message_tracking with internet_message_id for proper engagement attribution
					// The internetMessageId is consistent across delivery and engagement events
					if internetMessageID != "" {
						_, err = a.db.Exec(`
							UPDATE azure_message_tracking
							SET internet_message_id = $1
							WHERE azure_message_id = $2
						`, internetMessageID, messageID)

						if err != nil {
							a.log.Printf("error updating azure_message_tracking with internet_message_id: %v", err)
						}
					}
				} else if err != nil {
					a.log.Printf("error looking up tracking info for Azure message %s: %v", messageID, err)
				}

				if !shouldRecord {
					// Delivered successfully, no bounce to record
					continue
				}

				// Try to get campaign UUID from X-headers first (if Azure preserves them)
				// Check for both header formats: with and without "X-" prefix
				campaignUUID := ""
				for _, key := range []string{"XListmonkCampaign", "X-Listmonk-Campaign", "xListmonkCampaign"} {
					if val, ok := event.Data[key].(string); ok && val != "" {
						campaignUUID = val
						a.log.Printf("Azure Event Grid: found campaign UUID in header %s", key)
						break
					}
				}

				// Fallback: Look up campaign UUID
				// If we already have campaignID from recipient email fallback, use that directly
				if campaignUUID == "" && campaignID > 0 {
					err = a.db.QueryRow(`SELECT uuid FROM campaigns WHERE id = $1`, campaignID).Scan(&campaignUUID)
					if err != nil {
						a.log.Printf("error looking up campaign UUID for campaign %d: %v", campaignID, err)
					}
				} else if campaignUUID == "" && messageID != "" {
					// Try azure_message_tracking table as secondary fallback
					err = a.db.QueryRow(`
						SELECT c.uuid
						FROM azure_message_tracking amt
						JOIN campaigns c ON amt.campaign_id = c.id
						WHERE amt.azure_message_id = $1
					`, messageID).Scan(&campaignUUID)

					if err != nil {
						a.log.Printf("error looking up campaign for Azure message %s: %v", messageID, err)
					}
				}

				b.CampaignUUID = campaignUUID

				// CRITICAL: Set subscriber ID if we found it via fallback
				// The bounces table requires subscriber_id, not just email
				if subscriberID > 0 {
					b.SubscriberID = subscriberID
				}

				// Debug: Log bounce details before adding
				if campaignUUID != "" {
					a.log.Printf("Adding bounce to queue: email=%s, subscriberID=%d, type=%s, campaign=%s", b.Email, b.SubscriberID, b.Type, campaignUUID)
				} else {
					a.log.Printf("WARNING: Adding bounce with empty campaign UUID: email=%s, subscriberID=%d, type=%s", b.Email, b.SubscriberID, b.Type)
				}

				bounces = append(bounces, b)

			case "Microsoft.Communication.EmailEngagementTrackingReportReceived":
				// Handle engagement tracking events
				engagement, err := a.bounce.Azure.ProcessEngagementEvent(event.Data)
				if err != nil {
					a.log.Printf("error processing Azure engagement event: %v", err)
					continue
				}

				// Debug: Log the entire event data
				if eventDataJSON, err := json.Marshal(event.Data); err == nil {
					a.log.Printf("DEBUG: Azure engagement event data: %s", string(eventDataJSON))
				}

				var campaignID, subscriberID int
				recipient, _ := event.Data["recipient"].(string)

				// Try to get campaign/subscriber IDs from X-headers first (if Azure preserves them)
				campaignUUID := ""
				subscriberUUID := ""

				for _, key := range []string{"XListmonkCampaign", "X-Listmonk-Campaign", "xListmonkCampaign"} {
					if val, ok := event.Data[key].(string); ok && val != "" {
						campaignUUID = val
						break
					}
				}

				for _, key := range []string{"XListmonkSubscriber", "X-Listmonk-Subscriber", "xListmonkSubscriber"} {
					if val, ok := event.Data[key].(string); ok && val != "" {
						subscriberUUID = val
						break
					}
				}

				// If we have both UUIDs from headers, look them up directly
				if campaignUUID != "" && subscriberUUID != "" {
					err = a.db.QueryRow(`
						SELECT c.id, s.id
						FROM campaigns c, subscribers s
						WHERE c.uuid = $1 AND s.uuid = $2
					`, campaignUUID, subscriberUUID).Scan(&campaignID, &subscriberID)

					if err != nil {
						a.log.Printf("error looking up campaign/subscriber by UUIDs: %v", err)
						continue
					}
				} else {
					// Fallback: Look up from azure_message_tracking table using internet_message_id
					// This is the KEY FIX: Azure uses different message IDs for delivery vs engagement events
					// but internet_message_id (email Message-ID header) is consistent across all event types
					err = a.db.QueryRow(`
						SELECT campaign_id, subscriber_id
						FROM azure_message_tracking
						WHERE internet_message_id = $1
					`, engagement.InternetMessageID).Scan(&campaignID, &subscriberID)

					// If Message-ID lookup fails, try recipient email
					if err != nil && recipient != "" {
						a.log.Printf("Message-ID lookup failed for engagement %s, trying recipient email %s", engagement.MessageID, recipient)
						err = a.db.QueryRow(`
							SELECT c.id, s.id
							FROM campaigns c
							CROSS JOIN subscribers s
							WHERE s.email = $1
							AND c.status IN ('running', 'finished', 'paused', 'cancelled')
							ORDER BY c.updated_at DESC
							LIMIT 1
						`, recipient).Scan(&campaignID, &subscriberID)

						if err == nil {
							a.log.Printf("Found campaign %d, subscriber %d via recipient email fallback for engagement", campaignID, subscriberID)
						}
					}

					if err != nil {
						a.log.Printf("error looking up tracking info for Azure message %s: %v", engagement.MessageID, err)
						continue
					}
				}

				// Parse timestamp
				actionTime := time.Now()
				if t, err := time.Parse(time.RFC3339, engagement.UserActionTimeStamp); err == nil {
					actionTime = t
				}

				// Store engagement event in azure_engagement_events table
				_, err = a.db.Exec(`
					INSERT INTO azure_engagement_events
						(azure_message_id, internet_message_id, campaign_id, subscriber_id, engagement_type, engagement_context, user_agent, event_timestamp)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
				`, engagement.MessageID, engagement.InternetMessageID, campaignID, subscriberID, engagement.EngagementType, engagement.EngagementContext, engagement.UserAgent, actionTime)

				if err != nil {
					a.log.Printf("error storing Azure engagement event: %v", err)
				}

				switch engagement.EngagementType {
				case "view":
					// Insert into campaign_views (with de-duplication to prevent double-counting
					// when both native {{TrackView}} and Azure engagement tracking are enabled)
					// Only insert if this exact view doesn't already exist (same campaign, subscriber, and timestamp within 5 seconds)
					_, err := a.db.Exec(`
						INSERT INTO campaign_views (campaign_id, subscriber_id, created_at)
						SELECT $1, $2, $3
						WHERE NOT EXISTS (
							SELECT 1 FROM campaign_views
							WHERE campaign_id = $1
							AND subscriber_id = $2
							AND ABS(EXTRACT(EPOCH FROM (created_at - $3))) <= 5
						)
					`, campaignID, subscriberID, actionTime)
					if err != nil {
						a.log.Printf("error recording Azure view: %v", err)
					}

				case "click":
					// Insert into link_clicks (requires finding/creating link)
					if engagement.EngagementContext == "" {
						a.log.Printf("no engagement context (URL) in Azure click event")
						continue
					}

					// Find or create link
					var linkID int
					err := a.db.QueryRow(`
						INSERT INTO links (uuid, url, created_at)
						VALUES (gen_random_uuid(), $1, NOW())
						ON CONFLICT (url) DO UPDATE SET url = EXCLUDED.url
						RETURNING id
					`, engagement.EngagementContext).Scan(&linkID)

					if err != nil {
						a.log.Printf("error finding/creating link: %v", err)
						continue
					}

					// Insert click record (with de-duplication to prevent double-counting
					// when both native {{TrackLink}} and Azure engagement tracking are enabled)
					// Only insert if this exact click doesn't already exist (same campaign, subscriber, link, and timestamp within 5 seconds)
					_, err = a.db.Exec(`
						INSERT INTO link_clicks (campaign_id, subscriber_id, link_id, created_at)
						SELECT $1, $2, $3, $4
						WHERE NOT EXISTS (
							SELECT 1 FROM link_clicks
							WHERE campaign_id = $1
							AND subscriber_id = $2
							AND link_id = $3
							AND ABS(EXTRACT(EPOCH FROM (created_at - $4))) <= 5
						)
					`, campaignID, subscriberID, linkID, actionTime)

					if err != nil {
						a.log.Printf("error recording Azure click: %v", err)
					}
				}
			}
		}

	default:
		return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("bounces.unknownService"))
	}

	// Insert bounces into the DB.
	if len(bounces) > 0 {
		a.log.Printf("Processing %d bounce(s) from webhook", len(bounces))
	}
	for _, b := range bounces {
		a.log.Printf("Recording bounce: email=%s, type=%s, source=%s, campaign=%s", b.Email, b.Type, b.Source, b.CampaignUUID)
		if err := a.bounce.Record(b); err != nil {
			a.log.Printf("error recording bounce for %s: %v", b.Email, err)
			errorMsg = fmt.Sprintf("error recording bounce: %v", err)
		} else {
			a.log.Printf("Successfully recorded bounce for %s", b.Email)
		}
	}

	// Mark as successfully processed
	processed = true

	return c.JSON(http.StatusOK, okResp{true})
}

func (a *App) validateBounceFields(b models.Bounce) (models.Bounce, error) {
	if b.Email == "" && b.SubscriberUUID == "" {
		return b, echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "email / subscriber_uuid"))
	}

	if b.SubscriberUUID != "" && !reUUID.MatchString(b.SubscriberUUID) {
		return b, echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "subscriber_uuid"))
	}

	if b.Email != "" {
		em, err := a.importer.SanitizeEmail(b.Email)
		if err != nil {
			return b, echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		b.Email = em
	}

	if b.Type != models.BounceTypeHard && b.Type != models.BounceTypeSoft && b.Type != models.BounceTypeComplaint {
		return b, echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidFields", "name", "type"))
	}

	return b, nil
}
