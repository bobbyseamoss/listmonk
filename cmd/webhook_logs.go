package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"gopkg.in/volatiletech/null.v6"
)

// GetWebhookLogs retrieves webhook logs with pagination and filtering.
func (a *App) GetWebhookLogs(c echo.Context) error {
	var (
		webhookType  = c.QueryParam("webhook_type")
		eventType    = c.QueryParam("event_type")
		statusFilter = c.QueryParam("status_filter")
		pg           = a.pg.NewFromURL(c.Request().URL.Query())
	)

	// Get total count
	var total int
	if err := a.queries.GetWebhookLogsCount.Get(&total, webhookType, eventType, statusFilter); err != nil {
		a.log.Printf("error getting webhook logs count: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("globals.messages.internalError"))
	}

	// Get logs
	var logs []models.WebhookLog
	if err := a.queries.GetWebhookLogs.Select(&logs, webhookType, eventType, statusFilter, pg.Offset, pg.Limit); err != nil {
		a.log.Printf("error getting webhook logs: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("globals.messages.internalError"))
	}

	if len(logs) == 0 {
		return c.JSON(http.StatusOK, okResp{models.PageResults{Results: []models.WebhookLog{}}})
	}

	out := models.PageResults{
		Results: logs,
		Total:   total,
		Page:    pg.Page,
		PerPage: pg.PerPage,
	}

	return c.JSON(http.StatusOK, okResp{out})
}

// ExportWebhookLogs exports webhook logs as a JSON or CSV file with optional filtering.
func (a *App) ExportWebhookLogs(c echo.Context) error {
	var (
		webhookType  = c.QueryParam("webhook_type")
		eventType    = c.QueryParam("event_type")
		statusFilter = c.QueryParam("status_filter")
		format       = c.QueryParam("format") // "json" or "csv", defaults to "csv"
	)

	if format == "" {
		format = "csv"
	}

	// Get filtered logs without pagination
	var logs []models.WebhookLog
	if webhookType == "" && eventType == "" && statusFilter == "" {
		// No filters, get all logs
		if err := a.queries.GetAllWebhookLogs.Select(&logs); err != nil {
			a.log.Printf("error getting all webhook logs: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("globals.messages.internalError"))
		}
	} else {
		// Apply filters
		if err := a.queries.GetFilteredWebhookLogs.Select(&logs, webhookType, eventType, statusFilter); err != nil {
			a.log.Printf("error getting filtered webhook logs: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("globals.messages.internalError"))
		}
	}

	if format == "csv" {
		return exportWebhookLogsCSV(c, logs)
	}

	// JSON format
	c.Response().Header().Set("Content-Type", "application/json")
	c.Response().Header().Set("Content-Disposition", "attachment; filename=webhook-logs.json")
	return c.JSON(http.StatusOK, logs)
}

// exportWebhookLogsCSV exports webhook logs as CSV with extracted fields.
func exportWebhookLogsCSV(c echo.Context, logs []models.WebhookLog) error {
	c.Response().Header().Set("Content-Type", "text/csv")
	c.Response().Header().Set("Content-Disposition", "attachment; filename=webhook-logs.csv")

	// Write CSV header
	header := "id,webhook_type,event_type,recipient,status,engagement_type,message_id,sender,created_at,processed,error_message,request_body\n"
	c.Response().Write([]byte(header))

	// Write each log as a CSV row
	for _, log := range logs {
		// Extract fields from request body JSON
		recipient, status, engagementType, messageID, sender := extractWebhookFields(log.RequestBody)

		// Format the row
		row := fmt.Sprintf("%d,%s,%s,%s,%s,%s,%s,%s,%s,%t,%s,%s\n",
			log.ID,
			escapeCSV(log.WebhookType),
			escapeCSV(nullStringValue(log.EventType)),
			escapeCSV(recipient),
			escapeCSV(status),
			escapeCSV(engagementType),
			escapeCSV(messageID),
			escapeCSV(sender),
			log.CreatedAt.Format("2006-01-02 15:04:05"),
			log.Processed,
			escapeCSV(nullStringValue(log.ErrorMessage)),
			escapeCSV(log.RequestBody),
		)
		c.Response().Write([]byte(row))
	}

	return nil
}

// extractWebhookFields extracts useful fields from the webhook request body JSON.
func extractWebhookFields(requestBody string) (recipient, status, engagementType, messageID, sender string) {
	var events []struct {
		EventType string `json:"eventType"`
		Data      struct {
			Recipient      string `json:"recipient"`
			Status         string `json:"status"`
			EngagementType string `json:"engagementType"`
			MessageID      string `json:"messageId"`
			Sender         string `json:"sender"`
		} `json:"data"`
	}

	if err := json.Unmarshal([]byte(requestBody), &events); err != nil {
		return "", "", "", "", ""
	}

	if len(events) > 0 {
		e := events[0]
		return e.Data.Recipient, e.Data.Status, e.Data.EngagementType, e.Data.MessageID, e.Data.Sender
	}

	return "", "", "", "", ""
}

// escapeCSV escapes a string for CSV format.
func escapeCSV(s string) string {
	if s == "" {
		return ""
	}
	// If contains comma, quote, or newline, wrap in quotes and escape quotes
	if containsCSVSpecial(s) {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}

// containsCSVSpecial checks if a string contains CSV special characters.
func containsCSVSpecial(s string) bool {
	return strings.ContainsAny(s, ",\"\n\r")
}

// nullStringValue returns the string value of a null.String or empty string.
func nullStringValue(ns null.String) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// DeleteWebhookLogs handles deletion of webhook logs.
func (a *App) DeleteWebhookLogs(c echo.Context) error {
	all, _ := strconv.ParseBool(c.QueryParam("all"))

	if all {
		// Delete all webhook logs
		if _, err := a.queries.DeleteAllWebhookLogs.Exec(); err != nil {
			a.log.Printf("error deleting all webhook logs: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("globals.messages.internalError"))
		}
	} else {
		// Delete specific logs by ID
		ids, err := parseStringIDs(c.Request().URL.Query()["id"])
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidID", "error", err.Error()))
		}
		if len(ids) == 0 {
			return echo.NewHTTPError(http.StatusBadRequest, a.i18n.Ts("globals.messages.invalidID"))
		}

		if _, err := a.queries.DeleteWebhookLogs.Exec(ids); err != nil {
			a.log.Printf("error deleting webhook logs: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("globals.messages.internalError"))
		}
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// logWebhook logs a webhook request to the database.
func (a *App) logWebhook(webhookType, eventType string, headers http.Header, body []byte, status int, responseBody string, processed bool, errorMsg string) {
	// Convert headers to JSONB
	headersMap := make(map[string]string)
	for key, values := range headers {
		if len(values) > 0 {
			headersMap[key] = values[0]
		}
	}

	headersJSON, err := json.Marshal(headersMap)
	if err != nil {
		a.log.Printf("error marshalling webhook headers: %v", err)
		headersJSON = []byte("{}")
	}

	// Convert eventType and responseBody to null.String
	var eventTypeNull, responseBodyNull, errorMsgNull interface{}
	if eventType != "" {
		eventTypeNull = eventType
	}
	if responseBody != "" {
		responseBodyNull = responseBody
	}
	if errorMsg != "" {
		errorMsgNull = errorMsg
	}

	// Insert webhook log
	_, err = a.queries.CreateWebhookLog.Exec(
		webhookType,
		eventTypeNull,
		headersJSON,
		string(body),
		status,
		responseBodyNull,
		processed,
		errorMsgNull,
	)

	if err != nil {
		a.log.Printf("error creating webhook log: %v", err)
	}
}
