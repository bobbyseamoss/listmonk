# Campaign Sent Count Fix - Tasks

**Last Updated**: 2025-11-11 06:52 AM EST

## Completed Tasks ✅

- [x] Identify where campaign.sent should be updated for queue-based campaigns
  - Found that email_queue updates happen in queue processor
  - campaigns.sent never synced from email_queue

- [x] Find the bug causing sent count to stay at 0
  - Root cause: No sync mechanism between email_queue and campaigns table
  - Queue processor only updates email_queue.status, never campaigns.sent

- [x] Implement fix to sync campaign.sent when pausing
  - Modified autoPauseRunningCampaigns() in internal/queue/processor.go
  - Added inline UPDATE query to sync counts before pause
  - Tested: Campaign 65 count preserved after manual sync

- [x] Add periodic sync of campaign.sent during sending
  - Created StartCampaignStatsSync() function
  - Runs every 5 minutes via ticker
  - Syncs all running queue-based campaigns

- [x] Test the fix on the paused campaign
  - Manual sync test: Campaign 65 updated from 0 to 20,323
  - Verified against email_queue: 20,323 sent
  - Verified against azure_message_tracking: 20,318 tracked

- [x] Build and deploy fix to Bobby Seamoss
  - Docker build completed successfully
  - Image pushed to ACR
  - Deployed revision: listmonk420--deploy-20251111-065049

- [x] Verify campaign stats sync is working in production
  - Logs confirm: "started queue processor, auto-pause scheduler, and stats sync"
  - Logs confirm: "starting campaign stats sync (every 5 minutes)"
  - Campaign 65 count preserved at 20,323 (26% progress)

## No Pending Tasks

All tasks for this issue have been completed and verified in production.

## Future Enhancement Ideas (Not Prioritized)

- [ ] Add API endpoint for manual sync: `/api/campaigns/:id/sync-counts`
- [ ] Sync counts when resuming paused campaigns (currently only syncs during running)
- [ ] Make sync interval configurable via settings (currently hardcoded to 5 minutes)
- [ ] Add sync lag/discrepancy metrics to admin dashboard
- [ ] Extend sync to non-queue campaigns for consistency
- [ ] Add Sentry tracking for sync errors
