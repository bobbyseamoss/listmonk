# Shopify Integration - Task List

**Last Updated**: 2025-11-10 04:35:00 ET
**Status**: ✅ ALL TASKS COMPLETE (Including UTM-Based Attribution)

---

## Phase 1: Database & Models ✅

- [x] Create migration v7.0.0
  - [x] Create `purchase_attributions` table
  - [x] Add indexes (campaign_id, subscriber_id, order_id, created_at)
  - [x] Insert default Shopify settings into `settings` table
  - [x] Run migration on production database

- [x] Add Shopify models to `models/settings.go`
  - [x] Create `Shopify` struct (enabled, webhook_secret, attribution_window_days)
  - [x] Add to `Settings` struct

- [x] Add purchase models to `models/models.go`
  - [x] Create `PurchaseAttribution` struct
  - [x] Create `CampaignPurchaseStats` struct
  - [x] Fix: Use `float64` for TotalPrice (not `null.Float`)

- [x] Add SQL queries to `queries.sql`
  - [x] insert-purchase-attribution
  - [x] find-recent-link-click
  - [x] get-campaign-purchase-stats
  - [x] get-subscriber-by-email

- [x] Register queries in `models/queries.go`
  - [x] Add all 4 query struct fields

---

## Phase 2: Backend Implementation ✅

- [x] Create webhook verification layer
  - [x] Create `internal/bounce/webhooks/shopify.go`
  - [x] Implement HMAC-SHA256 verification
  - [x] Implement JSON order parsing
  - [x] Define `ShopifyOrder` struct

- [x] Create webhook handler
  - [x] Create `cmd/shopify.go`
  - [x] Implement `ShopifyWebhook()` endpoint
  - [x] Implement `attributePurchase()` logic
  - [x] Implement `GetCampaignPurchaseStats()` endpoint

- [x] Register routes in `cmd/handlers.go`
  - [x] Add webhook route (public, conditional on enabled)
  - [x] Add stats route (authenticated)

- [x] Add configuration loading in `cmd/init.go`
  - [x] Load `shopify.enabled`
  - [x] Load `shopify.webhook_secret`
  - [x] Load `shopify.attribution_window_days` (default: 7)

- [x] Add password handling in `cmd/settings.go`
  - [x] Mask webhook_secret in `GetSettings()`
  - [x] Preserve webhook_secret in `UpdateSettings()`

- [x] Register migration in `cmd/upgrade.go`
  - [x] Add v7.0.0 to migList

---

## Phase 3: Frontend Implementation ✅

- [x] Create Shopify settings page
  - [x] Create `frontend/src/views/settings/shopify.vue`
  - [x] Add enable toggle
  - [x] Add webhook URL display
  - [x] Add copy button (with type="button")
  - [x] Add webhook secret input (password masked)
  - [x] Add attribution window selector
  - [x] Add instructions/documentation
  - [x] Fix: Use `form` prop (not `data`)
  - [x] Fix: Use `form.shopify.*` in v-model bindings

- [x] Add Shopify tab to Settings
  - [x] Import ShopifySettings in `Settings.vue`
  - [x] Add tab with correct prop (`:form="form"`)
  - [x] Add webhook_secret password handling in `onSubmit()`

- [x] Add Purchase Analytics to Campaign view
  - [x] Add Purchase Analytics tab to `Campaign.vue`
  - [x] Add `purchaseStats` data property
  - [x] Add `fetchPurchaseStats()` method
  - [x] Display total orders, revenue, avg order value

- [x] Add API method
  - [x] Add `getCampaignPurchaseStats()` to `api/index.js`

- [x] Add translations
  - [x] Add 16 Shopify translation keys to `i18n/en.json`

---

## Phase 4: Testing & Deployment ✅

- [x] Fix compilation errors
  - [x] Fix: `null.Float` → `float64`
  - [x] Fix: Missing query definitions in queries.go
  - [x] Fix: Unused import in shopify.go

- [x] Fix frontend issues
  - [x] Fix: Copy button causing page reload
  - [x] Fix: Settings not persisting (prop naming)
  - [x] Fix: Translation keys showing instead of labels

- [x] Fix backend issues
  - [x] Fix: Database connection without SSL
  - [x] Fix: Migration not running (missing env var)
  - [x] Fix: Backend password handling missing

- [x] Deploy to Azure
  - [x] Build Docker image with all changes
  - [x] Push to Azure Container Registry
  - [x] Deploy to Azure Container Apps
  - [x] Verify application starts correctly
  - [x] Current revision: listmonk420--deploy-20251108-091424

---

## Phase 5: Production Testing ✅

### Testing Documentation Created ✅
- [x] Created comprehensive testing guide: `SHOPIFY-WEBHOOK-TESTING.md`
- [x] Documented step-by-step webhook setup process
- [x] Added SQL queries for verification
- [x] Included troubleshooting for common issues
- [x] Created test checklist

### Production Testing Completed ✅
- [x] Shopify webhook configured and tested
- [x] End-to-end attribution working
- [x] 2 real orders attributed successfully
- [x] Revenue data showing in campaigns page

---

## Phase 6: Attribution Logic Changes (Nov 9, 2025) ✅

### Attribution Method Change ✅
- [x] Changed from email-open-based to subscriber-based attribution
  - [x] Removed email open requirement
  - [x] Changed `attributed_via` from 'email_open' to 'is_subscriber'
  - [x] Changed `confidence` from 'medium' to 'high'
  - [x] Updated `cmd/shopify.go` attribution logic

### Three-Tier Campaign Fallback ✅
- [x] Tier 1: Find most recent delivered campaign (azure_delivery_events)
- [x] Tier 2: Find most recent running campaign for subscriber's lists (NEW)
- [x] Tier 3: Set campaign_id to NULL if no running campaigns
- [x] Implemented in `cmd/shopify.go` (lines 85-132)
- [x] Implemented in `reprocess-shopify-webhooks.sh` (lines 94-115)

### Retroactive Processing ✅
- [x] Created `reprocess-shopify-webhooks.sh` script
- [x] Ran retroactive processing on existing webhooks
- [x] Successfully attributed 2 existing orders to campaign 64
- [x] Total revenue: $90.00

### Frontend Display Fix ✅
- [x] Fixed campaigns page showing $0 instead of actual purchase data
- [x] Root cause: snake_case vs camelCase property names
- [x] Updated `frontend/src/views/Campaigns.vue` (lines 175-186)
- [x] Changed to camelCase: purchaseOrders, purchaseRevenue, purchaseCurrency
- [x] Verified fix with Playwright automation

### Azure Webhook Attribution Fix ✅
- [x] Fixed "sql: no rows in result set" errors for paused campaigns
- [x] Updated fallback queries in `cmd/bounce.go` (lines 317, 472)
- [x] Now includes campaigns with status: running, finished, paused, cancelled
- [x] Verified webhooks processing correctly for campaign 64 (paused)

### Deployment ✅
- [x] Built and deployed all changes to production
- [x] Deployment time: 2025-11-10 00:32 UTC
- [x] Revision: listmonk420--deploy-20251109-193154
- [x] Verified no errors in container logs
- [x] Verified Azure webhooks processing successfully

---

## Phase 7: UTM-Based Attribution for Non-Subscribers (Nov 10, 2025) ✅

### Problem Identification ✅
- [x] Discovered non-subscriber purchase not attributed (order #6810503840022)
- [x] Customer email not in subscribers table: v6ss3@2200freefonts.com
- [x] Found UTM parameters in Shopify webhook: `landing_site` field
- [x] Confirmed UTM contained: `utm_source=listmonk&utm_medium=email`

### Solution Implementation ✅
- [x] Added `LandingSite` field to `ShopifyOrder` struct
- [x] Created `containsUTMSource()` helper function
- [x] Implemented UTM-based attribution logic for non-subscribers
- [x] Added new attribution type: `attributed_via='utm_listmonk'`, `confidence='medium'`
- [x] Modified `cmd/shopify.go` attribution flow to check UTM when no subscriber

### Attribution Logic Flow ✅
- [x] Subscriber found → Use three-tier logic (existing)
- [x] No subscriber + has `utm_source=listmonk` → Find running campaign (NEW)
- [x] No subscriber + no UTM → Return error (no attribution)

### Files Modified ✅
- [x] `/home/adam/listmonk/internal/bounce/webhooks/shopify.go` (line 26: add LandingSite)
- [x] `/home/adam/listmonk/cmd/shopify.go` (lines 3, 98-131, 225-237: UTM logic)

### Testing & Verification ✅
- [x] Created verification script: `verify-purchase-attribution.js`
- [x] Manually attributed missed order to campaign 64
- [x] Verified API endpoint returns 4 purchases, $141 revenue
- [x] Confirmed attribution breakdown in database

### Deployment ✅
- [x] Built and deployed to production
- [x] Revision: `listmonk420--deploy-20251110-042929`
- [x] Verified in production with API call
- [x] All 4 purchases showing correctly

### Campaign 64 Final Stats ✅
- [x] Total Purchases: 4
- [x] Total Revenue: $141.00 USD
- [x] Breakdown: 3 subscribers ($99) + 1 non-subscriber with UTM ($42)

---

## Future Enhancements (BACKLOG)

### Priority 1: Core Improvements
- [ ] Add refund webhook support
- [ ] Add webhook retry logic for failures
- [ ] Add attribution confidence levels beyond "high"
- [ ] Add product-level attribution (which products from which campaigns)

### Priority 2: Analytics & Reporting
- [ ] Create attribution reports dashboard
- [ ] Add conversion rate metrics to campaigns
- [ ] Add revenue trends over time
- [ ] Add top performing campaigns by revenue
- [ ] Export attribution data to CSV

### Priority 3: Advanced Attribution
- [ ] Multi-touch attribution models (first-click, linear, time-decay)
- [ ] Attribution across multiple campaigns
- [ ] Time-to-purchase analytics
- [ ] Customer journey tracking
- [ ] Add click-through attribution (attributed_via = 'click_through', confidence = 'very_high')
- [ ] Add attribution window enforcement (currently unlimited)

### Priority 4: Integration Enhancements
- [ ] Currency conversion for multi-currency stores
- [ ] Purchase value vs. shipping/tax breakdown
- [ ] Bulk import of historical orders
- [ ] Integration with other e-commerce platforms (WooCommerce, Magento)

### Priority 5: Testing & Monitoring
- [ ] Unit tests for attribution logic
- [ ] Integration tests for webhook handling
- [ ] Performance testing with high volume
- [ ] Monitoring dashboard for webhook failures

### Priority 6: Performance Optimization
- [ ] Add composite index on subscriber_lists(subscriber_id) if needed
- [ ] Cache subscriber-to-campaign mappings for high-volume shops
- [ ] Pre-compute "most likely campaign" for each subscriber

---

## Task Completion Summary

**Phase 1-5 Tasks**: 47 ✅
**Phase 6 Tasks (Attribution Changes)**: 14 ✅
**Phase 7 Tasks (UTM Attribution)**: 17 ✅
**Total Completed**: 78 ✅
**In Progress**: 0
**Pending**: 0
**Backlog**: 25+ (future enhancements)

**Completion Rate**: 100% (all requested features)
**Deployment Status**: ✅ Deployed to production (Rev: listmonk420--deploy-20251110-042929)
**Production Status**: ✅ Fully operational, no errors

---

## Documentation Created

1. `SHOPIFY-WEBHOOK-TESTING.md` - Initial testing guide (Nov 8)
2. `SESSION-SUMMARY-NOV-9-ATTRIBUTION-CHANGES.md` - Attribution changes session (Nov 9)
3. `SESSION-NOV-10-UTM-ATTRIBUTION.md` - UTM-based attribution session (Nov 10)
4. `shopify-integration-context.md` - Implementation context
5. `shopify-integration-tasks.md` - This file (task tracking)

---

## Key Technical Insights

### Attribution Logic Evolution
1. **Original** (Nov 8): Email-open + attribution window → campaign_id
2. **Revised** (Nov 9): Subscriber check → three-tier campaign lookup → campaign_id or NULL
3. **Current** (Nov 10): Subscriber OR UTM check → three-tier OR running campaign → campaign_id or NULL

### Why Three Tiers?
- **Tier 1** (Most Accurate): Subscriber actually received this campaign (azure_delivery_events)
- **Tier 2** (Best Guess): Subscriber in list targeted by running campaign (handles deleted campaigns)
- **Tier 3** (Fallback): No running campaigns, but track as subscriber purchase (campaign_id = NULL)

### Performance Considerations
- Tier 1 query: Single table, fast (<10ms)
- Tier 2 query: Three-way join, may need optimization for high volume
- Webhook processing: 1-2 queries per webhook, handles <100/day easily

---

## Next Immediate Steps

**No immediate steps required.** All work is complete and deployed.

### For Future Sessions:
1. Monitor webhook logs for any attribution issues
2. Check attribution accuracy with real customer purchases
3. Consider adding click-through attribution for higher confidence
4. Consider adding attribution window to prevent stale attributions

---

**End of Task List**
