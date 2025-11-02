package main

import (
	"net/http"
	"strconv"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// GetCampaignAzureDeliveryEvents retrieves Azure delivery events for a specific campaign.
func (a *App) GetCampaignAzureDeliveryEvents(c echo.Context) error {
	var (
		campID  = getID(c)
		pg      = a.pg.NewFromURL(c.Request().URL.Query())
		status  = c.QueryParam("status")
		orderBy = c.FormValue("order_by")
		order   = c.FormValue("order")
	)

	// Default ordering
	if orderBy == "" {
		orderBy = "event_timestamp"
	}
	if order == "" {
		order = "DESC"
	}

	// Build query
	query := `
		SELECT id, azure_message_id, campaign_id, subscriber_id, status,
		       status_reason, delivery_status_details, event_timestamp, created_at
		FROM azure_delivery_events
		WHERE campaign_id = $1
	`
	countQuery := `
		SELECT COUNT(*) FROM azure_delivery_events WHERE campaign_id = $1
	`

	args := []interface{}{campID}
	argCount := 1

	// Filter by status if provided
	if status != "" {
		argCount++
		query += ` AND status = $` + strconv.Itoa(argCount)
		countQuery += ` AND status = $2`
		args = append(args, status)
	}

	// Add ordering and pagination
	query += ` ORDER BY ` + orderBy + ` ` + order + ` LIMIT $` + strconv.Itoa(argCount+1) + ` OFFSET $` + strconv.Itoa(argCount+2)
	args = append(args, pg.Limit, pg.Offset)

	var events []models.AzureDeliveryEvent
	if err := a.db.Select(&events, query, args...); err != nil {
		a.log.Printf("error querying Azure delivery events: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.Ts("globals.messages.errorFetching", "name", "delivery events"))
	}

	// Get total count
	var total int
	countArgs := args[:len(args)-2] // Remove LIMIT and OFFSET
	if err := a.db.Get(&total, countQuery, countArgs...); err != nil {
		a.log.Printf("error counting Azure delivery events: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.Ts("globals.messages.errorFetching", "name", "delivery events"))
	}

	if len(events) == 0 {
		return c.JSON(http.StatusOK, okResp{models.PageResults{Results: []models.AzureDeliveryEvent{}}})
	}

	out := models.PageResults{
		Results: events,
		Total:   total,
		Page:    pg.Page,
		PerPage: pg.PerPage,
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GetSubscriberAzureDeliveryEvents retrieves Azure delivery events for a specific subscriber.
func (a *App) GetSubscriberAzureDeliveryEvents(c echo.Context) error {
	var (
		subID = getID(c)
		pg    = a.pg.NewFromURL(c.Request().URL.Query())
	)

	query := `
		SELECT id, azure_message_id, campaign_id, subscriber_id, status,
		       status_reason, delivery_status_details, event_timestamp, created_at
		FROM azure_delivery_events
		WHERE subscriber_id = $1
		ORDER BY event_timestamp DESC
		LIMIT $2 OFFSET $3
	`

	var events []models.AzureDeliveryEvent
	if err := a.db.Select(&events, query, subID, pg.Limit, pg.Offset); err != nil {
		a.log.Printf("error querying Azure delivery events for subscriber: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.Ts("globals.messages.errorFetching", "name", "delivery events"))
	}

	if len(events) == 0 {
		return c.JSON(http.StatusOK, okResp{[]models.AzureDeliveryEvent{}})
	}

	return c.JSON(http.StatusOK, okResp{events})
}

// GetCampaignAzureEngagementEvents retrieves Azure engagement events for a specific campaign.
func (a *App) GetCampaignAzureEngagementEvents(c echo.Context) error {
	var (
		campID         = getID(c)
		pg             = a.pg.NewFromURL(c.Request().URL.Query())
		engagementType = c.QueryParam("type")
		orderBy        = c.FormValue("order_by")
		order          = c.FormValue("order")
	)

	// Default ordering
	if orderBy == "" {
		orderBy = "event_timestamp"
	}
	if order == "" {
		order = "DESC"
	}

	// Build query
	query := `
		SELECT id, azure_message_id, campaign_id, subscriber_id, engagement_type,
		       engagement_context, user_agent, event_timestamp, created_at
		FROM azure_engagement_events
		WHERE campaign_id = $1
	`
	countQuery := `
		SELECT COUNT(*) FROM azure_engagement_events WHERE campaign_id = $1
	`

	args := []interface{}{campID}
	argCount := 1

	// Filter by engagement type if provided
	if engagementType != "" {
		argCount++
		query += ` AND engagement_type = $` + strconv.Itoa(argCount)
		countQuery += ` AND engagement_type = $2`
		args = append(args, engagementType)
	}

	// Add ordering and pagination
	query += ` ORDER BY ` + orderBy + ` ` + order + ` LIMIT $` + strconv.Itoa(argCount+1) + ` OFFSET $` + strconv.Itoa(argCount+2)
	args = append(args, pg.Limit, pg.Offset)

	var events []models.AzureEngagementEvent
	if err := a.db.Select(&events, query, args...); err != nil {
		a.log.Printf("error querying Azure engagement events: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.Ts("globals.messages.errorFetching", "name", "engagement events"))
	}

	// Get total count
	var total int
	countArgs := args[:len(args)-2] // Remove LIMIT and OFFSET
	if err := a.db.Get(&total, countQuery, countArgs...); err != nil {
		a.log.Printf("error counting Azure engagement events: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.Ts("globals.messages.errorFetching", "name", "engagement events"))
	}

	if len(events) == 0 {
		return c.JSON(http.StatusOK, okResp{models.PageResults{Results: []models.AzureEngagementEvent{}}})
	}

	out := models.PageResults{
		Results: events,
		Total:   total,
		Page:    pg.Page,
		PerPage: pg.PerPage,
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GetSubscriberAzureEngagementEvents retrieves Azure engagement events for a specific subscriber.
func (a *App) GetSubscriberAzureEngagementEvents(c echo.Context) error {
	var (
		subID = getID(c)
		pg    = a.pg.NewFromURL(c.Request().URL.Query())
	)

	query := `
		SELECT id, azure_message_id, campaign_id, subscriber_id, engagement_type,
		       engagement_context, user_agent, event_timestamp, created_at
		FROM azure_engagement_events
		WHERE subscriber_id = $1
		ORDER BY event_timestamp DESC
		LIMIT $2 OFFSET $3
	`

	var events []models.AzureEngagementEvent
	if err := a.db.Select(&events, query, subID, pg.Limit, pg.Offset); err != nil {
		a.log.Printf("error querying Azure engagement events for subscriber: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.Ts("globals.messages.errorFetching", "name", "engagement events"))
	}

	if len(events) == 0 {
		return c.JSON(http.StatusOK, okResp{[]models.AzureEngagementEvent{}})
	}

	return c.JSON(http.StatusOK, okResp{events})
}

// GetCampaignAzureAnalytics retrieves aggregated Azure analytics for a specific campaign.
func (a *App) GetCampaignAzureAnalytics(c echo.Context) error {
	campID := getID(c)

	type deliveryStats struct {
		Status string `db:"status" json:"status"`
		Count  int    `db:"count" json:"count"`
	}

	type engagementStats struct {
		Type  string `db:"engagement_type" json:"type"`
		Count int    `db:"count" json:"count"`
	}

	type analytics struct {
		DeliveryStats   []deliveryStats   `json:"delivery_stats"`
		EngagementStats []engagementStats `json:"engagement_stats"`
	}

	var result analytics

	// Get delivery stats
	deliveryQuery := `
		SELECT status, COUNT(*) as count
		FROM azure_delivery_events
		WHERE campaign_id = $1
		GROUP BY status
		ORDER BY count DESC
	`
	if err := a.db.Select(&result.DeliveryStats, deliveryQuery, campID); err != nil {
		a.log.Printf("error querying delivery stats: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.Ts("globals.messages.errorFetching", "name", "delivery stats"))
	}

	// Get engagement stats
	engagementQuery := `
		SELECT engagement_type, COUNT(*) as count
		FROM azure_engagement_events
		WHERE campaign_id = $1
		GROUP BY engagement_type
		ORDER BY count DESC
	`
	if err := a.db.Select(&result.EngagementStats, engagementQuery, campID); err != nil {
		a.log.Printf("error querying engagement stats: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.Ts("globals.messages.errorFetching", "name", "engagement stats"))
	}

	return c.JSON(http.StatusOK, okResp{result})
}
