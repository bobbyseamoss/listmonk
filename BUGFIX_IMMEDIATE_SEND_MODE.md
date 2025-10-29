# Bug Fix: Queue Scheduler Not Sending First Round Immediately

## Problem
When creating a new campaign with the automatic (queue-based) messenger, emails were scheduled hours into the future instead of being sent immediately. For example, a 1,218-email campaign was spread from 14:37 to 20:08 (almost 6 hours).

### Observed Behavior
- Campaign created at 14:37:34
- First email scheduled for 14:37:34
- Last email scheduled for 20:08:47
- 6-hour spread for ~1,200 emails
- User expected immediate sending with processor handling rate limiting

## Root Cause
The scheduler in `internal/queue/scheduler.go` was designed to spread emails over time based on:
1. Daily capacity calculations
2. Send rate per minute
3. Time window boundaries

Even when no time windows were configured and capacity was unlimited, the scheduler still incremented `scheduled_at` times for each email:

```go
// OLD CODE
for _, email := range emails {
    // ... assign server ...

    // Update email with scheduled time
    UPDATE email_queue SET scheduled_at = $1 WHERE id = $2

    // Advance time for next email
    if sendRatePerMinute > 0 {
        secondsPerEmail := 60.0 / float64(sendRatePerMinute)
        currentTime = currentTime.Add(time.Duration(secondsPerEmail * float64(time.Second)))
    } else {
        currentTime = currentTime.Add(1 * time.Second)
    }
}
```

This caused emails to be scheduled seconds/minutes apart, spreading them over hours.

## Solution
Added **Immediate Mode** detection to the scheduler:

### When Immediate Mode Activates
```go
immediateMode := (s.cfg.TimeWindowStart == "" || s.cfg.TimeWindowEnd == "") &&
                  totalCapacity >= len(emails)
```

Immediate mode activates when:
1. **No time window configured** (24/7 sending allowed)
2. **Sufficient capacity** (servers can handle all emails today)

### Behavior in Immediate Mode
- All emails scheduled for NOW (same `scheduled_at` time)
- No time incrementing between emails
- Server assignment still uses round-robin
- **Processor handles rate limiting in real-time** using sliding window checks

### Code Changes in `scheduler.go`

**Lines 138-147**: Added immediate mode detection
```go
// Check if we should schedule for immediate sending
immediateMode := (s.cfg.TimeWindowStart == "" || s.cfg.TimeWindowEnd == "") &&
                  totalCapacity >= len(emails)

if immediateMode {
    s.log.Printf("immediate mode: scheduling all %d emails for NOW (processor will handle rate limiting)", len(emails))
} else {
    s.log.Printf("scheduled mode: send rate %d emails/min, starting at %s", sendRatePerMinute, startTime.Format("2006-01-02 15:04:05"))
}
```

**Lines 275-289**: Conditional time advancement
```go
// Advance time based on mode
if immediateMode {
    // In immediate mode, keep all emails scheduled for NOW
    // The processor will handle rate limiting in real-time
    currentTime = startTime
} else {
    // In scheduled mode, spread emails over time based on send rate
    if sendRatePerMinute > 0 {
        secondsPerEmail := 60.0 / float64(sendRatePerMinute)
        currentTime = currentTime.Add(time.Duration(secondsPerEmail * float64(time.Second)))
    } else {
        currentTime = currentTime.Add(1 * time.Second)
    }
}
```

**Line 292**: Skip time window checking in immediate mode
```go
// Check if we've exceeded the sending window for today (only in scheduled mode)
if !immediateMode && !s.isWithinSendingWindow(currentTime) {
    // Move to next day's sending window
    currentTime = s.getNextSendingWindow()
    // ...
}
```

## Expected Behavior After Fix

### Immediate Mode (No Time Windows)
- Campaign with 1,218 emails
- All scheduled for NOW: 14:37:34
- Queue processor picks up first batch immediately
- Respects sliding window: 4 emails per server per 30 minutes
- Distributes across all 30 servers
- **First 120 emails sent in first batch** (30 servers × 4 emails)
- Next batch processed after poll interval (30 seconds default)

### Scheduled Mode (With Time Windows)
- Continues existing behavior
- Spreads emails over time
- Respects time windows (e.g., 08:00-20:00)
- Useful for multi-day campaigns

## Log Examples

### Immediate Mode
```
2025/10/29 15:37:34 scheduler.go:144: immediate mode: scheduling all 1218 emails for NOW (processor will handle rate limiting)
2025/10/29 15:37:34 scheduler.go:310: successfully scheduled 1218 emails for campaign 7
```

### Scheduled Mode
```
2025/10/29 10:00:00 scheduler.go:146: scheduled mode: send rate 50 emails/min, starting at 2025-10-29 10:00:00
2025/10/29 10:00:00 scheduler.go:310: successfully scheduled 10000 emails for campaign 8
```

## Processor Behavior (Unchanged)
The queue processor (`internal/queue/processor.go`) still handles:
- Batch processing (100 emails per poll)
- Sliding window enforcement (4 emails per 30 minutes per server)
- Server capacity tracking
- Round-robin distribution

The key difference:
- **Before**: Scheduler pre-spread emails, processor just executed schedule
- **After (Immediate Mode)**: Scheduler schedules all for NOW, processor handles all rate limiting

## Testing

1. Clear queue and reset capacity (Test Mode)
2. Create campaign with automatic messenger
3. Check email_queue table:
   ```sql
   SELECT COUNT(*),
          MIN(scheduled_at AT TIME ZONE 'UTC') as first,
          MAX(scheduled_at AT TIME ZONE 'UTC') as last
   FROM email_queue
   WHERE campaign_id = X AND status = 'queued';
   ```
   **Expected**: `first` == `last` (all scheduled for same time)

4. Check queue processor logs for "immediate mode"
5. Verify emails sent in batches respecting sliding window

## Benefits
- ✅ Emails start sending immediately
- ✅ Better user experience (no 6-hour waits)
- ✅ Processor still respects all rate limits
- ✅ Still supports scheduled mode for time-window campaigns
- ✅ No breaking changes to existing functionality

## Related Files
- `internal/queue/scheduler.go` - Main fix (lines 138-147, 275-289, 292)
- `internal/queue/processor.go` - Unchanged (handles rate limiting)
- `internal/core/campaigns.go` - Calls scheduler (unchanged)
