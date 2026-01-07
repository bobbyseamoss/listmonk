# Uncommitted Changes Summary

**Last Updated**: 2025-12-09

## Modified Files (deployed but not committed)

### Backend Core Changes

| File | Description |
|------|-------------|
| `cmd/init.go` | Fixed sliding window bug - only apply limit when feature enabled |
| `cmd/upgrade.go` | Registered v7.3.0 migration |
| `internal/queue/processor.go` | Bypass time window logic for queue processing |
| `internal/core/campaigns.go` | Pass bypass_time_window in Create/Update |
| `models/models.go` | Added BypassTimeWindow field to Campaign struct |
| `queries.sql` | Added bypass_time_window to create-campaign and update-campaign |

### Frontend Changes

| File | Description |
|------|-------------|
| `frontend/src/views/Campaign.vue` | Added bypass time window toggle UI |

### New Files

| File | Description |
|------|-------------|
| `internal/migrations/v7.3.0.go` | Migration for bypass_time_window column |

### Documentation

| File | Description |
|------|-------------|
| `CLAUDE.md` | Updated with bypass time window feature docs |
| `dev/HANDOFF-2025-11-26.md` | Previous session handoff |
| `dev/HANDOFF-2025-11-28.md` | Current session handoff |
| `dev/active/bypass-time-window/context.md` | Feature implementation context |
| `dev/active/sliding-window-bug-fix/context.md` | Bug fix context |

## What These Changes Do

### 1. Bypass Time Window Feature (v7.3.0)
- Adds per-campaign option to bypass global sending time window
- Useful for testing campaigns outside normal hours
- Toggle appears in Campaign form when messenger is "automatic"

### 2. Sliding Window Bug Fix
- Fixed bug where sliding window limit was enforced even when disabled
- Queue processor now respects `app.message_sliding_window = false` setting

## Deployment Status

All changes are **DEPLOYED TO PRODUCTION** via Docker image:
- Image: `listmonk420acr.azurecr.io/listmonk420:latest`
- Revision: `listmonk420--deploy-20251126-093256`
- Status: Healthy

## Additional Changes (December 9, 2025)

### New Files Created

| File | Description |
|------|-------------|
| `internal/ratelimit/tracker.go` | Rate limit tracker module - auto-disables throttled SMTP servers |
| `internal/migrations/v7.6.0.go` | Migration for email header settings (abuse_email, feedback_sender_id) |

### Backend Changes (Dec 9)

| File | Description |
|------|-------------|
| `cmd/main.go` | Added rateLimitTracker initialization and App struct field |
| `cmd/bounce.go` | Added Azure webhook rate limit detection |
| `internal/queue/processor.go` | Round-robin SMTP selection, rate limit handler callback |
| `internal/manager/manager.go` | Added Reply-To, X-Report-Abuse, Feedback-ID headers |
| `models/settings.go` | Added AppAbuseEmail, AppFeedbackSenderId fields |
| `cmd/init.go` | Load new settings from koanf |
| `cmd/upgrade.go` | Registered v7.6.0 migration |

### Frontend Changes (Dec 9)

| File | Description |
|------|-------------|
| `frontend/src/views/settings/general.vue` | Added Abuse Email and Feedback Sender ID fields |
| `frontend/src/views/WebhookLogs.vue` | Renamed columns, updated status display |

## To Commit (All Changes)

```bash
# Stage all changes
git add cmd/init.go cmd/upgrade.go internal/queue/processor.go \
        internal/core/campaigns.go models/models.go queries.sql \
        frontend/src/views/Campaign.vue internal/migrations/v7.3.0.go \
        internal/migrations/v7.6.0.go internal/ratelimit/tracker.go \
        cmd/main.go cmd/bounce.go internal/manager/manager.go \
        models/settings.go frontend/src/views/settings/general.vue \
        frontend/src/views/WebhookLogs.vue CLAUDE.md

git commit -m "Add multi-SMTP round-robin, email headers, rate limit auto-disable

Features:
- Round-robin SMTP server selection when servers have equal capacity
- Email headers: Reply-To, X-Report-Abuse, Feedback-ID per Google spec
- Rate limit auto-disable: Throttled servers disabled with auto-re-enable
- Settings UI for Abuse Email and Feedback Sender ID
- Webhook Logs UI improvements

Bug fixes:
- Fix sliding window limit being applied when feature is disabled
- Fix all emails going to single server despite multiple enabled

Migrations:
- v7.3.0: Add bypass_time_window BOOLEAN column to campaigns table
- v7.6.0: Add app.abuse_email and app.feedback_sender_id settings"
```

## Deployment Status

**Latest Revision**: `listmonk420--deploy-20251209-134645`
**All changes are DEPLOYED TO PRODUCTION**
