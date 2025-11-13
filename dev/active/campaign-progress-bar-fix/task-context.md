# Campaign Progress Bar Fix - Task Context

**Last Updated**: 2025-11-12 07:00 UTC

## Status: ✅ COMPLETED

## Problem Statement

The progress bar on the Campaigns page was showing incorrect numbers for paused queue-based campaigns:
- **Showing**: 21,934 / 78,190 (Azure delivered count / total)
- **Should Show**: 44,173 / 78,190 (queue sent count / total)

The bug only affected **paused** campaigns. Running campaigns displayed correctly.

## Root Cause Analysis

Found TWO bugs that needed to be fixed together:

### Bug 1: SQL Query Only Fetched Running Campaigns
**File**: `queries.sql` (lines 726-766)
**Query**: `get-campaign-queue-stats`

The query had multiple WHERE clauses filtering for `status = 'running'`:
```sql
-- Lines 730, 735, 740, 765
WHERE campaign_id IN (SELECT id FROM campaigns WHERE status = 'running' ...)
```

When a campaign was paused, it was excluded from queue stats entirely.

### Bug 2: Go Backend Only Fetched Running Campaigns
**File**: `internal/core/campaigns.go` (lines 400-443)
**Function**: `GetRunningCampaignStats()`

Even when the SQL query was fixed to include paused campaigns, the Go function only fetched running campaigns:
```go
// BEFORE - Only running campaigns
if err := c.q.GetCampaignStatus.Select(&out, models.CampaignStatusRunning); err != nil {
    // ...
}
// Then queue stats were merged, but paused campaigns weren't in 'out'
```

This meant queue stats for paused campaigns had nowhere to merge into, so they were lost.

## Solution Implemented

### Fix 1: Update SQL Query
**File**: `queries.sql`
**Lines Modified**: 730, 735, 740, 765

Changed all instances:
```sql
-- BEFORE
WHERE status = 'running'

-- AFTER
WHERE status IN ('running', 'paused')
```

### Fix 2: Update Go Backend
**File**: `internal/core/campaigns.go`
**Lines Modified**: 400-443

Modified `GetRunningCampaignStats()` to fetch BOTH running and paused campaigns:

```go
func (c *Core) GetRunningCampaignStats() ([]models.CampaignStats, error) {
    out := []models.CampaignStats{}

    // Fetch running campaigns
    if err := c.q.GetCampaignStatus.Select(&out, models.CampaignStatusRunning); err != nil && err != sql.ErrNoRows {
        c.log.Printf("error fetching running campaign stats: %v", err)
        return nil, echo.NewHTTPError(http.StatusInternalServerError,
            c.i18n.Ts("globals.messages.errorFetching", "name", "{globals.terms.campaign}", "error", pqErrMsg(err)))
    }

    // Also fetch paused campaigns (for queue-based campaigns that are paused)
    pausedCampaigns := []models.CampaignStats{}
    if err := c.q.GetCampaignStatus.Select(&pausedCampaigns, models.CampaignStatusPaused); err != nil && err != sql.ErrNoRows {
        c.log.Printf("error fetching paused campaign stats: %v", err)
        // Don't fail - just log and continue without paused campaigns
    }

    // Combine running and paused campaigns
    out = append(out, pausedCampaigns...)

    if len(out) == 0 {
        return nil, nil
    }

    // Fetch queue stats for campaigns using the queue system
    // (now paused campaigns are in 'out' so stats will be merged correctly)
    // ... rest of function unchanged
}
```

## Files Modified

1. **queries.sql** (4 lines changed)
   - Line 730: views CTE filter
   - Line 735: clicks CTE filter
   - Line 740: bounces CTE filter
   - Line 765: main WHERE clause

2. **internal/core/campaigns.go** (43 lines modified)
   - Lines 400-443: GetRunningCampaignStats function
   - Added separate query for paused campaigns
   - Append paused to running campaigns before merging queue stats

## Deployment

**Environment**: Bobby Seamoss Production
**URL**: https://list.bobbyseamoss.com
**Revision**: listmonk420--deploy-20251112-065523
**Deployment Time**: 2025-11-12 06:55 UTC

**Build Commands Used**:
```bash
CGO_ENABLED=0 go build -o listmonk -ldflags="-s -w -X 'main.version=dev' -X 'main.versionString=dev'" ./cmd
export LISTMONK_DB_PASSWORD='T@intshr3dd3r' && ./deploy.sh
```

**Deployment Status**: ✅ Successful
- All 30 SMTP servers initialized
- Container healthy and running
- Queue processor started successfully

## Verification

### Database Verification
Directly queried the database to confirm data:
```sql
SELECT id, name, status, use_queue, sent as campaign_sent,
       (SELECT COUNT(*) FROM email_queue WHERE campaign_id = 65 AND status = 'sent') as queue_sent,
       (SELECT COUNT(*) FROM azure_delivery_events WHERE campaign_id = 65 AND status = 'Delivered') as azure_sent
FROM campaigns WHERE id = 65;
```

**Result**:
- Campaign 65: "Non Gmail Gummies"
- Status: paused
- use_queue: true
- campaign_sent: 44167
- queue_sent: 44173 ✅
- azure_sent: 21934

### User Verification
User manually verified at https://list.bobbyseamoss.com:
- Progress bar now shows **44,173 / 78,249** ✅
- Previously showed 21,934 / 78,249 ❌

### Container Logs
Checked deployment logs - all services started successfully:
```bash
az containerapp logs show --name listmonk420 --resource-group rg-listmonk420 --tail 50
```

Confirmed:
- All 30 SMTP messengers initialized
- Queue processor started
- Stats sync goroutine running

## Technical Decisions

### Why Fetch Paused Campaigns in Go Instead of SQL?
Could have modified SQL to return paused campaigns, but the Go approach is cleaner:
1. Maintains separation of concerns (SQL returns what's asked, Go decides what to ask for)
2. Easier to extend in the future (e.g., add "scheduled" status)
3. More explicit about intent in code
4. Doesn't require modifying the SQL query signature

### Why Not Change Function Name?
Kept function name as `GetRunningCampaignStats()` even though it now fetches paused campaigns because:
1. The primary purpose is still getting stats for "active" campaigns (running or paused)
2. The return type and signature didn't change
3. Frontend still calls it through the same endpoint
4. Changing the name would require frontend changes with no functional benefit

### Error Handling for Paused Campaigns
If fetching paused campaigns fails, we log but don't fail the entire request:
```go
if err != nil && err != sql.ErrNoRows {
    c.log.Printf("error fetching paused campaign stats: %v", err)
    // Don't fail - just continue without paused campaigns
}
```

This ensures running campaigns still display even if there's an issue with paused campaigns.

## Testing Challenges

### Playwright Authentication Issues
Multiple attempts to verify via Playwright failed due to authentication:
- Login appeared successful but redirected back to login page
- Screenshots showed login form, not campaigns page
- Could be due to:
  - Session cookie issues
  - CSRF token problems
  - Timing issues with SPA routing

### Workarounds Used
1. Direct database queries to verify data correctness
2. Container log verification for deployment health
3. User manual verification in browser

## Test Campaign Details

**Campaign ID**: 65
**Campaign Name**: "Non Gmail Gummies"
**Status**: paused
**Messenger**: automatic (queue-based)
**use_queue**: true

**Expected Numbers**:
- Queue Sent: 44,173 (from email_queue table)
- Queue Total: 78,249
- Azure Delivered: 21,934 (irrelevant for queue-based display)

## Rollback Procedure

If this fix causes issues, rollback to previous revision:

```bash
az containerapp revision list \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --query "[].name" -o tsv

# Previous stable revision
az containerapp revision activate \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --revision listmonk420--deploy-20251112-064547
```

## Related Work

### Previous Fixes
- **2025-11-11**: Campaign sent count fix for queue-based campaigns
  - Added dual-sync approach (before pause + periodic)
  - Related to this issue but different problem

### Frontend Logic (No Changes Needed)
The frontend in `Campaigns.vue` already had correct logic:
```javascript
// Lines 382-386
if (stats.use_queue || stats.useQueue) {
  const queueTotal = stats.queue_total || stats.queueTotal || 0;
  const queueSent = stats.queue_sent || stats.queueSent || 0;
  return `${queueSent} / ${queueTotal}`;
}
```

The frontend was always ready to display queue stats correctly - the backend just wasn't providing them for paused campaigns.

## Next Steps

### Immediate
- ✅ DONE: Fix is deployed and verified working

### Future Considerations
1. **Add automated tests** for paused campaign stats
2. **Consider** if "scheduled" campaigns need similar treatment
3. **Monitor** for any edge cases with other campaign statuses
4. **Document** in user-facing docs that paused campaigns show queue progress

## Git Status

**Modified Files (Not Yet Committed)**:
- `queries.sql`
- `internal/core/campaigns.go`

**Recommended Commit**:
```bash
git add queries.sql internal/core/campaigns.go

git commit -m "Fix progress bar for paused queue-based campaigns

- Modified GetRunningCampaignStats to fetch both running AND paused campaigns
- Updated get-campaign-queue-stats SQL query to include paused status
- Previously showed Azure delivered count (21934) for paused campaigns
- Now correctly shows queue sent count (44173) for paused campaigns

Fixes progress bar display bug where paused queue-based campaigns
showed incorrect sent count from Azure webhooks instead of actual
queue sent count.

Deployed to Bobby Seamoss: listmonk420--deploy-20251112-065523"
```

## Key Lessons Learned

1. **Check Both SQL and Go**: Bug required fixes in BOTH layers - fixing just one wouldn't work
2. **Paused Campaigns Are Still Active**: They need stats just like running campaigns
3. **Database First**: Direct database queries were most reliable verification method
4. **Error Handling Matters**: Graceful degradation (paused fetch failure doesn't break running stats)
5. **Frontend Was Right**: The bug was entirely backend - frontend code was already correct
