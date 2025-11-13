const { chromium } = require('playwright');

async function testWebhookTabs() {
  console.log('\n=== Testing Webhook Logs Tabs ===\n');

  const browser = await chromium.launch({
    headless: false
  });
  const context = await browser.newContext({
    ignoreHTTPSErrors: true
  });
  const page = await context.newPage();

  // Track network requests
  const requests = [];
  page.on('request', request => {
    if (request.url().includes('/api/webhook-logs')) {
      requests.push({
        url: request.url(),
        method: request.method()
      });
    }
  });

  try {
    // Login
    console.log('Logging in...');
    await page.goto('https://list.bobbyseamoss.com/admin/login');
    await page.fill('input[name="username"]', 'adam');
    await page.fill('input[name="password"]', 'T@inshr3dd3r');
    await page.click('button[type="submit"]');
    await page.waitForURL('**/admin/**', { timeout: 10000 });
    console.log('✓ Logged in\n');

    // Navigate to webhook logs
    console.log('Navigating to webhook logs page...');
    await page.goto('https://list.bobbyseamoss.com/admin/settings/webhook-logs');
    await page.waitForTimeout(3000);
    console.log('✓ Page loaded\n');

    // Check what's actually on the page
    console.log('Checking page structure...');

    // Look for tabs with different selectors
    const bTabs = await page.locator('.b-tabs').count();
    const tabs = await page.locator('.tabs li').count();
    const tabItems = await page.locator('.tab-item').count();

    console.log(`  .b-tabs elements: ${bTabs}`);
    console.log(`  .tabs li elements: ${tabs}`);
    console.log(`  .tab-item elements: ${tabItems}`);

    // Take screenshot to see what's actually there
    await page.screenshot({
      path: 'screenshots/webhook-logs-page.png',
      fullPage: true
    });
    console.log('✓ Screenshot saved: webhook-logs-page.png');

    // Get page HTML
    const bodyHTML = await page.locator('body').innerHTML();
    console.log('\nPage HTML (first 1000 chars):');
    console.log(bodyHTML.substring(0, 1000));
    console.log('\n');

    // Test each tab
    const tabNames = ['Delivered', 'Failed', 'Views', 'Clicks'];
    const results = {};

    for (let i = 0; i < tabNames.length; i++) {
      const tabName = tabNames[i];
      console.log(`\n=== Testing ${tabName} Tab ===`);

      // Clear previous requests
      requests.length = 0;

      // Click tab
      await page.locator('.tabs li').nth(i).click();
      await page.waitForTimeout(2000);

      // Check what API request was made
      console.log('API Requests:');
      requests.forEach(req => {
        console.log(`  ${req.method} ${req.url}`);
      });

      // Count rows in table
      const rows = await page.locator('.b-table tbody tr').count();
      console.log(`Rows visible: ${rows}`);

      // Get total count from header
      const header = await page.locator('.page-header h1').textContent();
      const totalMatch = header.match(/\((\d+)\)/);
      const total = totalMatch ? totalMatch[1] : '?';
      console.log(`Total from header: ${total}`);

      // Take screenshot
      await page.screenshot({
        path: `screenshots/webhook-${tabName.toLowerCase()}-tab.png`,
        fullPage: true
      });
      console.log(`✓ Screenshot saved: webhook-${tabName.toLowerCase()}-tab.png`);

      results[tabName] = {
        requests: [...requests],
        rowCount: rows,
        total: total
      };
    }

    // Summary
    console.log('\n\n=== SUMMARY ===\n');
    for (const [tabName, data] of Object.entries(results)) {
      console.log(`${tabName}:`);
      console.log(`  Total: ${data.total}`);
      console.log(`  Rows: ${data.rowCount}`);
      console.log(`  API URL: ${data.requests[0]?.url || 'No request'}`);
      console.log('');
    }

    // Check if all tabs have the same total
    const totals = Object.values(results).map(r => r.total);
    const allSame = totals.every(t => t === totals[0]);

    if (allSame) {
      console.log('❌ PROBLEM CONFIRMED: All tabs show the same total count!');
      console.log(`   All tabs show: ${totals[0]} records\n`);
    } else {
      console.log('✅ Tabs show different totals (filtering is working)\n');
    }

  } catch (error) {
    console.error('Error:', error.message);
    await page.screenshot({ path: 'screenshots/error.png' });
  } finally {
    await browser.close();
  }
}

testWebhookTabs();
