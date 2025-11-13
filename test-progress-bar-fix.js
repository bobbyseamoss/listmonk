const { chromium } = require('playwright');

async function testProgressBarFix() {
  console.log('=== Testing Progress Bar Fix on Bobby Seamoss ===\n');

  const browser = await chromium.launch({
    headless: false
  });
  const context = await browser.newContext({
    ignoreHTTPSErrors: true
  });
  const page = await context.newPage();

  try {
    // Login
    console.log('1. Logging in to Bobby Seamoss...');
    await page.goto('https://list.bobbyseamoss.com/admin/login');
    await page.fill('input[name="username"]', 'adam');
    await page.fill('input[name="password"]', 'bobbysea');
    await page.click('button[type="submit"]');
    await page.waitForURL('**/admin/**', { timeout: 10000 });
    console.log('✓ Logged in successfully\n');

    // Navigate to Campaigns page
    console.log('2. Navigating to Campaigns page...');
    await page.goto('https://list.bobbyseamoss.com/admin/campaigns');
    await page.waitForTimeout(3000);
    console.log('✓ Campaigns page loaded\n');

    // Intercept API calls to see what data is being returned
    console.log('3. Intercepting API calls...');
    const apiResponses = [];

    page.on('response', async (response) => {
      const url = response.url();
      if (url.includes('/api/campaigns') && response.status() === 200) {
        try {
          const data = await response.json();
          apiResponses.push({ url, data });
        } catch (e) {
          // Not JSON, ignore
        }
      }
    });

    // Reload to capture API calls
    await page.reload();
    await page.waitForTimeout(3000);

    // Find paused campaigns
    console.log('4. Looking for paused campaigns...');
    const campaignRows = await page.locator('table tbody tr').all();
    console.log(`Found ${campaignRows.length} campaigns\n`);

    let foundPausedQueueCampaign = false;

    for (let i = 0; i < campaignRows.length; i++) {
      const row = campaignRows[i];

      // Check if it's a paused campaign
      const statusBadge = row.locator('.tag');
      const statusText = await statusBadge.textContent().catch(() => '');

      if (statusText.toLowerCase().includes('paused')) {
        // Get campaign name
        const nameCell = row.locator('td').nth(1);
        const campaignName = await nameCell.textContent().catch(() => 'Unknown');

        // Get progress text
        const progressCell = row.locator('td').nth(4); // Adjust index if needed
        const progressText = await progressCell.textContent().catch(() => 'N/A');

        console.log(`Found PAUSED campaign: ${campaignName.trim()}`);
        console.log(`  Status: ${statusText.trim()}`);
        console.log(`  Progress: ${progressText.trim()}`);

        // Check if it's using queue (messenger = automatic)
        const messengerCell = row.locator('td').nth(3);
        const messengerText = await messengerCell.textContent().catch(() => '');
        console.log(`  Messenger: ${messengerText.trim()}\n`);

        if (messengerText.toLowerCase().includes('automatic') || messengerText.toLowerCase().includes('queue')) {
          foundPausedQueueCampaign = true;

          // Extract the numbers from progress text
          const match = progressText.match(/(\d+)\s*\/\s*(\d+)/);
          if (match) {
            const sent = parseInt(match[1]);
            const total = parseInt(match[2]);

            console.log(`📊 Progress Bar Analysis:`);
            console.log(`   Sent: ${sent}`);
            console.log(`   Total: ${total}`);

            if (sent === 21934) {
              console.log('   ❌ PROBLEM: Showing Azure delivered count (21934)');
              console.log('   Expected: Should show queue_sent count (~44173)\n');
            } else if (sent === 44173) {
              console.log('   ✅ SUCCESS: Showing correct queue_sent count!\n');
            } else {
              console.log(`   ⚠️  UNEXPECTED: Showing ${sent} (neither Azure nor expected queue count)\n`);
            }
          }
        }
      }
    }

    if (!foundPausedQueueCampaign) {
      console.log('⚠️  No paused queue-based campaigns found\n');
    }

    // Analyze API responses
    console.log('5. API Response Analysis:');
    for (const resp of apiResponses) {
      console.log(`\n   URL: ${resp.url}`);

      if (resp.url.includes('/api/campaigns?')) {
        // Main campaigns list
        if (resp.data && resp.data.data && resp.data.data.results) {
          console.log(`   Found ${resp.data.data.results.length} campaigns`);
        }
      } else if (resp.url.includes('/api/campaigns/running/stats')) {
        // Queue stats endpoint
        console.log('   Queue Stats Response:');
        if (resp.data && resp.data.data) {
          resp.data.data.forEach(stat => {
            if (stat.status === 'paused' && stat.use_queue) {
              console.log(`     Campaign ${stat.id}:`);
              console.log(`       queue_sent: ${stat.queue_sent || 'MISSING'}`);
              console.log(`       queue_total: ${stat.queue_total || 'MISSING'}`);
              console.log(`       azure_sent: ${stat.azure_sent || 0}`);
            }
          });
        }
      }
    }

    // Take screenshot
    console.log('\n6. Taking screenshot...');
    await page.screenshot({
      path: 'test-results/progress-bar-bobby.png',
      fullPage: true
    });
    console.log('✓ Screenshot saved to test-results/progress-bar-bobby.png\n');

    // Check if test-results directory exists
    const fs = require('fs');
    if (!fs.existsSync('test-results')) {
      fs.mkdirSync('test-results', { recursive: true });
    }

  } catch (error) {
    console.error('\n❌ Error during test:', error.message);
    console.error(error.stack);
  } finally {
    await browser.close();
  }
}

testProgressBarFix();
