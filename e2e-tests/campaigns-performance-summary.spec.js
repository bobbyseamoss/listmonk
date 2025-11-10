// @ts-check
import { test, expect } from '@playwright/test';

/**
 * Bug Fix Verification: Email Performance Last 30 Days Metrics
 *
 * Context:
 * - Fixed SQL query to use azure_delivery_events table instead of non-existent table
 * - Fixed Vue template property names from snake_case to camelCase
 * - Metrics were showing 0.00% for all values before fix
 *
 * This test verifies that:
 * 1. The performance summary section loads successfully
 * 2. All metrics display non-zero values (confirming the fix works)
 * 3. Values are within reasonable ranges
 */

test.describe('Bug Fix: Email Performance Summary Metrics', () => {
  // Bobby Sea Moss production credentials
  const BASE_URL = 'https://list.bobbyseamoss.com';
  const USERNAME = 'adam';
  // Note: Password should be stored in environment variable or retrieved from browser
  // For now, we'll rely on saved credentials or manual login

  test.beforeEach(async ({ page }) => {
    // Navigate to login page first
    await page.goto(`${BASE_URL}/admin/login`);

    // Check if already logged in (redirected to campaigns)
    await page.waitForTimeout(1000);
    const currentURL = page.url();

    if (currentURL.includes('/admin/campaigns')) {
      // Already logged in
      await page.waitForLoadState('networkidle');
      return;
    }

    // Need to login
    const usernameInput = page.locator('input[name="username"]');
    const passwordInput = page.locator('input[name="password"]');

    await usernameInput.fill(USERNAME);

    // Get password from environment variable or use default for testing
    const password = process.env.LISTMONK_PASSWORD || '';

    if (!password) {
      console.log('\n⚠️  No password provided. Set LISTMONK_PASSWORD environment variable or login manually.');
      console.log('   Waiting 30 seconds for manual login...\n');

      // Wait for user to login manually
      await page.waitForURL(`${BASE_URL}/admin/campaigns`, { timeout: 30000 });
    } else {
      await passwordInput.fill(password);

      const loginButton = page.getByRole('button', { name: /log in|sign in|login/i });
      await loginButton.click();

      // Wait for successful login
      await page.waitForURL(`${BASE_URL}/admin/campaigns`, { timeout: 10000 });
    }

    // Wait for the page to fully load
    await page.waitForLoadState('networkidle');
  });

  test('should display Email Performance Last 30 Days section', async ({ page }) => {
    // Verify the performance summary section exists
    const performanceSection = page.locator('.performance-summary');
    await expect(performanceSection).toBeVisible();

    // Verify the section header
    const header = performanceSection.locator('summary');
    await expect(header).toContainText(/Email performance last 30 days/i);
  });

  test('should display non-zero Average Open Rate', async ({ page }) => {
    // Wait for the performance summary to load
    const performanceSection = page.locator('.performance-summary');
    await expect(performanceSection).toBeVisible();

    // Locate the Average Open Rate stat
    const openRateStat = performanceSection.locator('.stat-item').filter({
      hasText: /Average open rate/i
    });
    await expect(openRateStat).toBeVisible();

    // Get the stat value
    const statValue = openRateStat.locator('.stat-value');
    const valueText = await statValue.textContent();

    // Verify it's not 0.00%
    expect(valueText).not.toBe('0.00%');

    // Verify it's a valid percentage format
    expect(valueText).toMatch(/^\d+\.\d{2}%$/);

    // Extract numeric value and verify it's in reasonable range (0-100%)
    const numericValue = parseFloat(valueText.replace('%', ''));
    expect(numericValue).toBeGreaterThan(0);
    expect(numericValue).toBeLessThanOrEqual(100);

    console.log(`✓ Average Open Rate: ${valueText}`);
  });

  test('should display non-zero Average Click Rate', async ({ page }) => {
    // Wait for the performance summary to load
    const performanceSection = page.locator('.performance-summary');
    await expect(performanceSection).toBeVisible();

    // Locate the Average Click Rate stat
    const clickRateStat = performanceSection.locator('.stat-item').filter({
      hasText: /Average click rate/i
    });
    await expect(clickRateStat).toBeVisible();

    // Get the stat value
    const statValue = clickRateStat.locator('.stat-value');
    const valueText = await statValue.textContent();

    // Verify it's not 0.00%
    expect(valueText).not.toBe('0.00%');

    // Verify it's a valid percentage format
    expect(valueText).toMatch(/^\d+\.\d{2}%$/);

    // Extract numeric value and verify it's in reasonable range (0-100%)
    const numericValue = parseFloat(valueText.replace('%', ''));
    expect(numericValue).toBeGreaterThan(0);
    expect(numericValue).toBeLessThanOrEqual(100);

    console.log(`✓ Average Click Rate: ${valueText}`);
  });

  test('should display non-zero Placed Order rate', async ({ page }) => {
    // Wait for the performance summary to load
    const performanceSection = page.locator('.performance-summary');
    await expect(performanceSection).toBeVisible();

    // Locate the Placed Order stat
    const orderRateStat = performanceSection.locator('.stat-item').filter({
      hasText: /Placed Order/i
    });
    await expect(orderRateStat).toBeVisible();

    // Get the stat value
    const statValue = orderRateStat.locator('.stat-value');
    const valueText = await statValue.textContent();

    // Verify it's not 0.00%
    expect(valueText).not.toBe('0.00%');

    // Verify it's a valid percentage format
    expect(valueText).toMatch(/^\d+\.\d{2,}%$/);

    // Extract numeric value and verify it's in reasonable range (0-100%)
    const numericValue = parseFloat(valueText.replace('%', ''));
    expect(numericValue).toBeGreaterThan(0);
    expect(numericValue).toBeLessThanOrEqual(100);

    console.log(`✓ Placed Order Rate: ${valueText}`);
  });

  test('should display non-zero Revenue Per Recipient', async ({ page }) => {
    // Wait for the performance summary to load
    const performanceSection = page.locator('.performance-summary');
    await expect(performanceSection).toBeVisible();

    // Locate the Revenue Per Recipient stat
    const revenueStat = performanceSection.locator('.stat-item').filter({
      hasText: /Revenue per recipient/i
    });
    await expect(revenueStat).toBeVisible();

    // Get the stat value
    const statValue = revenueStat.locator('.stat-value');
    const valueText = await statValue.textContent();

    // Verify it's not $0.00
    expect(valueText).not.toBe('$0.00');

    // Verify it's a valid currency format
    expect(valueText).toMatch(/^\$\d+\.\d{2,}$/);

    // Extract numeric value and verify it's positive
    const numericValue = parseFloat(valueText.replace('$', ''));
    expect(numericValue).toBeGreaterThan(0);

    console.log(`✓ Revenue Per Recipient: ${valueText}`);
  });

  test('should display all metrics within expected ranges', async ({ page }) => {
    // Wait for the performance summary to load
    const performanceSection = page.locator('.performance-summary');
    await expect(performanceSection).toBeVisible();

    // Take a screenshot for verification
    await page.screenshot({
      path: 'test-results/email-performance-summary.png',
      fullPage: false
    });

    // Collect all metric values
    const metrics = {
      avgOpenRate: null,
      avgClickRate: null,
      orderRate: null,
      revenuePerRecipient: null
    };

    // Get Average Open Rate
    const openRateStat = performanceSection.locator('.stat-item').filter({
      hasText: /Average open rate/i
    });
    metrics.avgOpenRate = await openRateStat.locator('.stat-value').textContent();

    // Get Average Click Rate
    const clickRateStat = performanceSection.locator('.stat-item').filter({
      hasText: /Average click rate/i
    });
    metrics.avgClickRate = await clickRateStat.locator('.stat-value').textContent();

    // Get Placed Order Rate
    const orderRateStat = performanceSection.locator('.stat-item').filter({
      hasText: /Placed Order/i
    });
    metrics.orderRate = await orderRateStat.locator('.stat-value').textContent();

    // Get Revenue Per Recipient
    const revenueStat = performanceSection.locator('.stat-item').filter({
      hasText: /Revenue per recipient/i
    });
    metrics.revenuePerRecipient = await revenueStat.locator('.stat-value').textContent();

    // Log all metrics
    console.log('\n=== Email Performance Summary Metrics ===');
    console.log(`Average Open Rate: ${metrics.avgOpenRate}`);
    console.log(`Average Click Rate: ${metrics.avgClickRate}`);
    console.log(`Placed Order Rate: ${metrics.orderRate}`);
    console.log(`Revenue Per Recipient: ${metrics.revenuePerRecipient}`);
    console.log('=========================================\n');

    // Verify expected ranges based on context
    // Expected: ~39.29% open rate, ~0.36% click rate, ~0.006% order rate, ~$0.0021 revenue

    const openRate = parseFloat(metrics.avgOpenRate.replace('%', ''));
    const clickRate = parseFloat(metrics.avgClickRate.replace('%', ''));
    const orderRate = parseFloat(metrics.orderRate.replace('%', ''));
    const revenue = parseFloat(metrics.revenuePerRecipient.replace('$', ''));

    // Open rate should be between 20-60% (reasonable range for email marketing)
    expect(openRate).toBeGreaterThanOrEqual(20);
    expect(openRate).toBeLessThanOrEqual(60);

    // Click rate should be between 0.1-10% (typical range)
    expect(clickRate).toBeGreaterThanOrEqual(0.1);
    expect(clickRate).toBeLessThanOrEqual(10);

    // Order rate should be between 0.001-5% (e-commerce typical)
    expect(orderRate).toBeGreaterThanOrEqual(0.001);
    expect(orderRate).toBeLessThanOrEqual(5);

    // Revenue should be positive (no upper bound check as it depends on products)
    expect(revenue).toBeGreaterThan(0);
  });

  test('should verify API response contains correct data', async ({ page }) => {
    // Intercept the API call to verify the backend is returning correct data
    let apiResponse = null;

    page.on('response', async (response) => {
      if (response.url().includes('/api/campaigns/performance-summary')) {
        apiResponse = await response.json();
      }
    });

    // Navigate to campaigns page to trigger API call
    await page.goto(`${BASE_URL}/admin/campaigns`);
    await page.waitForLoadState('networkidle');

    // Wait for API response
    await page.waitForTimeout(2000);

    // Verify API response was captured
    expect(apiResponse).not.toBeNull();

    // Verify response has the expected camelCase properties (not snake_case)
    expect(apiResponse).toHaveProperty('avgOpenRate');
    expect(apiResponse).toHaveProperty('avgClickRate');
    expect(apiResponse).toHaveProperty('orderRate');
    expect(apiResponse).toHaveProperty('revenuePerRecipient');

    // Verify all values are non-zero numbers
    expect(typeof apiResponse.avgOpenRate).toBe('number');
    expect(typeof apiResponse.avgClickRate).toBe('number');
    expect(typeof apiResponse.orderRate).toBe('number');
    expect(typeof apiResponse.revenuePerRecipient).toBe('number');

    expect(apiResponse.avgOpenRate).toBeGreaterThan(0);
    expect(apiResponse.avgClickRate).toBeGreaterThan(0);
    expect(apiResponse.orderRate).toBeGreaterThan(0);
    expect(apiResponse.revenuePerRecipient).toBeGreaterThan(0);

    console.log('\n=== API Response Data ===');
    console.log(JSON.stringify(apiResponse, null, 2));
    console.log('========================\n');
  });
});
