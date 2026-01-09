package models

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/textproto"
	"regexp"
	"strings"
	txttpl "text/template"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/lib/pq"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	null "gopkg.in/volatiletech/null.v6"
)

// Enum values for various statuses.
const (
	// Subscriber.
	SubscriberStatusEnabled     = "enabled"
	SubscriberStatusDisabled    = "disabled"
	SubscriberStatusBlockListed = "blocklisted"

	// Subscription.
	SubscriptionStatusUnconfirmed  = "unconfirmed"
	SubscriptionStatusConfirmed    = "confirmed"
	SubscriptionStatusUnsubscribed = "unsubscribed"

	// Campaign.
	CampaignStatusDraft         = "draft"
	CampaignStatusScheduled     = "scheduled"
	CampaignStatusRunning       = "running"
	CampaignStatusPaused        = "paused"
	CampaignStatusFinished      = "finished"
	CampaignStatusCancelled     = "cancelled"
	CampaignTypeRegular         = "regular"
	CampaignTypeOptin           = "optin"
	CampaignContentTypeRichtext = "richtext"
	CampaignContentTypeHTML     = "html"
	CampaignContentTypeMarkdown = "markdown"
	CampaignContentTypePlain    = "plain"
	CampaignContentTypeVisual   = "visual"

	// List.
	ListTypePrivate = "private"
	ListTypePublic  = "public"
	ListOptinSingle = "single"
	ListOptinDouble = "double"

	// BaseTpl is the name of the base template.
	BaseTpl = "base"

	// ContentTpl is the name of the compiled message.
	ContentTpl = "content"

	// Headers attached to e-mails for bounce tracking.
	EmailHeaderSubscriberUUID = "X-Listmonk-Subscriber"
	EmailHeaderCampaignUUID   = "X-Listmonk-Campaign"

	// Standard e-mail headers.
	EmailHeaderDate        = "Date"
	EmailHeaderFrom        = "From"
	EmailHeaderSubject     = "Subject"
	EmailHeaderMessageId   = "Message-Id"
	EmailHeaderDeliveredTo = "Delivered-To"
	EmailHeaderReceived    = "Received"

	BounceTypeHard      = "hard"
	BounceTypeSoft      = "soft"
	BounceTypeComplaint = "complaint"

	// Templates.
	TemplateTypeCampaign       = "campaign"
	TemplateTypeCampaignVisual = "campaign_visual"
	TemplateTypeTx             = "tx"

	// Sign-up form types.
	FormTypePopup    = "popup"
	FormTypeFlyout   = "flyout"
	FormTypeFullpage = "fullpage"
	FormTypeEmbed    = "embed"
	FormTypeBanner   = "banner"

	// Sign-up form statuses.
	FormStatusDraft    = "draft"
	FormStatusActive   = "active"
	FormStatusPaused   = "paused"
	FormStatusArchived = "archived"
)

// Headers represents an array of string maps used to represent SMTP, HTTP headers etc.
// similar to url.Values{}
type Headers []map[string]string

// regTplFunc represents contains a regular expression for wrapping and
// substituting a Go template function from the user's shorthand to a full
// function call.
type regTplFunc struct {
	regExp  *regexp.Regexp
	replace string
}

var regTplFuncs = []regTplFunc{
	// Regular expression for matching {{ TrackLink "http://link.com" }} in the template
	// and substituting it with {{ TrackLink "http://link.com" . }} (the dot context)
	// before compilation. This is to make linking easier for users.
	{
		regExp:  regexp.MustCompile(`{{\s*TrackLink\s+"([^"]+)"\s*}}`),
		replace: `{{ TrackLink "$1" . }}`,
	},

	// Convert the shorthand https://google.com@TrackLink to {{ TrackLink ... }}.
	// This is for WYSIWYG editors that encode and break quotes {{ "" }} when inserted
	// inside <a href="{{ TrackLink "https://these-quotes-break" }}>.
	// The regex matches all characters that may occur in an URL
	// (see "2. Characters" in RFC3986: https://www.ietf.org/rfc/rfc3986.txt)
	{
		regExp:  regexp.MustCompile(`(https?://[\p{L}\p{N}_\-\.~!#$&'()*+,/:;=?@\[\]]*)@TrackLink`),
		replace: `{{ TrackLink "$1" . }}`,
	},

	{
		regExp:  regexp.MustCompile(`{{(\s+)?(TrackView|UnsubscribeURL|ManageURL|OptinURL|MessageURL)(\s+)?}}`),
		replace: `{{ $2 . }}`,
	},
}

// PageResults is a generic HTTP response container for paginated results of list of items.
type PageResults struct {
	Results any `json:"results"`

	Search  string `json:"search"`
	Query   string `json:"query"`
	Total   int    `json:"total"`
	PerPage int    `json:"per_page"`
	Page    int    `json:"page"`
}

// Base holds common fields shared across models.
type Base struct {
	ID        int       `db:"id" json:"id"`
	CreatedAt null.Time `db:"created_at" json:"created_at"`
	UpdatedAt null.Time `db:"updated_at" json:"updated_at"`
}

// Subscriber represents an e-mail subscriber.
type Subscriber struct {
	Base

	UUID    string         `db:"uuid" json:"uuid"`
	Email   string         `db:"email" json:"email" form:"email"`
	Name    string         `db:"name" json:"name" form:"name"`
	Attribs JSON           `db:"attribs" json:"attribs"`
	Status  string         `db:"status" json:"status"`
	Lists   types.JSONText `db:"lists" json:"lists"`
}
type subLists struct {
	SubscriberID int            `db:"subscriber_id"`
	Lists        types.JSONText `db:"lists"`
}

// Subscription represents a list attached to a subscriber.
type Subscription struct {
	List
	SubscriptionStatus    null.String     `db:"subscription_status" json:"subscription_status"`
	SubscriptionCreatedAt null.String     `db:"subscription_created_at" json:"subscription_created_at"`
	Meta                  json.RawMessage `db:"meta" json:"meta"`
}

// SubscriberExportProfile represents a subscriber's collated data in JSON for export.
type SubscriberExportProfile struct {
	Email         string          `db:"email" json:"-"`
	Profile       json.RawMessage `db:"profile" json:"profile,omitempty"`
	Subscriptions json.RawMessage `db:"subscriptions" json:"subscriptions,omitempty"`
	CampaignViews json.RawMessage `db:"campaign_views" json:"campaign_views,omitempty"`
	LinkClicks    json.RawMessage `db:"link_clicks" json:"link_clicks,omitempty"`
}

// JSON is the wrapper for reading and writing arbitrary JSONB fields from the DB.
type JSON map[string]any

// StringIntMap is used to define DB Scan()s.
type StringIntMap map[string]int

// Subscribers represents a slice of Subscriber.
type Subscribers []Subscriber

// SubscriberExport represents a subscriber record that is exported to raw data.
type SubscriberExport struct {
	Base

	UUID    string `db:"uuid" json:"uuid"`
	Email   string `db:"email" json:"email"`
	Name    string `db:"name" json:"name"`
	Attribs string `db:"attribs" json:"attribs"`
	Status  string `db:"status" json:"status"`
}

// List represents a mailing list.
type List struct {
	Base

	UUID             string         `db:"uuid" json:"uuid"`
	Name             string         `db:"name" json:"name"`
	Type             string         `db:"type" json:"type"`
	Optin            string         `db:"optin" json:"optin"`
	Tags             pq.StringArray `db:"tags" json:"tags"`
	Description      string         `db:"description" json:"description"`
	SubscriberCount  int            `db:"subscriber_count" json:"subscriber_count"`
	SubscriberCounts StringIntMap   `db:"subscriber_statuses" json:"subscriber_statuses"`
	SubscriberID     int            `db:"subscriber_id" json:"-"`

	// This is only relevant when querying the lists of a subscriber.
	SubscriptionStatus    string    `db:"subscription_status" json:"subscription_status,omitempty"`
	SubscriptionCreatedAt null.Time `db:"subscription_created_at" json:"subscription_created_at,omitempty"`
	SubscriptionUpdatedAt null.Time `db:"subscription_updated_at" json:"subscription_updated_at,omitempty"`

	// Pseudofield for getting the total number of subscribers
	// in searches and queries.
	Total int `db:"total" json:"-"`
}

// SegmentCondition represents a single condition in a segment filter.
type SegmentCondition struct {
	Field     string         `json:"field"`            // e.g., "email", "name", "status", "attribs.plan", "behavior.email_opens"
	FieldType string         `json:"field_type"`       // text, number, date, boolean, array, behavior
	Operator  string         `json:"operator"`         // equals, not_equals, contains, greater_than, at_least, zero, etc.
	Value     any            `json:"value"`            // The comparison value
	Params    map[string]any `json:"params,omitempty"` // Additional parameters (time_range_days, campaign_id, etc.)
}

// SegmentConditionGroup represents a group of conditions with AND/OR logic.
// It can contain SegmentConditions or nested SegmentConditionGroups.
type SegmentConditionGroup struct {
	Operator   string `json:"operator"`   // "and" or "or"
	Conditions []any  `json:"conditions"` // Can be SegmentCondition or nested SegmentConditionGroup
}

// Scan implements the sql.Scanner interface for JSONB columns.
func (s *SegmentConditionGroup) Scan(src any) error {
	if src == nil {
		return nil
	}

	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("unsupported type for SegmentConditionGroup: %T", src)
	}

	return json.Unmarshal(b, s)
}

// Value implements the driver.Valuer interface for JSONB columns.
func (s SegmentConditionGroup) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Segment represents a dynamic subscriber segment defined by conditions.
type Segment struct {
	Base

	UUID        string                `db:"uuid" json:"uuid"`
	Name        string                `db:"name" json:"name"`
	Description string                `db:"description" json:"description"`
	Conditions  SegmentConditionGroup `db:"conditions" json:"conditions"`
	CachedCount null.Int              `db:"cached_count" json:"cached_count"`
	CachedAt    null.Time             `db:"cached_at" json:"cached_at"`
	Tags        pq.StringArray        `db:"tags" json:"tags"`

	// Computed field for subscriber count (from preview/query)
	SubscriberCount int `db:"-" json:"subscriber_count,omitempty"`

	// Pseudofield for pagination total
	Total int `db:"total" json:"-"`
}

// Segments is a slice of Segment.
type Segments []Segment

// CampaignMinimal represents minimal campaign info for dropdowns.
type CampaignMinimal struct {
	ID   int    `db:"id" json:"id"`
	UUID string `db:"uuid" json:"uuid"`
	Name string `db:"name" json:"name"`
}

// CampaignPerformance represents performance metrics for a campaign.
type CampaignPerformance struct {
	ID           int       `db:"id" json:"id"`
	UUID         string    `db:"uuid" json:"uuid"`
	Name         string    `db:"name" json:"name"`
	SentAt       time.Time `db:"sent_at" json:"sent_at"`
	Recipients   int       `db:"recipients" json:"recipients"`
	Delivered    int       `db:"delivered" json:"delivered"`
	UniqueOpens  int       `db:"unique_opens" json:"unique_opens"`
	UniqueClicks int       `db:"unique_clicks" json:"unique_clicks"`
	OpenRate     float64   `db:"open_rate" json:"open_rate"`
	ClickRate    float64   `db:"click_rate" json:"click_rate"`
	Total        int       `db:"total" json:"-"`
}

// Campaign represents an e-mail campaign.
type Campaign struct {
	Base
	CampaignMeta

	UUID              string          `db:"uuid" json:"uuid"`
	Type              string          `db:"type" json:"type"`
	Name              string          `db:"name" json:"name"`
	Subject           string          `db:"subject" json:"subject"`
	FromEmail         string          `db:"from_email" json:"from_email"`
	Body              string          `db:"body" json:"body"`
	BodySource        null.String     `db:"body_source" json:"body_source"`
	AltBody           null.String     `db:"altbody" json:"altbody"`
	SendAt            null.Time       `db:"send_at" json:"send_at"`
	Status            string          `db:"status" json:"status"`
	ContentType       string          `db:"content_type" json:"content_type"`
	Tags              pq.StringArray  `db:"tags" json:"tags"`
	Headers           Headers         `db:"headers" json:"headers"`
	TemplateID        null.Int        `db:"template_id" json:"template_id"`
	Messenger         string          `db:"messenger" json:"messenger"`
	Archive           bool            `db:"archive" json:"archive"`
	ArchiveSlug       null.String     `db:"archive_slug" json:"archive_slug"`
	ArchiveTemplateID null.Int        `db:"archive_template_id" json:"archive_template_id"`
	ArchiveMeta       json.RawMessage `db:"archive_meta" json:"archive_meta"`

	// BypassTimeWindow allows campaigns to be sent outside the configured time window
	// (useful for test campaigns when outside normal sending hours)
	BypassTimeWindow bool `db:"bypass_time_window" json:"bypass_time_window"`

	// TargetType determines how the campaign targets subscribers: 'lists' (default) or 'segments'
	TargetType string `db:"target_type" json:"target_type"`

	// TemplateBody is joined in from templates by the next-campaigns query.
	TemplateBody        string             `db:"template_body" json:"-"`
	ArchiveTemplateBody string             `db:"archive_template_body" json:"-"`
	Tpl                 *template.Template `json:"-"`
	SubjectTpl          *txttpl.Template   `json:"-"`
	AltBodyTpl          *template.Template `json:"-"`

	// List of media (attachment) IDs obtained from the next-campaign query
	// while sending a campaign.
	MediaIDs pq.Int64Array `json:"-" db:"media_id"`

	// Fetched bodies of the attachments.
	Attachments []Attachment `json:"-" db:"-"`

	// Pseudofield for getting the total number of subscribers
	// in searches and queries.
	Total int `db:"total" json:"-"`
}

// CampaignMeta contains fields tracking a campaign's progress.
type CampaignMeta struct {
	CampaignID int `db:"campaign_id" json:"-"`
	Views      int `db:"views" json:"views"`
	Clicks     int `db:"clicks" json:"clicks"`
	Bounces    int `db:"bounces" json:"bounces"`

	// This is a list of {list_id, name} pairs unlike Subscriber.Lists[]
	// because lists can be deleted after a campaign is finished, resulting
	// in null lists data to be returned. For that reason, campaign_lists maintains
	// campaign-list associations with a historical record of id + name that persist
	// even after a list is deleted.
	Lists types.JSONText `db:"lists" json:"lists"`
	Media types.JSONText `db:"media" json:"media"`

	// Segments is a list of {segment_id, name} pairs for segment-targeted campaigns.
	// Similar to Lists, maintains historical record even after segment is deleted.
	Segments types.JSONText `db:"segments" json:"segments"`

	StartedAt null.Time `db:"started_at" json:"started_at"`
	ToSend    int       `db:"to_send" json:"to_send"`
	Sent      int       `db:"sent" json:"sent"`
	Delivered int       `db:"delivered" json:"delivered"`
	AzureSent int       `db:"azure_sent" json:"azure_sent"`

	// Purchase attribution stats (from Shopify integration)
	PurchaseOrders  int     `db:"purchase_orders" json:"purchase_orders"`
	PurchaseRevenue float64 `db:"purchase_revenue" json:"purchase_revenue"`
	PurchaseCurrency string `db:"purchase_currency" json:"purchase_currency"`
}

type CampaignStats struct {
	ID        int       `db:"id" json:"id"`
	Status    string    `db:"status" json:"status"`
	ToSend    int       `db:"to_send" json:"to_send"`
	Sent      int       `db:"sent" json:"sent"`
	Views     int       `db:"views" json:"views"`
	Clicks    int       `db:"clicks" json:"clicks"`
	AzureSent int       `db:"azure_sent" json:"azure_sent"`
	Started   null.Time `db:"started_at" json:"started_at"`
	UpdatedAt null.Time `db:"updated_at" json:"updated_at"`
	Rate      int       `json:"rate"`
	NetRate   int       `json:"net_rate"`

	// Queue-specific stats (for campaigns using messenger='automatic')
	UseQueue       bool   `db:"use_queue" json:"use_queue"`
	Messenger      string `db:"messenger" json:"messenger"`
	QueueQueued    int    `db:"queue_queued" json:"queue_queued"`
	QueueSending   int    `db:"queue_sending" json:"queue_sending"`
	QueueSent      int    `db:"queue_sent" json:"queue_sent"`
	QueueFailed    int    `db:"queue_failed" json:"queue_failed"`
	QueueCancelled int    `db:"queue_cancelled" json:"queue_cancelled"`
	QueueTotal     int    `db:"queue_total" json:"queue_total"`

	// Auto-pause tracking for time window functionality
	AutoPaused   bool      `db:"auto_paused" json:"auto_paused"`
	AutoPausedAt null.Time `db:"auto_paused_at" json:"auto_paused_at"`
}

type CampaignAnalyticsCount struct {
	CampaignID int       `db:"campaign_id" json:"campaign_id"`
	Count      int       `db:"count" json:"count"`
	Timestamp  time.Time `db:"timestamp" json:"timestamp"`
}

type CampaignAnalyticsLink struct {
	URL   string `db:"url" json:"url"`
	Count int    `db:"count" json:"count"`
}

// CampaignAzureDeliveryCount represents Azure delivery event counts by status
type CampaignAzureDeliveryCount struct {
	CampaignID int    `db:"campaign_id" json:"campaign_id"`
	Status     string `db:"status" json:"status"`
	Count      int    `db:"count" json:"count"`
}

// CampaignUnsubscriber represents a subscriber who unsubscribed after receiving a campaign
type CampaignUnsubscriber struct {
	ID             int       `db:"id" json:"id"`
	UUID           string    `db:"uuid" json:"uuid"`
	Email          string    `db:"email" json:"email"`
	Name           string    `db:"name" json:"name"`
	UnsubscribedAt time.Time `db:"unsubscribed_at" json:"unsubscribed_at"`
}

// Campaigns represents a slice of Campaigns.
type Campaigns []Campaign

// Template represents a reusable e-mail template.
type Template struct {
	Base

	Name string `db:"name" json:"name"`
	// Subject is only for type=tx.
	Subject    string      `db:"subject" json:"subject"`
	Type       string      `db:"type" json:"type"`
	Body       string      `db:"body" json:"body,omitempty"`
	BodySource null.String `db:"body_source" json:"body_source,omitempty"`
	IsDefault  bool        `db:"is_default" json:"is_default"`

	// Only relevant to tx (transactional) templates.
	SubjectTpl *txttpl.Template   `json:"-"`
	Tpl        *template.Template `json:"-"`
}

// Bounce represents a single bounce event.
type Bounce struct {
	ID        int             `db:"id" json:"id"`
	Type      string          `db:"type" json:"type"`
	Source    string          `db:"source" json:"source"`
	Meta      json.RawMessage `db:"meta" json:"meta"`
	CreatedAt time.Time       `db:"created_at" json:"created_at"`

	// One of these should be provided.
	Email            string `db:"email" json:"email,omitempty"`
	SubscriberUUID   string `db:"subscriber_uuid" json:"subscriber_uuid,omitempty"`
	SubscriberID     int    `db:"subscriber_id" json:"subscriber_id,omitempty"`
	SubscriberStatus string `db:"subscriber_status" json:"subscriber_status"`

	CampaignUUID string           `db:"campaign_uuid" json:"campaign_uuid,omitempty"`
	Campaign     *json.RawMessage `db:"campaign" json:"campaign"`

	// Pseudofield for getting the total number of bounces
	// in searches and queries.
	Total int `db:"total" json:"-"`
}

// WebhookLog represents a logged webhook request.
type WebhookLog struct {
	ID             int64           `db:"id" json:"id"`
	WebhookType    string          `db:"webhook_type" json:"webhook_type"`
	EventType      null.String     `db:"event_type" json:"event_type"`
	RequestHeaders json.RawMessage `db:"request_headers" json:"request_headers"`
	RequestBody    string          `db:"request_body" json:"request_body"`
	ResponseStatus int             `db:"response_status" json:"response_status"`
	ResponseBody   null.String     `db:"response_body" json:"response_body"`
	Processed      bool            `db:"processed" json:"processed"`
	ErrorMessage   null.String     `db:"error_message" json:"error_message"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`

	// Pseudofield for getting the total count in queries.
	Total int `db:"total" json:"-"`
}

// ActivityEvent represents a unified activity event from various sources
// (email engagement, site events, etc.) for the activity feed.
type ActivityEvent struct {
	ID               int64           `db:"id" json:"id"`
	EventType        string          `db:"event_type" json:"event_type"`
	SubscriberID     null.Int        `db:"subscriber_id" json:"subscriber_id"`
	SubscriberEmail  string          `db:"subscriber_email" json:"subscriber_email"`
	SubscriberName   string          `db:"subscriber_name" json:"subscriber_name"`
	SubscriberUUID   string          `db:"subscriber_uuid" json:"subscriber_uuid"`
	EventDescription string          `db:"event_description" json:"event_description"`
	CampaignName     null.String     `db:"campaign_name" json:"campaign_name"`
	CampaignID       null.Int        `db:"campaign_id" json:"campaign_id"`
	PageURL          null.String     `db:"page_url" json:"page_url"`
	Properties       json.RawMessage `db:"properties" json:"properties"`
	CreatedAt        time.Time       `db:"created_at" json:"created_at"`

	// Pseudofield for getting the total count in queries.
	Total int `db:"total" json:"-"`
}

// PurchaseAttribution represents a purchase attributed to a campaign.
type PurchaseAttribution struct {
	ID             int64           `db:"id" json:"id"`
	CampaignID     null.Int        `db:"campaign_id" json:"campaign_id"`
	SubscriberID   null.Int        `db:"subscriber_id" json:"subscriber_id"`
	OrderID        string          `db:"order_id" json:"order_id"`
	OrderNumber    null.String     `db:"order_number" json:"order_number"`
	CustomerEmail  string          `db:"customer_email" json:"customer_email"`
	TotalPrice     float64         `db:"total_price" json:"total_price"`
	Currency       null.String     `db:"currency" json:"currency"`
	AttributedVia  null.String     `db:"attributed_via" json:"attributed_via"`
	Confidence     null.String     `db:"confidence" json:"confidence"`
	ShopifyData    json.RawMessage `db:"shopify_data" json:"shopify_data"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`

	// Pseudofield for getting the total count in queries.
	Total int `db:"total" json:"-"`
}

// CampaignPurchaseStats represents aggregate purchase statistics for a campaign.
type CampaignPurchaseStats struct {
	TotalPurchases int     `db:"total_purchases" json:"total_purchases"`
	TotalRevenue   float64 `db:"total_revenue" json:"total_revenue"`
	AvgOrderValue  float64 `db:"avg_order_value" json:"avg_order_value"`
	Currency       string  `db:"currency" json:"currency"`
}

// CampaignsPerformanceSummary represents aggregate performance metrics for all campaigns.
type CampaignsPerformanceSummary struct {
	AvgOpenRate  float64 `db:"avg_open_rate" json:"avgOpenRate"`
	AvgClickRate float64 `db:"avg_click_rate" json:"avgClickRate"`
	TotalSent    int     `db:"total_sent" json:"totalSent"`
	TotalOrders  int     `db:"total_orders" json:"totalOrders"`
	TotalRevenue float64 `db:"total_revenue" json:"totalRevenue"`
	OrderRate    float64 `db:"order_rate" json:"orderRate"`
	ErrorRate    float64 `db:"error_rate" json:"errorRate"`
}

// CampaignPurchaseStatsListItem represents purchase stats for a single campaign in a list.
type CampaignPurchaseStatsListItem struct {
	CampaignID   int     `db:"campaign_id" json:"campaign_id"`
	TotalOrders  int     `db:"total_orders" json:"total_orders"`
	TotalRevenue float64 `db:"total_revenue" json:"total_revenue"`
	Currency     string  `db:"currency" json:"currency"`
}

// Message is the message pushed to a Messenger.
type Message struct {
	From        string
	To          []string
	Subject     string
	ContentType string
	Body        []byte
	AltBody     []byte
	Headers     textproto.MIMEHeader
	Attachments []Attachment

	Subscriber Subscriber

	// Campaign is generally the same instance for a large number of subscribers.
	Campaign *Campaign

	// Messenger is the messenger backend to use: email|postback.
	Messenger string
}

// Attachment represents a file or blob attachment that can be
// sent along with a message by a Messenger.
type Attachment struct {
	Name    string
	Header  textproto.MIMEHeader
	Content []byte
}

// TxMessage represents an e-mail campaign.
type TxMessage struct {
	SubscriberEmails []string `json:"subscriber_emails"`
	SubscriberIDs    []int    `json:"subscriber_ids"`

	// Deprecated.
	SubscriberEmail string `json:"subscriber_email"`
	SubscriberID    int    `json:"subscriber_id"`

	TemplateID  int            `json:"template_id"`
	Data        map[string]any `json:"data"`
	FromEmail   string         `json:"from_email"`
	Headers     Headers        `json:"headers"`
	ContentType string         `json:"content_type"`
	Messenger   string         `json:"messenger"`
	Subject     string         `json:"subject"`

	// File attachments added from multi-part form data.
	Attachments []Attachment `json:"-"`

	Body       []byte             `json:"-"`
	Tpl        *template.Template `json:"-"`
	SubjectTpl *txttpl.Template   `json:"-"`
}

// markdown is a global instance of Markdown parser and renderer.
var markdown = goldmark.New(
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
	goldmark.WithRendererOptions(
		html.WithXHTML(),
		html.WithUnsafe(),
	),
	goldmark.WithExtensions(
		extension.Table,
		extension.Strikethrough,
		extension.TaskList,
		extension.NewTypographer(
			extension.WithTypographicSubstitutions(extension.TypographicSubstitutions{
				extension.LeftDoubleQuote:  []byte(`"`),
				extension.RightDoubleQuote: []byte(`"`),
			}),
		),
	),
)

// GetIDs returns the list of subscriber IDs.
func (subs Subscribers) GetIDs() []int {
	IDs := make([]int, len(subs))
	for i, c := range subs {
		IDs[i] = c.ID
	}

	return IDs
}

// LoadLists lazy loads the lists for all the subscribers
// in the Subscribers slice and attaches them to their []Lists property.
func (subs Subscribers) LoadLists(stmt *sqlx.Stmt) error {
	var sl []subLists
	err := stmt.Select(&sl, pq.Array(subs.GetIDs()))
	if err != nil {
		return err
	}

	if len(subs) != len(sl) {
		return errors.New("campaign stats count does not match")
	}

	for i, s := range sl {
		if s.SubscriberID == subs[i].ID {
			subs[i].Lists = s.Lists
		}
	}

	return nil
}

// Value returns the JSON marshalled SubscriberAttribs.
func (s JSON) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Scan unmarshals JSONB from the DB.
func (s JSON) Scan(src any) error {
	if src == nil {
		s = make(JSON)
		return nil
	}

	if data, ok := src.([]byte); ok {
		return json.Unmarshal(data, &s)
	}
	return fmt.Errorf("could not not decode type %T -> %T", src, s)
}

// Scan unmarshals JSONB from the DB.
func (s StringIntMap) Scan(src any) error {
	if src == nil {
		s = make(StringIntMap)
		return nil
	}

	if data, ok := src.([]byte); ok {
		return json.Unmarshal(data, &s)
	}
	return fmt.Errorf("could not not decode type %T -> %T", src, s)
}

// GetIDs returns the list of campaign IDs.
func (camps Campaigns) GetIDs() []int {
	IDs := make([]int, len(camps))
	for i, c := range camps {
		IDs[i] = c.ID
	}

	return IDs
}

// LoadStats lazy loads campaign stats onto a list of campaigns.
func (camps Campaigns) LoadStats(stmt *sqlx.Stmt) error {
	var meta []CampaignMeta
	if err := stmt.Select(&meta, pq.Array(camps.GetIDs())); err != nil {
		return err
	}

	if len(camps) != len(meta) {
		return errors.New("campaign stats count does not match")
	}

	for i, c := range meta {
		if c.CampaignID == camps[i].ID {
			camps[i].Lists = c.Lists
			camps[i].Views = c.Views
			camps[i].Clicks = c.Clicks
			camps[i].Bounces = c.Bounces
			camps[i].Media = c.Media
			camps[i].Delivered = c.Delivered
			camps[i].AzureSent = c.AzureSent
			camps[i].Segments = c.Segments
		}
	}

	return nil
}

// CompileTemplate compiles a campaign body template into its base
// template and sets the resultant template to Campaign.Tpl.
func (c *Campaign) CompileTemplate(f template.FuncMap) error {
	// If the subject line has a template string, compile it.
	if strings.Contains(c.Subject, "{{") {
		subj := c.Subject
		for _, r := range regTplFuncs {
			subj = r.regExp.ReplaceAllString(subj, r.replace)
		}

		var txtFuncs map[string]any = f
		subjTpl, err := txttpl.New(ContentTpl).Funcs(txtFuncs).Parse(subj)
		if err != nil {
			return fmt.Errorf("error compiling subject: %v", err)
		}
		c.SubjectTpl = subjTpl
	}

	// Compile the base template.
	body := c.TemplateBody

	if body == "" || c.ContentType == CampaignContentTypeVisual {
		body = `{{ template "content" . }}`
	}

	for _, r := range regTplFuncs {
		body = r.regExp.ReplaceAllString(body, r.replace)
	}

	baseTPL, err := template.New(BaseTpl).Funcs(f).Parse(body)
	if err != nil {
		return fmt.Errorf("error compiling base template: %v", err)
	}

	// If the format is markdown, convert Markdown to HTML.
	if c.ContentType == CampaignContentTypeMarkdown {
		var b bytes.Buffer
		if err := markdown.Convert([]byte(c.Body), &b); err != nil {
			return err
		}
		body = b.String()
	} else {
		body = c.Body
	}

	// Compile the campaign message.
	for _, r := range regTplFuncs {
		body = r.regExp.ReplaceAllString(body, r.replace)
	}

	msgTpl, err := template.New(ContentTpl).Funcs(f).Parse(body)
	if err != nil {
		return fmt.Errorf("error compiling message: %v", err)
	}

	out, err := baseTPL.AddParseTree(ContentTpl, msgTpl.Tree)
	if err != nil {
		return fmt.Errorf("error inserting child template: %v", err)
	}
	c.Tpl = out

	if strings.Contains(c.AltBody.String, "{{") {
		b := c.AltBody.String
		for _, r := range regTplFuncs {
			b = r.regExp.ReplaceAllString(b, r.replace)
		}
		bTpl, err := template.New(ContentTpl).Funcs(f).Parse(b)
		if err != nil {
			return fmt.Errorf("error compiling alt plaintext message: %v", err)
		}
		c.AltBodyTpl = bTpl
	}

	return nil
}

// ConvertContent converts a campaign's body from one format to another,
// for example, Markdown to HTML.
func (c *Campaign) ConvertContent(from, to string) (string, error) {
	body := c.Body
	for _, r := range regTplFuncs {
		body = r.regExp.ReplaceAllString(body, r.replace)
	}

	// If the format is markdown, convert Markdown to HTML.
	var out string
	if from == CampaignContentTypeMarkdown &&
		(to == CampaignContentTypeHTML || to == CampaignContentTypeRichtext) {
		var b bytes.Buffer
		if err := markdown.Convert([]byte(c.Body), &b); err != nil {
			return out, err
		}
		out = b.String()
	} else {
		return out, errors.New("unknown formats to convert")
	}

	return out, nil
}

// Compile compiles a template body and subject (only for tx templates) and
// caches the templat references to be executed later.
func (t *Template) Compile(f template.FuncMap) error {
	tpl, err := template.New(BaseTpl).Funcs(f).Parse(t.Body)
	if err != nil {
		return fmt.Errorf("error compiling transactional template: %v", err)
	}
	t.Tpl = tpl

	// If the subject line has a template string, compile it.
	if strings.Contains(t.Subject, "{{") {
		subj := t.Subject

		subjTpl, err := txttpl.New(BaseTpl).Funcs(txttpl.FuncMap(f)).Parse(subj)
		if err != nil {
			return fmt.Errorf("error compiling subject: %v", err)
		}
		t.SubjectTpl = subjTpl
	}

	return nil
}

func (m *TxMessage) Render(sub Subscriber, tpl *Template) error {
	data := struct {
		Subscriber Subscriber
		Tx         *TxMessage
	}{sub, m}

	// Render the body.
	b := bytes.Buffer{}
	if err := tpl.Tpl.ExecuteTemplate(&b, BaseTpl, data); err != nil {
		return err
	}
	m.Body = make([]byte, b.Len())
	copy(m.Body, b.Bytes())
	b.Reset()

	// Was a subject provided in the message?
	var (
		subjTpl *txttpl.Template
		subject = m.Subject
	)
	if subject != "" {
		if strings.Contains(m.Subject, "{{") {
			// If the subject has a template string, render that.
			s, err := txttpl.New(BaseTpl).Funcs(txttpl.FuncMap(nil)).Parse(m.Subject)
			if err != nil {
				return fmt.Errorf("error compiling subject: %v", err)
			}
			subjTpl = s
		}
	} else {
		// Use the subject from the template.
		subject = tpl.Subject
		subjTpl = tpl.SubjectTpl
	}

	// If the subject is also a template, render that.
	if subjTpl != nil {
		if err := subjTpl.ExecuteTemplate(&b, BaseTpl, data); err != nil {
			return err
		}
		m.Subject = b.String()
		b.Reset()
	} else {
		m.Subject = subject
	}

	return nil
}

// FirstName splits the name by spaces and returns the first chunk
// of the name that's greater than 2 characters in length, assuming
// that it is the subscriber's first name.
func (s Subscriber) FirstName() string {
	for _, s := range strings.Split(s.Name, " ") {
		if len(s) > 2 {
			return s
		}
	}

	return s.Name
}

// LastName splits the name by spaces and returns the last chunk
// of the name that's greater than 2 characters in length, assuming
// that it is the subscriber's last name.
func (s Subscriber) LastName() string {
	chunks := strings.Split(s.Name, " ")
	for i := len(chunks) - 1; i >= 0; i-- {
		chunk := chunks[i]
		if len(chunk) > 2 {
			return chunk
		}
	}

	return s.Name
}

// Scan implements the sql.Scanner interface.
func (h *Headers) Scan(src any) error {
	var b []byte
	switch src := src.(type) {
	case []byte:
		b = src
	case string:
		b = []byte(src)
	case nil:
		return nil
	}

	if err := json.Unmarshal(b, h); err != nil {
		return err
	}

	return nil
}

// Value implements the driver.Valuer interface.
func (h Headers) Value() (driver.Value, error) {
	if h == nil {
		return nil, nil
	}

	if n := len(h); n > 0 {
		b, err := json.Marshal(h)
		if err != nil {
			return nil, err
		}

		return b, nil
	}

	return "[]", nil
}

// AzureDeliveryEvent represents a delivery status event from Azure Event Grid.
type AzureDeliveryEvent struct {
	ID                    int64     `db:"id" json:"id"`
	AzureMessageID        string    `db:"azure_message_id" json:"azure_message_id"`
	CampaignID            int       `db:"campaign_id" json:"campaign_id"`
	SubscriberID          int       `db:"subscriber_id" json:"subscriber_id"`
	Status                string    `db:"status" json:"status"`
	StatusReason          string    `db:"status_reason" json:"status_reason"`
	DeliveryStatusDetails string    `db:"delivery_status_details" json:"delivery_status_details"`
	EventTimestamp        time.Time `db:"event_timestamp" json:"event_timestamp"`
	CreatedAt             time.Time `db:"created_at" json:"created_at"`
}

// AzureEngagementEvent represents an engagement event (view/click) from Azure Event Grid.
type AzureEngagementEvent struct {
	ID                int64     `db:"id" json:"id"`
	AzureMessageID    string    `db:"azure_message_id" json:"azure_message_id"`
	CampaignID        int       `db:"campaign_id" json:"campaign_id"`
	SubscriberID      int       `db:"subscriber_id" json:"subscriber_id"`
	EngagementType    string    `db:"engagement_type" json:"engagement_type"`
	EngagementContext string    `db:"engagement_context" json:"engagement_context"`
	UserAgent         string    `db:"user_agent" json:"user_agent"`
	EventTimestamp    time.Time `db:"event_timestamp" json:"event_timestamp"`
	CreatedAt         time.Time `db:"created_at" json:"created_at"`
}

// SignUpForm represents a sign-up form with Klaviyo-like features.
type SignUpForm struct {
	Base

	UUID        string         `db:"uuid" json:"uuid"`
	Name        string         `db:"name" json:"name"`
	Description string         `db:"description" json:"description"`
	FormType    string         `db:"form_type" json:"form_type"`
	Status      string         `db:"status" json:"status"`
	BodySource  JSON           `db:"body_source" json:"body_source"`
	CustomCSS   string         `db:"custom_css" json:"custom_css"`
	Steps       JSON           `db:"steps" json:"steps"`
	Settings    JSON           `db:"settings" json:"settings"`
	Targeting   JSON           `db:"targeting" json:"targeting"`
	Triggers    JSON           `db:"triggers" json:"triggers"`
	Frequency   JSON           `db:"frequency" json:"frequency"`
	ListIDs     pq.Int64Array  `db:"list_ids" json:"list_ids"`
	CouponConfig JSON          `db:"coupon_config" json:"coupon_config"`

	// Pseudofield for pagination total
	Total int `db:"total" json:"-"`
}

// SignUpForms is a slice of SignUpForm.
type SignUpForms []SignUpForm

// FormSubmission represents a form submission with all captured data.
type FormSubmission struct {
	ID           int64          `db:"id" json:"id"`
	FormID       int            `db:"form_id" json:"form_id"`
	SubscriberID null.Int       `db:"subscriber_id" json:"subscriber_id"`
	Email        string         `db:"email" json:"email"`
	Data         JSON           `db:"data" json:"data"`
	PageURL      null.String    `db:"page_url" json:"page_url"`
	Referrer     null.String    `db:"referrer" json:"referrer"`
	UTMSource    null.String    `db:"utm_source" json:"utm_source"`
	UTMMedium    null.String    `db:"utm_medium" json:"utm_medium"`
	UTMCampaign  null.String    `db:"utm_campaign" json:"utm_campaign"`
	UTMTerm      null.String    `db:"utm_term" json:"utm_term"`
	UTMContent   null.String    `db:"utm_content" json:"utm_content"`
	DeviceType   null.String    `db:"device_type" json:"device_type"`
	UserAgent    null.String    `db:"user_agent" json:"user_agent"`
	IPAddress    null.String    `db:"ip_address" json:"ip_address"`
	Country      null.String    `db:"country" json:"country"`
	CouponCode   null.String    `db:"coupon_code" json:"coupon_code"`
	CreatedAt    time.Time      `db:"created_at" json:"created_at"`

	// Pseudofield for pagination total
	Total int `db:"total" json:"-"`
}

// FormImpression represents a form impression for analytics.
type FormImpression struct {
	ID          int64       `db:"id" json:"id"`
	FormID      int         `db:"form_id" json:"form_id"`
	SessionID   string      `db:"session_id" json:"session_id"`
	VisitorID   null.String `db:"visitor_id" json:"visitor_id"`
	PageURL     null.String `db:"page_url" json:"page_url"`
	DeviceType  null.String `db:"device_type" json:"device_type"`
	Country     null.String `db:"country" json:"country"`
	ShownAt     time.Time   `db:"shown_at" json:"shown_at"`
	ClosedAt    null.Time   `db:"closed_at" json:"closed_at"`
	SubmittedAt null.Time   `db:"submitted_at" json:"submitted_at"`
	StepReached int         `db:"step_reached" json:"step_reached"`
}

// FormCouponCode represents a unique coupon code for a form.
type FormCouponCode struct {
	ID          int         `db:"id" json:"id"`
	FormID      int         `db:"form_id" json:"form_id"`
	Code        string      `db:"code" json:"code"`
	Used        bool        `db:"used" json:"used"`
	UsedByEmail null.String `db:"used_by_email" json:"used_by_email"`
	UsedAt      null.Time   `db:"used_at" json:"used_at"`
	CreatedAt   time.Time   `db:"created_at" json:"created_at"`
}

// FormAnalytics represents aggregated analytics for a form.
type FormAnalytics struct {
	FormID         int     `json:"form_id"`
	Impressions    int     `json:"impressions"`
	Submissions    int     `json:"submissions"`
	ConversionRate float64 `json:"conversion_rate"`
	AvgStepReached float64 `json:"avg_step_reached"`
}

// FormAnalyticsTimeSeries represents time-series analytics data.
type FormAnalyticsTimeSeries struct {
	Date        time.Time `db:"date" json:"date"`
	Impressions int       `db:"impressions" json:"impressions"`
	Submissions int       `db:"submissions" json:"submissions"`
}
