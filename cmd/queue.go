package main

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/knadh/listmonk/internal/auth"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
)

// queueItemDB is the database model for queue items
type queueItemDB struct {
	ID                     int            `db:"id"`
	CampaignID             int            `db:"campaign_id"`
	CampaignName           string         `db:"campaign_name"`
	CampaignUUID           string         `db:"campaign_uuid"`
	SubscriberID           int            `db:"subscriber_id"`
	SubscriberEmail        string         `db:"subscriber_email"`
	SubscriberUUID         string         `db:"subscriber_uuid"`
	Status                 string         `db:"status"`
	Priority               int            `db:"priority"`
	ScheduledAt            sql.NullTime   `db:"scheduled_at"`
	SentAt                 sql.NullTime   `db:"sent_at"`
	AssignedSMTPServerUUID sql.NullString `db:"assigned_smtp_server_uuid"`
	RetryCount             int            `db:"retry_count"`
	ErrorMessage           sql.NullString `db:"error_message"`
	CreatedAt              sql.NullTime   `db:"created_at"`
	UpdatedAt              sql.NullTime   `db:"updated_at"`
	Total                  int            `db:"total"`
}

// queueItemResp is the JSON response model for queue items
type queueItemResp struct {
	ID                     int     `json:"id"`
	CampaignID             int     `json:"campaignId"`
	CampaignName           string  `json:"campaignName"`
	CampaignUUID           string  `json:"campaignUuid"`
	SubscriberID           int     `json:"subscriberId"`
	SubscriberEmail        string  `json:"subscriberEmail"`
	SubscriberUUID         string  `json:"subscriberUuid"`
	Status                 string  `json:"status"`
	Priority               int     `json:"priority"`
	ScheduledAt            *string `json:"scheduledAt"`
	SentAt                 *string `json:"sentAt"`
	AssignedSMTPServerUUID *string `json:"assignedSmtpServerUuid"`
	RetryCount             int     `json:"retryCount"`
	ErrorMessage           *string `json:"errorMessage"`
	CreatedAt              *string `json:"createdAt"`
	UpdatedAt              *string `json:"updatedAt"`
}

// queueStatsResp represents summary statistics for the queue
type queueStatsResp struct {
	Queued          int     `db:"queued" json:"queued"`
	Sending         int     `db:"sending" json:"sending"`
	Sent            int     `db:"sent" json:"sent"`
	Failed          int     `db:"failed" json:"failed"`
	Cancelled       int     `db:"cancelled" json:"cancelled"`
	NextScheduledAt *string `json:"nextScheduledAt"`
}

// smtpCapacityResp represents SMTP server capacity information
type smtpCapacityResp struct {
	UUID           string `db:"smtp_uuid" json:"uuid"`
	Name           string `db:"smtp_name" json:"name"`
	DailyLimit     int    `db:"daily_limit" json:"daily_limit"`
	DailyUsed      int    `db:"daily_used" json:"daily_used"`
	DailyRemaining int    `db:"daily_remaining" json:"daily_remaining"`
	FromEmail      string `db:"from_email" json:"from_email"`
}

// convertQueueItemToResp converts a database queue item to a JSON response struct
func convertQueueItemToResp(item queueItemDB) queueItemResp {
	resp := queueItemResp{
		ID:              item.ID,
		CampaignID:      item.CampaignID,
		CampaignName:    item.CampaignName,
		CampaignUUID:    item.CampaignUUID,
		SubscriberID:    item.SubscriberID,
		SubscriberEmail: item.SubscriberEmail,
		SubscriberUUID:  item.SubscriberUUID,
		Status:          item.Status,
		Priority:        item.Priority,
		RetryCount:      item.RetryCount,
	}

	// Convert sql.NullTime to *string
	if item.ScheduledAt.Valid {
		timeStr := item.ScheduledAt.Time.Format("2006-01-02T15:04:05Z07:00")
		resp.ScheduledAt = &timeStr
	}
	if item.SentAt.Valid {
		timeStr := item.SentAt.Time.Format("2006-01-02T15:04:05Z07:00")
		resp.SentAt = &timeStr
	}
	if item.CreatedAt.Valid {
		timeStr := item.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
		resp.CreatedAt = &timeStr
	}
	if item.UpdatedAt.Valid {
		timeStr := item.UpdatedAt.Time.Format("2006-01-02T15:04:05Z07:00")
		resp.UpdatedAt = &timeStr
	}

	// Convert sql.NullString to *string
	if item.AssignedSMTPServerUUID.Valid {
		resp.AssignedSMTPServerUUID = &item.AssignedSMTPServerUUID.String
	}
	if item.ErrorMessage.Valid {
		resp.ErrorMessage = &item.ErrorMessage.String
	}

	return resp
}

// GetQueueItems handles retrieval of queue items with pagination and filters
func (a *App) GetQueueItems(c echo.Context) error {
	// Get the authenticated user and check permissions
	user := auth.GetUser(c)
	if !user.HasPerm(auth.PermCampaignsGet) {
		return echo.NewHTTPError(http.StatusForbidden, "Insufficient permissions")
	}

	var (
		pg = a.pg.NewFromURL(c.Request().URL.Query())

		campaignID, _    = strconv.Atoi(c.QueryParam("campaign_id"))
		status           = c.QueryParams()["status"]
		smtpServerUUID   = strings.TrimSpace(c.QueryParam("smtp_server_uuid"))
		subscriberSearch = strings.TrimSpace(c.QueryParam("subscriber"))
	)

	// Prepare status filter
	var statusFilter pq.StringArray
	if len(status) > 0 {
		statusFilter = pq.StringArray(status)
	} else {
		statusFilter = pq.StringArray{}
	}

	// Query queue items from database
	var dbItems []queueItemDB
	if err := a.queries.GetQueueItems.Select(&dbItems,
		campaignID,
		statusFilter,
		smtpServerUUID,
		"%"+subscriberSearch+"%",
		pg.Offset,
		pg.Limit,
	); err != nil {
		a.log.Printf("error fetching queue items: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error fetching queue items")
	}

	// Get total from the first record or default to 0
	total := 0
	if len(dbItems) > 0 {
		total = dbItems[0].Total
	}

	// Paginate the response
	if len(dbItems) == 0 {
		return c.JSON(http.StatusOK, okResp{models.PageResults{Results: []queueItemResp{}}})
	}

	// Convert database items to response items
	respItems := make([]queueItemResp, len(dbItems))
	for i, item := range dbItems {
		respItems[i] = convertQueueItemToResp(item)
	}

	out := models.PageResults{
		Results: respItems,
		Total:   total,
		Page:    pg.Page,
		PerPage: pg.PerPage,
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// GetEmailQueueStats handles retrieval of queue summary statistics
func (a *App) GetEmailQueueStats(c echo.Context) error {
	// Get the authenticated user and check permissions
	user := auth.GetUser(c)
	if !user.HasPerm(auth.PermCampaignsGet) {
		return echo.NewHTTPError(http.StatusForbidden, "Insufficient permissions")
	}

	var stats queueStatsResp

	// Get summary statistics
	if err := a.queries.GetQueueSummaryStats.Get(&stats); err != nil {
		a.log.Printf("error fetching queue stats: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error fetching queue statistics")
	}

	// Get next scheduled email
	var nextScheduled sql.NullTime
	if err := a.queries.GetNextScheduledEmail.Get(&nextScheduled); err != nil && err != sql.ErrNoRows {
		a.log.Printf("error fetching next scheduled email: %v", err)
		// Don't fail the entire request, just leave it null
	}

	// Convert sql.NullTime to *string
	if nextScheduled.Valid {
		timeStr := nextScheduled.Time.Format("2006-01-02T15:04:05Z07:00")
		stats.NextScheduledAt = &timeStr
	}

	return c.JSON(http.StatusOK, okResp{stats})
}

// GetSMTPServerCapacity handles retrieval of SMTP server capacity information
func (a *App) GetSMTPServerCapacity(c echo.Context) error {
	// Get the authenticated user and check permissions
	user := auth.GetUser(c)
	if !user.HasPerm(auth.PermSettingsGet) {
		return echo.NewHTTPError(http.StatusForbidden, "Insufficient permissions")
	}

	var capacity []smtpCapacityResp

	// Query SMTP server capacity
	if err := a.queries.GetSMTPServerCapacity.Select(&capacity); err != nil {
		a.log.Printf("error fetching SMTP server capacity: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error fetching server capacity")
	}

	if len(capacity) == 0 {
		return c.JSON(http.StatusOK, okResp{[]smtpCapacityResp{}})
	}

	return c.JSON(http.StatusOK, okResp{capacity})
}

// CancelQueueItem handles cancellation of a specific queue item
func (a *App) CancelQueueItem(c echo.Context) error {
	// Get the authenticated user and check permissions
	user := auth.GetUser(c)
	if !user.HasPerm(auth.PermCampaignsManage) {
		return echo.NewHTTPError(http.StatusForbidden, "Insufficient permissions")
	}

	// Get the queue item ID
	id := getID(c)

	// Cancel the queue item
	res, err := a.queries.CancelQueueItem.Exec(id)
	if err != nil {
		a.log.Printf("error cancelling queue item %d: %v", id, err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error cancelling queue item")
	}

	// Check if any rows were affected
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		a.log.Printf("error getting rows affected: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error checking cancellation status")
	}

	if rowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "Queue item not found or already processed")
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// RetryQueueItem handles retry of a failed or cancelled queue item
func (a *App) RetryQueueItem(c echo.Context) error {
	// Get the authenticated user and check permissions
	user := auth.GetUser(c)
	if !user.HasPerm(auth.PermCampaignsManage) {
		return echo.NewHTTPError(http.StatusForbidden, "Insufficient permissions")
	}

	// Get the queue item ID
	id := getID(c)

	// Retry the queue item
	res, err := a.queries.RetryQueueItem.Exec(id)
	if err != nil {
		a.log.Printf("error retrying queue item %d: %v", id, err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error retrying queue item")
	}

	// Check if any rows were affected
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		a.log.Printf("error getting rows affected: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error checking retry status")
	}

	if rowsAffected == 0 {
		return echo.NewHTTPError(http.StatusNotFound, "Queue item not found or not in a retriable state")
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// ClearAllQueuedEmails handles clearing (cancelling) all queued emails
func (a *App) ClearAllQueuedEmails(c echo.Context) error {
	// Get the authenticated user and check permissions
	user := auth.GetUser(c)
	if !user.HasPerm(auth.PermCampaignsManage) {
		return echo.NewHTTPError(http.StatusForbidden, "Insufficient permissions")
	}

	// Get counts before clearing
	var counts struct {
		Queued    int `db:"queued"`
		Cancelled int `db:"cancelled"`
		Sent      int `db:"sent"`
		Failed    int `db:"failed"`
	}
	err := a.db.Get(&counts, `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'queued' THEN 1 ELSE 0 END), 0) as queued,
			COALESCE(SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END), 0) as cancelled,
			COALESCE(SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END), 0) as sent,
			COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) as failed
		FROM email_queue
	`)
	if err != nil {
		a.log.Printf("error getting queue counts: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error getting queue counts")
	}

	a.log.Printf("clearing queue: %d queued, %d cancelled, %d sent, %d failed",
		counts.Queued, counts.Cancelled, counts.Sent, counts.Failed)

	// Use TRUNCATE for fast clearing of the entire table
	// This is much faster than DELETE and releases locks immediately
	_, err = a.db.Exec(`TRUNCATE email_queue`)
	if err != nil {
		a.log.Printf("error truncating email_queue: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error clearing queue")
	}

	a.log.Printf("truncated email_queue table")

	resetCapacity := false
	resetSlidingWindow := false
	resetAccountRateLimit := false

	// Reset SMTP daily usage (to reset server capacity)
	_, err = a.db.Exec(`DELETE FROM smtp_daily_usage`)
	if err != nil {
		a.log.Printf("error resetting SMTP daily usage: %v", err)
	} else {
		resetCapacity = true
		a.log.Println("reset all SMTP server daily usage")
	}

	// Reset SMTP sliding window state (to reset rate limits)
	_, err = a.db.Exec(`DELETE FROM smtp_rate_limit_state`)
	if err != nil {
		a.log.Printf("error resetting SMTP sliding window state: %v", err)
	} else {
		resetSlidingWindow = true
		a.log.Println("reset all SMTP server sliding window state")
	}

	// Reset account-wide rate limit state (to reset account-wide counters)
	_, err = a.db.Exec(`DELETE FROM account_rate_limit_state`)
	if err != nil {
		a.log.Printf("error resetting account rate limit state: %v", err)
	} else {
		resetAccountRateLimit = true
		a.log.Println("reset account-wide rate limit state")
	}

	return c.JSON(http.StatusOK, okResp{map[string]interface{}{
		"success":                  true,
		"queued_count":             counts.Queued,
		"canceled_count":           counts.Cancelled,
		"sent_count":               counts.Sent,
		"failed_count":             counts.Failed,
		"reset_capacity":           resetCapacity,
		"reset_sliding_window":     resetSlidingWindow,
		"reset_account_rate_limit": resetAccountRateLimit,
	}})
}

// ToggleQueuePause handles pausing/resuming queue processing
func (a *App) ToggleQueuePause(c echo.Context) error {
	// Get the authenticated user and check permissions
	user := auth.GetUser(c)
	if !user.HasPerm(auth.PermSettingsManage) {
		return echo.NewHTTPError(http.StatusForbidden, "Insufficient permissions")
	}

	// Parse request body
	var req struct {
		Paused bool `json:"paused"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request")
	}

	// Get current settings
	settings, err := a.core.GetSettings()
	if err != nil {
		a.log.Printf("error getting settings: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error getting settings")
	}

	// Update the queue paused setting
	settings.AppQueuePaused = req.Paused

	// Save settings
	if err := a.core.UpdateSettings(settings); err != nil {
		a.log.Printf("error updating queue pause setting: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error updating queue pause setting")
	}

	status := "resumed"
	if req.Paused {
		status = "paused"
	}
	a.log.Printf("queue %s", status)

	return c.JSON(http.StatusOK, okResp{map[string]interface{}{
		"success": true,
		"paused":  req.Paused,
	}})
}

// SendAllQueuedEmails handles sending all queued emails immediately
func (a *App) SendAllQueuedEmails(c echo.Context) error {
	// Get the authenticated user and check permissions
	user := auth.GetUser(c)
	if !user.HasPerm(auth.PermCampaignsManage) {
		return echo.NewHTTPError(http.StatusForbidden, "Insufficient permissions")
	}

	// Send all queued emails immediately (sets scheduled_at to NOW)
	res, err := a.queries.SendAllQueuedEmails.Exec()
	if err != nil {
		a.log.Printf("error sending all queued emails: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error sending queued emails")
	}

	// Get number of rows affected
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		a.log.Printf("error getting rows affected: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error checking send status")
	}

	a.log.Printf("scheduled %d queued emails for immediate sending", rowsAffected)

	return c.JSON(http.StatusOK, okResp{map[string]interface{}{
		"success": true,
		"count":   rowsAffected,
	}})
}
