# Sliding Window Bug Fix - Implementation Context

**Last Updated**: 2025-11-28
**Status**: COMPLETED - Deployed to Production

## Bug Description

The queue processor was incorrectly enforcing a sliding window rate limit of 16 emails/minute even when the feature was disabled in settings (`app.message_sliding_window = false`).

### Symptoms
- Error logs: `no SMTP server available for email X - all servers at capacity`
- Queue processing slowed/stopped despite unlimited capacity settings
- Only appeared when using queue-based campaigns

### Root Cause

In `cmd/init.go` line 762 (before fix), the queue processor config was unconditionally setting:

```go
SlidingWindowLimit: settings.AppMessageSlidingWindowRate,
```

This set the limit to 16 (from `app.message_sliding_window_rate = 16`) regardless of whether `app.message_sliding_window = false`.

## Fix Implementation

**File**: `cmd/init.go` (lines 746-768)

**Before** (buggy):
```go
func initQueueProcessor(db *sqlx.DB, settings models.Settings) *queue.Processor {
    var slidingDuration time.Duration
    if settings.AppMessageSlidingWindowDuration != "" {
        d, err := time.ParseDuration(settings.AppMessageSlidingWindowDuration)
        if err == nil {
            slidingDuration = d
        }
    }

    cfg := queue.Config{
        PollInterval:          time.Second * 5,
        BatchSize:             100,
        TimeWindowStart:       settings.AppSendTimeStart,
        TimeWindowEnd:         settings.AppSendTimeEnd,
        SlidingWindowDuration: slidingDuration,
        SlidingWindowLimit:    settings.AppMessageSlidingWindowRate,  // BUG: Always set!
    }
    // ...
}
```

**After** (fixed):
```go
func initQueueProcessor(db *sqlx.DB, settings models.Settings) *queue.Processor {
    var slidingDuration time.Duration
    var slidingLimit int
    if settings.AppMessageSlidingWindow {
        // Only apply sliding window settings if the feature is enabled
        if settings.AppMessageSlidingWindowDuration != "" {
            d, err := time.ParseDuration(settings.AppMessageSlidingWindowDuration)
            if err == nil {
                slidingDuration = d
            }
        }
        slidingLimit = settings.AppMessageSlidingWindowRate
    }

    cfg := queue.Config{
        PollInterval:          time.Second * 5,
        BatchSize:             100,
        TimeWindowStart:       settings.AppSendTimeStart,
        TimeWindowEnd:         settings.AppSendTimeEnd,
        SlidingWindowDuration: slidingDuration,
        SlidingWindowLimit:    slidingLimit,  // Now 0 when disabled
    }
    // ...
}
```

## How the Sliding Window Works

In `internal/queue/processor.go`, the `getServerCapacities()` function checks:

```go
capacity.CanSendNow = (capacity.DailyRemaining > 0 || capacity.DailyLimit == 0) &&
    (capacity.SlidingWindowUsed < capacity.SlidingWindowLimit || capacity.SlidingWindowLimit == 0)
```

When `SlidingWindowLimit = 0`, the condition `|| capacity.SlidingWindowLimit == 0` makes it always pass.

## Database Settings Reference

```sql
SELECT key, value FROM settings WHERE key LIKE '%sliding%';
```

| Key | Value | Description |
|-----|-------|-------------|
| app.message_sliding_window | false | Feature toggle (MASTER SWITCH) |
| app.message_sliding_window_rate | 16 | Rate limit when enabled |
| app.message_sliding_window_duration | "1m" | Window duration when enabled |

## Deployment

1. Built Docker image: `listmonk420acr.azurecr.io/listmonk420:latest`
2. Pushed to ACR
3. Deployed to Azure Container Apps
4. Revision: `listmonk420--deploy-20251126-093256`
5. Status: Healthy, 100% traffic

## Verification

Logs show capacity is now unlimited when sliding window disabled:
```
📧 email X → SMTP server 'email-mail2' | capacity: 999999999/0 daily remaining
```

The `999999999/0` format means:
- `999999999` = Effective unlimited daily remaining (when daily_limit=0)
- `0` = Daily limit setting (0 means unlimited)

## Related Components

- `internal/queue/processor.go` - Queue processing logic
- `internal/queue/models.go` - ServerCapacity struct
- `cmd/init.go` - Queue processor initialization
- `models/settings.go` - Settings struct definitions
