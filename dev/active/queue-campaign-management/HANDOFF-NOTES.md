# HANDOFF NOTES - Queue Campaign Management
**Last Updated**: 2025-11-07 Evening Session
**Status**: AWAITING DEPLOYMENT

## ⚠️ IMMEDIATE ATTENTION REQUIRED

### Uncommitted Changes Ready for Deployment

**What Changed**: Remove Sent Subscribers feature modified from soft-delete to hard-delete

**Files Modified** (not yet deployed):
1. `/home/adam/listmonk/cmd/campaigns.go` - Lines 844-845, 900, 909
2. `/home/adam/listmonk/frontend/src/views/Campaign.vue` - Line 186

**Git Status**: Modified files not committed or deployed

### Deployment Plan

**When**: Tonight after user pauses current campaign
**Why Waiting**: User has active queue-based campaign running, doesn't want to interrupt it

**Deployment Commands**:
```bash
cd /home/adam/listmonk
cd frontend && yarn build
cd ..
./deploy.sh
```

**Expected Result**:
- Frontend rebuild: ~1-2 minutes
- Docker build and push: ~3-5 minutes
- Azure deployment: ~2-3 minutes
- Total: ~8 minutes

### What the Change Does

**OLD (Currently in Production - Revision listmonk420--deploy-20251107-052900)**:
- Calls `a.core.UnsubscribeLists(subIDs, listIDs, nil)`
- Executes SQL: `UPDATE subscriber_lists SET status='unsubscribed'`
- Result: Rows remain in table, status changed

**NEW (Ready to Deploy)**:
- Calls `a.core.DeleteSubscriptions(subIDs, listIDs)`
- Executes SQL: `DELETE FROM subscriber_lists WHERE (subscriber_id, list_id) = ANY(...)`
- Result: Rows completely removed from table

### Testing After Deployment

Run this SQL query to verify hard delete:
```sql
-- After clicking "Remove Sent Subscribers" on a campaign
SELECT COUNT(*) FROM subscriber_lists
WHERE subscriber_id IN (
    SELECT DISTINCT subscriber_id
    FROM email_queue
    WHERE campaign_id = [CAMPAIGN_ID] AND status = 'sent'
)
AND list_id IN ([CAMPAIGN_LIST_IDS]);

-- Should return 0 (rows deleted)
-- Old behavior returned N rows with status='unsubscribed'
```

### Rollback Instructions

If issues occur after deployment:

**Quick Rollback**:
```bash
# Revert the change in cmd/campaigns.go line 900
# Change back to: a.core.UnsubscribeLists(subIDs, listIDs, nil)
cd frontend && yarn build && cd .. && ./deploy.sh
```

**Alternative**: Roll back to previous revision:
```bash
az containerapp revision list \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --query "[].name" -o tsv

# Activate previous revision (listmonk420--deploy-20251107-052900)
az containerapp revision activate \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --revision [PREVIOUS_REVISION_NAME]
```

### Additional Context

**User Quote**: "When I click the button, users who were sent emails in that campaign are unsubscribed from the list. I'd like them to be simply removed from the list but not unsubscribed if that is possible."

**Interpretation**: User wants hard delete (row removal) not soft delete (status change)

**Impact**:
- Subscribers can be re-added to lists cleanly (no unsubscribed status to deal with)
- More destructive action (rows deleted permanently)
- Confirmation dialog already warns "cannot be undone"

### Related Documentation

- Full plan: `/home/adam/listmonk/dev/active/queue-campaign-management/queue-campaign-management-plan.md`
- Context: `/home/adam/listmonk/dev/active/queue-campaign-management/queue-campaign-management-context.md`
- Tasks: `/home/adam/listmonk/dev/active/queue-campaign-management/queue-campaign-management-tasks.md`

### Other Pending Work

**Campaign Resume Feature**: Code deployed but not yet tested by user
- User has paused campaign ready to test
- Expected to test resume functionality tonight
- Should see log message: "re-queued N cancelled emails for campaign X"

### Environment Details

**Production URL**: https://list.bobbyseamoss.com
**Current Revision**: listmonk420--deploy-20251107-052900 (soft delete version)
**Resource Group**: rg-listmonk420
**Database**: listmonk420-db.postgres.database.azure.com
**User Timezone**: EST (America/New_York)
**Server Timezone**: UTC

### No Blockers

All dependencies satisfied:
- ✅ Go backend compiles successfully
- ✅ Frontend builds without errors
- ✅ Azure infrastructure healthy
- ✅ Database accessible
- ✅ No migration required (uses existing delete-subscriptions query)
- ✅ No schema changes needed

### Next Steps for New Session

1. Wait for user to indicate campaign is paused
2. Run deployment commands
3. Test the hard delete behavior
4. Update documentation with deployment results
5. Mark Phase 3.5 as complete
6. Test campaign resume feature if user is ready

---

**Session Summary**: Simple 3-line code change with 1 frontend text update. Waiting on user timing for deployment. Zero technical blockers.
