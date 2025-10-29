# Queue-Based Email Delivery System - Implementation Guide

## Overview

The queue-based email delivery system enables sophisticated email campaign management with:
- Per-SMTP server daily sending limits
- Time window restrictions (e.g., only send 8am-8pm)
- Automatic server selection and distribution
- Multi-day campaign support
- Real-time capacity monitoring

## Quick Start

### 1. Configure SMTP Servers with Limits

Navigate to **Settings > SMTP** and configure each SMTP server:

```
Server: mail2.bobbyseamoss.com
From Email: adam@mail2.bobbyseamoss.com
Daily Limit: 1000
```

Repeat for mail3, mail4, etc. (up to 30+ servers)

### 2. Set Time Windows (Optional)

Navigate to **Settings > Performance**:

```
Send Start Time: 08:00
Send End Time: 20:00
```

Leave empty for 24/7 sending.

### 3. Create Campaign with Automatic Messenger

When creating a campaign:
1. Select **"automatic (queue-based)"** from the Messenger dropdown
2. Configure campaign as normal
3. Start the campaign

The system will:
- Queue all emails to the database
- Distribute across SMTP servers based on capacity
- Send only during configured time window
- Respect daily limits for each server

## Configuration Details

### SMTP Server Settings

**From Email** (per server):
- Default sender address for this SMTP server
- Example: `adam@mail2.bobbyseamoss.com`
- Each server should have its own domain-specific from address

**Daily Limit** (per server):
- Maximum emails this server can send per day
- Set to `0` for unlimited (not recommended with queue system)
- Example: `1000` = max 1000 emails/day per server
- Resets at midnight (00:00) server time

### Performance Settings

**Send Time Window**:
- Restricts sending to specific hours each day
- Format: 24-hour time (HH:MM)
- Example: Start=08:00, End=20:00 (send only 8am-8pm)
- Leave both empty for 24/7 sending

**Sliding Window Limits** (existing feature):
- Still applies to queue-based sending
- Example: 2 emails per 30 minutes
- Combined with daily limits and time windows

## How It Works

### Campaign Lifecycle

1. **Campaign Created**:
   - User selects "automatic (queue-based)" messenger
   - Campaign configured with recipients, content, etc.

2. **Campaign Started**:
   - System queries all subscribers from campaign lists
   - Creates entry in `email_queue` table for each email
   - Campaign marked with `use_queue=true`, `queued_at=NOW()`
   - Admin sees log: "queued 50000 emails for campaign 123"

3. **Queue Processing**:
   - Queue processor polls every N seconds (configurable)
   - Checks if current time is within send window
   - Fetches batch of queued emails (oldest first, highest priority)
   - For each email:
     - Checks server capacities (daily remaining, sliding window)
     - Selects server with most available capacity
     - Assigns email to that server
     - Marks email as "sending"
     - Sends email via selected SMTP server
     - Updates usage counters
     - Marks email as "sent"

4. **Campaign Completion**:
   - All emails sent or failed
   - Campaign status updated to "finished"
   - Queue processor moves to next campaign

### Server Selection Algorithm

The queue processor selects SMTP servers based on:

1. **Daily Capacity**: Servers that haven't hit daily limit
2. **Sliding Window**: Servers that haven't hit rate limit
3. **Most Available**: Among eligible servers, picks one with most remaining daily capacity

Example with 3 servers (daily_limit=1000 each):
```
Server 1: 800 used, 200 remaining
Server 2: 500 used, 500 remaining ← Selected
Server 3: 900 used, 100 remaining
```

### Time Window Behavior

**During Window** (e.g., 8am-8pm):
- Queue processor actively sends emails
- Respects daily and sliding window limits
- Distributes across available servers

**Outside Window** (e.g., 8pm-8am):
- Queue processor pauses
- No emails sent
- Resumes automatically when window opens

**Multi-Day Campaigns**:
- If daily capacity insufficient for all emails
- Remaining emails stay queued
- Automatically resume next day when limits reset
- Example: 50k emails, 30k capacity/day = 2 days

## API Endpoints

### Get Queue Statistics

```bash
GET /api/queue/stats
```

Response:
```json
{
  "data": {
    "total_queued": 45000,
    "total_sending": 50,
    "total_sent": 5000,
    "total_failed": 0
  }
}
```

### Get Server Capacities

```bash
GET /api/queue/servers
```

Response:
```json
{
  "data": [
    {
      "uuid": "abc-123",
      "name": "email-mail2",
      "daily_limit": 1000,
      "daily_used": 800,
      "daily_remaining": 200,
      "from_email": "adam@mail2.bobbyseamoss.com"
    },
    {
      "uuid": "def-456",
      "name": "email-mail3",
      "daily_limit": 1000,
      "daily_used": 500,
      "daily_remaining": 500,
      "from_email": "adam@mail3.bobbyseamoss.com"
    }
  ]
}
```

## Database Schema

### email_queue Table

```sql
CREATE TABLE email_queue (
    id BIGSERIAL PRIMARY KEY,
    campaign_id INT NOT NULL,
    subscriber_id INT NOT NULL,
    status VARCHAR(20) DEFAULT 'queued',
    priority INT DEFAULT 0,
    scheduled_at TIMESTAMP WITH TIME ZONE,
    sent_at TIMESTAMP WITH TIME ZONE,
    assigned_smtp_server_uuid VARCHAR(255),
    retry_count INT DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE
);
```

**Status Values**:
- `queued` - Waiting to be sent
- `sending` - Currently being processed
- `sent` - Successfully sent
- `failed` - Failed after retries
- `cancelled` - Campaign was cancelled

### smtp_daily_usage Table

```sql
CREATE TABLE smtp_daily_usage (
    id BIGSERIAL PRIMARY KEY,
    smtp_server_uuid VARCHAR(255) NOT NULL,
    usage_date DATE NOT NULL,
    emails_sent INT DEFAULT 0,
    UNIQUE(smtp_server_uuid, usage_date)
);
```

Tracks how many emails each server sent today. Resets automatically at midnight.

### smtp_rate_limit_state Table

```sql
CREATE TABLE smtp_rate_limit_state (
    id BIGSERIAL PRIMARY KEY,
    smtp_server_uuid VARCHAR(255) NOT NULL UNIQUE,
    window_start TIMESTAMP WITH TIME ZONE,
    emails_in_window INT DEFAULT 0
);
```

Tracks sliding window rate limits per server.

## Monitoring and Troubleshooting

### Check Queue Status

```sql
-- See queue summary
SELECT status, COUNT(*)
FROM email_queue
GROUP BY status;

-- See queued emails by campaign
SELECT campaign_id, COUNT(*)
FROM email_queue
WHERE status = 'queued'
GROUP BY campaign_id;
```

### Check Daily Usage

```sql
-- Today's usage by server
SELECT s.name, u.emails_sent, s.daily_limit
FROM smtp_daily_usage u
JOIN settings...smtp s ON s.uuid = u.smtp_server_uuid
WHERE u.usage_date = CURRENT_DATE;
```

### View Failed Emails

```sql
-- See failed emails with errors
SELECT id, campaign_id, subscriber_id, last_error, retry_count
FROM email_queue
WHERE status = 'failed'
ORDER BY updated_at DESC
LIMIT 100;
```

## Best Practices

### Daily Limits

**Conservative Approach**:
- Start with 500-1000 emails/day per server
- Monitor deliverability and bounce rates
- Gradually increase if performance is good

**Aggressive Approach**:
- Some providers allow 5000+ emails/day
- Check your SMTP provider's limits
- Consider reputation warming for new domains

### Time Windows

**Business Hours**:
- 08:00-20:00 for B2B campaigns
- Avoid night-time sends when engagement is low

**All Day**:
- Leave empty for time-insensitive campaigns
- Newsletter announcements, transactional emails

### Server Distribution

**30 Servers with 1000/day limit each**:
- Total capacity: 30,000 emails/day
- 12-hour window: ~2,500 emails/hour
- With sliding window (2 per 30min): Automatically throttled

**Scale Example**:
- 100,000 recipient campaign
- 30 servers × 1000 limit = 30k/day capacity
- Campaign completes in ~4 days

## Troubleshooting

### Emails Not Sending

**Check Time Window**:
- Verify current time is within send_time_start and send_time_end
- Check server timezone matches expected timezone

**Check Daily Limits**:
- GET /api/queue/servers to see remaining capacity
- If all servers at limit, wait for midnight reset

**Check Queue Processor**:
- Ensure queue processor is running
- Check logs for errors

### Slow Sending

**Sliding Window Too Restrictive**:
- Check Performance > Sliding Window settings
- Example: 2 emails per 30min = only 96 emails/day per server
- Increase rate or widen window

**Too Few Servers**:
- Add more SMTP servers to increase total capacity
- Distribute load across more servers

### Emails Stuck in "sending"

**Server Timeout**:
- Email marked "sending" but never completed
- Check SMTP server connectivity
- Review wait_timeout and idle_timeout settings

**Manual Fix**:
```sql
-- Reset stuck emails back to queued
UPDATE email_queue
SET status = 'queued', assigned_smtp_server_uuid = NULL
WHERE status = 'sending'
  AND updated_at < NOW() - INTERVAL '10 minutes';
```

## Migration from Direct Sending

### Before Migration

1. **Test with Small Campaign**:
   - Create test campaign with 100 recipients
   - Select "automatic" messenger
   - Verify emails send correctly
   - Check logs and queue stats

2. **Configure Limits**:
   - Set conservative daily limits initially
   - Configure time windows if needed
   - Monitor first few days

### During Migration

1. **Old Campaigns** (direct send):
   - Continue using named messengers (email-mail2, etc.)
   - No changes required

2. **New Campaigns** (queue-based):
   - Select "automatic (queue-based)" messenger
   - System handles distribution automatically

3. **Gradual Rollout**:
   - Mix direct and queued campaigns
   - Test queue system with increasing volumes
   - Monitor deliverability closely

## Future Enhancements

Potential improvements to consider:

1. **Priority Queues**:
   - Mark certain campaigns as high priority
   - High priority emails sent first

2. **Dynamic Server Selection**:
   - Consider server reputation scores
   - Route based on recipient domain

3. **Smart Scheduling**:
   - Optimal send time prediction
   - Timezone-aware sending per recipient

4. **Delivery Estimation UI**:
   - Show estimated completion time when creating campaign
   - "This campaign will complete in 2 days" message

5. **Queue Dashboard**:
   - Real-time queue visualization
   - Server capacity graphs
   - Campaign progress tracking

## Support

For issues or questions:
- Check logs: Queue processor logs all operations
- Query database: Direct SQL access for debugging
- API endpoints: Real-time monitoring via /api/queue/*
- CLAUDE.md: Technical implementation details
