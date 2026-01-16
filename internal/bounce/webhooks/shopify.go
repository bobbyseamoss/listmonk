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
	webhookSecrets []string // Multiple secrets to try (client_secret, webhook_secret)
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

// ShopifyOrderFull represents the complete order data from a Shopify webhook.
type ShopifyOrderFull struct {
	ID                int64              `json:"id"`
	Name              string             `json:"name"`
	OrderNumber       int                `json:"order_number"`
	Email             string             `json:"email"`
	CreatedAt         string             `json:"created_at"`
	ProcessedAt       string             `json:"processed_at"`
	TotalPrice        string             `json:"total_price"`
	SubtotalPrice     string             `json:"subtotal_price"`
	TotalTax          string             `json:"total_tax"`
	TotalDiscounts    string             `json:"total_discounts"`
	Currency          string             `json:"currency"`
	FinancialStatus   string             `json:"financial_status"`
	FulfillmentStatus string             `json:"fulfillment_status"`
	Tags              string             `json:"tags"`
	Note              string             `json:"note"`
	LandingSite       string             `json:"landing_site"`
	LineItems         []ShopifyLineItem  `json:"line_items"`
	RawJSON           []byte             `json:"-"`
}

// ShopifyLineItem represents a line item in a Shopify order.
type ShopifyLineItem struct {
	ID             int64                    `json:"id"`
	ProductID      int64                    `json:"product_id"`
	VariantID      int64                    `json:"variant_id"`
	Title          string                   `json:"title"`
	VariantTitle   string                   `json:"variant_title"`
	SKU            string                   `json:"sku"`
	Quantity       int                      `json:"quantity"`
	Price          string                   `json:"price"`
	TotalDiscount  string                   `json:"total_discount"`
	ProductType    string                   `json:"product_type,omitempty"`
	Vendor         string                   `json:"vendor"`
	Properties     []ShopifyLineItemProp    `json:"properties"`
}

// ShopifyLineItemProp represents a custom property on a line item.
type ShopifyLineItemProp struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// NewShopify creates a new Shopify webhook handler with one or more secrets.
// For Partner app webhooks, pass the client_secret first (primary).
// Multiple secrets are tried in order during verification.
func NewShopify(secrets ...string) *Shopify {
	// Filter out empty secrets
	validSecrets := make([]string, 0, len(secrets))
	for _, s := range secrets {
		if s != "" {
			validSecrets = append(validSecrets, s)
		}
	}
	return &Shopify{webhookSecrets: validSecrets}
}

// VerifyWebhook verifies the HMAC signature of a Shopify webhook.
// The signature is in the X-Shopify-Hmac-Sha256 header and is a base64-encoded
// HMAC-SHA256 hash of the raw request body.
// It tries all configured secrets in order and succeeds if any one matches.
func (s *Shopify) VerifyWebhook(hmacHeader string, body []byte) error {
	if len(s.webhookSecrets) == 0 {
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

	// Try each secret until one matches
	for _, secret := range s.webhookSecrets {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		computedMAC := mac.Sum(nil)

		if hmac.Equal(computedMAC, expectedMAC) {
			return nil // Success!
		}
	}

	return errors.New("HMAC verification failed")
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

// ProcessOrderFull parses a Shopify order webhook payload with complete details.
func (s *Shopify) ProcessOrderFull(body []byte) (*ShopifyOrderFull, error) {
	var order ShopifyOrderFull
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
