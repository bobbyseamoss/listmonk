# Test Mode Clear Queue Feature

## Overview
When Test Mode is enabled, the "Clear Queue" function will additionally reset SMTP server capacity and remove canceled emails, making it easy to run multiple test campaigns without hitting daily limits or accumulating canceled emails.

## Behavior

### Normal Mode (Test Mode OFF)
When you click "Clear Queue":
- Cancels all queued emails
- Sets their status to 'cancelled'

### Test Mode (Test Mode ON)
When you click "Clear Queue":
- Cancels all queued emails (same as normal)
- **Deletes all canceled emails** (removes them from database entirely)
- **Resets all SMTP daily usage counters** (clears smtp_daily_usage table)

## Benefits
- **Clean slate for testing**: No need to manually reset daily limits
- **Reduced clutter**: Canceled emails are removed instead of accumulating
- **Faster iteration**: Test multiple campaigns without restarting or manual cleanup

## UI Feedback
Success messages will show additional information when Test Mode is active:
- Normal: "Successfully cleared 500 queued emails"
- Test Mode: "Successfully cleared 500 queued emails (Test Mode: 200 canceled emails deleted, SMTP capacity reset)"

## Backend Implementation
**File**: `cmd/queue.go` (ClearAllQueuedEmails function)

```go
if testMode {
    // Delete all canceled emails
    a.db.Exec(`DELETE FROM email_queue WHERE status = 'cancelled'`)

    // Reset SMTP daily usage
    a.db.Exec(`DELETE FROM smtp_daily_usage`)
}
```

## Frontend Implementation
**File**: `frontend/src/views/Queue.vue` (confirmClearQueue method)

The frontend displays extended feedback when `test_mode` is true in the API response.

## How to Enable
1. Go to **Settings → Performance**
2. Enable **"Test Mode"** switch
3. Click **Save**
4. Now when you clear the queue, SMTP capacity and canceled counts will also reset

## Safety
- This feature only affects local development
- Test Mode should **never** be enabled in production
- All operations are logged in the backend

## Technical Details
### Database Operations
1. `UPDATE email_queue SET status = 'cancelled' WHERE status = 'queued'` (normal operation)
2. `DELETE FROM email_queue WHERE status = 'cancelled'` (test mode only)
3. `DELETE FROM smtp_daily_usage` (test mode only)

### API Response
```json
{
  "success": true,
  "count": 500,
  "test_mode": true,
  "canceled_count": 200,
  "reset_capacity": true
}
```
