# Shopify Webhook Testing Guide

## Step 1: Enable Shopify in Listmonk

1. Go to https://list.bobbyseamoss.com
2. Navigate to **Settings > Shopify** tab
3. **Enable** the Shopify toggle
4. Set **Attribution Window**: 7 days (or your preference)
5. **Leave webhook secret blank for now** (we'll get it from Shopify)
6. Click **Save**
7. **Copy the Webhook URL** displayed on the page:
   ```
   https://list.bobbyseamoss.com/webhooks/shopify/orders
   ```

## Step 2: Configure Webhook in Shopify

1. Log into your **Shopify Admin**
2. Go to **Settings > Notifications**
3. Scroll down to **Webhooks** section
4. Click **Create webhook**
5. Configure:
   - **Event**: `Order creation`
   - **Format**: `JSON`
   - **URL**: `https://list.bobbyseamoss.com/webhooks/shopify/orders`
   - **Webhook API version**: Latest (2024-10 or newer)
6. Click **Save**
7. **Copy the webhook signing secret** (shown after saving)

## Step 3: Add Webhook Secret to Listmonk

1. Go back to **Listmonk > Settings > Shopify**
2. Paste the **webhook secret** from Shopify
3. Click **Save**
4. The secret will be masked (••••••) after saving

## Step 4: Prepare Test Data

### Option A: Use Existing Subscriber & Campaign

Find a subscriber email that has clicked a link recently:

```sql
-- Connect to database
PGPASSWORD='T@intshr3dd3r' psql -h listmonk420-db.postgres.database.azure.com -p 5432 -U listmonkadmin -d listmonk --set=sslmode=require

-- Find subscribers with recent link clicks
SELECT
    s.email,
    s.id as subscriber_id,
    lc.campaign_id,
    c.name as campaign_name,
    lc.created_at as clicked_at
FROM link_clicks lc
JOIN subscribers s ON s.id = lc.subscriber_id
JOIN campaigns c ON c.id = lc.campaign_id
WHERE lc.created_at >= NOW() - INTERVAL '7 days'
ORDER BY lc.created_at DESC
LIMIT 10;
```

**Note the email** - you'll use this when creating the test order in Shopify.

### Option B: Create Test Subscriber & Campaign

If you don't have recent clicks:

1. **Create a test subscriber** in Listmonk with an email you control
2. **Send a test campaign** to that subscriber
3. **Click a link** in the test email
4. Wait a few minutes for the click to be recorded

## Step 5: Send Test Webhook from Shopify

### Option 1: Use Shopify's "Send test notification" button

1. In Shopify Admin > Settings > Notifications > Webhooks
2. Find your webhook
3. Click the **"•••"** menu
4. Click **"Send test notification"**
5. ⚠️ **Important**: Edit the test payload to use a real subscriber email:
   ```json
   {
     "email": "subscriber@example.com"  ← Change this to your subscriber's email
   }
   ```

### Option 2: Create a Real Test Order

1. In Shopify Admin > Orders
2. Click **Create order**
3. Add a product
4. **Use the subscriber email** from Step 4
5. Set payment status to **Paid**
6. Click **Create order**
7. The webhook will fire automatically

## Step 6: Verify Webhook Received

### Check Application Logs

```bash
az containerapp logs show --name listmonk420 --resource-group rg-listmonk420 --tail 50 | grep -i shopify
```

**Look for:**
- ✅ "Registered Shopify webhook route"
- ✅ "attributed purchase (order XXX, $X.XX USD) to campaign X for subscriber X"
- ❌ "HMAC verification failed" (if webhook secret is wrong)
- ❌ "no subscriber found with email" (if email doesn't exist)
- ❌ "no recent link clicks found" (if attribution window expired)

### Check Webhook Logs Table

```sql
-- View recent webhook logs
SELECT
    id,
    service,
    event_type,
    status_code,
    processed,
    error,
    created_at,
    LEFT(raw_request::text, 100) as request_preview
FROM webhook_logs
WHERE service = 'shopify'
ORDER BY created_at DESC
LIMIT 5;
```

**Expected:**
- `service`: 'shopify'
- `event_type`: 'order'
- `status_code`: 200
- `processed`: true (if attribution successful) or false (if no attribution)
- `error`: Empty if successful, or error message if failed

### Check Purchase Attribution

```sql
-- Check if purchase attribution was created
SELECT
    pa.id,
    pa.campaign_id,
    c.name as campaign_name,
    pa.subscriber_id,
    s.email,
    pa.order_id,
    pa.order_number,
    pa.total_price,
    pa.currency,
    pa.attributed_via,
    pa.confidence,
    pa.created_at
FROM purchase_attributions pa
LEFT JOIN campaigns c ON c.id = pa.campaign_id
LEFT JOIN subscribers s ON s.id = pa.subscriber_id
ORDER BY pa.created_at DESC
LIMIT 5;
```

**Expected if successful:**
- Record exists with your order details
- `campaign_id`: The campaign that was clicked
- `subscriber_id`: The subscriber's ID
- `attributed_via`: 'link_click'
- `confidence`: 'high'

## Step 7: Check Campaign Analytics

1. Go to **Campaigns** in Listmonk
2. Click on the campaign that was attributed
3. Go to **Purchase Analytics** tab
4. You should see:
   - Total Orders: 1 (or more)
   - Total Revenue: $XX.XX
   - Avg Order Value: $XX.XX

## Common Issues & Solutions

### Issue: 401 Unauthorized

**Cause**: HMAC signature mismatch
**Solution**:
- Verify webhook secret is correct in Listmonk
- Make sure you copied the entire secret from Shopify
- Secret should NOT have spaces or newlines
- Try regenerating the webhook in Shopify

### Issue: No attribution created (but webhook logs show processed=false)

**Causes**:
1. **Email mismatch**: Order email doesn't match any subscriber
2. **No recent clicks**: Subscriber hasn't clicked within attribution window
3. **Attribution window expired**: Click was more than X days ago

**Debug queries**:

```sql
-- Check if subscriber exists
SELECT id, email, status FROM subscribers
WHERE LOWER(email) = LOWER('customer@example.com');

-- Check recent link clicks for subscriber
SELECT * FROM link_clicks
WHERE subscriber_id = 123  -- Use subscriber ID from above
AND created_at >= NOW() - INTERVAL '7 days'
ORDER BY created_at DESC;
```

**Solutions**:
- Ensure subscriber exists in listmonk
- Ensure subscriber clicked a link recently
- Increase attribution window if needed
- Create test campaign and click link before testing

### Issue: Webhook not received at all

**Checks**:
1. Verify Shopify integration is enabled in Listmonk settings
2. Check webhook URL is exactly: `https://list.bobbyseamoss.com/webhooks/shopify/orders`
3. Check Shopify webhook status (should be "Active")
4. Test webhook connectivity: `curl -I https://list.bobbyseamoss.com/webhooks/shopify/orders`

### Issue: Database shows order but campaign stats don't update

**Solution**: Refresh the campaign page (stats are calculated on-demand)

## Test Checklist

- [ ] Shopify integration enabled in Listmonk
- [ ] Webhook secret configured
- [ ] Webhook created in Shopify
- [ ] Test subscriber exists
- [ ] Test subscriber has recent link clicks (within attribution window)
- [ ] Test order created with matching email
- [ ] Webhook logs show success (status 200, processed=true)
- [ ] Purchase attribution record created
- [ ] Campaign analytics show revenue

## Success Criteria

✅ **Webhook received**: webhook_logs table has record with status 200
✅ **Attribution created**: purchase_attributions table has record
✅ **Campaign stats updated**: Purchase Analytics tab shows data
✅ **No errors**: Application logs show successful attribution
✅ **HMAC verified**: No "HMAC verification failed" errors

## Advanced Testing

### Test Attribution Window Boundaries

1. Create test order with email that clicked 6 days ago → ✅ Should attribute
2. Create test order with email that clicked 8 days ago (7-day window) → ❌ Should NOT attribute

### Test Multiple Campaigns

1. Subscriber clicks Link A from Campaign 1
2. Wait 1 hour
3. Subscriber clicks Link B from Campaign 2
4. Subscriber makes purchase
5. **Expected**: Attributed to Campaign 2 (most recent click)

### Test No Attribution Cases

1. **New customer** (no subscriber record) → No attribution, logged error
2. **Subscriber without clicks** → No attribution, logged error
3. **Old click** (outside attribution window) → No attribution, logged error

All cases should return HTTP 200 (so Shopify considers webhook successful).

---

**End of Testing Guide**
