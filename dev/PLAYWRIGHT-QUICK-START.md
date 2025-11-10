# Playwright Quick Start for Listmonk

Quick reference for using Playwright to test the listmonk frontend.

## Location

All Playwright files are in `/tmp/` (temporary location)

## Authentication

```
URL: https://list.bobbyseamoss.com/admin/
Username: adam
Password: T@intshr3dd3r
```

## Quick Test

```bash
node /tmp/test-campaigns-page.js
```

## Basic Usage

```javascript
const {
  createAuthenticatedSession,
  navigateTo,
  takeScreenshot,
  closeSession
} = require('/tmp/playwright-helpers');

(async () => {
  const session = await createAuthenticatedSession();
  await navigateTo(session, '/campaigns');
  await takeScreenshot(session, '/tmp/test.png');
  await closeSession(session);
})();
```

## Available Pages

Common paths to test:
- `/campaigns` - Campaigns list
- `/subscribers` - Subscribers list
- `/lists` - Mailing lists
- `/settings` - Settings pages
- `/users` - User management

## Helper Functions

```javascript
// Authenticate and get session
const session = await createAuthenticatedSession();

// Navigate to page
await navigateTo(session, '/campaigns');

// Take screenshot
await takeScreenshot(session, '/tmp/screenshot.png');

// Check for errors
const { errors, consoleErrors } = await checkForErrors(session);

// Close browser
await closeSession(session);
```

## Common Selectors

```javascript
// Click button
await page.click('button:has-text("New")');

// Fill input
await page.fill('input[name="name"]', 'value');

// Count elements
const count = await page.$$eval('.campaign-progress', els => els.length);

// Get text
const text = await page.textContent('.status');
```

## Debugging

Run with visible browser:
```javascript
const session = await createAuthenticatedSession({
  headless: false
});
```

## Full Documentation

See `/tmp/PLAYWRIGHT-README.md` for complete documentation and examples.

## Examples

See `/tmp/playwright-example.js` for five complete usage examples.

## Diagnostic Tool

If login issues occur:
```bash
node /tmp/diagnose-login.js
```
