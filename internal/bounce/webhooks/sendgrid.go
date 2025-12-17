package webhooks

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/knadh/listmonk/models"
)

// SendgridEvent represents a single event from SendGrid webhook
type SendgridEvent struct {
	Email                string `json:"email"`
	Timestamp            int64  `json:"timestamp"`
	Event                string `json:"event"`
	BounceClassification string `json:"bounce_classification"`
	Reason               string `json:"reason"`
	Type                 string `json:"type"`
	SGMessageID          string `json:"sg_message_id"`
	SMTPID               string `json:"smtp-id"`
	URL                  string `json:"url"`       // For click events
	UserAgent            string `json:"useragent"` // For open/click events
	IP                   string `json:"ip"`

	// SendGrid flattens all X-headers and adds them to the event notification.
	CampaignUUID   string `json:"XListmonkCampaign"`
	SubscriberUUID string `json:"XListmonkSubscriber"`
}

// SendgridEngagement represents an engagement event (open, click)
type SendgridEngagement struct {
	Email          string
	Event          string // "open" or "click"
	URL            string // For clicks
	UserAgent      string
	IP             string
	Timestamp      time.Time
	CampaignUUID   string
	SubscriberUUID string
}

// SendgridResult contains the processed results from SendGrid webhooks
type SendgridResult struct {
	Bounces     []models.Bounce
	Engagements []SendgridEngagement
	Events      []SendgridEvent // All raw events for logging
}

// Sendgrid handles Sendgrid webhook notifications
type Sendgrid struct{}

// NewSendgrid returns a new Sendgrid instance.
func NewSendgrid(key string) (*Sendgrid, error) {
	// Key is ignored - no signature validation
	return &Sendgrid{}, nil
}

// ProcessEvents processes all SendGrid webhook events and returns bounces, engagements, and raw events
func (s *Sendgrid) ProcessEvents(b []byte) (*SendgridResult, error) {
	var events []SendgridEvent
	if err := json.Unmarshal(b, &events); err != nil {
		return nil, fmt.Errorf("error unmarshalling Sendgrid notification: %v", err)
	}

	result := &SendgridResult{
		Bounces:     make([]models.Bounce, 0),
		Engagements: make([]SendgridEngagement, 0),
		Events:      events,
	}

	for _, e := range events {
		tstamp := time.Unix(e.Timestamp, 0)

		switch e.Event {
		case "bounce":
			typ := models.BounceTypeHard
			if e.BounceClassification == "technical" || e.BounceClassification == "content" {
				typ = models.BounceTypeSoft
			}
			result.Bounces = append(result.Bounces, models.Bounce{
				CampaignUUID:   e.CampaignUUID,
				SubscriberUUID: e.SubscriberUUID,
				Email:          strings.ToLower(e.Email),
				Type:           typ,
				Meta:           json.RawMessage(b),
				Source:         "sendgrid",
				CreatedAt:      tstamp,
			})

		case "dropped":
			typ := models.BounceTypeHard
			if e.Type == "expired" {
				typ = models.BounceTypeSoft
			}
			result.Bounces = append(result.Bounces, models.Bounce{
				CampaignUUID:   e.CampaignUUID,
				SubscriberUUID: e.SubscriberUUID,
				Email:          strings.ToLower(e.Email),
				Type:           typ,
				Meta:           json.RawMessage(b),
				Source:         "sendgrid",
				CreatedAt:      tstamp,
			})

		case "spamreport", "unsubscribe":
			result.Bounces = append(result.Bounces, models.Bounce{
				CampaignUUID:   e.CampaignUUID,
				SubscriberUUID: e.SubscriberUUID,
				Email:          strings.ToLower(e.Email),
				Type:           models.BounceTypeComplaint,
				Meta:           json.RawMessage(b),
				Source:         "sendgrid",
				CreatedAt:      tstamp,
			})

		case "blocked":
			result.Bounces = append(result.Bounces, models.Bounce{
				CampaignUUID:   e.CampaignUUID,
				SubscriberUUID: e.SubscriberUUID,
				Email:          strings.ToLower(e.Email),
				Type:           models.BounceTypeHard,
				Meta:           json.RawMessage(b),
				Source:         "sendgrid",
				CreatedAt:      tstamp,
			})

		case "deferred":
			result.Bounces = append(result.Bounces, models.Bounce{
				CampaignUUID:   e.CampaignUUID,
				SubscriberUUID: e.SubscriberUUID,
				Email:          strings.ToLower(e.Email),
				Type:           models.BounceTypeSoft,
				Meta:           json.RawMessage(b),
				Source:         "sendgrid",
				CreatedAt:      tstamp,
			})

		case "open":
			result.Engagements = append(result.Engagements, SendgridEngagement{
				Email:          strings.ToLower(e.Email),
				Event:          "open",
				UserAgent:      e.UserAgent,
				IP:             e.IP,
				Timestamp:      tstamp,
				CampaignUUID:   e.CampaignUUID,
				SubscriberUUID: e.SubscriberUUID,
			})

		case "click":
			result.Engagements = append(result.Engagements, SendgridEngagement{
				Email:          strings.ToLower(e.Email),
				Event:          "click",
				URL:            e.URL,
				UserAgent:      e.UserAgent,
				IP:             e.IP,
				Timestamp:      tstamp,
				CampaignUUID:   e.CampaignUUID,
				SubscriberUUID: e.SubscriberUUID,
			})

		// delivered, processed, etc. are logged but no action needed
		}
	}

	return result, nil
}

// ProcessBounce is kept for backwards compatibility but now just wraps ProcessEvents
func (s *Sendgrid) ProcessBounce(sig, timestamp string, b []byte) ([]models.Bounce, error) {
	result, err := s.ProcessEvents(b)
	if err != nil {
		return nil, err
	}
	return result.Bounces, nil
}
