package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

// Shopify handles Shopify webhook verification and parsing.
type Shopify struct {
	webhookSecret string
}

// ShopifyOrder represents the relevant fields from a Shopify order webhook.
type ShopifyOrder struct {
	ID          int64   `json:"id"`
	OrderNumber int     `json:"order_number"`
	Email       string  `json:"email"`
	TotalPrice  string  `json:"total_price"`
	Currency    string  `json:"currency"`
	CreatedAt   string  `json:"created_at"`
	LandingSite string  `json:"landing_site"`
	RawJSON     []byte  `json:"-"`
}

// NewShopify creates a new Shopify webhook handler.
func NewShopify(secret string) *Shopify {
	return &Shopify{webhookSecret: secret}
}

// VerifyWebhook verifies the HMAC signature of a Shopify webhook.
// The signature is in the X-Shopify-Hmac-Sha256 header and is a base64-encoded
// HMAC-SHA256 hash of the raw request body.
func (s *Shopify) VerifyWebhook(hmacHeader string, body []byte) error {
	if s.webhookSecret == "" {
		return errors.New("webhook secret not configured")
	}

	if hmacHeader == "" {
		return errors.New("missing HMAC header")
	}

	// Decode the base64-encoded HMAC from the header
	expectedMAC, err := base64.StdEncoding.DecodeString(hmacHeader)
	if err != nil {
		return fmt.Errorf("error decoding HMAC: %v", err)
	}

	// Compute the HMAC of the body using the webhook secret
	mac := hmac.New(sha256.New, []byte(s.webhookSecret))
	mac.Write(body)
	computedMAC := mac.Sum(nil)

	// Compare the computed HMAC with the expected HMAC
	if !hmac.Equal(computedMAC, expectedMAC) {
		return errors.New("HMAC verification failed")
	}

	return nil
}

// ProcessOrder parses a Shopify order webhook payload and extracts relevant data.
func (s *Shopify) ProcessOrder(body []byte) (*ShopifyOrder, error) {
	var order ShopifyOrder
	if err := json.Unmarshal(body, &order); err != nil {
		return nil, fmt.Errorf("error parsing order JSON: %v", err)
	}

	// Store the raw JSON for later reference
	order.RawJSON = body

	// Validate required fields
	if order.Email == "" {
		return nil, errors.New("order missing email address")
	}

	if order.ID == 0 {
		return nil, errors.New("order missing ID")
	}

	return &order, nil
}
