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

3. **Data Property** (line 361):
   ```javascript
   data() {
     return {
       performanceSummary: null,  // Added
       // ... other data
     };
   }
   ```

4. **Methods** (lines 556-573):
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
**Lines**: 107-113
**Purpose**: Added translation keys

**Changes**:
```json
"campaigns.performanceSummary": "Email performance last 30 days",
"campaigns.avgOpenRate": "Average open rate",
"campaigns.avgClickRate": "Average click rate",
"campaigns.placedOrder": "Placed Order",
"campaigns.revenuePerRecipient": "Revenue per recipient",
"campaigns.recipient": "recipient",
"campaigns.recipients": "recipients"
```

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

**Lines Added**: ~200
- Backend: ~100 lines
- Frontend: ~100 lines

**Deployment Revisions**: 3
- Rev 1: GROUP BY error
- Rev 2: NULL handling error
- Rev 3: ✅ Success
