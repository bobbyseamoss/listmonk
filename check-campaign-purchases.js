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

        // Find campaign 64
        const campaign64 = campaigns.find(c => c.id === 64);
        if (campaign64) {
          console.log('\n=== CAMPAIGN 64 DATA FROM API ===');
          console.log(`ID: ${campaign64.id}`);
          console.log(`Name: ${campaign64.name}`);
          console.log(`Status: ${campaign64.status}`);
          console.log(`Total Purchases: ${campaign64.total_purchases !== undefined ? campaign64.total_purchases : 'MISSING'}`);
          console.log(`Total Revenue: ${campaign64.total_revenue !== undefined ? campaign64.total_revenue : 'MISSING'}`);
          console.log(`Purchase Stats: ${JSON.stringify(campaign64.purchase_stats || 'MISSING')}`);
          console.log('\nFull Campaign 64 Object:');
          console.log(JSON.stringify(campaign64, null, 2));
        }
      }
    });

    // Also check what's displayed on the page
    console.log('\n=== PAGE DISPLAY ===');
    const campaignRows = await page.$$('tbody tr');
    for (const row of campaignRows) {
      const nameCell = await row.$('td:nth-child(2)');
      if (nameCell) {
        const name = await nameCell.textContent();
        if (name && name.includes('Subscriber')) {
          console.log('\nFound campaign row with "Subscriber" in name');
          const rowText = await row.textContent();
          console.log(`Row text: ${rowText}`);
        }
      }
    }

  } catch (error) {
    console.error('Error:', error);
  } finally {
    await browser.close();
  }
})();
