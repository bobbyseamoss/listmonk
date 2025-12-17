package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// Effects to track
var shopifyEffects = []string{"CALM", "SLEEP", "FOCUS", "ENERGY"}

// GraphQL query for fetching orders
const ordersQuery = `
query GetOrders($query: String!, $cursor: String) {
    orders(first: 50, query: $query, after: $cursor) {
        pageInfo {
            hasNextPage
            endCursor
        }
        edges {
            node {
                id
                name
                createdAt
                lineItems(first: 100) {
                    edges {
                        node {
                            quantity
                            variant {
                                id
                                selectedOptions {
                                    name
                                    value
                                }
                                product {
                                    title
                                }
                                metafields(first: 5) {
                                    edges {
                                        node {
                                            namespace
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
    }
}
`

// GraphQL request/response structures
type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

type ordersResponse struct {
	Orders struct {
		PageInfo struct {
			HasNextPage bool   `json:"hasNextPage"`
			EndCursor   string `json:"endCursor"`
		} `json:"pageInfo"`
		Edges []struct {
			Node orderNode `json:"node"`
		} `json:"edges"`
	} `json:"orders"`
}

type orderNode struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
	LineItems struct {
		Edges []struct {
			Node lineItemNode `json:"node"`
		} `json:"edges"`
	} `json:"lineItems"`
}

type lineItemNode struct {
	Quantity int `json:"quantity"`
	Variant  *struct {
		ID              string `json:"id"`
		SelectedOptions []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"selectedOptions"`
		Product *struct {
			Title string `json:"title"`
		} `json:"product"`
		Metafields struct {
			Edges []struct {
				Node struct {
					Namespace string `json:"namespace"`
					Key       string `json:"key"`
					Value     string `json:"value"`
				} `json:"node"`
			} `json:"edges"`
		} `json:"metafields"`
	} `json:"variant"`
}

// API response structures
type ShopifyOrderTallyRequest struct {
	StartDate string `json:"start_date" query:"start_date"`
	EndDate   string `json:"end_date" query:"end_date"`
}

type FlavorTally struct {
	Flavor string `json:"flavor"`
	Count  int    `json:"count"`
}

type EffectTally struct {
	Effect  string        `json:"effect"`
	Flavors []FlavorTally `json:"flavors"`
	Total   int           `json:"total"`
}

type ShopifyOrderTallyResponse struct {
	StartDate   string        `json:"start_date"`
	EndDate     string        `json:"end_date"`
	OrderCount  int           `json:"order_count"`
	Effects     []EffectTally `json:"effects"`
	GrandTotal  int           `json:"grand_total"`
}

// handleGetShopifyOrderTally fetches and aggregates Shopify orders by effect and flavor
func handleGetShopifyOrderTally(c echo.Context) error {
	var app = c.Get("app").(*App)

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

	// Parse request parameters
	var req ShopifyOrderTallyRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request parameters")
	}

	if req.StartDate == "" || req.EndDate == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "start_date and end_date are required (format: YYYY-MM-DD)")
	}

	// Validate dates
	if _, err := time.Parse("2006-01-02", req.StartDate); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid start_date format. Use YYYY-MM-DD")
	}
	if _, err := time.Parse("2006-01-02", req.EndDate); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid end_date format. Use YYYY-MM-DD")
	}

	// Fetch orders from Shopify
	orders, err := fetchShopifyOrders(storeURL, accessToken, req.StartDate, req.EndDate)
	if err != nil {
		app.log.Printf("error fetching Shopify orders: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Error fetching orders: %v", err))
	}

	// Aggregate results
	results := aggregateOrders(orders)

	response := ShopifyOrderTallyResponse{
		StartDate:  req.StartDate,
		EndDate:    req.EndDate,
		OrderCount: len(orders),
		Effects:    results,
	}

	// Calculate grand total
	for _, effect := range results {
		response.GrandTotal += effect.Total
	}

	return c.JSON(http.StatusOK, okResp{response})
}

func fetchShopifyOrders(storeURL, accessToken, startDate, endDate string) ([]orderNode, error) {
	var orders []orderNode
	var cursor *string
	hasNextPage := true

	url := fmt.Sprintf("https://%s/admin/api/2024-10/graphql.json", storeURL)
	dateQuery := fmt.Sprintf("created_at:>=%s created_at:<=%s", startDate, endDate)

	for hasNextPage {
		variables := map[string]interface{}{
			"query": dateQuery,
		}
		if cursor != nil {
			variables["cursor"] = *cursor
		}

		reqBody := graphQLRequest{
			Query:     ordersQuery,
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
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading response: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Shopify API error (status %d): %s", resp.StatusCode, string(respBody))
		}

		var gqlResp graphQLResponse
		if err := json.Unmarshal(respBody, &gqlResp); err != nil {
			return nil, fmt.Errorf("error parsing response: %v", err)
		}

		if len(gqlResp.Errors) > 0 {
			return nil, fmt.Errorf("GraphQL errors: %s", gqlResp.Errors[0].Message)
		}

		var ordersResp ordersResponse
		if err := json.Unmarshal(gqlResp.Data, &ordersResp); err != nil {
			return nil, fmt.Errorf("error parsing orders data: %v", err)
		}

		for _, edge := range ordersResp.Orders.Edges {
			orders = append(orders, edge.Node)
		}

		hasNextPage = ordersResp.Orders.PageInfo.HasNextPage
		if hasNextPage {
			cursor = &ordersResp.Orders.PageInfo.EndCursor
		}
	}

	return orders, nil
}

func aggregateOrders(orders []orderNode) []EffectTally {
	// Map: effect -> flavor -> count
	results := make(map[string]map[string]int)

	for _, order := range orders {
		for _, lineEdge := range order.LineItems.Edges {
			lineItem := lineEdge.Node
			if lineItem.Variant == nil || lineItem.Variant.Product == nil {
				continue
			}

			productTitle := lineItem.Variant.Product.Title

			// Extract effect from product title
			effect := extractEffectFromTitle(productTitle)
			if effect == "" {
				continue
			}

			// Get flavor - first try selectedOptions, then fall back to product title
			flavor := getFlavorFromOptions(lineItem.Variant.SelectedOptions)
			if flavor == "" {
				flavor = extractFlavorFromTitle(productTitle)
			}
			if flavor == "" {
				flavor = "Unknown"
			}

			// Get quantity_actual from metafields (defaults to 1 if not found)
			quantityActual := getQuantityActual(lineItem.Variant.Metafields.Edges)

			// Multiply by line item quantity
			total := quantityActual * lineItem.Quantity

			// Initialize maps if needed
			if _, ok := results[effect]; !ok {
				results[effect] = make(map[string]int)
			}
			results[effect][flavor] += total
		}
	}

	// Convert to response format
	var effectTallies []EffectTally
	for _, effect := range shopifyEffects {
		if flavors, ok := results[effect]; ok {
			var flavorList []FlavorTally
			total := 0
			for flavor, count := range flavors {
				flavorList = append(flavorList, FlavorTally{
					Flavor: flavor,
					Count:  count,
				})
				total += count
			}

			// Sort flavors alphabetically
			sort.Slice(flavorList, func(i, j int) bool {
				return flavorList[i].Flavor < flavorList[j].Flavor
			})

			effectTallies = append(effectTallies, EffectTally{
				Effect:  effect,
				Flavors: flavorList,
				Total:   total,
			})
		}
	}

	return effectTallies
}

func extractEffectFromTitle(title string) string {
	titleUpper := strings.ToUpper(title)
	for _, effect := range shopifyEffects {
		if strings.Contains(titleUpper, effect) {
			return effect
		}
	}
	return ""
}

func extractFlavorFromTitle(title string) string {
	// Pattern: EFFECT - ... - FLAVOR
	parts := strings.Split(title, " - ")
	if len(parts) >= 3 {
		return strings.TrimSpace(parts[len(parts)-1])
	}
	return ""
}

func getFlavorFromOptions(options []struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}) string {
	for _, opt := range options {
		if opt.Name == "Flavor" {
			return opt.Value
		}
	}
	return ""
}

func getQuantityActual(metafields []struct {
	Node struct {
		Namespace string `json:"namespace"`
		Key       string `json:"key"`
		Value     string `json:"value"`
	} `json:"node"`
}) int {
	for _, edge := range metafields {
		if edge.Node.Namespace == "custom" && edge.Node.Key == "quantity_actual" {
			var qty int
			fmt.Sscanf(edge.Node.Value, "%d", &qty)
			if qty > 0 {
				return qty
			}
		}
	}
	return 1
}
