package core

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx/types"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"gopkg.in/volatiletech/null.v6"
)

// RecordSiteEvent stores a site event for a known visitor.
func (c *Core) RecordSiteEvent(subscriberID int, sessionID string, event *models.TrackingEvent, ipAddress string) (int64, error) {
	// Parse session UUID
	sessUUID, err := uuid.Parse(sessionID)
	if err != nil {
		sessUUID = uuid.New()
	}

	// Convert properties to JSON
	props := types.JSONText("{}")
	if len(event.Properties) > 0 {
		props = types.JSONText(event.Properties)
	}

	var id int64
	if err := c.q.RecordSiteEvent.Get(&id,
		subscriberID,
		sessUUID,
		event.EventType,
		null.NewString(event.EventName, event.EventName != ""),
		null.NewString(event.PageURL, event.PageURL != ""),
		null.NewString(event.PageTitle, event.PageTitle != ""),
		null.NewString(event.Referrer, event.Referrer != ""),
		props,
		null.NewString(event.BrowserID, event.BrowserID != ""), // Use browser ID as user agent marker
		null.NewString(ipAddress, ipAddress != ""),
	); err != nil {
		c.log.Printf("error recording site event: %v", err)
		return 0, echo.NewHTTPError(http.StatusInternalServerError, c.i18n.T("globals.messages.errorGeneric"))
	}

	return id, nil
}

// IdentifyBrowser links a browser ID to a subscriber.
func (c *Core) IdentifyBrowser(browserID string, subscriberID int, via string, campaignID int) (int, error) {
	var id int
	campID := null.Int{}
	if campaignID > 0 {
		campID = null.IntFrom(campaignID)
	}

	if err := c.q.IdentifyBrowser.Get(&id, browserID, subscriberID, via, campID); err != nil {
		c.log.Printf("error identifying browser: %v", err)
		return 0, echo.NewHTTPError(http.StatusInternalServerError, c.i18n.T("globals.messages.errorGeneric"))
	}

	return id, nil
}

// GetBrowserSubscriber looks up a subscriber by browser ID.
func (c *Core) GetBrowserSubscriber(browserID string) (*models.Subscriber, error) {
	var sub models.Subscriber
	if err := c.q.GetBrowserSubscriber.Get(&sub, browserID); err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil // Not found, but not an error
		}
		c.log.Printf("error getting browser subscriber: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, c.i18n.T("globals.messages.errorGeneric"))
	}

	return &sub, nil
}

// UpdateBrowserLastSeen updates the last_seen_at timestamp for a browser.
func (c *Core) UpdateBrowserLastSeen(browserID string) error {
	if _, err := c.q.UpdateBrowserLastSeen.Exec(browserID); err != nil {
		c.log.Printf("error updating browser last seen: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, c.i18n.T("globals.messages.errorGeneric"))
	}
	return nil
}

// GetSubscriberSiteActivity returns site activity for a subscriber with pagination.
func (c *Core) GetSubscriberSiteActivity(subscriberID, limit, offset int) ([]models.SiteEvent, int, error) {
	var events []models.SiteEvent
	if err := c.q.GetSubscriberSiteActivity.Select(&events, subscriberID, limit, offset); err != nil {
		c.log.Printf("error getting subscriber site activity: %v", err)
		return nil, 0, echo.NewHTTPError(http.StatusInternalServerError, c.i18n.T("globals.messages.errorGeneric"))
	}

	// Get total count
	var count int
	if err := c.q.GetSubscriberSiteActivityCount.Get(&count, subscriberID); err != nil {
		c.log.Printf("error getting subscriber site activity count: %v", err)
		return nil, 0, echo.NewHTTPError(http.StatusInternalServerError, c.i18n.T("globals.messages.errorGeneric"))
	}

	return events, count, nil
}

// GetSubscriberRecentProducts returns recently viewed products for a subscriber.
func (c *Core) GetSubscriberRecentProducts(subscriberID, limit int) ([]models.SiteEvent, error) {
	var events []models.SiteEvent
	if err := c.q.GetSubscriberRecentProducts.Select(&events, subscriberID, limit); err != nil {
		c.log.Printf("error getting subscriber recent products: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, c.i18n.T("globals.messages.errorGeneric"))
	}
	return events, nil
}

// GetSiteTrackingStats returns aggregate statistics for site tracking.
func (c *Core) GetSiteTrackingStats() (*models.SiteTrackingStats, error) {
	var stats models.SiteTrackingStats
	if err := c.q.GetSiteEventsStats.Get(&stats); err != nil {
		c.log.Printf("error getting site tracking stats: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, c.i18n.T("globals.messages.errorGeneric"))
	}
	return &stats, nil
}

// DeleteOldSiteEvents removes site events older than the specified number of days.
func (c *Core) DeleteOldSiteEvents(days int) error {
	if _, err := c.q.DeleteOldSiteEvents.Exec(days); err != nil {
		c.log.Printf("error deleting old site events: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, c.i18n.T("globals.messages.errorGeneric"))
	}
	return nil
}

// GetSubscriberIDByUUID returns the subscriber ID for a given UUID (for tracking).
func (c *Core) GetSubscriberIDByUUID(subUUID string) (int, error) {
	var id int
	parsedUUID, err := uuid.Parse(subUUID)
	if err != nil {
		return 0, echo.NewHTTPError(http.StatusBadRequest, "Invalid subscriber UUID")
	}

	if err := c.q.GetSubscriberByUUIDForTracking.Get(&id, parsedUUID); err != nil {
		if err.Error() == "sql: no rows in result set" {
			return 0, echo.NewHTTPError(http.StatusNotFound, "Subscriber not found")
		}
		c.log.Printf("error getting subscriber by UUID: %v", err)
		return 0, echo.NewHTTPError(http.StatusInternalServerError, c.i18n.T("globals.messages.errorGeneric"))
	}

	return id, nil
}

// GetSubscriberByEmail returns a subscriber by their email address (for tracking identification).
func (c *Core) GetSubscriberByEmail(email string) (*models.Subscriber, error) {
	var sub models.Subscriber
	if err := c.q.GetSubscriberByEmail.Get(&sub, email); err != nil {
		if err.Error() == "sql: no rows in result set" {
			return nil, nil // Not found, but not an error
		}
		c.log.Printf("error getting subscriber by email: %v", err)
		return nil, echo.NewHTTPError(http.StatusInternalServerError, c.i18n.T("globals.messages.errorGeneric"))
	}
	return &sub, nil
}
