# Bug Fix: Queue Processor Not Respecting Sliding Window Limits Within Batch

## Problem
The queue processor was sending all 25 emails in a batch to the same SMTP server, completely ignoring the sliding window limit of 4 emails per 30 minutes.

### Observed Behavior
- Started campaign with automatic messenger
- Queue processor immediately sent 25 emails to ONE server (mail29) in 200ms
- Time range: 15:05:38.205135 to 15:05:38.401181
- Other 28 servers received 0 emails
- Violates configured limit: 4 emails per 30 minutes per server

## Root Cause
In `internal/queue/processor.go`, function `processQueue()`:

1. Server capacities were fetched **once** at the beginning of batch processing
2. The same `capacities` map was used for all 25 emails in the loop
3. The capacities map was **never updated** during the loop
4. `selectServer()` repeatedly returned the same "best" server (one with most daily capacity)
5. All emails were sent to that server without checking sliding window limits within the batch

### Code Flow (BEFORE FIX)
```go
capacities := getServerCapacities()  // Called ONCE

for each email in batch {
    server = selectServer(capacities)  // Returns same server every time!
    send(email, server)
    // capacities never updated!
}
```

## Solution
Added **batch-level tracking** to respect sliding window limits within a single batch:

###Changes in `processQueue()` function:
1. Added `batchUsage` map to track emails sent per server within the batch
2. Before sending, check if server has reached its sliding window limit for this batch
3. After sending, update both `batchUsage` and `capacities`
4. Decrement `DailyRemaining` in the capacities map after each send

### Code Flow (AFTER FIX)
```go
capacities := getServerCapacities()
batchUsage := make(map[string]int)  // NEW: Track batch usage

for each email in batch {
    server = selectServer(capacities)

    // NEW: Check batch limit
    if batchUsage[server] >= slidingWindowLimit {
        continue  // Skip, this server hit its limit for this batch
    }

    send(email, server)

    // NEW: Update tracking
    batchUsage[server]++
    capacities[server].DailyRemaining--
    if capacities[server].DailyRemaining <= 0 {
        capacities[server].CanSendNow = false
    }
}
```

## Implementation Details

**File**: `internal/queue/processor.go`

**Function**: `processQueue()` (lines 99-187)

**Key Changes**:
1. Line 127: Added `batchUsage := make(map[string]int)`
2. Lines 138-145: Added sliding window check before sending
3. Lines 173-183: Update batch usage and capacity after each send

```go
// Line 138-145: NEW sliding window check
if cap, exists := capacities[serverUUID]; exists && cap.SlidingWindowLimit > 0 {
    if batchUsage[serverUUID] >= cap.SlidingWindowLimit {
        continue  // Skip this email, server at batch limit
    }
}

// Lines 173-183: NEW capacity tracking
batchUsage[serverUUID]++
if cap, exists := capacities[serverUUID]; exists {
    if cap.DailyLimit > 0 {
        cap.DailyRemaining--
        if cap.DailyRemaining <= 0 {
            cap.CanSendNow = false
        }
    }
}
```

## Expected Behavior After Fix

With batch size = 100 and sliding window limit = 4:

**Batch 1** (emails 1-100):
- Server 1: 4 emails
- Server 2: 4 emails
- Server 3: 4 emails
- ...
- Server 25: 4 emails
- Total: 100 emails across 25 servers (4 each)

**Batch 2** (next minute):
- Same distribution pattern
- Each server respects its 4-email-per-30-minute limit

## Testing
1. Enable Test Mode in Settings → Performance
2. Create campaign with automatic messenger and 100+ subscribers
3. Start campaign
4. Check Queue page - emails should be distributed across ALL servers
5. Each server should have ≤ 4 emails sent per batch
6. Use SQL to verify:
```sql
SELECT assigned_smtp_server_uuid, COUNT(*), MIN(sent_at), MAX(sent_at)
FROM email_queue
WHERE status = 'sent' AND sent_at > NOW() - INTERVAL '1 minute'
GROUP BY assigned_smtp_server_uuid
ORDER BY COUNT(*) DESC;
```

Expected result: Multiple servers, each with ~4 emails (not 25+ to one server)

## Related Files
- `internal/queue/processor.go` - Main fix
- `internal/queue/models.go` - ServerCapacity struct definition
- `internal/queue/scheduler.go` - Initial email assignment (not affected)

## Impact
- ✅ Respects sliding window limits within batches
- ✅ Distributes emails across all available servers
- ✅ Prevents overwhelming single server
- ✅ More efficient use of SMTP server capacity
- ✅ Follows configured rate limits correctly
