# Playwright Testing Infrastructure Setup - Context

**Status**: ✅ COMPLETED
**Last Updated**: 2025-11-09
**Location**: `/tmp/` (temporary, can be moved to permanent location if needed)

## Overview

Established comprehensive Playwright testing infrastructure for automated frontend testing of listmonk admin interface. Fixed authentication issues and created reusable helper library.

## Problem Solved

User needed Playwright for frontend development, but authentication was failing with "Invalid login or password" error.

**Root Causes**:
1. Wrong credentials (was using `admin`, should be `adam`)
2. `networkidle` wait strategy timing out (listmonk has active polling)
3. Navigation timeouts (needed longer timeout and better wait strategy)

## Solution Implemented

### 1. Authentication Fix

**Correct Credentials**:
- Username: `adam` (not `admin`)
- Password: `T@intshr3dd3r`

**Login Flow**:
1. Navigate to `https://list.bobbyseamoss.com/admin/`
2. Fill `input[name="username"]` with username
3. Fill `input[name="password"]` with password
4. Click `button[type="submit"]`
5. Wait for navigation with `waitUntil: 'load'` (not `networkidle`)
6. Verify URL doesn't contain `/login` (indicates success)

### 2. Wait Strategy Adjustment

**Changed**: From `waitUntil: 'networkidle'` to `waitUntil: 'load'`

**Reason**: Listmonk has active polling/websockets that prevent network from going idle. Using `load` waits for DOM load event, then adds explicit timeout for Vue rendering.

**Pattern**:
```javascript
await page.goto(url, { waitUntil: 'load', timeout: 15000 });
await page.waitForTimeout(1500); // Wait for Vue to render
```

## Files Created

### 1. `/tmp/playwright-helpers.js` - Core Library (142 lines)

Reusable helper functions for all Playwright testing.

#### Main Functions:

**`createAuthenticatedSession(options)`**
- Creates browser, authenticates, returns session object
- Options: baseUrl, username, password, headless
- Returns: `{ browser, context, page }`
- Throws error if login fails
- Default timeout: 15000ms

**`navigateTo(session, path)`**
- Navigate to admin page (e.g., `/campaigns`)
- Uses `load` wait strategy + 1500ms settle time
- Automatically constructs full URL

**`takeScreenshot(session, filename, options)`**
- Captures full page screenshot by default
- Saves to specified path
- Accepts standard Playwright screenshot options

**`checkForErrors(session)`**
- Sets up error listeners on page
- Waits 2000ms to collect errors
- Returns: `{ errors: string[], consoleErrors: string[] }`

**`closeSession(session)`**
- Closes browser and cleans up resources
- Always call in `finally` block

#### Usage Example:
```javascript
const session = await createAuthenticatedSession();
await navigateTo(session, '/campaigns');
await takeScreenshot(session, '/tmp/test.png');
const { errors } = await checkForErrors(session);
await closeSession(session);
```

### 2. `/tmp/test-campaigns-page.js` - Quick Test Script (53 lines)

Simple script demonstrating basic usage:
- Authenticates
- Navigates to campaigns page
- Checks for errors
- Counts progress bars and campaigns
- Takes screenshot
- Proper cleanup in finally block

**Run**: `node /tmp/test-campaigns-page.js`

### 3. `/tmp/playwright-example.js` - Comprehensive Examples (149 lines)

Five complete examples:

**Example 1**: Test specific page with error checking
**Example 2**: Test multiple pages in sequence
**Example 3**: Interact with page elements (click, fill forms)
**Example 4**: Custom authentication options (visible browser)
**Example 5**: Check for specific UI elements (count, query)

**Run**: `node /tmp/playwright-example.js`

### 4. `/tmp/diagnose-login.js` - Diagnostic Tool (107 lines)

Debug script for troubleshooting login issues:
- Shows form HTML structure
- Lists all input fields and buttons
- Tests selectors
- Checks for error messages
- Takes screenshots at each step
- Useful for debugging authentication problems

**Run**: `node /tmp/diagnose-login.js`

### 5. `/tmp/PLAYWRIGHT-README.md` - Documentation (298 lines)

Complete guide covering:
- Setup instructions
- API reference for all helper functions
- Common patterns and examples
- Troubleshooting guide
- Authentication details
- Tips and best practices
- File descriptions

## Login Form Details

### HTML Structure
```html
<form method="post" action="/admin/login" class="form">
  <input type="hidden" name="nonce" value="...">
  <input type="hidden" name="next" value="/admin">
  <input id="username" type="text" name="username" required minlength="3">
  <input id="password" type="password" name="password" required minlength="8">
  <button type="submit" class="button">Login</button>
</form>
```

### Selectors Used
- Username: `input[name="username"]` or `#username`
- Password: `input[name="password"]` or `#password`
- Submit: `button[type="submit"]`

### Error Detection
Error messages appear in: `.error`, `.notification.is-danger`, `.message.is-danger`

After successful login:
- Redirects to `/admin` (dashboard)
- Any URL containing `/login` indicates failure

## Testing Patterns Established

### Pattern 1: Basic Page Test
```javascript
const session = await createAuthenticatedSession();
await navigateTo(session, '/campaigns');
const { errors } = await checkForErrors(session);
await takeScreenshot(session, '/tmp/test.png');
await closeSession(session);
```

### Pattern 2: Element Interaction
```javascript
const session = await createAuthenticatedSession();
const { page } = session;
await navigateTo(session, '/campaigns');
await page.click('button:has-text("New")');
await page.fill('input[name="name"]', 'Test Campaign');
const elements = await page.$$('.campaign-progress');
await closeSession(session);
```

### Pattern 3: Multi-Page Testing
```javascript
const session = await createAuthenticatedSession();
const pages = ['/campaigns', '/subscribers', '/lists'];
for (const path of pages) {
  await navigateTo(session, path);
  await takeScreenshot(session, `/tmp${path}.png`);
}
await closeSession(session);
```

## Key Technical Decisions

### 1. Why `waitUntil: 'load'` Not `networkidle`?
Listmonk frontend has:
- Active polling (campaign stats, queue status)
- WebSocket connections (real-time updates)
- Background API requests

These prevent network from going idle. Using `load` waits for DOM load, then we add explicit timeout for Vue rendering.

### 2. Why 1500ms Wait After Navigation?
Vue 2 needs time to:
- Mount components
- Fetch initial data
- Render dynamic content
- Execute lifecycle hooks

1500ms is empirically determined sweet spot for most pages.

### 3. Why 15000ms Login Timeout?
Login involves:
- Form submission
- Server authentication
- Session creation
- Redirect to dashboard
- Dashboard initial load

Standard 10000ms was occasionally timing out, 15000ms provides buffer.

### 4. Error Listener Pattern
Set up listeners, then wait 2000ms to collect errors. This catches:
- Immediate errors (template, props)
- Mounted hook errors
- Initial data fetch errors
- Console.error() calls

## Verification Results

**Test Run Output**:
```
🚀 Launching browser...
🔐 Navigating to login page...
✏️  Filling credentials...
🔘 Clicking login button...
✅ Authentication successful!
📍 Current URL: https://list.bobbyseamoss.com/admin
🧭 Navigating to https://list.bobbyseamoss.com/admin/campaigns...
✅ Navigation complete

✅ No JavaScript errors detected
✅ No console errors detected

📊 Found 0 progress bars on the page
📋 Found 2 campaigns in the table
📸 Taking screenshot: /tmp/campaigns-authenticated-working.png...
✅ Screenshot saved

✨ Test completed successfully!
🧹 Closing browser...
✅ Browser closed
```

## Browser Configuration

**Default Settings**:
- Browser: Chromium
- Headless: true
- Viewport: Default (1280x720)
- Timeout: 15000ms for navigation

**Custom Options**:
```javascript
await createAuthenticatedSession({
  headless: false, // Show browser
  baseUrl: 'https://other-instance.com',
  username: 'custom-user',
  password: 'custom-pass'
});
```

## Common Use Cases

### 1. Verify UI Changes After Deployment
```bash
node /tmp/test-campaigns-page.js
# Check screenshot: /tmp/campaigns-authenticated-working.png
```

### 2. Debug Frontend Errors
```javascript
const session = await createAuthenticatedSession();
await navigateTo(session, '/problematic-page');
const { errors, consoleErrors } = await checkForErrors(session);
console.log('Errors:', errors);
```

### 3. Test Form Submission
```javascript
const session = await createAuthenticatedSession();
const { page } = session;
await navigateTo(session, '/campaigns');
await page.click('button:has-text("New")');
await page.fill('input[name="name"]', 'Test');
await page.click('button:has-text("Save")');
await page.waitForTimeout(2000);
```

### 4. Validate Data Display
```javascript
const session = await createAuthenticatedSession();
const { page } = session;
await navigateTo(session, '/campaigns');
const campaignCount = await page.$$eval('tbody tr', rows => rows.length);
console.log(`Found ${campaignCount} campaigns`);
```

## Installation Requirements

Playwright is already installed. If reinstalling needed:

```bash
npm install playwright
npx playwright install chromium
```

Version used: playwright@latest (as of 2025-11-09)

## Troubleshooting Guide

### Issue: Login Fails
**Check**:
1. Credentials correct? (`adam` / `T@intshr3dd3r`)
2. Base URL correct? (`https://list.bobbyseamoss.com`)
3. Network connection working?

**Debug**:
```bash
node /tmp/diagnose-login.js
# Check debug screenshots in /tmp/
```

### Issue: Navigation Timeout
**Solution**: Increase timeout or change wait strategy
```javascript
await page.goto(url, { waitUntil: 'load', timeout: 30000 });
```

### Issue: Elements Not Found
**Solution**: Add explicit wait
```javascript
await page.waitForSelector('.campaign-progress', { timeout: 5000 });
```

### Issue: Screenshot Shows Wrong State
**Solution**: Add longer wait for dynamic content
```javascript
await page.waitForTimeout(3000); // Wait for updates
await takeScreenshot(session, '/tmp/test.png');
```

## Performance Considerations

- Each test creates new browser instance (clean state)
- Browser launch: ~1-2 seconds
- Login: ~2-3 seconds
- Page navigation: ~1-2 seconds per page
- Full test cycle: ~5-10 seconds

For faster iteration during development:
```javascript
const session = await createAuthenticatedSession({ headless: false });
// Keep session open, navigate multiple times
await navigateTo(session, '/page1');
await navigateTo(session, '/page2');
await navigateTo(session, '/page3');
await closeSession(session);
```

## Integration with CI/CD

Can be integrated into deployment pipeline:

```bash
# After deployment, verify UI
node /tmp/test-campaigns-page.js

# Check exit code
if [ $? -eq 0 ]; then
  echo "UI verification passed"
else
  echo "UI verification failed"
  exit 1
fi
```

## Future Enhancements

Potential additions:
1. **Session persistence**: Save cookies to skip login
2. **Parallel execution**: Run multiple tests simultaneously
3. **Video recording**: Capture test execution video
4. **Visual regression**: Compare screenshots against baseline
5. **Network interception**: Mock API responses
6. **Performance metrics**: Measure page load times
7. **Accessibility testing**: Check ARIA labels, contrast, etc.

## File Locations

All files currently in `/tmp/`:
- `playwright-helpers.js` - Core library
- `test-campaigns-page.js` - Quick test
- `playwright-example.js` - Examples
- `diagnose-login.js` - Diagnostic tool
- `PLAYWRIGHT-README.md` - Documentation
- `verify-campaigns-fix.js` - Ad-hoc verification script
- `check_campaigns_better.js` - Old test script (can delete)
- `check_campaigns_authenticated.js` - Old test script (can delete)

**Recommendation**: Move to permanent location like `/home/adam/listmonk/testing/playwright/` if this becomes long-term testing infrastructure.

## Status

✅ **COMPLETED AND VERIFIED**
- Authentication working with correct credentials
- All helper functions tested and working
- Documentation complete
- Example scripts functional
- Diagnostic tools available
- Ready for frontend development and testing

**Next User Request**: User can now use Playwright for all frontend development needs
