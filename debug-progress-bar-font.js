const { chromium } = require('playwright');

(async () => {
  const browser = await chromium.launch({ headless: false });
  const context = await browser.newContext();
  const page = await context.newPage();

  try {
    console.log('🔍 Debugging Progress Bar Font Size\n');

    // Login
    console.log('1. Logging in...');
    await page.goto('https://list.bobbyseamoss.com/admin/login', { waitUntil: 'load' });
    await page.fill('input[name="username"]', 'adam');
    await page.fill('input[name="password"]', 'T@intshr3dd3r');
    await page.click('button[type="submit"]');
    await page.waitForTimeout(2000);

    // Navigate to campaigns
    console.log('2. Navigating to campaigns page...');
    await page.goto('https://list.bobbyseamoss.com/admin/campaigns', { waitUntil: 'load' });
    await page.waitForTimeout(3000);

    // Find all progress bar elements
    console.log('3. Inspecting progress bar elements...\n');

    const progressBarInfo = await page.evaluate(() => {
      const results = [];

      // Find all progress elements (could be .progress, .progress-bar, etc.)
      const progressElements = document.querySelectorAll('progress, .progress, .progress-bar, [role="progressbar"]');

      progressElements.forEach((el, index) => {
        const computedStyle = window.getComputedStyle(el);

        results.push({
          index,
          tagName: el.tagName,
          className: el.className,
          text: el.textContent?.trim().substring(0, 50) || 'No text',
          computedFontSize: computedStyle.fontSize,
          inlineStyle: el.getAttribute('style') || 'None',
          cssText: computedStyle.cssText.substring(0, 200),
        });
      });

      // Also check for text inside progress containers
      const progressContainers = document.querySelectorAll('.progress-text, .progress-label, .campaign-progress');
      progressContainers.forEach((el, index) => {
        const computedStyle = window.getComputedStyle(el);

        results.push({
          index: `text-${index}`,
          tagName: el.tagName,
          className: el.className,
          text: el.textContent?.trim().substring(0, 50) || 'No text',
          computedFontSize: computedStyle.fontSize,
          inlineStyle: el.getAttribute('style') || 'None',
          cssText: computedStyle.cssText.substring(0, 200),
        });
      });

      return results;
    });

    console.log('=== PROGRESS BAR ELEMENTS FOUND ===');
    if (progressBarInfo.length === 0) {
      console.log('❌ No progress bar elements found!');
    } else {
      progressBarInfo.forEach(info => {
        console.log(`\n--- Element #${info.index} ---`);
        console.log(`Tag: ${info.tagName}`);
        console.log(`Class: ${info.className}`);
        console.log(`Text: ${info.text}`);
        console.log(`Computed Font Size: ${info.computedFontSize}`);
        console.log(`Inline Style: ${info.inlineStyle}`);
        console.log(`CSS Text (first 200 chars): ${info.cssText}`);
      });
    }

    // Look for specific campaign progress elements in the table
    console.log('\n\n=== CHECKING TABLE PROGRESS CELLS ===');
    const tableProgressInfo = await page.evaluate(() => {
      const results = [];

      // Find all table rows
      const rows = document.querySelectorAll('tbody tr');

      rows.forEach((row, rowIndex) => {
        // Look for progress-related cells
        const cells = row.querySelectorAll('td');
        cells.forEach((cell, cellIndex) => {
          const text = cell.textContent?.trim();

          // Check if this cell contains progress-like content (percentages, fractions)
          if (text && (text.includes('%') || text.match(/\d+\s*\/\s*\d+/))) {
            const allElements = cell.querySelectorAll('*');

            allElements.forEach((el, elIndex) => {
              const computedStyle = window.getComputedStyle(el);

              results.push({
                row: rowIndex,
                cell: cellIndex,
                element: elIndex,
                tagName: el.tagName,
                className: el.className,
                text: el.textContent?.trim().substring(0, 50),
                computedFontSize: computedStyle.fontSize,
                inlineStyle: el.getAttribute('style') || 'None',
                classList: Array.from(el.classList).join(', ') || 'None',
              });
            });
          }
        });
      });

      return results;
    });

    if (tableProgressInfo.length === 0) {
      console.log('❌ No progress elements found in table cells!');
    } else {
      tableProgressInfo.forEach(info => {
        console.log(`\n--- Row ${info.row}, Cell ${info.cell}, Element ${info.element} ---`);
        console.log(`Tag: ${info.tagName}`);
        console.log(`Classes: ${info.classList}`);
        console.log(`Text: ${info.text}`);
        console.log(`Font Size: ${info.computedFontSize}`);
        console.log(`Inline Style: ${info.inlineStyle}`);
      });
    }

    // Check for any CSS rules that might be setting font-size to .5rem
    console.log('\n\n=== CSS RULES WITH .5rem FONT SIZE ===');
    const cssRules = await page.evaluate(() => {
      const results = [];

      // Get all stylesheets
      for (const sheet of document.styleSheets) {
        try {
          const rules = sheet.cssRules || sheet.rules;
          for (const rule of rules) {
            if (rule.style && rule.style.fontSize === '0.5rem') {
              results.push({
                selector: rule.selectorText,
                fontSize: rule.style.fontSize,
                href: sheet.href || 'inline',
              });
            }
          }
        } catch (e) {
          // CORS issues with external stylesheets
        }
      }

      return results;
    });

    if (cssRules.length === 0) {
      console.log('✅ No CSS rules found with font-size: .5rem');
    } else {
      cssRules.forEach(rule => {
        console.log(`\nSelector: ${rule.selector}`);
        console.log(`Font Size: ${rule.fontSize}`);
        console.log(`Stylesheet: ${rule.href}`);
      });
    }

    // Take a screenshot for visual reference
    await page.screenshot({ path: '/tmp/progress-bar-debug.png', fullPage: true });
    console.log('\n\n📸 Screenshot saved to /tmp/progress-bar-debug.png');

    console.log('\n\n=== DEBUGGING COMPLETE ===');
    console.log('Browser will remain open for 30 seconds for manual inspection...');
    await page.waitForTimeout(30000);

  } catch (error) {
    console.error('❌ Error:', error);
  } finally {
    await browser.close();
  }
})();
