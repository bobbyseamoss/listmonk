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

// ShopifyCustomer represents a customer from Shopify webhook.
type ShopifyCustomer struct {
	ID                    int64                       `json:"id"`
	Email                 string                      `json:"email"`
	FirstName             string                      `json:"first_name"`
	LastName              string                      `json:"last_name"`
	Phone                 string                      `json:"phone"`
	Tags                  string                      `json:"tags"`
	Note                  string                      `json:"note"`
	OrdersCount           int                         `json:"orders_count"`
	TotalSpent            string                      `json:"total_spent"`
	Currency              string                      `json:"currency"`
	State                 string                      `json:"state"`
	VerifiedEmail         bool                        `json:"verified_email"`
	CreatedAt             string                      `json:"created_at"`
	UpdatedAt             string                      `json:"updated_at"`
	AcceptsMarketing      bool                        `json:"accepts_marketing"`
	EmailMarketingConsent ShopifyMarketingConsent     `json:"email_marketing_consent"`
	DefaultAddress        *ShopifyAddress             `json:"default_address"`
	Addresses             []ShopifyAddress            `json:"addresses"`
	RawJSON               []byte                      `json:"-"`
}

// ShopifyMarketingConsent represents email marketing consent from Shopify.
type ShopifyMarketingConsent struct {
	State            string `json:"state"`
	OptInLevel       string `json:"opt_in_level"`
	ConsentUpdatedAt string `json:"consent_updated_at"`
}

// ShopifyAddress represents a customer address from Shopify.
type ShopifyAddress struct {
	ID           int64  `json:"id"`
	CustomerID   int64  `json:"customer_id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Company      string `json:"company"`
	Address1     string `json:"address1"`
	Address2     string `json:"address2"`
	City         string `json:"city"`
	Province     string `json:"province"`
	ProvinceCode string `json:"province_code"`
	Country      string `json:"country"`
	CountryCode  string `json:"country_code"`
	Zip          string `json:"zip"`
	Phone        string `json:"phone"`
	Default      bool   `json:"default"`
}

// ProcessCustomer parses a Shopify customer webhook payload.
func (s *Shopify) ProcessCustomer(body []byte) (*ShopifyCustomer, error) {
	var customer ShopifyCustomer
	if err := json.Unmarshal(body, &customer); err != nil {
		return nil, fmt.Errorf("error parsing customer JSON: %v", err)
	}

	// Store the raw JSON for later reference
	customer.RawJSON = body

	// Validate required fields
	if customer.Email == "" {
		return nil, errors.New("customer missing email address")
	}

	if customer.ID == 0 {
		return nil, errors.New("customer missing ID")
	}

	return &customer, nil
}
