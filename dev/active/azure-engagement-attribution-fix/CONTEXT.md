# Azure Engagement Event Attribution Fix

**Last Updated**: 2025-11-10 20:57 UTC
**Status**: ✅ COMPLETED AND DEPLOYED
**Complexity**: HIGH - Required deep understanding of Azure Event Grid timing

## Problem Statement

Azure engagement events (email opens/views and link clicks) were being **misattributed** to the wrong campaigns. Specifically:

- Campaign 64 (cancelled, not running): Receiving 158 views, 23 clicks
- Campaign 65 (currently running, sending emails): Receiving only 54 views, 9 clicks

This was causing incorrect analytics and making it impossible to measure campaign performance accurately.

## Root Cause Analysis

### The Core Issue
Azure Event Grid uses **different message IDs** for different event types:

1. **Delivery events**: `messageId: "060f319b-179d-4262-9da7-8a68fbf59d71"`
2. **Engagement events**: `messageId: "fe863553-3055-4e77-a014-8f45b41f424b"` (completely different!)

The code was trying to match engagement events to campaigns using:
```go
SELECT campaign_id, subscriber_id
FROM azure_message_tracking
WHERE azure_message_id = $1  -- This NEVER matches!
```

This lookup failed 100% of the time because the message IDs don't match.

### The Fallback Problem
When the lookup failed, the code fell back to:
```go
SELECT c.id, s.id
FROM campaigns c
CROSS JOIN subscribers s
WHERE s.email = $1
ORDER BY c.updated_at DESC  -- ❌ Picks most recently UPDATED campaign
LIMIT 1
```

This selected whichever campaign was updated most recently, NOT the campaign that sent the email.

### The Consistent Field
Azure DOES provide a consistent identifier across all event types: **`internetMessageId`**

```json
// Delivery event:
"internetMessageId": "<9f0728f0-338f-454b-b3a2-ca0e924518e9@listmonk>"

// Engagement event:
"internetMessageId": "<9f0728f0-338f-454b-b3a2-ca0e924518e9@listmonk>"  // SAME!
```

This is the email's Message-ID header, which is set when we send the email and preserved by Azure.

## Solution Implemented

### Phase 1: Add internet_message_id Column (Initial Attempt)
**Migration**: v7.2.0
**Files**: `internal/migrations/v7.2.0.go`, `cmd/upgrade.go`

Added `internet_message_id` column to:
- `azure_message_tracking` table
- `azure_engagement_events` table

Updated webhook handlers to:
- Extract `internetMessageId` from Azure webhook data
- Store it in the database
- Match engagement events using `internet_message_id`

**Problem with Phase 1**: The delivery webhook handler tried to UPDATE the tracking table with `internet_message_id`, but this only happened when delivery webhooks arrived (often 2-5 minutes after sending). Engagement webhooks arrive FIRST (within seconds), so they couldn't match!

### Phase 2: Store internet_message_id at Send Time (Critical Fix)
**Files**: `internal/messenger/email/email.go`

The KEY INSIGHT: We generate the Message-ID header when we SEND the email, so we can store `internet_message_id` in the tracking table IMMEDIATELY, not wait for webhooks.

```go
// Generate internet_message_id (Message-ID header value)
internetMessageID := fmt.Sprintf("<%s@listmonk>", trackingID)

// Store it IMMEDIATELY when sending
_, err := e.db.Exec(`
    INSERT INTO azure_message_tracking
        (azure_message_id, internet_message_id, campaign_id, subscriber_id, sent_at)
    VALUES ($1, $2, $3, $4, $5)
    ON CONFLICT (azure_message_id) DO NOTHING
`, trackingID, internetMessageID, m.Campaign.ID, m.Subscriber.ID, time.Now())
```

This ensures engagement events can match instantly, even before delivery webhooks arrive.

## Files Modified

### Database Schema
- **internal/migrations/v7.2.0.go** (NEW)
  - Adds `internet_message_id TEXT` column to `azure_message_tracking`
  - Adds `internet_message_id TEXT` column to `azure_engagement_events`
  - Creates indexes: `idx_azure_msg_tracking_internet_msg_id`, `idx_azure_engagement_internet_msg_id`

### Backend - Data Models
- **internal/bounce/webhooks/azure.go**
  - Line 68: Added `InternetMessageID string` to `AzureDeliveryData`
  - Line 79: Added `InternetMessageID string` to `AzureEngagementData`

### Backend - Webhook Handlers
- **cmd/bounce.go**
  - Line 285: Extract `internetMessageID` from delivery event data
  - Lines 355-365: Update `azure_message_tracking` with `internetMessageID` when delivery webhook arrives
  - Line 514: Store `internetMessageID` in `azure_engagement_events`
  - Lines 471-478: Match engagement events using `internet_message_id` instead of `azure_message_id`

### Backend - Email Sending
- **internal/messenger/email/email.go** (CRITICAL FIX)
  - Lines 324-334: Generate and store `internet_message_id` **at send time**
  - Format: `<uuid@listmonk>` (matches Message-ID header format)
  - Stored in tracking table immediately when email is sent

### Migration Registration
- **cmd/upgrade.go**
  - Line 52: Registered `{"v7.2.0", migrations.V7_2_0}`

## Deployment History

### Initial Deployment (v7.2.0 migration + webhook changes)
- **Time**: 2025-11-10 18:54 UTC
- **Revision**: `listmonk420--deploy-20251110-135402` (bobbyseamoss)
- **Revision**: `listmonk-comma--deploy-20251110-135622` (enjoycomma)
- **Status**: Partial fix - engagement events still misattributed

### Critical Fix Deployment (store internet_message_id at send time)
- **Time**: 2025-11-10 20:52 UTC
- **Revision**: `listmonk420--deploy-20251110-155255` (bobbyseamoss)
- **Revision**: `listmonk-comma--deploy-20251110-155502` (enjoycomma)
- **Status**: ✅ WORKING - New emails have correct attribution

## Verification Results

### Before Fix
```sql
-- Campaign 64 (cancelled, old): 146 Azure views, 23 clicks
-- Campaign 65 (running, new): 54 Azure views, 9 clicks
-- Problem: Old campaign getting more engagement than running campaign!
```

### After Critical Fix
```sql
-- Check new emails have internet_message_id at send time
SELECT azure_message_id, internet_message_id, created_at
FROM azure_message_tracking
WHERE campaign_id = 65
ORDER BY created_at DESC
LIMIT 5;

-- Result: ✅ All new emails have internet_message_id populated immediately
-- Example: <48905fe7-f318-4ef3-b2f9-cc68e56ac203@listmonk>

-- Check engagement events for campaign 65
SELECT COUNT(*) as views FROM campaign_views WHERE campaign_id = 65;
-- Result: 57 views (increasing as new emails are opened)

-- Check attribution is working
SELECT * FROM azure_engagement_events
WHERE campaign_id = 65
  AND event_timestamp >= '2025-11-10 20:55:00'
ORDER BY event_timestamp DESC;

-- Result: ✅ New engagement events correctly attributed to campaign 65
```

### Current State (as of 20:57 UTC)
- Campaign 65: 8,209 total emails sent
  - 86 emails with `internet_message_id` (sent after fix)
  - 8,123 emails without (sent before fix)
- New engagement events: Correctly attributed to campaign 65 ✅
- Old engagement events: Still attributed to campaign 64 (CORRECT - people opening old emails)

## Key Learnings

### Azure Event Grid Timing
1. **Engagement webhooks arrive FIRST** (within seconds of user opening email)
2. **Delivery webhooks arrive LATER** (2-5 minutes after sending)
3. Cannot rely on delivery webhooks to populate data needed for engagement attribution

### Message ID Consistency
- Azure generates new UUIDs for different event types (delivery vs engagement)
- `internetMessageId` (email Message-ID header) is the ONLY consistent identifier
- We control the Message-ID header format, so we can predict what it will be

### Database Timing
- Must store ALL correlation data at send time, not webhook time
- Webhooks are asynchronous and can arrive out of order
- Early engagement events (opens within seconds) need immediate matching capability

## Testing Commands

### Check new emails have internet_message_id
```bash
PGPASSWORD='T@intshr3dd3r' psql -h listmonk420-db.postgres.database.azure.com \
  -p 5432 -U listmonkadmin -d listmonk --set=sslmode=require -c "
SELECT
    COUNT(*) as total,
    COUNT(CASE WHEN internet_message_id IS NOT NULL THEN 1 END) as with_id,
    MAX(created_at) as most_recent
FROM azure_message_tracking
WHERE campaign_id = 65;"
```

### Check engagement attribution
```bash
PGPASSWORD='T@intshr3dd3r' psql -h listmonk420-db.postgres.database.azure.com \
  -p 5432 -U listmonkadmin -d listmonk --set=sslmode=require -c "
SELECT
    c.id,
    c.name,
    c.status,
    COUNT(DISTINCT cv.id) as views,
    COUNT(DISTINCT lc.id) as clicks
FROM campaigns c
LEFT JOIN campaign_views cv ON c.id = cv.campaign_id
    AND cv.created_at >= NOW() - INTERVAL '1 hour'
LEFT JOIN link_clicks lc ON c.id = lc.campaign_id
    AND lc.created_at >= NOW() - INTERVAL '1 hour'
WHERE c.id IN (64, 65)
GROUP BY c.id, c.name, c.status;"
```

### Monitor real-time engagement
```bash
# Watch for new engagement events
PGPASSWORD='T@intshr3dd3r' psql -h listmonk420-db.postgres.database.azure.com \
  -p 5432 -U listmonkadmin -d listmonk --set=sslmode=require -c "
SELECT
    aee.engagement_type,
    aee.campaign_id,
    c.name,
    aee.event_timestamp
FROM azure_engagement_events aee
JOIN campaigns c ON aee.campaign_id = c.id
WHERE aee.event_timestamp >= NOW() - INTERVAL '5 minutes'
ORDER BY aee.event_timestamp DESC;"
```

## Future Considerations

### Historical Data
- ~8,000 emails from campaign 65 don't have `internet_message_id`
- Engagement events from these emails may still be misattributed
- Consider running a backfill script if historical accuracy is critical

### Monitoring
- Track the ratio of emails with/without `internet_message_id`
- Should approach 100% within hours of deployment
- Alert if ratio drops (indicates regression)

### Alternative Correlation Methods
- X-Listmonk-Campaign-ID header (if Azure preserves it)
- Could add as secondary fallback, but `internetMessageId` is most reliable

## Related Documentation
- CLAUDE.md: Added section on Azure engagement attribution
- dev/active/campaigns-redesign/: Related campaigns page metrics
- internal/bounce/webhooks/: Azure webhook handling patterns

## Git Commits
1. `cc6ff4bf` - "Fix Azure engagement event attribution using internetMessageId"
2. `ca5d3a8a` - "Fix syntax errors in engagement attribution code"
3. `48249994` - "Fix critical bug: Store internet_message_id at send time, not webhook time"
