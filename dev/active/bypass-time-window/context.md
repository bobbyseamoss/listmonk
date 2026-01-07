# Bypass Time Window Feature - Implementation Context

**Last Updated**: 2025-11-28
**Status**: DEPLOYED TO PRODUCTION

## Feature Overview

Added a per-campaign option to bypass the global sending time window. This allows test campaigns to be sent immediately regardless of the configured sending hours (e.g., outside 8am-8pm window).

## Problem Solved

When testing queue-based campaigns outside the configured sending time window, the campaign would be auto-paused and emails wouldn't be sent until the time window opened again. This made testing difficult during off-hours.

## Implementation Summary

### Files Modified

1. **`models/models.go`** (line ~246)
   - Added `BypassTimeWindow bool` field to Campaign struct
   - JSON tag: `bypass_time_window`
   - DB tag: `bypass_time_window`

2. **`internal/migrations/v7.3.0.go`** (NEW FILE)
   - Migration to add `bypass_time_window BOOLEAN NOT NULL DEFAULT FALSE` column
   - Registered in `cmd/upgrade.go` as last entry in migList

3. **`internal/queue/processor.go`**
   - Added `hasBypassTimeWindowCampaigns()` function (lines ~678-694) - checks if any running campaigns with bypass=true have queued emails
   - Modified `getNextBatch(bypassOnly bool)` (lines ~697-771) - when `bypassOnly=true`, only fetches emails from campaigns with bypass_time_window=true
   - Modified `processQueue()` (lines ~343-372) - outside time window, checks for bypass campaigns and only processes those
   - Modified `autoPauseRunningCampaigns()` (lines ~204-268) - added `AND bypass_time_window = false` to exclude bypass campaigns from auto-pausing

4. **`queries.sql`**
   - `create-campaign` query: Added `bypass_time_window` as column and `$21` parameter
   - `update-campaign` query: Added `bypass_time_window=$20` to SET clause

5. **`internal/core/campaigns.go`**
   - `CreateCampaign()`: Added `o.BypassTimeWindow` as 21st parameter
   - `UpdateCampaign()`: Added `o.BypassTimeWindow` as 20th parameter

6. **`frontend/src/views/Campaign.vue`**
   - Added `bypassTimeWindow: false` to form data (~line 439)
   - Added UI toggle (lines ~141-148) - only shows when messenger is "automatic"
   - Added to `createCampaign()` data object (~line 650)
   - Added to `updateCampaign()` data object (~line 680)

## Key Technical Decisions

1. **Per-Campaign Flag**: Implemented as a boolean on the campaign rather than a global setting, allowing granular control per campaign.

2. **Queue Processor Logic**: When outside time window, the processor now:
   - First checks if ANY bypass campaigns exist with queued emails
   - If yes, fetches ONLY emails from those bypass campaigns
   - If no, returns immediately (existing behavior)

3. **Auto-Pause Exclusion**: Bypass campaigns are excluded from auto-pause by adding `AND bypass_time_window = false` to the pause query.

4. **UI Visibility**: Toggle only appears when messenger is "automatic" (queue-based), since this feature is irrelevant for direct SMTP sends.

## Deployment (COMPLETED 2025-11-28)

Migration was applied manually:
```sql
ALTER TABLE campaigns ADD COLUMN IF NOT EXISTS bypass_time_window BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE settings SET value = '[..., "v7.3.0"]' WHERE key = 'migrations';
```

Deployed with Docker image to Azure Container Apps revision `listmonk420--deploy-20251126-093256`.

## Testing Checklist

- [x] Migration applied to production database
- [x] Backend deployed and running
- [x] Frontend deployed with toggle
- [ ] Create campaign with automatic messenger
- [ ] Enable "Bypass Sending Time Window" toggle
- [ ] Start campaign outside configured time window
- [ ] Verify campaign is NOT auto-paused
- [ ] Verify emails are processed
- [ ] Verify normal campaigns are still auto-paused
- [ ] Verify toggle saves and loads correctly when editing campaign

## Related Session Work

This session also completed:
- **Azure Engagement Tracking Disabled**: Ran script to disable Azure User Engagement Tracking on all 118 domains across 31 email services (88 disabled, 30 already disabled)
- Background task d83bc4 completed successfully

## Build Verification

```bash
# Backend builds successfully with no errors
go build -o /dev/null cmd/*.go
```
