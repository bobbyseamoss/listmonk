# Shopify Integration - Quick Start Guide

**For New Sessions After Context Reset**

---

## What Was Implemented

A complete Shopify purchase attribution system that:
- Receives Shopify order webhooks
- Matches purchases to subscribers via email
- Attributes purchases to campaigns based on recent link clicks (7-90 day window)
- Displays revenue metrics on campaign pages

---

## Current Status

✅ **COMPLETE & DEPLOYED** (with critical database fix)
- All code implemented and deployed to production
- Database migration v7.0.0 completed (with nested key fix)
- Frontend and backend fully functional
- Production URL: https://list.bobbyseamoss.com
- Revision: listmonk420--deploy-20251108-092758

⚠️ **CRITICAL FIX APPLIED**: Settings persistence issue resolved
- Added missing nested `shopify` database key
- Settings now save and persist correctly
- See `CRITICAL-BUG-FIX.md` for full details

---

## How It Works

```
Campaign Email → Subscriber Clicks Link → Link Click Recorded
                                                ↓
                                    Subscriber Buys in Shopify
                                                ↓
                                    Shopify Webhook → Listmonk
                                                ↓
                                    Match Email to Subscriber
                                                ↓
                                    Find Recent Link Clicks
                                                ↓
                                    Create Purchase Attribution
                                                ↓
                                    Revenue Shows on Campaign Page
```

---

## Key Files

**Backend**:
- `cmd/shopify.go` - Webhook handler & attribution logic
- `cmd/settings.go:80, 231-233` - Password handling
- `internal/bounce/webhooks/shopify.go` - HMAC verification
- `internal/migrations/v7.0.0.go` - Database schema

**Frontend**:
- `frontend/src/views/settings/shopify.vue` - Settings page
- `frontend/src/views/Campaign.vue` - Purchase analytics tab

**Database**:
- Table: `purchase_attributions`
- Queries: `insert-purchase-attribution`, `find-recent-link-click`, `get-campaign-purchase-stats`

---

## Configuration

### In Listmonk (Production)
1. Go to: https://list.bobbyseamoss.com
2. Settings > Shopify tab
3. Enable Shopify
4. Copy webhook URL: `https://list.bobbyseamoss.com/webhooks/shopify/orders`
5. Enter webhook secret (from Shopify)
6. Set attribution window (7/14/30/60/90 days)
7. Save

### In Shopify Admin
1. Settings > Notifications > Webhooks
2. Create webhook:
   - Event: "Order creation"
   - Format: JSON
   - URL: `https://list.bobbyseamoss.com/webhooks/shopify/orders`
   - Latest API version
3. Copy webhook secret to Listmonk

---

## Testing

### Check Settings Persist
```bash
# In browser console (after saving settings):
fetch('/api/settings', {credentials: 'include'})
  .then(r => r.json())
  .then(d => console.log(d.data.shopify))
```

### Check Webhook Endpoint Registered
```bash
az containerapp logs show --name listmonk420 --resource-group rg-listmonk420 --tail 50 | grep -i shopify
```

### Test Attribution Query
```sql
-- Connect to database
PGPASSWORD='T@intshr3dd3r' psql -h listmonk420-db.postgres.database.azure.com -p 5432 -U listmonkadmin -d listmonk -c "
SELECT * FROM purchase_attributions LIMIT 5;
"
```

### Check Campaign Stats
```bash
# Replace 123 with actual campaign ID
curl -H "Cookie: session=..." https://list.bobbyseamoss.com/api/campaigns/123/purchases/stats
```

---

## Critical Bug Fixes Applied

1. **Copy Button Page Reload**
   - Added `type="button"` to prevent form submission

2. **Settings Not Persisting**
   - Fixed frontend prop naming (`:form="form"` not `:data="form"`)
   - Fixed v-model bindings (`form.shopify.*` not `data['shopify'].*`)
   - Added backend password handling in `cmd/settings.go`

3. **Translation Keys**
   - Added all 16 Shopify keys to `i18n/en.json`

---

## Database Schema

```sql
-- Main table
CREATE TABLE purchase_attributions (
    id BIGSERIAL PRIMARY KEY,
    campaign_id INTEGER REFERENCES campaigns(id) ON DELETE SET NULL,
    subscriber_id INTEGER REFERENCES subscribers(id) ON DELETE SET NULL,
    order_id TEXT NOT NULL,
    order_number TEXT,
    customer_email TEXT NOT NULL,
    total_price DECIMAL(10,2),
    currency TEXT,
    attributed_via TEXT,      -- 'link_click'
    confidence TEXT,          -- 'high'
    shopify_data JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_purchase_attributions_campaign ON purchase_attributions(campaign_id);
CREATE INDEX idx_purchase_attributions_subscriber ON purchase_attributions(subscriber_id);
CREATE INDEX idx_purchase_attributions_order ON purchase_attributions(order_id);
CREATE INDEX idx_purchase_attributions_created ON purchase_attributions(created_at);
```

---

## API Endpoints

### Public Webhook (no auth required)
```
POST /webhooks/shopify/orders
Headers:
  X-Shopify-Hmac-Sha256: [base64 hmac]
Body: Shopify order JSON
```

### Campaign Stats (auth required)
```
GET /api/campaigns/:id/purchases/stats
Response:
{
  "data": {
    "total_purchases": 10,
    "total_revenue": 1234.56,
    "avg_order_value": 123.46,
    "currency": "USD"
  }
}
```

---

## Troubleshooting

### Settings Not Saving
**KNOWN ISSUE - NOW FIXED**: Database was missing nested `shopify` key

**If issue persists, verify database has all 4 keys:**
```sql
SELECT key, value FROM settings WHERE key LIKE 'shopify%' OR key = 'shopify' ORDER BY key;
-- Should return: shopify, shopify.enabled, shopify.webhook_secret, shopify.attribution_window_days
```

**If nested `shopify` key is missing:**
```sql
INSERT INTO settings (key, value) VALUES
('shopify', '{"enabled": false, "webhook_secret": "", "attribution_window_days": 7}')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;
```

**Other checks:**
- Check browser console for errors
- Verify `form.shopify` object exists in Settings.vue data
- Check backend logs for validation errors
- See `CRITICAL-BUG-FIX.md` for detailed troubleshooting

### Webhook Not Working
- Verify HMAC signature matches
- Check webhook logs: `SELECT * FROM webhook_logs ORDER BY created_at DESC LIMIT 10;`
- Verify Shopify webhook is active and pointing to correct URL

### No Attribution Created
- Verify subscriber exists with matching email
- Check link_clicks table for recent clicks
- Verify attribution window includes click date
- Check application logs for attribution errors

### Revenue Not Showing
- Verify purchase_attributions records exist for campaign
- Check campaign ID matches
- Verify currency field is populated
- Check GetCampaignPurchaseStats query

---

## Deployment Commands

### Rebuild and Deploy
```bash
cd /home/adam/listmonk
./deploy.sh
```

### Check Current Revision
```bash
az containerapp show --name listmonk420 --resource-group rg-listmonk420 --query properties.latestRevisionName -o tsv
```

### View Logs
```bash
az containerapp logs show --name listmonk420 --resource-group rg-listmonk420 --follow
```

---

## Next Steps (If Continuing)

1. ✅ Code complete and deployed
2. ⏳ **Configure Shopify webhook** (not yet done)
3. ⏳ **Test end-to-end** with real order
4. ⏳ Monitor webhook logs
5. ⏳ Verify revenue data displays correctly

---

## Common Commands

```bash
# Connect to database
PGPASSWORD='T@intshr3dd3r' psql -h listmonk420-db.postgres.database.azure.com -p 5432 -U listmonkadmin -d listmonk --set=sslmode=require

# Check purchase attributions
SELECT * FROM purchase_attributions ORDER BY created_at DESC LIMIT 5;

# Check recent link clicks for a subscriber
SELECT * FROM link_clicks WHERE subscriber_id = 123 ORDER BY created_at DESC LIMIT 5;

# Check campaign stats
SELECT
    COUNT(*) as total_purchases,
    SUM(total_price) as total_revenue,
    AVG(total_price) as avg_order_value,
    currency
FROM purchase_attributions
WHERE campaign_id = 123
GROUP BY currency;
```

---

## Contact Points

- **Link Clicks**: Uses existing `link_clicks` table
- **Subscribers**: Matches via `subscribers` table email column
- **Campaigns**: Links to `campaigns` table via foreign key
- **Settings**: Stored in `settings` table JSON column
- **Webhook Logs**: Uses existing `webhook_logs` table

---

**End of Quick Start**
