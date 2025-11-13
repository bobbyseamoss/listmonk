const { chromium } = require('playwright');

async function debugProgressBar() {
  console.log('=== Debugging Progress Bar on Bobby Seamoss ===\n');

  const browser = await chromium.launch({
    headless: false
  });
  const context = await browser.newContext({
    ignoreHTTPSErrors: true
  });
  const page = await context.newPage();

  // Capture API responses
  const apiData = {};
  page.on('response', async (response) => {
    const url = response.url();
    if (url.includes('/api/campaigns')) {
      try {
        const data = await response.json();
        apiData[url] = data;
        console.log(`📡 API Call: ${url.split('/api/')[1]}`);
      } catch (e) {
        // Not JSON
      }
    }
  });

  try {
    // Login
    console.log('1. Logging in...');
    await page.goto('https://list.bobbyseamoss.com/admin/login');
    await page.fill('input[name="username"]', 'adam');
    await page.fill('input[name="password"]', 'bobbysea');
    await page.click('button[type="submit"]');
    await page.waitForURL('**/admin/**', { timeout: 10000 });
    console.log('✓ Logged in\n');

    // Navigate to Campaigns
    console.log('2. Loading Campaigns page...');
    await page.goto('https://list.bobbyseamoss.com/admin/campaigns');
    await page.waitForTimeout(5000); // Wait for data to load
    console.log('✓ Page loaded\n');

    // Check page HTML structure
    console.log('3. Analyzing page structure...');
    const html = await page.content();

    // Check if table exists
    const hasTable = html.includes('<table');
    console.log(`   Table element exists: ${hasTable}`);

    // Count rows different ways
    const tableBody = await page.locator('table tbody').count();
    console.log(`   <tbody> count: ${tableBody}`);

    const allRows = await page.locator('tr').count();
    console.log(`   Total <tr> count: ${allRows}`);

    const dataRows = await page.locator('tbody tr').count();
    console.log(`   Data rows in tbody: ${dataRows}\n`);

    // Try to get all table rows
    console.log('4. Extracting campaign data...');
    const rows = await page.locator('tbody tr').all();
    console.log(`   Found ${rows.length} rows\n`);

    if (rows.length > 0) {
      for (let i = 0; i < Math.min(rows.length, 10); i++) {
        const row = rows[i];
        const cells = await row.locator('td').all();

        console.log(`   Row ${i + 1}:`);
        for (let j = 0; j < cells.length; j++) {
          const text = await cells[j].textContent().catch(() => '');
          console.log(`      Cell ${j}: ${text.trim().substring(0, 50)}`);
        }
        console.log('');
      }
    } else {
      console.log('   ⚠️  No rows found in table\n');
    }

    // Check API responses
    console.log('5. Checking API responses...');
    for (const [url, data] of Object.entries(apiData)) {
      console.log(`\n   ${url}`);

      if (url.includes('/running/stats')) {
        console.log('   → This is queue stats endpoint');
        if (data && data.data) {
          console.log(`   → Found ${data.data.length} campaigns`);
          data.data.forEach((camp, idx) => {
            if (idx < 3) {
              console.log(`      Campaign ${camp.id}:`);
              console.log(`        status: ${camp.status}`);
              console.log(`        use_queue: ${camp.use_queue}`);
              console.log(`        queue_sent: ${camp.queue_sent || 'N/A'}`);
              console.log(`        queue_total: ${camp.queue_total || 'N/A'}`);
              console.log(`        azure_sent: ${camp.azure_sent || 0}`);
            }
          });
        }
      } else if (url.includes('/api/campaigns?')) {
        console.log('   → This is campaigns list endpoint');
        if (data && data.data && data.data.results) {
          console.log(`   → Found ${data.data.results.length} campaigns`);
          // Find paused campaigns
          const paused = data.data.results.filter(c => c.status === 'paused');
          console.log(`   → ${paused.length} paused campaigns`);
          if (paused.length > 0) {
            paused.forEach((c, idx) => {
              if (idx < 3) {
                console.log(`      Campaign ${c.id}: ${c.name}`);
                console.log(`        messenger: ${c.messenger}`);
              }
            });
          }
        }
      }
    }

    // Take screenshot
    console.log('\n6. Taking screenshot...');
    await page.screenshot({
      path: 'test-results/debug-campaigns-page.png',
      fullPage: true
    });
    console.log('✓ Screenshot saved\n');

    // Wait before closing so we can see
    console.log('Keeping browser open for 5 seconds for inspection...');
    await page.waitForTimeout(5000);

  } catch (error) {
    console.error('\n❌ Error:', error.message);
  } finally {
    await browser.close();
  }
}

debugProgressBar();
