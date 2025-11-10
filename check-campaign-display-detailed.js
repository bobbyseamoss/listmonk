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

    // Go to campaigns page
    await page.goto('https://list.bobbyseamoss.com/admin/campaigns', { waitUntil: 'domcontentloaded' });
    await page.waitForTimeout(5000);

    // Get the actual HTML for campaign 64's purchase column
    const purchaseData = await page.evaluate(() => {
      const rows = Array.from(document.querySelectorAll('tbody tr'));
      for (const row of rows) {
        const nameCell = row.querySelector('td:nth-child(2)');
        if (nameCell && nameCell.textContent.includes('Gummies New')) {
          // Find the purchase column (should be around column 7-8)
          const cells = Array.from(row.querySelectorAll('td'));
          const purchaseCell = cells.find(cell => {
            const label = cell.querySelector('label');
            return label && label.textContent.includes('$');
          });

          if (purchaseCell) {
            const label = purchaseCell.querySelector('label');
            const span = purchaseCell.querySelector('span');
            return {
              found: true,
              labelText: label ? label.textContent : 'NO LABEL',
              spanText: span ? span.textContent : 'NO SPAN',
              fullHTML: purchaseCell.innerHTML,
              allCellsText: cells.map(c => c.textContent.trim()).join(' | ')
            };
          }
        }
      }
      return { found: false };
    });

    console.log('\n=== PURCHASE DISPLAY FOR CAMPAIGN 64 ===');
    console.log(JSON.stringify(purchaseData, null, 2));

    // Also check Vue component data
    const vueData = await page.evaluate(() => {
      const app = window.__vue__;
      if (app && app.$children && app.$children[0]) {
        const campaignsView = app.$children[0].$children.find(c => c.campaigns);
        if (campaignsView && campaignsView.campaigns && campaignsView.campaigns.results) {
          const campaign64 = campaignsView.campaigns.results.find(c => c.id === 64);
          if (campaign64) {
            return {
              found: true,
              purchase_orders: campaign64.purchase_orders,
              purchase_revenue: campaign64.purchase_revenue,
              purchase_currency: campaign64.purchase_currency,
              typeOf_purchase_orders: typeof campaign64.purchase_orders,
              typeOf_purchase_revenue: typeof campaign64.purchase_revenue
            };
          }
        }
      }
      return { found: false, reason: 'Could not access Vue data' };
    });

    console.log('\n=== VUE COMPONENT DATA FOR CAMPAIGN 64 ===');
    console.log(JSON.stringify(vueData, null, 2));

  } catch (error) {
    console.error('Error:', error);
  } finally {
    await browser.close();
  }
})();
