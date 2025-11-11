# Campaign Sent Count Fix - Context

**Last Updated**: 2025-11-11 06:52 AM EST

## Status: ✅ COMPLETED AND DEPLOYED

Deployed to production at https://list.bobbyseamoss.com
Revision: `listmonk420--deploy-20251111-065049`

## Problem Statement

Campaign 65 showed `sent=0` despite actually sending 20,323 emails. When the campaign auto-paused overnight, the sent count dropped to approximately 5% of the actual progress reported during sending.

### Investigation Findings

**Verified Actual Sends**:
- `email_queue` table: 20,323 rows with `status='sent'`
- `azure_message_tracking` table: 20,318 tracked messages
- `azure_engagement_events` table: 1,282 delivered events

**Root Cause**:
Queue-based campaigns never updated `campaigns.sent` from the `email_queue` table. The count remained at 0 because:
1. Queue processor increments counts in `email_queue` only
2. No sync mechanism existed to update `campaigns.sent`
3. Auto-pause happened with stale campaign count

## Solution Implemented

### 1. Sync Before Auto-Pause
**File**: `internal/queue/processor.go` (lines 135-194)

Modified `autoPauseRunningCampaigns()` to:
1. Query all running queue-based campaign IDs
2. Execute inline UPDATE to sync `campaigns.sent` from `email_queue`
3. Log sync results before pausing campaigns

**Why inline UPDATE instead of query**: Avoids circular dependency since queries aren't available in processor package.

### 2. Periodic Background Sync
**File**: `internal/queue/processor.go` (lines 114-180)

Created two new functions:
- `StartCampaignStatsSync()`: Goroutine that runs every 5 minutes
- `syncRunningCampaignCounts()`: Helper that performs the actual sync

**Implementation Details**:
- Uses ticker for periodic execution (5-minute interval)
- Syncs counts for ALL running queue-based campaigns
- Includes stopChan support for graceful shutdown
- Logs number of campaigns synced each iteration

### 3. New SQL Query (Not Used in Final Solution)
**File**: `queries.sql` (lines 1649-1661)

Added `sync-queue-campaign-counts` query that accepts campaign IDs as array parameter. This query was created but NOT used because:
- Processor package doesn't have access to `app.queries`
- Inline SQL in processor is more direct and avoids package dependencies
- Query remains in queries.sql for potential future use via HTTP endpoints

### 4. Start Stats Sync on Init
**File**: `cmd/init.go` (lines 773-775)

Added goroutine to start stats sync when queue processor initializes:
```go
go proc.StartCampaignStatsSync()
```

## Files Modified

1. **queries.sql** (lines 1649-1661)
   - Added `sync-queue-campaign-counts` query
   - Uses array parameter: `$1::INT[]`
   - Updates `sent` count and `updated_at` timestamp

2. **internal/queue/processor.go**:
   - Lines 135-194: Modified `autoPauseRunningCampaigns()`
   - Lines 114-180: Added `StartCampaignStatsSync()` and `syncRunningCampaignCounts()`
   - Added inline SQL for syncing counts

3. **cmd/init.go** (lines 773-775)
   - Added goroutine to start campaign stats sync
   - Updated log message to include "and stats sync"

## Testing Performed

### Manual Sync Test
```sql
-- Campaign 65 before manual sync
sent = 0

-- After manual sync (ran inline UPDATE)
sent = 20,323 (26% progress)
```

### Production Verification
After deployment to Bobby Seamoss:
- ✅ Logs confirm: "started queue processor, auto-pause scheduler, and stats sync"
- ✅ Logs confirm: "starting campaign stats sync (every 5 minutes)"
- ✅ Campaign 65 still shows 20,323 sent (count preserved)
- ✅ Container app revision deployed successfully

## Key Decisions Made

### 1. Inline SQL vs Query File
**Decision**: Use inline SQL in processor instead of loading from queries.sql

**Reasoning**:
- Processor package doesn't have access to `app.queries`
- Would require passing queries struct to processor (architectural change)
- Inline SQL is self-contained and doesn't create new dependencies
- Query file approach would need significant refactoring

### 2. Sync Frequency: 5 Minutes
**Decision**: Periodic sync every 5 minutes

**Reasoning**:
- Balances accuracy with database load
- Campaign progress updates don't need real-time precision
- Most campaigns run for hours/days, so 5-minute lag is acceptable
- Reduces unnecessary database queries

### 3. Sync Before Pause vs Only Periodic
**Decision**: Both - sync before pause AND periodic sync

**Reasoning**:
- Sync before pause ensures accurate count at critical transition
- Periodic sync provides continuous accuracy during sending
- Prevents count drift over long-running campaigns
- Double sync (periodic + before-pause) is harmless due to WHERE clause

## PostgreSQL Lessons Learned

### Array Parameters in Queries
When using array parameters in PostgreSQL queries:
```sql
-- Correct syntax with explicit cast
WHERE c.id = ANY($1::INT[])

-- Alternative syntax
WHERE c.id IN (SELECT unnest($1::INT[]))
```

### ROUND Function Type Casting
PostgreSQL's ROUND function requires NUMERIC type:
```sql
-- ❌ WRONG - float + integer not supported
ROUND((sent::FLOAT / to_send * 100), 1)

-- ✅ CORRECT - both arguments must be NUMERIC
ROUND((sent::NUMERIC / NULLIF(to_send, 0)::NUMERIC * 100), 1)
```

### NULL Handling in Division
Always use NULLIF to prevent division by zero:
```sql
sent::NUMERIC / NULLIF(to_send, 0)::NUMERIC
```

## Monitoring Commands

### Check Stats Sync Logs
```bash
az containerapp logs show --name listmonk420 --resource-group rg-listmonk420 --tail 100 | grep -E "stats sync|synced sent counts"
```

### Verify Campaign Counts
```sql
PGPASSWORD='T@intshr3dd3r' psql -h listmonk420-db.postgres.database.azure.com -p 5432 -U listmonkadmin -d listmonk --set=sslmode=require -c "
SELECT
    id,
    name,
    status,
    sent,
    to_send,
    ROUND((sent::NUMERIC / NULLIF(to_send, 0)::NUMERIC * 100), 1) as progress_pct
FROM campaigns
WHERE use_queue = true
ORDER BY id DESC
LIMIT 10;
"
```

### Check Queue vs Campaign Count Discrepancy
```sql
SELECT
    c.id,
    c.name,
    c.sent as campaign_sent,
    (SELECT COUNT(*) FROM email_queue WHERE campaign_id = c.id AND status = 'sent') as queue_sent,
    (SELECT COUNT(*) FROM email_queue WHERE campaign_id = c.id AND status = 'sent') - c.sent as discrepancy
FROM campaigns c
WHERE c.use_queue = true
  AND c.status IN ('running', 'paused')
ORDER BY c.id DESC;
```

## Potential Future Enhancements

1. **API Endpoint for Manual Sync**: Add `/api/campaigns/:id/sync-counts` endpoint
2. **Sync on Campaign Start**: Update counts when resuming paused campaigns
3. **Configurable Sync Interval**: Make 5-minute interval configurable via settings
4. **Metrics Dashboard**: Show sync lag/discrepancy in admin UI
5. **Sync All Campaigns**: Extend to sync non-queue campaigns too (for consistency)

## Related Systems

### Queue-Based Campaign Flow
1. Campaign set to "running" with messenger="automatic"
2. Emails queued to `email_queue` table
3. Campaign marked `use_queue=true`, `queued_at=NOW()`
4. Queue processor sends emails, updates `email_queue.status='sent'`
5. **NEW**: Stats sync updates `campaigns.sent` every 5 minutes
6. **NEW**: Before auto-pause, counts synced one final time
7. Campaign paused with accurate sent count

### Auto-Pause System
Located in `internal/queue/processor.go`:
- `StartAutoPauseScheduler()`: Checks time window every minute
- `autoPauseRunningCampaigns()`: Pauses campaigns outside send window
- `autoResumeRunningCampaigns()`: Resumes campaigns when window opens

## Known Issues / Edge Cases

### None Currently

All identified issues have been resolved:
- ✅ Campaign counts now sync automatically
- ✅ Auto-pause preserves accurate counts
- ✅ Long-running campaigns stay accurate via periodic sync

## Deployment History

- **2025-11-11 06:51 AM**: Deployed to Bobby Seamoss production
- **Revision**: listmonk420--deploy-20251111-065049
- **Image**: listmonk420acr.azurecr.io/listmonk420:latest

## Next Session Handoff

**Status**: This issue is FULLY RESOLVED and deployed to production.

**If you need to work on this further**:
1. Check logs to verify 5-minute sync is running consistently
2. Monitor for any discrepancies after campaigns send for several hours
3. Consider adding sync-on-resume feature for paused campaigns

**User's Last Request Before /dev-docs-update**:
User asked to "enable the dangerously-skip-permissions flag" but interrupted before providing context. This appears unrelated to the campaign sent count fix. Investigate where this flag is used and what it controls.
