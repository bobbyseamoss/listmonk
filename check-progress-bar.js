const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext();
  const page = await context.newPage();

  try {
    // Login
    await page.goto('https://list.bobbyseamoss.com/admin/login', { waitUntil: 'domcontentloaded' });
    await page.fill('input[name="username"]', 'adam');
    await page.fill('input[name="password"]', 'T@intshr3dd3r');
    await page.click('button[type="submit"]');

    // Wait for navigation to complete
    await page.waitForTimeout(2000);

    // Go to campaigns page
    await page.goto('https://list.bobbyseamoss.com/admin/campaigns', { waitUntil: 'domcontentloaded' });

    // Wait for table to be visible
    await page.waitForSelector('tbody tr', { timeout: 5000 });

    // Wait a bit for any polling to complete
    await page.waitForTimeout(3000);

    // Find campaign 64 - look for the paused campaign
    const campaigns = await page.$$eval('tbody tr', rows => {
      return rows.map(row => {
        const nameCell = row.querySelector('td:nth-child(2)');
        const statusCell = row.querySelector('td:nth-child(1)');
        const progressBar = row.querySelector('.campaign-progress');

        if (!nameCell || !statusCell) return null;

        const name = nameCell.textContent.trim();
        const status = statusCell.textContent.trim();
        const progressText = progressBar ? progressBar.textContent.trim() : 'No progress bar';

        return {
          name,
          status,
          progressText
        };
      }).filter(c => c !== null);
    });

    console.log('\n=== Campaigns on Page ===');
    campaigns.forEach((c, i) => {
      console.log(`\nCampaign ${i + 1}:`);
      console.log(`  Name: ${c.name}`);
      console.log(`  Status: ${c.status}`);
      console.log(`  Progress: ${c.progressText}`);
    });

    // Find the paused campaign specifically
    const pausedCampaign = campaigns.find(c => c.status.includes('Paused') || c.status.includes('paused'));

    if (pausedCampaign) {
      console.log('\n=== PAUSED CAMPAIGN (Campaign 64) ===');
      console.log(`Name: ${pausedCampaign.name}`);
      console.log(`Status: ${pausedCampaign.status}`);
      console.log(`Progress Bar Text: ${pausedCampaign.progressText}`);

      // Check if it shows 51,311
      if (pausedCampaign.progressText.includes('51')) {
        console.log('\n✅ CORRECT: Progress bar shows ~51k sent');
      } else if (pausedCampaign.progressText.includes('0')) {
        console.log('\n❌ WRONG: Progress bar still shows 0 sent');
      } else {
        console.log('\n⚠️  UNCLEAR: Progress bar shows unexpected value');
      }
    } else {
      console.log('\n⚠️  Could not find paused campaign');
    }

  } catch (error) {
    console.error('Error:', error);
  } finally {
    await browser.close();
  }
})();
