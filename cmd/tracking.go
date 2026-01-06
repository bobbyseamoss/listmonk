package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/knadh/listmonk/internal/tracking"
	"github.com/knadh/listmonk/models"
	"github.com/labstack/echo/v4"
)

// handleServeTrackingJS serves the onsite tracking JavaScript library with embedded configuration.
func (a *App) handleServeTrackingJS(c echo.Context) error {
	// Check if onsite tracking is enabled
	if !a.cfg.OnsiteTrackingEnabled {
		return c.String(http.StatusNotFound, "// Onsite tracking is disabled")
	}

	// Build the JavaScript with embedded configuration
	js := buildTrackingJS(
		a.urlCfg.RootURL,
		a.cfg.OnsiteTrackingIdentityParam,
		a.cfg.OnsiteTrackingCookieName,
		a.cfg.OnsiteTrackingCookieExpiryDays,
		a.cfg.OnsiteTrackingTrackPageViews,
		a.cfg.OnsiteTrackingTrackProducts,
	)

	c.Response().Header().Set("Content-Type", "application/javascript; charset=utf-8")
	c.Response().Header().Set("Cache-Control", "public, max-age=3600")
	return c.String(http.StatusOK, js)
}

// setCORSHeaders sets CORS headers for tracking endpoints.
func (a *App) setCORSHeaders(c echo.Context) bool {
	origin := c.Request().Header.Get("Origin")
	if origin == "" {
		return true // No origin, allow (same-origin request)
	}

	// Check if origin is allowed
	if !isAllowedOrigin(origin, a.cfg.OnsiteTrackingAllowedDomains) {
		return false
	}

	// Set CORS headers
	c.Response().Header().Set("Access-Control-Allow-Origin", origin)
	c.Response().Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	c.Response().Header().Set("Access-Control-Allow-Headers", "Content-Type")
	c.Response().Header().Set("Access-Control-Max-Age", "86400")
	return true
}

// handleTrackEventOptions handles CORS preflight for track endpoint.
func (a *App) handleTrackEventOptions(c echo.Context) error {
	if !a.setCORSHeaders(c) {
		return c.NoContent(http.StatusForbidden)
	}
	return c.NoContent(http.StatusNoContent)
}

// handleTrackEvent receives tracking events from the JavaScript library.
func (a *App) handleTrackEvent(c echo.Context) error {
	// Check if onsite tracking is enabled
	if !a.cfg.OnsiteTrackingEnabled {
		return c.JSON(http.StatusOK, okResp{true})
	}

	// Set CORS headers
	if !a.setCORSHeaders(c) {
		return c.JSON(http.StatusForbidden, okResp{false})
	}

	// Parse request body
	var req models.TrackingBatchRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, okResp{false})
	}

	// Get client IP if recording is enabled
	clientIP := ""
	if a.cfg.OnsiteTrackingRecordIPAddress {
		clientIP = c.RealIP()
	}

	// Process each event
	for _, event := range req.Events {
		// Skip if no browser ID
		if event.BrowserID == "" {
			continue
		}

		// Look up subscriber by browser ID
		sub, err := a.core.GetBrowserSubscriber(event.BrowserID)
		if err != nil || sub == nil {
			// Not an identified browser, skip
			continue
		}

		// Record the event
		_, _ = a.core.RecordSiteEvent(sub.ID, event.SessionID, &event, clientIP)

		// Update last seen
		_ = a.core.UpdateBrowserLastSeen(event.BrowserID)
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleIdentifyVisitor handles explicit visitor identification via email.
func (a *App) handleIdentifyVisitor(c echo.Context) error {
	// Check if onsite tracking is enabled
	if !a.cfg.OnsiteTrackingEnabled {
		return c.JSON(http.StatusOK, okResp{true})
	}

	// Set CORS headers
	if !a.setCORSHeaders(c) {
		return c.JSON(http.StatusForbidden, okResp{false})
	}

	// Parse request body
	var req models.IdentifyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, okResp{false})
	}

	if req.BrowserID == "" || req.Email == "" {
		return c.JSON(http.StatusBadRequest, okResp{false})
	}

	// Look up subscriber by email
	sub, err := a.core.GetSubscriberByEmail(req.Email)
	if err != nil || sub == nil {
		// Subscriber not found, just return OK (don't leak info)
		return c.JSON(http.StatusOK, okResp{true})
	}

	// Link browser to subscriber
	_, err = a.core.IdentifyBrowser(req.BrowserID, sub.ID, models.IdentifiedViaAPICall, 0)
	if err != nil {
		a.log.Printf("error identifying browser: %v", err)
	}

	return c.JSON(http.StatusOK, okResp{true})
}

// handleGetSubscriberActivity returns site activity for a subscriber (admin API).
func (a *App) handleGetSubscriberActivity(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid subscriber ID")
	}

	// Get pagination params
	pg := a.pg.NewFromURL(c.Request().URL.Query())

	events, total, err := a.core.GetSubscriberSiteActivity(id, pg.PerPage, pg.Offset)
	if err != nil {
		return err
	}

	pg.SetTotal(total)

	return c.JSON(http.StatusOK, okResp{Data: struct {
		Results    []models.SiteEvent `json:"results"`
		Total      int                `json:"total"`
		Page       int                `json:"page"`
		PerPage    int                `json:"per_page"`
		TotalPages int                `json:"total_pages"`
	}{
		Results:    events,
		Total:      total,
		Page:       pg.Page,
		PerPage:    pg.PerPage,
		TotalPages: pg.TotalPages,
	}})
}

// handleGetSiteTrackingStats returns aggregate site tracking statistics (admin API).
func (a *App) handleGetSiteTrackingStats(c echo.Context) error {
	stats, err := a.core.GetSiteTrackingStats()
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, okResp{Data: stats})
}

// handleIdentifyFromToken processes identity token from email link redirects.
// This is called from LinkRedirect when _lmx parameter is present.
func (a *App) handleIdentifyFromToken(browserID, tokenStr string) error {
	if !a.cfg.OnsiteTrackingEnabled {
		return nil
	}

	// Validate the token
	token, err := tracking.ValidateToken(tokenStr, a.cfg.OnsiteTrackingEncryptionKey, 24)
	if err != nil {
		a.log.Printf("invalid identity token: %v", err)
		return err
	}

	// Get subscriber ID from UUID
	subID, err := a.core.GetSubscriberIDByUUID(token.SubscriberUUID)
	if err != nil {
		return err
	}

	// Get campaign ID if present
	campID := 0
	if token.CampaignUUID != "" {
		// Could look up campaign ID here, but for now just use 0
		// The campaign UUID is stored in the known_browsers table via identified_via
	}

	// Link browser to subscriber
	_, err = a.core.IdentifyBrowser(browserID, subID, models.IdentifiedViaEmailClick, campID)
	if err != nil {
		a.log.Printf("error identifying browser from token: %v", err)
		return err
	}

	return nil
}

// isAllowedOrigin checks if the origin is in the allowed domains list.
func isAllowedOrigin(origin string, allowedDomains []string) bool {
	if len(allowedDomains) == 0 {
		return true // No restrictions
	}

	// Extract domain from origin
	origin = strings.TrimPrefix(origin, "https://")
	origin = strings.TrimPrefix(origin, "http://")

	for _, domain := range allowedDomains {
		if origin == domain || strings.HasSuffix(origin, "."+domain) {
			return true
		}
	}

	return false
}

// buildTrackingJS returns the JavaScript tracking library with embedded configuration.
func buildTrackingJS(rootURL, identityParam, cookieName string, cookieExpiryDays int, trackPageViews, trackProducts bool) string {
	config := map[string]interface{}{
		"endpoint":         rootURL + "/public/track",
		"identifyEndpoint": rootURL + "/public/identify",
		"identityParam":    identityParam,
		"cookieName":       cookieName,
		"cookieExpiryDays": cookieExpiryDays,
		"trackPageViews":   trackPageViews,
		"trackProducts":    trackProducts,
	}

	configJSON, _ := json.Marshal(config)

	return `(function() {
	'use strict';

	var CONFIG = ` + string(configJSON) + `;

	// Generate or retrieve browser ID
	function getBrowserId() {
		var id = getCookie(CONFIG.cookieName);
		if (!id) {
			id = generateId();
			setCookie(CONFIG.cookieName, id, CONFIG.cookieExpiryDays);
		}
		return id;
	}

	// Generate random ID
	function generateId() {
		var chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
		var id = '';
		for (var i = 0; i < 22; i++) {
			id += chars.charAt(Math.floor(Math.random() * chars.length));
		}
		return id;
	}

	// Generate session ID
	function getSessionId() {
		var key = CONFIG.cookieName + '_sess';
		var id = sessionStorage.getItem(key);
		if (!id) {
			id = generateUUID();
			sessionStorage.setItem(key, id);
		}
		return id;
	}

	// Generate UUID
	function generateUUID() {
		return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
			var r = Math.random() * 16 | 0;
			var v = c === 'x' ? r : (r & 0x3 | 0x8);
			return v.toString(16);
		});
	}

	// Cookie helpers
	function setCookie(name, value, days) {
		var expires = '';
		if (days) {
			var date = new Date();
			date.setTime(date.getTime() + (days * 24 * 60 * 60 * 1000));
			expires = '; expires=' + date.toUTCString();
		}
		document.cookie = name + '=' + encodeURIComponent(value) + expires + '; path=/; SameSite=Lax';
	}

	function getCookie(name) {
		var nameEQ = name + '=';
		var ca = document.cookie.split(';');
		for (var i = 0; i < ca.length; i++) {
			var c = ca[i];
			while (c.charAt(0) === ' ') c = c.substring(1);
			if (c.indexOf(nameEQ) === 0) return decodeURIComponent(c.substring(nameEQ.length));
		}
		return null;
	}

	// Check for identity token in URL
	function checkIdentityToken() {
		var params = new URLSearchParams(window.location.search);
		var token = params.get(CONFIG.identityParam);
		if (token) {
			// Store token temporarily - server will validate on first event
			sessionStorage.setItem(CONFIG.cookieName + '_token', token);
			// Clean URL
			params.delete(CONFIG.identityParam);
			var newUrl = window.location.pathname + (params.toString() ? '?' + params.toString() : '') + window.location.hash;
			window.history.replaceState({}, '', newUrl);
		}
	}

	// Send tracking event
	function sendEvent(event) {
		var browserId = getBrowserId();
		event.browser_id = browserId;
		event.session_id = getSessionId();
		event.timestamp = Date.now();

		var xhr = new XMLHttpRequest();
		xhr.open('POST', CONFIG.endpoint, true);
		xhr.setRequestHeader('Content-Type', 'application/json');
		xhr.send(JSON.stringify({ events: [event] }));
	}

	// Track page view
	function trackPageView(properties) {
		sendEvent({
			event_type: 'page_view',
			page_url: window.location.href,
			page_title: document.title,
			referrer: document.referrer,
			properties: properties || {}
		});
	}

	// Track custom event
	function track(eventName, properties) {
		sendEvent({
			event_type: 'custom',
			event_name: eventName,
			page_url: window.location.href,
			page_title: document.title,
			referrer: document.referrer,
			properties: properties || {}
		});
	}

	// Identify visitor by email
	function identify(email, properties) {
		var xhr = new XMLHttpRequest();
		xhr.open('POST', CONFIG.identifyEndpoint, true);
		xhr.setRequestHeader('Content-Type', 'application/json');
		xhr.send(JSON.stringify({
			browser_id: getBrowserId(),
			email: email,
			properties: properties || {}
		}));
	}

	// Auto-detect Shopify product
	function detectShopifyProduct() {
		var product = null;

		// Try ShopifyAnalytics global
		if (window.ShopifyAnalytics && window.ShopifyAnalytics.meta && window.ShopifyAnalytics.meta.product) {
			var p = window.ShopifyAnalytics.meta.product;
			product = {
				product_id: String(p.id),
				product_name: p.title || '',
				product_vendor: p.vendor || '',
				product_price: p.price ? parseFloat(p.price) / 100 : 0,
				product_url: window.location.href
			};
		}

		// Try meta tags
		if (!product) {
			var ogType = document.querySelector('meta[property="og:type"]');
			if (ogType && ogType.content === 'product') {
				product = {
					product_name: (document.querySelector('meta[property="og:title"]') || {}).content || document.title,
					product_url: window.location.href,
					product_image: (document.querySelector('meta[property="og:image"]') || {}).content || '',
					product_price: parseFloat((document.querySelector('meta[property="product:price:amount"]') || {}).content || 0)
				};
			}
		}

		// Try window.meta (Shopify theme variable)
		if (!product && window.meta && window.meta.product) {
			product = {
				product_id: String(window.meta.product.id),
				product_name: window.meta.product.title || '',
				product_vendor: window.meta.product.vendor || '',
				product_url: window.location.href
			};
		}

		return product;
	}

	// Track product view
	function trackProduct(product) {
		sendEvent({
			event_type: 'viewed_product',
			page_url: window.location.href,
			page_title: document.title,
			referrer: document.referrer,
			properties: product
		});
	}

	// Initialize
	function init() {
		// Check for identity token
		checkIdentityToken();

		// Auto-track page view
		if (CONFIG.trackPageViews) {
			trackPageView();
		}

		// Auto-track product views
		if (CONFIG.trackProducts) {
			var product = detectShopifyProduct();
			if (product) {
				trackProduct(product);
			}
		}
	}

	// Expose API
	window.listmonk = {
		track: track,
		trackPageView: trackPageView,
		trackProduct: trackProduct,
		identify: identify,
		getBrowserId: getBrowserId
	};

	// Run on DOM ready
	if (document.readyState === 'loading') {
		document.addEventListener('DOMContentLoaded', init);
	} else {
		init();
	}
})();
`
}
