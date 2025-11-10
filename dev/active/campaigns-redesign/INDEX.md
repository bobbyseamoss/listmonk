# Campaigns Page Redesign - Documentation Index

**Project Status**: ✅ Complete
**Current Revision**: `listmonk420--0000097`
**Production URL**: https://list.bobbyseamoss.com
**Last Updated**: November 9, 2025, 10:15 AM EST

## Quick Start

**New to this project?** Start here:
1. Read [QUICK-REFERENCE.md](QUICK-REFERENCE.md) - One-page overview ⚡
2. Read [HANDOFF-NOV-9.md](HANDOFF-NOV-9.md) - Latest session summary
3. Check [TASKS.md](TASKS.md) - See what's done and what's next

**Continuing from a previous session?**
- Read [QUICK-REFERENCE.md](QUICK-REFERENCE.md) - One-page overview ⚡
- Or [HANDOFF-NOV-9.md](HANDOFF-NOV-9.md) - Latest state and quick reference

**Need to debug or understand the code?**
- Read [CONTEXT.md](CONTEXT.md) - Technical challenges and solutions
- Read [FILES-MODIFIED.md](FILES-MODIFIED.md) - Detailed code changes

## Documentation Files

### Overview Documents

**[QUICK-REFERENCE.md](QUICK-REFERENCE.md)** ⚡ **ONE-PAGE OVERVIEW**
- Single-page quick reference guide
- What was built, how to build/deploy, key patterns
- Common issues and solutions
- **Perfect for quick orientation**

**[README.md](README.md)**
- High-level project overview
- Feature list with screenshots
- Quick reference for what was built
- **Read this first** if new to the project

**[HANDOFF-NOV-9.md](HANDOFF-NOV-9.md)** ⭐ **LATEST SESSION**
- Latest session handoff notes (Nov 9, 2025)
- Current production state
- Quick reference for build/deploy commands
- Troubleshooting guide
- **Most useful for continuing work**

### Task Tracking

**[TASKS.md](TASKS.md)**
- Complete task breakdown with checkboxes
- 6 phases of development documented
- Summary statistics and time tracking
- Future enhancements backlog
- **Use this to track progress**

### Technical Documentation

**[CONTEXT.md](CONTEXT.md)**
- Detailed implementation context
- Technical challenges and solutions
- PostgreSQL patterns and fixes
- Vue reactivity patterns
- All 8 deployment iterations explained
- **Deep dive into how things work**

**[FILES-MODIFIED.md](FILES-MODIFIED.md)**
- Complete file-by-file change reference
- Code snippets for all modifications
- Backend and frontend changes
- Line numbers for easy navigation
- **Reference for understanding code changes**

**[SQL-LESSONS.md](SQL-LESSONS.md)**
- PostgreSQL patterns learned
- GROUP BY and aggregate best practices
- NULL handling techniques
- Performance optimization tips
- **Reference for database work**

### Phase-Specific Documentation

**[PHASE-6-PROGRESS-BAR.md](PHASE-6-PROGRESS-BAR.md)** (November 9, 2025)
- Progress bar implementation details
- ESLint bug fixes required
- Build and deployment process
- Testing notes
- **Most recent phase documentation**

### Session Summaries

**[SESSION-SUMMARY-NOV-9.md](SESSION-SUMMARY-NOV-9.md)** (November 9, 2025)
- Complete session activity log
- Problems solved and decisions made
- Metrics and timing
- Lessons learned
- **Historical record of work done**

## Documentation by Use Case

### I want to...

#### ...understand what was built
1. [README.md](README.md) - Feature overview
2. [TASKS.md](TASKS.md) - Complete task list

#### ...continue development
1. [HANDOFF-NOV-9.md](HANDOFF-NOV-9.md) - Current state and next steps
2. [TASKS.md](TASKS.md) - Future enhancements backlog
3. [CONTEXT.md](CONTEXT.md) - Technical patterns to follow

#### ...debug an issue
1. [CONTEXT.md](CONTEXT.md) - Known issues and solutions
2. [FILES-MODIFIED.md](FILES-MODIFIED.md) - Find affected code
3. [SQL-LESSONS.md](SQL-LESSONS.md) - Database troubleshooting
4. [HANDOFF-NOV-9.md](HANDOFF-NOV-9.md) - Troubleshooting section

#### ...understand a specific implementation
1. [PHASE-6-PROGRESS-BAR.md](PHASE-6-PROGRESS-BAR.md) - Progress bar (Nov 9)
2. [CONTEXT.md](CONTEXT.md) - Phases 1-5 detailed
3. [FILES-MODIFIED.md](FILES-MODIFIED.md) - Exact code changes

#### ...deploy changes
1. [HANDOFF-NOV-9.md](HANDOFF-NOV-9.md) - Build/deploy commands
2. [PHASE-6-PROGRESS-BAR.md](PHASE-6-PROGRESS-BAR.md) - Deployment process

#### ...learn from this project
1. [SQL-LESSONS.md](SQL-LESSONS.md) - PostgreSQL best practices
2. [CONTEXT.md](CONTEXT.md) - Technical challenges solved
3. [SESSION-SUMMARY-NOV-9.md](SESSION-SUMMARY-NOV-9.md) - Decisions and rationale

## Project Timeline

### November 8, 2025
**Phases 1-5**: Initial implementation and refinements
- 9:00 AM - 10:59 AM: Phase 1 - Performance summary and Placed Order column
- 11:15 AM - 11:14 AM: Phase 2 - Column reorganization
- 11:26 AM - 11:26 AM: Phase 3 - Reactivity fix
- 11:33 AM - 11:33 AM: Phase 4 - Queue campaign support
- 11:38 AM - 11:38 AM: Phase 5 - View/click counts

### November 9, 2025
**Phase 6**: Progress bar implementation
- 10:00 AM - 10:15 AM: Progress bar with email counter + ESLint fixes

**Total Development Time**: ~2 hours 52 minutes across 2 days

## Key Metrics

**Backend**:
- 6 files modified
- ~100 lines added
- 2 SQL queries created
- 3 Go structs defined
- 1 API endpoint added

**Frontend**:
- 5 files modified
- ~220 lines added
- 5 table columns redesigned
- 2 calculation methods (40 lines each)
- 3 helper methods
- 12 i18n translation keys
- 1 progress bar component

**Deployment**:
- 8 total revisions (2 failed, 6 successful)
- Production: https://list.bobbyseamoss.com
- Container: `listmonk420--0000097`

## Architecture Patterns

### Backend Patterns
- **SQL Queries**: Named queries in queries.sql loaded via goyesql
- **Data Models**: Go structs in models/models.go
- **API Endpoints**: HTTP handlers in cmd/ directory
- **Bulk Fetching**: Use `ANY($1)` for efficient bulk operations
- **NULL Safety**: Always COALESCE aggregates

### Frontend Patterns
- **Vue 2.7**: Composition-style components
- **Buefy**: UI component library
- **Reactivity**: Direct method calls in templates for polling data
- **Campaign Types**: Handle both queue-based and regular campaigns
- **Field Names**: Support both camelCase and snake_case

### Deployment Patterns
- **Docker**: Multi-stage builds with Alpine Linux
- **Azure**: Container Apps with zero-downtime deployments
- **Revisions**: Incremental deployments for quick iteration
- **Rollback**: Previous revisions maintained for safety

## Future Work

From [TASKS.md](TASKS.md) Future Enhancements:
1. Date range selector for performance summary
2. Loading states for performance summary fetch
3. Currency formatting based on actual campaign currency
4. Error handling UI for failed fetches
5. Summary data caching

## File Structure

```
dev/active/campaigns-redesign/
├── INDEX.md (this file)              # Navigation and overview
├── README.md                          # Project overview
├── HANDOFF-NOV-9.md                  # Latest session handoff ⭐
├── TASKS.md                          # Task tracking
├── CONTEXT.md                        # Technical deep dive
├── FILES-MODIFIED.md                 # Code changes reference
├── SQL-LESSONS.md                    # Database patterns
├── PHASE-6-PROGRESS-BAR.md          # Phase 6 details
└── SESSION-SUMMARY-NOV-9.md         # Nov 9 session summary
```

## Production Access

**URL**: https://list.bobbyseamoss.com
**Azure Resource Group**: rg-listmonk420
**Container App**: listmonk420
**Container Registry**: listmonk420acr.azurecr.io
**Database**: listmonk420-db.postgres.database.azure.com

## Quick Commands

```bash
# Build frontend
cd /home/adam/listmonk/frontend && yarn build

# Build distribution
cd /home/adam/listmonk && make dist

# Build and push Docker image
docker build -t listmonk420acr.azurecr.io/listmonk:latest .
az acr login --name listmonk420acr
docker push listmonk420acr.azurecr.io/listmonk:latest

# Deploy to Azure
az containerapp update \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --image listmonk420acr.azurecr.io/listmonk:latest

# Check deployment status
az containerapp revision list \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --query "[?properties.active==\`true\`]" \
  --output table
```

---

**Last Updated**: November 9, 2025, 10:15 AM EST
**Documentation Maintained By**: Claude Code
**Project Status**: ✅ Complete and Production-Ready
