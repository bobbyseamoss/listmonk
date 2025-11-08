package main

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/knadh/listmonk/internal/bounce/webhooks"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
	"gopkg.in/volatiletech/null.v6"
)

// ShopifyWebhook handles incoming Shopify order webhooks for purchase attribution.
func (app *App) ShopifyWebhook(c echo.Context) error {
	var (
		service      = "shopify"
		eventType    = "order"
		rawReq       []byte
		err          error
		processed    = false
		errorMsg     = ""
		responseCode = http.StatusOK
	)

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

	// Parse the order data
	order, err := shopifyHandler.ProcessOrder(rawReq)
	if err != nil {
		errorMsg = fmt.Sprintf("error processing order: %v", err)
		responseCode = http.StatusBadRequest
		app.logWebhook(service, eventType, c.Request().Header, rawReq, responseCode, "", processed, errorMsg)
		return echo.NewHTTPError(responseCode, errorMsg)
	}

	// Attempt to attribute the purchase to a campaign
	if err := app.attributePurchase(order); err != nil {
		// Log the error but don't fail the webhook
		// This allows Shopify to consider the webhook successfully received
		app.log.Printf("error attributing purchase (order %d): %v", order.ID, err)
		errorMsg = fmt.Sprintf("attribution error: %v", err)
	} else {
		processed = true
	}

	// Log the webhook
	app.logWebhook(service, eventType, c.Request().Header, rawReq, responseCode, "", processed, errorMsg)

	return c.JSON(http.StatusOK, okResp{true})
}

// attributePurchase attempts to attribute a purchase to a campaign based on recent link clicks.
func (app *App) attributePurchase(order *webhooks.ShopifyOrder) error {
	// Find subscriber by email
	var sub models.Subscriber
	err := app.queries.GetSubscriberByEmail.Get(&sub, order.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("no subscriber found with email: %s", order.Email)
		}
		return fmt.Errorf("error finding subscriber: %v", err)
	}

	// Look for recent link clicks within the attribution window
	var linkClick struct {
		CampaignID int       `db:"campaign_id"`
		LinkID     int       `db:"link_id"`
		CreatedAt  null.Time `db:"created_at"`
	}

	err = app.queries.FindRecentLinkClick.Get(&linkClick, sub.ID, app.cfg.ShopifyAttributionWindowDays)
	if err != nil {
		if err == sql.ErrNoRows {
			// No recent link clicks found, no attribution
			return fmt.Errorf("no recent link clicks found for subscriber %d", sub.ID)
		}
		return fmt.Errorf("error finding link clicks: %v", err)
	}

	// Parse total price
	totalPrice, err := strconv.ParseFloat(order.TotalPrice, 64)
	if err != nil {
		app.log.Printf("error parsing total price '%s': %v", order.TotalPrice, err)
		totalPrice = 0
	}

	// Create purchase attribution record
	var attribution models.PurchaseAttribution
	err = app.queries.InsertPurchaseAttribution.Get(&attribution,
		linkClick.CampaignID,
		sub.ID,
		fmt.Sprintf("%d", order.ID),
		order.OrderNumber,
		order.Email,
		totalPrice,
		order.Currency,
		"link_click",
		"high",
		order.RawJSON,
	)

	if err != nil {
		return fmt.Errorf("error inserting purchase attribution: %v", err)
	}

	app.log.Printf("attributed purchase (order %d, $%.2f %s) to campaign %d for subscriber %d",
		order.ID, totalPrice, order.Currency, linkClick.CampaignID, sub.ID)

	return nil
}

// GetCampaignPurchaseStats returns purchase statistics for a campaign.
func (app *App) GetCampaignPurchaseStats(c echo.Context) error {
	var (
		campaignID, _ = strconv.Atoi(c.Param("id"))
		stats         models.CampaignPurchaseStats
	)

	if campaignID < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, app.i18n.T("globals.messages.invalidID"))
	}

	// Get purchase stats for the campaign
	err := app.queries.GetCampaignPurchaseStats.Get(&stats, campaignID)
	if err != nil {
		if err == sql.ErrNoRows {
			// No purchases found, return zero stats
			stats = models.CampaignPurchaseStats{
				TotalPurchases: 0,
				TotalRevenue:   0,
				AvgOrderValue:  0,
				Currency:       "USD",
			}
		} else {
			app.log.Printf("error fetching purchase stats for campaign %d: %v", campaignID, err)
			return echo.NewHTTPError(http.StatusInternalServerError, app.i18n.T("globals.messages.errorFetching"))
		}
	}

	return c.JSON(http.StatusOK, okResp{stats})
}

// GetCampaignsPerformanceSummary returns aggregate performance metrics for all campaigns in the last 30 days.
func (app *App) GetCampaignsPerformanceSummary(c echo.Context) error {
	var summary models.CampaignsPerformanceSummary

	// Get aggregate performance stats
	err := app.queries.GetCampaignsPerformanceSummary.Get(&summary)
	if err != nil {
		if err == sql.ErrNoRows {
			// No campaigns found, return zero stats
			summary = models.CampaignsPerformanceSummary{
				AvgOpenRate:         0,
				AvgClickRate:        0,
				TotalSent:           0,
				TotalOrders:         0,
				TotalRevenue:        0,
				OrderRate:           0,
				RevenuePerRecipient: 0,
			}
		} else {
			app.log.Printf("error fetching campaigns performance summary: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, app.i18n.T("globals.messages.errorFetching"))
		}
	}

	return c.JSON(http.StatusOK, okResp{summary})
}
