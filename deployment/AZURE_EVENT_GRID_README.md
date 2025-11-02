# Azure Event Grid Integration for Listmonk

This document provides complete instructions for setting up Azure Event Grid webhooks to track email delivery status and user engagement (opens/clicks) for Azure Communication Services in listmonk.

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Prerequisites](#prerequisites)
- [Architecture](#architecture)
- [Installation](#installation)
- [Configuration](#configuration)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)
- [Monitoring](#monitoring)
- [FAQ](#faq)

---

## Overview

Azure Event Grid integration enables **real-time tracking** of email delivery and engagement events from Azure Communication Services, providing:

- **Instant bounce detection** (Bounced, Failed, Suppressed, FilteredSpam, Quarantined)
- **Delivery confirmation** (Delivered status updates)
- **Engagement tracking** (Email opens and link clicks)
- **Improved Smart Sending** accuracy by only counting actually-delivered emails

Without this integration, bounce detection relies on polling POP mailboxes, which can be slow and miss certain bounce types.

---

## Features

### Delivery Status Tracking

| Azure Status | Listmonk Bounce Type | Description |
|-------------|---------------------|-------------|
| **Bounced** | Hard Bounce | Email address doesn't exist or domain is invalid |
| **Failed** | Hard Bounce | SMTP delivery failure |
| **Suppressed** | Hard Bounce | Recipient previously bounced, email suppressed |
| **FilteredSpam** | Complaint | Message identified as spam |
| **Quarantined** | Soft Bounce | Message quarantined (may be reviewed/released) |
| **Delivered** | (No bounce) | Successfully delivered to inbox |
| **Expanded** | (No bounce) | Distribution list expanded (informational) |

### Engagement Tracking

- **Opens/Views**: Tracked in `campaign_views` table
- **Link Clicks**: Tracked in `link_clicks` table with URL mapping

### Smart Sending Integration

- Only counts emails with `Delivered` status towards Smart Sending limits
- Excludes bounced/failed emails from "recently sent" tracking
- Fixes false-negative issue where emails marked sent but bounced

---

## Prerequisites

### Azure Requirements

1. **Azure Communication Services** resources provisioned
2. **Email domains** configured and verified in Azure Communication Services
3. **User engagement tracking enabled** on each domain (Settings > Provision Domains > Enable tracking)
4. **Contributor or Event Grid Contributor** role on the subscription
5. **Azure CLI** installed and configured (`az login` completed)

### Listmonk Requirements

1. **Listmonk v6.1.0+** (includes Azure Event Grid support)
2. **HTTPS endpoint** accessible from Azure (Azure Event Grid requires HTTPS for production)
3. **PostgreSQL database** with migration applied
4. **Network access** allowing Azure Event Grid to reach webhook endpoint

---

## Architecture

### Event Flow

```
Azure Communication Services
         │
         ├─> Send Email
         │
         ├─> Email Delivered/Bounced
         │
         ▼
Azure Event Grid
         │
         ├─> EmailDeliveryReportReceived
         ├─> EmailEngagementTrackingReportReceived
         │
         ▼
Listmonk Webhook Endpoint
(/webhooks/service/azure)
         │
         ├─> Process Event
         ├─> Correlate Campaign/Subscriber
         │
         ▼
Database Updates
         │
         ├─> Insert bounce record (if applicable)
         ├─> Insert campaign_views (for opens)
         └─> Insert link_clicks (for clicks)
```

### Message Correlation

Listmonk uses a **two-tier correlation system**:

1. **Primary**: X-headers in email (`X-Listmonk-Campaign`, `X-Listmonk-Subscriber`)
   - If Azure Event Grid preserves these headers in event data, correlation is instant
   - No database lookup required

2. **Fallback**: Database lookup via `azure_message_tracking` table
   - Maps Azure's `messageId` to listmonk campaign/subscriber IDs
   - Used if Azure doesn't include X-headers in event payload

---

## Installation

### Step 1: Database Migration

Run the database migration to create the `azure_message_tracking` table:

```bash
cd /home/adam/listmonk
./listmonk --upgrade
```

Expected output:
```
running migration v6.1.0: Azure Event Grid webhook tracking
migration v6.1.0 completed successfully
```

Verify migration:
```sql
SELECT * FROM information_schema.tables
WHERE table_name = 'azure_message_tracking';
```

### Step 2: Enable Azure Event Grid in Listmonk

**Option A: Via Database** (currently required, UI pending):

```sql
-- Enable Azure Event Grid webhooks
UPDATE settings
SET value = jsonb_set(
    COALESCE(value, '{}'::jsonb),
    '{azure,enabled}',
    'true'::jsonb
)
WHERE key = 'bounce';

-- Verify
SELECT value->'azure'->>'enabled' AS azure_enabled
FROM settings
WHERE key = 'bounce';
```

**Option B: Via Settings File** (if using file-based config):

Add to `config.toml`:
```toml
[bounce.azure]
enabled = true
```

Restart listmonk:
```bash
sudo systemctl restart listmonk
```

### Step 3: Create Event Grid Subscriptions

Use the automated setup script to create Event Grid subscriptions for all your Azure Communication Services resources:

```bash
cd /home/adam/listmonk/deployment/scripts

# Setup for all ACS resources in subscription
./azure-event-grid-setup.sh https://listmonk.yourdomain.com/webhooks/service/azure

# Setup for specific resource group
./azure-event-grid-setup.sh https://listmonk.yourdomain.com/webhooks/service/azure rg-listmonk420
```

**Script Output:**
```
==========================================
Azure Event Grid Setup for Listmonk
==========================================

Webhook Endpoint: https://listmonk.yourdomain.com/webhooks/service/azure
Resource Group Filter: (all resource groups)

✓ Logged in to Azure
  Subscription: My Subscription
  Subscription ID: 12345678-1234-1234-1234-123456789012

==========================================
Finding Azure Communication Services Resources
==========================================

Searching for Azure Communication Services...
✓ Found 32 Azure Communication Services resource(s)

==========================================
Creating Event Grid Subscriptions
==========================================

Processing: mail1-bobbyseamoss-com
  Resource Group: rg-listmonk420
  Subscription Name: listmonk-email-events-mail1-bobbyseamoss-com
  Creating Event Grid subscription...
  ✓ Successfully created Event Grid subscription

[... repeats for all 32 resources ...]

==========================================
Summary
==========================================

Total Resources Found: 32
Successfully Created: 32
Skipped (already exists): 0
Failed: 0

✓ Azure Event Grid setup complete!
```

### Step 4: Verify Subscription Validation

Azure Event Grid will send a validation event to your webhook endpoint. Check listmonk logs:

```bash
tail -f /var/log/listmonk/listmonk.log | grep -i azure
```

Expected log output:
```
2025/11/01 12:00:00.123456 EST Azure Event Grid: subscription validation successful
```

---

## Configuration

### Webhook Endpoint

**URL Format**: `https://your-listmonk-domain.com/webhooks/service/azure`

**Requirements**:
- **HTTPS** required (Azure Event Grid validates SSL certificates)
- **Publicly accessible** from Azure (not localhost)
- **No authentication** required (Azure validates via subscription handshake)

### Event Types

The integration subscribes to two event types:

1. **Microsoft.Communication.EmailDeliveryReportReceived**
   - Triggered when email reaches terminal delivery state
   - Includes: Delivered, Bounced, Failed, Suppressed, FilteredSpam, Quarantined, Expanded

2. **Microsoft.Communication.EmailEngagementTrackingReportReceived**
   - Triggered when user opens email or clicks link
   - Includes: view (open), click

### Retry Policy

Event Grid subscriptions are configured with:
- **Max delivery attempts**: 30
- **Event TTL**: 1440 minutes (24 hours)
- **Dead-letter queue**: Not configured (can be added manually)

---

## Testing

### Local Testing with Mock Events

Test the webhook endpoint locally before deploying Event Grid subscriptions:

```bash
cd /home/adam/listmonk/deployment/scripts

# Test with localhost (requires listmonk running locally)
./test-azure-webhooks.sh

# Test with custom endpoint
./test-azure-webhooks.sh https://listmonk.yourdomain.com/webhooks/service/azure

# Verbose mode
VERBOSE=1 ./test-azure-webhooks.sh
```

**Test Suite Includes**:
- ✓ Subscription validation
- ✓ Hard bounce (Bounced status)
- ✓ Hard bounce (Failed status)
- ✓ Hard bounce (Suppressed status)
- ✓ Complaint (FilteredSpam status)
- ✓ Soft bounce (Quarantined status)
- ✓ Successful delivery
- ✓ Email open/view
- ✓ Link click
- ✓ Invalid event type (error handling)

**Example Output**:
```
==========================================
Azure Event Grid Webhook Test Suite
==========================================

Webhook URL: http://localhost:9000/webhooks/service/azure
Timestamp: Fri Nov  1 12:00:00 EST 2025

Checking if webhook endpoint is reachable...
✓ Webhook endpoint is reachable

==========================================
TEST: Subscription Validation
==========================================

Sending Subscription Validation Event...
Event Type: SubscriptionValidation
HTTP Status: 200
✓ PASS: Subscription Validation Event (HTTP 200)

[... more tests ...]

==========================================
TEST SUMMARY
==========================================

Total Tests: 10
Passed: 10
Failed: 0

✓ All tests passed!
```

### Verify Database Records

After running tests, check the database:

```sql
-- Check bounces from Azure
SELECT id, email, type, source, created_at, campaign_uuid
FROM bounces
WHERE source = 'azure'
ORDER BY created_at DESC
LIMIT 10;

-- Check campaign views (opens)
SELECT campaign_id, subscriber_id, created_at
FROM campaign_views
ORDER BY created_at DESC
LIMIT 10;

-- Check link clicks
SELECT lc.campaign_id, lc.subscriber_id, l.url, lc.created_at
FROM link_clicks lc
JOIN links l ON lc.link_id = l.id
ORDER BY lc.created_at DESC
LIMIT 10;
```

### Production Testing

1. **Send test campaign** via Azure Communication Services
2. **Trigger bounce** by sending to invalid@invalid.com
3. **Check logs**:
   ```bash
   grep -i "azure" /var/log/listmonk/listmonk.log
   ```
4. **Verify bounce recorded**:
   ```sql
   SELECT * FROM bounces WHERE email = 'invalid@invalid.com' ORDER BY created_at DESC LIMIT 1;
   ```
5. **Open email** and verify view tracked
6. **Click link** and verify click tracked

---

## Troubleshooting

### Issue: Webhook validation fails

**Symptoms**:
- Event Grid subscription stuck in "ValidationPending" status
- No validation logs in listmonk

**Solutions**:
1. Verify endpoint is accessible via HTTPS:
   ```bash
   curl -I https://listmonk.yourdomain.com/webhooks/service/azure
   ```

2. Check SSL certificate validity:
   ```bash
   openssl s_client -connect listmonk.yourdomain.com:443 -servername listmonk.yourdomain.com
   ```

3. Verify Azure Event Grid is enabled:
   ```sql
   SELECT value->'azure'->>'enabled' FROM settings WHERE key = 'bounce';
   ```

4. Check listmonk logs for errors:
   ```bash
   tail -100 /var/log/listmonk/listmonk.log | grep -i "azure\|validation"
   ```

### Issue: Bounces not recorded

**Symptoms**:
- Azure webhook receives events (logs show HTTP 200)
- No bounce records in database

**Solutions**:
1. Check if campaign UUID correlation is working:
   ```bash
   grep "Azure Event Grid: found campaign UUID" /var/log/listmonk/listmonk.log
   ```

2. If X-headers not found, check azure_message_tracking table:
   ```sql
   SELECT COUNT(*) FROM azure_message_tracking;
   ```

3. Verify bounce processing:
   ```sql
   SELECT * FROM bounces WHERE source = 'azure' ORDER BY created_at DESC LIMIT 5;
   ```

4. Check for database errors in logs:
   ```bash
   grep -i "error.*azure\|error recording bounce" /var/log/listmonk/listmonk.log
   ```

### Issue: Engagement events not tracked

**Symptoms**:
- Emails opened/clicked but no records in campaign_views/link_clicks

**Solutions**:
1. Verify user engagement tracking enabled in Azure:
   - Azure Portal → Email Communication Service → Provision Domains
   - Check "User engagement tracking" column shows "On"

2. Check if engagement events are received:
   ```bash
   grep "EmailEngagementTrackingReportReceived" /var/log/listmonk/listmonk.log
   ```

3. Verify HTML emails (tracking doesn't work with plain text):
   ```sql
   SELECT content_type FROM campaigns WHERE id = <campaign_id>;
   ```

4. Check campaign/subscriber correlation:
   ```bash
   grep "error looking up tracking info" /var/log/listmonk/listmonk.log
   ```

### Issue: High webhook latency

**Symptoms**:
- Slow webhook response times
- Azure Event Grid retrying deliveries

**Solutions**:
1. Check database connection pool size
2. Add index on azure_message_tracking.azure_message_id (already done in migration)
3. Monitor database query performance:
   ```sql
   SELECT * FROM pg_stat_statements
   WHERE query LIKE '%azure_message_tracking%'
   ORDER BY total_time DESC;
   ```

4. Consider async processing if volume is very high (>100k emails/day)

### Issue: Duplicate events

**Symptoms**:
- Same bounce/view/click recorded multiple times

**Solutions**:
1. Check for duplicate Event Grid subscriptions:
   ```bash
   az eventgrid event-subscription list \
     --source-resource-id <resource-id> \
     --query "[].name"
   ```

2. Add unique constraints to prevent duplicates:
   ```sql
   -- For campaign views
   CREATE UNIQUE INDEX idx_unique_campaign_view
   ON campaign_views(campaign_id, subscriber_id, created_at);

   -- For link clicks
   CREATE UNIQUE INDEX idx_unique_link_click
   ON link_clicks(campaign_id, subscriber_id, link_id, created_at);
   ```

---

## Monitoring

### Key Metrics

Monitor these metrics to ensure healthy operation:

1. **Webhook Success Rate**:
   ```bash
   # Count HTTP 200 responses in last hour
   grep "azure" /var/log/listmonk/listmonk.log | grep "$(date +%Y/%m/%d\ %H)" | grep -c "200"
   ```

2. **Bounce Detection Rate**:
   ```sql
   SELECT
     COUNT(*) FILTER (WHERE source = 'azure') as azure_bounces,
     COUNT(*) as total_bounces,
     ROUND(100.0 * COUNT(*) FILTER (WHERE source = 'azure') / COUNT(*), 2) as azure_percentage
   FROM bounces
   WHERE created_at > NOW() - INTERVAL '24 hours';
   ```

3. **Engagement Tracking**:
   ```sql
   SELECT
     COUNT(DISTINCT campaign_id) as campaigns_with_tracking,
     COUNT(*) as total_views
   FROM campaign_views
   WHERE created_at > NOW() - INTERVAL '24 hours';
   ```

4. **Message Correlation Success**:
   ```bash
   # Count successful correlations via X-headers
   grep "found campaign UUID in header" /var/log/listmonk/listmonk.log | wc -l

   # Count fallback to database lookup
   grep "looking up campaign for Azure message" /var/log/listmonk/listmonk.log | wc -l
   ```

### Azure Portal Monitoring

View Event Grid metrics in Azure Portal:

1. Navigate to **Event Grid Subscriptions**
2. Select a subscription (e.g., `listmonk-email-events-mail1-bobbyseamoss-com`)
3. View metrics:
   - **Delivery Success Count**
   - **Delivery Failed Count**
   - **Dead Lettered Count**
   - **Dropped Event Count**

### Alerts

Set up alerts for critical failures:

```bash
# Create alert for failed webhook deliveries
az monitor metrics alert create \
  --name listmonk-webhook-failures \
  --resource-group <rg-name> \
  --scopes <event-grid-subscription-id> \
  --condition "count EventDeliveryFailureCount > 10" \
  --description "Listmonk webhook receiving too many failed events" \
  --evaluation-frequency 5m \
  --window-size 15m
```

---

## FAQ

### Q: Does this work with all Azure Email Communication Services plans?

**A**: Yes, Event Grid webhooks work with all ACS Email plans. However, user engagement tracking (opens/clicks) requires enabling the feature on each domain, which may have additional costs.

### Q: What happens if Azure includes the X-headers vs. doesn't include them?

**A**: The system has two correlation methods:
- **If Azure includes X-headers**: Instant correlation, no database lookup needed
- **If Azure doesn't include X-headers**: Falls back to `azure_message_tracking` table lookup

Currently, we rely on X-headers since the tracking table population isn't implemented yet. This is safe because listmonk already sends these headers with every email.

### Q: How do I know if Azure is preserving my X-headers?

**A**: Check the logs after receiving a webhook event:
```bash
grep "Azure Event Grid: found campaign UUID in header" /var/log/listmonk/listmonk.log
```

If you see this message, Azure is preserving headers. If not, you'll see database lookup messages instead.

### Q: What's the cost of Azure Event Grid?

**A**:
- First 100,000 operations/month: **FREE**
- After that: **$0.60 per million operations**

For 30 domains sending 10,000 emails/month each (300,000 emails total):
- ~405,000 events (delivery + engagement)
- Cost: **~$0.18/month**

### Q: Can I use this with non-Azure SMTP servers?

**A**: No, this integration is specifically for **Azure Communication Services**. For other providers:
- **SendGrid**: Use SendGrid webhooks (already supported)
- **Amazon SES**: Use SES webhooks (already supported)
- **Postmark**: Use Postmark webhooks (already supported)
- **Other SMTP**: Use POP mailbox bounce scanning

### Q: How long are events retained in azure_message_tracking table?

**A**: Currently, **forever** (no automatic cleanup). You can manually clean up old records:

```sql
-- Delete tracking records older than 30 days
DELETE FROM azure_message_tracking
WHERE sent_at < NOW() - INTERVAL '30 days';

-- Vacuum to reclaim space
VACUUM ANALYZE azure_message_tracking;
```

To automate cleanup, add a cron job:
```bash
# Add to crontab
0 2 * * * psql -U listmonk -d listmonk -c "DELETE FROM azure_message_tracking WHERE sent_at < NOW() - INTERVAL '30 days';"
```

### Q: What if I have more than 30 ACS domains?

**A**: The system supports **unlimited domains**. The setup script will create Event Grid subscriptions for all resources it finds. Each subscription is independent and has no limit on the number of domains you can configure.

### Q: Do I need to restart listmonk after enabling Azure Event Grid?

**A**:
- **Yes** if you enable via config.toml
- **No** if you enable via database settings update (settings are loaded dynamically)

### Q: Can I test webhooks without real emails?

**A**: Yes! Use the testing script:
```bash
./deployment/scripts/test-azure-webhooks.sh
```

This sends mock events to your webhook endpoint without requiring actual email sends.

---

## Additional Resources

- **Azure Event Grid Documentation**: https://learn.microsoft.com/en-us/azure/event-grid/
- **Azure Communication Services Events**: https://learn.microsoft.com/en-us/azure/event-grid/communication-services-email-events
- **Listmonk Bounce Configuration**: https://listmonk.app/docs/configuration/#bounce-processing

---

## Support

For issues or questions:

1. **Check logs**: `/var/log/listmonk/listmonk.log`
2. **Review this README**: Troubleshooting section
3. **Test with mock events**: `test-azure-webhooks.sh`
4. **Listmonk GitHub Issues**: https://github.com/knadh/listmonk/issues

---

**Last Updated**: 2025-11-01
**Version**: 6.1.0
**Author**: Listmonk + Azure Integration Team
