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
	"github.com/lib/pq"
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

// GraphQL query for fetching a customer's orders by customer ID (complete data)
const customerOrdersQuery = `
query GetCustomerOrders($customerId: ID!, $cursor: String) {
    customer(id: $customerId) {
        orders(first: 50, after: $cursor) {
            pageInfo {
                hasNextPage
                endCursor
            }
            edges {
                node {
                    id
                    name
                    createdAt
                    processedAt
                    totalPriceSet {
                        shopMoney {
                            amount
                            currencyCode
                        }
                    }
                    subtotalPriceSet {
                        shopMoney {
                            amount
                        }
                    }
                    totalTaxSet {
                        shopMoney {
                            amount
                        }
                    }
                    totalDiscountsSet {
                        shopMoney {
                            amount
                        }
                    }
                    displayFinancialStatus
                    displayFulfillmentStatus
                    tags
                    note
                    lineItems(first: 100) {
                        edges {
                            node {
                                id
                                quantity
                                originalUnitPriceSet {
                                    shopMoney {
                                        amount
                                    }
                                }
                                totalDiscountSet {
                                    shopMoney {
                                        amount
                                    }
                                }
                                sku
                                variant {
                                    id
                                    title
                                    product {
                                        id
                                        title
                                        productType
                                        vendor
                                    }
                                }
                                customAttributes {
                                    key
                                    value
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
`

// GraphQL query for fetching a single customer by email
const customerByEmailQuery = `
query GetCustomerByEmail($email: String!) {
    customers(first: 1, query: $email) {
        edges {
            node {
                id
                email
                firstName
                lastName
                phone
                tags
                note
                numberOfOrders
                amountSpent {
                    amount
                    currencyCode
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
                numberOfOrders
                amountSpent {
                    amount
                    currencyCode
                }
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
	ID             string   `json:"id"`
	Email          string   `json:"email"`
	FirstName      string   `json:"firstName"`
	LastName       string   `json:"lastName"`
	Phone          string   `json:"phone"`
	Tags           []string `json:"tags"`
	Note           string   `json:"note"`
	NumberOfOrders string   `json:"numberOfOrders"`
	AmountSpent    *struct {
		Amount       string `json:"amount"`
		CurrencyCode string `json:"currencyCode"`
	} `json:"amountSpent"`
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

// customerOrdersResponse is the GraphQL response for customer orders query
type customerOrdersResponse struct {
	Customer struct {
		Orders struct {
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
			Edges []struct {
				Node customerOrderNode `json:"node"`
			} `json:"edges"`
		} `json:"orders"`
	} `json:"customer"`
}

type customerOrderNode struct {
	ID                       string   `json:"id"`
	Name                     string   `json:"name"`
	CreatedAt                string   `json:"createdAt"`
	ProcessedAt              string   `json:"processedAt"`
	DisplayFinancialStatus   string   `json:"displayFinancialStatus"`
	DisplayFulfillmentStatus string   `json:"displayFulfillmentStatus"`
	Tags                     []string `json:"tags"`
	Note                     string   `json:"note"`
	TotalPriceSet            *struct {
		ShopMoney struct {
			Amount       string `json:"amount"`
			CurrencyCode string `json:"currencyCode"`
		} `json:"shopMoney"`
	} `json:"totalPriceSet"`
	SubtotalPriceSet *struct {
		ShopMoney struct {
			Amount string `json:"amount"`
		} `json:"shopMoney"`
	} `json:"subtotalPriceSet"`
	TotalTaxSet *struct {
		ShopMoney struct {
			Amount string `json:"amount"`
		} `json:"shopMoney"`
	} `json:"totalTaxSet"`
	TotalDiscountsSet *struct {
		ShopMoney struct {
			Amount string `json:"amount"`
		} `json:"shopMoney"`
	} `json:"totalDiscountsSet"`
	LineItems struct {
		Edges []struct {
			Node customerLineItemNode `json:"node"`
		} `json:"edges"`
	} `json:"lineItems"`
}

type customerLineItemNode struct {
	ID       string `json:"id"`
	Quantity int    `json:"quantity"`
	SKU      string `json:"sku"`
	Variant  *struct {
		ID      string `json:"id"`
		Title   string `json:"title"`
		Product *struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			ProductType string `json:"productType"`
			Vendor      string `json:"vendor"`
		} `json:"product"`
	} `json:"variant"`
	OriginalUnitPriceSet *struct {
		ShopMoney struct {
			Amount string `json:"amount"`
		} `json:"shopMoney"`
	} `json:"originalUnitPriceSet"`
	TotalDiscountSet *struct {
		ShopMoney struct {
			Amount string `json:"amount"`
		} `json:"shopMoney"`
	} `json:"totalDiscountSet"`
	CustomAttributes []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"customAttributes"`
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

	// Initialize Shopify webhook handler with both client_secret (for Partner apps)
	// and webhook_secret (for manual webhooks). It tries both during verification.
	shopifyHandler := webhooks.NewShopify(app.cfg.ShopifyClientSecret, app.cfg.ShopifyWebhookSecret)

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
// It creates a new subscriber if one doesn't exist, or updates the existing one.
func (app *App) syncShopifyCustomer(customer *webhooks.ShopifyCustomer, listID int) error {
	// Build Shopify attribs
	shopifyAttribs := buildShopifyAttribs(customer)

	// Find subscriber by email
	sub, err := app.core.GetSubscriber(0, "", customer.Email)
	if err != nil {
		// Check if subscriber doesn't exist
		if httpErr, ok := err.(*echo.HTTPError); ok && httpErr.Code == http.StatusBadRequest {
			// Subscriber not found - create a new one
			app.log.Printf("Shopify customer %s not found, creating new subscriber", customer.Email)

			// Build subscriber name from Shopify first/last name
			name := strings.TrimSpace(customer.FirstName + " " + customer.LastName)

			newSub := models.Subscriber{
				Email:  customer.Email,
				Name:   name,
				Status: models.SubscriberStatusEnabled,
				Attribs: models.JSON{
					"shopify": shopifyAttribs,
				},
			}

			// Create subscriber with the configured list ID, preconfirmed
			listIDs := []int{}
			if listID > 0 {
				listIDs = []int{listID}
			}

			createdSub, _, createErr := app.core.InsertSubscriber(newSub, listIDs, nil, true, false)
			if createErr != nil {
				// Check if this is a duplicate key error - subscriber was created by another
				// process (race condition) or already exists despite lookup failing
				if strings.Contains(createErr.Error(), "duplicate key") ||
					strings.Contains(createErr.Error(), "idx_subs_email") {
					app.log.Printf("Shopify customer %s: subscriber already exists (race condition), fetching and updating", customer.Email)
					// Retry the lookup and update
					sub, err = app.core.GetSubscriber(0, "", customer.Email)
					if err != nil {
						return fmt.Errorf("error finding existing subscriber after duplicate key: %v", err)
					}
					// Fall through to update logic below
					goto updateSubscriber
				}
				return fmt.Errorf("error creating subscriber: %v", createErr)
			}

			app.log.Printf("created new subscriber %d (%s) from Shopify customer", createdSub.ID, customer.Email)
			return nil
		}
		return fmt.Errorf("error finding subscriber: %v", err)
	}

updateSubscriber:
	// Subscriber exists - update with Shopify attribs
	if sub.Attribs == nil {
		sub.Attribs = make(models.JSON)
	}
	sub.Attribs["shopify"] = shopifyAttribs

	// Update subscriber
	_, err = app.core.UpdateSubscriber(sub.ID, sub)
	if err != nil {
		return fmt.Errorf("error updating subscriber: %v", err)
	}

	// If list ID is configured, try to add subscriber to that list
	// (AddSubscribersToLists handles duplicates gracefully)
	if listID > 0 {
		if err := app.core.AddSubscribersToLists([]int{sub.ID}, []int{listID}); err != nil {
			app.log.Printf("error adding subscriber %d to list %d: %v", sub.ID, listID, err)
		}
	}

	app.log.Printf("updated subscriber %d (%s) with Shopify customer data", sub.ID, customer.Email)
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

	// NOTE: We intentionally do NOT sync marketing_consent from Shopify.
	// listmonk is the authority on email marketing consent.

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
	app.log.Printf("StartShopifyCustomerSync: API endpoint called")

	// Check if sync is already in progress
	syncState.RLock()
	if syncState.InProgress {
		syncState.RUnlock()
		app.log.Printf("StartShopifyCustomerSync: sync already in progress, rejecting")
		return echo.NewHTTPError(http.StatusConflict, "Customer sync already in progress")
	}
	syncState.RUnlock()

	// Get settings
	settings, err := app.core.GetSettings()
	if err != nil {
		app.log.Printf("StartShopifyCustomerSync: error fetching settings: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Error fetching settings")
	}

	storeURL := settings.Shopify.StoreURL
	accessToken := settings.Shopify.AccessToken

	app.log.Printf("StartShopifyCustomerSync: store_url='%s', access_token_length=%d", storeURL, len(accessToken))

	if storeURL == "" || accessToken == "" {
		app.log.Printf("StartShopifyCustomerSync: missing credentials - store_url empty: %t, access_token empty: %t", storeURL == "", accessToken == "")
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

	app.log.Printf("StartShopifyCustomerSync: sync state initialized, starting background goroutine")

	// Start bulk sync in background
	go app.runBulkCustomerSync(storeURL, accessToken, settings.Shopify.CustomerSyncListID)

	app.log.Printf("StartShopifyCustomerSync: returning success response to client")

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
	app.log.Printf("runBulkCustomerSync: STARTING bulk sync for store %s", storeURL)

	defer func() {
		app.log.Printf("runBulkCustomerSync: COMPLETED - total=%d, synced=%d, skipped=%d, errors=%d",
			syncState.TotalCount, syncState.SyncedCount, syncState.SkippedCount, syncState.ErrorCount)

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
	app.log.Printf("runBulkCustomerSync: Shopify GraphQL URL = %s", url)

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

		app.log.Printf("runBulkCustomerSync: Shopify API response status=%d, body_length=%d", resp.StatusCode, len(respBody))

		if resp.StatusCode != http.StatusOK {
			syncState.Lock()
			syncState.LastError = fmt.Sprintf("Shopify API error (status %d): %s", resp.StatusCode, string(respBody))
			syncState.Unlock()
			app.log.Printf("bulk sync error: %s", syncState.LastError)
			return
		}

		// Log first 500 chars of response for debugging
		respPreview := string(respBody)
		if len(respPreview) > 500 {
			respPreview = respPreview[:500] + "..."
		}
		app.log.Printf("runBulkCustomerSync: API response preview: %s", respPreview)

		var gqlResp graphQLResponse
		if err := json.Unmarshal(respBody, &gqlResp); err != nil {
			syncState.Lock()
			syncState.LastError = fmt.Sprintf("error parsing response: %v", err)
			syncState.Unlock()
			app.log.Printf("bulk sync error: %s", syncState.LastError)
			return
		}

		if len(gqlResp.Errors) > 0 {
			errMsg := gqlResp.Errors[0].Message
			// Provide more helpful error message for common issues
			if strings.Contains(errMsg, "Access denied") {
				errMsg = "Shopify API access denied: Your access token needs the 'read_customers' scope. Go to Shopify Admin > Settings > Apps > Develop apps > [Your App] > Configure Admin API scopes, add 'read_customers', save, and regenerate the access token."
			}
			syncState.Lock()
			syncState.LastError = errMsg
			syncState.Unlock()
			app.log.Printf("bulk sync error: %s", errMsg)
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

		app.log.Printf("runBulkCustomerSync: fetched %d customers in this batch (hasNextPage=%v)",
			len(customersResp.Customers.Edges), customersResp.Customers.PageInfo.HasNextPage)

		// Process each customer
		for _, edge := range customersResp.Customers.Edges {
			app.log.Printf("runBulkCustomerSync: processing customer email='%s' id='%s'", edge.Node.Email, edge.Node.ID)
			syncState.Lock()
			syncState.TotalCount++
			syncState.Unlock()

			if err := app.syncCustomerFromGraphQL(&edge.Node, listID, storeURL, accessToken); err != nil {
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
func (app *App) syncCustomerFromGraphQL(customer *graphQLCustomer, listID int, storeURL, accessToken string) error {
	if customer.Email == "" {
		app.log.Printf("syncCustomerFromGraphQL: skipping customer with empty email (ID=%s)", customer.ID)
		return fmt.Errorf("customer missing email")
	}

	// Find subscriber by email
	app.log.Printf("syncCustomerFromGraphQL: looking up subscriber with email='%s'", customer.Email)
	sub, err := app.core.GetSubscriber(0, "", customer.Email)
	if err != nil {
		app.log.Printf("syncCustomerFromGraphQL: subscriber NOT FOUND for email='%s': %v", customer.Email, err)
		return fmt.Errorf("subscriber not found: %s", customer.Email)
	}
	app.log.Printf("syncCustomerFromGraphQL: FOUND subscriber id=%d for email='%s'", sub.ID, customer.Email)

	// Build Shopify attribs from GraphQL customer
	attribs := map[string]interface{}{
		"customer_id":  customer.ID,
		"first_name":   customer.FirstName,
		"last_name":    customer.LastName,
		"phone":        customer.Phone,
		"orders_count": customer.NumberOfOrders,
		"synced_at":    time.Now().UTC().Format(time.RFC3339),
	}

	// Add amount spent if available
	if customer.AmountSpent != nil {
		attribs["total_spent"] = customer.AmountSpent.Amount
		attribs["currency"] = customer.AmountSpent.CurrencyCode
	}

	// Tags
	if len(customer.Tags) > 0 {
		attribs["tags"] = customer.Tags
	}

	// NOTE: We intentionally do NOT sync marketing_consent from Shopify.
	// listmonk is the authority on email marketing consent.

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

	// Sync order history to database and extract purchased effects
	if storeURL != "" && accessToken != "" {
		effects, err := app.syncCustomerOrdersToDB(storeURL, accessToken, customer.ID, sub.ID)
		if err != nil {
			app.log.Printf("syncCustomerFromGraphQL: error syncing orders for customer %s: %v", customer.Email, err)
			// Continue without effects - not fatal
		} else if len(effects) > 0 {
			attribs["purchased_effects"] = effects
			app.log.Printf("syncCustomerFromGraphQL: synced orders and found %d effects for customer %s: %v", len(effects), customer.Email, effects)
		}
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

// syncCustomerAfterOrder fetches customer data from Shopify by email and updates subscriber attribs.
// This is called after an order webhook to ensure subscriber data is up-to-date.
func (app *App) syncCustomerAfterOrder(email string, subscriberID int) error {
	// Get Shopify settings
	settings, err := app.core.GetSettings()
	if err != nil {
		return fmt.Errorf("error fetching settings: %v", err)
	}

	storeURL := settings.Shopify.StoreURL
	accessToken := settings.Shopify.AccessToken

	if storeURL == "" || accessToken == "" {
		return fmt.Errorf("Shopify not configured (missing store URL or access token)")
	}

	// Fetch customer from Shopify by email
	url := fmt.Sprintf("https://%s/admin/api/2024-10/graphql.json", storeURL)

	// Build GraphQL request - search by email
	reqBody := struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables,omitempty"`
	}{
		Query: customerByEmailQuery,
		Variables: map[string]interface{}{
			"email": fmt.Sprintf("email:%s", email),
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("error marshalling request: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Shopify-Access-Token", accessToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Shopify API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var graphqlResp struct {
		Data   customersResponse `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&graphqlResp); err != nil {
		return fmt.Errorf("error decoding response: %v", err)
	}

	if len(graphqlResp.Errors) > 0 {
		return fmt.Errorf("GraphQL error: %s", graphqlResp.Errors[0].Message)
	}

	// Check if customer was found
	if len(graphqlResp.Data.Customers.Edges) == 0 {
		app.log.Printf("syncCustomerAfterOrder: no Shopify customer found for email %s", email)
		return nil
	}

	customer := graphqlResp.Data.Customers.Edges[0].Node

	// Get subscriber
	sub, err := app.core.GetSubscriber(subscriberID, "", "")
	if err != nil {
		return fmt.Errorf("error getting subscriber: %v", err)
	}

	// Build Shopify attribs
	attribs := map[string]interface{}{
		"customer_id":  customer.ID,
		"first_name":   customer.FirstName,
		"last_name":    customer.LastName,
		"phone":        customer.Phone,
		"orders_count": customer.NumberOfOrders,
		"synced_at":    time.Now().UTC().Format(time.RFC3339),
	}

	// Add amount spent if available
	if customer.AmountSpent != nil {
		attribs["total_spent"] = customer.AmountSpent.Amount
		attribs["currency"] = customer.AmountSpent.CurrencyCode
	}

	// Tags
	if len(customer.Tags) > 0 {
		attribs["tags"] = customer.Tags
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

	// Update subscriber name if empty and we have name data
	if sub.Name == "" && (customer.FirstName != "" || customer.LastName != "") {
		sub.Name = strings.TrimSpace(customer.FirstName + " " + customer.LastName)
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

	app.log.Printf("syncCustomerAfterOrder: updated subscriber %d (%s) with Shopify data (orders=%s, total_spent=%v)",
		sub.ID, email, customer.NumberOfOrders, attribs["total_spent"])

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

// syncCustomerOrdersToDB fetches all orders for a customer, stores them in the database,
// and returns unique effects from product titles
func (app *App) syncCustomerOrdersToDB(storeURL, accessToken, customerID string, subscriberID int) ([]string, error) {
	effectsMap := make(map[string]bool)
	var cursor *string
	hasNextPage := true
	ordersStored := 0

	url := fmt.Sprintf("https://%s/admin/api/2024-10/graphql.json", storeURL)

	for hasNextPage {
		variables := map[string]interface{}{
			"customerId": customerID,
		}
		if cursor != nil {
			variables["cursor"] = *cursor
		}

		reqBody := struct {
			Query     string                 `json:"query"`
			Variables map[string]interface{} `json:"variables,omitempty"`
		}{
			Query:     customerOrdersQuery,
			Variables: variables,
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("error marshalling request: %v", err)
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
		if err != nil {
			return nil, fmt.Errorf("error creating request: %v", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Shopify-Access-Token", accessToken)

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error making request: %v", err)
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("error reading response: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Shopify API error (status %d): %s", resp.StatusCode, string(respBody))
		}

		var gqlResp struct {
			Data   json.RawMessage `json:"data"`
			Errors []struct {
				Message string `json:"message"`
			} `json:"errors,omitempty"`
		}
		if err := json.Unmarshal(respBody, &gqlResp); err != nil {
			return nil, fmt.Errorf("error parsing response: %v", err)
		}

		if len(gqlResp.Errors) > 0 {
			return nil, fmt.Errorf("GraphQL errors: %s", gqlResp.Errors[0].Message)
		}

		var ordersResp customerOrdersResponse
		if err := json.Unmarshal(gqlResp.Data, &ordersResp); err != nil {
			return nil, fmt.Errorf("error parsing orders data: %v", err)
		}

		// Store each order and its line items, extract effects
		for _, orderEdge := range ordersResp.Customer.Orders.Edges {
			order := orderEdge.Node

			// Parse order timestamps
			createdAt, _ := time.Parse(time.RFC3339, order.CreatedAt)
			var processedAt *time.Time
			if order.ProcessedAt != "" {
				t, err := time.Parse(time.RFC3339, order.ProcessedAt)
				if err == nil {
					processedAt = &t
				}
			}

			// Parse price fields
			var totalPrice, subtotalPrice, totalTax, totalDiscounts float64
			var currency string
			if order.TotalPriceSet != nil {
				fmt.Sscanf(order.TotalPriceSet.ShopMoney.Amount, "%f", &totalPrice)
				currency = order.TotalPriceSet.ShopMoney.CurrencyCode
			}
			if order.SubtotalPriceSet != nil {
				fmt.Sscanf(order.SubtotalPriceSet.ShopMoney.Amount, "%f", &subtotalPrice)
			}
			if order.TotalTaxSet != nil {
				fmt.Sscanf(order.TotalTaxSet.ShopMoney.Amount, "%f", &totalTax)
			}
			if order.TotalDiscountsSet != nil {
				fmt.Sscanf(order.TotalDiscountsSet.ShopMoney.Amount, "%f", &totalDiscounts)
			}

			// Upsert order
			var orderID int
			err := app.queries.UpsertShopifyOrder.Get(&orderID,
				subscriberID,
				order.ID,
				order.Name, // order_number - Name contains "#1234" format
				order.Name, // order_name
				createdAt,
				processedAt,
				totalPrice,
				subtotalPrice,
				totalTax,
				totalDiscounts,
				currency,
				order.DisplayFinancialStatus,
				order.DisplayFulfillmentStatus,
				pq.Array(order.Tags),
				order.Note,
			)
			if err != nil {
				app.log.Printf("syncCustomerOrdersToDB: error upserting order %s: %v", order.ID, err)
				continue
			}
			ordersStored++

			// Delete existing line items and re-insert
			_, _ = app.queries.DeleteShopifyOrderLineItems.Exec(orderID)

			// Insert line items
			for _, lineEdge := range order.LineItems.Edges {
				lineItem := lineEdge.Node

				var productID, productTitle, variantID, variantTitle, productType, vendor string
				if lineItem.Variant != nil {
					variantID = lineItem.Variant.ID
					variantTitle = lineItem.Variant.Title
					if lineItem.Variant.Product != nil {
						productID = lineItem.Variant.Product.ID
						productTitle = lineItem.Variant.Product.Title
						productType = lineItem.Variant.Product.ProductType
						vendor = lineItem.Variant.Product.Vendor
					}
				}

				var price, totalDiscount float64
				if lineItem.OriginalUnitPriceSet != nil {
					fmt.Sscanf(lineItem.OriginalUnitPriceSet.ShopMoney.Amount, "%f", &price)
				}
				if lineItem.TotalDiscountSet != nil {
					fmt.Sscanf(lineItem.TotalDiscountSet.ShopMoney.Amount, "%f", &totalDiscount)
				}

				// Convert custom attributes to JSON (default to empty object for JSONB column)
				var propertiesJSON []byte = []byte("{}")
				if len(lineItem.CustomAttributes) > 0 {
					props := make(map[string]string)
					for _, attr := range lineItem.CustomAttributes {
						props[attr.Key] = attr.Value
					}
					propertiesJSON, _ = json.Marshal(props)
				}

				_, err := app.queries.InsertShopifyLineItem.Exec(
					orderID,
					lineItem.ID,
					productID,
					productTitle,
					variantID,
					variantTitle,
					lineItem.SKU,
					lineItem.Quantity,
					price,
					totalDiscount,
					productType,
					vendor,
					propertiesJSON,
				)
				if err != nil {
					app.log.Printf("syncCustomerOrdersToDB: error inserting line item %s: %v", lineItem.ID, err)
				}

				// Extract effects
				if productTitle != "" {
					effect := extractEffectFromProductTitle(productTitle)
					if effect != "" {
						effectsMap[effect] = true
					}
				}
			}
		}

		hasNextPage = ordersResp.Customer.Orders.PageInfo.HasNextPage
		if hasNextPage {
			cursor = &ordersResp.Customer.Orders.PageInfo.EndCursor
		}
	}

	app.log.Printf("syncCustomerOrdersToDB: stored %d orders for subscriber %d", ordersStored, subscriberID)

	// Convert map to slice
	effects := make([]string, 0, len(effectsMap))
	for effect := range effectsMap {
		effects = append(effects, effect)
	}

	return effects, nil
}

// extractEffectFromProductTitle extracts the effect (CALM, SLEEP, FOCUS, ENERGY) from product title
func extractEffectFromProductTitle(title string) string {
	titleUpper := strings.ToUpper(title)
	effects := []string{"CALM", "SLEEP", "FOCUS", "ENERGY"}
	for _, effect := range effects {
		if strings.Contains(titleUpper, effect) {
			return effect
		}
	}
	return ""
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

// REST API response structures for orders
type restOrdersResponse struct {
	Orders []restOrder `json:"orders"`
}

type restOrder struct {
	ID                int64           `json:"id"`
	Name              string          `json:"name"`
	Email             string          `json:"email"`
	CreatedAt         string          `json:"created_at"`
	ProcessedAt       string          `json:"processed_at"`
	TotalPrice        string          `json:"total_price"`
	SubtotalPrice     string          `json:"subtotal_price"`
	TotalTax          string          `json:"total_tax"`
	TotalDiscounts    string          `json:"total_discounts"`
	Currency          string          `json:"currency"`
	FinancialStatus   string          `json:"financial_status"`
	FulfillmentStatus string          `json:"fulfillment_status"`
	Tags              string          `json:"tags"`
	Note              string          `json:"note"`
	Customer          *restCustomer   `json:"customer"`
	LineItems         []restLineItem  `json:"line_items"`
}

type restCustomer struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
}

type restLineItem struct {
	ID            int64                    `json:"id"`
	ProductID     int64                    `json:"product_id"`
	Title         string                   `json:"title"`
	VariantID     int64                    `json:"variant_id"`
	VariantTitle  string                   `json:"variant_title"`
	Quantity      int                      `json:"quantity"`
	Price         string                   `json:"price"`
	SKU           string                   `json:"sku"`
	Vendor        string                   `json:"vendor"`
	ProductExists bool                     `json:"product_exists"`
	Properties    []map[string]interface{} `json:"properties"`
}

// orderSyncState tracks the state of bulk order sync operations
type orderSyncState struct {
	sync.RWMutex
	InProgress   bool      `json:"in_progress"`
	StartedAt    time.Time `json:"started_at,omitempty"`
	TotalOrders  int       `json:"total_orders"`
	SyncedOrders int       `json:"synced_orders"`
	MatchedSubs  int       `json:"matched_subscribers"`
	SkippedCount int       `json:"skipped_count"`
	ErrorCount   int       `json:"error_count"`
	LastError    string    `json:"last_error,omitempty"`
	CompletedAt  time.Time `json:"completed_at,omitempty"`
}

var orderSyncStateGlobal = &orderSyncState{}

// GetOrderSyncStatus returns the current status of order sync
func (app *App) GetOrderSyncStatus(c echo.Context) error {
	orderSyncStateGlobal.RLock()
	defer orderSyncStateGlobal.RUnlock()

	return c.JSON(http.StatusOK, orderSyncStateGlobal)
}

// TriggerOrderSync starts a bulk order sync via REST API
func (app *App) TriggerOrderSync(c echo.Context) error {
	orderSyncStateGlobal.Lock()
	if orderSyncStateGlobal.InProgress {
		orderSyncStateGlobal.Unlock()
		return c.JSON(http.StatusConflict, map[string]string{
			"error": "Order sync already in progress",
		})
	}
	orderSyncStateGlobal.InProgress = true
	orderSyncStateGlobal.StartedAt = time.Now()
	orderSyncStateGlobal.TotalOrders = 0
	orderSyncStateGlobal.SyncedOrders = 0
	orderSyncStateGlobal.MatchedSubs = 0
	orderSyncStateGlobal.SkippedCount = 0
	orderSyncStateGlobal.ErrorCount = 0
	orderSyncStateGlobal.LastError = ""
	orderSyncStateGlobal.CompletedAt = time.Time{}
	orderSyncStateGlobal.Unlock()

	// Get Shopify settings
	settings, err := app.core.GetSettings()
	if err != nil {
		orderSyncStateGlobal.Lock()
		orderSyncStateGlobal.InProgress = false
		orderSyncStateGlobal.LastError = fmt.Sprintf("error getting settings: %v", err)
		orderSyncStateGlobal.Unlock()
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if !settings.Shopify.Enabled {
		orderSyncStateGlobal.Lock()
		orderSyncStateGlobal.InProgress = false
		orderSyncStateGlobal.LastError = "Shopify integration is not enabled"
		orderSyncStateGlobal.Unlock()
		return echo.NewHTTPError(http.StatusBadRequest, "Shopify integration is not enabled")
	}

	storeURL := settings.Shopify.StoreURL
	accessToken := settings.Shopify.AccessToken

	if storeURL == "" || accessToken == "" {
		orderSyncStateGlobal.Lock()
		orderSyncStateGlobal.InProgress = false
		orderSyncStateGlobal.LastError = "Shopify credentials not configured"
		orderSyncStateGlobal.Unlock()
		return echo.NewHTTPError(http.StatusBadRequest, "Shopify Store URL and Access Token must be configured")
	}

	// Start sync in background
	go app.syncAllOrdersViaREST(storeURL, accessToken)

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Order sync started",
	})
}

// syncAllOrdersViaREST fetches ALL orders from Shopify via REST API and syncs them to the database.
// This bypasses the 60-day GraphQL limitation.
func (app *App) syncAllOrdersViaREST(storeURL, accessToken string) {
	defer func() {
		orderSyncStateGlobal.Lock()
		orderSyncStateGlobal.InProgress = false
		orderSyncStateGlobal.CompletedAt = time.Now()
		orderSyncStateGlobal.Unlock()
	}()

	app.log.Printf("syncAllOrdersViaREST: starting bulk order sync from %s", storeURL)

	// Build email to subscriber ID map for quick lookups
	emailToSubID := make(map[string]int)
	var subscribers []struct {
		ID    int    `db:"id"`
		Email string `db:"email"`
	}
	err := app.db.Select(&subscribers, `SELECT id, email FROM subscribers WHERE email IS NOT NULL AND email != ''`)
	if err != nil {
		app.log.Printf("syncAllOrdersViaREST: error fetching subscribers: %v", err)
		orderSyncStateGlobal.Lock()
		orderSyncStateGlobal.LastError = fmt.Sprintf("error fetching subscribers: %v", err)
		orderSyncStateGlobal.Unlock()
		return
	}

	for _, sub := range subscribers {
		emailToSubID[strings.ToLower(sub.Email)] = sub.ID
	}
	app.log.Printf("syncAllOrdersViaREST: loaded %d subscribers for matching", len(emailToSubID))

	// Track effects per subscriber for later update
	subscriberEffects := make(map[int]map[string]bool)

	// Fetch all orders from REST API with pagination
	client := &http.Client{Timeout: 60 * time.Second}
	baseURL := fmt.Sprintf("https://%s/admin/api/2024-10/orders.json", storeURL)

	var pageInfo string
	hasNextPage := true
	totalOrders := 0
	syncedOrders := 0
	matchedSubs := 0
	skippedCount := 0
	errorCount := 0

	for hasNextPage {
		// Build URL with pagination
		// Note: Shopify cursor-based pagination - page_info CANNOT be combined with other params
		var reqURL string
		if pageInfo != "" {
			// Use only page_info for subsequent pages
			reqURL = baseURL + "?page_info=" + pageInfo
		} else {
			// First page uses status and limit
			reqURL = baseURL + "?status=any&limit=250"
		}

		req, err := http.NewRequest("GET", reqURL, nil)
		if err != nil {
			app.log.Printf("syncAllOrdersViaREST: error creating request: %v", err)
			orderSyncStateGlobal.Lock()
			orderSyncStateGlobal.LastError = fmt.Sprintf("error creating request: %v", err)
			orderSyncStateGlobal.Unlock()
			return
		}

		req.Header.Set("X-Shopify-Access-Token", accessToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			app.log.Printf("syncAllOrdersViaREST: error making request: %v", err)
			orderSyncStateGlobal.Lock()
			orderSyncStateGlobal.LastError = fmt.Sprintf("error making request: %v", err)
			orderSyncStateGlobal.Unlock()
			return
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			app.log.Printf("syncAllOrdersViaREST: error reading response: %v", err)
			orderSyncStateGlobal.Lock()
			orderSyncStateGlobal.LastError = fmt.Sprintf("error reading response: %v", err)
			orderSyncStateGlobal.Unlock()
			return
		}

		if resp.StatusCode != http.StatusOK {
			app.log.Printf("syncAllOrdersViaREST: API error (status %d): %s", resp.StatusCode, string(body))
			orderSyncStateGlobal.Lock()
			orderSyncStateGlobal.LastError = fmt.Sprintf("API error (status %d)", resp.StatusCode)
			orderSyncStateGlobal.Unlock()
			return
		}

		var ordersResp restOrdersResponse
		if err := json.Unmarshal(body, &ordersResp); err != nil {
			app.log.Printf("syncAllOrdersViaREST: error parsing response: %v", err)
			orderSyncStateGlobal.Lock()
			orderSyncStateGlobal.LastError = fmt.Sprintf("error parsing response: %v", err)
			orderSyncStateGlobal.Unlock()
			return
		}

		totalOrders += len(ordersResp.Orders)

		// Process each order
		for _, order := range ordersResp.Orders {
			// Get customer email - try order email first, then customer email
			email := strings.ToLower(order.Email)
			if email == "" && order.Customer != nil {
				email = strings.ToLower(order.Customer.Email)
			}

			if email == "" {
				skippedCount++
				continue
			}

			// Find subscriber by email
			subscriberID, found := emailToSubID[email]
			if !found {
				skippedCount++
				continue
			}
			matchedSubs++

			// Parse order data
			createdAt, _ := time.Parse(time.RFC3339, order.CreatedAt)
			var processedAt *time.Time
			if order.ProcessedAt != "" {
				t, err := time.Parse(time.RFC3339, order.ProcessedAt)
				if err == nil {
					processedAt = &t
				}
			}

			var totalPrice, subtotalPrice, totalTax, totalDiscounts float64
			fmt.Sscanf(order.TotalPrice, "%f", &totalPrice)
			fmt.Sscanf(order.SubtotalPrice, "%f", &subtotalPrice)
			fmt.Sscanf(order.TotalTax, "%f", &totalTax)
			fmt.Sscanf(order.TotalDiscounts, "%f", &totalDiscounts)

			// Convert tags string to array
			var tags []string
			if order.Tags != "" {
				for _, tag := range strings.Split(order.Tags, ",") {
					tags = append(tags, strings.TrimSpace(tag))
				}
			}

			// Map financial status
			financialStatus := strings.ToUpper(order.FinancialStatus)
			fulfillmentStatus := strings.ToUpper(order.FulfillmentStatus)
			if fulfillmentStatus == "" {
				fulfillmentStatus = "UNFULFILLED"
			}

			// Create Shopify GID format for order ID
			shopifyOrderID := fmt.Sprintf("gid://shopify/Order/%d", order.ID)

			// Upsert order
			var orderID int
			err := app.queries.UpsertShopifyOrder.Get(&orderID,
				subscriberID,
				shopifyOrderID,
				order.Name, // order_number
				order.Name, // order_name
				createdAt,
				processedAt,
				totalPrice,
				subtotalPrice,
				totalTax,
				totalDiscounts,
				order.Currency,
				financialStatus,
				fulfillmentStatus,
				pq.Array(tags),
				order.Note,
			)
			if err != nil {
				app.log.Printf("syncAllOrdersViaREST: error upserting order %d: %v", order.ID, err)
				errorCount++
				continue
			}
			syncedOrders++

			// Delete existing line items and re-insert
			_, _ = app.queries.DeleteShopifyOrderLineItems.Exec(orderID)

			// Insert line items and extract effects
			for _, lineItem := range order.LineItems {
				// Create Shopify GID formats
				lineItemID := fmt.Sprintf("gid://shopify/LineItem/%d", lineItem.ID)
				productID := ""
				if lineItem.ProductID > 0 {
					productID = fmt.Sprintf("gid://shopify/Product/%d", lineItem.ProductID)
				}
				variantID := ""
				if lineItem.VariantID > 0 {
					variantID = fmt.Sprintf("gid://shopify/ProductVariant/%d", lineItem.VariantID)
				}

				var price float64
				fmt.Sscanf(lineItem.Price, "%f", &price)

				// Convert properties to JSON
				var propertiesJSON []byte = []byte("{}")
				if len(lineItem.Properties) > 0 {
					propertiesJSON, _ = json.Marshal(lineItem.Properties)
				}

				_, err := app.queries.InsertShopifyLineItem.Exec(
					orderID,
					lineItemID,
					productID,
					lineItem.Title,
					variantID,
					lineItem.VariantTitle,
					lineItem.SKU,
					lineItem.Quantity,
					price,
					0.0, // total_discount not available per-item in REST API
					"",  // product_type not available in REST line items
					lineItem.Vendor,
					propertiesJSON,
				)
				if err != nil {
					app.log.Printf("syncAllOrdersViaREST: error inserting line item %d: %v", lineItem.ID, err)
				}

				// Extract effect from product title
				effect := extractEffectFromProductTitle(lineItem.Title)
				if effect != "" {
					if subscriberEffects[subscriberID] == nil {
						subscriberEffects[subscriberID] = make(map[string]bool)
					}
					subscriberEffects[subscriberID][effect] = true
				}
			}
		}

		// Update progress
		orderSyncStateGlobal.Lock()
		orderSyncStateGlobal.TotalOrders = totalOrders
		orderSyncStateGlobal.SyncedOrders = syncedOrders
		orderSyncStateGlobal.MatchedSubs = matchedSubs
		orderSyncStateGlobal.SkippedCount = skippedCount
		orderSyncStateGlobal.ErrorCount = errorCount
		orderSyncStateGlobal.Unlock()

		// Check for pagination via Link header
		linkHeader := resp.Header.Get("Link")
		hasNextPage = false
		if linkHeader != "" {
			// Parse Link header for next page
			// Format: <url?page_info=xyz>; rel="next", <url?page_info=abc>; rel="previous"
			for _, link := range strings.Split(linkHeader, ",") {
				if strings.Contains(link, `rel="next"`) {
					// Extract page_info from URL
					start := strings.Index(link, "page_info=")
					if start != -1 {
						end := strings.Index(link[start:], ">")
						if end != -1 {
							pageInfo = link[start+10 : start+end]
							hasNextPage = true
						}
					}
					break
				}
			}
		}

		app.log.Printf("syncAllOrdersViaREST: processed %d orders so far, synced %d, matched %d subscribers",
			totalOrders, syncedOrders, matchedSubs)

		// Rate limit protection - small delay between pages
		time.Sleep(500 * time.Millisecond)
	}

	// Update purchased_effects for all subscribers with orders
	app.log.Printf("syncAllOrdersViaREST: updating purchased_effects for %d subscribers", len(subscriberEffects))
	for subID, effects := range subscriberEffects {
		effectsList := make([]string, 0, len(effects))
		for effect := range effects {
			effectsList = append(effectsList, effect)
		}

		// Convert to JSON array for JSONB column
		effectsJSON, err := json.Marshal(effectsList)
		if err != nil {
			app.log.Printf("syncAllOrdersViaREST: error marshalling effects for subscriber %d: %v", subID, err)
			continue
		}

		// Update subscriber attribs with purchased_effects
		_, err = app.db.Exec(`
			UPDATE subscribers
			SET attribs = jsonb_set(
				COALESCE(attribs, '{}'::jsonb),
				'{purchased_effects}',
				$1::jsonb
			)
			WHERE id = $2
		`, string(effectsJSON), subID)
		if err != nil {
			app.log.Printf("syncAllOrdersViaREST: error updating purchased_effects for subscriber %d: %v", subID, err)
		}
	}

	// Clear segment caches since data changed
	_, _ = app.db.Exec(`UPDATE segments SET cached_count = NULL, cached_at = NULL`)

	app.log.Printf("syncAllOrdersViaREST: completed. Total: %d, Synced: %d, Matched: %d, Skipped: %d, Errors: %d",
		totalOrders, syncedOrders, matchedSubs, skippedCount, errorCount)
}
