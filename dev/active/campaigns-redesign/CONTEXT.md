# Campaigns Page Redesign - Context Document

## Overview
Redesigned the All Campaigns page to integrate Shopify purchase attribution metrics, showing revenue data alongside traditional campaign metrics.

**Status**: ✅ Complete and deployed
**Deployment**: Revision `listmonk420--deploy-20251108-105926`
**URL**: https://list.bobbyseamoss.com

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

## Deployment History

1. **Revision 1** (listmonk420--deploy-20251108-095703): Failed - GROUP BY error
2. **Revision 2** (listmonk420--deploy-20251108-105513): Started but runtime error - NULL handling
3. **Revision 3** (listmonk420--deploy-20251108-105926): ✅ Success - All issues resolved

## Next Steps / Future Enhancements

- Consider adding date range selector for performance summary
- Add loading states for performance summary fetch
- Add error handling UI for failed summary fetch
- Consider caching summary data to reduce database load
- Add currency formatting based on actual campaign currency (currently hardcoded $)
