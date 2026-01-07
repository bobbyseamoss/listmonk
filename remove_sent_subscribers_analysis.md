# Remove Sent Subscribers Analysis Report
**Date**: 2025-11-13
**Campaign Analyzed**: Campaign 64 - "Copy of Copy of Gummies New"

## Executive Summary

**Good news!** Only **1 subscriber** was affected by the "Remove Sent Subscribers" functionality on campaign 64. This subscriber's list membership was deleted rather than being properly unsubscribed.

## Campaign Details

- **Campaign ID**: 64
- **Campaign UUID**: 78128b7e-9031-49ba-8c13-c652cd1a9918
- **Campaign Name**: Copy of Copy of Gummies New
- **Subject**: Quick hello + a small gift for you
- **Messenger**: automatic (queue-based)
- **Status**: cancelled
- **Target List**: List 7 - "Kali - ALL Gummies"
- **Total Recipients**: 29,126 unique subscribers

## Breakdown of All Recipients

| Category | Count | Percentage |
|----------|-------|------------|
| **Still subscribed to list 7** | 20,800 | 71.4% |
| **Unsubscribed from list 7 (normal)** | 8,324 | 28.6% |
| **No list relationship (deleted by button)** | **1** | **0.003%** |
| **Subscriber blocklisted** | 105 | 0.4% |
| **Subscriber disabled** | 0 | 0% |
| **TOTAL** | 29,126 | 100% |

## Affected Subscriber Details

### Subscriber Information
- **Subscriber ID**: 452
- **UUID**: bc5657ac-84ac-4b83-933f-0fb499c9c2d2
- **Email**: info@bobbyseamoss.com
- **Name**: Info
- **Status**: enabled (still active)
- **Created**: 2025-10-28 18:18:05 UTC
- **Last Updated**: 2025-11-01 11:34:34 UTC

### Campaign Engagement
- **Email Opens**: 2
- **Email Clicks**: 0

### Current List Subscriptions
- **List 2** - "Testers" (unconfirmed status)
- **List 7** - "Kali - ALL Gummies" (REMOVED - this is the affected subscription)

## Analysis

### Key Findings

1. **Minimal Impact**: Only 1 subscriber out of 29,126 (0.003%) was affected by the "Remove Sent Subscribers" button.

2. **Test Account**: The affected subscriber (info@bobbyseamoss.com) appears to be a test/internal account, not a customer.

3. **Normal Unsubscribes**: The 8,324 subscribers who are marked as "unsubscribed" likely did so through normal means (clicking unsubscribe links in emails), NOT through the "Remove Sent Subscribers" button.

4. **Feature Mechanism**: The research confirmed that "Remove Sent Subscribers" performs a hard DELETE on the `subscriber_lists` table rather than setting status to 'unsubscribed'. This leaves no audit trail.

### Why Only 1 Affected?

The low number suggests that:
- The button was likely clicked only once
- Most recipients remained subscribed after receiving the campaign
- The feature's impact was limited to this single test account

## SQL Query for Future Analysis

```sql
-- Find subscribers who received a campaign but are no longer subscribed to its lists
WITH sent_subscribers AS (
  SELECT DISTINCT subscriber_id
  FROM campaign_views
  WHERE campaign_id = :campaign_id
),
campaign_target_lists AS (
  SELECT list_id
  FROM campaign_lists
  WHERE campaign_id = :campaign_id
)
SELECT
  s.id as subscriber_id,
  s.uuid,
  s.email,
  s.name,
  s.status,
  s.created_at,
  s.updated_at,
  ctl.list_id as missing_list_id,
  (SELECT COUNT(*) FROM campaign_views WHERE subscriber_id = s.id AND campaign_id = :campaign_id) as views,
  (SELECT COUNT(*) FROM link_clicks WHERE subscriber_id = s.id AND campaign_id = :campaign_id) as clicks
FROM sent_subscribers ss
INNER JOIN subscribers s ON ss.subscriber_id = s.id
CROSS JOIN campaign_target_lists ctl
LEFT JOIN subscriber_lists sl ON ss.subscriber_id = sl.subscriber_id AND sl.list_id = ctl.list_id
WHERE sl.subscriber_id IS NULL  -- Not subscribed to the list
  AND s.status != 'blocklisted'  -- Still active
ORDER BY s.id;
```

## Recommendations

### Immediate Actions
1. **Re-subscribe the test account** (info@bobbyseamoss.com) to list 7 if desired:
   ```sql
   INSERT INTO subscriber_lists (subscriber_id, list_id, status, created_at, updated_at)
   VALUES (452, 7, 'confirmed', NOW(), NOW())
   ON CONFLICT (subscriber_id, list_id) DO NOTHING;
   ```

2. **No customer impact**: Since only a test account was affected, no customer communications or remediation needed.

### Future Prevention
1. **Modify the feature** to use proper unsubscribe (status='unsubscribed') instead of hard DELETE
2. **Add audit logging** to track when and by whom this feature is used
3. **Add confirmation dialog** with clear explanation of the impact
4. **Consider renaming** the feature to accurately reflect what it does

### Feature Behavior Documentation

**Current Behavior** (as of this analysis):
- Location: Campaign page, visible for campaigns with `messenger='automatic'`
- Action: Hard DELETEs rows from `subscriber_lists` table
- SQL: `DELETE FROM subscriber_lists WHERE (subscriber_id, list_id) = ANY(...)`
- No audit trail or timestamp recorded
- No way to distinguish from manual admin deletion

**Recommended Behavior**:
- Use `UPDATE subscriber_lists SET status='unsubscribed', updated_at=NOW()` instead
- Add `unsubscribed_by` field to track source (manual, campaign-removal, etc.)
- Log action to admin audit log
- Preserve the relationship for historical reporting

## Conclusion

The impact of the "Remove Sent Subscribers" feature on campaign 64 was **minimal**, affecting only 1 test account. No customer accounts were impacted. The feature itself is aggressive (hard deletes instead of unsubscribes) but in this case caused no significant harm.

**No immediate remediation needed for customers.**
