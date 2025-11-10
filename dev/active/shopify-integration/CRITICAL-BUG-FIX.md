# CRITICAL BUG FIX - Settings Not Persisting

**Date**: 2025-11-08 14:30:00 UTC
**Severity**: CRITICAL
**Status**: ✅ FIXED

---

## Problem

Shopify settings would not save or persist in the database. After clicking Save:
- No errors in console or logs
- Settings would revert to defaults on page reload
- Form changes appeared to work but didn't persist

---

## Root Cause

**listmonk uses a dual-format settings storage pattern that was not followed:**

### The Pattern (Required for ALL nested struct settings)

Settings must exist in BOTH formats in the database:

1. **Flat keys** (individual fields):
   ```
   shopify.enabled
   shopify.webhook_secret
   shopify.attribution_window_days
   ```

2. **Nested key** (complete JSON object):
   ```
   shopify = {"enabled": false, "webhook_secret": "", "attribution_window_days": 7}
   ```

### Why Both Are Needed

```sql
-- get-settings query (queries.sql:1127)
SELECT JSON_OBJECT_AGG(key, value) AS settings
FROM (SELECT * FROM settings ORDER BY key) t;
```

This query builds a JSON object from ALL keys in the settings table:
- Flat keys like `app.batch_size` become top-level JSON properties
- Nested keys like `security.oidc` become nested JSON objects
- The Go Settings struct unmarshals from this combined JSON

**Without the nested key, the Go struct field remains nil/zero-valued.**

### What Was Wrong

Migration v7.0.0 only created flat keys:
```sql
INSERT INTO settings (key, value) VALUES
('shopify.enabled', 'false'),
('shopify.webhook_secret', '""'),
('shopify.attribution_window_days', '7')
-- MISSING: ('shopify', '{"enabled": false, ...}')
```

### Evidence

Comparing with working nested settings:

```sql
-- OIDC (working) - has BOTH formats:
SELECT key FROM settings WHERE key LIKE 'security.oidc%';
/*
security.oidc
security.oidc.enabled
security.oidc.client_id
security.oidc.client_secret
security.oidc.provider_url
security.oidc.provider_name
security.oidc.auto_create_users
security.oidc.default_user_role_id
security.oidc.default_list_role_id
*/

-- Shopify (broken) - only had flat keys:
SELECT key FROM settings WHERE key LIKE 'shopify%';
/*
shopify.enabled
shopify.webhook_secret
shopify.attribution_window_days
-- MISSING: shopify
*/
```

---

## Fix Applied

### 1. Updated Migration

File: `internal/migrations/v7.0.0.go`

**Before:**
```go
if _, err := db.Exec(`
    INSERT INTO settings (key, value) VALUES
    ('shopify.enabled', 'false'),
    ('shopify.webhook_secret', '""'),
    ('shopify.attribution_window_days', '7')
    ON CONFLICT (key) DO NOTHING;
`); err != nil {
    return err
}
```

**After:**
```go
if _, err := db.Exec(`
    INSERT INTO settings (key, value) VALUES
    ('shopify', '{"enabled": false, "webhook_secret": "", "attribution_window_days": 7}'),
    ('shopify.enabled', 'false'),
    ('shopify.webhook_secret', '""'),
    ('shopify.attribution_window_days', '7')
    ON CONFLICT (key) DO NOTHING;
`); err != nil {
    return err
}
```

### 2. Manual Database Fix (Production)

Since the migration already ran, manually inserted the missing key:

```sql
INSERT INTO settings (key, value) VALUES
('shopify', '{"enabled": false, "webhook_secret": "", "attribution_window_days": 7}')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;
```

### 3. Redeployed

Revision: `listmonk420--deploy-20251108-092758`

---

## Verification

### Check Database Has All Keys

```sql
SELECT key, value FROM settings
WHERE key LIKE 'shopify%' OR key = 'shopify'
ORDER BY key;
```

**Expected Output (4 rows):**
```
               key               |                      value
---------------------------------+--------------------------------------------------------------------
 shopify                         | {"enabled": false, "webhook_secret": "", "attribution_window_days": 7}
 shopify.attribution_window_days | 7
 shopify.enabled                 | false
 shopify.webhook_secret          | ""
```

### Test Settings Save/Load

1. Go to Settings > Shopify
2. Toggle enable ON
3. Enter webhook secret: "test-secret-123"
4. Set attribution window: 14 days
5. Click Save
6. Refresh page (Ctrl+Shift+R)
7. Settings should persist:
   - Enable toggle still ON
   - Webhook secret shows masked (••••••••••••)
   - Attribution window shows 14 days

---

## How This Pattern Works

### Settings Storage Architecture

```
Database (PostgreSQL)
  ↓
settings table
  key (TEXT PRIMARY KEY)  |  value (JSONB)
  ------------------------+------------------
  app.batch_size          |  1000
  shopify                 |  {"enabled": false, ...}  ← Nested object
  shopify.enabled         |  false                    ← Flat field
  shopify.webhook_secret  |  ""                       ← Flat field
  security.oidc           |  {"enabled": false, ...}  ← Nested object
  security.oidc.enabled   |  false                    ← Flat field
  ↓
get-settings query (JSON_OBJECT_AGG)
  ↓
Combined JSON
{
  "app.batch_size": 1000,
  "shopify": {"enabled": false, "webhook_secret": "", "attribution_window_days": 7},
  "shopify.enabled": false,
  "shopify.webhook_secret": "",
  "security.oidc": {"enabled": false, ...},
  "security.oidc.enabled": false,
  ...
}
  ↓
Go json.Unmarshal
  ↓
models.Settings struct
type Settings struct {
    AppBatchSize int `json:"app.batch_size"`

    Shopify struct {                          ← Unmarshals from "shopify" key
        Enabled bool   `json:"enabled"`
        WebhookSecret string `json:"webhook_secret"`
        AttributionWindowDays int `json:"attribution_window_days"`
    } `json:"shopify"`

    OIDC struct {                             ← Unmarshals from "security.oidc" key
        Enabled bool   `json:"enabled"`
        ...
    } `json:"security.oidc"`
}
```

### Why Flat Keys Exist

Individual flat keys allow:
1. Direct SQL updates without JSON manipulation
2. Backward compatibility with old code
3. Per-field change tracking
4. Simpler database queries for single settings

### Why Nested Key Exists

The nested key enables:
1. Efficient bulk retrieval via `JSON_OBJECT_AGG`
2. Proper Go struct unmarshaling
3. Type-safe settings access in code
4. Clean Settings API

---

## Critical Learning

### When Adding Nested Settings

**ALWAYS add BOTH formats:**

1. **Individual flat keys** for each field:
   ```sql
   INSERT INTO settings (key, value) VALUES
   ('my_feature.field1', 'value1'),
   ('my_feature.field2', 'value2'),
   ```

2. **Single nested key** with complete object:
   ```sql
   ('my_feature', '{"field1": "value1", "field2": "value2"}'),
   ```

### Template for New Nested Settings

```go
// In migration:
if _, err := db.Exec(`
    INSERT INTO settings (key, value) VALUES
    ('feature_name', '{"field1": "default1", "field2": "default2"}'),
    ('feature_name.field1', '"default1"'),
    ('feature_name.field2', '"default2"')
    ON CONFLICT (key) DO NOTHING;
`); err != nil {
    return err
}
```

```go
// In models/settings.go:
FeatureName struct {
    Field1 string `json:"field1"`
    Field2 string `json:"field2"`
} `json:"feature_name"`
```

---

## Impact

**Before Fix:**
- ❌ Shopify settings unusable
- ❌ Integration non-functional
- ❌ Silent failure (no errors)

**After Fix:**
- ✅ Settings save and persist correctly
- ✅ Integration fully functional
- ✅ Password masking works
- ✅ Form reactivity works

---

## Related Files

- Migration: `/home/adam/listmonk/internal/migrations/v7.0.0.go`
- Settings model: `/home/adam/listmonk/models/settings.go`
- Settings query: `/home/adam/listmonk/queries.sql:1127`
- Settings core: `/home/adam/listmonk/internal/core/settings.go`

---

## For Future Developers

If you're adding a new nested settings struct:

1. **Check existing patterns** - Look at OIDC, SecurityCaptcha, BouncePostmark
2. **Add nested key** - Don't just add flat keys
3. **Test thoroughly** - Save settings, reload page, verify persistence
4. **Verify database** - Check both key formats exist

**Common symptoms of missing nested key:**
- Settings don't persist
- No error messages
- Frontend form seems to work
- Database has flat keys but not nested key
- Settings struct field is nil/zero-valued

---

**End of Critical Bug Fix Documentation**
