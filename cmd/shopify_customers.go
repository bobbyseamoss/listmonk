package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/knadh/listmonk/internal/bounce/webhooks"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// customerSyncState tracks the state of bulk customer sync operations.
type customerSyncState struct {
	sync.RWMutex
	InProgress   bool      `json:"in_progress"`
	StartedAt    time.Time `json:"started_at,omitempty"`
	TotalCount   int       `json:"total_count"`
	SyncedCount  int       `json:"synced_count"`
	SkippedCount int       `json:"skipped_count"`
	ErrorCount   int       `json:"error_count"`
	LastError    string    `json:"last_error,omitempty"`
	CompletedAt  time.Time `json:"completed_at,omitempty"`
}

var syncState = &customerSyncState{}

// GraphQL query for fetching customers
const customersQuery = `
query GetCustomers($cursor: String) {
    customers(first: 50, after: $cursor) {
        pageInfo {
            hasNextPage
            endCursor
        }
        edges {
            node {
                id
                email
                firstName
                lastName
                phone
                tags
                note
                ordersCount
                totalSpent
                emailMarketingConsent {
                    marketingState
                    marketingOptInLevel
                    consentUpdatedAt
                }
                defaultAddress {
                    address1
                    address2
                    city
                    province
                    provinceCode
                    country
                    countryCode
                    zip
                    phone
                }
                metafield(namespace: "custom", key: "birthday") {
                    value
                }
                createdAt
                updatedAt
            }
        }
    }
}
`

// GraphQL customer response structures
type customersResponse struct {
	Customers struct {
		PageInfo struct {
			HasNextPage bool   `json:"hasNextPage"`
			EndCursor   string `json:"endCursor"`
		} `json:"pageInfo"`
		Edges []struct {
			Node graphQLCustomer `json:"node"`
		} `json:"edges"`
	} `json:"customers"`
}

type graphQLCustomer struct {
	ID                    string `json:"id"`
	Email                 string `json:"email"`
	FirstName             string `json:"firstName"`
	LastName              string `json:"lastName"`
	Phone                 string `json:"phone"`
	Tags                  []string `json:"tags"`
	Note                  string `json:"note"`
	OrdersCount           string `json:"ordersCount"` // GraphQL returns this as string
	TotalSpent            string `json:"totalSpent"`
	EmailMarketingConsent *struct {
		MarketingState      string `json:"marketingState"`
		MarketingOptInLevel string `json:"marketingOptInLevel"`
		ConsentUpdatedAt    string `json:"consentUpdatedAt"`
	} `json:"emailMarketingConsent"`
	DefaultAddress *struct {
		Address1     string `json:"address1"`
		Address2     string `json:"address2"`
		City         string `json:"city"`
		Province     string `json:"province"`
		ProvinceCode string `json:"provinceCode"`
		Country      string `json:"country"`
		CountryCode  string `json:"countryCode"`
		Zip          string `json:"zip"`
		Phone        string `json:"phone"`
	} `json:"defaultAddress"`
	Metafield *struct {
		Value string `json:"value"`
	} `json:"metafield"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// ShopifyCustomerWebhook handles incoming Shopify customer webhooks (create/update).
func (app *App) ShopifyCustomerWebhook(c echo.Context) error {
	var (
		service      = "shopify"
		eventType    = "customer"
		rawReq       []byte
		err          error
		processed    = false
		errorMsg     = ""
		responseCode = http.StatusOK
	)

	// Check if customer sync is enabled
	settings, err := app.core.GetSettings()
	if err != nil {
		errorMsg = fmt.Sprintf("error fetching settings: %v", err)
		responseCode = http.StatusInternalServerError
		app.logWebhook(service, eventType, c.Request().Header, nil, responseCode, "", processed, errorMsg)
		return echo.NewHTTPError(responseCode, errorMsg)
	}

	if !settings.Shopify.CustomerSyncEnabled {
		// Customer sync is disabled, acknowledge but don't process
		return c.JSON(http.StatusOK, okResp{true})
	}

	// Read the raw request body for HMAC verification and logging
	rawReq, err = io.ReadAll(c.Request().Body)
	if err != nil {
		errorMsg = fmt.Sprintf("error reading request body: %v", err)
		responseCode = http.StatusBadRequest
		app.logWebhook(service, eventType, c.Request().Header, rawReq, responseCode, "", processed, errorMsg)
		return echo.NewHTTPError(responseCode, errorMsg)
	}

	// Initialize Shopify webhook handler
	shopifyHandler := webhooks.NewShopify(app.cfg.ShopifyWebhookSecret)

	// Verify HMAC signature
	hmacHeader := c.Request().Header.Get("X-Shopify-Hmac-Sha256")
	if err := shopifyHandler.VerifyWebhook(hmacHeader, rawReq); err != nil {
		errorMsg = fmt.Sprintf("HMAC verification failed: %v", err)
		responseCode = http.StatusUnauthorized
		app.logWebhook(service, eventType, c.Request().Header, rawReq, responseCode, "", processed, errorMsg)
		return echo.NewHTTPError(responseCode, errorMsg)
	}

	// Check the webhook topic
	topic := c.Request().Header.Get("X-Shopify-Topic")
	if topic == "customers/delete" {
		// Ignore delete webhooks as per plan
		app.log.Printf("ignoring Shopify customer delete webhook")
		return c.JSON(http.StatusOK, okResp{true})
	}

	// Parse the customer data
	customer, err := shopifyHandler.ProcessCustomer(rawReq)
	if err != nil {
		errorMsg = fmt.Sprintf("error processing customer: %v", err)
		responseCode = http.StatusBadRequest
		app.logWebhook(service, eventType, c.Request().Header, rawReq, responseCode, "", processed, errorMsg)
		return echo.NewHTTPError(responseCode, errorMsg)
	}

	// Sync the customer data to subscriber
	if err := app.syncShopifyCustomer(customer, settings.Shopify.CustomerSyncListID); err != nil {
		errorMsg = fmt.Sprintf("error syncing customer: %v", err)
		app.log.Printf("error syncing Shopify customer %d (%s): %v", customer.ID, customer.Email, err)
	} else {
		processed = true
		app.log.Printf("synced Shopify customer %d (%s)", customer.ID, customer.Email)
	}

	// Log the webhook
	app.logWebhook(service, eventType, c.Request().Header, rawReq, responseCode, "", processed, errorMsg)

	return c.JSON(http.StatusOK, okResp{true})
}

// syncShopifyCustomer syncs a Shopify customer to the corresponding subscriber.
func (app *App) syncShopifyCustomer(customer *webhooks.ShopifyCustomer, listID int) error {
	// Find subscriber by email
	sub, err := app.core.GetSubscriber(0, "", customer.Email)
	if err != nil {
		// Check if subscriber doesn't exist
		if err.(*echo.HTTPError).Code == http.StatusBadRequest {
			// Subscriber not found - log and skip
			return fmt.Errorf("subscriber not found: %s", customer.Email)
		}
		return fmt.Errorf("error finding subscriber: %v", err)
	}

	// Build Shopify attribs
	shopifyAttribs := buildShopifyAttribs(customer)

	// Merge with existing attribs
	if sub.Attribs == nil {
		sub.Attribs = make(models.JSON)
	}
	sub.Attribs["shopify"] = shopifyAttribs

	// Update subscriber
	_, err = app.core.UpdateSubscriber(sub.ID, sub)
	if err != nil {
		return fmt.Errorf("error updating subscriber: %v", err)
	}

	return nil
}

// buildShopifyAttribs converts Shopify customer data to an attribs map.
func buildShopifyAttribs(customer *webhooks.ShopifyCustomer) map[string]interface{} {
	attribs := map[string]interface{}{
		"customer_id":    customer.ID,
		"first_name":     customer.FirstName,
		"last_name":      customer.LastName,
		"phone":          customer.Phone,
		"orders_count":   customer.OrdersCount,
		"total_spent":    customer.TotalSpent,
		"currency":       customer.Currency,
		"verified_email": customer.VerifiedEmail,
		"synced_at":      time.Now().UTC().Format(time.RFC3339),
	}

	// Parse tags from comma-separated string
	if customer.Tags != "" {
		tags := strings.Split(customer.Tags, ",")
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}
		attribs["tags"] = tags
	}

	// Marketing consent
	if customer.EmailMarketingConsent.State != "" {
		attribs["marketing_consent"] = map[string]interface{}{
			"state":      customer.EmailMarketingConsent.State,
			"opt_in":     customer.EmailMarketingConsent.OptInLevel,
			"updated_at": customer.EmailMarketingConsent.ConsentUpdatedAt,
		}
	}

	// Address from default address
	if customer.DefaultAddress != nil {
		attribs["address"] = map[string]interface{}{
			"address1":      customer.DefaultAddress.Address1,
			"address2":      customer.DefaultAddress.Address2,
			"city":          customer.DefaultAddress.City,
			"province":      customer.DefaultAddress.Province,
			"province_code": customer.DefaultAddress.ProvinceCode,
			"country":       customer.DefaultAddress.Country,
			"country_code":  customer.DefaultAddress.CountryCode,
			"zip":           customer.DefaultAddress.Zip,
		}
	}

	return attribs
}

// StartShopifyCustomerSync starts a bulk sync of Shopify customers.
func (app *App) StartShopifyCustomerSync(c echo.Context) error {
	// Check if sync is already in progress
	syncState.RLock()
	if syncState.InProgress {
		syncState.RUnlock()
		return echo.NewHTTPError(http.StatusConflict, "Customer sync already in progress")
	}
	syncState.RUnlock()

	// Get settings
	settings, err := app.core.GetSettings()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Error fetching settings")
	}

	storeURL := settings.Shopify.StoreURL
	accessToken := settings.Shopify.AccessToken

	if storeURL == "" || accessToken == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Shopify store URL and access token must be configured in Settings")
	}

	// Initialize sync state
	syncState.Lock()
	syncState.InProgress = true
	syncState.StartedAt = time.Now()
	syncState.TotalCount = 0
	syncState.SyncedCount = 0
	syncState.SkippedCount = 0
	syncState.ErrorCount = 0
	syncState.LastError = ""
	syncState.CompletedAt = time.Time{}
	syncState.Unlock()

	// Update database to mark sync in progress
	app.updateSyncInProgress(true)

	// Start bulk sync in background
	go app.runBulkCustomerSync(storeURL, accessToken, settings.Shopify.CustomerSyncListID)

	return c.JSON(http.StatusOK, okResp{map[string]interface{}{
		"message": "Customer sync started",
		"status":  syncState,
	}})
}

// GetShopifyCustomerSyncStatus returns the current sync status.
func (app *App) GetShopifyCustomerSyncStatus(c echo.Context) error {
	syncState.RLock()
	defer syncState.RUnlock()

	return c.JSON(http.StatusOK, okResp{syncState})
}

// runBulkCustomerSync fetches all customers from Shopify and syncs them.
func (app *App) runBulkCustomerSync(storeURL, accessToken string, listID int) {
	defer func() {
		syncState.Lock()
		syncState.InProgress = false
		syncState.CompletedAt = time.Now()
		syncState.Unlock()

		// Update database
		app.updateSyncInProgress(false)
		app.updateLastSyncTime()
	}()

	var cursor *string
	hasNextPage := true

	url := fmt.Sprintf("https://%s/admin/api/2024-10/graphql.json", storeURL)

	for hasNextPage {
		variables := map[string]interface{}{}
		if cursor != nil {
			variables["cursor"] = *cursor
		}

		reqBody := graphQLRequest{
			Query:     customersQuery,
			Variables: variables,
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			syncState.Lock()
			syncState.LastError = fmt.Sprintf("error marshalling request: %v", err)
			syncState.Unlock()
			app.log.Printf("bulk sync error: %s", syncState.LastError)
			return
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
		if err != nil {
			syncState.Lock()
			syncState.LastError = fmt.Sprintf("error creating request: %v", err)
			syncState.Unlock()
			app.log.Printf("bulk sync error: %s", syncState.LastError)
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Shopify-Access-Token", accessToken)

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			syncState.Lock()
			syncState.LastError = fmt.Sprintf("error making request: %v", err)
			syncState.Unlock()
			app.log.Printf("bulk sync error: %s", syncState.LastError)
			return
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			syncState.Lock()
			syncState.LastError = fmt.Sprintf("error reading response: %v", err)
			syncState.Unlock()
			app.log.Printf("bulk sync error: %s", syncState.LastError)
			return
		}

		if resp.StatusCode != http.StatusOK {
			syncState.Lock()
			syncState.LastError = fmt.Sprintf("Shopify API error (status %d): %s", resp.StatusCode, string(respBody))
			syncState.Unlock()
			app.log.Printf("bulk sync error: %s", syncState.LastError)
			return
		}

		var gqlResp graphQLResponse
		if err := json.Unmarshal(respBody, &gqlResp); err != nil {
			syncState.Lock()
			syncState.LastError = fmt.Sprintf("error parsing response: %v", err)
			syncState.Unlock()
			app.log.Printf("bulk sync error: %s", syncState.LastError)
			return
		}

		if len(gqlResp.Errors) > 0 {
			syncState.Lock()
			syncState.LastError = fmt.Sprintf("GraphQL errors: %s", gqlResp.Errors[0].Message)
			syncState.Unlock()
			app.log.Printf("bulk sync error: %s", syncState.LastError)
			return
		}

		var customersResp customersResponse
		if err := json.Unmarshal(gqlResp.Data, &customersResp); err != nil {
			syncState.Lock()
			syncState.LastError = fmt.Sprintf("error parsing customers data: %v", err)
			syncState.Unlock()
			app.log.Printf("bulk sync error: %s", syncState.LastError)
			return
		}

		// Process each customer
		for _, edge := range customersResp.Customers.Edges {
			syncState.Lock()
			syncState.TotalCount++
			syncState.Unlock()

			if err := app.syncCustomerFromGraphQL(&edge.Node, listID); err != nil {
				syncState.Lock()
				if strings.Contains(err.Error(), "subscriber not found") {
					syncState.SkippedCount++
				} else {
					syncState.ErrorCount++
					syncState.LastError = err.Error()
				}
				syncState.Unlock()
			} else {
				syncState.Lock()
				syncState.SyncedCount++
				syncState.Unlock()
			}
		}

		hasNextPage = customersResp.Customers.PageInfo.HasNextPage
		if hasNextPage {
			cursor = &customersResp.Customers.PageInfo.EndCursor
		}

		// Rate limit: Shopify allows 2 requests per second for GraphQL
		time.Sleep(500 * time.Millisecond)
	}

	app.log.Printf("bulk customer sync completed: %d total, %d synced, %d skipped, %d errors",
		syncState.TotalCount, syncState.SyncedCount, syncState.SkippedCount, syncState.ErrorCount)
}

// syncCustomerFromGraphQL syncs a customer from GraphQL response to subscriber.
func (app *App) syncCustomerFromGraphQL(customer *graphQLCustomer, listID int) error {
	if customer.Email == "" {
		return fmt.Errorf("customer missing email")
	}

	// Find subscriber by email
	sub, err := app.core.GetSubscriber(0, "", customer.Email)
	if err != nil {
		return fmt.Errorf("subscriber not found: %s", customer.Email)
	}

	// Build Shopify attribs from GraphQL customer
	attribs := map[string]interface{}{
		"customer_id":  customer.ID,
		"first_name":   customer.FirstName,
		"last_name":    customer.LastName,
		"phone":        customer.Phone,
		"orders_count": customer.OrdersCount,
		"total_spent":  customer.TotalSpent,
		"synced_at":    time.Now().UTC().Format(time.RFC3339),
	}

	// Tags
	if len(customer.Tags) > 0 {
		attribs["tags"] = customer.Tags
	}

	// Marketing consent
	if customer.EmailMarketingConsent != nil {
		attribs["marketing_consent"] = map[string]interface{}{
			"state":      customer.EmailMarketingConsent.MarketingState,
			"opt_in":     customer.EmailMarketingConsent.MarketingOptInLevel,
			"updated_at": customer.EmailMarketingConsent.ConsentUpdatedAt,
		}
	}

	// Address
	if customer.DefaultAddress != nil {
		attribs["address"] = map[string]interface{}{
			"address1":      customer.DefaultAddress.Address1,
			"address2":      customer.DefaultAddress.Address2,
			"city":          customer.DefaultAddress.City,
			"province":      customer.DefaultAddress.Province,
			"province_code": customer.DefaultAddress.ProvinceCode,
			"country":       customer.DefaultAddress.Country,
			"country_code":  customer.DefaultAddress.CountryCode,
			"zip":           customer.DefaultAddress.Zip,
		}
	}

	// Birthday from metafield
	if customer.Metafield != nil && customer.Metafield.Value != "" {
		attribs["birthday"] = customer.Metafield.Value
	}

	// Merge with existing attribs
	if sub.Attribs == nil {
		sub.Attribs = make(models.JSON)
	}
	sub.Attribs["shopify"] = attribs

	// Update subscriber
	_, err = app.core.UpdateSubscriber(sub.ID, sub)
	if err != nil {
		return fmt.Errorf("error updating subscriber: %v", err)
	}

	return nil
}

// updateSyncInProgress updates the database to reflect sync status.
func (app *App) updateSyncInProgress(inProgress bool) {
	_, err := app.db.Exec(`
		UPDATE settings
		SET value = jsonb_set(value, '{customer_sync_in_progress}', $1::jsonb)
		WHERE key = 'shopify'
	`, fmt.Sprintf("%t", inProgress))
	if err != nil {
		app.log.Printf("error updating sync in progress status: %v", err)
	}
}

// updateLastSyncTime updates the last sync time in the database.
func (app *App) updateLastSyncTime() {
	_, err := app.db.Exec(`
		UPDATE settings
		SET value = jsonb_set(value, '{last_customer_sync}', $1::jsonb)
		WHERE key = 'shopify'
	`, fmt.Sprintf(`"%s"`, time.Now().UTC().Format(time.RFC3339)))
	if err != nil {
		app.log.Printf("error updating last sync time: %v", err)
	}
}

// GetSubscriberByEmail is a helper query used by shopify webhook handler
// Defined in queries.sql as get-subscriber-by-email
func (app *App) getSubscriberByEmail(email string) (models.Subscriber, error) {
	var sub models.Subscriber
	err := app.queries.GetSubscriberByEmail.Get(&sub, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return sub, fmt.Errorf("subscriber not found: %s", email)
		}
		return sub, fmt.Errorf("error finding subscriber: %v", err)
	}
	return sub, nil
}
