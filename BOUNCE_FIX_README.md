# Listmonk Bounce & Webhook Diagnostics

## Overview

This package contains diagnostic and repair tools for investigating webhook processing issues in listmonk, specifically:
- Bounces not blocklisting subscribers
- Click events not showing in analytics
- View events not showing in analytics

## Files Created

### 📊 Diagnostic Tools

1. **`diagnose_bounces.sql`**
   - Comprehensive SQL diagnostic script
   - Checks bounce configuration, recent bounces, Azure events, campaign stats
   - Safe to run - read-only queries

2. **`check_webhook_config.py`**
   - Python diagnostic script (alternative to SQL)
   - Requires `psycopg2` library
   - Same checks as SQL script but with better formatting

3. **`run_diagnostics.sh`**
   - Bash wrapper to run diagnostic scripts easily
   - Handles database connection
   - Supports interactive fixes

### 🔧 Repair Tools

4. **`fix_bounce_config.sql`**
   - Updates `bounce.actions` configuration to properly blocklist
   - Changes hard/complaint bounces to blocklist after 1 occurrence
   - **Requires listmonk restart after running**

5. **`blocklist_bounced_subscribers.sql`**
   - Finds subscribers with hard/complaint bounces
   - Provides preview before blocklisting
   - UPDATE statement is commented out for safety

### 📖 Documentation

6. **`WEBHOOK_ANALYSIS.md`**
   - Complete technical analysis of webhook processing
   - Code flow diagrams
   - Issue explanations
   - Step-by-step fixes

7. **`BOUNCE_FIX_README.md`** (this file)
   - Quick start guide
   - Usage instructions

## Quick Start

### Step 1: Run Diagnostics

```bash
./run_diagnostics.sh
```

This will show:
- Current bounce configuration
- Recent bounces and their subscriber status
- Azure webhook events
- Campaign statistics
- Webhook logs

**Look for:**
- ❌ `bounce.actions` with `"action": "none"`
- ⚠️ Subscribers with multiple bounces but not blocklisted
- 🔴 Mismatched campaign statistics

### Step 2: Fix Bounce Configuration (if needed)

If diagnostics show bounce.actions is wrong:

```bash
./run_diagnostics.sh fix-config
```

This will:
1. Show current configuration
2. Ask for confirmation
3. Update configuration to blocklist after 1 hard/complaint bounce
4. Remind you to restart listmonk

**⚠️ IMPORTANT:** After running, restart listmonk:
```bash
# Azure Container Apps
az containerapp restart --name listmonk420 --resource-group <your-rg>

# Or via systemd
sudo systemctl restart listmonk

# Or via UI
Settings > Reload (button in top-right)
```

### Step 3: Blocklist Existing Bounced Subscribers (if needed)

If diagnostics show subscribers that should be blocklisted:

```bash
./run_diagnostics.sh blocklist
```

This will:
1. Show subscribers that will be affected
2. Display counts by bounce type
3. Instructions for confirming the blocklist

To actually blocklist them, edit `blocklist_bounced_subscribers.sql` and uncomment the UPDATE section, then run again.

**Alternative:** Use the API endpoint:
```bash
curl -X PUT https://your-listmonk-url/api/bounces/blocklist \
  -H "Authorization: Bearer YOUR_API_TOKEN"
```

## Manual SQL Queries

If you prefer to run queries directly:

### Check Bounce Configuration

```sql
SELECT value FROM settings WHERE key = 'bounce.actions';
```

Should return something like:
```json
{
  "hard": {"count": 1, "action": "blocklist"},
  "soft": {"count": 2, "action": "none"},
  "complaint": {"count": 1, "action": "blocklist"}
}
```

### Find Subscribers That Should Be Blocklisted

```sql
SELECT
    s.id,
    s.email,
    s.status,
    COUNT(CASE WHEN b.type = 'hard' THEN 1 END) as hard_bounces,
    COUNT(CASE WHEN b.type = 'complaint' THEN 1 END) as complaint_bounces
FROM subscribers s
JOIN bounces b ON s.id = b.subscriber_id
WHERE s.status != 'blocklisted'
GROUP BY s.id, s.email, s.status
HAVING
    COUNT(CASE WHEN b.type = 'hard' THEN 1 END) >= 1
    OR COUNT(CASE WHEN b.type = 'complaint' THEN 1 END) >= 1;
```

### Manually Blocklist a Subscriber

```sql
UPDATE subscribers
SET status = 'blocklisted', updated_at = NOW()
WHERE email = 'bounced@example.com';
```

### Check Campaign Stats

```sql
SELECT
    c.id,
    c.name,
    c.views as reported_views,
    (SELECT COUNT(*) FROM campaign_views WHERE campaign_id = c.id) as actual_views,
    c.clicks as reported_clicks,
    (SELECT COUNT(*) FROM link_clicks WHERE campaign_id = c.id) as actual_clicks
FROM campaigns c
ORDER BY c.updated_at DESC
LIMIT 10;
```

## Understanding Bounce Configuration

### Configuration Format

```json
{
  "hard": {
    "count": 1,        // Blocklist after 1 hard bounce
    "action": "blocklist"  // Options: "blocklist", "delete", "none"
  },
  "soft": {
    "count": 2,        // Soft bounces need 2 before action
    "action": "none"   // Don't blocklist soft bounces (temporary issues)
  },
  "complaint": {
    "count": 1,        // Blocklist after 1 spam complaint
    "action": "blocklist"
  }
}
```

### Bounce Types

- **`hard`**: Permanent delivery failures (mailbox doesn't exist, domain invalid, etc.)
- **`soft`**: Temporary failures (mailbox full, server temporarily unavailable)
- **`complaint`**: Subscriber marked email as spam

### Actions

- **`blocklist`**: Change subscriber status to `blocklisted` (stops all emails)
- **`delete`**: Permanently delete the subscriber (⚠️ destructive!)
- **`none`**: Record the bounce but take no action

### Azure Status Mapping

Azure delivery statuses are mapped to bounce types:

| Azure Status | Bounce Type | Default Action |
|--------------|-------------|----------------|
| Bounced | hard | blocklist |
| Failed | hard | blocklist |
| Suppressed | hard | blocklist |
| Quarantined | soft | none |
| FilteredSpam | complaint | blocklist |
| Delivered | (none) | n/a |

## Common Issues & Solutions

### Issue: Bounces Not Blocklisting

**Symptoms:**
- Bounces appear in bounce logs
- Subscribers show `enabled` or `disabled` status
- Multiple bounces for same subscriber

**Causes:**
1. `bounce.actions` set to `"action": "none"`
2. Bounce count threshold too high
3. Subscriber lookup failures

**Solution:**
```bash
./run_diagnostics.sh fix-config
# Then restart listmonk
./run_diagnostics.sh blocklist  # For existing bounces
```

### Issue: Campaign Stats Not Showing Clicks/Views

**Symptoms:**
- Campaign shows 0 views/clicks
- Azure engagement events exist in `azure_engagement_events` table
- Campaign stats don't match reality

**Diagnosis:**
```sql
SELECT
    c.id,
    c.name,
    c.views,
    (SELECT COUNT(*) FROM campaign_views WHERE campaign_id = c.id) as actual_views,
    (SELECT COUNT(*) FROM azure_engagement_events
     WHERE campaign_id = c.id AND engagement_type = 'view') as azure_views
FROM campaigns c
WHERE c.id = YOUR_CAMPAIGN_ID;
```

**If `actual_views` and `azure_views` match but `c.views` is wrong:**
- Campaign stats need to be refreshed
- This is a display/caching issue, not a data issue

**If `azure_views` > 0 but `actual_views` = 0:**
- Webhooks are being received but not processed correctly
- Check webhook logs for errors

### Issue: Message Tracking Not Working

**Symptoms:**
- `azure_message_tracking` table is empty or very low count
- Bounce webhooks fail to find subscriber
- Error logs: "bounced subscriber not found"

**Diagnosis:**
```sql
SELECT COUNT(*) FROM azure_message_tracking
WHERE sent_at > NOW() - INTERVAL '24 hours';
```

If this returns 0:
- Messages aren't being tracked when sent
- Check that Azure webhook is configured in listmonk
- Check that X-Listmonk-* headers are being added to emails

**Solution:**
- Verify Azure Event Grid subscription is active
- Check webhook endpoint URL is correct
- Review webhook logs in Azure Portal

## Webhook Processing Flow

### Bounces

1. Email sent with `X-Listmonk-Campaign` and `X-Listmonk-Subscriber` headers
2. Message ID tracked in `azure_message_tracking` table
3. Azure sends delivery webhook with bounce status
4. Webhook handler looks up subscriber by Message ID
5. Bounce recorded in `bounces` table
6. SQL query checks bounce count vs threshold
7. If threshold exceeded, subscriber blocklisted

**Code locations:**
- Webhook handler: `cmd/bounce.go:241-408`
- Bounce recording: `internal/core/bounces.go:60-93`
- SQL query: `queries.sql:1134-1163`

### Clicks

1. Email sent with tracked links
2. Subscriber clicks link
3. Azure sends engagement webhook with `engagement_type = 'click'`
4. Webhook handler looks up campaign/subscriber
5. Link URL stored in `links` table (if new)
6. Click recorded in `link_clicks` table
7. Click also stored in `azure_engagement_events` table

**Code locations:**
- Webhook handler: `cmd/bounce.go:517-546`
- SQL insert: Direct DB insert (lines 538-541)

### Views

1. Email sent with tracking pixel
2. Subscriber opens email
3. Azure sends engagement webhook with `engagement_type = 'view'`
4. Webhook handler looks up campaign/subscriber
5. View recorded in `campaign_views` table
6. View also stored in `azure_engagement_events` table

**Code locations:**
- Webhook handler: `cmd/bounce.go:506-515`
- SQL insert: Direct DB insert (lines 508-510)

## Testing Webhooks

### Test Bounce Webhook

```bash
curl -X POST https://your-listmonk-url/webhooks/bounce/azure \
  -H "Content-Type: application/json" \
  -d '[{
    "eventType": "Microsoft.Communication.EmailDeliveryReportReceived",
    "data": {
      "recipient": "test@example.com",
      "messageId": "test-message-id",
      "status": "Bounced",
      "deliveryAttemptTimeStamp": "2025-11-04T12:00:00Z"
    }
  }]'
```

Check webhook logs:
```sql
SELECT * FROM webhook_logs ORDER BY created_at DESC LIMIT 1;
```

### Test View Webhook

```bash
curl -X POST https://your-listmonk-url/webhooks/bounce/azure \
  -H "Content-Type: application/json" \
  -d '[{
    "eventType": "Microsoft.Communication.EmailEngagementTrackingReportReceived",
    "data": {
      "recipient": "test@example.com",
      "messageId": "test-message-id",
      "engagementType": "view",
      "userActionTimeStamp": "2025-11-04T12:00:00Z"
    }
  }]'
```

## Support

If you continue to have issues after running diagnostics and fixes:

1. Check the webhook logs table for specific errors:
   ```sql
   SELECT * FROM webhook_logs WHERE processed = false;
   ```

2. Check listmonk application logs for errors

3. Verify Azure Event Grid subscription is active and delivering events

4. Review `WEBHOOK_ANALYSIS.md` for detailed technical information

## License

These diagnostic tools are provided as-is for debugging listmonk webhook issues.
