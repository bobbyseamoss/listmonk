# Manual Verification Guide
## Email Performance Last 30 Days Bug Fix

This guide provides step-by-step instructions for manually verifying the bug fix without running automated tests.

## Quick Verification (1 minute)

1. **Open Browser**: Navigate to https://list.bobbyseamoss.com/admin/campaigns

2. **Login**:
   - Username: `adam`
   - Password: [Use your saved credentials]

3. **Locate Performance Summary**: Look for the section at the top of the page titled "Email performance last 30 days"

4. **Verify Metrics** - Check that ALL values are NON-ZERO:

   | Metric | Expected | Status |
   |--------|----------|--------|
   | Average Open Rate | ~39.29% (NOT 0.00%) | ☐ PASS / ☐ FAIL |
   | Average Click Rate | ~0.36% (NOT 0.00%) | ☐ PASS / ☐ FAIL |
   | Placed Order | ~0.006% (NOT 0.00%) | ☐ PASS / ☐ FAIL |
   | Revenue Per Recipient | ~$0.0021 (NOT $0.00) | ☐ PASS / ☐ FAIL |

## Detailed Verification Steps

### Step 1: Check Section Visibility

- [ ] The "Email performance last 30 days" section is visible at the top of the campaigns page
- [ ] The section can be expanded/collapsed using the arrow icon
- [ ] The section is open by default

### Step 2: Verify Average Open Rate

- [ ] Metric label says "Average open rate"
- [ ] Value is displayed as a percentage (e.g., "39.29%")
- [ ] Value is NOT "0.00%"
- [ ] Value is between 20% and 60% (reasonable range)

**Actual Value**: ____________

### Step 3: Verify Average Click Rate

- [ ] Metric label says "Average click rate"
- [ ] Value is displayed as a percentage (e.g., "0.36%")
- [ ] Value is NOT "0.00%"
- [ ] Value is between 0.1% and 10% (reasonable range)

**Actual Value**: ____________

### Step 4: Verify Placed Order Rate

- [ ] Metric label says "Placed Order"
- [ ] Value is displayed as a percentage (e.g., "0.006%")
- [ ] Value is NOT "0.00%"
- [ ] Value is between 0.001% and 5% (reasonable range)

**Actual Value**: ____________

### Step 5: Verify Revenue Per Recipient

- [ ] Metric label says "Revenue per recipient"
- [ ] Value is displayed as currency (e.g., "$0.0021")
- [ ] Value is NOT "$0.00"
- [ ] Value is greater than $0.00

**Actual Value**: ____________

### Step 6: Verify Layout and Styling

- [ ] All four metrics are displayed in a single row
- [ ] Each metric has equal width (25% each)
- [ ] Values are prominently displayed above labels
- [ ] Styling is consistent with the rest of the UI

### Step 7: Browser DevTools Verification (Optional)

If you want to verify the backend is working correctly:

1. **Open DevTools**: Press F12 or right-click → Inspect
2. **Go to Network Tab**: Click on "Network" at the top
3. **Reload Page**: Press Ctrl+R or Cmd+R
4. **Find API Call**: Look for a request to `/api/campaigns/performance-summary`
5. **Check Response**: Click on the request and view the "Response" tab

You should see JSON like:
```json
{
  "avgOpenRate": 39.29,
  "avgClickRate": 0.36,
  "orderRate": 0.006,
  "revenuePerRecipient": 0.0021
}
```

**Verify**:
- [ ] All property names are in camelCase (NOT snake_case like `avg_open_rate`)
- [ ] All values are numbers (NOT strings)
- [ ] All values are greater than 0

## Before & After Comparison

### BEFORE (Bug):
```
Email performance last 30 days

Average open rate         Average click rate
0.00%                     0.00%

Placed Order              Revenue per recipient
0.00%                     $0.00
```

### AFTER (Fixed):
```
Email performance last 30 days

Average open rate         Average click rate
39.29%                    0.36%

Placed Order              Revenue per recipient
0.006%                    $0.0021
```

## Reporting Results

### If All Tests Pass ✅

The bug fix is **VERIFIED** and working correctly. All metrics are displaying non-zero values from the `azure_delivery_events` table.

### If Any Tests Fail ❌

Please report:
1. Which metric(s) are still showing 0.00%
2. Screenshot of the performance summary section
3. DevTools Network response for `/api/campaigns/performance-summary`
4. Any console errors in DevTools

## Additional Checks

### Edge Cases to Verify:

- [ ] **No data scenario**: If there's truly no data, does it handle gracefully?
- [ ] **Large numbers**: Do very large revenue values display correctly?
- [ ] **Small percentages**: Do very small percentages (< 0.01%) display with enough precision?

### Browser Compatibility:

Test in multiple browsers if possible:
- [ ] Chrome
- [ ] Firefox
- [ ] Safari
- [ ] Edge

### Responsiveness:

- [ ] Desktop view (wide screen)
- [ ] Tablet view (medium screen)
- [ ] Mobile view (narrow screen)

## Screenshot Checklist

For documentation purposes, take screenshots showing:

1. **Full campaigns page** with performance summary expanded
2. **Close-up of performance summary** showing all four metrics clearly
3. **DevTools Network tab** showing the API response

## Sign-off

**Verified by**: _______________
**Date**: _______________
**Browser**: _______________
**Overall Result**: ☐ PASS / ☐ FAIL

**Notes**:
________________________________________________
________________________________________________
________________________________________________
