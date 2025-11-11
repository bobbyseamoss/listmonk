---
name: browser-testing
description: Browser testing guidance for frontend fixes using Playwright. Triggers on keywords like browser, test in browser, frontend fix, Chrome, Firefox, playwright, verify fix, screenshot test, DOM inspection, cross-browser testing. Provides environment configs and testing best practices.
---

# Browser Testing for Frontend Fixes

## Purpose

Provides comprehensive guidance for testing frontend fixes across multiple environments using Playwright. Ensures proper cross-browser verification and consistent testing methodology.

## When to Use This Skill

Use this skill when:
- Testing frontend fixes in browsers
- Verifying UI changes work correctly
- Cross-browser compatibility testing (Chrome, Firefox)
- Visual regression testing with screenshots
- DOM inspection and selector verification
- Debugging CSS or JavaScript issues in production environments

## Test Environments

### 1. Local Development
- **URL**: `http://localhost:9000`
- **Username**: `adam`
- **Password**: `T@intshr3dd3r`
- **Use For**: Development and initial testing

### 2. Bobby Seamoss (Primary Test Environment)
- **URL**: `https://list.bobbyseamoss.com`
- **Username**: `adam`
- **Password**: `bobbysea`
- **Use For**: Most production-like testing, primary validation environment
- **Note**: This is the default environment for testing unless otherwise specified

### 3. Comma
- **URL**: `https://list.enjoycomma.com`
- **Username**: `comma`
- **Password**: `C0mm@69420`
- **Use For**: Secondary production environment testing

## Browser Testing Requirements

### Default Behavior
- **ALWAYS test in both Chrome AND Firefox** unless explicitly told otherwise
- Chrome is typically stricter with CSS/JavaScript
- Firefox is more lenient but should still be verified

### Testing Order
1. Test in Chrome first (catches more issues)
2. Test in Firefox second (verify cross-browser compatibility)

## Testing Methodology

### 1. Selector-Based Testing

Use Playwright selectors to verify elements exist and are visible:

```javascript
// Check element exists
const element = await page.locator('.my-element');
await expect(element).toBeVisible();

// Check element has correct properties
const bbox = await element.boundingBox();
expect(bbox.height).toBeGreaterThan(0);

// Check computed styles
const styles = await element.evaluate((el) => {
  const computed = window.getComputedStyle(el);
  return {
    display: computed.display,
    visibility: computed.visibility,
    height: computed.height,
  };
});
expect(styles.display).not.toBe('none');
```

### 2. Screenshot Verification

Take screenshots to visually verify the fix:

```javascript
// Full page screenshot
await page.screenshot({
  path: 'test-results/fullpage.png',
  fullPage: true
});

// Element screenshot
await element.screenshot({
  path: 'test-results/element.png'
});

// Viewport screenshot only
await page.screenshot({
  path: 'test-results/viewport.png'
});
```

### 3. Combined Approach (RECOMMENDED)

Always use BOTH methods for thorough verification:

```javascript
async function verifyFix(page, browserName) {
  console.log(`\n=== Testing in ${browserName} ===`);

  // 1. Selector-based checks
  const element = await page.locator('.target-element');
  const isVisible = await element.isVisible();
  const bbox = await element.boundingBox();

  console.log(`Element visible: ${isVisible}`);
  console.log(`Element dimensions:`, bbox);

  // 2. Screenshot for visual verification
  await page.screenshot({
    path: `test-results/${browserName}-result.png`,
    fullPage: true
  });

  // 3. Return results
  return {
    visible: isVisible,
    dimensions: bbox,
    screenshot: `${browserName}-result.png`
  };
}
```

## Standard Test Template

```javascript
const { chromium, firefox } = require('playwright');

async function testFix(browserType, browserName, baseUrl, credentials) {
  console.log(`\n=== Testing in ${browserName} ===`);

  const browser = await browserType.launch({
    headless: false // Set to true for CI/CD
  });
  const context = await browser.newContext({
    ignoreHTTPSErrors: true
  });
  const page = await context.newPage();

  try {
    // Login
    await page.goto(`${baseUrl}/admin/login`);
    await page.fill('input[name="username"]', credentials.username);
    await page.fill('input[name="password"]', credentials.password);
    await page.click('button[type="submit"]');
    await page.waitForURL('**/admin/**');

    // Navigate to test page
    await page.goto(`${baseUrl}/admin/your-page`);
    await page.waitForTimeout(2000);

    // Test your fix here
    const element = await page.locator('.your-element');
    const isVisible = await element.isVisible();

    console.log(`Fix verified: ${isVisible ? '✅ YES' : '❌ NO'}`);

    // Take screenshot
    await page.screenshot({
      path: `test-results/${browserName}-fix.png`,
      fullPage: true
    });

    return isVisible;

  } catch (error) {
    console.error(`Error in ${browserName}:`, error.message);
    return false;
  } finally {
    await browser.close();
  }
}

// Environment configs
const ENVIRONMENTS = {
  local: {
    url: 'http://localhost:9000',
    username: 'adam',
    password: 'T@intshr3dd3r'
  },
  bobby: {
    url: 'https://list.bobbyseamoss.com',
    username: 'adam',
    password: 'bobbysea'
  },
  comma: {
    url: 'https://list.enjoycomma.com',
    username: 'comma',
    password: 'C0mm@69420'
  }
};

// Test in both browsers
(async () => {
  const env = ENVIRONMENTS.bobby; // Default to Bobby Seamoss

  const chromeResult = await testFix(chromium, 'Chrome', env.url, env);
  const firefoxResult = await testFix(firefox, 'Firefox', env.url, env);

  console.log('\n=== Results ===');
  console.log(`Chrome: ${chromeResult ? '✅ PASS' : '❌ FAIL'}`);
  console.log(`Firefox: ${firefoxResult ? '✅ PASS' : '❌ FAIL'}`);
})();
```

## Common Verification Patterns

### CSS Fix Verification

```javascript
// Verify CSS properties
const styles = await element.evaluate((el) => {
  const computed = window.getComputedStyle(el);
  return {
    display: computed.display,
    visibility: computed.visibility,
    opacity: computed.opacity,
    height: computed.height,
    width: computed.width,
  };
});

console.log('Computed styles:', styles);
expect(styles.display).toBe('block');
expect(styles.visibility).toBe('visible');
expect(parseFloat(styles.height)).toBeGreaterThan(0);
```

### JavaScript/Rendering Fix Verification

```javascript
// Check if element renders and has content
const element = await page.locator('.dynamic-element');
await element.waitFor({ state: 'visible', timeout: 5000 });

const text = await element.textContent();
const html = await element.innerHTML();

console.log('Element text:', text);
console.log('Element HTML:', html);
```

### Layout Fix Verification

```javascript
// Check positioning and dimensions
const bbox = await element.boundingBox();

console.log('Position:', { x: bbox.x, y: bbox.y });
console.log('Size:', { width: bbox.width, height: bbox.height });

// Verify element is in viewport
const viewport = page.viewportSize();
const inViewport = bbox.y >= 0 &&
                   bbox.y + bbox.height <= viewport.height &&
                   bbox.x >= 0 &&
                   bbox.x + bbox.width <= viewport.width;

console.log('In viewport:', inViewport);
```

## Best Practices

### 1. Always Use Headless: false During Development

```javascript
const browser = await browserType.launch({
  headless: false // Watch the test run
});
```

This lets you:
- See what's happening in real-time
- Manually inspect if something fails
- Verify visual appearance

### 2. Add Appropriate Waits

```javascript
// Wait for navigation
await page.waitForURL('**/admin/**');

// Wait for element
await page.waitForSelector('.my-element', { state: 'visible' });

// Wait for network idle
await page.waitForLoadState('networkidle');

// Simple timeout (use sparingly)
await page.waitForTimeout(2000);
```

### 3. Handle Login Failures Gracefully

```javascript
try {
  await page.waitForURL('**/admin/**', { timeout: 10000 });
} catch (e) {
  console.log('⚠️  Navigation timeout, checking if logged in...');
  const currentUrl = page.url();
  if (currentUrl.includes('/admin/login')) {
    throw new Error('Login failed');
  }
}
```

### 4. Clean Up Test Artifacts

```javascript
const fs = require('fs');
const path = require('path');

// Create test results directory
const resultsDir = path.join(__dirname, 'test-results');
if (!fs.existsSync(resultsDir)) {
  fs.mkdirSync(resultsDir, { recursive: true });
}
```

### 5. Compare Before/After Screenshots

```javascript
// Take before screenshot
await page.screenshot({ path: 'test-results/before.png' });

// Apply fix / make changes

// Take after screenshot
await page.screenshot({ path: 'test-results/after.png' });

// Use visual comparison tool or manual inspection
```

## Troubleshooting

### Element Not Found
- Check if element is in an iframe: `page.frame()`
- Wait longer: increase timeout
- Check selector: use DevTools to verify
- Verify page loaded: `await page.waitForLoadState('domcontentloaded')`

### Login Issues
- Verify credentials are correct for environment
- Check for CAPTCHA or 2FA
- Ensure cookies are enabled
- Try manual login first to verify

### Screenshot Is Blank
- Element might be scrolled out of view: `await element.scrollIntoViewIfNeeded()`
- Page might not be fully loaded: `await page.waitForLoadState('networkidle')`
- Check viewport size: `await page.setViewportSize({ width: 1920, height: 1080 })`

### Cross-Browser Differences
- Chrome uses Blink engine, Firefox uses Gecko
- CSS rendering can differ slightly
- Check for vendor-specific prefixes
- Verify both browsers load the same CSS files

## Quick Reference

### Environment URLs
- **Local**: `http://localhost:9000`
- **Bobby Seamoss**: `https://list.bobbyseamoss.com` (DEFAULT)
- **Comma**: `https://list.enjoycomma.com`

### Default Testing
- ✅ Test in Chrome first
- ✅ Test in Firefox second
- ✅ Use selectors to verify
- ✅ Take screenshots for visual confirmation
- ✅ Log results clearly

### Typical Test Flow
1. Launch both browsers
2. Login to environment (Bobby Seamoss by default)
3. Navigate to page with fix
4. Verify with selectors (element exists, visible, correct properties)
5. Take screenshots (full page recommended)
6. Compare results between browsers
7. Report findings clearly

---

**Remember**: Bobby Seamoss is the primary test environment. Always test in both Chrome and Firefox unless explicitly told otherwise. Use both selectors and screenshots for thorough verification.
