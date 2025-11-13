const { chromium } = require('playwright');

async function verifyWebhookTabs() {
  console.log('\n=== Verifying Webhook Logs Tabs ===\n');

  const browser = await chromium.launch({
    headless: false,
    slowMo: 100
  });

  const context = await browser.newContext({
    ignoreHTTPSErrors: true,
    // Disable cache to ensure we get fresh content
    javaScriptEnabled: true,
    bypassCSP: false
  });

  const page = await context.newPage();

  try {
    // Login
    console.log('1. Navigating to login page...');
    await page.goto('https://list.bobbyseamoss.com/admin/login');
    await page.waitForSelector('input[name="username"]', { timeout: 10000 });

    console.log('2. Entering credentials...');
    await page.fill('input[name="username"]', 'adam');
    await page.fill('input[name="password"]', 'T@inshr3dd3r');

    console.log('3. Submitting login form...');
    await page.click('button[type="submit"]');

    // Wait for successful login
    await page.waitForURL('**/admin/**', { timeout: 15000 });
    console.log('✓ Login successful!\n');

    // Navigate to webhook logs
    console.log('4. Navigating to webhook logs...');
    await page.goto('https://list.bobbyseamoss.com/admin/settings/webhook-logs');

    // Force hard reload to bypass cache
    console.log('4a. Forcing hard reload to bypass cache...');
    await page.reload({ waitUntil: 'networkidle' });
    await page.waitForTimeout(3000);

    // Check for tabs
    console.log('5. Checking for tabs...');

    // Print out the page HTML to debug
    const pageHTML = await page.content();
    console.log('\n===== PAGE HTML (first 2000 chars) =====');
    console.log(pageHTML.substring(0, 2000));
    console.log('========================================\n');

    const tabsExist = await page.locator('.b-tabs').count();

    if (tabsExist === 0) {
      console.log('❌ No tabs found! Taking screenshot...');
      await page.screenshot({ path: 'screenshots/no-tabs.png', fullPage: true });
      throw new Error('Tabs not found on page');
    }

    console.log(`✓ Found ${tabsExist} tab container(s)\n`);

    // Get tab labels
    const tabLabels = await page.locator('.tabs li a').allTextContents();
    console.log('Tab labels:', tabLabels);
    console.log('');

    // Test clicking each tab and checking the count
    const results = {};

    for (let i = 0; i < tabLabels.length; i++) {
      const label = tabLabels[i].trim();
      console.log(`Testing "${label}" tab...`);

      // Click the tab
      await page.locator('.tabs li').nth(i).click();
      await page.waitForTimeout(2000);

      // Get the total count from header
      const headerText = await page.locator('.page-header h1').textContent();
      const match = headerText.match(/\((\d+)\)/);
      const count = match ? match[1] : '0';

      console.log(`  Total records: ${count}`);

      // Take screenshot
      await page.screenshot({
        path: `screenshots/tab-${label.toLowerCase()}.png`,
        fullPage: true
      });
      console.log(`  ✓ Screenshot: tab-${label.toLowerCase()}.png\n`);

      results[label] = parseInt(count);
    }

    // Verify tabs show different counts
    console.log('\n=== RESULTS ===');
    const counts = Object.values(results);
    const allSame = counts.every(c => c === counts[0]);

    for (const [label, count] of Object.entries(results)) {
      console.log(`${label}: ${count} records`);
    }
    console.log('');

    if (allSame) {
      console.log('❌ PROBLEM: All tabs show the same count!');
      console.log('   Filtering is NOT working.\n');
    } else {
      console.log('✅ SUCCESS: Tabs show different counts!');
      console.log('   Filtering is working correctly.\n');
    }

    await page.waitForTimeout(2000);

  } catch (error) {
    console.error('\n❌ Error:', error.message);
    await page.screenshot({ path: 'screenshots/error.png', fullPage: true });
    console.log('Screenshot saved: screenshots/error.png\n');
  } finally {
    await browser.close();
  }
}

verifyWebhookTabs();
