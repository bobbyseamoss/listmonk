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
    await page.waitForTimeout(2000);

    // Intercept API calls
    const apiResponses = [];
    page.on('response', async (response) => {
      const url = response.url();
      if (url.includes('/api/campaigns') && !url.includes('/api/campaigns/')) {
        const contentType = response.headers()['content-type'] || '';
        if (contentType.includes('application/json')) {
          try {
            const body = await response.json();
            apiResponses.push({
              url,
              status: response.status(),
              body
            });
          } catch (e) {
            // Ignore non-JSON responses
          }
        }
      }
    });

    // Go to campaigns page
    await page.goto('https://list.bobbyseamoss.com/admin/campaigns', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(5000);

    console.log('\n=== API RESPONSES ===');
    apiResponses.forEach((resp, i) => {
      console.log(`\nResponse ${i + 1}:`);
      console.log(`URL: ${resp.url}`);
      console.log(`Status: ${resp.status}`);

      if (resp.body && resp.body.data && resp.body.data.results) {
        const campaigns = resp.body.data.results;
        console.log(`\nFound ${campaigns.length} campaigns`);

        // Find paused campaign
        const pausedCampaign = campaigns.find(c => c.status === 'paused');
        if (pausedCampaign) {
          console.log('\n=== PAUSED CAMPAIGN DATA ===');
          console.log(`ID: ${pausedCampaign.id}`);
          console.log(`Name: ${pausedCampaign.name}`);
          console.log(`Status: ${pausedCampaign.status}`);
          console.log(`Sent: ${pausedCampaign.sent}`);
          console.log(`To Send: ${pausedCampaign.to_send}`);
          console.log(`Azure Sent: ${pausedCampaign.azure_sent !== undefined ? pausedCampaign.azure_sent : 'MISSING'}`);
          console.log(`Views: ${pausedCampaign.views}`);
          console.log(`Clicks: ${pausedCampaign.clicks}`);
        }
      }
    });

  } catch (error) {
    console.error('Error:', error);
  } finally {
    await browser.close();
  }
})();
