const { chromium } = require('playwright');

async function verifyProgressBarFix() {
  console.log('=== Verifying Progress Bar Fix ===\n');

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
    await page.waitForTimeout(3000); // Wait for data to load
    console.log('✓ Campaigns page loaded\n');

    // Find campaign 65 (paused queue-based campaign)
    console.log('3. Looking for campaign 65...');
    const campaignRows = await page.locator('table tbody tr').all();
    console.log(`Found ${campaignRows.length} campaigns\n`);

    let foundCampaign65 = false;
    let progressText = '';

    for (let i = 0; i < campaignRows.length; i++) {
      const row = campaignRows[i];

      // Get campaign ID from the first cell
      const idCell = row.locator('td').nth(0);
      const idText = await idCell.textContent().catch(() => '');

      if (idText.includes('65')) {
        foundCampaign65 = true;

        // Get campaign name
        const nameCell = row.locator('td').nth(1);
        const campaignName = await nameCell.textContent().catch(() => 'Unknown');

        // Get status
        const statusBadge = row.locator('.tag');
        const statusText = await statusBadge.textContent().catch(() => '');

        // Get progress text (should be in one of the cells)
        const cells = await row.locator('td').all();
        for (let j = 0; j < cells.length; j++) {
          const text = await cells[j].textContent().catch(() => '');
          // Look for pattern like "44173 / 78190"
          if (text.match(/\d+\s*\/\s*\d+/)) {
            progressText = text.trim();
            break;
          }
        }

        console.log(`Found Campaign 65:`);
        console.log(`  Name: ${campaignName.trim()}`);
        console.log(`  Status: ${statusText.trim()}`);
        console.log(`  Progress: ${progressText}`);
        console.log('');

        // Verify the fix
        const match = progressText.match(/(\d+)\s*\/\s*(\d+)/);
        if (match) {
          const sent = parseInt(match[1]);
          const total = parseInt(match[2]);

          console.log('=== VERIFICATION ===');
          console.log(`Sent count: ${sent}`);
          console.log(`Total count: ${total}`);
          console.log('');

          if (sent === 21934) {
            console.log('❌ FAILED: Still showing Azure delivered count (21934)');
            console.log('Expected: Should show queue_sent count (around 44173)');
          } else if (sent >= 44000 && sent <= 45000) {
            console.log('✅ SUCCESS: Showing correct queue_sent count!');
            console.log('The fix is working correctly.');
          } else {
            console.log(`⚠️  UNEXPECTED: Showing ${sent}`);
            console.log('Expected: Around 44173 (queue_sent count)');
          }
        } else {
          console.log('⚠️  Could not parse progress text');
        }

        break;
      }
    }

    if (!foundCampaign65) {
      console.log('❌ Campaign 65 not found on the page');
    }

    // Take screenshot
    console.log('\n4. Taking screenshot...');
    await page.screenshot({
      path: 'screenshots/verify-progress-bar-fix.png',
      fullPage: true
    });
    console.log('✓ Screenshot saved to screenshots/verify-progress-bar-fix.png\n');

    // Keep browser open for inspection
    console.log('Keeping browser open for 5 seconds...');
    await page.waitForTimeout(5000);

  } catch (error) {
    console.error('\n❌ Error during verification:', error.message);
    console.error(error.stack);
  } finally {
    await browser.close();
  }
}

verifyProgressBarFix();
