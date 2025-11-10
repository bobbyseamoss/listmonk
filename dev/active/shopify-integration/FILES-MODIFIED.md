# Shopify Integration - Complete File Manifest

**Last Updated**: 2025-11-08 14:15:00 UTC

---

## Files Created

### Backend
1. **internal/migrations/v7.0.0.go** (60 lines) ⚠️ CRITICAL FIX APPLIED
   - Purpose: Database migration for Shopify integration
   - Creates: purchase_attributions table
   - Inserts: Default Shopify settings
   - **CRITICAL FIX** (line 49): Added nested `shopify` key for proper settings persistence
   - Before: Only flat keys (shopify.enabled, shopify.webhook_secret, shopify.attribution_window_days)
   - After: Both nested key AND flat keys (required for listmonk settings pattern)

2. **internal/bounce/webhooks/shopify.go** (89 lines)
   - Purpose: Shopify webhook verification and parsing
   - Key Functions:
     - `NewShopify(webhookSecret)` - Constructor
     - `VerifyWebhook(hmacHeader, body)` - HMAC-SHA256 verification
     - `ProcessOrder(body)` - JSON parsing
   - Structs: `Shopify`, `ShopifyOrder`

3. **cmd/shopify.go** (164 lines)
   - Purpose: Webhook handler and attribution logic
   - Key Functions:
     - `ShopifyWebhook(c echo.Context)` - Main webhook endpoint
     - `attributePurchase(order)` - Attribution logic
     - `GetCampaignPurchaseStats(c echo.Context)` - Analytics endpoint
   - Routes:
     - `POST /webhooks/shopify/orders` (public)
     - `GET /api/campaigns/:id/purchases/stats` (authenticated)

### Frontend
4. **frontend/src/views/settings/shopify.vue** (120 lines)
   - Purpose: Shopify settings UI page
   - Components:
     - Enable toggle
     - Webhook URL display + copy button
     - Webhook secret input (password)
     - Attribution window selector
     - Instructions

### Documentation
5. **dev/active/shopify-integration/shopify-integration-context.md**
   - Complete implementation context
   - Bug fixes documentation
   - Configuration guide

6. **dev/active/shopify-integration/shopify-integration-tasks.md**
   - Task completion tracking
   - Testing checklist
   - Future enhancements backlog

7. **dev/active/shopify-integration/QUICKSTART.md**
   - Quick reference for new sessions
   - Testing commands
   - Troubleshooting guide

8. **dev/active/shopify-integration/FILES-MODIFIED.md** (this file)
   - Complete file manifest

9. **SHOPIFY-WEBHOOK-TESTING.md** (Testing Guide)
   - Complete testing guide for production webhooks
   - Step-by-step Shopify webhook configuration
   - SQL queries for test data and verification
   - Troubleshooting common issues
   - Test checklist and success criteria

10. **test-shopify-webhook.sh** (Test Script)
    - Alternative curl-based test script
    - Generates proper HMAC-SHA256 signatures
    - Simulates Shopify order webhook
    - Not needed if testing with real Shopify webhooks

---

## Files Modified

### Backend

9. **models/settings.go** (Modified)
   - Line ~115: Added `Shopify` struct to `Settings`
   ```go
   Shopify struct {
       Enabled               bool   `json:"enabled"`
       WebhookSecret         string `json:"webhook_secret,omitempty"`
       AttributionWindowDays int    `json:"attribution_window_days"`
   } `json:"shopify"`
   ```

10. **models/models.go** (Modified)
    - Line ~390: Added `PurchaseAttribution` struct (12 fields)
    - Line ~405: Added `CampaignPurchaseStats` struct (4 fields)
    ```go
    type PurchaseAttribution struct {
        ID             int64           `db:"id" json:"id"`
        CampaignID     null.Int        `db:"campaign_id" json:"campaign_id"`
        SubscriberID   null.Int        `db:"subscriber_id" json:"subscriber_id"`
        OrderID        string          `db:"order_id" json:"order_id"`
        OrderNumber    null.String     `db:"order_number" json:"order_number"`
        CustomerEmail  string          `db:"customer_email" json:"customer_email"`
        TotalPrice     float64         `db:"total_price" json:"total_price"`
        Currency       null.String     `db:"currency" json:"currency"`
        AttributedVia  null.String     `db:"attributed_via" json:"attributed_via"`
        Confidence     null.String     `db:"confidence" json:"confidence"`
        ShopifyData    json.RawMessage `db:"shopify_data" json:"shopify_data"`
        CreatedAt      time.Time       `db:"created_at" json:"created_at"`
        Total int `db:"total" json:"-"`
    }
    ```

11. **models/queries.go** (Modified)
    - Line ~165: Added 4 query fields
    ```go
    InsertPurchaseAttribution  *sqlx.Stmt `query:"insert-purchase-attribution"`
    FindRecentLinkClick        *sqlx.Stmt `query:"find-recent-link-click"`
    GetCampaignPurchaseStats   *sqlx.Stmt `query:"get-campaign-purchase-stats"`
    GetSubscriberByEmail       *sqlx.Stmt `query:"get-subscriber-by-email"`
    ```

12. **queries.sql** (Modified)
    - Added 4 queries after line 600:
    ```sql
    -- name: insert-purchase-attribution (RETURNING *)
    -- name: find-recent-link-click (with time window filter)
    -- name: get-campaign-purchase-stats (aggregates revenue)
    -- name: get-subscriber-by-email (LOWER email match)
    ```

13. **cmd/handlers.go** (Modified)
    - Line ~180: Added authenticated stats route
    ```go
    g.GET("/api/campaigns/:id/purchases/stats", 
          pm(hasID(a.GetCampaignPurchaseStats), "campaigns:get_analytics"))
    ```
    - Line ~250: Added public webhook route (conditional)
    ```go
    if a.cfg.ShopifyEnabled {
        g.POST("/webhooks/shopify/orders", a.ShopifyWebhook)
        a.log.Printf("Registered Shopify webhook route: POST /webhooks/shopify/orders")
    }
    ```

14. **cmd/settings.go** (Modified)
    - Line 80: Added password masking in GetSettings()
    ```go
    s.Shopify.WebhookSecret = strings.Repeat(pwdMask, 
        utf8.RuneCountInString(s.Shopify.WebhookSecret))
    ```
    - Lines 231-233: Added password preservation in UpdateSettings()
    ```go
    if set.Shopify.WebhookSecret == "" {
        set.Shopify.WebhookSecret = cur.Shopify.WebhookSecret
    }
    ```

15. **cmd/init.go** (Modified)
    - Line ~620: Added Shopify config loading
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

16. **cmd/upgrade.go** (Modified)
    - Added to migList array:
    ```go
    {"v7.0.0", migrations.V7_0_0},
    ```

### Frontend

17. **frontend/src/views/Settings.vue** (Modified)
    - Line 54: Added Shopify tab
    ```vue
    <b-tab-item :label="$t('settings.shopify.name', 'Shopify')">
      <shopify-settings :form="form" :key="key" />
    </b-tab-item>
    ```
    - Line 86: Imported ShopifySettings component
    ```javascript
    import ShopifySettings from './settings/shopify.vue';
    ```
    - Line 100: Added to components object
    ```javascript
    components: {
      // ...
      ShopifySettings,
      // ...
    }
    ```
    - Lines 193-198: Added webhook_secret password handling in onSubmit()
    ```javascript
    // Shopify webhook secret
    if (this.isDummy(form.shopify.webhook_secret)) {
        form.shopify.webhook_secret = '';
    } else if (this.hasDummy(form.shopify.webhook_secret)) {
        hasDummy = 'shopify';
    }
    ```

18. **frontend/src/views/Campaign.vue** (Modified)
    - Added data properties:
    ```javascript
    purchaseStats: null,
    purchaseStatsLoading: false,
    ```
    - Added method:
    ```javascript
    async fetchPurchaseStats() {
        this.purchaseStatsLoading = true;
        try {
            const response = await this.$api.getCampaignPurchaseStats(this.data.id);
            this.purchaseStats = response.data;
        } catch (error) {
            console.error('Error fetching purchase stats:', error);
        } finally {
            this.purchaseStatsLoading = false;
        }
    }
    ```
    - Added Purchase Analytics tab in template:
    ```vue
    <b-tab-item :label="$t('campaigns.purchaseAnalytics', 'Purchase Analytics')" 
                icon="cart">
      <div v-if="purchaseStatsLoading">
        <b-loading :active="true" />
      </div>
      <div v-else-if="purchaseStats">
        <div class="columns is-multiline">
          <!-- Total Orders, Total Revenue, Avg Order Value -->
        </div>
      </div>
    </b-tab-item>
    ```

19. **frontend/src/api/index.js** (Modified)
    - Added API method:
    ```javascript
    export const getCampaignPurchaseStats = async (id) => http.get(
      `/api/campaigns/${id}/purchases/stats`,
      { loading: models.campaigns },
    );
    ```

20. **i18n/en.json** (Modified)
    - Lines 561-576: Added 16 Shopify translation keys:
    ```json
    "settings.shopify.name": "Shopify",
    "settings.shopify.title": "Shopify Integration",
    "settings.shopify.description": "Attribute purchases to email campaigns when customers buy through your Shopify store.",
    "settings.shopify.enable": "Enable Shopify",
    "settings.shopify.webhookUrl": "Webhook URL",
    "settings.shopify.webhookUrlHelp": "Copy this URL and configure it in Shopify Admin → Settings → Notifications → Webhooks",
    "settings.shopify.webhookSecret": "Webhook Secret",
    "settings.shopify.webhookSecretHelp": "Copy this from your Shopify webhook configuration. Leave blank to keep existing value.",
    "settings.shopify.attributionWindow": "Attribution Window (Days)",
    "settings.shopify.attributionWindowHelp": "How many days after a link click should purchases be attributed to a campaign?",
    "settings.shopify.days": "days",
    "settings.shopify.howItWorks": "How it works:",
    "settings.shopify.step1": "Send campaigns with tracked links",
    "settings.shopify.step2": "Subscribers click links in your emails",
    "settings.shopify.step3": "When they purchase in Shopify, the webhook sends order data to listmonk",
    "settings.shopify.step4": "Listmonk attributes the purchase to the campaign if the subscriber clicked within the attribution window"
    ```

---

## Summary Statistics

**Total Files**: 22
- **Created**: 10 files (4 backend, 1 frontend, 5 documentation)
- **Modified**: 12 files (8 backend, 4 frontend)

**Lines of Code**:
- Backend: ~600 lines (new code)
- Frontend: ~200 lines (new code)
- Database: 1 migration, 4 queries, 1 table
- Documentation: ~2500 lines

**Database Objects**:
- Tables: 1 (purchase_attributions)
- Indexes: 4
- Queries: 4
- Foreign Keys: 2
- Settings Keys: 3

**API Endpoints**:
- Public: 1 (webhook)
- Authenticated: 1 (stats)

**Translation Keys**: 16

---

## Git Diff Summary

If you need to review changes:

```bash
# View all Shopify-related changes
git diff HEAD -- '*shopify*'

# View backend changes
git diff HEAD -- cmd/ internal/ models/

# View frontend changes
git diff HEAD -- frontend/

# View specific files
git diff HEAD -- cmd/settings.go
git diff HEAD -- frontend/src/views/Settings.vue
```

---

## Dependency Chain

### Database → Backend
- migration v7.0.0 → purchase_attributions table
- queries.sql → models/queries.go

### Backend → Frontend
- cmd/settings.go → Settings.vue (password handling)
- cmd/shopify.go → shopify.vue (webhook URL)
- cmd/shopify.go → Campaign.vue (stats endpoint)

### Frontend Internal
- Settings.vue → shopify.vue (form prop, tab)
- Campaign.vue → api/index.js (stats API call)
- All components → i18n/en.json (translations)

---

**End of File Manifest**
