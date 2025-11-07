# Webhook Processing Analysis

## Summary
After thorough investigation of the webhook processing code, I've identified the flow for bounces, clicks, and views, along with several potential issues.

---

## 1. BOUNCE WEBHOOK PROCESSING

### Flow
1. **Azure webhook arrives** → `cmd/bounce.go:241-571` (BounceWebhook handler)
2. **Delivery event parsed** → `internal/bounce/webhooks/azure.go:118-179` (ProcessDeliveryEvent)
3. **Bounce recorded** → `internal/bounce/bounce.go:196-199` (Record) → adds to queue
4. **Queue processed** → `internal/bounce/bounce.go:136-144` (Run) → calls RecordBounceCB
5. **Core records bounce** → `internal/core/bounces.go:60-93` (RecordBounce)
6. **SQL executed** → `queries.sql:1134-1163` (record-bounce)

### Critical SQL Logic (queries.sql:1134-1163)

```sql
-- Blocklisting happens in block1:
block1 AS (
    UPDATE subscribers SET status='blocklisted'
    WHERE $9 = 'blocklist'                    -- Action must be 'blocklist'
    AND (SELECT num FROM num) >= $8           -- Bounce count >= threshold
    AND id = (SELECT id FROM sub)
    AND (SELECT status FROM sub) != 'blocklisted'
)
```

**Parameters:**
- $8 = threshold count (from bounce.actions config)
- $9 = action ('blocklist' or 'delete' or 'none')

---

## 2. POTENTIAL ISSUES WITH BOUNCES

### Issue #1: Bounce Configuration Missing or Incorrect
**Location:** Database `settings` table, key `bounce.actions`

**Default configuration:**
```json
{
  "soft": {"count": 2, "action": "none"},
  "hard": {"count": 2, "action": "blocklist"},
  "complaint": {"count": 2, "action": "blocklist"}
}
```

**Problem:** If this setting is:
- Missing → No action is taken
- Set to "none" → No blocklisting happens
- Threshold too high → Subscribers need multiple bounces before blocklisting

**How to check:**
```sql
SELECT value FROM settings WHERE key = 'bounce.actions';
```

**How to fix (via SQL):**
```sql
UPDATE settings
SET value = '{"soft": {"count": 2, "action": "none"}, "hard": {"count": 1, "action": "blocklist"}, "complaint": {"count": 1, "action": "blocklist"}}'
WHERE key = 'bounce.actions';
```

---

### Issue #2: Subscriber Lookup Failures

**Location:** `cmd/bounce.go:296-408`

The code tries to find the subscriber in this order:
1. **Message-ID lookup** (lines 302-307) - from `azure_message_tracking` table
2. **Recipient email fallback** (lines 310-325) - cross-joins campaigns and subscribers

**Problem:** If azure_message_tracking doesn't have the record, the fallback query might:
- Find the wrong campaign (most recent one)
- Find the wrong subscriber (if multiple campaigns are running)
- Not find the subscriber at all

**Evidence in code (lines 395-406):**
```go
// CRITICAL: Set subscriber ID if we found it via fallback
// The bounces table requires subscriber_id, not just email
if subscriberID > 0 {
    b.SubscriberID = subscriberID
}
```

If `subscriberID` is 0 (not found), the bounce will fail to record properly.

---

### Issue #3: Bounce Type Mapping

**Location:** `internal/bounce/webhooks/azure.go:137-158`

Azure status codes are mapped to bounce types:
```go
StatusBounced, StatusFailed, StatusSuppressed → BounceTypeHard (blocklist)
StatusQuarantined → BounceTypeSoft (no action by default)
StatusFilteredSpam → BounceTypeComplaint (blocklist)
```

**Problem:** If Azure sends different status codes than expected, bounces might be categorized incorrectly.

---

### Issue #4: Bounce Recording Errors Silently Ignored

**Location:** `internal/core/bounces.go:76-92`

If a bounce fails to record because the subscriber isn't found:
```go
if pqErr, ok := err.(*pq.Error); ok && pqErr.Column == "subscriber_id" {
    c.log.Printf("bounced subscriber not found - email=%s, type=%s, source=%s, meta=%s",
        b.Email, b.Type, b.Source, metaStr)
    return nil  // ← Error is ignored!
}
```

This means bounces can silently fail without triggering blocklisting.

---

## 3. CLICK EVENT PROCESSING

### Flow
1. **Azure engagement webhook arrives** → `cmd/bounce.go:410-546`
2. **Event parsed** → `internal/bounce/webhooks/azure.go:183-195` (ProcessEngagementEvent)
3. **Campaign/Subscriber lookup** → lines 423-486
4. **Link created/found** → lines 524-535
5. **Click recorded** → lines 537-545

**Tables updated:**
- `links` table (URL stored)
- `link_clicks` table (click recorded)
- `azure_engagement_events` table (raw Azure event)

**✓ This appears correct!** Clicks are being inserted into `link_clicks` table.

---

## 4. VIEW EVENT PROCESSING

### Flow
Same as clicks, but simpler:
1. **View event detected** → line 506
2. **Inserted into campaign_views** → lines 508-514
3. **Also stored in azure_engagement_events** → lines 495-503

**Tables updated:**
- `campaign_views` table (view recorded)
- `azure_engagement_events` table (raw Azure event)

**✓ This appears correct!** Views are being inserted into `campaign_views` table.

---

## 5. ANALYTICS INTEGRATION

### Campaign Stats Query
**Location:** `queries.sql:624-645`

```sql
views AS (
    SELECT campaign_id, COUNT(campaign_id) as num FROM campaign_views
    WHERE campaign_id = ANY($1)
    GROUP BY campaign_id
),
clicks AS (
    SELECT campaign_id, COUNT(campaign_id) as num FROM link_clicks
    WHERE campaign_id = ANY($1)
    GROUP BY campaign_id
)
```

**✓ Stats are read from the correct tables!**

### Loading Stats
**Location:** `internal/core/campaigns.go:77`

```go
if err := out.LoadStats(c.q.GetCampaignStats); err != nil {
    c.log.Printf("error fetching campaign stats: %v", err)
    return nil, 0, echo.NewHTTPError(...)
}
```

**✓ Stats are loaded when fetching campaigns!**

---

## 6. DIAGNOSIS STEPS

### Step 1: Check Bounce Configuration
```sql
SELECT key, value FROM settings WHERE key = 'bounce.actions';
```

Expected output:
```json
{
  "hard": {"count": 2, "action": "blocklist"},
  "soft": {"count": 2, "action": "none"},
  "complaint": {"count": 2, "action": "blocklist"}
}
```

### Step 2: Check Recent Bounces
```sql
SELECT
    b.email,
    b.type,
    b.source,
    b.created_at,
    s.status as subscriber_status,
    COUNT(*) OVER (PARTITION BY b.subscriber_id, b.type) as bounce_count
FROM bounces b
LEFT JOIN subscribers s ON b.subscriber_id = s.id
WHERE b.created_at > NOW() - INTERVAL '24 hours'
ORDER BY b.created_at DESC;
```

Look for subscribers with:
- `bounce_count >= 2` (or threshold from config)
- `type = 'hard' or 'complaint'`
- `subscriber_status != 'blocklisted'` ← **These are the problematic ones!**

### Step 3: Check Message Tracking
```sql
SELECT COUNT(*) FROM azure_message_tracking
WHERE sent_at > NOW() - INTERVAL '24 hours';
```

If this is 0 or very low, message tracking might not be working, which breaks subscriber lookup.

### Step 4: Check Campaign Stats
```sql
SELECT
    c.id,
    c.name,
    c.views as reported_views,
    (SELECT COUNT(*) FROM campaign_views cv WHERE cv.campaign_id = c.id) as actual_views,
    c.clicks as reported_clicks,
    (SELECT COUNT(*) FROM link_clicks lc WHERE lc.campaign_id = c.id) as actual_clicks
FROM campaigns c
WHERE c.updated_at > NOW() - INTERVAL '7 days'
ORDER BY c.updated_at DESC;
```

If `reported_views != actual_views` or `reported_clicks != actual_clicks`, there's a stats caching issue.

### Step 5: Check Webhook Logs
```sql
SELECT * FROM webhook_logs
WHERE webhook_type = 'azure'
AND created_at > NOW() - INTERVAL '24 hours'
ORDER BY created_at DESC
LIMIT 50;
```

Look for:
- `processed = false` (failed webhooks)
- `error_message` (specific errors)

---

## 7. RECOMMENDED FIXES

### Fix #1: Ensure Bounce Configuration is Correct
Update bounce.actions to blocklist after first hard bounce:

```sql
UPDATE settings
SET value = '{"soft": {"count": 2, "action": "none"}, "hard": {"count": 1, "action": "blocklist"}, "complaint": {"count": 1, "action": "blocklist"}}'
WHERE key = 'bounce.actions';
```

Then restart listmonk to reload the configuration.

### Fix #2: Add Better Error Logging
The code at `internal/core/bounces.go:78-86` silently ignores subscriber lookup failures. This should log more prominently or send alerts.

### Fix #3: Verify Message Tracking is Working
Check if messages are being tracked properly in `azure_message_tracking` table. If not, the webhook won't be able to link bounces back to subscribers.

### Fix #4: Manual Blocklisting
If you have specific bounced subscribers that weren't blocklisted, you can manually blocklist them:

```sql
-- Find subscribers with hard bounces
SELECT DISTINCT s.id, s.email, s.status, COUNT(b.id) as bounce_count
FROM subscribers s
JOIN bounces b ON s.id = b.subscriber_id
WHERE b.type IN ('hard', 'complaint')
AND s.status != 'blocklisted'
GROUP BY s.id
HAVING COUNT(b.id) >= 1;

-- Blocklist them
UPDATE subscribers
SET status = 'blocklisted'
WHERE id IN (
    SELECT DISTINCT s.id
    FROM subscribers s
    JOIN bounces b ON s.id = b.subscriber_id
    WHERE b.type IN ('hard', 'complaint')
    AND s.status != 'blocklisted'
    GROUP BY s.id
    HAVING COUNT(b.id) >= 2
);
```

Or use the API:
```bash
curl -X PUT http://your-listmonk-url/api/bounces/blocklist
```

---

## 8. CODE LOCATIONS REFERENCE

### Bounce Processing
- Webhook handler: `cmd/bounce.go:117-572`
- Azure webhook parser: `internal/bounce/webhooks/azure.go`
- Bounce recording: `internal/core/bounces.go:60-93`
- SQL query: `queries.sql:1134-1163`

### Click/View Processing
- Click handling: `cmd/bounce.go:517-546`
- View handling: `cmd/bounce.go:505-515`

### Analytics
- Stats query: `queries.sql:624-645`
- Stats loading: `internal/core/campaigns.go:77`
- Campaigns API: `cmd/campaigns.go:54-109`

---

## 9. NEXT STEPS

1. **Run diagnostic queries** in Steps 1-5 above
2. **Check bounce.actions configuration** and fix if needed
3. **Verify message tracking** is populating azure_message_tracking
4. **Check webhook logs** for specific errors
5. **Manually blocklist** any subscribers that should be blocklisted
6. **Monitor** future bounces to ensure they blocklist properly

---

## 10. CONCLUSION

The webhook processing code is mostly correct, with proper integration for:
- ✓ Views → `campaign_views` table → campaign stats
- ✓ Clicks → `link_clicks` table → campaign stats
- ⚠️ Bounces → depends on configuration and subscriber lookup

**Most likely issue:** The `bounce.actions` configuration is either:
1. Not set to "blocklist" for hard bounces
2. Has a threshold that's too high
3. Is missing entirely

**Secondary issue:** Azure message tracking might not be recording message IDs properly, causing subscriber lookups to fail.
