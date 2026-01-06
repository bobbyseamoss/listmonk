package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx/types"
	"gopkg.in/volatiletech/null.v6"
)

// SiteEvent represents a tracked event from website visitors.
type SiteEvent struct {
	ID           int64          `db:"id" json:"id"`
	SubscriberID null.Int       `db:"subscriber_id" json:"subscriber_id"`
	SessionID    uuid.UUID      `db:"session_id" json:"session_id"`
	EventType    string         `db:"event_type" json:"event_type"`
	EventName    null.String    `db:"event_name" json:"event_name"`
	PageURL      null.String    `db:"page_url" json:"page_url"`
	PageTitle    null.String    `db:"page_title" json:"page_title"`
	Referrer     null.String    `db:"referrer" json:"referrer"`
	Properties   types.JSONText `db:"properties" json:"properties"`
	UserAgent    null.String    `db:"user_agent" json:"user_agent"`
	IPAddress    null.String    `db:"ip_address" json:"ip_address"`
	CreatedAt    time.Time      `db:"created_at" json:"created_at"`
}

// KnownBrowser represents a browser that has been linked to a subscriber.
type KnownBrowser struct {
	ID            int       `db:"id" json:"id"`
	BrowserID     string    `db:"browser_id" json:"browser_id"`
	SubscriberID  int       `db:"subscriber_id" json:"subscriber_id"`
	IdentifiedVia string    `db:"identified_via" json:"identified_via"`
	CampaignID    null.Int  `db:"campaign_id" json:"campaign_id"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
	LastSeenAt    time.Time `db:"last_seen_at" json:"last_seen_at"`
}

// SiteEventInput represents the input for creating a new site event.
type SiteEventInput struct {
	SessionID  string          `json:"session_id"`
	EventType  string          `json:"event_type"`
	EventName  string          `json:"event_name,omitempty"`
	PageURL    string          `json:"page_url"`
	PageTitle  string          `json:"page_title,omitempty"`
	Referrer   string          `json:"referrer,omitempty"`
	Properties json.RawMessage `json:"properties,omitempty"`
}

// TrackingEvent represents an event received from the JavaScript tracker.
type TrackingEvent struct {
	BrowserID  string          `json:"browser_id"`
	SessionID  string          `json:"session_id"`
	EventType  string          `json:"event_type"`
	EventName  string          `json:"event_name,omitempty"`
	PageURL    string          `json:"page_url"`
	PageTitle  string          `json:"page_title,omitempty"`
	Referrer   string          `json:"referrer,omitempty"`
	Properties json.RawMessage `json:"properties,omitempty"`
	Timestamp  int64           `json:"timestamp,omitempty"`
}

// TrackingBatchRequest represents a batch of tracking events.
type TrackingBatchRequest struct {
	Events []TrackingEvent `json:"events"`
}

// IdentifyRequest represents a request to identify a browser.
type IdentifyRequest struct {
	BrowserID  string `json:"browser_id"`
	Email      string `json:"email,omitempty"`
	Properties json.RawMessage `json:"properties,omitempty"`
}

// SiteTrackingStats represents aggregate statistics for site tracking.
type SiteTrackingStats struct {
	TotalEvents    int64 `db:"total_events" json:"total_events"`
	UniqueVisitors int64 `db:"unique_visitors" json:"unique_visitors"`
	TotalSessions  int64 `db:"total_sessions" json:"total_sessions"`
	PageViews      int64 `db:"page_views" json:"page_views"`
	ProductViews   int64 `db:"product_views" json:"product_views"`
	Events24h      int64 `db:"events_24h" json:"events_24h"`
}

// ProductView represents a viewed product event properties.
type ProductView struct {
	ProductID      string  `json:"product_id"`
	ProductName    string  `json:"product_name"`
	ProductURL     string  `json:"product_url"`
	ProductImage   string  `json:"product_image,omitempty"`
	ProductPrice   float64 `json:"product_price,omitempty"`
	ProductVariant string  `json:"product_variant,omitempty"`
	ProductVendor  string  `json:"product_vendor,omitempty"`
}

// Event type constants.
const (
	EventTypePageView      = "page_view"
	EventTypeViewedProduct = "viewed_product"
	EventTypeCustom        = "custom"
	EventTypeIdentify      = "identify"
)

// Identification method constants.
const (
	IdentifiedViaEmailClick = "email_click"
	IdentifiedViaFormSubmit = "form_submit"
	IdentifiedViaAPICall    = "api_identify"
)
