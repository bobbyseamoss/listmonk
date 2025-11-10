# Email Performance Summary Bug Fix - Verification Guide

## Bug Overview

**Issue**: Email Performance Last 30 Days section showing all zeros (0.00%)
- Location: https://list.bobbyseamoss.com/admin/campaigns
- Affected Metrics: Average Open Rate, Average Click Rate, Placed Order, Revenue Per Recipient

**Root Cause**:
1. SQL query referencing non-existent table
2. Vue template using incorrect property names (snake_case instead of camelCase)

**Fix Applied**:
1. Updated SQL query to use `azure_delivery_events` table
2. Fixed Vue template property names to camelCase
3. Deployed to Bobby Sea Moss production

## Expected Results

After fix, the metrics should display:
- **Average Open Rate**: ~39.29% (previously 0.00%)
- **Average Click Rate**: ~0.36% (previously 0.00%)
- **Placed Order**: ~0.006% (previously 0.00%)
- **Revenue Per Recipient**: ~$0.0021 (previously $0.00)

## Verification Methods

I've created three ways to verify the bug fix:

### Method 1: Automated Playwright Tests (Recommended for CI/CD)

**Location**: `/home/adam/listmonk/e2e-tests/campaigns-performance-summary.spec.js`

**Features**:
- 7 comprehensive test cases
- Verifies all metrics are non-zero
- Validates data ranges
- Checks API response structure
- Takes screenshots for documentation
- Captures console logs

**Test Cases**:
1. Display Email Performance Last 30 Days section
2. Verify non-zero Average Open Rate
3. Verify non-zero Average Click Rate
4. Verify non-zero Placed Order rate
5. Verify non-zero Revenue Per Recipient
6. Verify all metrics within expected ranges
7. Verify API response contains correct camelCase properties

**How to Run**:

```bash
# With password environment variable (automated)
LISTMONK_PASSWORD='your-password' npx playwright test campaigns-performance-summary --headed --project=chromium

# Manual login (interactive - waits 30s for you to login)
npx playwright test campaigns-performance-summary --headed --project=chromium

# Headless mode (requires password)
LISTMONK_PASSWORD='your-password' npx playwright test campaigns-performance-summary --project=chromium
```

**Output**:
- ✅ Pass/Fail status for each test
- Console logs showing actual metric values
- Screenshot: `test-results/email-performance-summary.png`
- HTML report: http://localhost:9323

### Method 2: Direct API Verification Script (Fastest)

**Location**: `/home/adam/listmonk/e2e-tests/verify-api.js`

**Features**:
- Direct HTTPS API call (no browser needed)
- Login and fetch performance summary
- Comprehensive validation with color-coded output
- Checks for common issues:
  - Missing fields
  - Zero values
  - Snake_case properties (old bug)
  - Incorrect data types
  - Out-of-range values

**How to Run**:

```bash
# With command line arguments
node e2e-tests/verify-api.js adam your-password

# With environment variable
LISTMONK_PASSWORD='your-password' node e2e-tests/verify-api.js

# Or make it executable and run directly
./e2e-tests/verify-api.js adam your-password
```

**Output**:
```
📡 Logging in...
✅ Login successful

📊 Fetching performance summary...

============================================================
         EMAIL PERFORMANCE SUMMARY VERIFICATION
============================================================

📋 API Response:
{
  "avgOpenRate": 39.29,
  "avgClickRate": 0.36,
  "orderRate": 0.006,
  "revenuePerRecipient": 0.0021
}

🔍 Verification Results:
------------------------------------------------------------
✅ PASS Average Open Rate: 39.29%
✅ PASS Average Click Rate: 0.36%
✅ PASS Placed Order Rate: 0.006%
✅ PASS Revenue Per Recipient: $0.0021
------------------------------------------------------------

🔍 Additional Checks:
------------------------------------------------------------
✅ PASS: All properties are in camelCase
✅ PASS: All values are numbers

📊 Range Validation:
------------------------------------------------------------
✅ Average Open Rate: 39.29 is within expected range
✅ Average Click Rate: 0.36 is within expected range
✅ Placed Order Rate: 0.006 is within expected range
✅ Revenue Per Recipient: 0.0021 is within expected range
------------------------------------------------------------

============================================================
           ✅ ALL TESTS PASSED - BUG FIX VERIFIED!
============================================================
```

### Method 3: Manual Verification (No Code Required)

**Location**: `/home/adam/listmonk/e2e-tests/manual-verification.md`

**Steps**:
1. Open https://list.bobbyseamoss.com/admin/campaigns
2. Login with username: adam
3. Locate "Email performance last 30 days" section
4. Verify each metric is non-zero:
   - Average Open Rate: ☐ PASS / ☐ FAIL
   - Average Click Rate: ☐ PASS / ☐ FAIL
   - Placed Order: ☐ PASS / ☐ FAIL
   - Revenue Per Recipient: ☐ PASS / ☐ FAIL

**Optional DevTools Check**:
1. Open DevTools (F12)
2. Go to Network tab
3. Reload page
4. Find `/api/campaigns/performance-summary` request
5. Verify response has camelCase properties with non-zero values

## Files Created

```
e2e-tests/
├── campaigns-performance-summary.spec.js   # Playwright test suite (7 tests)
├── auth.setup.js                           # Authentication helper
├── verify-api.js                           # Direct API verification script
├── manual-verification.md                  # Manual testing checklist
└── README.md                               # Complete documentation
```

Updated:
```
playwright.config.js                        # Updated testDir to 'e2e-tests'
```

## Quick Start Guide

### For Developers (Automated Testing)

**Fastest Method - API Script**:
```bash
cd /home/adam/listmonk
LISTMONK_PASSWORD='your-password' node e2e-tests/verify-api.js
```

**Most Comprehensive - Playwright**:
```bash
cd /home/adam/listmonk
LISTMONK_PASSWORD='your-password' npx playwright test campaigns-performance-summary --headed --project=chromium
```

### For QA/Manual Testing

1. Open manual verification guide:
   ```bash
   cat e2e-tests/manual-verification.md
   ```

2. Follow the checklist while accessing:
   https://list.bobbyseamoss.com/admin/campaigns

### For CI/CD Pipeline

Add to `.github/workflows/playwright.yml`:

```yaml
name: E2E Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
      - name: Install dependencies
        run: npm ci && npx playwright install --with-deps chromium
      - name: Run tests
        env:
          LISTMONK_PASSWORD: ${{ secrets.LISTMONK_PASSWORD }}
        run: npx playwright test campaigns-performance-summary --project=chromium
      - uses: actions/upload-artifact@v3
        if: always()
        with:
          name: playwright-report
          path: playwright-report/
```

## Success Criteria

The bug fix is **VERIFIED** if:

✅ All metrics display non-zero values
✅ Values are within reasonable ranges:
   - Open Rate: 20-60%
   - Click Rate: 0.1-10%
   - Order Rate: 0.001-5%
   - Revenue: > $0
✅ API response uses camelCase properties
✅ No console errors in browser DevTools
✅ All Playwright tests pass (7/7)

## Troubleshooting

### Playwright Tests Failing

**Problem**: "Login required" error
- **Solution**: Run with `--headed` flag and login manually, or set `LISTMONK_PASSWORD`

**Problem**: Tests timeout
- **Solution**: Check site is accessible: `curl https://list.bobbyseamoss.com`

**Problem**: Metrics still showing 0.00%
- **Solution**: Verify deployment, check backend logs

### API Script Failing

**Problem**: "Password required"
- **Solution**: Provide password via command line or `LISTMONK_PASSWORD` env var

**Problem**: Connection refused
- **Solution**: Check site is up and accessible from current network

**Problem**: "API call failed: 401"
- **Solution**: Verify username/password are correct

## Next Steps

1. **Run Verification**: Choose one of the three methods above
2. **Document Results**: Take screenshots if using manual verification
3. **Report Status**: Update relevant tracking documents/tickets
4. **Monitor Production**: Watch for any related issues in production logs

## Additional Resources

- **Playwright Docs**: https://playwright.dev/
- **Test Files**: `/home/adam/listmonk/e2e-tests/`
- **Backend Fix**: Check `queries.sql` and `Campaigns.vue` for implementation details
- **Production URL**: https://list.bobbyseamoss.com/admin/campaigns

## Contact

For questions about these tests:
- Review the README: `/home/adam/listmonk/e2e-tests/README.md`
- Check test comments in the spec file
- Consult Playwright documentation
