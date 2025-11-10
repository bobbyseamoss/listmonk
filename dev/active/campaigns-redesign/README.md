# Campaigns Page Redesign

**Status**: ✅ Complete and Deployed
**Revision**: `listmonk420--deploy-20251108-113812` (7 deployment iterations)
**URL**: https://list.bobbyseamoss.com
**Date**: November 8, 2025

## Quick Summary

Redesigned the All Campaigns page to display Shopify purchase attribution metrics alongside traditional campaign metrics. Added:
1. Email performance summary section showing aggregate metrics for last 30 days
2. Simplified campaign table layout with focused metric columns
3. Open Rate and Click Rate columns with percentages and raw counts
4. Start Date column showing when campaign started
5. Placed Order column showing revenue and order counts per campaign
6. Support for both regular and queue-based campaigns

## Documentation Files

- **CONTEXT.md** - Complete implementation details, technical challenges, and solutions
- **SQL-LESSONS.md** - PostgreSQL aggregate function rules and NULL handling patterns
- **FILES-MODIFIED.md** - Detailed list of all file changes with code snippets

## Key Features

### Performance Summary Section
- Displays at top of campaigns page
- Shows last 30 days of aggregate data:
  - Average open rate
  - Average click rate
  - Placed Order rate
  - Revenue per recipient
- Collapsible for cleaner UI
- Gracefully handles missing data (shows 0% when no campaigns or purchases exist)

### Simplified Campaign Table Layout
- **Removed** verbose Stats column (views/clicks/sent/bounces)
- **Removed** verbose Timestamps column (created/started/ended/duration)
- **Added** focused metric columns for better readability

### Start Date Column
- Shows when campaign started (or "—" if not yet started)
- Sortable for easy chronological ordering
- Uses human-readable date format

### Open Rate and Click Rate Columns
- Displays engagement rate as percentage (e.g., "5.23%")
- Shows raw count below percentage (e.g., "15 views", "3 clicks")
- Works with both regular and queue-based campaigns
- Real-time updates for running campaigns via polling
- Shows "—" when no emails sent yet

### Placed Order Column
- Shows in campaigns table for each campaign
- Displays:
  - Total revenue from purchases (formatted as currency)
  - Number of recipients who made purchases
- Falls back to "$0.00 / 0 recipients" for campaigns with no purchases

## Technical Highlights

### Backend
- Efficient SQL using CTEs and aggregate functions
- Bulk fetching of purchase stats using `ANY($1)` to avoid N+1 queries
- Proper NULL handling with COALESCE to prevent runtime errors
- New API endpoint: `GET /api/campaigns/performance/summary`

### Frontend
- Vue 2.7 component integration
- Responsive Bulma/Buefy styling
- Helper functions for formatting percentages and currency
- Smart campaign type detection for rate calculations
- Real-time reactivity for live campaign updates
- i18n support for all new UI text

## Critical Fixes Applied

1. **PostgreSQL GROUP BY Constraint**: Wrapped CROSS JOIN columns in MAX() aggregate
2. **NULL Value Handling**: Added COALESCE to all aggregates for empty data scenarios
3. **Vue Reactivity**: Fixed template expressions to properly track polling updates
4. **Queue Campaign Support**: Enhanced rate calculations to handle both regular and queue-based campaigns

See SQL-LESSONS.md for PostgreSQL patterns and CONTEXT.md for post-deployment enhancements.

## Testing

All features tested and working:
- [x] Performance summary displays correctly
- [x] Handles missing purchase data gracefully
- [x] Placed Order column shows in table
- [x] Zero-revenue campaigns display properly
- [x] Open/Click Rate columns show percentages and counts
- [x] Real-time updates work for running campaigns
- [x] Queue-based campaigns calculate rates correctly
- [x] Start Date column displays properly
- [x] Deployed successfully with no errors (7 iterations)

## Future Enhancements

- Date range selector for performance summary
- Loading states for summary fetch
- Currency formatting based on actual campaign currency
- Error handling UI for failed fetches
- Summary data caching to reduce database load

## Related Work

This builds on the existing Shopify integration documented in:
- `/dev/active/shopify-integration/`

## Getting Started

To understand this implementation:
1. Read CONTEXT.md for overview and technical details
2. Review SQL-LESSONS.md to understand the PostgreSQL patterns used
3. Check FILES-MODIFIED.md for specific code changes
