# Session Summary - November 10, 2025 (Evening)

**Time**: 18:00 - 21:00 UTC
**Duration**: 3 hours
**Status**: ✅ COMPLETED - Critical bug fixed and deployed

## Primary Issue Resolved

### Azure Engagement Event Misattribution
**Severity**: HIGH - Affecting all campaign analytics
**Impact**: Views and clicks were being attributed to wrong campaigns

### Root Cause
Azure Event Grid uses **different message IDs** for different event types:
- Delivery events: `messageId: "uuid-1"`
- Engagement events: `messageId: "uuid-2"` (completely different!)

The lookup logic failed because these IDs don't match, causing fallback to "most recently updated campaign" which was incorrect.

### Solution
Use `internetMessageId` (email Message-ID header) which is **consistent** across all Azure event types.

**Critical insight**: Must store `internet_message_id` at EMAIL SEND TIME, not when delivery webhooks arrive, because:
1. Engagement webhooks arrive within seconds of user opening email
2. Delivery webhooks arrive 2-5 minutes after sending
3. Early engagement events need immediate matching capability

## Changes Delivered

### Database Migration v7.2.0
Created migration to add `internet_message_id` columns:
- `azure_message_tracking.internet_message_id`
- `azure_engagement_events.internet_message_id`
- Indexes for fast lookups

### Code Changes
1. **internal/messenger/email/email.go** (CRITICAL)
   - Store `internet_message_id` at send time
   - Format: `<uuid@listmonk>` (matches Message-ID header)

2. **cmd/bounce.go**
   - Extract `internetMessageId` from webhook data
   - Match engagement events using `internet_message_id`

3. **internal/bounce/webhooks/azure.go**
   - Add `InternetMessageID` fields to data structures

### Deployments
- **list.bobbyseamoss.com**: `listmonk420--deploy-20251110-155255`
- **list.enjoycomma.com**: `listmonk-comma--deploy-20251110-155502`

## Verification Results

### Before Fix
- Campaign 64 (cancelled): 158 views, 23 clicks ❌ WRONG
- Campaign 65 (running): 54 views, 9 clicks

### After Fix
- Campaign 65: New engagement events correctly attributed ✅
- 86 new emails sent with `internet_message_id` populated immediately
- First correctly attributed view confirmed at 20:55:59 UTC

## Technical Insights

### Azure Event Grid Timing (Critical Discovery)
```
Time 0:00 - Email sent
Time 0:02 - User opens email → Engagement webhook arrives
Time 0:05 - Email delivered → Delivery webhook arrives
```

**Lesson**: Cannot rely on delivery webhooks to populate data needed for engagement attribution. Must store ALL correlation data at send time.

### Message ID Consistency
- Azure message IDs are NOT consistent across event types
- `internetMessageId` (Message-ID header) IS consistent
- We control the Message-ID format, so we can predict its value

## Files Modified (All Committed)

### New Files
- `internal/migrations/v7.2.0.go`
- `dev/active/azure-engagement-attribution-fix/CONTEXT.md`
- `dev/active/azure-engagement-attribution-fix/TASKS.md`
- `dev/active/azure-engagement-attribution-fix/HANDOFF.md`

### Modified Files
- `cmd/upgrade.go` - Register v7.2.0 migration
- `cmd/bounce.go` - Update webhook handlers
- `internal/bounce/webhooks/azure.go` - Add InternetMessageID fields
- `internal/messenger/email/email.go` - **Store internet_message_id at send time**

## Git Commits
```
48249994 - Fix critical bug: Store internet_message_id at send time, not webhook time
ca5d3a8a - Fix syntax errors in engagement attribution code
cc6ff4bf - Fix Azure engagement event attribution using internetMessageId
```

## Current System State

### Production Status
- ✅ Both sites running fixed code
- ✅ Database migrations applied
- ✅ New emails have correct attribution
- ✅ Engagement events matching correctly

### Campaign 65 Metrics (20:57 UTC)
- Total emails sent: 8,209
- Emails with internet_message_id: 86 (sent after fix)
- Views: 57 (increasing as new emails opened)
- Clicks: 11
- Attribution: ✅ WORKING

### Expected Behavior
- All NEW engagement events: Correctly attributed to sending campaign
- Old engagement events from campaign 64: Still arriving (CORRECT - people opening old emails)
- internet_message_id population rate: Should reach 100% within hours

## Monitoring Commands

### Check Attribution Working
```bash
PGPASSWORD='T@intshr3dd3r' psql -h listmonk420-db.postgres.database.azure.com \
  -p 5432 -U listmonkadmin -d listmonk --set=sslmode=require -c "
SELECT c.id, c.name, c.status,
  COUNT(DISTINCT cv.id) as views_last_hour
FROM campaigns c
LEFT JOIN campaign_views cv ON c.id = cv.campaign_id
  AND cv.created_at >= NOW() - INTERVAL '1 hour'
WHERE c.id IN (64, 65)
GROUP BY c.id, c.name, c.status;"
```

### Check internet_message_id Population
```bash
PGPASSWORD='T@intshr3dd3r' psql -h listmonk420-db.postgres.database.azure.com \
  -p 5432 -U listmonkadmin -d listmonk --set=sslmode=require -c "
SELECT
  COUNT(*) as total,
  COUNT(CASE WHEN internet_message_id IS NOT NULL THEN 1 END) as with_id,
  ROUND(100.0 * COUNT(CASE WHEN internet_message_id IS NOT NULL THEN 1 END) / COUNT(*), 2) as percent
FROM azure_message_tracking
WHERE campaign_id = 65
  AND created_at >= NOW() - INTERVAL '1 hour';"
```

## Related Work This Session

### Earlier Today
- Fixed timezone and auto-pause/resume functionality (v7.1.0)
- Fixed campaign progress bar for queue-based campaigns
- Fixed performance summary metrics display

### Continuous Work
- Campaign redesign with enhanced metrics
- Azure engagement tracking system improvements

## Key Learnings

### What Made This Complex
1. **Two-phase fix required**: Initial attempt didn't work due to timing issue
2. **Testing difficulty**: Required production data to observe webhook timing
3. **Asynchronous nature**: Webhooks arrive out of order
4. **Azure architecture**: Different identifiers for different events

### What Made Success Possible
1. **Deep investigation**: Checked actual webhook data and timing
2. **Database queries**: Verified message IDs didn't match
3. **Understanding Azure**: Read docs on Event Grid message structure
4. **Testing in production**: Observed real timing behavior

### Future Applications
- Always consider webhook timing when designing correlation logic
- Store correlation data as early as possible (ideally at source)
- Test with production data patterns when possible
- Document timing dependencies explicitly

## No Pending Work

All tasks completed and deployed. System working correctly.

### Next Session (if needed)
- Monitor metrics to confirm sustained correct attribution
- Consider historical data backfill if needed
- Add monitoring dashboard for attribution accuracy

## Context Preserved

Full technical details in:
- `/dev/active/azure-engagement-attribution-fix/CONTEXT.md`
- `/dev/active/azure-engagement-attribution-fix/TASKS.md`
- `/dev/active/azure-engagement-attribution-fix/HANDOFF.md`

All work committed, documented, and deployed. Ready for context reset.
