const { chromium, firefox } = require('playwright');

async function testProgressBarVisibility(browserType, browserName) {
  console.log(`\n=== Testing in ${browserName} ===`);

  const browser = await browserType.launch({
    headless: false,
    args: ['--disable-web-security'] // For testing across origins
  });
  const context = await browser.newContext({
    ignoreHTTPSErrors: true
  });
  const page = await context.newPage();

  try {
    // Login - use list.bobbyseamoss.com
    const baseUrl = 'https://list.bobbyseamoss.com';
    await page.goto(`${baseUrl}/admin/login`);

    await page.fill('input[name="username"]', 'adam');
    await page.fill('input[name="password"]', 'bobbysea');
    await page.click('button[type="submit"]');
    await page.waitForURL('**/admin/dashboard');

    // Navigate to campaigns
    await page.goto(`${baseUrl}/admin/campaigns`);
    await page.waitForTimeout(3000);

    // Find a campaign with progress bar
    const progressBars = await page.locator('.campaign-progress').all();
    console.log(`Found ${progressBars.length} progress bars`);

    if (progressBars.length > 0) {
      for (let i = 0; i < Math.min(3, progressBars.length); i++) {
        const progressBar = progressBars[i];

        // Check visibility
        const isVisible = await progressBar.isVisible();
        console.log(`Progress bar ${i + 1}: visible = ${isVisible}`);

        // Get computed styles
        const bbox = await progressBar.boundingBox();
        console.log(`Progress bar ${i + 1} bounding box:`, bbox);

        // Get the actual progress element
        const progressElement = progressBar.locator('.progress').first();
        const progressVisible = await progressElement.isVisible();
        const progressBox = await progressElement.boundingBox();
        console.log(`  .progress element: visible = ${progressVisible}, bbox =`, progressBox);

        // Get computed styles
        const styles = await progressBar.evaluate((el) => {
          const computed = window.getComputedStyle(el);
          return {
            display: computed.display,
            visibility: computed.visibility,
            opacity: computed.opacity,
            height: computed.height,
            width: computed.width,
            marginTop: computed.marginTop,
            marginBottom: computed.marginBottom,
          };
        });
        console.log(`  Container styles:`, styles);

        const progressStyles = await progressElement.evaluate((el) => {
          const computed = window.getComputedStyle(el);
          return {
            display: computed.display,
            visibility: computed.visibility,
            opacity: computed.opacity,
            height: computed.height,
            width: computed.width,
            marginBottom: computed.marginBottom,
          };
        });
        console.log(`  Progress element styles:`, progressStyles);

        // Check for the progress bar's inner elements
        const progressBar_el = progressElement.locator('.progress-bar').first();
        const progressBarExists = await progressBar_el.count() > 0;
        console.log(`  .progress-bar exists: ${progressBarExists}`);

        if (progressBarExists) {
          const progressBarBox = await progressBar_el.boundingBox();
          console.log(`  .progress-bar bbox:`, progressBarBox);
        }
      }
    } else {
      console.log('No progress bars found. Checking if campaigns exist with progress...');

      // Look for running/paused campaigns
      const campaignRows = await page.locator('table tbody tr').all();
      console.log(`Found ${campaignRows.length} campaign rows`);

      for (let i = 0; i < Math.min(3, campaignRows.length); i++) {
        const row = campaignRows[i];
        const html = await row.innerHTML();
        console.log(`\nCampaign row ${i + 1}:`);
        console.log(html.substring(0, 500));
      }
    }

    await page.waitForTimeout(5000); // Keep browser open to inspect

  } catch (error) {
    console.error(`Error in ${browserName}:`, error);
  } finally {
    await browser.close();
  }
}

(async () => {
  // Test in Chrome
  await testProgressBarVisibility(chromium, 'Chrome');

  // Test in Firefox
  await testProgressBarVisibility(firefox, 'Firefox');
})();
