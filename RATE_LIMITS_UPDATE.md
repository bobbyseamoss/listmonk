# ✅ Azure Rate Limits Updated

## Summary

Successfully updated all rate limits in the database to reflect your new Azure Communication Services subscription limits of **500/minute** and **2000/hour**.

---

## 📊 Changes Applied

### Account-Wide Limits

| Setting | Old Value | New Value |
|---------|-----------|-----------|
| `app.account_rate_limit_per_minute` | 30 | **500** |
| `app.account_rate_limit_per_hour` | 100 | **2000** |

These are the **hard limits** enforced across all 29 SMTP servers combined.

---

### Individual Server Limits (All 29 Servers)

| Setting | Old Value | New Value | Calculation |
|---------|-----------|-----------|-------------|
| `sliding_window_rate` | 4 per 30min | **16 per 1min** | 500/min ÷ 29 servers ≈ 17/min (using 16 for safety) |
| `sliding_window_duration` | 30m | **1m** | Changed to 1-minute windows for better control |
| `daily_limit` | 80 | **1,650** | (2000/hour × 24 hours) ÷ 29 ≈ 1,655/day (using 1,650 for safety) |

**Servers updated:** 29/29 ✓

---

## 🎯 Effective Throughput

### Account-Wide Capacity:
- **Per minute:** Up to 500 emails
- **Per hour:** Up to 2,000 emails
- **Per day:** Up to 48,000 emails (2000/hour × 24)

### Per-Server Capacity:
- **Per minute:** 16 emails (total across all servers: 464/min)
- **Per hour:** 960 emails (16/min × 60, total: ~27,840/hour)
- **Per day:** 1,650 emails (total across all servers: 47,850/day)

### Safety Margins Built-In:
- **Minute limit:** Using 464/min of 500/min available (93% utilization, 7% buffer)
- **Daily limit:** Using 47,850/day of 48,000/day available (99.7% utilization, 0.3% buffer)

This conservative approach prevents hitting Azure's hard limits due to timing variations or burst sending.

---

## 📈 Performance Comparison

### Before (Old Limits):
- Account: 30/min, 100/hour
- Per server: 4 per 30min = 8/hour
- Total capacity: ~232 emails/hour (29 servers × 8/hour)

### After (New Limits):
- Account: 500/min, 2000/hour
- Per server: 16/min = 960/hour
- Total capacity: **2,000 emails/hour** (limited by account)

**Improvement:** **8.6x faster** (from ~232/hour to 2,000/hour)

---

## 🚀 What Happens Now

### Immediate Effect:
The new limits are **active immediately** in the database.

### Requires Restart:
Listmonk needs to be **restarted** to reload the settings from the database:

```bash
# If using Azure Container Apps:
az containerapp restart \
  --name listmonk420 \
  --resource-group <your-resource-group>

# If using systemd:
sudo systemctl restart listmonk

# Or via the UI:
Settings > Reload (button)
```

### After Restart:
- Queue processor will use new limits
- Campaigns will send at up to 500 emails/minute
- Each server respects its 16/minute sliding window
- Account-wide limits prevent exceeding 500/min and 2000/hour

---

## ⚙️ How the Limits Work

### Multi-Layer Rate Limiting:

1. **Per-Server Sliding Window** (First line of defense)
   - Each of 29 servers limited to 16 emails/minute
   - Prevents any single server from hogging capacity
   - Total theoretical max: 464 emails/minute

2. **Account-Wide Rate Limit** (Second line of defense)
   - Hard limit: 500 emails/minute across all servers
   - Hard limit: 2,000 emails/hour across all servers
   - Queue processor checks before sending each batch

3. **Azure API Limit** (Final enforcement)
   - Azure enforces 500/minute and 2000/hour at API level
   - Our settings keep us safely under these limits

---

## 📋 Verification Queries

### Check Current Settings:

```sql
-- Account-wide limits
SELECT key, value
FROM settings
WHERE key IN ('app.account_rate_limit_per_minute', 'app.account_rate_limit_per_hour');

-- Sample server limits
SELECT
    elem->>'name' as server,
    elem->>'sliding_window_rate' as rate,
    elem->>'sliding_window_duration' as duration,
    elem->>'daily_limit' as daily_limit
FROM jsonb_array_elements((SELECT value::jsonb FROM settings WHERE key = 'smtp')) elem
LIMIT 5;
```

### Monitor Usage:

```sql
-- Check daily usage per server
SELECT
    smtp_server_uuid,
    usage_date,
    emails_sent
FROM smtp_daily_usage
WHERE usage_date = CURRENT_DATE
ORDER BY emails_sent DESC;

-- Check if any server is hitting daily limit
SELECT
    COUNT(*) as servers_at_limit
FROM smtp_daily_usage sdu
WHERE usage_date = CURRENT_DATE
  AND emails_sent >= 1650;
```

---

## ⚠️ Important Notes

### Daily Limits:
Each server can send up to **1,650 emails/day**. Total across all 29 servers = **47,850/day**.

If a campaign has more than 47,850 emails, it will:
- Send 47,850 on day 1
- Continue sending remaining emails on day 2
- Automatically respect daily limits

### Sliding Window:
With **1-minute windows**, the system checks every minute:
- "Have we sent 16 emails in the last 60 seconds on this server?"
- If yes, wait until the window rolls over
- If no, send the next batch

This provides smooth, consistent sending rather than bursts.

### Queue Behavior:
With `messenger="automatic"`, the queue processor will:
1. Check account-wide limits (500/min, 2000/hour)
2. Select server with most remaining capacity
3. Check server's sliding window (16/min)
4. Send batch if all limits allow
5. Update usage counters

---

## 🎯 Recommended Campaign Settings

For optimal throughput with your new limits:

### Large Campaigns (10,000+ emails):
- **Messenger:** `automatic` (queue-based)
- **Expected throughput:** 500 emails/minute = 30,000/hour
- **Time to send 10,000:** ~20 minutes
- **Time to send 50,000:** ~100 minutes (~1.7 hours)

### Immediate Campaigns:
- **Messenger:** `email` (direct sending, uses round-robin)
- **Expected throughput:** ~464 emails/minute (per-server limits)
- **Use for:** Time-sensitive campaigns under 5,000 emails

---

## 📊 Updated Architecture

```
Campaign (automatic messenger)
    ↓
Email Queue (queues all emails)
    ↓
Queue Processor (checks limits)
    ↓
Account Rate Limit Check (500/min, 2000/hour)
    ↓
Server Selection (29 servers available)
    ↓
Server Rate Limit Check (16/min per server)
    ↓
Daily Limit Check (1650/day per server)
    ↓
Azure SMTP API (500/min, 2000/hour enforced by Azure)
```

---

## ✅ Status

- [x] Account-wide rate limits updated to 500/min, 2000/hour
- [x] All 29 SMTP servers updated to 16/min, 1650/day
- [x] Verified all servers have new settings
- [x] Conservative buffer built in to prevent limit violations
- [ ] **Restart listmonk to apply changes**

---

## 📞 Next Steps

1. **Restart listmonk** to load new settings
2. **Test with a small campaign** (100-500 emails) to verify
3. **Monitor usage** for first few hours
4. **Scale up** to larger campaigns once verified

---

**Date Updated:** 2025-11-04
**Updated By:** Claude Code
**Servers Updated:** 29/29
**Status:** ✅ Complete - Awaiting Restart
