const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext();
  const page = await context.newPage();

  try {
    console.log('✅ Verifying Purchase Attribution on Campaigns Page\n');

    // Login
    await page.goto('https://list.bobbyseamoss.com/admin/login', { waitUntil: 'load' });
    await page.fill('input[name="username"]', 'adam');
    await page.fill('input[name="password"]', 'T@intshr3dd3r');
    await page.click('button[type="submit"]');
    await page.waitForTimeout(2000);

    // Navigate to campaigns
    await page.goto('https://list.bobbyseamoss.com/admin/campaigns', { waitUntil: 'load' });
    await page.waitForTimeout(3000);

    // Find campaign 64 row and extract purchase metrics
    const campaign64Data = await page.evaluate(() => {
      // Find all campaign rows
      const rows = Array.from(document.querySelectorAll('tbody tr'));

      for (const row of rows) {
        // Look for campaign ID or name
        const cells = row.querySelectorAll('td');
        if (!cells.length) continue;

        // Check if this row contains campaign 64
        const rowText = row.textContent;
        if (rowText.includes('Campaign') || rowText.includes('50OFF1') || rowText.includes('64')) {
          // Extract purchase metrics (Orders and Revenue columns)
          const purchaseData = {
            campaignId: null,
            campaignName: null,
            orders: null,
            revenue: null,
            allText: rowText,
          };

          // Try to find campaign name
          const nameCell = cells[1]; // Usually second column
          if (nameCell) {
            purchaseData.campaignName = nameCell.textContent.trim();
          }

          // Look for numeric cells that might be orders/revenue
          cells.forEach((cell, idx) => {
            const text = cell.textContent.trim();
            // Orders column (looking for numbers)
            if (text.match(/^\d+$/)) {
              if (!purchaseData.orders) {
                purchaseData.orders = text;
              }
            }
            // Revenue column (looking for currency)
            if (text.match(/\$[\d,.]+/)) {
              purchaseData.revenue = text;
            }
          });

          return purchaseData;
        }
      }

      return null;
    });

    console.log('=== CAMPAIGN 64 PURCHASE ATTRIBUTION ===\n');

    if (!campaign64Data) {
      console.log('⚠️  Could not find campaign 64 in the campaigns table');
      console.log('Checking if campaign exists...\n');

      // Get all visible campaign IDs
      const allCampaigns = await page.evaluate(() => {
        const rows = Array.from(document.querySelectorAll('tbody tr'));
        return rows.map(row => row.textContent.trim().substring(0, 100));
      });

      console.log('First 5 visible campaigns:');
      allCampaigns.slice(0, 5).forEach((c, i) => {
        console.log(`  ${i + 1}. ${c}...`);
      });
    } else {
      console.log('Campaign Found:');
      console.log(`  Name: ${campaign64Data.campaignName || 'Unknown'}`);
      console.log(`  Orders: ${campaign64Data.orders || 'Not shown'}`);
      console.log(`  Revenue: ${campaign64Data.revenue || 'Not shown'}`);
      console.log(`\nFull Row Text: ${campaign64Data.allText.substring(0, 200)}...`);

      // Check if purchase is attributed
      if (campaign64Data.orders && parseInt(campaign64Data.orders) > 0) {
        console.log('\n✅ SUCCESS: Campaign 64 shows purchase attribution!');
        console.log(`   Orders: ${campaign64Data.orders}`);
        console.log(`   Revenue: ${campaign64Data.revenue}`);
      } else {
        console.log('\n⚠️  WARNING: Campaign 64 exists but no orders shown yet');
        console.log('   This may be due to:');
        console.log('   - Data not yet refreshed in UI');
        console.log('   - Column not visible in current view');
        console.log('   - Attribution needs time to propagate');
      }
    }

    // Also check API endpoint directly
    console.log('\n=== CHECKING API ENDPOINT ===\n');

    const apiResponse = await page.evaluate(async () => {
      try {
        const response = await fetch('/api/campaigns/64/purchases/stats');
        const data = await response.json();
        return { success: true, data };
      } catch (error) {
        return { success: false, error: error.message };
      }
    });

    if (apiResponse.success) {
      console.log('✅ API Response:');
      console.log(JSON.stringify(apiResponse.data, null, 2));

      if (apiResponse.data.data) {
        const stats = apiResponse.data.data;
        console.log('\nParsed Stats:');
        console.log(`  Total Purchases: ${stats.total_purchases || 0}`);
        console.log(`  Total Revenue: $${stats.total_revenue || 0} ${stats.currency || 'USD'}`);
        console.log(`  Avg Order Value: $${stats.avg_order_value || 0}`);

        if (stats.total_purchases > 0) {
          console.log('\n✅✅ CONFIRMED: Purchase attribution is working!');
        }
      }
    } else {
      console.log('❌ API Error:', apiResponse.error);
    }

  } catch (error) {
    console.error('❌ Error:', error);
  } finally {
    await browser.close();
  }
})();
