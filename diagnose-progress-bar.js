const { chromium } = require('playwright');

async function diagnoseProgressBar() {
  console.log('\n🔍 Diagnosing Campaign Progress Bar Visibility Issue\n');
  console.log('=' .repeat(70));

  const browser = await chromium.launch({ headless: false });
  const context = await browser.newContext({ ignoreHTTPSErrors: true });
  const page = await context.newPage();

  // Enable console logging from the page
  page.on('console', msg => console.log('PAGE LOG:', msg.text()));
  page.on('pageerror', error => console.log('PAGE ERROR:', error.message));

  try {
    // Login
    const baseUrl = 'https://list.bobbyseamoss.com';
    console.log(`\n📍 Navigating to ${baseUrl}...`);
    await page.goto(`${baseUrl}/admin/login`);

    await page.fill('input[name="username"]', 'adam');
    await page.fill('input[name="password"]', 'bobbysea');
    await page.click('button[type="submit"]');

    // Wait for either dashboard or campaigns (in case redirect differs)
    try {
      await page.waitForURL('**/admin/**', { timeout: 10000 });
      console.log('✅ Logged in successfully');
    } catch (e) {
      console.log('⚠️  Navigation timeout, but may still be logged in');
      await page.waitForTimeout(2000);
    }

    // Navigate to campaigns
    console.log('\n📍 Navigating to campaigns page...');
    await page.goto(`${baseUrl}/admin/campaigns`);
    await page.waitForTimeout(3000);

    // Check for campaigns
    const campaignRows = await page.locator('table tbody tr').count();
    console.log(`\n📊 Found ${campaignRows} campaign rows`);

    if (campaignRows === 0) {
      console.log('❌ No campaigns found. Cannot test progress bar.');
      await browser.close();
      return;
    }

    // Check for progress bar elements
    console.log('\n🔍 Searching for progress bar elements...\n');

    const progressContainers = await page.locator('.campaign-progress').count();
    console.log(`Progress containers (.campaign-progress): ${progressContainers}`);

    if (progressContainers === 0) {
      console.log('\n⚠️  No .campaign-progress elements found!');
      console.log('This means either:');
      console.log('  1. No campaigns have status: running, paused, cancelled, or finished');
      console.log('  2. The showProgress() method is returning false');
      console.log('  3. The v-if condition is not being met');

      // Check campaign statuses
      console.log('\n🔍 Checking campaign statuses...');
      const statuses = await page.evaluate(() => {
        const rows = document.querySelectorAll('table tbody tr');
        return Array.from(rows).slice(0, 5).map(row => {
          const statusBadge = row.querySelector('.tag');
          return statusBadge ? statusBadge.textContent.trim() : 'unknown';
        });
      });
      console.log('First 5 campaign statuses:', statuses);
    } else {
      // Inspect each progress container
      for (let i = 0; i < Math.min(3, progressContainers); i++) {
        console.log(`\n--- Progress Container #${i + 1} ---`);

        const container = page.locator('.campaign-progress').nth(i);

        // Check visibility
        const isVisible = await container.isVisible();
        console.log(`  Visible: ${isVisible ? '✅ YES' : '❌ NO'}`);

        // Get bounding box
        const bbox = await container.boundingBox();
        console.log(`  Bounding Box:`, bbox);

        // Get computed styles
        const styles = await container.evaluate((el) => {
          const computed = window.getComputedStyle(el);
          return {
            display: computed.display,
            visibility: computed.visibility,
            opacity: computed.opacity,
            height: computed.height,
            minHeight: computed.minHeight,
            width: computed.width,
            position: computed.position,
            overflow: computed.overflow,
          };
        });
        console.log(`  Computed Styles:`, styles);

        // Check for nested progress element
        const progressElement = container.locator('.progress');
        const progressCount = await progressElement.count();
        console.log(`  Nested .progress elements: ${progressCount}`);

        if (progressCount > 0) {
          const progressVisible = await progressElement.first().isVisible();
          const progressBox = await progressElement.first().boundingBox();
          const progressStyles = await progressElement.first().evaluate((el) => {
            const computed = window.getComputedStyle(el);
            return {
              display: computed.display,
              height: computed.height,
              width: computed.width,
              visibility: computed.visibility,
              opacity: computed.opacity,
            };
          });

          console.log(`  .progress visible: ${progressVisible ? '✅ YES' : '❌ NO'}`);
          console.log(`  .progress bbox:`, progressBox);
          console.log(`  .progress styles:`, progressStyles);

          // Get the HTML to see the structure
          const html = await container.evaluate(el => el.outerHTML);
          console.log(`  HTML (first 300 chars):\n    ${html.substring(0, 300)}...`);
        }
      }
    }

    // Check if the CSS file was loaded
    console.log('\n🔍 Checking loaded stylesheets...');
    const stylesheets = await page.evaluate(() => {
      return Array.from(document.styleSheets)
        .filter(sheet => {
          try {
            return sheet.href && sheet.href.includes('Campaigns');
          } catch (e) {
            return false;
          }
        })
        .map(sheet => ({
          href: sheet.href,
          cssRules: sheet.cssRules ? sheet.cssRules.length : 0
        }));
    });
    console.log('Campaigns stylesheets:', stylesheets);

    // Check if the specific CSS rules exist
    console.log('\n🔍 Checking for campaign-progress CSS rules...');
    const hasProgressCSS = await page.evaluate(() => {
      const allStyles = Array.from(document.styleSheets);
      for (const sheet of allStyles) {
        try {
          const rules = Array.from(sheet.cssRules || []);
          for (const rule of rules) {
            if (rule.cssText && rule.cssText.includes('campaign-progress')) {
              return true;
            }
          }
        } catch (e) {
          // CORS error, skip
        }
      }
      return false;
    });
    console.log(`CSS rules for campaign-progress found: ${hasProgressCSS ? '✅ YES' : '❌ NO'}`);

    // Take a screenshot
    console.log('\n📸 Taking screenshot...');
    await page.screenshot({ path: '/home/adam/listmonk/chrome-campaigns-screenshot.png', fullPage: true });
    console.log('✅ Screenshot saved to: /home/adam/listmonk/chrome-campaigns-screenshot.png');

    console.log('\n' + '='.repeat(70));
    console.log('\n💡 Diagnosis complete. Press Ctrl+C to exit or wait 30 seconds...\n');
    await page.waitForTimeout(30000);

  } catch (error) {
    console.error('\n❌ Error during diagnosis:', error.message);
    console.error(error.stack);
  } finally {
    await browser.close();
  }
}

diagnoseProgressBar().catch(console.error);
