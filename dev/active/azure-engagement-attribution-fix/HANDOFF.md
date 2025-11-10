# Session Handoff - Azure Engagement Attribution Fix

**Session End Time**: 2025-11-10 20:57 UTC
**Status**: ✅ COMPLETED - All work finished and deployed
**Next Session Action**: Monitor engagement metrics for campaign 65

## What Was Accomplished This Session

### Problem Solved
Fixed critical bug where Azure engagement events (email opens and clicks) were being attributed to the wrong campaigns due to Azure Event Grid using different message IDs for delivery vs engagement events.

### Solution Delivered
- Created v7.2.0 database migration adding `internet_message_id` columns
- Updated webhook handlers to use `internetMessageId` field from Azure
- **Critical fix**: Store `internet_message_id` at email send time (not webhook time)
- Deployed to both production sites

### Files Modified (All Changes Committed)
1. `internal/migrations/v7.2.0.go` - NEW migration
2. `cmd/upgrade.go` - Register v7.2.0
3. `internal/bounce/webhooks/azure.go` - Add InternetMessageID fields
4. `cmd/bounce.go` - Update webhook handlers
5. `internal/messenger/email/email.go` - **Store internet_message_id at send time**

### Deployments
- list.bobbyseamoss.com: Revision `listmonk420--deploy-20251110-155255`
- list.enjoycomma.com: Revision `listmonk-comma--deploy-20251110-155502`

## Current System State

### Database
- Migration v7.2.0 applied to both databases
- `internet_message_id` column exists in:
  - `azure_message_tracking`
  - `azure_engagement_events`
- Indexes created for fast lookups

### Campaign 65 (Currently Running)
- Total emails sent: 8,209
- Emails with internet_message_id: 86 (sent after fix)
- Views: 57 (increasing)
- Clicks: 11
- **Status**: New engagement events correctly attributed ✅

### Campaign 64 (Cancelled)
- Still receiving engagement events from OLD emails (expected behavior)
- This is CORRECT - people are opening emails sent days ago

## No Pending Work

All tasks completed:
- ✅ Root cause identified
- ✅ Solution implemented
- ✅ Fix tested and verified
- ✅ Deployed to production
- ✅ Documentation updated

## If You Need to Investigate Further

### Check Engagement Attribution
```bash
PGPASSWORD='T@intshr3dd3r' psql -h listmonk420-db.postgres.database.azure.com \
  -p 5432 -U listmonkadmin -d listmonk --set=sslmode=require -c "
SELECT
    c.id,
    c.name,
    c.status,
    COUNT(DISTINCT cv.id) as views_last_hour,
    COUNT(DISTINCT lc.id) as clicks_last_hour
FROM campaigns c
LEFT JOIN campaign_views cv ON c.id = cv.campaign_id
    AND cv.created_at >= NOW() - INTERVAL '1 hour'
LEFT JOIN link_clicks lc ON c.id = lc.campaign_id
    AND lc.created_at >= NOW() - INTERVAL '1 hour'
WHERE c.id IN (64, 65)
GROUP BY c.id, c.name, c.status;"
```

### Check internet_message_id Population Rate
```bash
PGPASSWORD='T@intshr3dd3r' psql -h listmonk420-db.postgres.database.azure.com \
  -p 5432 -U listmonkadmin -d listmonk --set=sslmode=require -c "
SELECT
    campaign_id,
    COUNT(*) as total,
    COUNT(CASE WHEN internet_message_id IS NOT NULL THEN 1 END) as with_id,
    ROUND(100.0 * COUNT(CASE WHEN internet_message_id IS NOT NULL THEN 1 END) / COUNT(*), 2) as percent_with_id
FROM azure_message_tracking
WHERE campaign_id = 65
GROUP BY campaign_id;"
```

## Key Technical Insights for Future Reference

### Azure Event Grid Timing
- **Engagement webhooks arrive FIRST** (within seconds)
- **Delivery webhooks arrive LATER** (2-5 minutes)
- Must store correlation data at SEND time, not webhook time

### Message ID Formats
- Azure generates different UUIDs for different event types
- `internetMessageId` is the email's Message-ID header
- Format we use: `<uuid@listmonk>`
- This is the ONLY consistent identifier across all Azure event types

### Why Two Deployments Were Needed
1. **First deployment**: Added columns, updated webhooks - DIDN'T WORK
2. **Problem discovered**: internetMessageId only populated when delivery webhook arrived (too late)
3. **Second deployment**: Store internetMessageId at send time - WORKS ✅

## Context for Next Session

### Expected Behavior Going Forward
- All NEW emails will have `internet_message_id` populated immediately
- Engagement events will be correctly attributed to the sending campaign
- You may still see engagement from old emails (campaign 64) - this is CORRECT

### Monitoring Suggestions
- Watch campaign 65 metrics increase over next few hours
- By tomorrow, most of campaign 65's 8,209 emails should have been attempted
- Engagement rate should stabilize around the project's normal metrics

### No Action Required
System is working correctly. This was a complete fix, not a workaround.

## Related Documentation
- `/dev/active/azure-engagement-attribution-fix/CONTEXT.md` - Full technical details
- `/dev/active/azure-engagement-attribution-fix/TASKS.md` - Task checklist
- `CLAUDE.md` - Updated with Azure attribution information

## Git History
```
48249994 - Fix critical bug: Store internet_message_id at send time, not webhook time
ca5d3a8a - Fix syntax errors in engagement attribution code
cc6ff4bf - Fix Azure engagement event attribution using internetMessageId
```

All changes committed and deployed. No uncommitted work.
