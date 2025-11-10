const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext();
  const page = await context.newPage();

  try {
    console.log('✅ Verifying Progress Bar Top Position\n');

    // Login
    await page.goto('https://list.bobbyseamoss.com/admin/login', { waitUntil: 'load' });
    await page.fill('input[name="username"]', 'adam');
    await page.fill('input[name="password"]', 'T@intshr3dd3r');
    await page.click('button[type="submit"]');
    await page.waitForTimeout(2000);

    // Navigate to campaigns
    await page.goto('https://list.bobbyseamoss.com/admin/campaigns', { waitUntil: 'load' });
    await page.waitForTimeout(3000);

    // Check progress-value elements for top position
    const progressValueElements = await page.evaluate(() => {
      const elements = document.querySelectorAll('.progress-value');
      const results = [];

      elements.forEach((el, index) => {
        const computedStyle = window.getComputedStyle(el);
        results.push({
          index,
          text: el.textContent?.trim(),
          top: computedStyle.top,
          position: computedStyle.position,
        });
      });

      return results;
    });

    console.log('=== PROGRESS BAR TOP POSITION ===\n');

    if (progressValueElements.length === 0) {
      console.log('⚠️  No .progress-value elements found');
    } else {
      let allCorrect = true;

      progressValueElements.forEach(el => {
        console.log(`Element ${el.index}:`);
        console.log(`  Text: ${el.text}`);
        console.log(`  Position: ${el.position}`);
        console.log(`  Top: ${el.top}`);

        // Check if top is set to 16%
        const isCorrect = el.top === '16%';
        console.log(`  Expected: 16%`);
        console.log(`  Status: ${isCorrect ? '✅ CORRECT' : '⚠️  DIFFERENT'}\n`);

        if (!isCorrect) {
          allCorrect = false;
        }
      });

      console.log('\n=== SUMMARY ===');
      if (allCorrect) {
        console.log('✅ ALL PROGRESS BAR ELEMENTS HAVE top: 16% - FIX SUCCESSFUL!');
      } else {
        console.log('⚠️  Some elements have different top values');
        console.log('   Note: This may be expected if the element is not positioned');
      }
    }

    // Also check CSS rules
    const cssRules = await page.evaluate(() => {
      const results = [];

      for (const sheet of document.styleSheets) {
        try {
          const rules = sheet.cssRules || sheet.rules;
          for (const rule of rules) {
            if (rule.style && rule.selectorText?.includes('progress-value')) {
              results.push({
                selector: rule.selectorText,
                top: rule.style.top,
                position: rule.style.position,
                href: sheet.href || 'inline',
              });
            }
          }
        } catch (e) {
          // CORS issues
        }
      }

      return results;
    });

    console.log('\n=== CSS RULES FOR .progress-value ===\n');
    if (cssRules.length === 0) {
      console.log('⚠️  No CSS rules found for .progress-value');
    } else {
      cssRules.forEach(rule => {
        console.log(`Selector: ${rule.selector}`);
        console.log(`Top: ${rule.top || '(not set)'}`);
        console.log(`Position: ${rule.position || '(not set)'}`);
        console.log(`Source: ${rule.href.substring(0, 80)}...\n`);
      });
    }

  } catch (error) {
    console.error('❌ Error:', error);
  } finally {
    await browser.close();
  }
})();
