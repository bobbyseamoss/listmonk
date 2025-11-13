const { chromium, firefox } = require('playwright');

async function testProgressBarFix(browserType, browserName) {
  console.log(`\n=== Testing in ${browserName} ===`);

  const browser = await browserType.launch({
    headless: false
  });
  const context = await browser.newContext({
    ignoreHTTPSErrors: true
  });
  const page = await context.newPage();

  try {
    // Login to Bobby Seamoss
    console.log('1. Logging in...');
    await page.goto('https://list.bobbyseamoss.com/admin/login');
    await page.fill('input[name="username"]', 'adam');
    await page.fill('input[name="password"]', 'bobbysea');
    await page.click('button[type="submit"]');
    
    try {
      await page.waitForURL('**/admin/**', { timeout: 10000 });
      console.log('✓ Logged in successfully');
    } catch (e) {
      const currentUrl = page.url();
      if (currentUrl.includes('/admin/login')) {
        throw new Error('Login failed');
      }
    }

    // Navigate to Campaigns page
    console.log('2. Loading Campaigns page...');
    await page.goto('https://list.bobbyseamoss.com/admin/campaigns');
    await page.waitForTimeout(3000);
    await page.waitForSelector('table tbody tr', { timeout: 10000 });
    console.log('✓ Campaigns page loaded');

    // Find campaign 65 using selector
    console.log('3. Finding campaign 65...');
    const rows = await page.locator('table tbody tr').all();
    console.log(`Found ${rows.length} campaign rows\n`);

    let foundCampaign = false;
    let progressText = '';

    for (const row of rows) {
      const cells = await row.locator('td').all();
      if (cells.length === 0) continue;

      // Get ID from first cell
      const idText = await cells[0].textContent();
      
      if (idText.trim() === '65') {
        foundCampaign = true;

        // Get campaign name
        const name = await cells[1].textContent();
        
        // Get status  
        const statusTag = row.locator('.tag');
        const status = await statusTag.textContent().catch(() => '');

        // Find progress text (pattern: number / number)
        for (const cell of cells) {
          const text = await cell.textContent();
          if (text.match(/\d+\s*\/\s*\d+/)) {
            progressText = text.trim();
            break;
          }
        }

        console.log('=== Campaign 65 Found ===');
        console.log(`Name: ${name.trim()}`);
        console.log(`Status: ${status.trim()}`);
        console.log(`Progress: ${progressText}`);
        console.log('');

        // Verify the fix
        const match = progressText.match(/(\d+)\s*\/\s*(\d+)/);
        if (match) {
          const sent = parseInt(match[1]);
          const total = parseInt(match[2]);

          console.log('=== VERIFICATION ===');
          console.log(`Sent count: ${sent}`);
          console.log(`Total count: ${total}\n`);

          if (sent === 21934) {
            console.log(`❌ FAILED in ${browserName}`);
            console.log('Still showing Azure delivered count (21934)');
            console.log('Expected: queue_sent count (around 44173)');
            return false;
          } else if (sent >= 44000 && sent <= 45000) {
            console.log(`✅ SUCCESS in ${browserName}`);
            console.log('Showing correct queue_sent count!');
            return true;
          } else {
            console.log(`⚠️  UNEXPECTED in ${browserName}: ${sent}`);
            console.log('Expected: around 44173');
            return false;
          }
        } else {
          console.log('⚠️  Could not parse progress text');
          return false;
        }
      }
    }

    if (!foundCampaign) {
      console.log(`❌ Campaign 65 not found in ${browserName}`);
      return false;
    }

    return false;

  } catch (error) {
    console.error(`\n❌ Error in ${browserName}:`, error.message);
    return false;
  } finally {
    // Take screenshot before closing
    await page.screenshot({
      path: `test-results/${browserName.toLowerCase()}-campaigns.png`,
      fullPage: true
    }).catch(() => {});
    
    await browser.close();
  }
}

// Run tests in both browsers
(async () => {
  // Ensure test-results directory exists
  const fs = require('fs');
  const path = require('path');
  const resultsDir = path.join(__dirname, 'test-results');
  if (!fs.existsSync(resultsDir)) {
    fs.mkdirSync(resultsDir, { recursive: true });
  }

  console.log('=== Testing Progress Bar Fix on Bobby Seamoss ===');
  console.log('Testing in both Chrome and Firefox...\n');

  const chromeResult = await testProgressBarFix(chromium, 'Chrome');
  const firefoxResult = await testProgressBarFix(firefox, 'Firefox');

  console.log('\n=== FINAL RESULTS ===');
  console.log(`Chrome:  ${chromeResult ? '✅ PASS' : '❌ FAIL'}`);
  console.log(`Firefox: ${firefoxResult ? '✅ PASS' : '❌ FAIL'}`);
  console.log(`\nOverall: ${chromeResult && firefoxResult ? '✅ ALL TESTS PASSED' : '❌ SOME TESTS FAILED'}`);
})();
