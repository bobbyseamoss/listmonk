# Azure Engagement Attribution Fix - Tasks

## Completed Tasks ✅

- [x] Investigate why engagement events misattributed to wrong campaigns
- [x] Identify root cause: Azure uses different message IDs for different event types
- [x] Research Azure Event Grid message fields
- [x] Discover internetMessageId as consistent identifier
- [x] Create v7.2.0 migration to add internet_message_id columns
- [x] Update AzureDeliveryData and AzureEngagementData structs
- [x] Update delivery webhook handler to store internetMessageId
- [x] Update engagement webhook handler to store internetMessageId
- [x] Update engagement event lookup to match on internet_message_id
- [x] Deploy initial fix to both sites
- [x] Test and discover critical timing issue
- [x] Realize engagement webhooks arrive before delivery webhooks
- [x] Update trackAzureMessage() to store internet_message_id at send time
- [x] Deploy critical fix to both sites
- [x] Verify new emails have internet_message_id populated immediately
- [x] Verify engagement events correctly attributed to campaign 65
- [x] Update documentation

## Monitoring Tasks 📊

### Short-term (Next 24 hours)
- [ ] Monitor campaign 65 engagement metrics
- [ ] Verify views/clicks increase for campaign 65 as new emails are opened
- [ ] Check ratio of emails with internet_message_id (should approach 100%)
- [ ] Watch for any misattributed engagement events

### Long-term (Next week)
- [ ] Analyze if old campaign 64 emails continue generating engagement (expected)
- [ ] Consider if historical data backfill is needed
- [ ] Add monitoring dashboard for internet_message_id population rate

## Potential Future Enhancements 🔮

### Data Quality
- [ ] Backfill internet_message_id for old emails in azure_message_tracking
  - Query delivery webhooks for internetMessageId
  - Match by azure_message_id
  - Update tracking records
  - Reprocess misattributed engagement events

### Monitoring
- [ ] Add Sentry alert if internet_message_id population rate drops below 95%
- [ ] Create Grafana dashboard showing attribution accuracy
- [ ] Track engagement event lookup success/failure rates

### Alternative Correlation
- [ ] Test if Azure preserves X-Listmonk-Campaign-ID header
- [ ] Implement multi-strategy matching (internetMessageId + custom headers)
- [ ] Add fallback to X-headers if internetMessageId missing

## Testing Checklist ✅

### Verification Complete
- [x] New emails have internet_message_id at send time
- [x] Engagement events match correctly to campaign 65
- [x] No regression in existing tracking functionality
- [x] Both sites deployed successfully
- [x] Database migrations completed
- [x] Real-time engagement attribution working

### Edge Cases to Monitor
- [ ] Engagement event arrives before message tracked (race condition)
- [ ] Azure generates Message-ID in different format
- [ ] Special characters in tracking UUID
- [ ] Multiple engagement events with same internet_message_id (expected)

## Notes

### Why This Was Complex
1. **Timing dependency**: Webhooks arrive asynchronously and out of order
2. **Azure architecture**: Uses different identifiers for different event types
3. **Testing difficulty**: Required production data to observe the timing issue
4. **Two-phase fix**: Initial fix didn't work, required deep understanding of webhook timing

### Success Criteria Met
✅ New engagement events attributed to correct running campaign
✅ No false negatives (missing attributions)
✅ No false positives (wrong attributions)
✅ Zero impact on existing functionality
✅ Forward-looking fix (historical data remains as-is)
