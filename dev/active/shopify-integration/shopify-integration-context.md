# Shopify Integration - Implementation Context

**Last Updated**: 2025-11-08 14:45:00 UTC
**Status**: ✅ COMPLETE - Deployed to production (ready for user testing)
**Deployment Revision**: listmonk420--deploy-20251108-092758
**Current Phase**: Production testing (user has webhook access in Shopify)

---

## Overview

Implemented a complete Shopify purchase attribution system that tracks when campaign subscribers make purchases in a Shopify store and attributes those purchases to email campaigns based on link clicks.

---

## Implementation Summary

### Architecture Decisions

1. **Direct Webhook Integration**
   - Shopify webhooks POST directly to listmonk (not a separate Shopify app)
   - HMAC-SHA256 signature verification for security
   - Webhook endpoint: `POST /webhooks/shopify/orders`

2. **Attribution Strategy**
   - Match customer email to subscriber
   - Find recent link clicks within configurable attribution window (7-90 days)
   - Attribute purchase to most recent campaign clicked
   - Confidence level: "high" (link click based)
   - Attribution method: "link_click"

3. **Data Storage**
   - New `purchase_attributions` table for all purchase records
   - Campaign revenue metrics aggregated on-demand via SQL
   - Complete Shopify order data stored as JSONB

4. **Configuration**
   - Settings stored in `settings` table JSON column
   - Three settings: `enabled`, `webhook_secret`, `attribution_window_days`
   - Default attribution window: 7 days

---

## Files Modified/Created

### Database Layer

**internal/migrations/v7.0.0.go** (CREATED)
- Creates `purchase_attributions` table with indexes
- Inserts default Shopify settings into `settings` table
- Migration completed successfully on production (2025-11-08)

**queries.sql** (MODIFIED - lines added after line 600)
```sql
-- name: insert-purchase-attribution
-- name: find-recent-link-click
-- name: get-campaign-purchase-stats
-- name: get-subscriber-by-email
```

### Models Layer

**models/settings.go** (MODIFIED - line ~115)
- Added `Shopify` struct to `Settings`:
  ```go
  Shopify struct {
      Enabled               bool   `json:"enabled"`
      WebhookSecret         string `json:"webhook_secret,omitempty"`
      AttributionWindowDays int    `json:"attribution_window_days"`
  } `json:"shopify"`
  ```

**models/models.go** (MODIFIED - line ~390)
- Added `PurchaseAttribution` struct
- Added `CampaignPurchaseStats` struct
- **Critical Fix**: Used `float64` for TotalPrice (NOT `null.Float` which doesn't exist)

**models/queries.go** (MODIFIED - line ~165)
- Added 4 query struct fields:
  ```go
  InsertPurchaseAttribution  *sqlx.Stmt `query:"insert-purchase-attribution"`
  FindRecentLinkClick        *sqlx.Stmt `query:"find-recent-link-click"`
  GetCampaignPurchaseStats   *sqlx.Stmt `query:"get-campaign-purchase-stats"`
  GetSubscriberByEmail       *sqlx.Stmt `query:"get-subscriber-by-email"`
  ```

### Backend Handlers

**cmd/shopify.go** (CREATED)
- `ShopifyWebhook(c echo.Context)` - Main webhook handler
  - Reads raw body for HMAC verification
  - Verifies HMAC signature
  - Processes Shopify order JSON
  - Calls `attributePurchase()`
  - Logs webhook via `app.logWebhook()`

- `attributePurchase(order *webhooks.ShopifyOrder)` - Attribution logic
  - Finds subscriber by email
  - Queries `link_clicks` table for recent clicks
  - Inserts purchase attribution record
  - Returns error if no subscriber or no recent clicks found

- `GetCampaignPurchaseStats(c echo.Context)` - Analytics endpoint
  - Returns aggregated revenue metrics for campaign
  - Response: total_purchases, total_revenue, avg_order_value, currency

**cmd/handlers.go** (MODIFIED)
- Line ~180: Added authenticated route:
  ```go
  g.GET("/api/campaigns/:id/purchases/stats", pm(hasID(a.GetCampaignPurchaseStats), "campaigns:get_analytics"))
  ```
- Line ~250: Added public webhook route (conditional on enabled):
  ```go
  if a.cfg.ShopifyEnabled {
      g.POST("/webhooks/shopify/orders", a.ShopifyWebhook)
  }
  ```

**cmd/settings.go** (MODIFIED)
- Line 80: Added password masking in `GetSettings()`:
  ```go
  s.Shopify.WebhookSecret = strings.Repeat(pwdMask, utf8.RuneCountInString(s.Shopify.WebhookSecret))
  ```
- Lines 231-233: Added password preservation in `UpdateSettings()`:
  ```go
  if set.Shopify.WebhookSecret == "" {
      set.Shopify.WebhookSecret = cur.Shopify.WebhookSecret
  }
  ```

**cmd/init.go** (MODIFIED - line ~620)
- Added config loading:
  ```go
  ShopifyEnabled               bool
  ShopifyWebhookSecret         string
  ShopifyAttributionWindowDays int

  c.ShopifyEnabled = ko.Bool("shopify.enabled")
  c.ShopifyWebhookSecret = ko.String("shopify.webhook_secret")
  c.ShopifyAttributionWindowDays = ko.Int("shopify.attribution_window_days")
  if c.ShopifyAttributionWindowDays == 0 {
      c.ShopifyAttributionWindowDays = 7
  }
  ```

**cmd/upgrade.go** (MODIFIED)
- Registered v7.0.0 migration in `migList` array

### Webhook Verification

**internal/bounce/webhooks/shopify.go** (CREATED)
- `Shopify` struct with `NewShopify()` constructor
- `VerifyWebhook(hmacHeader, body)` - HMAC-SHA256 verification
- `ProcessOrder(body)` - JSON parsing
- `ShopifyOrder` struct with all order fields
- Returns complete order data including: ID, OrderNumber, Email, TotalPrice, Currency, RawJSON

### Frontend Layer

**frontend/src/views/settings/shopify.vue** (CREATED)
- Settings page with:
  - Enable/disable toggle
  - Webhook URL display with copy button (type="button" to prevent form submission)
  - Webhook secret input (password masked)
  - Attribution window selector (7/14/30/60/90 days)
  - Instructions on how it works
- **Critical Fix**: Uses `form` prop (not `data`) for consistency with other settings
- **Critical Fix**: All v-model bindings use `form.shopify.*` (not `data['shopify'].*`)

**frontend/src/views/Settings.vue** (MODIFIED)
- Line 54: Added Shopify tab:
  ```vue
  <b-tab-item :label="$t('settings.shopify.name', 'Shopify')">
    <shopify-settings :form="form" :key="key" />
  </b-tab-item>
  ```
- Line 86: Imported ShopifySettings component
- Lines 193-198: Added webhook_secret password handling in `onSubmit()`:
  ```javascript
  if (this.isDummy(form.shopify.webhook_secret)) {
      form.shopify.webhook_secret = '';
  } else if (this.hasDummy(form.shopify.webhook_secret)) {
      hasDummy = 'shopify';
  }
  ```

**frontend/src/views/Campaign.vue** (MODIFIED)
- Added Purchase Analytics tab
- Added `purchaseStats` data property
- Added `fetchPurchaseStats()` method
- Displays: Total Orders, Total Revenue, Avg Order Value
- Tab only shows if campaign has purchases

**frontend/src/api/index.js** (MODIFIED)
- Added `getCampaignPurchaseStats(id)` API method

**i18n/en.json** (MODIFIED - lines 561-576)
- Added 16 translation keys for Shopify UI:
  ```json
  "settings.shopify.name": "Shopify",
  "settings.shopify.title": "Shopify Integration",
  "settings.shopify.description": "Attribute purchases...",
  "settings.shopify.enable": "Enable Shopify",
  "settings.shopify.webhookUrl": "Webhook URL",
  "settings.shopify.webhookUrlHelp": "Copy this URL...",
  "settings.shopify.webhookSecret": "Webhook Secret",
  "settings.shopify.webhookSecretHelp": "Copy this from...",
  "settings.shopify.attributionWindow": "Attribution Window (Days)",
  "settings.shopify.attributionWindowHelp": "How many days...",
  "settings.shopify.days": "days",
  "settings.shopify.howItWorks": "How it works:",
  "settings.shopify.step1": "Send campaigns with tracked links",
  "settings.shopify.step2": "Subscribers click links in your emails",
  "settings.shopify.step3": "When they purchase in Shopify...",
  "settings.shopify.step4": "Listmonk attributes the purchase..."
  ```

---

## Critical Bug Fixes

### Issue 1: Copy Button Page Reload
**Problem**: Copy button was triggering form submission
**Cause**: Missing `type="button"` attribute
**Fix**: Added `type="button"` to copy button (shopify.vue:27)

### Issue 2: Settings Not Persisting
**Problem**: Settings not saving to database
**Root Cause**: Three issues:
1. Frontend used `:data="form"` instead of `:form="form"` (inconsistent with other tabs)
2. Frontend used `data['shopify']` instead of `form.shopify` in v-model bindings
3. Backend missing password handling for `shopify.webhook_secret`

**Fixes**:
1. Changed prop from `data` to `form` in Settings.vue:54 and shopify.vue:88
2. Changed all v-model bindings to use `form.shopify.*`
3. Added password masking in GetSettings() (settings.go:80)
4. Added password preservation in UpdateSettings() (settings.go:231-233)

### Issue 3: Translation Keys Showing
**Problem**: UI showing "settings.shopify.name" instead of "Shopify"
**Cause**: Missing i18n translations
**Fix**: Added all 16 Shopify translation keys to i18n/en.json

### Issue 4: Type Error
**Problem**: Compilation error `undefined: null.Float`
**Cause**: Used `null.Float` which doesn't exist in null package
**Fix**: Changed to `float64` for TotalPrice field

### Issue 5: Settings Not Persisting (CRITICAL DATABASE SCHEMA BUG)
**Problem**: Shopify settings would not save or persist in database
**Symptoms**:
- Frontend form changes not saved
- Settings revert to defaults on page reload
- No errors in console or logs

**Root Cause**: Database schema mismatch with Settings struct pattern
- listmonk stores settings in two formats:
  1. **Flat keys**: `shopify.enabled`, `shopify.webhook_secret`, `shopify.attribution_window_days`
  2. **Nested key**: `shopify` with full JSON object `{"enabled": false, ...}`
- Migration v7.0.0 only created flat keys, missing the nested key
- Other nested settings (OIDC, SecurityCaptcha) have BOTH formats
- The `get-settings` query uses `JSON_OBJECT_AGG` which builds a JSON object from ALL keys
- Without the nested `shopify` key, the Settings struct couldn't properly unmarshal

**Evidence**:
```sql
-- OIDC has both formats:
SELECT key FROM settings WHERE key LIKE 'security.oidc%';
-- Returns: security.oidc, security.oidc.enabled, security.oidc.client_id, etc.

-- Shopify was missing nested key:
SELECT key FROM settings WHERE key LIKE 'shopify%';
-- Returned only: shopify.enabled, shopify.webhook_secret, shopify.attribution_window_days
```

**Fix Applied**:
1. **Updated migration** (internal/migrations/v7.0.0.go:49):
   ```go
   INSERT INTO settings (key, value) VALUES
   ('shopify', '{"enabled": false, "webhook_secret": "", "attribution_window_days": 7}'),
   ('shopify.enabled', 'false'),
   ('shopify.webhook_secret', '""'),
   ('shopify.attribution_window_days', '7')
   ```

2. **Manually inserted missing key in production**:
   ```sql
   INSERT INTO settings (key, value) VALUES
   ('shopify', '{"enabled": false, "webhook_secret": "", "attribution_window_days": 7}')
   ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;
   ```

3. **Redeployed** (revision: listmonk420--deploy-20251108-092758)

**Verification**:
```sql
SELECT key, value FROM settings WHERE key LIKE 'shopify%' OR key = 'shopify' ORDER BY key;
-- Must return 4 rows: shopify, shopify.enabled, shopify.webhook_secret, shopify.attribution_window_days
```

**Why This Pattern Exists**:
- Flat keys allow individual field updates via SQL
- Nested key enables efficient bulk retrieval via `JSON_OBJECT_AGG`
- Go struct unmarshaling expects nested format for struct fields
- Both formats must exist for settings to work properly

**Lesson Learned**:
When adding new nested struct fields to Settings model, ALWAYS add both:
1. Individual flat keys for each field
2. Single nested key with complete JSON object

---

## Database Schema

### purchase_attributions Table

```sql
CREATE TABLE purchase_attributions (
    id BIGSERIAL PRIMARY KEY,
    campaign_id INTEGER REFERENCES campaigns(id) ON DELETE SET NULL,
    subscriber_id INTEGER REFERENCES subscribers(id) ON DELETE SET NULL,
    order_id TEXT NOT NULL,
    order_number TEXT,
    customer_email TEXT NOT NULL,
    total_price DECIMAL(10,2),
    currency TEXT,
    attributed_via TEXT,
    confidence TEXT,
    shopify_data JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_purchase_attributions_campaign ON purchase_attributions(campaign_id);
CREATE INDEX idx_purchase_attributions_subscriber ON purchase_attributions(subscriber_id);
CREATE INDEX idx_purchase_attributions_order ON purchase_attributions(order_id);
CREATE INDEX idx_purchase_attributions_created ON purchase_attributions(created_at);
```

### Settings JSON Structure

```json
{
  "shopify": {
    "enabled": false,
    "webhook_secret": "",
    "attribution_window_days": 7
  }
}
```

---

## Testing Notes

### Manual Testing Checklist
- [x] Settings page loads without errors
- [x] Enable/disable toggle works
- [x] Webhook URL displays correctly
- [x] Copy button copies URL without page reload
- [x] Webhook secret can be entered and saved
- [x] Webhook secret is masked on reload (••••••)
- [x] Attribution window selector works
- [x] Settings persist after save and page reload
- [x] Translation labels display correctly

### Production Testing (READY)
- [x] Created comprehensive testing guide (`SHOPIFY-WEBHOOK-TESTING.md`)
- [ ] User to send test Shopify webhook to verify HMAC verification
- [ ] User to verify purchase attribution with test order
- [ ] User to check campaign analytics tab displays revenue
- [ ] User to verify attribution window logic with different time ranges

**Testing Resources Created**:
- `/home/adam/listmonk/SHOPIFY-WEBHOOK-TESTING.md` - Complete step-by-step guide
- `/home/adam/listmonk/test-shopify-webhook.sh` - Alternative curl-based test script
- SQL queries for finding test subscribers with recent clicks
- Verification queries for webhook logs and purchase attributions
- Troubleshooting guide for common issues

---

## Deployment History

1. **First deployment** (2025-11-08 08:06:33) - Initial implementation with i18n
2. **Second deployment** (2025-11-08 09:10:07) - Frontend fixes (copy button, prop naming)
3. **Third deployment** (2025-11-08 09:14:24) - Backend password handling fixes
4. **Fourth deployment** (2025-11-08 09:27:58) - **CRITICAL FIX**: Database schema fix (added nested `shopify` key)

**Current Production Revision**: `listmonk420--deploy-20251108-092758`

### Deployment 4 Details (Critical Fix)
**Problem Solved**: Settings not persisting due to missing nested database key
**Changes**:
- Updated migration v7.0.0 to include nested `shopify` key
- Manually added missing key to production database
- No code changes, only migration and database fix

**Production Database State**:
```sql
-- Verified 4 keys exist:
shopify                         | {"enabled": false, "webhook_secret": "", "attribution_window_days": 7}
shopify.enabled                 | false
shopify.webhook_secret          | ""
shopify.attribution_window_days | 7
```

---

## Configuration for Shopify

1. **In Listmonk** (https://list.bobbyseamoss.com):
   - Go to Settings > Shopify
   - Enable Shopify integration
   - Copy the webhook URL: `https://list.bobbyseamoss.com/webhooks/shopify/orders`
   - Enter your webhook secret (from Shopify)
   - Set attribution window (default: 7 days)
   - Click Save

2. **In Shopify Admin**:
   - Go to Settings > Notifications > Webhooks
   - Create new webhook
   - Event: "Order creation"
   - Format: JSON
   - URL: `https://list.bobbyseamoss.com/webhooks/shopify/orders`
   - Webhook API version: Latest
   - Copy the webhook secret to Listmonk

---

## Known Limitations

1. **Attribution Method**: Only link clicks (not campaign views or other engagement)
2. **Time Window**: Fixed window from link click (not sliding window)
3. **Multi-Touch**: Last-click attribution (doesn't split credit across multiple campaigns)
4. **Currency**: Assumes single currency per campaign (doesn't convert)
5. **Refunds**: Not tracked (only initial purchases)

---

## Future Enhancements

- [ ] Support for refund webhooks
- [ ] Multi-touch attribution (first-click, linear, time-decay)
- [ ] Currency conversion for multi-currency stores
- [ ] Purchase value vs. shipping/tax breakdown
- [ ] Product-level attribution (which products from which campaigns)
- [ ] Attribution reports/dashboards
- [ ] Webhook retry logic for failed deliveries
- [ ] Bulk import of historical orders

---

## Related Files for Reference

**Password Handling Pattern**:
- See SMTP password handling in `cmd/settings.go` lines 64-66, 140-146
- See bounce password handling in `cmd/settings.go` lines 67-69, 173-181

**Webhook Patterns**:
- See bounce webhook handler in `cmd/bounces.go`
- See webhook logging in `cmd/webhooks.go`

**Settings UI Patterns**:
- See SMTP settings in `frontend/src/views/settings/smtp.vue`
- See bounce settings in `frontend/src/views/settings/bounces.vue`

---

## Contact Points with Other Systems

1. **Link Clicks Table**: Uses existing `link_clicks` table for attribution
2. **Campaigns Table**: Stats displayed on campaign detail page
3. **Subscribers Table**: Email matching for attribution
4. **Settings System**: Integrates with existing settings infrastructure
5. **Webhook Logging**: Uses existing webhook logging system

---

## Code Review Notes

**Security Considerations**:
- HMAC signature verification prevents spoofed webhooks
- Webhook secret stored as masked password (same as SMTP)
- No SQL injection (uses prepared statements)
- Input validation on all fields

**Performance Considerations**:
- Indexes on all foreign keys and query columns
- Single query for link click lookup (with time range filter)
- Stats aggregated on-demand (not cached)
- JSONB storage for raw Shopify data (future querying)

**Error Handling**:
- Webhook returns 200 even if attribution fails (Shopify success)
- Attribution errors logged but don't fail webhook
- Missing subscriber or link clicks return descriptive errors
- All errors captured in webhook log table

---

**End of Context Document**
