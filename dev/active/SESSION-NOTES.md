# Session Notes - 2025-11-13

## Session Overview

This session (continuation from 2025-11-12) included work on:
1. **Logo URL Configuration** (COMPLETED ✅) - Use app.logo_url setting for navigation logo
2. Previous session: Dynamic error rate monitoring system (completed)
3. Previous session: Campaign progress bar bug fix (completed)
4. Previous session: SMTP HELO hostname configuration update (completed)

## Major Work Completed

### 0. Logo URL Configuration (COMPLETED ✅) - 2025-11-13 SESSION

**User Request**: Use the Logo URL setting from General Settings page for the navigation bar logo.

**Problem**: Navigation logo was using environment variable + hardcoded fallback instead of the `app.logo_url` setting.

**Solution**: Frontend-only change to App.vue
- Modified `logoUrl()` computed property to prioritize settings over environment
- Priority chain: Settings → Environment → Default

**Files Modified**:
- `frontend/src/App.vue` (lines 195, 213-219)
  - Added `settings` to mapState
  - Updated `logoUrl()` to check `this.settings['app.logo_url']` first

**Code Change**:
```javascript
// Before
computed: {
  ...mapState(['serverConfig', 'profile']),
  logoUrl() {
    return import.meta.env.VITE_LOGO_URL || 'https://d3k81ch9hvuctc.cloudfront.net/company/XFsBBP/images/6066d5d2-0701-4193-a8e7-13b624efc474.png';
  },
}

// After
computed: {
  ...mapState(['serverConfig', 'profile', 'settings']),
  logoUrl() {
    if (this.settings && this.settings['app.logo_url']) {
      return this.settings['app.logo_url'];
    }
    return import.meta.env.VITE_LOGO_URL || 'https://d3k81ch9hvuctc.cloudfront.net/company/XFsBBP/images/6066d5d2-0701-4193-a8e7-13b624efc474.png';
  },
}
```

**Build Status**:
- ✅ Frontend built successfully with `make build-frontend`
- Output in `frontend/dist/`
- Ready to deploy when user decides

**Additional Discovery**:
- Favicon already configured correctly for both admin and subscriber pages
- Admin: Uses `updateFavicon()` method which sets favicon to logo URL
- Subscriber pages: Backend reads `app.favicon_url` from settings
- No additional work needed for favicon

**Technical Notes**:
- No backend changes required
- Settings already available in Vuex store
- Frontend asset needs to be embedded in binary and deployed for changes to take effect

### 1. Campaign Progress Bar Fix (COMPLETED ✅) - 2025-11-12 SESSION

**User Request**: Fix progress bar showing wrong count (21,934 Azure delivered instead of 44,173 queue sent) for paused queue-based campaigns.

**Problem**: Paused queue-based campaigns displayed Azure webhook delivered count instead of actual queue sent count in progress bar.

**Root Causes Found**:
1. SQL query `get-campaign-queue-stats` only fetched `status = 'running'` campaigns
2. Go function `GetRunningCampaignStats()` only fetched running campaigns, so paused campaign stats had nothing to merge into

**Files Modified**:
- `queries.sql` (lines 730, 735, 740, 765) - Changed to `status IN ('running', 'paused')`
- `internal/core/campaigns.go` (lines 400-443) - Fetch both running AND paused campaigns

**Deployment**:
- First attempt: listmonk420--deploy-20251112-064547 (SQL fix only - incomplete)
- Second deployment: listmonk420--deploy-20251112-065523 (SQL + Go fixes - working)
- Status: ✅ User verified fix is working
- Campaign 65 now shows 44,173 / 78,249 (correct) instead of 21,934 / 78,249 (wrong)

**Technical Decision**: Modified `GetRunningCampaignStats()` to fetch paused campaigns separately and append to running campaigns before merging queue stats. This ensures paused campaigns have a base record to merge queue stats into.

**Documentation Created**:
- `/dev/active/campaign-progress-bar-fix/task-context.md` - Complete analysis and implementation
- `/dev/active/campaign-progress-bar-fix/tasks.md` - Task checklist and timeline

### 2. SMTP HELO Hostname Update (COMPLETED ✅) - THIS SESSION

**User Request**: Update HELO hostname for all SMTP servers (mail2-mail30) to match their respective domain names.

**Problem**: Only mail2 had hello_hostname configured; mail3-mail30 had empty values.

**Solution**: Direct database update approach
- Retrieved SMTP settings JSON from database
- Updated hello_hostname field for 28 servers (mail3-mail30)
- mail2 already had correct value (mail2.bobbyseamoss.com)
- Saved back to database and restarted container

**Method**:
- Python script to parse/update JSON
- Direct UPDATE to settings table
- Container restart to reload configuration

**Result**: ✅ All 29 SMTP servers now have correct HELO hostnames
- mail2.bobbyseamoss.com through mail30.bobbyseamoss.com
- Verified in database and container logs
- All messengers initialized successfully

**Impact**: Improved SMTP compliance and sender reputation per domain

**Documentation Created**:
- `/dev/active/smtp-helo-hostname-update/task-context.md` - Complete implementation details

### 3. Error Rate Display Feature (COMPLETED ✅) - EARLIER TODAY

**User Request**: Add real-time error rate monitoring to Campaigns page with dropdown selector for different timeframes.

**Implementation Details**:
- Added dropdown selector: Today (default), 7 Days, 14 Days, 30 Days
- Replaced "Revenue Per Recipient" with "Error Rate" displayed in red
- Error rate calculates from Azure delivery webhook failures
- All metrics update dynamically when timeframe changes

**Files Modified**:
- `models/models.go` (lines 430-439) - Added ErrorRate field, changed to camelCase JSON
- `queries.sql` (lines 1707-1771) - Added days parameter, error_stats CTE
- `cmd/shopify.go` (lines 271-305) - Handler accepts days parameter
- `frontend/src/api/index.js` (lines 368-372) - API passes days parameter
- `frontend/src/views/Campaigns.vue` (lines 24-60, 297-330, 559-566) - Dropdown, watcher, method

**Deployed**: Bobby Seamoss at https://list.bobbyseamoss.com
- Revision: `listmonk420--deploy-20251112-062554`
- Status: ✅ Active and running
- All 30 SMTP servers initialized successfully

**Error Rate Calculation**:
```sql
error_rate = (COUNT of Failed azure_delivery_events / total sent) * 100
```

**Key Technical Decisions**:
1. Days parameter as integer (1, 7, 14, 30) for simple SQL construction
2. Calculate across ALL campaigns in timeframe (not per-campaign)
3. Use camelCase for all JSON field names for frontend consistency
4. Default to "Today" (1 day) for most actionable/recent data
5. Vue watcher for automatic refresh on selection change

**Documentation Created**:
- `/dev/active/error-rate-display/task-context.md` - Complete implementation context
- `/dev/active/error-rate-display/tasks.md` - Task checklist and troubleshooting

### 2. Previous Session Work (From Earlier Today)

**Database Error Fix** (`database-error-fix-deployment.md`):
- Fixed PostgreSQL array parameter error in queue processor
- Issue: `sql: converting argument $1 type: unsupported type []int`
- Solution: Wrapped with `pq.Array(runningCampaignIDs)`
- Deployed to both Bobby Seamoss and Comma

**Bobby Seamoss Failure Analysis** (`bobby_seamoss_failure_analysis.md`):
- Investigated 14,163 failed deliveries (13% failure rate)
- Root cause: Gmail domain reputation blocking
- All 29 Bobby Seamoss domains affected equally
- Created comprehensive remediation strategy

### 3. Campaign Sent Count Fix (From Previous Session 2025-11-11)

**Problem**: Queue-based campaigns showed 0 sent despite actually sending emails

**Solution**: Dual-sync approach
1. Sync before auto-pause (prevents count loss)
2. Periodic sync every 5 minutes (maintains accuracy)

**Files Modified**:
- `queries.sql` - Added sync-queue-campaign-counts query
- `internal/queue/processor.go` - Added sync functions
- `cmd/init.go` - Start stats sync goroutine

## Technical Patterns & Discoveries

### PostgreSQL Dynamic INTERVAL Construction
```sql
-- Using parameter in INTERVAL
WHERE sent_at >= NOW() - ($1::TEXT || ' days')::INTERVAL
```

### Vue.js Reactive Dropdown Pattern
```javascript
// Data
selectedTimeframe: 1,
timeframeOptions: [
  { label: 'Today', value: 1 },
  { label: '7 Days', value: 7 },
  // ...
],

// Watcher
watch: {
  selectedTimeframe() {
    this.getPerformanceSummary();
  },
}

// Template
<b-select v-model="selectedTimeframe">
  <option v-for="opt in timeframeOptions" :key="opt.value" :value="opt.value">
    {{ opt.label }}
  </option>
</b-select>
```

### Error Rate with NULL Handling
```sql
-- Safe division with COALESCE and NULLIF
CASE
  WHEN COALESCE(SUM(sent), 0) > 0
  THEN ((SELECT COALESCE(MAX(total_errors), 0) FROM error_stats)::FLOAT / SUM(sent)::FLOAT) * 100
  ELSE 0
END AS error_rate
```

### Go Backend Query Parameter Parsing
```go
// Parse with default value
days := 1
if daysParam := c.QueryParam("days"); daysParam != "" {
    parsedDays, err := strconv.Atoi(daysParam)
    if err != nil || parsedDays < 1 {
        return echo.NewHTTPError(http.StatusBadRequest, "Invalid 'days' parameter")
    }
    days = parsedDays
}
```

## Deployment Process

### Standard Deployment to Bobby Seamoss
```bash
# Set database password and deploy
export LISTMONK_DB_PASSWORD='T@intshr3dd3r' && ./deploy.sh
```

**Steps Performed**:
1. Build email-builder component
2. Build frontend with Vite
3. Build Go binary
4. Build Docker image with multi-stage build
5. Push to Azure Container Registry (listmonk420acr.azurecr.io)
6. Run database migrations (if LISTMONK_DB_PASSWORD set)
7. Deploy to Azure Container Apps with unique revision suffix
8. Force new image pull and container restart

**Verification**:
```bash
# Check revision
az containerapp show \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --query "properties.latestRevisionName" -o tsv

# View logs
az containerapp logs show \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --tail 50

# Test API
curl 'https://list.bobbyseamoss.com/api/campaigns/performance/summary?days=1'
```

## Credentials Reference

**Bobby Seamoss Production**:
- URL: https://list.bobbyseamoss.com
- Admin Username: adam
- Admin Password: T@intshr3dd3r
- Database:
  - Host: listmonk420-db.postgres.database.azure.com
  - Port: 5432
  - User: listmonkadmin
  - Password: T@intshr3dd3r
  - Database: listmonk
  - SSL Mode: require

**Azure Resources**:
- Subscription: a4b642f1-79bc-4f13-a6e5-c64ae683d6a1
- Resource Group: rg-listmonk420
- Container App: listmonk420
- Container Registry: listmonk420acr.azurecr.io
- Managed Environment: listmonk420-env
- Current Revision: listmonk420--deploy-20251112-062554
- Previous Stable: listmonk420--fix-20251111-164605

**Comma Environment** (for reference):
- Similar structure with "comma" naming
- Resource Group: comma-rg
- Container App: comma
- 30 domains: mail1-mail30.enjoycomma.com

## Monitoring & Verification

### Error Rate API Testing
```bash
# Test with different timeframes
curl 'https://list.bobbyseamoss.com/api/campaigns/performance/summary?days=1'
curl 'https://list.bobbyseamoss.com/api/campaigns/performance/summary?days=7'
curl 'https://list.bobbyseamoss.com/api/campaigns/performance/summary?days=30'

# Should return JSON with errorRate field
{
  "data": {
    "avgOpenRate": 12.34,
    "avgClickRate": 5.67,
    "totalSent": 100000,
    "totalOrders": 50,
    "totalRevenue": 1234.56,
    "orderRate": 0.05,
    "errorRate": 13.2  // ← New field
  }
}
```

### Database Queries for Error Rate
```sql
-- Check error rate data directly
WITH error_stats AS (
    SELECT COUNT(*) AS total_errors
    FROM azure_delivery_events
    WHERE event_timestamp >= NOW() - '1 days'::INTERVAL
      AND status = 'Failed'
),
recent_campaigns AS (
    SELECT id, sent
    FROM campaigns
    WHERE sent_at >= NOW() - '1 days'::INTERVAL
)
SELECT
    COALESCE(SUM(sent), 0) as total_sent,
    (SELECT COALESCE(MAX(total_errors), 0) FROM error_stats) as total_errors,
    CASE
        WHEN COALESCE(SUM(sent), 0) > 0
        THEN ((SELECT COALESCE(MAX(total_errors), 0) FROM error_stats)::FLOAT / SUM(sent)::FLOAT) * 100
        ELSE 0
    END AS error_rate
FROM recent_campaigns;
```

### Container Health Check
```bash
# Check container logs for errors
az containerapp logs show \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --follow

# Check specific log patterns
az containerapp logs show \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --tail 100 | grep -i "error\|failed\|panic"

# Check SMTP initialization
az containerapp logs show \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --tail 100 | grep "initialized.*messenger"
```

## Known Issues

### Build Warnings (Non-blocking)
1. **Sass Deprecation Warnings**: Bulma/Buefy dependencies use deprecated Sass features. Not blocking, will need Sass 3.0 migration eventually.
2. **Large Chunk Size**: Campaign.js is 1.67MB (includes TinyMCE). Consider code-splitting in future.
3. **ESLint Warnings**: Queue.vue has style warnings. Pre-existing, not introduced by changes.

### Migration Check During Deployment
- Migration runner shows connection error: `dial tcp [::1]:5432: connect: connection refused`
- Expected behavior - tries to connect to localhost instead of Azure DB
- Migrations are idempotent, safe to skip if database already up to date
- Production migrations run automatically on container startup via docker-entrypoint.sh

## Architectural Patterns Used

### Multi-Stage Docker Build
```dockerfile
Stage 1: email-builder (Node 20 Alpine) → Build email-builder
Stage 2: frontend-builder (Node 20 Alpine) → Build Vue frontend
Stage 3: backend-builder (Go 1.24 Alpine) → Compile Go binary
Stage 4: final (Alpine) → Minimal runtime image
```

### Vue Component Reactivity
- Use `data()` for state
- Use `watch` for side effects on state changes
- Use `methods` for reusable logic
- Use `computed` for derived values (not used in this feature)

### SQL Query Parameterization
- Named queries in `queries.sql` with `-- name: query-name` markers
- Loaded via goyesql into `app.queries` struct
- Parameters passed via positional placeholders: `$1`, `$2`, etc.
- Type casting in SQL: `$1::INT`, `$1::TEXT`

### Echo Handler Pattern
```go
func (app *App) HandlerName(c echo.Context) error {
    // 1. Parse parameters
    param := c.QueryParam("param")

    // 2. Query database
    var result Model
    err := app.queries.QueryName.Get(&result, param)

    // 3. Handle errors
    if err != nil {
        if err == sql.ErrNoRows {
            // Return zero value
        } else {
            return echo.NewHTTPError(http.StatusInternalServerError, msg)
        }
    }

    // 4. Return JSON
    return c.JSON(http.StatusOK, okResp{result})
}
```

## Related Work & Integration Points

### Azure Delivery Events
- Error rate depends on `azure_delivery_events` table being populated
- Webhooks received at `/webhooks/service/azure`
- Handler: `cmd/azure.go:AzureWebhook()`
- Events stored with status: Delivered, Failed, Bounced, Spam, etc.

### Campaign Queue System
- Queue-based campaigns use `email_queue` table
- Processor syncs sent counts to campaigns table
- Error rate can track both queue-based and traditional campaigns

### Multi-SMTP Architecture
- 30 SMTP servers for Bobby Seamoss (mail2-mail30.bobbyseamoss.com)
- Each has independent messenger: `email-email-mail1`, etc.
- Error rate aggregates across all SMTP servers

## Future Enhancement Ideas

### Immediate Next Steps (User May Request)
1. **Per-Campaign Error Rates**: Add error rate column to campaigns table
2. **Error Breakdown**: Show breakdown by error type (bounce, spam, invalid)
3. **Server-Specific Rates**: Show error rate per SMTP server
4. **Alerting**: Notify when error rate exceeds threshold

### Medium-Term Enhancements
1. **Trend Visualization**: Chart.js graph of error rate over time
2. **Industry Benchmarks**: Compare to average rates
3. **Auto-Remediation**: Pause campaigns or servers with high error rates
4. **Detailed Error Logs**: Link to specific failed deliveries

### Long-Term Features
1. **Predictive Analytics**: ML model to predict error rate trends
2. **A/B Testing**: Compare error rates between subject lines, content, etc.
3. **Reputation Monitoring**: Track sender reputation scores
4. **Smart Routing**: Route emails through servers with best delivery rates

## Git Status

**Modified Files (Not Committed from multiple features)**:
- `models/models.go` (error rate feature)
- `queries.sql` (error rate feature + progress bar fix)
- `cmd/shopify.go` (error rate feature)
- `frontend/src/api/index.js` (error rate feature)
- `frontend/src/views/Campaigns.vue` (error rate feature)
- `internal/core/campaigns.go` (progress bar fix)
- `frontend/src/App.vue` (logo URL feature)

**Recommended Commits** (create three separate commits):

**Commit 1: Logo URL Feature**
```bash
git add frontend/src/App.vue

git commit -m "Use Logo URL setting for navigation logo

- Modified App.vue logoUrl() computed property to prioritize settings
- Priority chain: Settings (app.logo_url) → Environment → Default
- Added settings to Vuex mapState
- Frontend-only change, no backend modifications needed

Built but not deployed yet (frontend assets need embedding + deployment)"
```

**Commit 2: Progress Bar Fix** (PRIORITY - bug fix)
```bash
git add queries.sql internal/core/campaigns.go

git commit -m "Fix progress bar for paused queue-based campaigns

- Modified GetRunningCampaignStats to fetch both running AND paused campaigns
- Updated get-campaign-queue-stats SQL query to include paused status
- Previously showed Azure delivered count (21934) for paused campaigns
- Now correctly shows queue sent count (44173) for paused campaigns

Two-part fix required:
1. SQL: Changed status = 'running' to status IN ('running', 'paused')
2. Go: Fetch paused campaigns and append before merging queue stats

Deployed to Bobby Seamoss: listmonk420--deploy-20251112-065523"
```

**Commit 3: Error Rate Feature**
```bash
git add models/models.go queries.sql cmd/shopify.go \
  frontend/src/api/index.js frontend/src/views/Campaigns.vue

git commit -m "Add dynamic error rate display to Campaigns page with timeframe selector

- Add timeframe dropdown (Today, 7 Days, 14 Days, 30 Days) defaulting to Today
- Replace Revenue Per Recipient with Error Rate displayed in red
- Update backend to calculate error rate from Azure delivery events
- Add days parameter to performance summary API endpoint
- Error rate calculated as (failed deliveries / total sent) * 100
- All metrics update dynamically when timeframe changes

Deployed to Bobby Seamoss: listmonk420--deploy-20251112-062554"
```

## Background Tasks

### Comma Engagement Tracking (Running)
There's a background bash process enabling engagement tracking for all 30 Comma domains:
```bash
# Process ID: fe4e6e
# Command: Loop through mail1-mail30.enjoycomma.com
# Action: Enable user engagement tracking via Azure CLI
# Status: Running in background
```

This is unrelated to the error rate feature but running concurrently.

## Context at Session End

**Context Usage**: ~120K/200K tokens used (60%)
**Reason for /dev-docs-update**: User manually triggered to document recent work
**Current State**:
- ✅ Logo URL feature completed and built (not deployed yet)
- ✅ Error rate feature fully completed and deployed
- ✅ Progress bar fix fully completed, deployed, and user-verified working
- ✅ SMTP HELO hostname update completed and verified
**Outstanding Work**: Logo URL feature needs deployment (frontend built, needs binary embedding + Azure deploy)

**Next Session Possibilities**:
1. User may want to deploy Logo URL changes (requires full deployment: make dist + deploy.sh)
2. User may want to commit changes to git (three separate commits recommended)
3. User may request additional metrics or visualizations
4. User may report issues with any feature
5. User may want similar features for Comma deployment
6. Unrelated new feature requests

## Documentation Created This Session

1. `/dev/active/error-rate-display/task-context.md`
   - Complete implementation details
   - Technical decisions documented
   - Code changes with line numbers
   - Deployment information
   - Rollback procedures
   - Testing commands

2. `/dev/active/error-rate-display/tasks.md`
   - All tasks marked complete
   - Future enhancement ideas
   - Troubleshooting guide
   - Quick fixes reference
   - API testing examples

3. `/dev/active/campaign-progress-bar-fix/task-context.md` **(NEW THIS SESSION)**
   - Complete root cause analysis (SQL + Go bugs)
   - Implementation details with code snippets
   - Deployment history (two attempts)
   - Verification methodology
   - Technical decisions explained
   - Testing challenges documented
   - Rollback procedures

4. `/dev/active/campaign-progress-bar-fix/tasks.md` **(NEW THIS SESSION)**
   - Complete task timeline
   - Root causes fixed
   - Files modified
   - Verification methods
   - Commit recommendations

5. `/dev/active/smtp-helo-hostname-update/task-context.md` **(NEW THIS SESSION)**
   - Configuration-only change (no code)
   - Database update approach
   - Verification in logs and database
   - Impact on deliverability
   - Rollback procedures
   - Future considerations for Comma

6. Updated `/dev/active/SESSION-NOTES.md` (this file)
   - Session overview
   - Technical patterns
   - Deployment process
   - Credentials reference
   - Monitoring commands

## Key Takeaways

### What Went Well
- Clear user requirements led to straightforward implementation
- No database migrations needed (all changes to queries)
- Frontend changes isolated to single component
- Deployment smooth with no errors
- All 30 SMTP servers initialized successfully

### Technical Highlights
- Dynamic SQL INTERVAL construction with parameters
- Vue watcher pattern for reactive updates
- Proper NULL handling in SQL aggregation
- camelCase JSON naming for frontend consistency

### Lessons Learned
- Migration runner in deployment script expects localhost DB
- This is expected - production migrations happen in container
- Always use COALESCE for aggregates to avoid NULL issues
- Vue watchers great for dependent state updates

## References

- Error Rate Feature: `/dev/active/error-rate-display/`
- Deployment Guide: `.claude/skills/migration-deployment-guidelines/SKILL.md`
- Project Overview: `CLAUDE.md`
- Bobby Seamoss Analysis: `/dev/active/bobby_seamoss_failure_analysis.md`
- Database Fix: `/dev/active/database-error-fix-deployment.md`
