package main

import (
	"net/http"
	"strconv"

	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// GetActivityFeed retrieves a paginated activity feed of subscriber events.
func (a *App) GetActivityFeed(c echo.Context) error {
	var (
		eventType = c.QueryParam("event_type")
		page, _   = strconv.Atoi(c.QueryParam("page"))
		perPage   = 20
	)

	if page < 1 {
		page = 1
	}

	offset := (page - 1) * perPage

	// Query the activity feed
	var events []models.ActivityEvent
	if err := a.queries.GetActivityFeed.Select(&events, eventType, perPage, offset); err != nil {
		a.log.Printf("error fetching activity feed: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError,
			a.i18n.Ts("globals.messages.errorFetching", "name", "activity feed", "error", err.Error()))
	}

	// Get total count from the first result if available
	total := 0
	if len(events) > 0 {
		total = events[0].Total
	}

	return c.JSON(http.StatusOK, okResp{struct {
		Results []models.ActivityEvent `json:"results"`
		Total   int                    `json:"total"`
		PerPage int                    `json:"per_page"`
		Page    int                    `json:"page"`
	}{
		Results: events,
		Total:   total,
		PerPage: perPage,
		Page:    page,
	}})
}
