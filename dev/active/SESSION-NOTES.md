# Session Notes - 2025-11-11

## Session Overview

This session focused on fixing a critical bug in queue-based campaigns where sent counts were not being properly tracked.

## Major Work Completed

### 1. Campaign Sent Count Fix (COMPLETED ✅)

**Problem**: Campaign 65 showed 0 sent emails despite actually sending 20,323 emails. Count dropped to ~5% when auto-paused.

**Solution**: Implemented dual-sync approach:
1. Sync before auto-pause (prevents count loss on pause)
2. Periodic sync every 5 minutes (maintains accuracy during sending)

**Files Modified**:
- `queries.sql` (lines 1649-1661) - Added sync-queue-campaign-counts query
- `internal/queue/processor.go` (lines 114-180, 135-194) - Added sync functions
- `cmd/init.go` (lines 773-775) - Start stats sync goroutine

**Deployed**: Production at https://list.bobbyseamoss.com (revision: listmonk420--deploy-20251111-065049)

**Verification**: Campaign 65 maintains accurate count of 20,323 sent (26% progress)

### 2. Previous Session Work (From Summary)

**Campaign Progress Bar CSS Fix**: Fixed visibility in Chrome
- File: `frontend/src/views/Campaigns.vue` (lines 697-717)
- Changed deprecated `::v-deep` to `:deep()`
- Added explicit heights and display properties

**Browser Testing Skill**: Created new skill for Playwright testing
- Files: `.claude/skills/browser-testing/SKILL.md`, `skill-rules.json`
- Configured environments: Bobby Seamoss, Comma, Local
- Credentials stored in skill

## Technical Discoveries

### PostgreSQL Array Parameters
```sql
-- Correct syntax for array parameters
WHERE c.id = ANY($1::INT[])
```

### PostgreSQL ROUND with Division
```sql
-- Must use NUMERIC type for ROUND
ROUND((sent::NUMERIC / NULLIF(to_send, 0)::NUMERIC * 100), 1)
```

### Go Goroutine Patterns for Background Tasks
```go
// Ticker-based periodic execution
ticker := time.NewTicker(5 * time.Minute)
defer ticker.Stop()

for {
    select {
    case <-ticker.C:
        // Do work
    case <-p.stopChan:
        return
    }
}
```

## Architectural Decisions

### Inline SQL vs Query Files
**Decision**: Used inline SQL in processor.go instead of loading from queries.sql

**Reasoning**:
- Processor package doesn't have access to app.queries
- Avoids circular dependencies
- Self-contained solution

### Sync Frequency
**Decision**: 5-minute interval for periodic sync

**Reasoning**:
- Balances accuracy with database load
- Campaign progress doesn't need real-time updates
- Most campaigns run for hours/days

## Deployment Process Used

```bash
# Build, push, and deploy in one command
export LISTMONK_DB_PASSWORD='T@intshr3dd3r' && ./deploy.sh

# Steps performed:
# 1. Build Docker image (multi-stage: email-builder -> frontend -> backend)
# 2. Push to Azure Container Registry
# 3. Run database migrations (though none needed this time)
# 4. Deploy to Azure Container Apps
# 5. Force new revision with image pull
```

## Credentials Reference

**Bobby Seamoss Production**:
- URL: https://list.bobbyseamoss.com
- Username: adam
- Password: T@intshr3dd3r
- DB Host: listmonk420-db.postgres.database.azure.com
- DB User: listmonkadmin
- DB Name: listmonk
- DB Password: T@intshr3dd3r

**Azure Resources**:
- Resource Group: rg-listmonk420
- Container App: listmonk420
- Container Registry: listmonk420acr.azurecr.io
- Managed Environment: listmonk420-env

## Monitoring & Verification Commands

### Check Stats Sync Logs
```bash
az containerapp logs show --name listmonk420 --resource-group rg-listmonk420 --tail 100 | grep -E "stats sync|synced sent counts"
```

### Verify Campaign Counts
```sql
PGPASSWORD='T@intshr3dd3r' psql -h listmonk420-db.postgres.database.azure.com -p 5432 -U listmonkadmin -d listmonk --set=sslmode=require -c "
SELECT
    id, name, status, sent, to_send,
    ROUND((sent::NUMERIC / NULLIF(to_send, 0)::NUMERIC * 100), 1) as progress_pct
FROM campaigns
WHERE use_queue = true
ORDER BY id DESC LIMIT 10;
"
```

### Check Sync Discrepancies
```sql
SELECT
    c.id, c.name,
    c.sent as campaign_sent,
    (SELECT COUNT(*) FROM email_queue WHERE campaign_id = c.id AND status = 'sent') as queue_sent,
    (SELECT COUNT(*) FROM email_queue WHERE campaign_id = c.id AND status = 'sent') - c.sent as discrepancy
FROM campaigns c
WHERE c.use_queue = true AND c.status IN ('running', 'paused')
ORDER BY c.id DESC;
```

## User's Last Request (Interrupted)

User asked to "enable the dangerously-skip-permissions flag" but interrupted the request with /dev-docs-update command.

**Context**: Unknown what this flag is for or where it's used. Initial grep searches found no matches in codebase for:
- "dangerously-skip-permissions"
- "skip.*permission" (case insensitive)

**Next Steps for New Session**:
1. Ask user for more context about what this flag is for
2. Check if it's a listmonk flag, Azure flag, or something else
3. Search for permission-related settings/flags in codebase
4. Check config.toml, command-line flags, environment variables

## Related Documentation

- Campaign sent count fix details: `/dev/active/campaign-sent-count-fix/context.md`
- Task completion status: `/dev/active/campaign-sent-count-fix/tasks.md`
- PostgreSQL lessons: Reference `CLAUDE.md` SQL Query Best Practices section

## Known Issues

None currently. All identified issues have been resolved and deployed.

## Context at Session End

**Context Usage**: ~58K/200K tokens used
**Reason for /dev-docs-update**: Approaching context limits, user requested documentation update
**Current State**: All work completed and deployed successfully
**Next Work**: Investigate "dangerously-skip-permissions" flag per user request
