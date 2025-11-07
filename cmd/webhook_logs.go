package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// GetWebhookLogs retrieves webhook logs with pagination and filtering.
func (a *App) GetWebhookLogs(c echo.Context) error {
	var (
		webhookType = c.QueryParam("webhook_type")
		eventType   = c.QueryParam("event_type")
		pg          = a.pg.NewFromURL(c.Request().URL.Query())
	)

	// Get total count
	var total int
	if err := a.queries.GetWebhookLogsCount.Get(&total, webhookType, eventType); err != nil {
		a.log.Printf("error getting webhook logs count: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("globals.messages.internalError"))
	}

	// Get logs
	var logs []models.WebhookLog
	if err := a.queries.GetWebhookLogs.Select(&logs, webhookType, eventType, pg.Offset, pg.Limit); err != nil {
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

// ExportWebhookLogs exports all webhook logs as a JSON file.
func (a *App) ExportWebhookLogs(c echo.Context) error {
	// Get all logs without pagination
	var logs []models.WebhookLog
	if err := a.queries.GetAllWebhookLogs.Select(&logs); err != nil {
		a.log.Printf("error getting all webhook logs: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, a.i18n.T("globals.messages.internalError"))
	}

	// Set headers for file download
	c.Response().Header().Set("Content-Type", "application/json")
	c.Response().Header().Set("Content-Disposition", "attachment; filename=webhook-logs.json")

	return c.JSON(http.StatusOK, logs)
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
