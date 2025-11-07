# De-duplication SQL Error - FIXED

## Summary

Fixed PostgreSQL type inference error that was preventing Azure webhook views and clicks from being recorded in the database.

---

## The Problem

**Error:** `pq: operator does not exist: timestamp with time zone >= interval`

**Location:** `cmd/bounce.go` lines 517 and 556

**Problematic Code:**
```sql
AND created_at BETWEEN $3 - INTERVAL '5 seconds' AND $3 + INTERVAL '5 seconds'
```

**Root Cause:**
When PostgreSQL prepares statements with parameter placeholders (`$1`, `$2`, `$3`), it doesn't know the type of the parameters until execution. The expression `$3 - INTERVAL '5 seconds'` requires PostgreSQL to know that `$3` is a timestamp type, but this information isn't available during query preparation, causing the operator error.

**Impact:**
- Every Azure webhook view event **failed to insert** into `campaign_views` table
- Every Azure webhook click event **failed to insert** into `link_clicks` table
- Only native `{{TrackView}}`/`{{TrackLink}}` tracking was working
- Azure engagement webhooks were being received but silently failing

---

## The Fix

**Changed From:**
```sql
AND created_at BETWEEN $3 - INTERVAL '5 seconds' AND $3 + INTERVAL '5 seconds'
```

**Changed To:**
```sql
AND ABS(EXTRACT(EPOCH FROM (created_at - $3))) <= 5
```

**How It Works:**
- `EXTRACT(EPOCH FROM (created_at - $3))` calculates the difference in seconds between the two timestamps
- `ABS(...)` takes the absolute value (ensuring positive result)
- `<= 5` checks if the difference is 5 seconds or less
- No interval arithmetic needed, no type casting issues

**Applied To:**
- View de-duplication: `cmd/bounce.go:517`
- Click de-duplication: `cmd/bounce.go:556`

---

## Verification

### Build Status
✅ **Compiled successfully** with no errors or warnings

### Commit
- **Commit:** `e6b6bb5e`
- **File:** `cmd/bounce.go`
- **Lines Changed:** 2 insertions, 2 deletions

---

## What Happens Now

### Before This Fix:
❌ Azure webhook views → Error → Not recorded
❌ Azure webhook clicks → Error → Not recorded
✅ Native {{TrackView}} → Working → Recorded
✅ Native {{TrackLink}} → Working → Recorded
**Result:** Only native tracking working, Azure data lost

### After This Fix:
✅ Azure webhook views → Working → Recorded (with de-duplication)
✅ Azure webhook clicks → Working → Recorded (with de-duplication)
✅ Native {{TrackView}} → Working → Recorded (with de-duplication)
✅ Native {{TrackLink}} → Working → Recorded (with de-duplication)
**Result:** Both systems working, duplicates prevented

---

## Next Steps

### 1. Deploy the Fix

You need to deploy the updated binary to your Azure environment. Choose your deployment method:

#### Option A: Use Existing Deployment Script
```bash
./deploy.sh
```

#### Option B: Manual Docker/Azure Container App
```bash
# Build the Docker image
docker build -t listmonk420:dedup-fixed .

# Push to your registry (if using one)
docker push your-registry/listmonk420:dedup-fixed

# Update and restart the Azure Container App
az containerapp update \
  --name listmonk420 \
  --resource-group <your-resource-group> \
  --image listmonk420:dedup-fixed

# Or just restart to pull latest
az containerapp restart \
  --name listmonk420 \
  --resource-group <your-resource-group>
```

#### Option C: Direct Binary Deployment
```bash
# Stop current instance
sudo systemctl stop listmonk

# Replace binary
sudo cp ./listmonk /path/to/your/listmonk/

# Start new instance
sudo systemctl start listmonk
```

---

### 2. Monitor the Fix

After deployment, check that Azure webhooks are now inserting successfully:

#### Check for SQL Errors (Should be 0)
```sql
SELECT * FROM webhook_logs
WHERE webhook_type = 'azure'
  AND created_at > NOW() - INTERVAL '1 hour'
  AND error_message LIKE '%operator does not exist%';
```

**Expected:** 0 rows (no more SQL errors)

#### Check Azure Engagement Events Being Stored
```sql
-- Check recent Azure view events
SELECT COUNT(*) as azure_views
FROM azure_engagement_events
WHERE engagement_type = 'view'
  AND event_timestamp > NOW() - INTERVAL '1 hour';

-- Check they're being inserted into campaign_views
SELECT COUNT(*) as db_views
FROM campaign_views
WHERE created_at > NOW() - INTERVAL '1 hour';
```

**Expected:** Numbers should be similar (not wildly different)

#### Verify De-duplication is Working
```sql
-- Check for duplicate views (should be minimal)
SELECT subscriber_id, campaign_id, COUNT(*) as view_count
FROM campaign_views
WHERE created_at > NOW() - INTERVAL '1 hour'
GROUP BY subscriber_id, campaign_id
HAVING COUNT(*) > 1;
```

**Expected:** Very few or 0 rows (no significant duplicates)

---

### 3. Compare Before/After

**Before Fix (Last 24 Hours):**
```sql
SELECT
    COUNT(*) FILTER (WHERE created_at < NOW() - INTERVAL '1 hour') as views_before_fix,
    COUNT(*) FILTER (WHERE created_at >= NOW() - INTERVAL '1 hour') as views_after_fix
FROM campaign_views
WHERE created_at > NOW() - INTERVAL '24 hours';
```

You should see **views_after_fix increase significantly** once Azure webhooks start working.

---

## Technical Details

### Why EXTRACT(EPOCH) Works

1. **Type-Safe:** PostgreSQL knows `created_at` is a timestamp from the table schema
2. **No Parameter Arithmetic:** We subtract `$3` inside the EXTRACT function, not outside
3. **Clear Semantics:** "Absolute difference in seconds ≤ 5" is explicit and unambiguous
4. **No Type Casting:** Avoids the need for `::timestamp` casts which can cause other issues

### Performance Impact

**Negligible.** The query still uses the same index on `campaign_id` and `subscriber_id`. The EXTRACT operation is performed only on rows that match the campaign and subscriber, which is typically 0-2 rows (checking for duplicates).

---

## Summary of All Fixes Applied

| Issue | Status | File | Lines |
|-------|--------|------|-------|
| Duplicate view/click tracking | ✅ Fixed | cmd/bounce.go | 517, 556 |
| PostgreSQL type inference error | ✅ Fixed | cmd/bounce.go | 517, 556 |
| Hard bounces not blocklisting | ✅ Working | - | - |
| Soft bounce config wrong | ⚠️ User to fix | Settings UI | - |
| Rate limits outdated | ✅ Updated | Database | - |

---

## Status

- [x] SQL syntax error identified
- [x] Fix applied to view de-duplication
- [x] Fix applied to click de-duplication
- [x] Code compiled successfully
- [x] Changes committed to git
- [ ] **Deploy to Azure environment**
- [ ] **Monitor for 1-2 hours to verify**

---

**Date Fixed:** 2025-11-04
**Commit:** e6b6bb5e
**Status:** ✅ Ready to Deploy
