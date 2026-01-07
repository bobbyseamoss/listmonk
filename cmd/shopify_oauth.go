package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/labstack/echo/v4"
)

// ShopifyOAuthCallback handles the OAuth callback from Shopify after app installation.
// This endpoint exchanges the authorization code for an access token.
func (app *App) ShopifyOAuthCallback(c echo.Context) error {
	// Get query parameters
	code := c.QueryParam("code")
	shop := c.QueryParam("shop")
	hmacParam := c.QueryParam("hmac")

	app.log.Printf("ShopifyOAuthCallback: received callback from shop=%s", shop)

	// Validate required parameters
	if code == "" || shop == "" {
		return c.HTML(http.StatusBadRequest, `
			<html><body>
			<h1>Error: Missing Parameters</h1>
			<p>Missing required OAuth parameters (code or shop).</p>
			</body></html>
		`)
	}

	// Get client credentials from settings
	settings, err := app.core.GetSettings()
	if err != nil {
		return c.HTML(http.StatusInternalServerError, `
			<html><body>
			<h1>Error: Could not load settings</h1>
			</body></html>
		`)
	}
	clientID := settings.Shopify.ClientID
	clientSecret := settings.Shopify.ClientSecret

	if clientID == "" || clientSecret == "" {
		return c.HTML(http.StatusBadRequest, `
			<html><body>
			<h1>Error: Missing Configuration</h1>
			<p>Shopify Client ID and Client Secret must be configured in Settings.</p>
			<p>Go to Settings → Shopify and enter your app credentials.</p>
			</body></html>
		`)
	}

	// Verify HMAC signature
	if hmacParam != "" {
		if !verifyShopifyOAuthHMAC(c.Request().URL.RawQuery, clientSecret) {
			app.log.Printf("ShopifyOAuthCallback: HMAC verification failed")
			return c.HTML(http.StatusUnauthorized, `
				<html><body>
				<h1>Error: Invalid Signature</h1>
				<p>The OAuth request signature is invalid.</p>
				</body></html>
			`)
		}
	}

	// Exchange code for access token
	tokenURL := fmt.Sprintf("https://%s/admin/oauth/access_token", shop)

	payload := map[string]string{
		"client_id":     clientID,
		"client_secret": clientSecret,
		"code":          code,
	}

	payloadBytes, _ := json.Marshal(payload)

	resp, err := http.Post(tokenURL, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		app.log.Printf("ShopifyOAuthCallback: error exchanging code: %v", err)
		return c.HTML(http.StatusInternalServerError, fmt.Sprintf(`
			<html><body>
			<h1>Error: Token Exchange Failed</h1>
			<p>Failed to exchange authorization code: %v</p>
			</body></html>
		`, err))
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		app.log.Printf("ShopifyOAuthCallback: token exchange failed: %s", string(body))
		return c.HTML(http.StatusBadRequest, fmt.Sprintf(`
			<html><body>
			<h1>Error: Token Exchange Failed</h1>
			<p>Shopify returned an error: %s</p>
			</body></html>
		`, string(body)))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		Scope       string `json:"scope"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return c.HTML(http.StatusInternalServerError, fmt.Sprintf(`
			<html><body>
			<h1>Error: Invalid Response</h1>
			<p>Failed to parse token response: %v</p>
			</body></html>
		`, err))
	}

	app.log.Printf("ShopifyOAuthCallback: successfully obtained token with scopes: %s", tokenResp.Scope)

	// Update settings with new token
	if err := app.updateShopifyToken(shop, tokenResp.AccessToken); err != nil {
		app.log.Printf("ShopifyOAuthCallback: error saving token: %v", err)
		// Still show the token to the user even if save fails
	}

	// Return success page with token info
	return c.HTML(http.StatusOK, fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>Shopify App Installed</title>
			<style>
				body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
				.success { background: #d4edda; border: 1px solid #c3e6cb; padding: 20px; border-radius: 8px; margin-bottom: 20px; }
				.token-box { background: #f8f9fa; border: 1px solid #dee2e6; padding: 15px; border-radius: 4px; font-family: monospace; word-break: break-all; }
				.scopes { background: #e7f3ff; border: 1px solid #b6d4fe; padding: 15px; border-radius: 4px; margin: 15px 0; }
				h1 { color: #28a745; }
				.warning { background: #fff3cd; border: 1px solid #ffc107; padding: 15px; border-radius: 4px; margin: 15px 0; }
			</style>
		</head>
		<body>
			<div class="success">
				<h1>✓ Shopify App Installed Successfully!</h1>
				<p>The Listmonk integration has been installed on <strong>%s</strong></p>
			</div>

			<h2>Access Token</h2>
			<p>Your access token has been automatically saved to listmonk settings.</p>
			<div class="token-box">%s</div>

			<div class="scopes">
				<strong>Granted Scopes:</strong> %s
			</div>

			<div class="warning">
				<strong>Note:</strong> This token is shown once. It has been saved to your listmonk Shopify settings.
				If you need to regenerate it, uninstall and reinstall the app.
			</div>

			<h2>Next Steps</h2>
			<ol>
				<li>Go to <a href="/admin/settings#shopify">listmonk Settings → Shopify</a></li>
				<li>Verify the Store URL is set to: <code>%s</code></li>
				<li>Set up webhooks in Shopify for real-time sync</li>
				<li>Test the customer sync</li>
			</ol>

			<p><a href="/admin/settings">← Back to listmonk Settings</a></p>
		</body>
		</html>
	`, shop, tokenResp.AccessToken, tokenResp.Scope, shop))
}

// updateShopifyToken updates the Shopify access token in settings.
func (app *App) updateShopifyToken(shop, token string) error {
	// Get current settings
	settings, err := app.core.GetSettings()
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}

	// Update Shopify settings
	settings.Shopify.StoreURL = shop
	settings.Shopify.AccessToken = token
	settings.Shopify.Enabled = true

	// Update the shopify settings JSON
	shopifyJSON, err := json.Marshal(settings.Shopify)
	if err != nil {
		return err
	}

	if _, err := app.db.Exec(`UPDATE settings SET value = $1 WHERE key = 'shopify'`, shopifyJSON); err != nil {
		return err
	}

	app.log.Printf("updateShopifyToken: saved new token for shop %s", shop)
	return nil
}

// verifyShopifyOAuthHMAC verifies the HMAC signature of Shopify OAuth callback.
func verifyShopifyOAuthHMAC(rawQuery, secret string) bool {
	// Parse the query string
	params, err := url.ParseQuery(rawQuery)
	if err != nil {
		return false
	}

	// Get and remove the hmac parameter
	providedHMAC := params.Get("hmac")
	params.Del("hmac")

	// Sort parameters alphabetically
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build the message string
	var pairs []string
	for _, k := range keys {
		for _, v := range params[k] {
			pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
		}
	}
	message := strings.Join(pairs, "&")

	// Compute HMAC
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	computedHMAC := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(computedHMAC), []byte(providedHMAC))
}

// ShopifyOAuthStart initiates the OAuth flow by redirecting to Shopify.
func (app *App) ShopifyOAuthStart(c echo.Context) error {
	shop := c.QueryParam("shop")
	if shop == "" {
		return c.HTML(http.StatusBadRequest, `
			<html><body>
			<h1>Shopify OAuth</h1>
			<form method="GET">
				<label>Shop Domain (e.g., mystore.myshopify.com):</label><br>
				<input type="text" name="shop" placeholder="yourstore.myshopify.com" style="width: 300px; padding: 10px;"><br><br>
				<button type="submit" style="padding: 10px 20px;">Connect to Shopify</button>
			</form>
			</body></html>
		`)
	}

	// Get client ID from settings
	settings, err := app.core.GetSettings()
	if err != nil {
		return c.HTML(http.StatusInternalServerError, `
			<html><body>
			<h1>Error: Could not load settings</h1>
			</body></html>
		`)
	}
	clientID := settings.Shopify.ClientID

	if clientID == "" {
		return c.HTML(http.StatusBadRequest, `
			<html><body>
			<h1>Error: Missing Configuration</h1>
			<p>Shopify Client ID must be configured in Settings before connecting.</p>
			<p>Go to <a href="/admin/settings">Settings → Shopify</a> and enter your Client ID.</p>
			</body></html>
		`)
	}

	// Build OAuth URL
	redirectURI := fmt.Sprintf("https://%s/webhooks/shopify/oauth/callback", c.Request().Host)
	scopes := "read_customers,read_orders,read_products"

	oauthURL := fmt.Sprintf(
		"https://%s/admin/oauth/authorize?client_id=%s&scope=%s&redirect_uri=%s",
		shop,
		clientID,
		scopes,
		url.QueryEscape(redirectURI),
	)

	app.log.Printf("ShopifyOAuthStart: redirecting to %s", oauthURL)
	return c.Redirect(http.StatusFound, oauthURL)
}
