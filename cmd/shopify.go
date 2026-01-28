package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/knadh/listmonk/internal/bounce/webhooks"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

// ShopifyWebhook handles incoming Shopify order webhooks for purchase attribution.
func (app *App) ShopifyWebhook(c echo.Context) error {
	var (
		service      = "shopify"
		eventType    = "order"
		rawReq       []byte
		err          error
		processed    = false
		errorMsg     = ""
		responseCode = http.StatusOK
	)

	// Read the raw request body for HMAC verification and logging
	rawReq, err = io.ReadAll(c.Request().Body)
	if err != nil {
		errorMsg = fmt.Sprintf("error reading request body: %v", err)
		responseCode = http.StatusBadRequest
		app.logWebhook(service, eventType, c.Request().Header, rawReq, responseCode, "", processed, errorMsg)
		return echo.NewHTTPError(responseCode, errorMsg)
	}

	// Initialize Shopify webhook handler with both client_secret (for Partner apps)
	// and webhook_secret (for manual webhooks). It tries both during verification.
	shopifyHandler := webhooks.NewShopify(app.cfg.ShopifyClientSecret, app.cfg.ShopifyWebhookSecret)

	// Verify HMAC signature
	hmacHeader := c.Request().Header.Get("X-Shopify-Hmac-Sha256")
	if err := shopifyHandler.VerifyWebhook(hmacHeader, rawReq); err != nil {
		errorMsg = fmt.Sprintf("HMAC verification failed: %v", err)
		responseCode = http.StatusUnauthorized
		app.logWebhook(service, eventType, c.Request().Header, rawReq, responseCode, "", processed, errorMsg)
		return echo.NewHTTPError(responseCode, errorMsg)
	}

	// Parse the order data
	order, err := shopifyHandler.ProcessOrder(rawReq)
	if err != nil {
		errorMsg = fmt.Sprintf("error processing order: %v", err)
		responseCode = http.StatusBadRequest
		app.logWebhook(service, eventType, c.Request().Header, rawReq, responseCode, "", processed, errorMsg)
		return echo.NewHTTPError(responseCode, errorMsg)
	}

	// Attempt to attribute the purchase to a campaign
	if err := app.attributePurchase(order); err != nil {
		// Log the error but don't fail the webhook
		// This allows Shopify to consider the webhook successfully received
		app.log.Printf("error attributing purchase (order %d): %v", order.ID, err)
		errorMsg = fmt.Sprintf("attribution error: %v", err)
	} else {
		processed = true
	}

	// Store the order in real-time for segmentation
	if err := app.storeOrderFromWebhook(rawReq); err != nil {
		app.log.Printf("error storing order from webhook (order %d): %v", order.ID, err)
		// Don't fail the webhook, just log the error
	}

	// Log the webhook
	app.logWebhook(service, eventType, c.Request().Header, rawReq, responseCode, "", processed, errorMsg)

	return c.JSON(http.StatusOK, okResp{true})
}

// storeOrderFromWebhook stores a Shopify order from webhook data in the database.
func (app *App) storeOrderFromWebhook(rawJSON []byte) error {
	shopifyHandler := webhooks.NewShopify() // Only used for parsing, no HMAC verification

	// Parse the full order data
	order, err := shopifyHandler.ProcessOrderFull(rawJSON)
	if err != nil {
		return fmt.Errorf("error parsing order: %v", err)
	}

	// Find subscriber by email
	var sub models.Subscriber
	err = app.queries.GetSubscriberByEmail.Get(&sub, order.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			// No subscriber found, skip storing order
			app.log.Printf("storeOrderFromWebhook: no subscriber found for email %s, skipping order storage", order.Email)
			return nil
		}
		return fmt.Errorf("error finding subscriber: %v", err)
	}

	// Parse order timestamps
	createdAt, _ := time.Parse(time.RFC3339, order.CreatedAt)
	processedAt, _ := time.Parse(time.RFC3339, order.ProcessedAt)

	// Parse price fields
	totalPrice, _ := strconv.ParseFloat(order.TotalPrice, 64)
	subtotalPrice, _ := strconv.ParseFloat(order.SubtotalPrice, 64)
	totalTax, _ := strconv.ParseFloat(order.TotalTax, 64)
	totalDiscounts, _ := strconv.ParseFloat(order.TotalDiscounts, 64)

	// Parse tags into array
	var tags []string
	if order.Tags != "" {
		for _, tag := range strings.Split(order.Tags, ",") {
			tags = append(tags, strings.TrimSpace(tag))
		}
	}

	// Upsert the order
	var orderID int
	err = app.queries.UpsertShopifyOrder.QueryRow(
		sub.ID,
		fmt.Sprintf("gid://shopify/Order/%d", order.ID), // Convert to global ID format
		fmt.Sprintf("%d", order.OrderNumber),
		order.Name,
		createdAt,
		processedAt,
		totalPrice,
		subtotalPrice,
		totalTax,
		totalDiscounts,
		order.Currency,
		order.FinancialStatus,
		order.FulfillmentStatus,
		pq.Array(tags),
		order.Note,
	).Scan(&orderID)

	if err != nil {
		return fmt.Errorf("error upserting order: %v", err)
	}

	// Delete existing line items and insert new ones
	_, err = app.queries.DeleteShopifyOrderLineItems.Exec(orderID)
	if err != nil {
		return fmt.Errorf("error deleting line items: %v", err)
	}

	// Insert line items
	for _, item := range order.LineItems {
		price, _ := strconv.ParseFloat(item.Price, 64)
		discount, _ := strconv.ParseFloat(item.TotalDiscount, 64)

		// Convert properties to JSON
		propsJSON, _ := json.Marshal(item.Properties)

		_, err = app.queries.InsertShopifyLineItem.Exec(
			orderID,
			fmt.Sprintf("gid://shopify/LineItem/%d", item.ID),
			fmt.Sprintf("gid://shopify/Product/%d", item.ProductID),
			item.Title,
			fmt.Sprintf("gid://shopify/ProductVariant/%d", item.VariantID),
			item.VariantTitle,
			item.SKU,
			item.Quantity,
			price,
			discount,
			item.ProductType,
			item.Vendor,
			propsJSON,
		)
		if err != nil {
			app.log.Printf("storeOrderFromWebhook: error inserting line item %d: %v", item.ID, err)
		}
	}

	app.log.Printf("storeOrderFromWebhook: stored order %s with %d line items for subscriber %d",
		order.Name, len(order.LineItems), sub.ID)

	// Sync customer data from Shopify to update subscriber attribs
	// This ensures name, address, total_spent, orders_count are up-to-date
	go func() {
		if err := app.syncCustomerAfterOrder(order.Email, sub.ID); err != nil {
			app.log.Printf("storeOrderFromWebhook: error syncing customer after order: %v", err)
		}
	}()

	return nil
}

// attributePurchase attempts to attribute a purchase to the most recent campaign sent to the subscriber.
func (app *App) attributePurchase(order *webhooks.ShopifyOrder) error {
	// Find subscriber by email
	var sub models.Subscriber
	var subscriberID interface{}
	err := app.queries.GetSubscriberByEmail.Get(&sub, order.Email)

	hasSubscriber := true
	if err != nil {
		if err == sql.ErrNoRows {
			hasSubscriber = false
			subscriberID = nil
			app.log.Printf("no subscriber found with email: %s, checking UTM parameters", order.Email)
		} else {
			return fmt.Errorf("error finding subscriber: %v", err)
		}
	} else {
		subscriberID = sub.ID
	}

	var campaignID interface{}
	attributedVia := "is_subscriber"
	confidence := "high"

	// If no subscriber, check UTM parameters
	if !hasSubscriber {
		// Check if landing_site contains utm_source=listmonk
		if order.LandingSite != "" && containsUTMSource(order.LandingSite, "listmonk") {
			app.log.Printf("found utm_source=listmonk in landing_site, attributing to running campaign")

			// Find most recent running campaign
			var runningCampaign struct {
				CampaignID int `db:"campaign_id"`
			}
			err = app.db.Get(&runningCampaign, `
				SELECT id as campaign_id
				FROM campaigns
				WHERE status = 'running'
				ORDER BY started_at DESC
				LIMIT 1
			`)

			if err != nil {
				if err == sql.ErrNoRows {
					app.log.Printf("no running campaigns found for UTM attribution")
					campaignID = nil
				} else {
					return fmt.Errorf("error finding running campaign for UTM: %v", err)
				}
			} else {
				campaignID = runningCampaign.CampaignID
				attributedVia = "utm_listmonk"
				confidence = "medium"
				app.log.Printf("attributed non-subscriber purchase to running campaign %d via UTM", campaignID)
			}
		} else {
			return fmt.Errorf("no subscriber found with email: %s and no listmonk UTM parameters", order.Email)
		}
	} else {
		// Subscriber exists - use original three-tier logic
		var recentCampaign struct {
			CampaignID int `db:"campaign_id"`
		}

		err = app.db.Get(&recentCampaign, `
			SELECT campaign_id
			FROM azure_delivery_events
			WHERE subscriber_id = $1
			  AND status = 'Delivered'
			ORDER BY event_timestamp DESC
			LIMIT 1
		`, sub.ID)

		if err != nil {
			if err == sql.ErrNoRows {
				// No campaigns delivered to this subscriber, try to find most recent running campaign for their lists
				app.log.Printf("no delivered campaigns found for subscriber %d, looking for most recent running campaign", sub.ID)

				err = app.db.Get(&recentCampaign, `
					SELECT c.id as campaign_id
					FROM campaigns c
					JOIN campaign_lists cl ON cl.campaign_id = c.id
					JOIN subscriber_lists sl ON sl.list_id = cl.list_id
					WHERE sl.subscriber_id = $1
					  AND c.status = 'running'
					ORDER BY c.started_at DESC
					LIMIT 1
				`, sub.ID)

				if err != nil {
					if err == sql.ErrNoRows {
						app.log.Printf("no running campaigns found for subscriber %d's lists, attributing without campaign", sub.ID)
						campaignID = nil
					} else {
						return fmt.Errorf("error finding running campaigns: %v", err)
					}
				} else {
					campaignID = recentCampaign.CampaignID
					app.log.Printf("no delivered campaigns, attributed to most recent running campaign %d for subscriber %d", campaignID, sub.ID)
				}
			} else {
				return fmt.Errorf("error finding recent campaigns: %v", err)
			}
		} else {
			campaignID = recentCampaign.CampaignID
		}
	}

	// Parse total price
	totalPrice, err := strconv.ParseFloat(order.TotalPrice, 64)
	if err != nil {
		app.log.Printf("error parsing total price '%s': %v", order.TotalPrice, err)
		totalPrice = 0
	}

	// Create purchase attribution record
	var attribution models.PurchaseAttribution
	err = app.queries.InsertPurchaseAttribution.Get(&attribution,
		campaignID,
		subscriberID,
		fmt.Sprintf("%d", order.ID),
		order.OrderNumber,
		order.Email,
		totalPrice,
		order.Currency,
		attributedVia,
		confidence,
		order.RawJSON,
	)

	if err != nil {
		return fmt.Errorf("error inserting purchase attribution: %v", err)
	}

	// Log attribution
	if subscriberID != nil {
		if campaignID != nil {
			app.log.Printf("attributed purchase (order %d, $%.2f %s) to campaign %v for subscriber %v via %s",
				order.ID, totalPrice, order.Currency, campaignID, subscriberID, attributedVia)
		} else {
			app.log.Printf("attributed purchase (order %d, $%.2f %s) to subscriber %v (no recent campaign)",
				order.ID, totalPrice, order.Currency, subscriberID)
		}
	} else {
		app.log.Printf("attributed non-subscriber purchase (order %d, $%.2f %s) to campaign %v via %s",
			order.ID, totalPrice, order.Currency, campaignID, attributedVia)
	}

	return nil
}

// containsUTMSource checks if a URL or query string contains a specific utm_source parameter.
func containsUTMSource(landingSite, expectedSource string) bool {
	// Simple check for utm_source parameter
	// landingSite format: "/?utm_source=listmonk&utm_medium=email&utm_campaign=50OFF1"
	searchStr := "utm_source=" + expectedSource
	return len(landingSite) > 0 && (
		// Check for utm_source at various positions
		landingSite == "/?"+searchStr ||
		strings.Contains(landingSite, "?"+searchStr+"&") ||
		strings.Contains(landingSite, "&"+searchStr+"&") ||
		strings.HasSuffix(landingSite, "&"+searchStr) ||
		strings.HasSuffix(landingSite, "?"+searchStr))
}

// GetCampaignPurchaseStats returns purchase statistics for a campaign.
func (app *App) GetCampaignPurchaseStats(c echo.Context) error {
	var (
		campaignID, _ = strconv.Atoi(c.Param("id"))
		stats         models.CampaignPurchaseStats
	)

	if campaignID < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	// Get purchase stats for the campaign
	err := app.queries.GetCampaignPurchaseStats.Get(&stats, campaignID)
	if err != nil {
		if err == sql.ErrNoRows {
			// No purchases found, return zero stats
			stats = models.CampaignPurchaseStats{
				TotalPurchases: 0,
				TotalRevenue:   0,
				AvgOrderValue:  0,
				Currency:       "USD",
			}
		} else {
			app.log.Printf("error fetching purchase stats for campaign %d: %v", campaignID, err)
			return echo.NewHTTPError(http.StatusInternalServerError, app.i18n.T("globals.messages.errorFetching"))
		}
	}

	return c.JSON(http.StatusOK, okResp{stats})
}

// GetCampaignsPerformanceSummary returns aggregate performance metrics for all campaigns in the last 30 days.
func (app *App) GetCampaignsPerformanceSummary(c echo.Context) error {
	var summary models.CampaignsPerformanceSummary

	// Parse and validate the "days" query parameter (defaults to 1 for "Today")
	days := 1
	if daysParam := c.QueryParam("days"); daysParam != "" {
		parsedDays, err := strconv.Atoi(daysParam)
		if err != nil || parsedDays < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid 'days' parameter. Must be a positive integer.")
		}
		days = parsedDays
	}

	// Get aggregate performance stats for the specified timeframe
	err := app.queries.GetCampaignsPerformanceSummary.Get(&summary, days)
	if err != nil {
		if err == sql.ErrNoRows {
			// No campaigns found, return zero stats
			summary = models.CampaignsPerformanceSummary{
				AvgOpenRate:  0,
				AvgClickRate: 0,
				TotalSent:    0,
				TotalOrders:  0,
				TotalRevenue: 0,
				OrderRate:    0,
				ErrorRate:    0,
			}
		} else {
			app.log.Printf("error fetching campaigns performance summary: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, app.i18n.T("globals.messages.errorFetching"))
		}
	}

	return c.JSON(http.StatusOK, okResp{summary})
}
