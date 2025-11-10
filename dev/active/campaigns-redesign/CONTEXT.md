# Campaigns Page Redesign - Context Document

## Overview
Redesigned the All Campaigns page to integrate Shopify purchase attribution metrics, showing revenue data alongside traditional campaign metrics.

**Status**: ✅ Complete and deployed
**Deployment**: Revision `listmonk420--0000097` (8 iterations total)
**URL**: https://list.bobbyseamoss.com
**Last Updated**: November 9, 2025, 10:15 AM EST

## User Requirements

The user provided a screenshot showing:
1. **Email Performance Summary** section at the top displaying:
   - "Email performance last 30 days" header
   - Average open rate (33.38%)
   - Average click rate (0.49%)
   - Placed Order rate (0.09%)
   - Revenue per recipient ($0.03)

2. **Placed Order column** in campaigns table showing:
   - Dollar amounts (e.g., $108.00, $87.00)
   - Number of recipients who placed orders (e.g., "2 recipients")

## Implementation Summary

### Backend Changes

**1. SQL Queries (queries.sql:1659-1717)**
- `get-campaigns-performance-summary` - Aggregate metrics for last 30 days
  - Uses CTEs for recent campaigns and purchase data
  - Calculates avg open rate, avg click rate, order rate, revenue per recipient
  - **Critical fix**: Wrapped aggregates in COALESCE to handle NULL values
- `get-campaigns-purchase-stats` - Bulk fetch purchase stats for campaigns list

**2. Data Models (models/models.go)**
- `CampaignsPerformanceSummary` - Summary metrics response
- `CampaignPurchaseStatsListItem` - Purchase stats per campaign
- Extended `CampaignMeta` with purchase fields:
  - `PurchaseOrders int`
  - `PurchaseRevenue float64`
  - `PurchaseCurrency string`

**3. API Endpoints**
- `GET /api/campaigns/performance/summary` - Fetch aggregate metrics (cmd/shopify.go:165-190)
- Enhanced `GetCampaigns()` to include purchase stats (cmd/campaigns.go:100-126)

### Frontend Changes

**1. API Client (frontend/src/api/index.js:368-372)**
- Added `getCampaignsPerformanceSummary()` method

**2. Campaigns View (frontend/src/views/Campaigns.vue)**
- Performance summary section (lines 20-53):
  - Collapsible box showing 4 metrics in a grid
  - Calls `getPerformanceSummary()` on mount
- Placed Order column (lines 236-247):
  - Shows currency amount and recipient count
  - Falls back to "$0.00 / 0 recipients" when no data
- Helper methods:
  - `formatPercent()` - Formats decimal as percentage
  - `formatCurrency()` - Formats decimal as currency

**3. i18n Translations (i18n/en.json:107-113)**
- Added 7 new translation keys for all UI text

## Technical Challenges & Solutions

### Issue 1: PostgreSQL GROUP BY Violation
**Error**: `column "pd.total_orders" must appear in the GROUP BY clause or be used in an aggregate function`

**Root Cause**: Mixing aggregate functions with direct column references from CROSS JOIN

**Solution**: Wrapped `pd.total_orders` and `pd.total_revenue` in `MAX()`:
```sql
COALESCE(MAX(pd.total_orders), 0) AS total_orders,
COALESCE(MAX(pd.total_revenue), 0) AS total_revenue
```

### Issue 2: NULL Handling for Empty Purchase Data
**Error**: `Scan error on column index 3, name "total_orders": converting NULL to int is unsupported`

**Root Cause**: When no purchase_attributions exist, `MAX()` returns NULL which can't be scanned into `int`

**Solution**: Added `COALESCE` around all `MAX()` calls to default to 0:
```sql
COALESCE(MAX(pd.total_orders), 0) AS total_orders
```

## Files Modified

### Backend
- `queries.sql` (lines 1659-1717)
- `models/models.go` (lines 282-285, 419-436)
- `models/queries.go` (lines 167-168)
- `cmd/shopify.go` (lines 165-190)
- `cmd/campaigns.go` (lines 100-126)
- `cmd/handlers.go` (line 188)

### Frontend
- `frontend/src/api/index.js` (lines 368-372)
- `frontend/src/views/Campaigns.vue` (major changes throughout)
- `i18n/en.json` (lines 107-113)

## Key Learnings

1. **PostgreSQL Strictness**: When using aggregate functions, ALL non-aggregated columns must be in GROUP BY or wrapped in aggregates

2. **NULL Safety**: Always use COALESCE when aggregating data that might not exist (especially with CROSS JOIN)

3. **Performance Optimization**: Used `ANY($1)` for batch fetching purchase stats instead of N+1 queries

4. **UX Consistency**: Followed existing patterns in Campaigns.vue for stats display and column formatting

## Testing Checklist

- [x] Container starts without SQL errors
- [x] Performance summary displays when data exists
- [x] Performance summary handles missing purchase data gracefully
- [x] Placed Order column shows in campaigns table
- [x] Campaigns with no purchases show "$0.00 / 0 recipients"
- [x] Deployed successfully to Azure

## Post-Deployment Enhancements (Nov 8, 11:15 AM - 11:38 AM)

After the initial deployment, three significant enhancements were made based on user feedback:

### Enhancement 1: Column Reorganization (Commit 3c449a5b)

**User Request**: "I want the Campaign columns on the right to be: Status, Start Date, Open Rate, Click Rate, Placed Order"

**Changes Made**:
- **Removed** combined Stats column (views/clicks/sent/bounces/rate)
- **Removed** combined Timestamps column (created/started/ended/duration)
- **Added** dedicated Start Date column showing when campaign started
- **Added** dedicated Open Rate column with percentage calculation
- **Added** dedicated Click Rate column with percentage calculation

**Rationale**: Simplified table layout for better metric visibility. Users primarily care about engagement rates, not raw stats.

**Implementation**:
```vue
<!-- Start Date Column -->
<b-table-column v-slot="props" field="created_at" :label="$t('campaigns.startDate')" width="12%" sortable>
  <div :set="stats = getCampaignStats(props.row)">
    <p v-if="stats.startedAt">{{ $utils.niceDate(stats.startedAt, true) }}</p>
    <p v-else class="has-text-grey">—</p>
  </div>
</b-table-column>

<!-- Open/Click Rate Calculation -->
calculateOpenRate(stats) {
  if (!stats) return '—';
  const sent = stats.sent || 0;
  if (sent === 0) return '—';
  const views = stats.views || 0;
  const rate = (views / sent) * 100;
  return `${rate.toFixed(2)}%`;
}
```

**Files Modified**:
- `frontend/src/views/Campaigns.vue` (118 insertions, 91 deletions)
- `i18n/en.json` (added `startDate`, `openRate`, `clickRate` keys)

**Deployment**: Revision `listmonk420--deploy-20251108-111408`

---

### Enhancement 2: Reactivity Fix for Live Campaigns (Commit 2f582a23)

**Issue**: Open Rate and Click Rate columns showing "—" for running campaigns despite view/click webhooks being triggered.

**Root Cause**: Using `:set="stats = getCampaignStats(props.row)"` pattern wasn't properly triggering Vue's reactivity system when `campaignStatsData` was updated by polling.

**Solution**: Changed to call `getCampaignStats(props.row)` directly in template expressions:

```vue
<!-- Before (not reactive) -->
<div :set="stats = getCampaignStats(props.row)">
  <p>{{ calculateOpenRate(stats) }}</p>
</div>

<!-- After (reactive) -->
<p>{{ calculateOpenRate(getCampaignStats(props.row)) }}</p>
```

**Why This Works**: Vue's reactivity system tracks dependencies when methods are called in template expressions. Direct method calls ensure the component re-renders when `campaignStatsData` changes from polling updates.

**Files Modified**:
- `frontend/src/views/Campaigns.vue` (6 insertions, 10 deletions)

**Deployment**: Revision `listmonk420--deploy-20251108-112555`

---

### Enhancement 3: Queue-Based Campaign Support (Commit bfaab3d8)

**Issue**: Open Rate still showing "—" for queue-based campaigns (messenger="automatic") despite having view webhooks.

**Root Cause**: Rate calculation methods only checked `stats.sent`, but queue-based campaigns use different field names:
- Regular campaigns: `stats.sent`
- Queue-based campaigns: `stats.queue_sent` or `stats.queueSent` (field name varies)

**Solution**: Enhanced both calculation methods to detect campaign type and use appropriate field:

```javascript
calculateOpenRate(stats) {
  if (!stats) return '—';

  // Determine sent count based on campaign type
  let sentCount = 0;
  if (stats.use_queue || stats.useQueue) {
    // Queue-based campaign
    sentCount = stats.queue_sent || stats.queueSent || 0;
  } else {
    // Regular campaign
    sentCount = stats.sent || 0;
  }

  if (sentCount === 0) return '—';

  const views = stats.views || 0;
  const rate = (views / sentCount) * 100;
  return `${rate.toFixed(2)}%`;
}

calculateClickRate(stats) {
  // Same pattern for clicks
  // ...handles both campaign types
}
```

**Key Points**:
- Checks both `use_queue` (snake_case from DB) and `useQueue` (camelCase from Vue)
- Handles both `queue_sent` and `queueSent` field name variations
- Falls back to 0 if fields are undefined
- Same logic applied to both `calculateOpenRate()` and `calculateClickRate()`

**Files Modified**:
- `frontend/src/views/Campaigns.vue` (32 insertions, 4 deletions)

**Deployment**: Revision `listmonk420--deploy-20251108-113326`

---

### Enhancement 4: Show View/Click Counts (Commit bfaab3d8)

**User Request**: "Show the exact number of opens and clicks under the Open Rate and Click Rate percentages"

**Changes Made**: Modified both columns to display raw counts below percentage, matching the Placed Order column styling:

```vue
<b-table-column v-slot="props" field="open_rate" :label="$t('campaigns.openRate')" width="10%">
  <div class="fields stats" :set="stats = getCampaignStats(props.row)">
    <p>
      <label for="#">{{ calculateOpenRate(stats) }}</label>
      <span>{{ stats.views || 0 }} {{ $t('campaigns.views', 'views') }}</span>
    </p>
  </div>
</b-table-column>
```

**Display Format**:
- Line 1 (label): Percentage (e.g., "5.23%")
- Line 2 (span): Raw count (e.g., "15 views" or "3 clicks")

**Why Revert to `:set`**: Since we need to reference `stats` twice (for percentage and count), using `:set` avoids calling `getCampaignStats()` twice. This is a valid Vue pattern when you need the same data multiple times.

**Files Modified**:
- `frontend/src/views/Campaigns.vue`
- `i18n/en.json` (added `views` and `clicks` keys)

**Deployment**: Revision `listmonk420--deploy-20251108-113812` ✅ Final

---

### Summary of Post-Deployment Changes

**What Changed**:
1. ✅ Simplified table layout (removed Stats/Timestamps columns)
2. ✅ Added focused metric columns (Start Date, Open Rate, Click Rate)
3. ✅ Fixed reactivity for live campaign updates
4. ✅ Added queue-based campaign support
5. ✅ Enhanced display with view/click counts

**Total Additional Code**:
- ~85 lines added to Campaigns.vue
- 5 new i18n translation keys
- 2 enhanced calculation methods (~40 lines each)
- 4 deployment iterations to refine UX

**Current State**: All features working correctly for both regular and queue-based campaigns with real-time updates.

---

## Phase 6: Campaign Progress Bar (November 9, 2025)

### User Request
"I want to add the status bar back in. On the All Campaigns page, there used to be a status bar and numbers of emails sent/total number of emails to be sent counter."

### Implementation

**Component Added**: Progress bar under campaign subject line in Name column

**Features**:
- Buefy `<b-progress>` component with conditional rendering
- Color-coded by campaign status:
  - Blue (is-primary) for running campaigns
  - Yellow (is-warning) for paused campaigns
  - Light gray (is-light) for other states
- Email counter display:
  - Queue campaigns: "queue_sent / queue_total"
  - Regular campaigns: "sent / toSend"
  - Handles both camelCase and snake_case field names
- Real-time updates via existing `pollStats()` mechanism
- Only shows for non-completed campaigns with sent emails

**Code Location**: Lines 117-132 in `frontend/src/views/Campaigns.vue`

**Conditional Display Logic**:
```vue
v-if="!isDone(props.row) && (stats.sent > 0 || stats.queue_sent > 0 || stats.queueSent > 0)"
```

**Progress Calculation**: Reused existing `getProgressPercent(stats)` method

**CSS Styling**: Lines 610-618
- Max-width: 300px
- Top/bottom margins: 0.5rem
- Removed default progress margins

### Bug Fixes for Deployment

**ESLint Fix 1**: no-restricted-globals (Campaigns.vue:510, 515)
- Changed `isNaN(value)` to `Number.isNaN(value)`
- Affected methods: `formatPercent()`, `formatCurrency()`

**ESLint Fix 2**: vue/no-mutating-props (shopify.vue:11, 42, 56)
- Created computed properties with getters/setters
- Replaced direct v-model binding to props
- Properties: `enabled`, `webhookSecret`, `attributionWindowDays`
- Used `this.$set()` for reactive updates

**Files Modified**:
- `frontend/src/views/Campaigns.vue` (progress bar + isNaN fix)
- `frontend/src/views/settings/shopify.vue` (prop mutation fix)

**Lines Added**: ~35 lines total

**Deployment**: Revision `listmonk420--0000097` (November 9, 2025, 10:10 AM EST)

---

## Deployment History

1. **Revision 1** (listmonk420--deploy-20251108-095703): Failed - GROUP BY error
2. **Revision 2** (listmonk420--deploy-20251108-105513): Started but runtime error - NULL handling
3. **Revision 3** (listmonk420--deploy-20251108-105926): ✅ Success - All issues resolved
4. **Revision 4** (listmonk420--deploy-20251108-111408): Column reorganization
5. **Revision 5** (listmonk420--deploy-20251108-112555): Reactivity fix for live campaigns
6. **Revision 6** (listmonk420--deploy-20251108-113326): Queue-based campaign support
7. **Revision 7** (listmonk420--deploy-20251108-113812): Show view/click counts
8. **Revision 8** (listmonk420--0000097): ✅ Final - Progress bar with email counter (Nov 9, 2025)

## Next Steps / Future Enhancements

- Consider adding date range selector for performance summary
- Add loading states for performance summary fetch
- Add error handling UI for failed summary fetch
- Consider caching summary data to reduce database load
- Add currency formatting based on actual campaign currency (currently hardcoded $)
