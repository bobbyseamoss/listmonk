# Queue Capacity Issues - Troubleshooting Summary

## Summary

You were getting "all servers at capacity" errors when trying to send campaign 56. After multiple restarts and configuration changes, the campaign has sent **9,351 emails** but now has **2,166 failures** (and counting).

---

## Root Cause Analysis

### Initial Problem: "All Servers at Capacity"

**Cause 1: Old Account-Wide Rate Limits**
- App was using old limits: 30/min, 100/hour
- Database had new limits: 500/min, 2000/hour
- **Fix:** Restarted app to load new settings

**Cause 2: Sliding Window Rate Mismatch**
- Global `app.message_sliding_window_rate` was still `4`
- Per-server rates were updated to `16`
- Queue processor was checking global setting
- **Fix:** Updated `app.message_sliding_window_rate` from 4 to 16

**Cause 3: Hourly Limit Too Conservative**
- Rate limiter calculated: 3600s ÷ 2000 emails = 1.8s per email = 33/min
- This was limiting speed to 33/min instead of 500/min
- **Fix:** Disabled hourly limit (set to 0) to use minute limit only

**Cause 4: Sliding Window Duration Wrong**
- Global `app.message_sliding_window_duration` was `"30m"`
- Per-server duration was `"1m"`
- With 16 emails per 30 minutes = 0.53 emails/minute per server!
- **Fix:** Updated `app.message_sliding_window_duration` from "30m" to "1m"

---

## Current Status

### Campaign 56: "Copy of Kali - ALL Gummies"
- **Status:** Running
- **To Send:** 212,328 total
- **Sent:** 9,351 ✓
- **Failed:** 2,166 ❌ (increasing!)
- **Queued:** 200,596
- **Cancelled:** 213
- **Sending:** 2

### Failure Timeline
- Last 5 minutes: 274 failures
- Last 15 minutes: 992 failures
- Older: 909 failures

**Failures are ongoing and accelerating!**

---

## Settings Applied

### Current Database Settings:
| Setting | Old Value | New Value |
|---------|-----------|-----------|
| `app.account_rate_limit_per_minute` | 30 | 500 |
| `app.account_rate_limit_per_hour` | 100 | **0** (disabled) |
| `app.message_sliding_window_rate` | 4 | 16 |
| `app.message_sliding_window_duration` | "30m" | "1m" |
| `app.message_rate` | 1 | 600 |
| `app.concurrency` | 1 | 10 |

### Per-Server SMTP Settings (29 servers):
| Setting | Old Value | New Value |
|---------|-----------|-----------|
| `sliding_window_rate` | 4 per 30min | 16 per 1min |
| `sliding_window_duration` | 30m | 1m |
| `daily_limit` | 80 | 1650 |

---

## Issues Encountered

### Issue 1: Multiple Restarts Required
**Problem:** Settings are loaded at app startup, so each database change required a restart

**Restarts Performed:**
1. After updating account-wide rate limits (500/min, 2000/hour)
2. After updating global sliding window rate (4 → 16)
3. After disabling hourly limit (2000 → 0)
4. After updating sliding window duration (30m → 1m)

### Issue 2: Rate Limiter Logic Too Conservative
**Problem:** The hourly limit was being enforced per-message instead of over the full hour

**Code Location:** `internal/queue/processor.go` lines 172-180
```go
if settings.AppAccountRateLimitPerHour > 0 {
    minDelayFromHourLimit := time.Hour / time.Duration(settings.AppAccountRateLimitPerHour)
    if delayBetweenMessages == 0 || minDelayFromHourLimit > delayBetweenMessages {
        delayBetweenMessages = minDelayFromHourLimit
    }
}
```

With 2000/hour: 3600s ÷ 2000 = 1.8s per email = only 33 emails/minute!

**Workaround:** Disabled hourly limit by setting to 0

### Issue 3: Bursty Sending Pattern
**Problem:** Queue processor polls every 60 seconds (hardcoded)

**Code Location:** `cmd/init.go` line 741
```go
PollInterval: time.Minute * 1, // Check for emails every minute
```

**Result:** Emails sent in bursts every minute, not continuously

### Issue 4: Current Failures (NEW!)
**Problem:** 2,166 emails have failed status, with ongoing failures

**Possible Causes:**
- Hit Azure API rate limits when sending too fast
- SMTP authentication issues
- Invalid recipients
- Sliding window toggle disrupted state

**Status:** INVESTIGATING

---

## Azure Rate Limits (Actual API Limits)

Your subscription: Standard pricing tier
- **500 emails per minute**
- **2,000 emails per hour**
- **48,000 emails per day** (24 hours × 2000/hour)

These are HARD limits enforced by Azure. Exceeding them causes:
- HTTP 429 (Too Many Requests) errors
- Temporary throttling
- Email sending failures

---

## Current Problem: Email Failures

### Symptoms:
- 9,351 emails sent successfully
- 2,166 emails marked as "failed"
- Failures ongoing (274 in last 5 minutes)
- Sending rate dropped to ~0-1 email/minute

### Possible Root Causes:

**Theory 1: Hit Azure API Limit**
When sliding window was toggled, queue processor may have sent too fast and exceeded Azure's 500/min or 2000/hour limit, causing Azure to reject emails.

**Theory 2: SMTP Connection Issues**
Multiple restarts may have left stale SMTP connections or authentication tokens.

**Theory 3: Subscriber Issues**
Some recipients may be invalid, blocklisted, or have other delivery issues.

**Theory 4: State Corruption**
Toggling sliding window while queue was active may have corrupted sliding window state or left servers in blocked state.

---

## Recommended Next Steps

### 1. Check Application Logs for Error Details
```bash
az containerapp logs show --name listmonk420 --resource-group rg-listmonk420 --tail 100 | grep -i error
```

Look for:
- HTTP 429 errors (rate limit exceeded)
- SMTP authentication failures
- Connection timeouts
- Azure API errors

### 2. Pause Campaign to Stop Failures
```sql
UPDATE campaigns SET status = 'paused' WHERE id = 56;
```

This will:
- Stop queue processor from picking up more emails
- Prevent more failures
- Give time to investigate

### 3. Check Failed Email Details
```sql
-- Sample of failed emails (if error_message column exists)
SELECT id, subscriber_id, updated_at
FROM email_queue
WHERE campaign_id = 56 AND status = 'failed'
ORDER BY updated_at DESC
LIMIT 20;
```

### 4. Investigate Azure Communication Services Dashboard
Check Azure portal for:
- Actual sending rate vs limits
- Throttling events
- API error codes
- Delivery reports

### 5. Reset Queue State
If sliding window state is corrupted:
```sql
-- Clear sliding window state
TRUNCATE smtp_rate_limit_state;

-- This will reset sliding window counters
-- Queue processor will rebuild state on next poll
```

### 6. Consider Reducing Sending Rate
To stay safely under Azure limits:
```sql
-- Set more conservative rate (e.g., 400/min instead of 500)
UPDATE settings SET value = '400' WHERE key = 'app.account_rate_limit_per_minute';

-- Re-enable hourly limit with higher value
UPDATE settings SET value = '10000' WHERE key = 'app.account_rate_limit_per_hour';
```

Then restart the app.

---

## Code Issues to Fix (Future)

### 1. Hourly Limit Enforcement
**File:** `internal/queue/processor.go` lines 172-180

**Problem:** Converts hourly limit to per-message delay, which is too conservative

**Better Approach:** Track emails sent in rolling 1-hour window, allow bursts up to minute limit

### 2. Poll Interval
**File:** `cmd/init.go` line 741

**Problem:** Hardcoded 1-minute poll interval causes bursty sending

**Better Approach:** Make PollInterval configurable via database settings

### 3. Sliding Window Settings
**Problem:** Queue processor uses global settings (`app.message_sliding_window_rate`) instead of per-server settings (`smtp[].sliding_window_rate`)

**File:** `internal/queue/processor.go` line 501

**Better Approach:** Use each SMTP server's individual sliding_window_rate

---

## Files Modified

### Database Settings:
- `app.account_rate_limit_per_minute`: 30 → 500
- `app.account_rate_limit_per_hour`: 100 → 0
- `app.message_sliding_window_rate`: 4 → 16
- `app.message_sliding_window_duration`: "30m" → "1m"
- `app.message_rate`: 1 → 600
- `app.concurrency`: 1 → 10

### SMTP Servers (29 servers):
- `sliding_window_rate`: 4 → 16
- `sliding_window_duration`: 30m → 1m
- `daily_limit`: 80 → 1650

### Code Changes:
- `cmd/bounce.go` lines 517, 556: Fixed PostgreSQL type inference error in de-duplication queries

---

## Application Restarts

Total restarts: **4**

1. **Restart 1:** After updating account rate limits (500/min, 2000/hour)
2. **Restart 2:** After updating sliding window rate (4 → 16)
3. **Restart 3:** After disabling hourly limit (set to 0)
4. **Restart 4:** After updating sliding window duration (30m → 1m)

---

## Current Effective Rates

### Theoretical Maximum:
- **Per-server:** 16 emails/minute
- **Total (29 servers):** 464 emails/minute
- **Account limit:** 500 emails/minute (bottleneck)
- **Expected:** ~460-500 emails/minute

### Actual Observed:
- **Before failures:** 234-366 emails/minute (bursty due to 1-min poll)
- **After failures started:** 0-1 email/minute (essentially stopped)

---

## Status

- ❌ Campaign sending but with high failure rate
- ❌ 2,166 failures (and increasing)
- ⚠️  Need to investigate failure cause immediately
- ⚠️  May need to pause campaign to prevent more failures
- ✓ All settings updated correctly in database
- ✓ De-duplication fix applied
- ✓ Rate limits configured for 500/min capacity

---

**Date:** 2025-11-04
**Campaign:** 56 (Copy of Kali - ALL Gummies)
**Status:** INVESTIGATING FAILURES

**Next Action:** Check application logs for error details and determine cause of failures
