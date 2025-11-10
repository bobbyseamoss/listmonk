package webhooks

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/knadh/listmonk/models"
)

// Azure Event Grid event type constants
const (
	EventTypeValidation = "Microsoft.EventGrid.SubscriptionValidationEvent"
	EventTypeDelivery   = "Microsoft.Communication.EmailDeliveryReportReceived"
	EventTypeEngagement = "Microsoft.Communication.EmailEngagementTrackingReportReceived"
)

// Azure delivery status constants
const (
	StatusDelivered    = "Delivered"
	StatusSuppressed   = "Suppressed"
	StatusBounced      = "Bounced"
	StatusQuarantined  = "Quarantined"
	StatusFilteredSpam = "FilteredSpam"
	StatusExpanded     = "Expanded"
	StatusFailed       = "Failed"
)

// AzureEventGridEvent represents the envelope for all Azure Event Grid events
type AzureEventGridEvent struct {
	ID              string                 `json:"id"`
	Topic           string                 `json:"topic"`
	Subject         string                 `json:"subject"`
	EventType       string                 `json:"eventType"`
	EventTime       string                 `json:"eventTime"`
	Data            map[string]interface{} `json:"data"`
	DataVersion     string                 `json:"dataVersion"`
	MetadataVersion string                 `json:"metadataVersion"`
}

// AzureValidationEvent represents the subscription validation event
type AzureValidationEvent struct {
	ID              string `json:"id"`
	Topic           string `json:"topic"`
	Subject         string `json:"subject"`
	EventType       string `json:"eventType"`
	EventTime       string `json:"eventTime"`
	Data            struct {
		ValidationCode string `json:"validationCode"`
		ValidationURL  string `json:"validationUrl"`
	} `json:"data"`
	DataVersion     string `json:"dataVersion"`
	MetadataVersion string `json:"metadataVersion"`
}

// AzureValidationResponse is sent back for subscription validation
type AzureValidationResponse struct {
	ValidationResponse string `json:"validationResponse"`
}

// AzureDeliveryData represents the delivery report data
type AzureDeliveryData struct {
	Sender                   string                 `json:"sender"`
	Recipient                string                 `json:"recipient"`
	MessageID                string                 `json:"messageId"`
	InternetMessageID        string                 `json:"internetMessageId"`
	Status                   string                 `json:"status"`
	DeliveryStatusDetails    map[string]interface{} `json:"deliveryStatusDetails"`
	DeliveryAttemptTimeStamp string                 `json:"deliveryAttemptTimeStamp"`
}

// AzureEngagementData represents the engagement tracking data
type AzureEngagementData struct {
	Sender              string `json:"sender"`
	Recipient           string `json:"recipient"` // May be empty for multi-recipient
	MessageID           string `json:"messageId"`
	InternetMessageID   string `json:"internetMessageId"`
	UserActionTimeStamp string `json:"userActionTimeStamp"`
	EngagementContext   string `json:"engagementContext"` // URL for clicks
	UserAgent           string `json:"userAgent"`
	EngagementType      string `json:"engagementType"` // "view" or "click"
}

// Azure handles Azure Event Grid webhook events
type Azure struct{}

// NewAzure returns a new Azure instance
func NewAzure() *Azure {
	return &Azure{}
}

// ProcessValidation handles the subscription validation handshake
// Azure Event Grid sends a validation event when creating a new subscription
// We must respond with the validation code to confirm the endpoint
func (a *Azure) ProcessValidation(b []byte) (AzureValidationResponse, error) {
	var events []AzureValidationEvent
	if err := json.Unmarshal(b, &events); err != nil {
		return AzureValidationResponse{}, fmt.Errorf("error unmarshalling validation event: %v", err)
	}

	if len(events) == 0 {
		return AzureValidationResponse{}, errors.New("no events in validation payload")
	}

	event := events[0]
	if event.EventType != EventTypeValidation {
		return AzureValidationResponse{}, fmt.Errorf("expected validation event, got: %s", event.EventType)
	}

	return AzureValidationResponse{
		ValidationResponse: event.Data.ValidationCode,
	}, nil
}

// ProcessDeliveryEvent processes a delivery report event and returns a Bounce object
// Azure sends delivery reports for all terminal states: Delivered, Bounced, Failed, etc.
// We convert failed deliveries into bounce records
func (a *Azure) ProcessDeliveryEvent(rawEvent map[string]interface{}) (models.Bounce, bool, error) {
	var bounce models.Bounce

	// Marshal back to JSON for storage in Meta field
	dataBytes, err := json.Marshal(rawEvent)
	if err != nil {
		return bounce, false, fmt.Errorf("error marshalling event data: %v", err)
	}

	// Parse into structured data
	var data AzureDeliveryData
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		return bounce, false, fmt.Errorf("error unmarshalling delivery data: %v", err)
	}

	// Determine if this is a bounce event and what type
	shouldRecord := false
	bounceType := models.BounceTypeHard

	switch data.Status {
	case StatusBounced, StatusFailed, StatusSuppressed:
		// Hard bounces - permanent delivery failures
		shouldRecord = true
		bounceType = models.BounceTypeHard
	case StatusQuarantined:
		// Soft bounce - temporary issue (spam filter, etc.)
		shouldRecord = true
		bounceType = models.BounceTypeSoft
	case StatusFilteredSpam:
		// Complaint - recipient marked as spam
		shouldRecord = true
		bounceType = models.BounceTypeComplaint
	case StatusDelivered:
		// Success - no bounce record needed
		return bounce, false, nil
	case StatusExpanded:
		// Distribution list expansion - informational only
		return bounce, false, nil
	default:
		return bounce, false, fmt.Errorf("unknown delivery status: %s", data.Status)
	}

	// Parse timestamp
	timestamp := time.Now()
	if data.DeliveryAttemptTimeStamp != "" {
		if t, err := time.Parse(time.RFC3339, data.DeliveryAttemptTimeStamp); err == nil {
			timestamp = t
		}
	}

	// Create bounce record
	// Note: CampaignUUID will be filled by the handler after DB lookup
	bounce = models.Bounce{
		Email:     strings.ToLower(data.Recipient),
		Type:      bounceType,
		Source:    "azure",
		Meta:      json.RawMessage(dataBytes),
		CreatedAt: timestamp,
	}

	return bounce, shouldRecord, nil
}

// ProcessEngagementEvent processes an engagement tracking event
// Azure sends engagement events for email opens (views) and link clicks
func (a *Azure) ProcessEngagementEvent(rawEvent map[string]interface{}) (AzureEngagementData, error) {
	dataBytes, err := json.Marshal(rawEvent)
	if err != nil {
		return AzureEngagementData{}, fmt.Errorf("error marshalling event data: %v", err)
	}

	var data AzureEngagementData
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		return AzureEngagementData{}, fmt.Errorf("error unmarshalling engagement data: %v", err)
	}

	return data, nil
}
