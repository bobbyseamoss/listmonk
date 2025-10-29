# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

listmonk is a standalone, self-hosted newsletter and mailing list manager written in Go (backend) and Vue 2 (frontend). It uses PostgreSQL as its database and is distributed as a single binary with embedded static assets.

## Build Commands

### Backend (Go)
- `make build` - Build the backend binary to `./listmonk`
- `make run` - Run backend in dev mode (loads frontend assets from `frontend/dist`)
- `make test` - Run Go tests
- `CGO_ENABLED=0 go build -o listmonk cmd/*.go` - Build without Make

### Frontend (Vue 2 + Buefy)
- `make build-frontend` - Build both main frontend and email-builder
- `make build-email-builder` - Build only the email-builder component
- `make run-frontend` - Run frontend dev server (requires email-builder to be built first)
- `cd frontend && yarn install` - Install frontend dependencies
- `cd frontend && yarn dev` - Run frontend dev server directly
- `cd frontend && yarn build` - Build frontend for production
- `cd frontend && yarn lint` - Lint frontend code

### Full Distribution
- `make dist` - Build complete distribution with embedded assets (requires stuffbin)
- `make pack-bin` - Pack static assets into binary using stuffbin

### Docker Development
- `make dev-docker` - Build and run complete development environment
- `make run-backend-docker` - Run backend with docker dev config
- `make init-dev-docker` - Initialize development database
- `make rm-dev-docker` - Tear down docker environment

### Database
- `./listmonk --install` - Install database schema
- `./listmonk --upgrade` - Upgrade existing database (idempotent)
- `./listmonk --new-config` - Generate config.toml

## Architecture

### Backend Structure

**cmd/** - HTTP handlers and application entry point
- Each file represents a domain area (campaigns, subscribers, lists, etc.)
- `main.go` - Application initialization and global App struct
- `handlers.go` - Generic HTTP handler utilities
- `install.go` / `upgrade.go` - Database setup and migrations
- HTTP framework: Echo v4

**internal/core/** - Business logic and data operations
- CRUD operations for all domain entities
- Returns `echo.HTTPError` that can be directly used in HTTP responses
- Used by cmd/ handlers to perform database operations
- Files: `campaigns.go`, `subscribers.go`, `lists.go`, `bounces.go`, etc.

**internal/manager/** - Campaign execution engine
- Handles scheduling, processing, and queuing of campaigns
- Manages message delivery through various messengers
- `pipe.go` - Campaign processing pipelines
- `message.go` - Message construction and template rendering

**internal/messenger/** - Message delivery backends
- `email/` - Email delivery via SMTP
- `postback/` - HTTP postback messenger
- Interface defined in `internal/manager/manager.go`

**internal/auth/** - Authentication system
- Session management
- OIDC support

**internal/bounce/** - Bounce handling
- `mailbox/` - POP3 mailbox polling for bounces
- `webhooks/` - Webhook handlers (SES, SendGrid, Postmark, etc.)

**internal/media/** - Media storage
- `providers/filesystem/` - Local filesystem storage
- `providers/s3/` - S3-compatible storage

**internal/migrations/** - Database migration files
- Version-specific migration Go files (e.g., `v2.4.0.go`)

**models/** - Data models and database queries
- `models.go` - Core data structures (Subscriber, Campaign, List, etc.)
- `queries.go` - Database query loader (uses goyesql)
- `settings.go` - Application settings structures

### Frontend Structure

**frontend/src/** - Vue 2 application
- `main.js` - Application entry point
- `App.vue` - Root component
- `router/` - Vue Router configuration
- `store/` - Vuex state management
- `api/` - HTTP API client
- `views/` - Page components (Dashboard, Campaigns, Subscribers, Settings, etc.)
- `components/` - Reusable components (Editor, ListSelector, CopyText, etc.)

**frontend/email-builder/** - Visual email editor
- Separate TypeScript/Vue project
- Built output copied to `frontend/public/static/email-builder`

### Database

- **PostgreSQL** - Primary data store
- **schema.sql** - Database schema with tables, indexes, enums
- **queries.sql** - Named SQL queries loaded via goyesql
- **permissions.json** - Role-based permission definitions

Key tables:
- `subscribers` - Subscriber records
- `lists` - Mailing lists
- `subscriber_lists` - Many-to-many subscription relationships
- `campaigns` - Email campaigns
- `templates` - Email templates
- `bounces` - Bounce records
- `users` - Admin users
- `roles` - RBAC roles

### Asset Embedding

listmonk uses `stuffbin` to embed static assets into the binary:
- Frontend dist files (`frontend/dist` → `/admin`)
- Email templates (`static/email-templates`)
- SQL files (`schema.sql`, `queries.sql`)
- Config sample (`config.toml.sample`)
- i18n translations (`i18n/`)

## Development Workflow

1. **Backend only changes**: Run `make run` or directly `go run cmd/*.go`
2. **Frontend only changes**: Run `make run-frontend` for live reload
3. **Full stack development**:
   - Terminal 1: `make run` (backend)
   - Terminal 2: `make run-frontend` (frontend dev server)
4. **Database changes**: Update `schema.sql` and create migration in `internal/migrations/`
5. **New SQL queries**: Add to `queries.sql` with `-- name: query-name` comments

## Important Patterns

### Database Query Pattern
- Queries defined in `queries.sql` with `-- name: query-name` markers
- Loaded via goyesql into `models.Queries` struct
- Access via `app.queries.QueryName.Exec()` or `.Get()`

### Error Handling
- Core methods return `echo.HTTPError`
- Can be directly returned from handlers: `return app.core.GetCampaign(...)`
- Custom errors use `echo.NewHTTPError(statusCode, message)`

### Template Rendering
- Campaign templates use Go's `html/template`
- Support for custom template functions (Sprig library + custom funcs)
- Email-specific template tags: `{{ TrackLink }}`, `{{ UnsubscribeURL }}`, etc.

### Messenger Interface
All messengers implement:
```go
type Messenger interface {
    Name() string
    Push(models.Message) error
    Flush() error
    Close() error
}
```

## Configuration

- **config.toml** - Main configuration file (see `config.toml.sample`)
- Environment variables override config values (use `LISTMONK_` prefix)
- Settings also stored in database `settings` table (UI-editable)

## Testing

- Go tests: `make test` or `go test ./...`
- Frontend tests: Cypress (see `frontend/package.json`)
- No test files present in current codebase structure

## Dependencies

**Backend:**
- Echo v4 - HTTP framework
- sqlx - Database toolkit
- koanf - Configuration management
- goyesql - SQL query file loader
- stuffbin - Asset embedding

**Frontend:**
- Vue 2.7
- Buefy (Bulma + Vue)
- Vue Router 3
- Vuex 3
- TinyMCE - Rich text editor
- CodeMirror 6 - Code editor
- Chart.js - Analytics charts

## Multi-SMTP Server Support (30+ Servers)

Listmonk supports unlimited named SMTP servers with individual bounce mailboxes:

### Named SMTP Servers
- Each SMTP server can have a `name` field (e.g., "primary", "secondary")
- Named servers create individual messengers: `email-primary`, `email-secondary`
- Unnamed servers are grouped into the default `email` messenger with random selection
- Campaigns can explicitly select which messenger (SMTP server) to use

### Bounce Mailboxes per SMTP Server
- Each SMTP server can be linked to a specific POP bounce mailbox via `bounce_mailbox_uuid`
- Multiple bounce mailboxes are supported (30+)
- Each bounce mailbox runs independently in its own goroutine
- Mailboxes are identified by UUID and optional name

### Configuration Flow
1. **Add Bounce Mailboxes**: Settings > Bounces > Add New
   - Set name (e.g., "bounce-primary")
   - Configure POP server, credentials, scan interval
   - Note the mailbox name/UUID

2. **Add SMTP Servers**: Settings > SMTP > Add New
   - Set name (e.g., "primary") to create dedicated messenger
   - Link to bounce mailbox via dropdown selector
   - Each server maintains its own connection pool

3. **Campaign Selection**: Create/Edit Campaign
   - Messenger dropdown shows: "email", "email-primary", "email-secondary", etc.
   - Select specific messenger to route through that SMTP server

### Backend Implementation
- **SMTP Initialization** (`cmd/init.go:613-661`): Creates messenger per named server
- **Bounce Manager** (`internal/bounce/bounce.go`): Supports multiple mailboxes
- **Settings Validation** (`cmd/settings.go:98-182`): UUID generation, password handling
- **Data Models** (`models/settings.go`):
  - `SMTP[].BounceMailboxUUID` links SMTP to bounce mailbox
  - `BounceBoxes[].Name` identifies bounce mailboxes

### Frontend Implementation
- **SMTP UI** (`frontend/src/views/settings/smtp.vue`): Bounce mailbox selector per SMTP
- **Bounce UI** (`frontend/src/views/settings/bounces.vue`): Add/remove/configure multiple mailboxes
- Each mailbox and SMTP server can be enabled/disabled independently

## Common Gotchas

- Frontend changes require rebuild (`make build-frontend`) before they appear in the binary
- Email-builder must be built before main frontend in dev mode
- Database migrations are version-specific and run during `--upgrade`
- The binary expects PostgreSQL to be running and configured
- Static assets are loaded from embedded FS by default, or from disk in dev mode (via `frontendDir` ldflags)
- SMTP server names are automatically prefixed with `email-` and sanitized (alphanumeric + hyphens only)
- Bounce mailbox scan intervals must be at least 1 minute
- Each bounce mailbox runs in a separate goroutine, scanning at its configured interval

## Queue-Based Email Delivery System

A sophisticated queue-based email delivery system has been implemented to support daily sending limits, time windows, and automatic server selection across multiple SMTP servers.

### Architecture Components

**internal/queue/** - Queue processing system
- `models.go` - Queue data structures (EmailQueueItem, ServerCapacity, DeliveryEstimate)
- `processor.go` - Core queue processing logic with capacity management
- `calculator.go` - Delivery estimation and capacity calculation

### Database Schema

**email_queue** - Stores all queued emails
- Columns: id, campaign_id, subscriber_id, status, priority, scheduled_at, sent_at
- Statuses: queued, sending, sent, failed, cancelled
- Indexed on status, scheduled_at, campaign_id, assigned_smtp_server_uuid

**smtp_daily_usage** - Tracks daily email counts per SMTP server
- Columns: smtp_server_uuid, usage_date, emails_sent
- Unique constraint on (smtp_server_uuid, usage_date)

**smtp_rate_limit_state** - Tracks sliding window rate limits
- Columns: smtp_server_uuid, window_start, emails_in_window

**campaigns** - Extended with queue tracking
- New columns: use_queue, queued_at, queue_completed_at

### SMTP Server Configuration

Each SMTP server now supports:
- **from_email** - Default sender address (e.g., adam@mail2.bobbyseamoss.com)
- **daily_limit** - Maximum emails per day (0 = unlimited)

### Performance Settings

**Time Window Configuration**:
- **app.send_time_start** - Start time for sending (24h format, e.g., "08:00")
- **app.send_time_end** - End time for sending (24h format, e.g., "20:00")
- Empty values = 24/7 sending

### Campaign Integration

**Automatic Messenger**:
- Select "automatic (queue-based)" in campaign messenger dropdown
- Campaign emails are queued to database instead of sent immediately
- Queue processor distributes emails across SMTP servers respecting capacity

**Queue Lifecycle**:
1. Campaign set to "running" with messenger="automatic"
2. All emails added to email_queue table
3. Campaign marked with use_queue=true, queued_at=NOW()
4. Queue processor polls for emails within time window
5. Processor selects server with most remaining capacity
6. Emails sent respecting daily limits and sliding window
7. Usage counters updated after each send

### API Endpoints

**GET /api/queue/stats** - Queue statistics
- Returns: total_queued, total_sending, total_sent, total_failed

**GET /api/queue/servers** - Server capacity information
- Returns: Array of {uuid, name, daily_limit, daily_used, daily_remaining, from_email}

### SQL Queries

**queue-campaign-emails** - Queue all emails for a campaign
**get-queued-email-count** - Count queued emails for campaign
**get-queue-stats** - Get queue statistics
**cancel-campaign-queue** - Cancel queued emails for campaign
**update-campaign-as-queued** - Mark campaign as using queue

### Frontend Components

**SMTP Settings** (`frontend/src/views/settings/smtp.vue`):
- From Email field per SMTP server
- Daily Limit field per SMTP server (number input, min 0)

**Performance Settings** (`frontend/src/views/settings/performance.vue`):
- Send Start Time field (HH:MM format)
- Send End Time field (HH:MM format)

**Campaign Form** (`frontend/src/views/Campaign.vue`):
- "automatic (queue-based)" option in messenger dropdown

### Queue Processor Features

**Capacity Management**:
- Tracks daily usage per SMTP server
- Respects daily limits (stops when limit reached)
- Respects sliding window rate limits (existing feature)
- Selects server with most remaining capacity

**Time Window Enforcement**:
- Only processes emails within configured time window
- Calculates sending hours per day for estimates
- Aligns scheduled times to window boundaries

**Delivery Estimation**:
- Calculates campaign completion time
- Distributes emails proportionally across servers
- Provides daily breakdown of sending schedule
- Indicates if campaign fits within single day

### Integration Points

**Campaign Status Update** (`internal/core/campaigns.go:UpdateCampaignStatus`):
- Detects messenger="automatic" when status set to "running"
- Calls QueueCampaignEmails() to queue all emails
- Logs number of emails queued

**Campaign Cancellation** (`cmd/campaigns.go:UpdateCampaignStatus`):
- Calls CancelCampaignQueue() when campaign paused/cancelled
- Updates all queued emails to status="cancelled"

### Migration

**v6.0.0 Migration** (`internal/migrations/v6.0.0.go`):
- Creates email_queue, smtp_daily_usage, smtp_rate_limit_state tables
- Adds queue tracking columns to campaigns table
- Registered in cmd/upgrade.go migList

### Common Use Cases

**Multi-Day Campaign**:
1. Configure SMTP servers with daily_limit=1000 each
2. Set time window to 08:00-20:00
3. Create campaign with 50,000 recipients
4. Select "automatic" messenger
5. System queues and distributes over multiple days

**Capacity Monitoring**:
- GET /api/queue/servers to see real-time capacity
- Shows daily_used, daily_remaining for each server
- Resets daily at midnight

**Queue Inspection**:
- GET /api/queue/stats for overall queue status
- Check email_queue table directly for detailed inspection

