# Queue Campaign Management - Context & Decisions

**Last Updated: 2025-11-07 (Session 2 - Hard Delete Change)**

## Project Context

This enhancement adds critical functionality to listmonk's queue-based campaign system, fixing a major bug in campaign pause/resume and providing tools for preventing duplicate email sends.

## CURRENT SESSION (2025-11-07 Evening) - UNCOMMITTED CHANGES

**Status**: Code changes complete, NOT YET DEPLOYED
**User Request**: Change Remove Sent Subscribers from soft-delete (unsubscribe) to hard-delete (remove from list)

### Changes Made (Uncommitted)

**Backend Change** (`/home/adam/listmonk/cmd/campaigns.go`):
- Line 844-845: Updated comment to clarify "hard delete from subscriber_lists table"
- Line 900: Changed `a.core.UnsubscribeLists(subIDs, listIDs, nil)` → `a.core.DeleteSubscriptions(subIDs, listIDs)`
- Line 909: Changed message "Unsubscribed X subscribers" → "Removed X subscribers"

**Frontend Change** (`/home/adam/listmonk/frontend/src/views/Campaign.vue`):
- Line 186: Changed "Unsubscribe all subscribers..." → "Remove all subscribers..."

### What This Changes

**OLD BEHAVIOR (Deployed in Production)**:
- Calls `UnsubscribeLists()` which executes `unsubscribe-subscribers-from-lists` query
- SQL: `UPDATE subscriber_lists SET status='unsubscribed'`
- Result: Soft delete - subscribers marked as unsubscribed but row remains

**NEW BEHAVIOR (Code Ready, Not Deployed)**:
- Calls `DeleteSubscriptions()` which executes `delete-subscriptions` query
- SQL: `DELETE FROM subscriber_lists WHERE (subscriber_id, list_id) = ANY(...)`
- Result: Hard delete - rows completely removed from subscriber_lists table

### Deployment Plan

**User will deploy tonight after pausing current campaign**

Deployment commands:
```bash
cd frontend && yarn build
cd ..
./deploy.sh
```

**Why waiting**: User has active queue-based campaign running and wants to pause it first before deploying changes.

### Testing After Deployment

1. Run queue-based campaign
2. Click "Remove Sent Subscribers" button
3. Query database to verify rows deleted (not just status changed):
   ```sql
   SELECT COUNT(*) FROM subscriber_lists
   WHERE subscriber_id IN (SELECT DISTINCT subscriber_id FROM email_queue WHERE campaign_id = X AND status = 'sent')
     AND list_id IN (campaign's list IDs);
   -- Should return 0 after removal
   ```

### Related Code Paths

**DeleteSubscriptions Core Method** (`/home/adam/listmonk/internal/core/subscriptions.go:54-63`):
```go
func (c *Core) DeleteSubscriptions(subIDs, listIDs []int) error {
    if _, err := c.q.DeleteSubscriptions.Exec(pq.Array(subIDs), pq.Array(listIDs)); err != nil {
        c.log.Printf("error deleting subscriptions: %v", err)
        return echo.NewHTTPError(http.StatusInternalServerError, ...)
    }
    return nil
}
```

**SQL Query** (`/home/adam/listmonk/queries.sql:222-224`):
```sql
-- name: delete-subscriptions
DELETE FROM subscriber_lists
    WHERE (subscriber_id, list_id) = ANY(SELECT a, b FROM UNNEST($1::INT[]) a, UNNEST($2::INT[]) b);
```

## Key Files Modified

### Backend Files

**`/home/adam/listmonk/queries.sql`**
- Lines 1605-1610: `get-sent-subscribers-today` query
- Lines 1617-1624: `requeue-cancelled-emails` query
- Purpose: Core SQL operations for new features
- Note: "today" in name is legacy; query is all-time

**`/home/adam/listmonk/models/queries.go`**
- Line 108: `GetSentSubscribersToday *sqlx.Stmt`
- Line 109: `RequeueCancelledEmails *sqlx.Stmt`
- Purpose: Register prepared statements for runtime use

**`/home/adam/listmonk/internal/core/campaigns.go`**
- Lines 310-337: Modified `UpdateCampaignStatus` with resume detection
- Lines 604-616: New `RequeueCancelledEmails` method
- Purpose: Core business logic for campaign lifecycle

**`/home/adam/listmonk/cmd/campaigns.go`**
- Lines 844-911: `RemoveSentSubscribersFromLists` HTTP handler
- Purpose: API endpoint for removing sent subscribers from lists
- **MODIFIED 2025-11-07 Evening**: Changed from UnsubscribeLists (soft delete) to DeleteSubscriptions (hard delete)
- **Status**: Changes uncommitted, pending deployment tonight

**`/home/adam/listmonk/cmd/handlers.go`**
- Line 179: POST route registration
- Purpose: Wire HTTP endpoint to handler

### Frontend Files

**`/home/adam/listmonk/frontend/src/api/index.js`**
- Lines 408-412: `removeSentSubscribersFromLists` API method
- Purpose: Type-safe API client method

**`/home/adam/listmonk/frontend/src/views/Campaign.vue`**
- Lines 181-194: UI box with Remove Sent button
- Lines 681-697: `removeSentFromLists` method implementation
- Purpose: User interface for feature access
- **MODIFIED 2025-11-07 Evening**: Updated description text from "Unsubscribe" to "Remove"
- **Status**: Changes uncommitted, pending deployment tonight

**`/home/adam/listmonk/frontend/src/assets/icons/fontello.css`**
- Lines 121-122: Play and pause icon definitions
- Purpose: Consistent campaign control icons

## Architecture Decisions

### Decision 1: Resume Detection Strategy
**Choice**: Detect pause→running transition in `UpdateCampaignStatus`

**Rationale:**
- Centralizes all status change logic in one place
- Leverages existing campaign fetch to check current status
- No new API endpoints required
- Consistent with existing campaign lifecycle management

**Alternatives Considered:**
- Separate "Resume" endpoint: Rejected (unnecessary API surface)
- Frontend-triggered re-queue: Rejected (puts logic in wrong layer)

**Trade-offs:**
- ✅ Simple implementation
- ✅ No breaking changes
- ⚠️ Slight increase in UpdateCampaignStatus complexity

### Decision 2: All-Time vs Today-Only Filtering
**Initial Choice**: Remove sent subscribers from "today" only

**Rationale:**
- Matches user's initial request
- Prevents accidental removal of historical data
- More conservative approach

**User Feedback**: "Maybe it should just be Remove Sent regardless of when they were emailed"

**Final Choice**: Remove ALL sent subscribers (all-time)

**Rationale:**
- User's actual use case: prevent duplicates in future campaigns
- "Today" filtering based on timezone caused confusion (server UTC vs user EST)
- Simpler query without timezone conversions
- More powerful tool for list management

**Trade-offs:**
- ✅ Eliminates timezone issues
- ✅ More useful for duplicate prevention
- ✅ Simpler query logic
- ⚠️ More permanent action (mitigated by confirmation dialog)

### Decision 3: Soft Delete Pattern
**Choice**: Unsubscribe (status change) instead of hard delete

**Rationale:**
- Preserves audit trail in subscriber_lists table
- Consistent with listmonk's existing unsubscribe pattern
- Can be reversed via database if needed
- Maintains referential integrity

**Alternatives Considered:**
- Hard delete from subscriber_lists: Rejected (data loss)
- New "removed_by_campaign" status: Rejected (unnecessary complexity)

**Trade-offs:**
- ✅ Reversible
- ✅ Audit trail preserved
- ✅ Reuses existing code path
- ⚠️ Requires understanding of soft-delete pattern

### Decision 4: Async vs Sync Re-queue
**Choice**: Re-queue synchronously on resume, queue asynchronously on new start

**Rationale:**
- New campaign start: Large operation (thousands of rows), use goroutine
- Resume: Smaller operation (only cancelled emails), return count to user
- User expects immediate feedback on resume
- Existing async pattern for new campaigns works well

**Trade-offs:**
- ✅ Better UX (resume count in logs)
- ✅ No blocking on new campaign start
- ⚠️ Mixed async/sync patterns (mitigated by clear comments)

### Decision 5: UI Placement
**Choice**: Add button to existing campaign edit page

**Rationale:**
- Feature is campaign-specific, not global
- User already navigates to campaign page for management
- No new routes or navigation changes
- Conditional visibility (only queue-based campaigns)

**Alternatives Considered:**
- Separate "Campaign Tools" page: Rejected (over-engineering)
- Bulk action in campaigns list: Rejected (doesn't fit existing UI)
- Settings page: Rejected (not a setting)

**Trade-offs:**
- ✅ Discoverable location
- ✅ Context-appropriate
- ✅ No routing changes
- ⚠️ Campaign page gets slightly more complex

## Dependencies

### Database Schema
**Tables Used:**
- `email_queue`: Source of truth for sent emails
- `campaigns`: Messenger type and lists
- `subscriber_lists`: Target for unsubscribe operations
- `lists`: Campaign targeting

**Required Indexes:**
- email_queue(campaign_id, status): For efficient sent subscriber lookup
- email_queue(campaign_id, subscriber_id): For re-queue exclusion
- subscriber_lists(subscriber_id, list_id): For bulk unsubscribe

### External Services
- **Azure Container Apps**: Deployment platform
- **Azure Container Registry**: Docker image storage
- **Azure Database for PostgreSQL**: Data storage

### Third-Party Libraries
- **sqlx**: Prepared statement handling
- **echo**: HTTP routing and middleware
- **pq**: PostgreSQL array types
- **Vue 2.7**: Frontend framework
- **Buefy**: UI components

## Data Flow

### Campaign Resume Flow
```
User clicks "Resume" button
  ↓
Frontend: PUT /api/campaigns/:id with status="running"
  ↓
Handler: UpdateCampaignStatus(id, "running")
  ↓
Core: Fetch current campaign
  ↓
Core: Detect status transition (paused → running)
  ↓
Core: Check messenger type ("automatic")
  ↓
Core: Call RequeueCancelledEmails(id)
  ↓
SQL: UPDATE email_queue SET status='queued' WHERE status='cancelled' AND NOT IN (sent)
  ↓
Core: Log "re-queued N emails"
  ↓
Handler: Return updated campaign
  ↓
Queue Processor: Picks up re-queued emails on next cycle
  ↓
Emails sent respecting SMTP capacity and time windows
```

### Remove Sent Subscribers Flow
```
User clicks "Remove Sent Subscribers"
  ↓
Frontend: Confirmation dialog
  ↓
Frontend: POST /api/campaigns/:id/remove-sent-today
  ↓
Handler: Permission check
  ↓
Handler: Fetch campaign details
  ↓
Handler: Unmarshal campaign.Lists (JSONText → []struct)
  ↓
Handler: Query GetSentSubscribersToday(campaign_id)
  ↓
SQL: SELECT DISTINCT subscriber_id FROM email_queue WHERE status='sent'
  ↓
Handler: Call UnsubscribeLists(subscriber_ids, list_ids)
  ↓
Core: Bulk UPDATE subscriber_lists SET status='unsubscribed'
  ↓
Handler: Return count and message
  ↓
Frontend: Toast notification with count
```

## Critical Code Patterns

### Pattern 1: JSONText Unmarshaling
**Problem**: Campaign.Lists is stored as JSON bytes, not a Go struct

**Solution:**
```go
var lists []struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}
if err := campaign.Lists.Unmarshal(&lists); err != nil {
    return echo.NewHTTPError(http.StatusInternalServerError, ...)
}
```

**Why It Matters:**
- Direct iteration of JSONText fails with type errors
- Must unmarshal into typed struct before accessing fields
- Common pattern throughout listmonk codebase

### Pattern 2: Prepared Statement Execution
**Problem**: Confusion between db.Select() and stmt.Select()

**Wrong:**
```go
a.db.Select(&result, a.queries.SomeQuery, params...)
```

**Correct:**
```go
a.queries.SomeQuery.Select(&result, params...)
```

**Why It Matters:**
- Prepared statements are already compiled
- Calling statement's method directly is faster
- Type safety from query registration

### Pattern 3: Frontend API Methods
**Problem**: Listmonk doesn't expose generic http.post()

**Wrong:**
```javascript
this.$api.post('/api/campaigns/123/action', {})
```

**Correct:**
```javascript
// In api/index.js
export const doAction = async (id) => http.post(`/api/campaigns/${id}/action`, {})

// In component
this.$api.doAction(123)
```

**Why It Matters:**
- All API calls must be pre-registered exports
- Provides type safety and loading state management
- Consistent error handling across app

### Pattern 4: Status Transition Detection
**Problem**: How to distinguish new start from resume?

**Solution:**
```go
if newStatus == Running && messenger == "automatic" {
    if currentCampaign.Status == Paused {
        // This is a RESUME
        RequeueCancelledEmails(id)
    } else {
        // This is a NEW START
        go QueueCampaignEmails(id)
    }
}
```

**Why It Matters:**
- Same API endpoint handles both start and resume
- Must check current status before taking action
- Different operations for different transitions

## Environment Configuration

### Production Environment
- **URL**: https://list.bobbyseamoss.com
- **Container App**: listmonk420
- **Resource Group**: rg-listmonk420
- **Database**: listmonk420-db.postgres.database.azure.com
- **Timezone**: Server runs in UTC, user in EST (America/New_York)

### Database Connection
```bash
PGPASSWORD='T@intshr3dd3r' psql \
  -h listmonk420-db.postgres.database.azure.com \
  -p 5432 \
  -U listmonkadmin \
  -d listmonk
```

### Deployment Process
```bash
# Build frontend
cd frontend && yarn build

# Build and deploy
./deploy.sh
```

## Testing Considerations

### Campaign Resume Testing
**Test Scenario:**
1. Start queue-based campaign with large subscriber list
2. Monitor email_queue: verify status='queued'
3. Pause campaign
4. Verify email_queue: status changes to 'cancelled'
5. Wait for some emails to send (status='sent')
6. Resume campaign
7. Check logs for "re-queued N emails" message
8. Verify: cancelled emails → queued (excluding sent)
9. Confirm no duplicate sends to sent subscribers

**Database Queries for Verification:**
```sql
-- Count emails by status
SELECT status, COUNT(*)
FROM email_queue
WHERE campaign_id = ?
GROUP BY status;

-- Check for duplicates
SELECT subscriber_id, COUNT(*)
FROM email_queue
WHERE campaign_id = ? AND status IN ('sent', 'queued')
GROUP BY subscriber_id
HAVING COUNT(*) > 1;
```

### Remove Sent Testing
**Test Scenario:**
1. Run queue-based campaign to completion
2. Verify subscribers in campaign's lists
3. Click "Remove Sent Subscribers"
4. Confirm dialog
5. Verify toast shows correct count
6. Check subscriber_lists: status='unsubscribed'
7. Create new campaign with same lists
8. Verify removed subscribers not targeted

**Database Queries for Verification:**
```sql
-- Get sent count
SELECT COUNT(DISTINCT subscriber_id)
FROM email_queue
WHERE campaign_id = ? AND status = 'sent';

-- Check unsubscribe status
SELECT sl.status, COUNT(*)
FROM subscriber_lists sl
JOIN email_queue eq ON sl.subscriber_id = eq.subscriber_id
WHERE eq.campaign_id = ? AND eq.status = 'sent'
  AND sl.list_id IN (-- campaign's lists --)
GROUP BY sl.status;
```

## Known Issues and Limitations

### Issue 1: Query Name Mismatch
**Description**: Query named "get-sent-subscribers-today" but returns all-time data

**Impact**: Confusing for future maintainers

**Workaround**: Code comments explain behavior

**Future Fix**: Rename to "get-sent-subscribers" (requires updating all references)

### Issue 2: No Undo for Unsubscribe
**Description**: Remove Sent operation is permanent via UI

**Impact**: User must use database to reverse mistakes

**Workaround**: Confirmation dialog warns users

**Future Enhancement**: Add admin UI for reverting unsubscribes

### Issue 3: Large Campaign Performance
**Description**: No chunking for very large campaigns (100k+ subscribers)

**Impact**: Single bulk query might be slow

**Current Status**: No reports of performance issues

**Future Enhancement**: Add chunking if users report slowness

## Debugging Tips

### Check Campaign Status Transitions
```sql
-- See campaign history (if logging enabled)
SELECT * FROM campaigns WHERE id = ?;

-- Check current email queue state
SELECT status, COUNT(*)
FROM email_queue
WHERE campaign_id = ?
GROUP BY status;
```

### Verify Re-queue Logic
```bash
# Check application logs
az containerapp logs show \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --tail 100 | grep "re-queued"
```

### Test API Endpoints
```bash
# Get campaign details
curl https://list.bobbyseamoss.com/api/campaigns/64

# Remove sent subscribers (requires auth cookie)
curl -X POST https://list.bobbyseamoss.com/api/campaigns/64/remove-sent-today \
  -H "Cookie: session=..." \
  -H "Content-Type: application/json"
```

## Future Enhancement Ideas

1. **Resume Progress UI**: Show re-queue count in frontend toast
2. **Batch Processing**: Chunk large unsubscribe operations
3. **Audit Log**: Track who removed subscribers and when
4. **Scheduled Removal**: Automatically remove sent subscribers after campaign completes
5. **List Exclusion**: Alternative to unsubscribe - exclude from future campaigns without status change
6. **Analytics Integration**: Track removal metrics in campaign analytics

## Related Documentation

- **Queue System**: See CLAUDE.md section "Queue-Based Email Delivery System"
- **Campaign Lifecycle**: See `internal/core/campaigns.go` comments
- **API Patterns**: See `frontend/src/api/index.js` file header
- **Database Schema**: See `schema.sql` for table definitions
- **Deployment**: See `deploy.sh` for Azure Container Apps deployment
