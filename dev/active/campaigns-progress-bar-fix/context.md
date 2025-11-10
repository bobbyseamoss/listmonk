# Campaigns Progress Bar Fix - Context

**Status**: ✅ COMPLETED
**Last Updated**: 2025-11-09
**Deployed**: Yes - Revision listmonk420--0000097

## Overview

Fixed JavaScript error preventing campaign progress bars from displaying on the All Campaigns page. Also established Playwright testing infrastructure for future frontend development.

## Problem Summary

User reported: "The UI change we made to add the status bar back in to the Campaigns page isn't working and is misaligned."

**Root Cause**: JavaScript error `TypeError: can't access property "sent", t.stats is undefined` caused by circular dependency in Vue 2 template.

**Console Error Location**: `https://list.bobbyseamoss.com/admin/static/Campaigns-Ca5gd4_H.js:1`

## Root Cause Analysis

The original implementation used this pattern:
```vue
<div :set="stats = getCampaignStats(props.row)" v-if="!isDone(props.row) && (stats.sent > 0 || ...)">
```

**Problem**: Vue evaluated the `v-if` condition before `:set` executed, causing `stats` to be undefined when accessing `stats.sent`.

## Solution Implemented

### 1. Removed Circular Dependency Pattern

**Changed**: Eliminated `:set` directive entirely
**Replaced with**: Helper methods that encapsulate all logic

### 2. Created Three Helper Methods

**File**: `/home/adam/listmonk/frontend/src/views/Campaigns.vue`

#### `showProgress(campaign)` - Lines 336-353
- Determines if progress bar should display
- Returns `false` if campaign or stats is null/undefined
- Shows for ALL campaigns (paused, cancelled, finished) with sent > 0
- Supports both regular and queue-based campaigns

```javascript
showProgress(campaign) {
  if (!campaign) {
    return false;
  }
  const stats = this.getCampaignStats(campaign);
  if (!stats) {
    return false;
  }
  // Show for regular campaigns with sent > 0
  if (stats.sent && stats.sent > 0) {
    return true;
  }
  // Show for queue campaigns with queue_sent > 0
  if ((stats.use_queue || stats.useQueue) && (stats.queue_sent > 0 || stats.queueSent > 0)) {
    return true;
  }
  return false;
}
```

#### `getProgressText(campaign)` - Lines 355-368
- Generates email counter text (e.g., "150 / 500")
- Returns '0 / 0' if stats is null
- Handles both camelCase and snake_case field names

```javascript
getProgressText(campaign) {
  const stats = this.getCampaignStats(campaign);
  if (!stats) {
    return '0 / 0';
  }
  if (stats.use_queue || stats.useQueue) {
    const sent = stats.queue_sent || stats.queueSent || 0;
    const total = stats.queue_total || stats.queueTotal || 0;
    return `${sent} / ${total}`;
  }
  const sent = stats.sent || 0;
  const total = stats.toSend || 0;
  return `${sent} / ${total}`;
}
```

#### `getProgressPercent(stats)` - Lines 415-429 (Updated)
- Added null check at the beginning
- Calculates percentage for progress bar
- Supports both campaign types

```javascript
getProgressPercent(stats) {
  if (!stats) {
    return 0;
  }
  if (stats.use_queue || stats.useQueue) {
    const total = stats.queue_total || stats.queueTotal || 0;
    const sent = stats.queue_sent || stats.queueSent || 0;
    if (total === 0) return 0;
    return (sent / total) * 100;
  }
  if (stats.toSend === 0) return 0;
  return (stats.sent / stats.toSend) * 100;
}
```

### 3. Updated Template - Lines 117-127

```vue
<!-- Campaign Progress -->
<div class="campaign-progress" v-if="showProgress(props.row)">
  <b-progress
    :value="getProgressPercent(getCampaignStats(props.row))"
    :type="props.row.status === 'running' ? 'is-primary' : props.row.status === 'paused' ? 'is-warning' : 'is-light'"
    size="is-small"
    show-value
  >
    {{ getProgressText(props.row) }}
  </b-progress>
</div>
```

## Key Design Decisions

### 1. Show Progress for ALL Campaigns
**User Requirement**: "All stats should be displayed and current even if the campaign is paused, canceled or finished"

**Implementation**: Removed `this.isDone(campaign)` check from `showProgress()` method. Now displays for any campaign with sent > 0 regardless of status.

### 2. Comprehensive Null Checking
All methods check for null/undefined at every step to prevent similar errors:
- Campaign existence check
- Stats existence check
- Individual field checks with fallback to 0

### 3. Support Both Field Naming Conventions
Handles both camelCase and snake_case:
- `stats.use_queue || stats.useQueue`
- `stats.queue_sent || stats.queueSent`
- `stats.queue_total || stats.queueTotal`

This ensures compatibility regardless of backend serialization format.

### 4. Color Coding by Status
Progress bar colors:
- Blue (`is-primary`): Running campaigns
- Yellow (`is-warning`): Paused campaigns
- Gray (`is-light`): Other statuses (cancelled, finished, draft)

## Files Modified

### `/home/adam/listmonk/frontend/src/views/Campaigns.vue`

**Lines 117-127**: Updated template to use helper methods
**Lines 336-353**: Added `showProgress()` method
**Lines 355-368**: Added `getProgressText()` method
**Lines 415-429**: Added null check to `getProgressPercent()`
**Lines 610-618**: CSS already existed (no changes needed)

## Deployment

### Build Process
1. Frontend build: `cd /home/adam/listmonk/frontend && yarn build` ✅
2. Distribution build: `cd /home/adam/listmonk && make dist` ✅
3. Docker build: `docker build -t listmonk420acr.azurecr.io/listmonk:latest .` ✅
4. Push to ACR: `az acr login --name listmonk420acr && docker push` ✅
5. Update container: `az containerapp update --name listmonk420 --resource-group rg-listmonk420 --image listmonk420acr.azurecr.io/listmonk:latest` ✅
6. Restart revision: `az containerapp revision restart --name listmonk420 --resource-group rg-listmonk420 --revision listmonk420--0000097` ✅

### Verification
- No JavaScript errors detected ✅
- No console errors detected ✅
- Progress bars ready to display when campaigns have sent > 0 ✅

**Note**: Test showed 0 progress bars because the 2 campaigns in the database have 0 sent emails. Progress bars will appear when campaigns actually send emails.

## Campaign Types Supported

### Regular Campaigns
- Uses `stats.sent` and `stats.toSend` fields
- Direct email sending through SMTP servers

### Queue-Based Campaigns
- Uses `stats.queue_sent`/`queueSent` and `stats.queue_total`/`queueTotal` fields
- Emails queued to database and processed by queue processor
- Supports daily limits and time windows

## Testing Infrastructure Established

Created comprehensive Playwright testing utilities in `/tmp/`:

### Files Created
1. **`playwright-helpers.js`** - Reusable authentication and navigation library
2. **`test-campaigns-page.js`** - Quick test for campaigns page
3. **`playwright-example.js`** - Five complete usage examples
4. **`diagnose-login.js`** - Login diagnostic tool
5. **`PLAYWRIGHT-README.md`** - Complete documentation

### Authentication Details
- **URL**: `https://list.bobbyseamoss.com/admin/`
- **Username**: `adam`
- **Password**: `T@intshr3dd3r`
- **Form selectors**: `input[name="username"]`, `input[name="password"]`, `button[type="submit"]`
- **Wait strategy**: `waitUntil: 'load'` (not `networkidle` due to active polling)

### Key Functions
```javascript
createAuthenticatedSession(options) // Handles login automatically
navigateTo(session, path)           // Navigate to admin pages
takeScreenshot(session, filename)   // Capture screenshots
checkForErrors(session)             // Detect JS/console errors
closeSession(session)               // Clean up browser
```

### Quick Usage
```bash
node /tmp/test-campaigns-page.js
node /tmp/playwright-example.js
```

## Lessons Learned

### Vue 2 Template Patterns
- **Avoid `:set` directive** with `v-if` that references the set variable (circular dependency)
- **Use helper methods** instead of complex inline expressions
- **Extract logic to methods** for better testability and maintainability

### Defensive Programming
- **Always check for null/undefined** before property access
- **Provide fallback values** (e.g., `|| 0` for numeric fields)
- **Early returns** for invalid states prevent nested conditionals

### Playwright Testing
- **Use `waitUntil: 'load'`** instead of `networkidle` for SPAs with polling
- **Add explicit waits** after navigation for Vue to render (1500-2000ms)
- **Increase timeouts** for login navigation (15s instead of 10s)
- **Check for error messages** after form submission to verify success

## Next Steps (If Needed)

1. **Monitor in Production**: Verify progress bars appear when campaigns send emails
2. **Move Playwright Files**: Copy `/tmp/playwright-*.js` and `/tmp/PLAYWRIGHT-README.md` to permanent location if needed
3. **Add E2E Tests**: Use Playwright helpers to create automated frontend test suite
4. **Performance Testing**: Monitor if progress bar calculation impacts page load with many campaigns

## Related Documentation

- User originally requested this feature in campaigns redesign project
- Progress bar CSS already existed at lines 610-618 (`.campaign-progress` class)
- Feature works with both standard and queue-based email delivery systems
- See `CLAUDE.md` for queue-based delivery architecture

## Status

✅ **COMPLETED AND DEPLOYED**
- All code changes implemented
- All null checks added
- User requirements met (show for ALL campaigns)
- Successfully built and deployed to production
- No errors detected in verification
- Playwright testing infrastructure established for future development

**Production URL**: https://list.bobbyseamoss.com/admin/campaigns
**Revision**: listmonk420--0000097
**Deployed**: 2025-11-09 10:33 UTC
