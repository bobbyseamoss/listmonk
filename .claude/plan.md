# Domain-Based SMTP Routing Configuration

## Overview

Add a new Settings section that allows users to configure which SMTP servers handle specific email domains (Gmail, Outlook, iCloud, Yahoo, Hotmail, AOL). Each domain can be routed to one or more selected SMTP servers via checkboxes.

## Implementation Plan

### 1. Database Schema (Settings JSON)

Add a new settings key `domain_routing` that stores routing rules:

```json
{
  "gmail.com": ["uuid-1", "uuid-2"],
  "outlook.com": ["uuid-1"],
  "icloud.com": ["uuid-3"],
  "yahoo.com": ["uuid-1", "uuid-2"],
  "hotmail.com": ["uuid-1"],
  "aol.com": ["uuid-1", "uuid-2", "uuid-3"]
}
```

**Files to modify:**
- `models/settings.go` - Add `DomainRouting map[string][]string` field
- `schema.sql` - Add default empty setting (in migration)
- `internal/migrations/v7.7.0.go` - Create migration to add default setting

### 2. Backend Changes

**Files to modify:**
- `models/settings.go` - Add DomainRouting struct field:
  ```go
  DomainRouting map[string][]string `json:"domain_routing"`
  ```

- `internal/queue/processor.go` - Replace hardcoded domain maps with dynamic settings:
  - Remove `sendgridOnlyDomains` and `azureOnlyDomains` variables
  - Modify `selectServer()` to fetch domain routing from settings
  - Look up domain in settings, get list of allowed server UUIDs
  - Select from allowed servers using round-robin when multiple are configured

### 3. Frontend Changes

**New file:** `frontend/src/views/settings/domain-routing.vue`

Component structure:
- List of 6 popular domains: Gmail.com, Outlook.com, iCloud.com, Yahoo.com, Hotmail.com, AOL.com
- For each domain, show a dropdown with checkboxes of all enabled SMTP servers
- Checkbox label shows server name (e.g., "email-mail2", "email-sendgrid2")
- Empty selection = round-robin across all servers (default behavior)

**Files to modify:**
- `frontend/src/views/Settings.vue` - Add new tab for Domain Routing
- Import and register the new component

### 4. Queue Processor Logic

**Modified `selectServer()` flow:**

```
1. Extract domain from subscriber email
2. Check if domain has routing rules in settings.DomainRouting
3. If yes:
   a. Get list of allowed server UUIDs for this domain
   b. Filter capacities to only include allowed servers
   c. Use round-robin among allowed servers with capacity
4. If no (or empty list):
   a. Use existing round-robin logic across ALL servers
```

**Caching consideration:**
- Settings are already fetched dynamically via `getSettings()`
- Domain routing will be included in settings fetch
- No additional caching needed

### 5. File Changes Summary

| File | Change Type | Description |
|------|-------------|-------------|
| `models/settings.go` | Modify | Add DomainRouting field |
| `internal/migrations/v7.7.0.go` | New | Migration for default setting |
| `cmd/upgrade.go` | Modify | Register migration |
| `schema.sql` | Modify | Add default domain_routing setting |
| `internal/queue/processor.go` | Modify | Dynamic domain routing logic |
| `frontend/src/views/settings/domain-routing.vue` | New | UI component |
| `frontend/src/views/Settings.vue` | Modify | Add tab for domain routing |

### 6. UI Design

```
┌─────────────────────────────────────────────────────────────────┐
│ Domain Routing                                                   │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│ Configure which SMTP servers handle specific email domains.      │
│ Leave empty to use all servers (round-robin).                   │
│                                                                  │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ gmail.com                                                    │ │
│ │ ┌─────────────────────────────────────────────────────────┐ │ │
│ │ │ Select servers...                               [▼]     │ │ │
│ │ │ ☑ email-mail2 (smtp.azurecomm.net)                     │ │ │
│ │ │ ☐ email-sendgrid2 (smtp.sendgrid.net)                  │ │ │
│ │ └─────────────────────────────────────────────────────────┘ │ │
│ └─────────────────────────────────────────────────────────────┘ │
│                                                                  │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ outlook.com                                                  │ │
│ │ ┌─────────────────────────────────────────────────────────┐ │ │
│ │ │ Select servers...                               [▼]     │ │ │
│ │ │ ☑ email-mail2 (smtp.azurecomm.net)                     │ │ │
│ │ │ ☐ email-sendgrid2 (smtp.sendgrid.net)                  │ │ │
│ │ └─────────────────────────────────────────────────────────┘ │ │
│ └─────────────────────────────────────────────────────────────┘ │
│                                                                  │
│ [Similar for iCloud.com, Yahoo.com, Hotmail.com, AOL.com]       │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 7. Implementation Order

1. **Backend model** - Add DomainRouting to Settings struct
2. **Migration** - Create v7.7.0 migration with default empty setting
3. **Queue processor** - Update selectServer() to use dynamic routing
4. **Frontend component** - Create domain-routing.vue
5. **Settings integration** - Add tab to Settings.vue
6. **Testing** - Verify routing works correctly
7. **Deploy** - Build and deploy to Azure

### 8. Edge Cases

- **No servers selected for a domain**: Fall back to round-robin across all servers
- **Selected server not enabled**: Skip it, try next selected server
- **All selected servers at capacity**: Fall back to round-robin
- **Domain not in list**: Round-robin across all servers
- **Subdomain handling**: Extract base domain (e.g., mail.google.com → google.com)

### 9. Estimated Changes

- ~20 lines in models/settings.go
- ~50 lines in migration
- ~80 lines in processor.go (replacing ~50 lines of hardcoded maps)
- ~150 lines in domain-routing.vue
- ~10 lines in Settings.vue
