const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext();
  const page = await context.newPage();

  try {
    console.log('✅ Verifying Progress Bar Font Size Fix\n');

    // Login
    await page.goto('https://list.bobbyseamoss.com/admin/login', { waitUntil: 'load' });
    await page.fill('input[name="username"]', 'adam');
    await page.fill('input[name="password"]', 'T@intshr3dd3r');
    await page.click('button[type="submit"]');
    await page.waitForTimeout(2000);

    // Navigate to campaigns
    await page.goto('https://list.bobbyseamoss.com/admin/campaigns', { waitUntil: 'load' });
    await page.waitForTimeout(3000);

    // Check progress-value elements
    const progressValueElements = await page.evaluate(() => {
      const elements = document.querySelectorAll('.progress-value');
      const results = [];

      elements.forEach((el, index) => {
        const computedStyle = window.getComputedStyle(el);
        results.push({
          index,
          text: el.textContent?.trim(),
          fontSize: computedStyle.fontSize,
          fontSizeInRem: parseFloat(computedStyle.fontSize) / 15, // Base font size is 15px
        });
      });

      return results;
    });

    console.log('=== PROGRESS BAR TEXT FONT SIZES ===\n');

    if (progressValueElements.length === 0) {
      console.log('⚠️  No .progress-value elements found');
    } else {
      let allCorrect = true;

      progressValueElements.forEach(el => {
        const expectedPx = 10.5; // 0.7rem * 15px base = 10.5px
        const actualPx = parseFloat(el.fontSize);
        const isCorrect = Math.abs(actualPx - expectedPx) < 0.1;

        console.log(`Element ${el.index}:`);
        console.log(`  Text: ${el.text}`);
        console.log(`  Font Size: ${el.fontSize} (${el.fontSizeInRem.toFixed(2)}rem)`);
        console.log(`  Expected: 10.5px (0.70rem)`);
        console.log(`  Status: ${isCorrect ? '✅ CORRECT' : '❌ INCORRECT'}\n`);

        if (!isCorrect) {
          allCorrect = false;
        }
      });

      console.log('\n=== SUMMARY ===');
      if (allCorrect) {
        console.log('✅ ALL PROGRESS BAR TEXT IS NOW 0.7rem (10.5px) - FIX SUCCESSFUL!');
      } else {
        console.log('❌ Some elements still have incorrect font size');
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
                fontSize: rule.style.fontSize,
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
    cssRules.forEach(rule => {
      console.log(`Selector: ${rule.selector}`);
      console.log(`Font Size: ${rule.fontSize}`);
      console.log(`Source: ${rule.href.substring(0, 80)}...\n`);
    });

  } catch (error) {
    console.error('❌ Error:', error);
  } finally {
    await browser.close();
  }
})();
