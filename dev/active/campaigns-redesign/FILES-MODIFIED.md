# Files Modified - Campaigns Page Redesign

## Backend Files

### queries.sql
**Lines**: 1659-1717
**Purpose**: Added SQL queries for performance metrics

**Changes**:
1. Added `get-campaigns-performance-summary` query
   - Aggregates campaign metrics for last 30 days
   - Uses CTEs for recent_campaigns and purchase_data
   - Calculates avg open rate, avg click rate, order rate, revenue per recipient
   - **Key fix**: COALESCE(MAX(...), 0) for NULL safety

2. Added `get-campaigns-purchase-stats` query
   - Bulk fetches purchase stats for multiple campaigns
   - Uses `WHERE campaign_id = ANY($1)` for efficiency

### models/models.go
**Lines**: 282-285, 419-436
**Purpose**: Added data structures for performance metrics

**Changes**:
1. Extended `CampaignMeta` struct (lines 282-285):
   ```go
   // Purchase attribution stats (from Shopify integration)
   PurchaseOrders  int     `db:"purchase_orders" json:"purchase_orders"`
   PurchaseRevenue float64 `db:"purchase_revenue" json:"purchase_revenue"`
   PurchaseCurrency string `db:"purchase_currency" json:"purchase_currency"`
   ```

2. Added `CampaignsPerformanceSummary` struct (lines 419-427):
   ```go
   type CampaignsPerformanceSummary struct {
       AvgOpenRate         float64
       AvgClickRate        float64
       TotalSent           int
       TotalOrders         int
       TotalRevenue        float64
       OrderRate           float64
       RevenuePerRecipient float64
   }
   ```

3. Added `CampaignPurchaseStatsListItem` struct (lines 429-436):
   ```go
   type CampaignPurchaseStatsListItem struct {
       CampaignID   int
       TotalOrders  int
       TotalRevenue float64
       Currency     string
   }
   ```

### models/queries.go
**Lines**: 167-168
**Purpose**: Register new SQL queries

**Changes**:
```go
GetCampaignsPerformanceSummary *sqlx.Stmt `query:"get-campaigns-performance-summary"`
GetCampaignsPurchaseStats     *sqlx.Stmt `query:"get-campaigns-purchase-stats"`
```

### cmd/shopify.go
**Lines**: 165-190
**Purpose**: Added API endpoint for performance summary

**Changes**:
```go
func (app *App) GetCampaignsPerformanceSummary(c echo.Context) error {
    var summary models.CampaignsPerformanceSummary

    err := app.queries.GetCampaignsPerformanceSummary.Get(&summary)
    if err != nil {
        if err == sql.ErrNoRows {
            // Return zero stats
            summary = models.CampaignsPerformanceSummary{...}
        } else {
            app.log.Printf("error fetching campaigns performance summary: %v", err)
            return echo.NewHTTPError(http.StatusInternalServerError, ...)
        }
    }

    return c.JSON(http.StatusOK, okResp{summary})
}
```

### cmd/campaigns.go
**Lines**: 100-126
**Purpose**: Enhanced campaigns list to include purchase stats

**Changes**:
```go
// After fetching campaigns list:

// 1. Collect campaign IDs
campaignIDs := make([]int, len(res))
for i, campaign := range res {
    campaignIDs[i] = campaign.ID
}

// 2. Bulk fetch purchase stats
var purchaseStats []models.CampaignPurchaseStatsListItem
if err := a.queries.GetCampaignsPurchaseStats.Select(&purchaseStats, pq.Array(campaignIDs)); err != nil {
    a.log.Printf("error fetching purchase stats for campaigns: %v", err)
    // Don't fail the request
}

// 3. Map stats to campaigns
purchaseStatsMap := make(map[int]models.CampaignPurchaseStatsListItem)
for _, stats := range purchaseStats {
    purchaseStatsMap[stats.CampaignID] = stats
}

// 4. Populate campaign results
for i := range res {
    if stats, ok := purchaseStatsMap[res[i].ID]; ok {
        res[i].PurchaseOrders = stats.TotalOrders
        res[i].PurchaseRevenue = stats.TotalRevenue
        res[i].PurchaseCurrency = stats.Currency
    }
}
```

### cmd/handlers.go
**Line**: 188
**Purpose**: Register performance summary API route

**Changes**:
```go
g.GET("/api/campaigns/performance/summary", pm(a.GetCampaignsPerformanceSummary, "campaigns:get_all", "campaigns:get"))
```

## Frontend Files

### frontend/src/api/index.js
**Lines**: 368-372
**Purpose**: Added API client method

**Changes**:
```javascript
// Get aggregate campaign performance summary (last 30 days)
export const getCampaignsPerformanceSummary = async () => http.get(
  '/api/campaigns/performance/summary',
  { loading: models.campaigns },
);
```

### frontend/src/views/Campaigns.vue
**Major changes throughout file**
**Purpose**: Main UI changes for performance summary and Placed Order column

**Key Changes**:

1. **Performance Summary Section** (lines 20-53):
   ```vue
   <section class="performance-summary" v-if="performanceSummary">
     <div class="box">
       <details open>
         <summary class="title is-6">
           {{ $t('campaigns.performanceSummary', 'Email performance last 30 days') }}
         </summary>
         <div class="columns stats-grid">
           <!-- 4 stat columns: open rate, click rate, placed order, revenue/recipient -->
         </div>
       </details>
     </div>
   </section>
   ```

2. **Placed Order Column** (lines 236-247):
   ```vue
   <b-table-column v-slot="props" field="purchase_revenue"
                   :label="$t('campaigns.placedOrder', 'Placed Order')" width="12%">
     <div class="fields stats">
       <p v-if="props.row.purchase_orders > 0">
         <label>{{ formatCurrency(props.row.purchase_revenue) }}</label>
         <span>{{ props.row.purchase_orders }} {{ ... }}</span>
       </p>
       <p v-else>
         <label>$0.00</label>
         <span>0 {{ $t('campaigns.recipients', 'recipients') }}</span>
       </p>
     </div>
   </b-table-column>
   ```

3. **Removed Columns** (post-deployment enhancement):
   - Stats column (views/clicks/sent/bounces/rate combined format)
   - Timestamps column (created/started/ended/duration)

4. **Added Start Date Column** (lines 132-142):
   ```vue
   <b-table-column v-slot="props" field="created_at"
                   :label="$t('campaigns.startDate', 'Start Date')"
                   width="12%" sortable header-class="cy-start-date">
     <div :set="stats = getCampaignStats(props.row)">
       <p v-if="stats.startedAt">
         {{ $utils.niceDate(stats.startedAt, true) }}
       </p>
       <p v-else class="has-text-grey">—</p>
     </div>
   </b-table-column>
   ```

5. **Added Open Rate Column** (lines 144-151):
   ```vue
   <b-table-column v-slot="props" field="open_rate"
                   :label="$t('campaigns.openRate', 'Open Rate')"
                   width="10%">
     <div class="fields stats" :set="stats = getCampaignStats(props.row)">
       <p>
         <label for="#">{{ calculateOpenRate(stats) }}</label>
         <span>{{ stats.views || 0 }} {{ $t('campaigns.views', 'views') }}</span>
       </p>
     </div>
   </b-table-column>
   ```

6. **Added Click Rate Column** (lines 153-160):
   ```vue
   <b-table-column v-slot="props" field="click_rate"
                   :label="$t('campaigns.clickRate', 'Click Rate')"
                   width="10%">
     <div class="fields stats" :set="stats = getCampaignStats(props.row)">
       <p>
         <label for="#">{{ calculateClickRate(stats) }}</label>
         <span>{{ stats.clicks || 0 }} {{ $t('campaigns.clicks', 'clicks') }}</span>
       </p>
     </div>
   </b-table-column>
   ```

7. **Data Property** (line 361):
   ```javascript
   data() {
     return {
       performanceSummary: null,  // Added
       // ... other data
     };
   }
   ```

8. **Enhanced Rate Calculation Methods** (lines 495-533):
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
   },

   calculateClickRate(stats) {
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

     const clicks = stats.clicks || 0;
     const rate = (clicks / sentCount) * 100;
     return `${rate.toFixed(2)}%`;
   }
   ```

9. **Helper Methods** (lines 556-573):
   ```javascript
   getPerformanceSummary() {
     this.$api.getCampaignsPerformanceSummary().then((data) => {
       this.performanceSummary = data;
     }).catch(() => {
       this.performanceSummary = null;
     });
   },

   formatPercent(value) {
     if (!value || isNaN(value)) return '0.00%';
     return `${value.toFixed(2)}%`;
   },

   formatCurrency(value) {
     if (!value || isNaN(value)) return '0.00';
     return value.toFixed(2);
   },
   ```

5. **mounted() Hook** (line 583):
   ```javascript
   mounted() {
     this.getCampaigns();
     this.pollStats();
     this.getPerformanceSummary();  // Added
   },
   ```

6. **CSS Styling** (lines 592-625):
   ```css
   <style scoped>
   .performance-summary { ... }
   .performance-summary .box { ... }
   .performance-summary summary { ... }
   .performance-summary .stats-grid { ... }
   .performance-summary .stat-item { ... }
   .performance-summary .stat-value { ... }
   .performance-summary .stat-label { ... }
   </style>
   ```

### i18n/en.json
**Lines**: 107-117
**Purpose**: Added translation keys

**Changes**:
```json
"campaigns.performanceSummary": "Email performance last 30 days",
"campaigns.avgOpenRate": "Average open rate",
"campaigns.avgClickRate": "Average click rate",
"campaigns.placedOrder": "Placed Order",
"campaigns.revenuePerRecipient": "Revenue per recipient",
"campaigns.recipient": "recipient",
"campaigns.recipients": "recipients",
"campaigns.startDate": "Start Date",
"campaigns.openRate": "Open Rate",
"campaigns.clickRate": "Click Rate",
"campaigns.views": "views",
"campaigns.clicks": "clicks"
```

**Total**: 12 new translation keys (7 initial + 5 from post-deployment enhancements)

## Summary Statistics

**Backend Files**: 6
- queries.sql
- models/models.go
- models/queries.go
- cmd/shopify.go
- cmd/campaigns.go
- cmd/handlers.go

**Frontend Files**: 3
- frontend/src/api/index.js
- frontend/src/views/Campaigns.vue
- i18n/en.json

**Total Files Modified**: 9

**Lines Added**: ~285 (cumulative across all iterations)
- Backend: ~100 lines (initial deployment)
- Frontend: ~185 lines (initial + post-deployment enhancements)
  - Initial: ~100 lines (performance summary, placed order column)
  - Post-deployment: ~85 lines (column reorganization, rate calculations)

**Deployment Revisions**: 7
- Rev 1: GROUP BY error (failed)
- Rev 2: NULL handling error (failed)
- Rev 3: ✅ Initial success - Performance summary + Placed Order
- Rev 4: Column reorganization (Start Date, Open Rate, Click Rate)
- Rev 5: Reactivity fix for live campaigns
- Rev 6: Queue-based campaign support
- Rev 7: ✅ Final - View/click counts display

**Code Changes Summary**:
- 2 new SQL queries
- 3 new Go structs
- 1 new API endpoint
- 4 removed Vue table columns (Stats, Timestamps)
- 5 added Vue table columns (Start Date, Open Rate, Click Rate + existing Placed Order)
- 2 enhanced calculation methods (~40 lines each)
- 3 helper methods (formatting)
- 12 i18n translation keys
- Custom CSS styling
