const { chromium, firefox } = require('playwright');

// Bobby Seamoss credentials
const BOBBY_SEAMOSS = {
  url: 'https://list.bobbyseamoss.com',
  username: 'adam',
  password: 'T@intshr3dd3r'
};

async function testProgressBar(browserType, browserName) {
  console.log(`\n${'='.repeat(70)}`);
  console.log(`🌐 Testing in ${browserName}`);
  console.log('='.repeat(70));

  const browser = await browserType.launch({
    headless: false,
    args: ['--disable-web-security']
  });

  const context = await browser.newContext({
    ignoreHTTPSErrors: true
  });

  const page = await context.newPage();

  try {
    // Login
    console.log(`\n📍 Navigating to ${BOBBY_SEAMOSS.url}...`);
    await page.goto(`${BOBBY_SEAMOSS.url}/admin/login`);

    console.log('🔐 Logging in...');
    await page.fill('input[name="username"]', BOBBY_SEAMOSS.username);
    await page.fill('input[name="password"]', BOBBY_SEAMOSS.password);
    await page.click('button[type="submit"]');

    // Wait for navigation
    try {
      await page.waitForURL('**/admin/**', { timeout: 10000 });
      console.log('✅ Logged in successfully');
    } catch (e) {
      console.log('⚠️  Navigation timeout, checking if logged in...');
      await page.waitForTimeout(2000);
    }

    // Navigate to campaigns
    console.log('\n📍 Navigating to campaigns page...');
    await page.goto(`${BOBBY_SEAMOSS.url}/admin/campaigns`);
    await page.waitForTimeout(3000);

    // Check for progress bar elements
    console.log('\n🔍 Searching for campaign progress bars...');
    const progressContainers = await page.locator('.campaign-progress').count();
    console.log(`   Found ${progressContainers} progress bar containers`);

    if (progressContainers === 0) {
      console.log('\n⚠️  No progress bars found!');
      console.log('   This could mean:');
      console.log('   - No campaigns with progress (running/paused/cancelled/finished)');
      console.log('   - showProgress() is returning false');

      // Check campaign statuses
      const statuses = await page.evaluate(() => {
        const rows = document.querySelectorAll('table tbody tr');
        return Array.from(rows).slice(0, 5).map(row => {
          const statusBadge = row.querySelector('.tag');
          const nameCell = row.querySelector('td:first-child');
          return {
            name: nameCell ? nameCell.textContent.trim().substring(0, 30) : 'unknown',
            status: statusBadge ? statusBadge.textContent.trim() : 'unknown'
          };
        });
      });
      console.log('\n   First 5 campaigns:');
      statuses.forEach((c, i) => {
        console.log(`   ${i + 1}. ${c.name} - Status: ${c.status}`);
      });

    } else {
      // Test each progress bar
      console.log('\n📊 Testing progress bars:\n');

      for (let i = 0; i < Math.min(3, progressContainers); i++) {
        const container = page.locator('.campaign-progress').nth(i);

        // Get campaign name
        const campaignName = await page.evaluate((idx) => {
          const progressBar = document.querySelectorAll('.campaign-progress')[idx];
          if (!progressBar) return 'Unknown';
          const row = progressBar.closest('tr');
          if (!row) return 'Unknown';
          const nameCell = row.querySelector('td:first-child');
          return nameCell ? nameCell.textContent.trim().substring(0, 40) : 'Unknown';
        }, i);

        console.log(`   Campaign #${i + 1}: ${campaignName}`);

        // Check visibility
        const isVisible = await container.isVisible();
        const visibleIcon = isVisible ? '✅' : '❌';
        console.log(`     ${visibleIcon} Visible: ${isVisible}`);

        // Get bounding box
        const bbox = await container.boundingBox();
        if (bbox) {
          console.log(`     📐 Dimensions: ${Math.round(bbox.width)}x${Math.round(bbox.height)}px`);
          console.log(`     📍 Position: (${Math.round(bbox.x)}, ${Math.round(bbox.y)})`);
        } else {
          console.log(`     ❌ No bounding box (element may be hidden)`);
        }

        // Get computed styles
        const styles = await container.evaluate((el) => {
          const computed = window.getComputedStyle(el);
          return {
            display: computed.display,
            minHeight: computed.minHeight,
            height: computed.height,
            visibility: computed.visibility,
            opacity: computed.opacity,
          };
        });
        console.log(`     🎨 Styles:`, JSON.stringify(styles, null, 2).replace(/\n/g, '\n        '));

        // Check nested .progress element
        const progressElement = container.locator('.progress').first();
        const progressVisible = await progressElement.isVisible().catch(() => false);
        const progressStyles = await progressElement.evaluate((el) => {
          const computed = window.getComputedStyle(el);
          return {
            display: computed.display,
            height: computed.height,
            width: computed.width,
          };
        }).catch(() => null);

        console.log(`     ${progressVisible ? '✅' : '❌'} .progress element visible: ${progressVisible}`);
        if (progressStyles) {
          console.log(`     🎨 .progress styles:`, JSON.stringify(progressStyles, null, 2).replace(/\n/g, '\n        '));
        }

        console.log('');
      }
    }

    // Take screenshot
    const screenshotPath = `/home/adam/listmonk/screenshots/${browserName.toLowerCase()}-campaigns.png`;
    await page.screenshot({
      path: screenshotPath,
      fullPage: true
    });
    console.log(`📸 Screenshot saved: ${screenshotPath}\n`);

    // Summary
    console.log('='.repeat(70));
    if (progressContainers > 0) {
      console.log(`✅ ${browserName} TEST PASSED: Progress bars found and analyzed`);
    } else {
      console.log(`⚠️  ${browserName} TEST WARNING: No progress bars found`);
    }
    console.log('='.repeat(70));

    // Keep browser open for manual inspection
    console.log('\n⏸️  Browser will stay open for 15 seconds for manual inspection...');
    await page.waitForTimeout(15000);

  } catch (error) {
    console.error(`\n❌ Error in ${browserName}:`, error.message);
    console.error(error.stack);
  } finally {
    await browser.close();
  }
}

// Main execution
(async () => {
  // Create screenshots directory
  const fs = require('fs');
  const screenshotsDir = '/home/adam/listmonk/screenshots';
  if (!fs.existsSync(screenshotsDir)) {
    fs.mkdirSync(screenshotsDir, { recursive: true });
  }

  console.log('\n🎯 Campaign Progress Bar Test');
  console.log('📍 Environment: Bobby Seamoss (list.bobbyseamoss.com)');
  console.log('🎨 Testing CSS Fix: Vue :deep() syntax + explicit heights\n');

  // Test in Chrome first (stricter CSS rendering)
  await testProgressBar(chromium, 'Chrome');

  // Test in Firefox second
  await testProgressBar(firefox, 'Firefox');

  console.log('\n\n' + '='.repeat(70));
  console.log('🏁 TESTING COMPLETE');
  console.log('='.repeat(70));
  console.log('\n📸 Screenshots saved in: /home/adam/listmonk/screenshots/');
  console.log('   - chrome-campaigns.png');
  console.log('   - firefox-campaigns.png\n');

})().catch(console.error);
