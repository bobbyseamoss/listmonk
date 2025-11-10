# E2E Tests for listmonk

## Email Performance Summary Bug Fix Verification

This test suite verifies that the "Email Performance Last 30 Days" metrics are displaying correctly on the Bobby Sea Moss production instance.

### Bug Context

**Issue**: All performance summary metrics (Open Rate, Click Rate, Placed Order, Revenue Per Recipient) were showing 0.00% values.

**Fix Applied**:
1. Updated SQL query in backend to use `azure_delivery_events` table
2. Fixed Vue template property names from snake_case to camelCase
3. Deployed to Bobby Sea Moss production

**Expected Results**:
- Average Open Rate: ~39.29% (NOT 0.00%)
- Average Click Rate: ~0.36% (NOT 0.00%)
- Placed Order: ~0.006% (NOT 0.00%)
- Revenue Per Recipient: ~$0.0021 (NOT $0.00)

### Running the Tests

#### Option 1: With Password Environment Variable (Automated)

```bash
LISTMONK_PASSWORD='your-password' npx playwright test campaigns-performance-summary --headed --project=chromium
```

#### Option 2: Manual Login (Interactive)

```bash
npx playwright test campaigns-performance-summary --headed --project=chromium
```

The test will:
1. Open the browser
2. Navigate to login page
3. Fill in username (adam)
4. Wait 30 seconds for you to enter the password manually
5. Continue with all tests

#### Option 3: Run Without Browser (Headless - requires password)

```bash
LISTMONK_PASSWORD='your-password' npx playwright test campaigns-performance-summary --project=chromium
```

### Test Scenarios

The test suite includes 7 test cases:

1. **Display Email Performance Section** - Verifies the section exists and is visible
2. **Non-zero Average Open Rate** - Verifies open rate is not 0.00% and is in valid range
3. **Non-zero Average Click Rate** - Verifies click rate is not 0.00% and is in valid range
4. **Non-zero Placed Order Rate** - Verifies order rate is not 0.00% and is in valid range
5. **Non-zero Revenue Per Recipient** - Verifies revenue is not $0.00 and is positive
6. **All Metrics Within Expected Ranges** - Comprehensive check of all metrics with screenshot
7. **API Response Verification** - Verifies backend returns correct camelCase properties with non-zero values

### Viewing Results

After running tests:

1. **Terminal Output**: Shows pass/fail status and console logs with actual metric values
2. **HTML Report**: Opens automatically at http://localhost:9323 (if tests fail)
3. **Screenshots**: Saved to `test-results/email-performance-summary.png`
4. **Trace Files**: Available for debugging failed tests

To view HTML report manually:
```bash
npx playwright show-report
```

### Test Structure

```
e2e-tests/
├── campaigns-performance-summary.spec.js  # Main test file
├── auth.setup.js                          # Authentication helper (optional)
└── README.md                              # This file
```

### Troubleshooting

**Problem**: Tests fail with "Login required" error
- **Solution**: Run with `--headed` flag and login manually, or provide LISTMONK_PASSWORD

**Problem**: Tests timeout waiting for elements
- **Solution**: Check that the production site is accessible and the performance summary is loading

**Problem**: Metrics are still showing 0.00%
- **Solution**: Verify the fix was deployed correctly to Bobby Sea Moss production

**Problem**: Browser doesn't open in headed mode
- **Solution**: Install Playwright browsers: `npx playwright install chromium`

### Debugging Tips

1. **Run in debug mode**:
   ```bash
   LISTMONK_PASSWORD='your-password' npx playwright test campaigns-performance-summary --debug --project=chromium
   ```

2. **Run only one test**:
   ```bash
   npx playwright test campaigns-performance-summary --headed --project=chromium --grep "non-zero Average Open Rate"
   ```

3. **Increase timeout**:
   ```bash
   npx playwright test campaigns-performance-summary --headed --project=chromium --timeout=60000
   ```

4. **View trace**:
   ```bash
   npx playwright show-trace test-results/.../trace.zip
   ```

### CI/CD Integration

To run these tests in CI/CD pipelines:

1. Set `LISTMONK_PASSWORD` as a secret environment variable
2. Add to GitHub Actions workflow:

```yaml
- name: Run E2E Tests
  env:
    LISTMONK_PASSWORD: ${{ secrets.LISTMONK_PASSWORD }}
  run: npx playwright test campaigns-performance-summary --project=chromium
```

### Success Criteria

The bug fix is verified as successful if:

✅ All 7 tests pass
✅ Average Open Rate is > 0.00% and between 20-60%
✅ Average Click Rate is > 0.00% and between 0.1-10%
✅ Placed Order Rate is > 0.00% and between 0.001-5%
✅ Revenue Per Recipient is > $0.00
✅ API response contains camelCase properties (not snake_case)
✅ Screenshot shows non-zero values for all metrics
