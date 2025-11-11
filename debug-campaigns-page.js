const { chromium } = require('playwright');

const BOBBY_SEAMOSS = {
  url: 'https://list.bobbyseamoss.com',
  username: 'adam',
  password: 'T@intshr3dd3r'
};

async function debugCampaignsPage() {
  console.log('\n🔍 Debugging Campaigns Page\n');

  const browser = await chromium.launch({ headless: false });
  const context = await browser.newContext({ ignoreHTTPSErrors: true });
  const page = await context.newPage();

  try {
    // Login
    await page.goto(`${BOBBY_SEAMOSS.url}/admin/login`);
    await page.fill('input[name="username"]', BOBBY_SEAMOSS.username);
    await page.fill('input[name="password"]', BOBBY_SEAMOSS.password);
    await page.click('button[type="submit"]');
    await page.waitForURL('**/admin/**', { timeout: 10000 }).catch(() => {});
    await page.waitForTimeout(2000);

    // Navigate to campaigns
    await page.goto(`${BOBBY_SEAMOSS.url}/admin/campaigns`);
    await page.waitForTimeout(5000); // Wait for data to load

    // Debug campaign data
    const campaignData = await page.evaluate(() => {
      const campaigns = [];
      const rows = document.querySelectorAll('table tbody tr');

      rows.forEach((row, idx) => {
        // Get campaign name
        const nameCell = row.querySelector('td:nth-child(2)'); // Usually second column
        const name = nameCell ? nameCell.textContent.trim() : 'unknown';

        // Get status badge
        const statusBadge = row.querySelector('.tag');
        const status = statusBadge ? statusBadge.textContent.trim() : 'unknown';

        // Check for progress bar
        const progressBar = row.querySelector('.campaign-progress');
        const hasProgressBar = !!progressBar;

        // Get all content from the name cell
        const nameCellHtml = nameCell ? nameCell.innerHTML : 'N/A';

        campaigns.push({
          index: idx + 1,
          name,
          status,
          hasProgressBar,
          nameCellPreview: nameCellHtml.substring(0, 200)
        });
      });

      return {
        totalRows: rows.length,
        campaigns
      };
    });

    console.log('📊 Campaigns Page Data:');
    console.log(`   Total campaign rows: ${campaignData.totalRows}\n`);

    campaignData.campaigns.slice(0, 10).forEach(c => {
      const icon = c.hasProgressBar ? '✅' : '❌';
      console.log(`${icon} Campaign #${c.index}`);
      console.log(`   Name: ${c.name.substring(0, 50)}`);
      console.log(`   Status: ${c.status}`);
      console.log(`   Has Progress Bar: ${c.hasProgressBar}`);
      console.log('');
    });

    // Check if the campaign-progress class exists in the DOM at all
    const allProgressElements = await page.evaluate(() => {
      return document.querySelectorAll('.campaign-progress').length;
    });
    console.log(`\n🔍 Total .campaign-progress elements in DOM: ${allProgressElements}`);

    // Check for specific campaign IDs we know should have progress
    const specificCampaigns = await page.evaluate(() => {
      const results = {};
      const targetIds = [63, 64, 65];

      targetIds.forEach(id => {
        // Look for any element that might reference this campaign ID
        const elements = document.querySelectorAll(`[data-id="${id}"], #campaign-${id}, tr`);
        results[id] = {
          found: elements.length > 0,
          count: elements.length
        };
      });

      return results;
    });

    console.log('\n🎯 Looking for specific campaigns (63, 64, 65):');
    Object.entries(specificCampaigns).forEach(([id, data]) => {
      console.log(`   Campaign ${id}: ${data.found ? `Found (${data.count} elements)` : 'Not found'}`);
    });

    // Take screenshot
    await page.screenshot({
      path: '/home/adam/listmonk/screenshots/debug-campaigns.png',
      fullPage: true
    });
    console.log('\n📸 Screenshot: /home/adam/listmonk/screenshots/debug-campaigns.png');

    // Keep browser open
    console.log('\n⏸️  Browser staying open for 30 seconds for manual inspection...');
    await page.waitForTimeout(30000);

  } catch (error) {
    console.error('\n❌ Error:', error.message);
  } finally {
    await browser.close();
  }
}

debugCampaignsPage().catch(console.error);
