# Bobby Seamoss Webhook Investigation

## Investigation Date
November 11, 2025

## Summary

Investigated failed email sends in Bobby Seamoss Azure deployment (listmonk420). Found two distinct but related issues:

1. **Domain Reputation Problem** - Gmail blocking emails due to low sender reputation
2. **Webhook Correlation Bug** - Missing azure_message_id column prevents proper event tracking

---

## Issue 1: Gmail Domain Reputation Blocking

### Impact
- 14,163 failed deliveries out of 109,932 total (12.9% failure rate)
- ~13,000+ failures are Gmail reputation blocks (550-5.7.1)
- Affects ALL 29 sending domains (mail2-mail30.bobbyseamoss.com) equally

### Timeline
Shows progressive reputation degradation as volume increased:

| Date | Failed | Delivered | Failure Rate |
|------|--------|-----------|--------------|
| Nov 5 | 8 | 2,945 | 0.27% ✓ |
| Nov 6 | 295 | 15,651 | 1.84% |
| Nov 7 | 888 | 20,266 | 4.15% |
| Nov 8 | 1,445 | 12,758 | 10.10% ⚠️ |
| Nov 9 | 5,337 | 11,883 | **30.74%** ❌ |
| Nov 10 | 3,835 | 15,444 | 19.71% |
| Nov 11 | 2,369 | 16,047 | 12.74% |

### Root Cause
- Cold domains with no established reputation
- High volume sending without warming period
- Gmail interprets as suspicious behavior

### Recommendations
See `/dev/active/bobby_seamoss_failure_analysis.md` for detailed remediation strategy including:
- Domain warming plan (4-6 weeks)
- Daily sending limits (100-200/day initially)
- Time window restrictions
- List hygiene improvements

---

## Issue 2: Webhook Message-ID Correlation Bug

### Discovery
While analyzing webhook logs, found repeated "Message-ID lookup failed" errors:

```
F 2025/11/11 04:18:25 PM bounce.go:312: Message-ID lookup failed for 
f7623789-2dc6-474a-8e43-66d6207efd50, trying recipient email abe.hamza@live.com
```

### Technical Analysis

**What's happening:**
1. Azure Communication Services sends delivery events to webhook endpoint
2. Events include `messageId` (Azure's UUID) and `internetMessageId` (email header)
3. Webhook handler tries to look up email by Azure `messageId`
4. Lookup fails because `email_queue` table doesn't store `azure_message_id`
5. System falls back to matching by recipient email address

**Database Schema Investigation:**

`email_queue` table columns:
```
id                        | bigint
campaign_id               | integer
subscriber_id             | integer
status                    | character varying
priority                  | integer
scheduled_at              | timestamp with time zone
sent_at                   | timestamp with time zone
assigned_smtp_server_uuid | character varying  ← Has SMTP UUID
retry_count               | integer
last_error                | text
created_at                | timestamp with time zone
updated_at                | timestamp with time zone
```

**Missing column:** `azure_message_id`

`azure_delivery_events` table:
```
id                      | bigint
azure_message_id        | uuid              ← Azure's message ID
campaign_id             | integer
subscriber_id           | integer
status                  | character varying
status_reason           | text
delivery_status_details | text
event_timestamp         | timestamp with time zone
created_at              | timestamp with time zone
```

### Impact

**Current behavior:**
- Webhook receives delivery events
- Cannot directly correlate events to email_queue entries by Azure message ID
- Falls back to matching by (campaign_id, subscriber_id, recipient email)
- This works but is less reliable and slower

**Potential issues:**
- If same subscriber receives multiple emails in same campaign, correlation may be ambiguous
- Cannot track individual email lifecycle from queue → send → delivery event
- More complex queries needed to join data

### Sample Webhook Event

```json
{
  "deliveryAttemptTimestamp": "2025-11-11T21:17:05.36+00:00",
  "deliveryStatusDetails": {
    "recipientMailServerHostName": "mx01.mail.icloud.com",
    "statusMessage": "[HM08] Message rejected due to local policy..."
  },
  "internetMessageId": "<e73f3798-4fcd-43ea-9bbd-1766fb2ded40@listmonk>",
  "messageId": "f1edbcfb-9d85-4275-ae62-d568141dbc89",  ← Azure message ID
  "recipient": "frays.arctic0@icloud.com",
  "sender": "adam@mail9.bobbyseamoss.com",
  "status": "Failed"
}
```

The `messageId` field needs to be stored in `email_queue` when email is sent.

### Proposed Fix

**Migration needed:**
```sql
-- Add azure_message_id column to email_queue
ALTER TABLE email_queue
ADD COLUMN IF NOT EXISTS azure_message_id UUID;

-- Add index for webhook lookups
CREATE INDEX IF NOT EXISTS idx_email_queue_azure_message_id
ON email_queue(azure_message_id);
```

**Code changes required:**
1. **Queue processor** - Store Azure message ID when sending via Azure Communication Services
2. **Webhook handler** - Use azure_message_id for primary lookup
3. **Keep fallback** - Maintain current email-based matching as backup

### Verification Query

Check how many webhook events can be correlated:

```sql
-- Events that can be matched by campaign+subscriber
SELECT COUNT(*) as matched_events
FROM azure_delivery_events ade
INNER JOIN email_queue eq
  ON ade.campaign_id = eq.campaign_id
  AND ade.subscriber_id = eq.subscriber_id;

-- Events that would match by azure_message_id (after fix)
-- Currently: 0 because column doesn't exist
SELECT COUNT(*) as direct_matches
FROM azure_delivery_events ade
LEFT JOIN email_queue eq
  ON ade.azure_message_id = eq.azure_message_id
WHERE eq.id IS NOT NULL;
```

---

## Related Files

**Analysis Reports:**
- `/dev/active/bobby_seamoss_failure_analysis.md` - Full delivery failure analysis

**Database Connection:**
```bash
PGPASSWORD='T@intshr3dd3r' psql \
  "sslmode=require host=listmonk420-db.postgres.database.azure.com user=listmonkadmin dbname=listmonk"
```

**Webhook Logs:**
```bash
az containerapp logs show \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --tail 500 \
  --follow
```

---

## Action Items

### Immediate (Domain Reputation)
- [ ] Pause or throttle campaigns targeting Gmail
- [ ] Implement daily sending limits (100-200/day per domain)
- [ ] Configure time window restrictions (8 AM - 8 PM)
- [ ] Begin domain warming strategy

### Short-term (Webhook Fix)
- [ ] Create migration to add `azure_message_id` column to `email_queue`
- [ ] Update queue processor to store Azure message ID when sending
- [ ] Update webhook handler to use direct message ID lookup
- [ ] Test on dev environment
- [ ] Deploy to production

### Long-term (Monitoring)
- [ ] Set up Gmail Postmaster Tools monitoring
- [ ] Create dashboard for daily failure rate tracking
- [ ] Implement automated alerts for reputation degradation
- [ ] Regular list hygiene (remove bounces, inactive subscribers)

---

## Credentials Added to Documentation

Updated `.claude/skills/migration-deployment-guidelines/SKILL.md` with:
- Database credentials (user, password, SSL mode)
- Connection strings
- Known issues section documenting both problems
- Azure resource names and configuration

---

## SQL Queries for Monitoring

**Daily delivery statistics:**
```sql
SELECT 
    DATE(event_timestamp) as date,
    COUNT(*) FILTER (WHERE status = 'Failed') as failed,
    COUNT(*) FILTER (WHERE status = 'Delivered') as delivered,
    COUNT(*) FILTER (WHERE status = 'Bounced') as bounced,
    ROUND(100.0 * COUNT(*) FILTER (WHERE status = 'Failed') / NULLIF(COUNT(*), 0), 2) as failure_rate
FROM azure_delivery_events
WHERE event_timestamp >= CURRENT_DATE - INTERVAL '7 days'
GROUP BY DATE(event_timestamp)
ORDER BY date;
```

**Failures by SMTP server:**
```sql
SELECT 
    eq.assigned_smtp_server_uuid,
    COUNT(*) FILTER (WHERE ade.status = 'Failed') as failed_count,
    COUNT(*) FILTER (WHERE ade.status = 'Delivered') as delivered_count,
    ROUND(100.0 * COUNT(*) FILTER (WHERE ade.status = 'Failed') / NULLIF(COUNT(*), 0), 2) as failure_rate
FROM email_queue eq
INNER JOIN azure_delivery_events ade 
  ON eq.campaign_id = ade.campaign_id 
  AND eq.subscriber_id = ade.subscriber_id
WHERE eq.assigned_smtp_server_uuid IS NOT NULL
GROUP BY eq.assigned_smtp_server_uuid
ORDER BY failed_count DESC;
```

**Top failure reasons:**
```sql
SELECT 
    status_reason, 
    COUNT(*) as count
FROM azure_delivery_events
WHERE status = 'Failed'
GROUP BY status_reason
ORDER BY count DESC
LIMIT 20;
```
