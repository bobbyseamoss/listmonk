# Campaigns Page Redesign - Quick Reference

**Last Updated**: November 9, 2025, 10:15 AM EST
**Status**: ✅ Complete and Deployed
**Revision**: `listmonk420--0000097`
**Production**: https://list.bobbyseamoss.com

## What Was Built

### Features Implemented
1. ✅ Email performance summary (30-day aggregates)
2. ✅ Placed Order column with revenue/counts
3. ✅ Start Date, Open Rate, Click Rate columns
4. ✅ Real-time stats updates for running campaigns
5. ✅ Queue-based campaign support
6. ✅ View/click counts display
7. ✅ **Campaign progress bar with email counter** (Nov 9)

### What It Looks Like
- **Performance Summary**: Collapsible box at top showing avg metrics
- **Placed Order Column**: Shows "$108.00 / 2 recipients"
- **Open Rate Column**: Shows "5.23%" with "15 views" below
- **Click Rate Column**: Shows "1.50%" with "3 clicks" below
- **Progress Bar**: Blue/yellow bar under campaign name with "150 / 1000"

## Quick Navigation

### New to this project?
1. Start: [INDEX.md](INDEX.md)
2. Then: [HANDOFF-NOV-9.md](HANDOFF-NOV-9.md)
3. Details: [CONTEXT.md](CONTEXT.md)

### Need to continue development?
- [HANDOFF-NOV-9.md](HANDOFF-NOV-9.md) - Current state
- [TASKS.md](TASKS.md) - Future enhancements backlog

### Need to debug?
- [CONTEXT.md](CONTEXT.md) - Technical solutions
- [SQL-LESSONS.md](SQL-LESSONS.md) - Database patterns

## Build & Deploy

```bash
# Frontend build
cd /home/adam/listmonk/frontend && yarn build

# Distribution build
make dist

# Docker build and push
docker build -t listmonk420acr.azurecr.io/listmonk:latest .
az acr login --name listmonk420acr
docker push listmonk420acr.azurecr.io/listmonk:latest

# Deploy to Azure
az containerapp update \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --image listmonk420acr.azurecr.io/listmonk:latest
```

## Key Technical Patterns

### PostgreSQL
- Always use `COALESCE(MAX(...), 0)` for aggregates
- Use `ANY($1)` for bulk operations
- NULL handling is critical

### Vue 2.7
- Direct method calls in templates for polling data
- Computed properties with getters/setters for props
- Support both camelCase and snake_case field names

### Campaign Types
- Queue: Use `queue_sent`, `queue_total`, `useQueue`
- Regular: Use `sent`, `toSend`
- Check: `stats.use_queue || stats.useQueue`

## Files Modified

### Backend (6 files)
- queries.sql (2 queries)
- models/models.go (3 structs)
- models/queries.go (2 registrations)
- cmd/shopify.go (1 endpoint)
- cmd/campaigns.go (bulk fetch)
- cmd/handlers.go (route)

### Frontend (5 files)
- frontend/src/api/index.js
- frontend/src/views/Campaigns.vue (main changes)
- frontend/src/views/settings/shopify.vue (bug fix)
- i18n/en.json (12 keys)

## Common Issues

### Progress bar not showing
- Campaign must have sent > 0 emails
- Campaign must not be "done"
- Check `getCampaignStats()` returns data

### ESLint errors
- Use `Number.isNaN()` not `isNaN()`
- Use computed properties for prop mutations

### NULL errors
- Always COALESCE aggregates
- Test with empty datasets

## Future Enhancements

1. Date range selector for performance summary
2. Loading states for performance summary fetch
3. Currency formatting based on actual campaign currency
4. Error handling UI for failed fetches
5. Summary data caching

## Production Info

**URL**: https://list.bobbyseamoss.com
**Resource Group**: rg-listmonk420
**Container App**: listmonk420
**Container Registry**: listmonk420acr.azurecr.io
**Database**: listmonk420-db.postgres.database.azure.com

## Documentation Files

- **INDEX.md** - Documentation navigation ⭐
- **HANDOFF-NOV-9.md** - Latest session handoff
- **TASKS.md** - Task tracking
- **CONTEXT.md** - Technical deep dive
- **FILES-MODIFIED.md** - Code changes
- **SQL-LESSONS.md** - Database patterns
- **PHASE-6-PROGRESS-BAR.md** - Phase 6 details
- **SESSION-SUMMARY-NOV-9.md** - Nov 9 summary
- **QUICK-REFERENCE.md** - This file

## Metrics

**Development Time**: ~2 hours 52 minutes (across 2 days)
**Deployments**: 8 revisions (2 failed, 6 successful)
**Lines Added**: ~320 lines total
**Files Modified**: 11 files

---

**Last Session**: November 9, 2025, 10:00-10:15 AM
**Next Steps**: See TASKS.md Future Enhancements
**Status**: Production-ready and stable ✅
