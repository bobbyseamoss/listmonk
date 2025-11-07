# Queue Campaign Management - Task Checklist

**Last Updated: 2025-11-07 (Session 2 - Hard Delete Change)**

## ⚠️ PENDING DEPLOYMENT - UNCOMMITTED CHANGES

**Status**: Code changes complete for hard delete modification, awaiting deployment
**When**: User will deploy tonight after pausing current campaign
**Commands**:
```bash
cd frontend && yarn build && cd .. && ./deploy.sh
```

**Changes Made**:
- Backend: Changed from soft-delete (UnsubscribeLists) to hard-delete (DeleteSubscriptions)
- Frontend: Updated UI text from "Unsubscribe" to "Remove"
- Files: `/home/adam/listmonk/cmd/campaigns.go`, `/home/adam/listmonk/frontend/src/views/Campaign.vue`

---

## Phase 3.5: Hard Delete Modification ⏳ IN PROGRESS (Uncommitted)

**Added**: 2025-11-07 Evening Session
**Status**: Code complete, awaiting deployment after user pauses campaign

### User Requirement Change

**Original**: Remove Sent Subscribers should unsubscribe users (soft delete)
**New**: Remove Sent Subscribers should completely remove users from lists (hard delete)
**Reason**: User wants subscribers removed from list membership, not just marked as unsubscribed

### Implementation Changes

- [x] Update backend handler comment to clarify "hard delete"
- [x] Change `UnsubscribeLists` call to `DeleteSubscriptions`
- [x] Update success message: "Unsubscribed" → "Removed"
- [x] Update frontend description text
- [ ] **PENDING**: Build frontend (`cd frontend && yarn build`)
- [ ] **PENDING**: Deploy to Azure (`./deploy.sh`)
- [ ] **PENDING**: Test deletion verification query

### Code Changes Detail

**File**: `/home/adam/listmonk/cmd/campaigns.go`
```diff
- Line 844-845: // from all lists associated with the campaign.
+ Line 844-845: // from all lists associated with the campaign (hard delete from subscriber_lists table).

- Line 900: if err := a.core.UnsubscribeLists(subIDs, listIDs, nil); err != nil {
+ Line 900: if err := a.core.DeleteSubscriptions(subIDs, listIDs); err != nil {

- Line 909: "message": fmt.Sprintf("Unsubscribed %d subscribers from %d lists", len(subIDs), len(listIDs)),
+ Line 909: "message": fmt.Sprintf("Removed %d subscribers from %d lists", len(subIDs), len(listIDs)),
```

**File**: `/home/adam/listmonk/frontend/src/views/Campaign.vue`
```diff
- Line 186: Unsubscribe all subscribers who received this campaign from the campaign's lists.
+ Line 186: Remove all subscribers who received this campaign from the campaign's lists.
```

### Deployment Timing

**Why Waiting**: User has active queue-based campaign currently running
**Plan**: User will pause campaign tonight, then deploy changes
**Impact**: No immediate deployment urgency, can wait for optimal time

### Post-Deployment Verification

After deployment, test with these steps:

1. Run a queue-based campaign to completion
2. Note the subscriber IDs who received emails (from email_queue)
3. Click "Remove Sent Subscribers" button
4. Run verification query:
   ```sql
   -- Check that rows are DELETED (not just status updated)
   SELECT COUNT(*) FROM subscriber_lists
   WHERE subscriber_id IN (
       SELECT DISTINCT subscriber_id
       FROM email_queue
       WHERE campaign_id = ? AND status = 'sent'
   )
   AND list_id IN (campaign's list IDs);

   -- Should return 0 (rows deleted)
   -- Old behavior would return N with status='unsubscribed'
   ```

### Rollback Plan

If issues arise, revert changes:
```diff
+ Line 900: if err := a.core.UnsubscribeLists(subIDs, listIDs, nil); err != nil {
- Line 900: if err := a.core.DeleteSubscriptions(subIDs, listIDs); err != nil {
```

And redeploy. Database changes are permanent but can be re-added via bulk import if needed.

---

## Phase 1: UI Icon Updates ✅ COMPLETED

- [x] Add Fontello play icon (󰐊) to icon font CSS
- [x] Add Fontello pause icon (󰏤) to icon font CSS
- [x] Update Campaign.vue to use play icon for start button
- [x] Update Campaign.vue to use pause icon for running campaigns
- [x] Verify all other icons still render correctly
- [x] Test icon display in browser
- [x] Deploy frontend changes

**Files Modified:**
- `/home/adam/listmonk/frontend/src/assets/icons/fontello.css:121-122`
- `/home/adam/listmonk/frontend/src/views/Campaign.vue`

---

## Phase 2: Campaign Resume Fix ✅ COMPLETED

### Backend Implementation

- [x] Create SQL query: `requeue-cancelled-emails`
  - [x] Update status from 'cancelled' to 'queued'
  - [x] Exclude subscribers with status='sent' (subquery)
  - [x] Update timestamp to NOW()
  - [x] Add to `/home/adam/listmonk/queries.sql:1617-1624`

- [x] Register query in models
  - [x] Add `RequeueCancelledEmails *sqlx.Stmt` field
  - [x] Add query tag: `query:"requeue-cancelled-emails"`
  - [x] File: `/home/adam/listmonk/models/queries.go:109`

- [x] Implement core method
  - [x] Create `RequeueCancelledEmails(campID int)` method
  - [x] Execute prepared statement
  - [x] Return count of re-queued emails
  - [x] Error handling with HTTP status codes
  - [x] Logging with context
  - [x] File: `/home/adam/listmonk/internal/core/campaigns.go:604-616`

- [x] Modify status update logic
  - [x] Detect pause→running transition in `UpdateCampaignStatus`
  - [x] Check messenger type for "automatic"
  - [x] Distinguish resume from new start
  - [x] Call `RequeueCancelledEmails` on resume
  - [x] Maintain async queue for new starts
  - [x] Log re-queue counts
  - [x] File: `/home/adam/listmonk/internal/core/campaigns.go:310-337`

### Testing

- [x] Code deployed to production
- [ ] **PENDING**: User test of paused campaign resume
  - [ ] Pause running queue-based campaign
  - [ ] Verify emails marked as 'cancelled'
  - [ ] Resume campaign
  - [ ] Check logs for "re-queued N emails" message
  - [ ] Verify no duplicate sends

---

## Phase 3: Remove Sent Subscribers Feature ✅ COMPLETED

### Backend Implementation

- [x] Create SQL query: `get-sent-subscribers-today`
  - [x] Select distinct subscriber IDs from email_queue
  - [x] Filter by campaign_id and status='sent'
  - [x] ~~Initial: Add timezone-based date filtering~~ (removed per user request)
  - [x] Final: All-time query (no date filter)
  - [x] File: `/home/adam/listmonk/queries.sql:1605-1610`

- [x] Register query in models
  - [x] Add `GetSentSubscribersToday *sqlx.Stmt` field
  - [x] Add query tag: `query:"get-sent-subscribers-today"`
  - [x] File: `/home/adam/listmonk/models/queries.go:108`

- [x] Implement HTTP handler
  - [x] Create `RemoveSentSubscribersFromLists` handler
  - [x] Permission check: campaigns:manage_all or campaigns:manage
  - [x] Fetch campaign details via `GetCampaign`
  - [x] Unmarshal JSONText campaign.Lists field
  - [x] Extract list IDs from unmarshaled data
  - [x] Query sent subscribers via `GetSentSubscribersToday`
  - [x] Handle edge case: no lists associated
  - [x] Handle edge case: no sent subscribers
  - [x] Call `UnsubscribeLists` with subscriber and list IDs
  - [x] Return count and message in response
  - [x] Error handling with context
  - [x] Logging with campaign ID and counts
  - [x] File: `/home/adam/listmonk/cmd/campaigns.go:844-911`

- [x] Register API route
  - [x] Add POST route: `/api/campaigns/:id/remove-sent-today`
  - [x] Apply permission middleware
  - [x] Map to `RemoveSentSubscribersFromLists` handler
  - [x] File: `/home/adam/listmonk/cmd/handlers.go:179`

### Frontend Implementation

- [x] Create API method
  - [x] Export `removeSentSubscribersFromLists` function
  - [x] Use http.post wrapper with proper signature
  - [x] Add loading state model
  - [x] File: `/home/adam/listmonk/frontend/src/api/index.js:408-412`

- [x] Build UI component
  - [x] Add UI box to Campaign.vue
  - [x] Conditional visibility: only queue-based campaigns (`messenger === 'automatic'`)
  - [x] Add title: "Remove Sent Subscribers"
  - [x] Add description explaining purpose
  - [x] ~~Initial: "Remove Sent Today" button~~ (changed per user request)
  - [x] Final: "Remove Sent Subscribers" button
  - [x] Add warning button style (is-warning)
  - [x] Add account-remove icon
  - [x] File: `/home/adam/listmonk/frontend/src/views/Campaign.vue:181-194`

- [x] Implement button handler
  - [x] Create `removeSentFromLists` method
  - [x] Add confirmation dialog with warning
  - [x] Call API method: `this.$api.removeSentSubscribersFromLists(id)`
  - [x] Show loading state during API call
  - [x] Display success toast with message
  - [x] Handle errors gracefully
  - [x] File: `/home/adam/listmonk/frontend/src/views/Campaign.vue:681-697`

### Deployment and Testing

- [x] Build frontend: `cd frontend && yarn build`
- [x] Deploy to Azure: `./deploy.sh`
- [x] Verify deployment health
- [x] Test on Campaign 64 (queue-based)
- [x] Verify correct subscriber count identified
- [x] Verify subscribers unsubscribed from lists
- [x] User confirmation: "Unsubscribe operation is working now" ✅

---

## Debugging and Fixes Applied

### Fix 1: SQL Query Registration ✅
- [x] **Issue**: Compilation error - queries not in models.Queries struct
- [x] **Fix**: Added two fields to `/home/adam/listmonk/models/queries.go`
- [x] **Verified**: Backend compiles without errors

### Fix 2: Frontend API Pattern ✅
- [x] **Issue**: `this.$api.post is not a function` error
- [x] **Root cause**: Listmonk uses named API exports, not generic post()
- [x] **Fix**: Added proper export to `api/index.js`
- [x] **Fix**: Updated component to call named method
- [x] **Verified**: API call executes successfully

### Fix 3: Timezone Issues ✅
- [x] **Issue**: User in EST, server in UTC, date comparison failed
- [x] **Initial fix**: Add timezone conversion to query
- [x] **User feedback**: "Should be Remove Sent regardless of when they were emailed"
- [x] **Final fix**: Remove all date filtering
- [x] **Verified**: Query finds all sent subscribers

### Fix 4: JSONText Unmarshaling ✅
- [x] **Issue**: Cannot iterate campaign.Lists directly
- [x] **Root cause**: JSONText type stores JSON as bytes
- [x] **Fix**: Unmarshal into typed struct before accessing
- [x] **Verified**: List IDs extracted correctly

### Fix 5: Prepared Statement Syntax ✅
- [x] **Issue**: Cannot use statement as string in db.Select()
- [x] **Root cause**: Must call statement's .Select() method directly
- [x] **Fix**: Changed to `a.queries.GetSentSubscribersToday.Select()`
- [x] **Verified**: Query executes successfully

---

## Production Deployment Checklist

### Pre-Deployment ✅
- [x] All backend code compiles
- [x] All frontend code builds without errors
- [x] No TypeScript/ESLint errors
- [x] SQL queries tested in local environment
- [x] Changes committed to git

### Deployment Steps ✅
- [x] Build frontend: `cd frontend && yarn build`
- [x] Run deployment script: `./deploy.sh`
- [x] Docker build completes successfully
- [x] Image pushed to Azure Container Registry
- [x] Container app updated with new revision
- [x] Health checks pass
- [x] Application accessible at https://list.bobbyseamoss.com

### Post-Deployment Verification ✅
- [x] UI icons display correctly
- [x] Remove Sent button visible on queue-based campaigns
- [x] Remove Sent button hidden on non-queue campaigns
- [x] Confirmation dialog appears on button click
- [x] API call succeeds
- [x] Subscribers unsubscribed from lists
- [x] Toast notification shows correct count
- [x] No errors in browser console
- [x] No errors in application logs

### Latest Deployment
- **Revision**: listmonk420--deploy-20251107-052900
- **Status**: Active and healthy ✅
- **URL**: https://list.bobbyseamoss.com
- **Verified**: Remove Sent Subscribers feature working

---

## Pending User Testing

### Campaign Resume Feature ⏳
- [ ] User has paused campaign ready to test
- [ ] Resume paused campaign
- [ ] Verify emails re-queued (check logs)
- [ ] Verify no duplicate sends
- [ ] Verify queue processor handles re-queued emails
- [ ] Confirm expected delivery timeline

**Expected Log Message:**
```
re-queued {count} cancelled emails for campaign {id} ({name})
```

**Database Verification:**
```sql
-- Before resume
SELECT status, COUNT(*) FROM email_queue WHERE campaign_id = ? GROUP BY status;
-- Should show: cancelled = X, sent = Y

-- After resume
SELECT status, COUNT(*) FROM email_queue WHERE campaign_id = ? GROUP BY status;
-- Should show: queued = X, sent = Y (cancelled = 0)
```

---

## Future Enhancements (Not in Scope)

### Nice-to-Have Features
- [ ] Resume progress indicator in UI (show re-queue count in toast)
- [ ] Undo button for unsubscribe operations
- [ ] Batch processing for very large campaigns (>100k subscribers)
- [ ] Campaign analytics integration (track removed subscribers)
- [ ] Scheduled auto-removal (remove sent subscribers after campaign completes)
- [ ] Query name consistency fix (rename "get-sent-subscribers-today" to "get-sent-subscribers")

### Performance Optimizations
- [ ] Chunked unsubscribe for campaigns >100k subscribers
- [ ] Progress bar for long-running operations
- [ ] Background job for removal (if needed)

### UX Improvements
- [ ] Show count preview before confirming removal
- [ ] Filter removed subscribers by date range (configurable)
- [ ] Bulk removal across multiple campaigns

---

## Success Criteria (Final Checklist)

### Campaign Resume ✅/⏳
- [x] Code deployed to production
- [x] Resume logic detects pause→running transition
- [x] Re-queue excludes already-sent subscribers
- [x] Log messages provide clear feedback
- [ ] **PENDING**: User confirms no duplicate sends after resume
- [ ] **PENDING**: User confirms campaign completes successfully

### Remove Sent Subscribers ✅ (Updated to Hard Delete)
- [x] Feature accessible via UI
- [x] Only visible for queue-based campaigns
- [x] Confirmation dialog prevents accidents
- [x] API call succeeds
- [x] ~~Subscribers unsubscribed from correct lists~~ **CHANGED**: Subscribers now hard-deleted from lists
- [x] Toast shows accurate count
- [x] Works across all time periods
- [x] User confirms: "Unsubscribe operation is working now" (soft delete version)
- [ ] **PENDING DEPLOYMENT**: Hard delete version (DeleteSubscriptions instead of UnsubscribeLists)
- [ ] **PENDING TEST**: Verify rows deleted from subscriber_lists table (not just status changed)

### Overall Quality ✅
- [x] No regression in existing features
- [x] No database performance issues
- [x] No frontend errors
- [x] Deployment successful
- [x] Zero downtime deployment
- [x] All icons functional
- [x] Logging provides debugging information

---

## Sign-off

**Implementation Status**: Complete ✅
**Deployment Status**: Production ✅
**Testing Status**: Partial (Remove Sent verified ✅, Resume pending user test ⏳)

**Next Action**: User to test paused campaign resume functionality when ready

**Production URL**: https://list.bobbyseamoss.com
**Latest Revision**: listmonk420--deploy-20251107-052900
**Deployment Date**: 2025-11-07
