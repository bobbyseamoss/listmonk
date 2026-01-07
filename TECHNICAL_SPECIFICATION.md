# Comprehensive Technical and Functional Specification for listmonk

**Version**: Based on current codebase state (2025)
**Last Updated**: 2025-11-13
**Document Type**: Technical Specification

---

## Executive Summary

listmonk is a standalone, self-hosted newsletter and mailing list manager built with:
- **Backend**: Go (102 .go files, ~10,130 total lines)
- **Frontend**: Vue 2.7 (60 files including .vue and .js)
- **Database**: PostgreSQL with sophisticated schema and materialized views
- **Architecture**: Microservices-style with separated concerns
- **Distribution**: Single binary with embedded static assets

---

## Table of Contents

1. [Frontend Specifications](#1-frontend-specifications)
2. [Backend Specifications](#2-backend-specifications)
3. [Database Specifications](#3-database-specifications)
4. [Advanced Features](#4-advanced-features)
5. [Deployment & Operations](#5-deployment--operations)
6. [Common Use Cases](#6-common-use-cases)
7. [Performance Considerations](#7-performance-considerations)
8. [Extensibility](#8-extensibility)
9. [Gotchas & Best Practices](#9-gotchas--best-practices)
10. [Conclusion](#10-conclusion)

---

## 1. FRONTEND SPECIFICATIONS

### 1.1 Architecture Overview

**Framework & Tooling**
- Vue 2.7 (latest Vue 2 version)
- Vue Router 3 (history mode)
- Vuex 3 (state management)
- Buefy (Bulma + Vue components)
- Build System: Vite

**Project Structure**
```
frontend/src/
├── main.js          # Application entry point
├── App.vue          # Root component
├── router/          # Route definitions
├── store/           # Vuex state management
├── api/             # HTTP client & API endpoints
├── views/           # Page components (45+ files)
├── components/      # Reusable UI components (11 files)
├── utils.js         # Helper functions
└── constants.js     # Application constants
```

**Key Dependencies**
- axios: HTTP client
- TinyMCE: Rich text editor
- CodeMirror 6: Code editor for templates
- Chart.js: Analytics visualization
- qs: Query string handling

### 1.2 Routing & Navigation

**Route Structure** (from router/index.js)

| Route | Component | Purpose |
|-------|-----------|---------|
| `/` | Dashboard | Main dashboard with stats |
| `/lists` | Lists | Mailing list management |
| `/lists/forms` | Forms | Public subscription forms |
| `/subscribers` | Subscribers | Subscriber management |
| `/subscribers/import` | Import | Bulk subscriber import |
| `/subscribers/bounces` | Bounces | Bounce management |
| `/campaigns` | Campaigns | Campaign list |
| `/campaigns/:id` | Campaign | Campaign editor |
| `/campaigns/media` | Media | Media library |
| `/campaigns/templates` | Templates | Template management |
| `/campaigns/analytics` | CampaignAnalytics | Campaign analytics |
| `/campaigns/queue` | Queue | Queue management (new) |
| `/settings` | Settings | Application settings |
| `/settings/logs` | Logs | System logs |
| `/settings/webhook-logs` | WebhookLogs | Webhook logs |
| `/users` | Users | User management |
| `/users/roles/users` | Roles | User role management |
| `/users/roles/lists` | Roles | List role management |
| `/settings/maintenance` | Maintenance | Database maintenance |
| `/user/profile` | UserProfile | User profile settings |

**Navigation Features**
- History mode (no hash routes)
- Scroll to hash support
- Meta tags for grouping menu items
- Breadcrumb support via route metadata

### 1.3 State Management (Vuex)

**Store Structure**
```javascript
{
  // Data models
  lists: [],
  subscribers: [],
  campaigns: [],
  media: [],
  templates: [],
  users: [],
  userRoles: [],
  listRoles: [],
  settings: {},
  serverConfig: {},
  logs: [],
  profile: {},

  // Loading states (per model)
  loading: {
    lists: false,
    campaigns: false,
    // ... etc
  }
}
```

**Key Patterns**
- Model-based state organization
- Loading state tracking per model
- Response data stored directly from API
- Mutations: `setModelResponse`, `setLoading`
- Getters for each model type

### 1.4 API Client Architecture

**HTTP Client Configuration** (api/index.js)
- Base URL: configurable via environment
- Axios-based with interceptors
- Request interceptor: Sets loading states
- Response interceptor:
  - Clears loading states
  - Transforms keys to camelCase (configurable)
  - Stores responses in Vuex
  - Handles errors with toast notifications
  - Blob response support for downloads

**API Endpoint Categories** (156+ endpoints)

**Lists**
- `getLists()`, `queryLists()`, `getList(id)`
- `createList()`, `updateList()`, `deleteList()`

**Subscribers**
- `getSubscribers()`, `getSubscriber(id)`
- `createSubscriber()`, `updateSubscriber()`, `deleteSubscriber()`
- `sendSubscriberOptin()`, `addSubscribersToLists()`
- `blocklistSubscribers()`, `deleteSubscribersByQuery()`
- `getSubscriberBounces()`, `deleteSubscriberBounces()`

**Campaigns**
- `getCampaigns()`, `getCampaign(id)`, `createCampaign()`
- `updateCampaign()`, `changeCampaignStatus()`, `deleteCampaign()`
- `testCampaign()`, `convertCampaignContent()`
- `getCampaignViewCounts()`, `getCampaignClickCounts()`
- `getCampaignBounceCounts()`, `getCampaignLinkCounts()`
- `getCampaignUnsubscribers()`
- `getCampaignAzureAnalytics()` (Azure Event Grid integration)
- `getCampaignPurchaseStats()` (Shopify integration)
- `getCampaignsPerformanceSummary()`

**Queue Management** (New Feature)
- `getQueueItems()`, `getQueueStats()`
- `getSMTPServerCapacity()`
- `cancelQueueItem()`, `retryQueueItem()`
- `clearAllQueuedEmails()`, `sendAllQueuedEmails()`
- `toggleQueuePause()`

**Templates**
- `getTemplates()`, `getTemplate(id)`
- `createTemplate()`, `updateTemplate()`, `deleteTemplate()`
- `makeTemplateDefault()`

**Media**
- `getMedia()`, `uploadMedia()`, `deleteMedia()`

**Settings**
- `getServerConfig()`, `getSettings()`, `updateSettings()`
- `testSMTP()`, `getLogs()`

**Users & Roles**
- `getUsers()`, `getUser(id)`, `createUser()`, `updateUser()`, `deleteUser()`
- `getUserProfile()`, `updateUserProfile()`
- `getUserRoles()`, `getListRoles()`
- `createUserRole()`, `updateUserRole()`, `deleteRole()`

**Bounces**
- `getBounces()`, `deleteBounce()`, `deleteBounces()`
- `blocklistBouncedSubscribers()`

**Webhook Logs**
- `getWebhookLogs()`, `deleteWebhookLogs()`, `exportWebhookLogs()`

**Import**
- `importSubscribers()`, `getImportStatus()`, `getImportLogs()`, `stopImport()`

**Maintenance**
- `deleteGCCampaignAnalytics()`, `deleteGCSubscribers()`, `deleteGCSubscriptions()`

**Dashboard**
- `getDashboardCounts()`, `getDashboardCharts()`

**Misc**
- `getHealth()`, `reloadApp()`, `logout()`

### 1.5 Key Components

**Page Components** (in views/)

1. **Dashboard.vue** - Main dashboard with statistics and charts
2. **Campaigns.vue** - Campaign list with filtering and stats
3. **Campaign.vue** - Campaign editor (create/edit)
4. **CampaignAnalytics.vue** - Campaign performance analytics
5. **Queue.vue** - Email queue management interface
6. **Subscribers.vue** - Subscriber list and management
7. **SubscriberForm.vue** - Subscriber create/edit form
8. **Lists.vue** - Mailing list management
9. **ListForm.vue** - List create/edit form
10. **Templates.vue** - Template library
11. **TemplateForm.vue** - Template editor
12. **Media.vue** - Media library
13. **Import.vue** - Bulk import interface
14. **Bounces.vue** - Bounce management
15. **Settings.vue** - Settings container
16. **Users.vue** - User management
17. **Roles.vue** - Role management
18. **Logs.vue** - System logs viewer
19. **WebhookLogs.vue** - Webhook logs viewer
20. **Maintenance.vue** - Database maintenance
21. **About.vue** - System information

**Settings Sub-Pages** (in views/settings/)
- `general.vue` - General app settings
- `smtp.vue` - SMTP server configuration (multi-server support)
- `bounces.vue` - Bounce mailbox configuration
- `performance.vue` - Performance & queue settings
- `messengers.vue` - Alternative messenger configuration
- `media.vue` - Media storage settings
- `appearance.vue` - UI customization
- `privacy.vue` - Privacy settings
- `security.vue` - Security settings (OIDC, CAPTCHA, CORS)
- `campaigns.vue` - Campaign-specific settings
- `shopify.vue` - Shopify integration settings

**Reusable Components** (in components/)
1. **Editor.vue** - Campaign/template editor wrapper
2. **RichtextEditor.vue** - TinyMCE wrapper
3. **VisualEditor.vue** - Visual email builder
4. **CodeEditor.vue** - CodeMirror wrapper for HTML/template editing
5. **ListSelector.vue** - List selection component
6. **CopyText.vue** - Copy-to-clipboard component
7. **Chart.vue** - Generic chart component
8. **BarChart.vue** - Bar chart component
9. **CampaignPreview.vue** - Campaign preview modal
10. **CampaignAzureAnalytics.vue** - Azure analytics component
11. **Navigation.vue** - Main navigation component
12. **LogView.vue** - Log viewer component
13. **EmptyPlaceholder.vue** - Empty state placeholder

### 1.6 Features by Module

**Campaign Management UI**
- List/grid view with status indicators
- Filtering by status, tags, query
- Real-time statistics (sent, views, clicks, bounces)
- Visual/HTML/Markdown/Plain editor support
- Template selection
- List targeting
- Scheduling
- Messenger selection (email, email-{name}, automatic)
- Media attachments
- Preview functionality
- Test email sending
- Analytics dashboard per campaign
- Unsubscriber tracking
- Azure Event Grid analytics
- Shopify purchase attribution

**Subscriber Management UI**
- Advanced search and filtering
- Query builder for segmentation
- Bulk operations (add to lists, blocklist, delete)
- Import from CSV
- Export data (GDPR compliance)
- Subscription status management
- Bounce history per subscriber
- Custom attributes (JSONB)
- List membership management

**List Management UI**
- Public/private list types
- Single/double opt-in
- Subscriber counts by status
- Bulk subscriber operations
- Public subscription forms
- List segmentation

**Template Editor**
- Campaign templates
- Transaction (tx) templates
- Visual editor integration
- HTML/rich text editing
- Preview functionality
- Default template selection
- Template functions (TrackLink, UnsubscribeURL, etc.)

**Analytics/Reporting UI**
- Dashboard with aggregate stats
- Campaign-level analytics:
  - Views (unique and total)
  - Clicks (unique and total) with link breakdown
  - Bounces by type
  - Azure delivery events
  - Unsubscribers
  - Purchase attribution (Shopify)
- Time-series charts
- Date range selection
- Performance summary across campaigns

**Queue Management UI**
- Queue item listing with filtering
- Server capacity monitoring
- Pause/resume queue
- Clear queue
- Send all queued emails
- Cancel/retry individual items
- Real-time stats display

**Settings UI**
- **SMTP Servers**:
  - Multi-server support (30+)
  - Named servers for dedicated messengers
  - Bounce mailbox linking per server
  - Daily limit configuration
  - From email per server
  - Connection pooling settings
  - TLS configuration
  - Test email functionality

- **Bounce Mailboxes**:
  - Multiple POP3 mailboxes (30+)
  - UUID identification
  - Scan interval configuration
  - Per-mailbox enable/disable

- **Performance**:
  - Time window configuration (send start/end times)
  - Batch size
  - Concurrency
  - Message rate
  - Sliding window rate limiting
  - Queue poll interval

- **Media Storage**: Filesystem or S3
- **Appearance**: Custom CSS/JS for admin and public pages
- **Privacy**: Individual tracking, exportable fields, domain blocklists
- **Security**: OIDC, CAPTCHA (hCaptcha, Altcha), CORS
- **Messengers**: Postback HTTP messengers
- **Shopify**: Webhook integration, attribution window

---

## 2. BACKEND SPECIFICATIONS

### 2.1 Architecture Overview

**Language & Framework**
- Go 1.x
- Echo v4 HTTP framework
- 102 Go files, approximately 10,130 lines of code

**Package Structure**
```
cmd/                    # HTTP handlers, main entry point
├── main.go            # Application initialization
├── init.go            # Component initialization
├── campaigns.go       # Campaign HTTP handlers
├── subscribers.go     # Subscriber HTTP handlers
├── lists.go           # List HTTP handlers
├── templates.go       # Template HTTP handlers
├── users.go           # User HTTP handlers
├── settings.go        # Settings HTTP handlers
├── bounce.go          # Bounce HTTP handlers
├── queue.go           # Queue HTTP handlers
├── media.go           # Media HTTP handlers
├── auth.go            # Authentication handlers
├── admin.go           # Admin operations
├── public.go          # Public pages (subscription, archive)
├── handlers.go        # Generic handler utilities
├── utils.go           # Utility functions
├── archive.go         # Campaign archive handlers
├── webhook_logs.go    # Webhook log handlers
├── azure_analytics.go # Azure analytics handlers
├── shopify.go         # Shopify integration handlers
├── roles.go           # Role management handlers
├── import.go          # Import handlers
├── maintenance.go     # Maintenance handlers
├── events.go          # SSE events
├── tx.go              # Transactional emails
├── install.go         # Database installation
├── upgrade.go         # Database migrations
└── ...

internal/               # Internal packages
├── core/              # Business logic (CRUD operations)
│   ├── core.go
│   ├── campaigns.go
│   ├── subscribers.go
│   ├── lists.go
│   ├── bounces.go
│   ├── templates.go
│   ├── users.go
│   ├── roles.go
│   ├── settings.go
│   ├── media.go
│   ├── dashboard.go
│   └── subscriptions.go
│
├── manager/           # Campaign execution engine
│   ├── manager.go     # Campaign manager
│   ├── pipe.go        # Campaign processing pipeline
│   └── message.go     # Message construction
│
├── messenger/         # Message delivery backends
│   ├── email/         # SMTP email delivery
│   ├── postback/      # HTTP postback
│   └── automatic/     # Queue-based delivery
│
├── queue/             # Queue processing system
│   ├── processor.go   # Queue processor
│   ├── calculator.go  # Capacity calculation
│   ├── scheduler.go   # Auto-pause scheduler
│   └── models.go      # Queue data structures
│
├── bounce/            # Bounce handling
│   ├── bounce.go      # Bounce manager
│   ├── mailbox/       # POP3 mailbox scanning
│   └── webhooks/      # Webhook handlers (SES, SendGrid, Postmark, Azure, Shopify)
│
├── auth/              # Authentication
│   ├── auth.go        # Auth manager
│   └── models.go      # Auth models
│
├── media/             # Media storage
│   ├── media.go
│   └── providers/
│       ├── filesystem/
│       └── s3/
│
├── i18n/              # Internationalization
├── captcha/           # CAPTCHA verification
├── subimporter/       # Bulk import
├── notifs/            # Admin notifications
├── buflog/            # Buffered logging
├── events/            # Server-sent events
└── migrations/        # Database migrations

models/                 # Data models and queries
├── models.go          # Core data structures
├── queries.go         # SQL query loader
└── settings.go        # Settings structures
```

**Key Dependencies**
- `sqlx`: Database toolkit with named queries
- `koanf`: Configuration management
- `goyesql`: SQL query file loader
- `stuffbin`: Asset embedding
- `echo`: HTTP framework
- `sprig`: Template functions
- `goldmark`: Markdown rendering

### 2.2 API Endpoints

**Complete HTTP Route Listing**

**Public Routes** (no authentication)
- `GET /` - Public home page
- `GET /subscription/:campUUID/:subUUID` - Subscription management page
- `POST /subscription/:campUUID/:subUUID` - Update subscription preferences
- `POST /subscription/optin/:subUUID` - Confirm double opt-in
- `GET /subscription/form/:listUUID` - Public subscription form
- `POST /subscription/form` - Submit subscription form
- `GET /link/:linkUUID/:campUUID/:subUUID` - Track link click and redirect
- `GET /campaign/:campUUID/:subUUID` - View campaign message
- `GET /campaign/:campUUID/:subUUID/px.png` - Track campaign view (pixel)
- `GET /public/lists` - Get public lists
- `GET /archive` - Campaign archive listing
- `GET /archive/:slug` - View archived campaign
- `GET /archive.xml` - RSS feed of campaigns
- `GET /api/health` - Health check

**Admin Routes** (authentication required)

**Campaigns**
- `GET /api/campaigns` - List campaigns
- `GET /api/campaigns/:id` - Get campaign
- `POST /api/campaigns` - Create campaign
- `PUT /api/campaigns/:id` - Update campaign
- `DELETE /api/campaigns/:id` - Delete campaign
- `PUT /api/campaigns/:id/status` - Change campaign status
- `PUT /api/campaigns/:id/archive` - Update archive settings
- `POST /api/campaigns/:id/test` - Send test emails
- `POST /api/campaigns/:id/content` - Convert campaign content format
- `GET /api/campaigns/:id/preview` - Preview campaign
- `POST /api/campaigns/:id/preview` - Preview with custom body
- `GET /api/campaigns/:id/archive/preview` - Preview archive
- `GET /api/campaigns/running/stats` - Get running campaign stats
- `GET /api/campaigns/analytics/views` - Get view counts
- `GET /api/campaigns/analytics/clicks` - Get click counts
- `GET /api/campaigns/analytics/bounces` - Get bounce counts
- `GET /api/campaigns/analytics/links` - Get link click counts
- `GET /api/campaigns/analytics/azure-delivery` - Get Azure delivery stats
- `GET /api/campaigns/:id/unsubscribers` - Get unsubscribers
- `GET /api/campaigns/:id/azure-analytics` - Get Azure analytics
- `GET /api/campaigns/:id/azure-delivery-events` - Get Azure delivery events
- `GET /api/campaigns/:id/azure-engagement-events` - Get Azure engagement events
- `GET /api/campaigns/:id/purchases/stats` - Get Shopify purchase stats
- `GET /api/campaigns/performance/summary` - Get aggregate performance
- `POST /api/campaigns/:id/remove-sent-today` - Remove sent subscribers from lists

**Queue**
- `GET /api/queue/items` - List queue items
- `GET /api/queue/stats` - Get queue statistics
- `GET /api/queue/servers` - Get SMTP server capacities
- `PUT /api/queue/:id/cancel` - Cancel queue item
- `PUT /api/queue/:id/retry` - Retry queue item
- `POST /api/queue/clear` - Clear all queued emails
- `PUT /api/queue/pause` - Pause/resume queue
- `POST /api/queue/send-all` - Send all queued emails immediately

**Subscribers**
- `GET /api/subscribers` - List subscribers
- `GET /api/subscribers/:id` - Get subscriber
- `POST /api/subscribers` - Create subscriber
- `PUT /api/subscribers/:id` - Update subscriber
- `DELETE /api/subscribers/:id` - Delete subscriber
- `DELETE /api/subscribers` - Bulk delete subscribers
- `POST /api/subscribers/:id/optin` - Send opt-in confirmation
- `PUT /api/subscribers/lists` - Add subscribers to lists
- `PUT /api/subscribers/query/lists` - Add subscribers to lists by query
- `PUT /api/subscribers/blocklist` - Blocklist subscribers
- `PUT /api/subscribers/query/blocklist` - Blocklist subscribers by query
- `POST /api/subscribers/query/delete` - Delete subscribers by query
- `GET /api/subscribers/:id/bounces` - Get subscriber bounces
- `DELETE /api/subscribers/:id/bounces` - Delete subscriber bounces
- `GET /api/subscribers/:id/azure-delivery-events` - Get Azure delivery events
- `GET /api/subscribers/:id/azure-engagement-events` - Get Azure engagement events

**Lists**
- `GET /api/lists` - List all lists
- `GET /api/lists/:id` - Get list
- `POST /api/lists` - Create list
- `PUT /api/lists/:id` - Update list
- `DELETE /api/lists/:id` - Delete list

**Templates**
- `GET /api/templates` - List templates
- `GET /api/templates/:id` - Get template
- `POST /api/templates` - Create template
- `PUT /api/templates/:id` - Update template
- `PUT /api/templates/:id/default` - Set as default template
- `DELETE /api/templates/:id` - Delete template

**Media**
- `GET /api/media` - List media
- `POST /api/media` - Upload media
- `DELETE /api/media/:id` - Delete media

**Bounces**
- `GET /api/bounces` - List bounces
- `DELETE /api/bounces/:id` - Delete bounce
- `DELETE /api/bounces` - Bulk delete bounces
- `PUT /api/bounces/blocklist` - Blocklist bounced subscribers

**Webhook Logs**
- `GET /api/webhook-logs` - List webhook logs
- `DELETE /api/webhook-logs` - Delete webhook logs
- `GET /api/webhook-logs/export` - Export webhook logs

**Webhooks** (external services)
- `POST /webhooks/bounce` - Generic bounce webhook
- `POST /webhooks/bounce/ses` - Amazon SES bounce webhook
- `POST /webhooks/bounce/sendgrid` - SendGrid bounce webhook
- `POST /webhooks/bounce/postmark` - Postmark bounce webhook
- `POST /webhooks/bounce/forwardemail` - ForwardEmail bounce webhook
- `POST /webhooks/azure/eventgrid` - Azure Event Grid webhook
- `POST /webhooks/shopify/orders` - Shopify order webhook

**Settings**
- `GET /api/config` - Get server config
- `GET /api/settings` - Get settings
- `PUT /api/settings` - Update settings
- `POST /api/settings/smtp/test` - Test SMTP connection
- `GET /api/logs` - Get system logs

**Users & Authentication**
- `GET /api/users` - List users
- `GET /api/users/:id` - Get user
- `POST /api/users` - Create user
- `PUT /api/users/:id` - Update user
- `DELETE /api/users/:id` - Delete user
- `GET /api/profile` - Get current user profile
- `PUT /api/profile` - Update current user profile
- `POST /api/login` - Login
- `POST /api/logout` - Logout
- `GET /api/auth/oidc` - OIDC authentication callback

**Roles**
- `GET /api/roles/users` - List user roles
- `GET /api/roles/lists` - List list roles
- `POST /api/roles/users` - Create user role
- `POST /api/roles/lists` - Create list role
- `PUT /api/roles/users/:id` - Update user role
- `PUT /api/roles/lists/:id` - Update list role
- `DELETE /api/roles/:id` - Delete role

**Import**
- `GET /api/import/subscribers` - Get import status
- `POST /api/import/subscribers` - Start import
- `DELETE /api/import/subscribers` - Stop import
- `GET /api/import/subscribers/logs` - Get import logs

**Maintenance**
- `DELETE /api/maintenance/analytics/:type` - Delete old analytics
- `DELETE /api/maintenance/subscribers/:type` - Delete subscribers (blocklisted/orphans)
- `DELETE /api/maintenance/subscriptions/unconfirmed` - Delete unconfirmed subscriptions

**Dashboard**
- `GET /api/dashboard/counts` - Get dashboard counts
- `GET /api/dashboard/charts` - Get dashboard charts

**Admin**
- `POST /api/admin/reload` - Reload application

**Transactional Emails**
- `POST /api/tx` - Send transactional email

### 2.3 Business Logic Layers

**cmd/ Handlers (HTTP Layer)**
- Request validation and parsing
- Permission checking
- Call core business logic
- Format HTTP responses
- Error handling with proper status codes
- Pagination support

**internal/core/ (Business Logic)**
- Pure business logic, no HTTP dependencies
- Returns `echo.HTTPError` for direct use in handlers
- Transaction management
- Data validation
- CRUD operations for all entities
- Database query execution
- Handles complex operations (bulk updates, queries)
- Key methods:
  - Campaign CRUD and status management
  - Subscriber CRUD and bulk operations
  - List management
  - Template management
  - Bounce recording and management
  - User and role management
  - Settings management
  - Dashboard statistics

**internal/manager/ (Campaign Execution)**
- Campaign processing pipeline
- Subscriber fetching in batches
- Message construction and rendering
- Template compilation
- Campaign scheduling
- Rate limiting (per-message and sliding window)
- Error tracking and campaign stopping
- Concurrency control
- Message queuing
- Link and view tracking
- Key features:
  - Batch processing (configurable batch size)
  - Concurrent workers (configurable concurrency)
  - Message rate limiting
  - Sliding window rate limiting (per-messenger)
  - Campaign statistics (send rate)
  - Template function injection
  - Campaign lifecycle management

**internal/messenger/ (Delivery Backends)**
- **email/** - SMTP email delivery
  - Connection pooling (per server)
  - Multiple SMTP server support (30+)
  - Named servers for dedicated messengers
  - Bounce mailbox linking
  - Daily limit tracking per server
  - From email per server
  - TLS configuration
  - Authentication (PLAIN, CRAM-MD5, LOGIN)
  - Idle/wait timeouts
  - Max retries
  - Testing mode (simulation)

- **postback/** - HTTP webhook delivery
  - POST requests with campaign data
  - Custom headers
  - Timeout configuration

- **automatic/** - Queue-based delivery
  - Enqueues to email_queue table
  - No direct sending

**internal/queue/ (Queue Processing)**
- Queue-based email delivery system
- Server capacity management
- Daily limit enforcement per SMTP server
- Sliding window rate limiting
- Time window enforcement (send start/end times)
- Auto-pause/resume scheduling
- Campaign statistics syncing
- Smart Sending integration
- Account-wide rate limiting
- Batch processing with concurrency control
- Server selection based on capacity
- Campaign completion detection
- Key features:
  - Poll interval: 5 seconds (responsive)
  - Configurable batch size (default: 100)
  - Time zone support
  - Automatic campaign status updates
  - Graceful shutdown

**internal/bounce/ (Bounce Handling)**
- Multiple bounce mailbox support (30+ via POP3)
- Webhook handlers for:
  - Amazon SES
  - SendGrid
  - Postmark
  - ForwardEmail
  - Azure Event Grid
  - Shopify (via Azure Event Grid)
- Bounce type classification (hard, soft, complaint)
- Configurable actions per bounce type
- Subscriber status management (blocklist, delete)
- Bounce count threshold enforcement

### 2.4 Key Features Implementation

**Campaign Management**
- **Creation**:
  - Template selection (campaign or campaign_visual)
  - List targeting (multiple lists)
  - Content types: richtext, HTML, markdown, plain, visual
  - Scheduling support
  - Archive settings (slug, template, metadata)
  - Media attachments
  - Custom headers
  - Tags

- **Scheduling**:
  - Immediate (status: running)
  - Scheduled (status: scheduled, send_at timestamp)
  - Queue-based (messenger: automatic)

- **Sending**:
  - Traditional: Campaign manager with worker pool
  - Queue-based: Automatic messenger + queue processor
  - Batch fetching from database
  - Concurrent workers
  - Rate limiting (message rate + sliding window)
  - Error tracking (max_send_errors threshold)
  - Progress tracking (sent, to_send counters)
  - Link and view tracking

- **Template Compilation**:
  - Base template + campaign body
  - Template functions (TrackLink, UnsubscribeURL, etc.)
  - Sprig functions (date formatting, string manipulation, etc.)
  - Subject line templating
  - Alt body (plain text) templating
  - Markdown to HTML conversion

**Queue-Based Email Delivery**
- **Architecture**:
  - email_queue table stores all queued emails
  - Queue processor polls for ready emails (every 5 seconds)
  - SMTP server selection based on capacity
  - Daily limit tracking per server
  - Sliding window rate limiting
  - Time window enforcement (send only during configured hours)

- **Flow**:
  1. Campaign status set to "running" with messenger="automatic"
  2. All campaign emails inserted into email_queue (status='queued')
  3. Campaign marked with use_queue=true, queued_at=NOW()
  4. Queue processor polls for queued emails with scheduled_at <= NOW()
  5. Processor selects SMTP server with most remaining capacity
  6. Email sent via specific messenger (email-{name})
  7. Status updated to 'sent', usage counters incremented
  8. Campaign marked as 'finished' when all emails processed

- **Capacity Management**:
  - Daily limit per SMTP server
  - Daily usage tracking in smtp_daily_usage table
  - Sliding window tracking in smtp_rate_limit_state table
  - Server selection: most remaining capacity
  - Automatic skip when all servers at capacity

- **Time Window**:
  - Configurable send start/end times (HH:MM format)
  - Timezone support (configurable per installation)
  - Auto-pause campaigns outside window
  - Auto-resume campaigns inside window
  - Runs every minute to check window status

- **Campaign Stats Sync**:
  - Runs every 5 minutes
  - Syncs campaign.sent from email_queue counts
  - Ensures accurate sent counts for running campaigns

- **Smart Sending**:
  - Prevents sending to same subscriber multiple times in configurable window
  - subscriber_last_send table tracks last send time
  - Configurable minimum hours between sends
  - Applied when queuing emails (not at fetch time for performance)

**Multi-SMTP Server Support**
- **Named Servers**:
  - Each SMTP server can have a "name" field
  - Creates dedicated messenger: email-{name}
  - Allows campaign to select specific server
  - Unnamed servers grouped into default "email" messenger

- **Configuration**:
  - Array of SMTP configs in settings
  - Each with: host, port, auth, credentials, name, from_email, daily_limit
  - Bounce mailbox linking via bounce_mailbox_uuid
  - Connection pool per server (max_conns, idle_timeout, wait_timeout)

- **Bounce Mailbox Linking**:
  - Each SMTP server can link to specific POP3 mailbox
  - Mailboxes identified by UUID
  - Multiple mailboxes supported (30+)
  - Each mailbox scans at configured interval
  - Runs in separate goroutine

**Analytics Tracking**
- **View Tracking**:
  - Transparent 1x1 pixel (px.png)
  - campaign_views table
  - Unique or total tracking (configurable)
  - Timestamp recording

- **Click Tracking**:
  - TrackLink template function
  - links table (URL → UUID mapping)
  - link_clicks table
  - Redirect after recording click
  - Top links by click count

- **Bounce Tracking**:
  - bounces table
  - Type: hard, soft, complaint
  - Source: webhook, mailbox
  - Metadata storage (JSONB)
  - Subscriber linking

- **Azure Event Grid Integration**:
  - Delivery events: Delivered, Bounced, Suppressed, Failed, etc.
  - Engagement events: view, click
  - azure_delivery_events table
  - azure_engagement_events table
  - Webhook handler at /webhooks/azure/eventgrid
  - Message ID tracking for correlation

- **Shopify Purchase Attribution**:
  - Order webhook integration
  - Attribution via recent link click or email open
  - Configurable attribution window (default: 7 days)
  - purchase_attributions table
  - Campaign-level revenue and order tracking
  - Aggregate performance metrics

**Bounce Handling**
- **Mailbox Scanning**:
  - POP3 protocol
  - Multiple mailboxes (30+)
  - Configurable scan interval (minimum 1 minute)
  - Each mailbox in separate goroutine
  - Email parsing for bounce detection
  - Return-Path matching

- **Webhooks**:
  - SES: SNS notifications
  - SendGrid: Event webhook
  - Postmark: Inbound bounce API
  - ForwardEmail: Webhook with key auth
  - Azure Event Grid: Delivery status events

- **Actions**:
  - Configurable per bounce type (soft, hard, complaint)
  - Count threshold
  - Actions: none, unsubscribe, blocklist, delete
  - Subscriber status updates
  - Subscription status updates (all lists to unsubscribed)

**Subscriber Management**
- **CRUD Operations**:
  - Create with list subscriptions
  - Update with list management
  - Delete (with cascade to subscriptions)
  - Blocklist (status + unsubscribe all lists)

- **Bulk Operations**:
  - Add to lists
  - Remove from lists
  - Blocklist
  - Delete
  - Query-based operations (SQL WHERE clause)

- **Attributes**:
  - JSONB column for arbitrary data
  - Fully searchable
  - Used in template rendering

- **Import**:
  - CSV import with field mapping
  - Duplicate handling (skip, overwrite)
  - Blocklist mode
  - Background processing
  - Progress tracking
  - Email notifications on completion

**Template System**
- **Types**:
  - campaign: Regular campaign wrapper
  - campaign_visual: Visual editor output
  - tx: Transactional email templates

- **Compilation**:
  - Go html/template + text/template
  - Sprig functions (80+ utility functions)
  - Custom functions:
    - TrackLink: Link tracking with UUID
    - TrackView: View tracking pixel
    - UnsubscribeURL: Unsubscribe link
    - ManageURL: Preference management link
    - OptinURL: Opt-in confirmation link
    - MessageURL: Campaign view URL
    - ArchiveURL: Archive listing URL
    - RootURL: Site root URL

- **Rendering**:
  - Template + campaign body → final HTML
  - Subject line templating
  - Alt body (plain text) templating
  - Markdown to HTML conversion
  - Subscriber data injection

**Authentication (sessions, OIDC)**
- **Session Management**:
  - Database-backed sessions (sessions table)
  - Cookie-based (httponly, secure)
  - Session data stored as JSONB
  - User caching for API tokens

- **OIDC**:
  - Generic OIDC provider support
  - Auto-create users option
  - Default role assignment
  - Configurable redirect URL

- **API Authentication**:
  - Username/password Basic Auth
  - API tokens (user type: api)
  - Per-request permission checking

- **RBAC**:
  - User roles (permissions for campaigns, subscribers, etc.)
  - List roles (permissions per list)
  - Permission inheritance
  - Role hierarchy

### 2.5 Configuration

**Config Sources** (priority order)
1. Environment variables (LISTMONK_ prefix)
2. config.toml file(s)
3. Database settings table

**config.toml Structure**
```toml
[app]
address = "0.0.0.0:9000"
site_name = "Mailing list"
root_url = "http://localhost:9000"
from_email = "listmonk <noreply@listmonk.yoursite.com>"
logo_url = ""
favicon_url = ""
lang = "en"
batch_size = 1000
concurrency = 10
message_rate = 10
max_send_errors = 1000
message_sliding_window = false
message_sliding_window_duration = "1h"
message_sliding_window_rate = 10000
send_optin_confirmation = true
check_updates = true
notify_emails = []
enable_public_subscription_page = true
enable_public_archive = true
enable_public_archive_rss_content = true
cache_slow_queries = false
cache_slow_queries_interval = "0 3 * * *"
testing_mode = false
send_time_start = ""  # HH:MM format
send_time_end = ""    # HH:MM format
timezone = "America/New_York"
smart_sending_enabled = false
smart_sending_hours = 24
account_rate_limit_per_minute = 0
account_rate_limit_per_hour = 0
queue_paused = false

[privacy]
individual_tracking = false
unsubscribe_header = true
allow_blocklist = true
allow_export = true
allow_wipe = true
allow_preferences = true
exportable = ["profile", "subscriptions", "campaign_views", "link_clicks"]
domain_blocklist = []
domain_allowlist = []
record_optin_ip = false

[security]
cors_origins = []

[security.oidc]
enabled = false
provider_url = ""
provider_name = ""
client_id = ""
client_secret = ""
auto_create_users = false
default_user_role_id = 0
default_list_role_id = 0

[security.captcha.altcha]
enabled = false
complexity = 300000

[security.captcha.hcaptcha]
enabled = false
key = ""
secret = ""

[[smtp]]
enabled = true
name = ""  # Optional: creates email-{name} messenger
host = "smtp.yoursite.com"
port = 25
auth_protocol = "cram"  # none, cram, plain, login
username = "username"
password = "password"
hello_hostname = ""
max_conns = 10
idle_timeout = "15s"
wait_timeout = "5s"
max_msg_retries = 2
tls_type = "STARTTLS"  # none, STARTTLS, TLS
tls_skip_verify = false
email_headers = []
bounce_mailbox_uuid = ""  # Links to bounce mailbox
from_email = ""  # Per-server sender address
daily_limit = 0  # 0 = unlimited

[[messengers]]
enabled = false
name = "my-postback"
url = "https://mysite.com/webhooks/listmonk"
method = "POST"
username = ""
password = ""
max_conns = 10
timeout = "5s"
content_type = "application/json"

[upload]
provider = "filesystem"  # or "s3"
max_file_size = 5000  # KB
extensions = ["jpg", "jpeg", "png", "gif", "svg", "*"]

[upload.filesystem]
upload_path = "uploads"
upload_uri = "/uploads"

[upload.s3]
url = "https://ap-south-1.s3.amazonaws.com"
public_url = ""
aws_access_key_id = ""
aws_secret_access_key = ""
aws_default_region = "ap-south-1"
bucket = ""
bucket_domain = ""
bucket_path = "/"
bucket_type = "public"  # or "private"
expiry = "167h"

[bounce]
enabled = false
webhooks_enabled = false
ses_enabled = false
sendgrid_enabled = false
sendgrid_key = ""

[bounce.postmark]
enabled = false
username = ""
password = ""

[bounce.forwardemail]
enabled = false
key = ""

[bounce.azure]
enabled = false

[bounce.actions]
soft = {count = 2, action = "none"}
hard = {count = 1, action = "blocklist"}
complaint = {count = 1, action = "blocklist"}

[[bounce.mailboxes]]
enabled = false
uuid = ""  # Auto-generated
name = ""
type = "pop"
host = "pop.yoursite.com"
port = 995
auth_protocol = "userpass"
username = "username"
password = "password"
return_path = "bounce@listmonk.yoursite.com"
scan_interval = "15m"
tls_enabled = true
tls_skip_verify = false

[shopify]
enabled = false
webhook_secret = ""
attribution_window_days = 7

[appearance.admin]
custom_css = ""
custom_js = ""

[appearance.public]
custom_css = ""
custom_js = ""

[db]
host = "localhost"
port = 5432
user = "listmonk"
password = "listmonk"
database = "listmonk"
ssl_mode = "disable"
params = ""
max_open = 25
max_idle = 25
max_lifetime = "300s"
```

**Database Settings** (settings table)
- All app.* settings from config.toml
- SMTP server array
- Messenger array
- Bounce mailbox array
- Upload settings
- Privacy settings
- Security settings
- Appearance settings
- Editable via Settings UI

---

## 3. DATABASE SPECIFICATIONS

### 3.1 Schema Overview

**Database**: PostgreSQL
**Key Features**:
- JSONB columns for flexible data
- Materialized views for performance
- Full-text search (tsvector, tsquery)
- Array types (pq.StringArray)
- Enum types for status fields
- Extensive indexing
- Foreign key constraints with cascades

### 3.2 Enum Types

```sql
CREATE TYPE list_type AS ENUM ('public', 'private', 'temporary');
CREATE TYPE list_optin AS ENUM ('single', 'double');
CREATE TYPE subscriber_status AS ENUM ('enabled', 'disabled', 'blocklisted');
CREATE TYPE subscription_status AS ENUM ('unconfirmed', 'confirmed', 'unsubscribed');
CREATE TYPE campaign_status AS ENUM ('draft', 'running', 'scheduled', 'paused', 'cancelled', 'finished');
CREATE TYPE campaign_type AS ENUM ('regular', 'optin');
CREATE TYPE content_type AS ENUM ('richtext', 'html', 'plain', 'markdown', 'visual');
CREATE TYPE bounce_type AS ENUM ('soft', 'hard', 'complaint');
CREATE TYPE template_type AS ENUM ('campaign', 'campaign_visual', 'tx');
CREATE TYPE user_type AS ENUM ('user', 'api');
CREATE TYPE user_status AS ENUM ('enabled', 'disabled');
CREATE TYPE role_type AS ENUM ('user', 'list');
```

### 3.3 Key Tables

**subscribers** (Subscriber records)
```sql
id              SERIAL PRIMARY KEY
uuid            UUID NOT NULL UNIQUE
email           TEXT NOT NULL UNIQUE
name            TEXT NOT NULL
attribs         JSONB DEFAULT '{}'
status          subscriber_status DEFAULT 'enabled'
created_at      TIMESTAMP WITH TIME ZONE
updated_at      TIMESTAMP WITH TIME ZONE

Indexes:
- UNIQUE idx_subs_email ON LOWER(email)
- idx_subs_status ON (status)
- idx_subs_id_status ON (id, status)
- idx_subs_created_at, idx_subs_updated_at
```

**lists** (Mailing lists)
```sql
id              SERIAL PRIMARY KEY
uuid            UUID NOT NULL UNIQUE
name            TEXT NOT NULL
type            list_type NOT NULL
optin           list_optin DEFAULT 'single'
tags            VARCHAR(100)[]
description     TEXT DEFAULT ''
created_at      TIMESTAMP WITH TIME ZONE
updated_at      TIMESTAMP WITH TIME ZONE

Indexes:
- idx_lists_type, idx_lists_optin, idx_lists_name
- idx_lists_created_at, idx_lists_updated_at
```

**subscriber_lists** (Many-to-many subscriptions)
```sql
subscriber_id   INTEGER REFERENCES subscribers(id) ON DELETE CASCADE
list_id         INTEGER REFERENCES lists(id) ON DELETE CASCADE
meta            JSONB DEFAULT '{}'
status          subscription_status DEFAULT 'unconfirmed'
created_at      TIMESTAMP WITH TIME ZONE
updated_at      TIMESTAMP WITH TIME ZONE

PRIMARY KEY (subscriber_id, list_id)

Indexes:
- idx_sub_lists_sub_id ON (subscriber_id)
- idx_sub_lists_list_id ON (list_id)
- idx_sub_lists_status ON (status)
```

**campaigns** (Email campaigns)
```sql
id                      SERIAL PRIMARY KEY
uuid                    UUID NOT NULL UNIQUE
name                    TEXT NOT NULL
subject                 TEXT NOT NULL
from_email              TEXT NOT NULL
body                    TEXT NOT NULL
body_source             TEXT NULL
altbody                 TEXT NULL
content_type            content_type DEFAULT 'richtext'
send_at                 TIMESTAMP WITH TIME ZONE
headers                 JSONB DEFAULT '[]'
status                  campaign_status DEFAULT 'draft'
tags                    VARCHAR(100)[]
type                    campaign_type DEFAULT 'regular'
messenger               TEXT NOT NULL
template_id             INTEGER REFERENCES templates(id) ON DELETE SET NULL
to_send                 INT DEFAULT 0
sent                    INT DEFAULT 0
max_subscriber_id       INT DEFAULT 0
last_subscriber_id      INT DEFAULT 0
archive                 BOOLEAN DEFAULT false
archive_slug            TEXT UNIQUE
archive_template_id     INTEGER REFERENCES templates(id)
archive_meta            JSONB DEFAULT '{}'
started_at              TIMESTAMP WITH TIME ZONE
created_at              TIMESTAMP WITH TIME ZONE
updated_at              TIMESTAMP WITH TIME ZONE

-- Queue tracking fields
use_queue               BOOLEAN DEFAULT false
queued_at               TIMESTAMP WITH TIME ZONE
queue_completed_at      TIMESTAMP WITH TIME ZONE
auto_paused             BOOLEAN DEFAULT false
auto_paused_at          TIMESTAMP WITH TIME ZONE

Indexes:
- idx_camps_status ON (status)
- idx_camps_name ON (name)
- idx_camps_created_at, idx_camps_updated_at
```

**campaign_lists** (Campaign → List associations)
```sql
id              BIGSERIAL PRIMARY KEY
campaign_id     INTEGER REFERENCES campaigns(id) ON DELETE CASCADE
list_id         INTEGER REFERENCES lists(id) ON DELETE SET NULL
list_name       TEXT DEFAULT ''

UNIQUE (campaign_id, list_id)

Indexes:
- idx_camp_lists_camp_id ON (campaign_id)
- idx_camp_lists_list_id ON (list_id)
```

**templates** (Email templates)
```sql
id              SERIAL PRIMARY KEY
name            TEXT NOT NULL
type            template_type DEFAULT 'campaign'
subject         TEXT NOT NULL
body            TEXT NOT NULL
body_source     TEXT NULL
is_default      BOOLEAN DEFAULT false
created_at      TIMESTAMP WITH TIME ZONE
updated_at      TIMESTAMP WITH TIME ZONE

UNIQUE INDEX ON (is_default) WHERE is_default = true
```

**campaign_views** (View tracking)
```sql
id              BIGSERIAL PRIMARY KEY
campaign_id     INTEGER REFERENCES campaigns(id) ON DELETE CASCADE
subscriber_id   INTEGER REFERENCES subscribers(id) ON DELETE SET NULL
created_at      TIMESTAMP WITH TIME ZONE

Indexes:
- idx_views_camp_id ON (campaign_id)
- idx_views_subscriber_id ON (subscriber_id)
- idx_views_date ON (TIMEZONE('UTC', created_at)::DATE)
```

**links** (URL tracking)
```sql
id              SERIAL PRIMARY KEY
uuid            UUID NOT NULL UNIQUE
url             TEXT NOT NULL UNIQUE
created_at      TIMESTAMP WITH TIME ZONE
```

**link_clicks** (Click tracking)
```sql
id              BIGSERIAL PRIMARY KEY
campaign_id     INTEGER REFERENCES campaigns(id) ON DELETE CASCADE
link_id         INTEGER REFERENCES links(id) ON DELETE CASCADE
subscriber_id   INTEGER REFERENCES subscribers(id) ON DELETE SET NULL
created_at      TIMESTAMP WITH TIME ZONE

Indexes:
- idx_clicks_camp_id ON (campaign_id)
- idx_clicks_link_id ON (link_id)
- idx_clicks_sub_id ON (subscriber_id)
- idx_clicks_date ON (TIMEZONE('UTC', created_at)::DATE)
```

**bounces** (Bounce records)
```sql
id              SERIAL PRIMARY KEY
subscriber_id   INTEGER REFERENCES subscribers(id) ON DELETE CASCADE
campaign_id     INTEGER REFERENCES campaigns(id) ON DELETE SET NULL
type            bounce_type DEFAULT 'hard'
source          TEXT DEFAULT ''
meta            JSONB DEFAULT '{}'
created_at      TIMESTAMP WITH TIME ZONE

Indexes:
- idx_bounces_sub_id ON (subscriber_id)
- idx_bounces_camp_id ON (campaign_id)
- idx_bounces_source ON (source)
- idx_bounces_date ON (TIMEZONE('UTC', created_at)::DATE)
```

**media** (Uploaded files)
```sql
id              SERIAL PRIMARY KEY
uuid            UUID NOT NULL UNIQUE
provider        TEXT DEFAULT ''
filename        TEXT NOT NULL
content_type    TEXT DEFAULT 'application/octet-stream'
thumb           TEXT NOT NULL
meta            JSONB DEFAULT '{}'
created_at      TIMESTAMP WITH TIME ZONE

Indexes:
- idx_media_filename ON (provider, filename)
```

**campaign_media** (Campaign attachments)
```sql
campaign_id     INTEGER REFERENCES campaigns(id) ON DELETE CASCADE
media_id        INTEGER REFERENCES media(id) ON DELETE SET NULL
filename        TEXT DEFAULT ''

UNIQUE INDEX idx_camp_media_id ON (campaign_id, media_id)
```

**users** (Admin users)
```sql
id              SERIAL PRIMARY KEY
username        TEXT NOT NULL UNIQUE
password_login  BOOLEAN DEFAULT false
password        TEXT NULL
email           TEXT NOT NULL UNIQUE
name            TEXT NOT NULL
avatar          TEXT NULL
type            user_type DEFAULT 'user'
user_role_id    INTEGER REFERENCES roles(id) ON DELETE RESTRICT
list_role_id    INTEGER REFERENCES roles(id) ON DELETE CASCADE
status          user_status DEFAULT 'disabled'
loggedin_at     TIMESTAMP WITH TIME ZONE
created_at      TIMESTAMP WITH TIME ZONE
updated_at      TIMESTAMP WITH TIME ZONE
```

**roles** (RBAC roles)
```sql
id              SERIAL PRIMARY KEY
type            role_type DEFAULT 'user'
parent_id       INTEGER REFERENCES roles(id) ON DELETE CASCADE
list_id         INTEGER REFERENCES lists(id) ON DELETE CASCADE
permissions     TEXT[] DEFAULT '{}'
name            TEXT NULL
created_at      TIMESTAMP WITH TIME ZONE
updated_at      TIMESTAMP WITH TIME ZONE

UNIQUE INDEX idx_roles ON (parent_id, list_id)
UNIQUE INDEX idx_roles_name ON (type, name) WHERE name IS NOT NULL
```

**sessions** (User sessions)
```sql
id              TEXT PRIMARY KEY
data            JSONB DEFAULT '{}'
created_at      TIMESTAMP WITHOUT TIME ZONE

INDEX idx_sessions ON (id, created_at)
```

**email_queue** (Queue-based delivery)
```sql
id                          BIGSERIAL PRIMARY KEY
campaign_id                 INTEGER REFERENCES campaigns(id) ON DELETE CASCADE
subscriber_id               INTEGER REFERENCES subscribers(id) ON DELETE CASCADE
status                      TEXT DEFAULT 'queued'
priority                    INTEGER DEFAULT 0
scheduled_at                TIMESTAMP WITH TIME ZONE
sent_at                     TIMESTAMP WITH TIME ZONE
assigned_smtp_server_uuid   TEXT
retry_count                 INTEGER DEFAULT 0
last_error                  TEXT
created_at                  TIMESTAMP WITH TIME ZONE
updated_at                  TIMESTAMP WITH TIME ZONE

Indexes:
- idx_email_queue_status ON (status)
- idx_email_queue_scheduled_at ON (scheduled_at)
- idx_email_queue_campaign_id ON (campaign_id)
- idx_email_queue_assigned_smtp ON (assigned_smtp_server_uuid)
```

**smtp_daily_usage** (Daily send tracking)
```sql
smtp_server_uuid    TEXT NOT NULL
usage_date          DATE NOT NULL
emails_sent         INTEGER DEFAULT 0
created_at          TIMESTAMP WITH TIME ZONE
updated_at          TIMESTAMP WITH TIME ZONE

UNIQUE (smtp_server_uuid, usage_date)
```

**smtp_rate_limit_state** (Sliding window tracking)
```sql
id                  BIGSERIAL PRIMARY KEY
smtp_server_uuid    TEXT NOT NULL UNIQUE
window_start        TIMESTAMP WITH TIME ZONE
emails_in_window    INTEGER DEFAULT 0
created_at          TIMESTAMP WITH TIME ZONE
updated_at          TIMESTAMP WITH TIME ZONE
```

**account_rate_limit_state** (Account-wide rate limiting)
```sql
minute_window_start     TIMESTAMP WITH TIME ZONE
emails_in_minute        INTEGER DEFAULT 0
hour_window_start       TIMESTAMP WITH TIME ZONE
emails_in_hour          INTEGER DEFAULT 0
created_at              TIMESTAMP WITH TIME ZONE
updated_at              TIMESTAMP WITH TIME ZONE
```

**subscriber_last_send** (Smart Sending tracking)
```sql
subscriber_id           INTEGER PRIMARY KEY REFERENCES subscribers(id) ON DELETE CASCADE
last_campaign_send_at   TIMESTAMP WITH TIME ZONE
updated_at              TIMESTAMP WITH TIME ZONE
```

**azure_delivery_events** (Azure Event Grid delivery tracking)
```sql
id                          BIGSERIAL PRIMARY KEY
azure_message_id            TEXT NOT NULL
campaign_id                 INTEGER REFERENCES campaigns(id)
subscriber_id               INTEGER REFERENCES subscribers(id)
status                      TEXT NOT NULL
status_reason               TEXT
delivery_status_details     TEXT
event_timestamp             TIMESTAMP WITH TIME ZONE
created_at                  TIMESTAMP WITH TIME ZONE

Indexes:
- idx_azure_delivery_message_id ON (azure_message_id)
- idx_azure_delivery_campaign_id ON (campaign_id)
- idx_azure_delivery_subscriber_id ON (subscriber_id)
```

**azure_engagement_events** (Azure Event Grid engagement tracking)
```sql
id                      BIGSERIAL PRIMARY KEY
azure_message_id        TEXT NOT NULL
campaign_id             INTEGER REFERENCES campaigns(id)
subscriber_id           INTEGER REFERENCES subscribers(id)
engagement_type         TEXT NOT NULL
engagement_context      TEXT
user_agent              TEXT
event_timestamp         TIMESTAMP WITH TIME ZONE
created_at              TIMESTAMP WITH TIME ZONE

Indexes:
- idx_azure_engagement_message_id ON (azure_message_id)
- idx_azure_engagement_campaign_id ON (campaign_id)
- idx_azure_engagement_subscriber_id ON (subscriber_id)
```

**purchase_attributions** (Shopify purchase tracking)
```sql
id                  BIGSERIAL PRIMARY KEY
campaign_id         INTEGER REFERENCES campaigns(id)
subscriber_id       INTEGER REFERENCES subscribers(id)
order_id            TEXT NOT NULL UNIQUE
order_number        TEXT
customer_email      TEXT NOT NULL
total_price         NUMERIC(10,2)
currency            TEXT
attributed_via      TEXT
confidence          TEXT
shopify_data        JSONB
created_at          TIMESTAMP WITH TIME ZONE

Indexes:
- idx_purchase_campaign_id ON (campaign_id)
- idx_purchase_subscriber_id ON (subscriber_id)
- idx_purchase_order_id ON (order_id)
```

**webhook_logs** (Webhook request logging)
```sql
id                  BIGSERIAL PRIMARY KEY
webhook_type        TEXT NOT NULL
event_type          TEXT
request_headers     JSONB
request_body        TEXT
response_status     INTEGER
response_body       TEXT
processed           BOOLEAN DEFAULT false
error_message       TEXT
created_at          TIMESTAMP WITH TIME ZONE

Indexes:
- idx_webhook_logs_type ON (webhook_type)
- idx_webhook_logs_created_at ON (created_at)
```

**settings** (Application settings)
```sql
key             TEXT NOT NULL UNIQUE
value           JSONB DEFAULT '{}'
updated_at      TIMESTAMP WITH TIME ZONE

INDEX idx_settings_key ON (key)
```

### 3.4 Relationships

**Subscriber ↔ List** (Many-to-Many)
- Through: subscriber_lists
- Cascade: Delete subscriber deletes subscriptions
- Set NULL: Delete list sets list_id to NULL

**Campaign ↔ List** (Many-to-Many)
- Through: campaign_lists
- Cascade: Delete campaign deletes associations
- Set NULL: Delete list sets list_id to NULL but keeps list_name

**Campaign → Template** (Many-to-One)
- Foreign key: template_id
- Set NULL: Delete template sets campaign template to NULL

**Campaign ↔ Media** (Many-to-Many)
- Through: campaign_media
- Cascade: Delete campaign deletes associations
- Set NULL: Delete media sets media_id to NULL but keeps filename

**Campaign → View/Click/Bounce** (One-to-Many)
- Foreign keys in tracking tables
- Cascade: Delete campaign deletes tracking records
- Set NULL: Delete subscriber sets subscriber_id to NULL but keeps counts

**User → Role** (Many-to-One)
- Foreign keys: user_role_id, list_role_id
- Restrict: Cannot delete role if users reference it (user_role)
- Cascade: Delete list role cascades to user (list_role)

**Role Hierarchy** (Self-referencing)
- Foreign key: parent_id
- Cascade: Delete parent role deletes child roles

### 3.5 SQL Queries (queries.sql)

**Query Structure** (goyesql format)
```sql
-- name: query-name
-- Optional directives (raw: true, etc.)
SELECT ...
```

**Key Query Categories**

**Subscribers** (200+ lines)
- get-subscriber, get-subscribers-by-emails
- get-subscriber-lists, get-subscriber-lists-lazy
- insert-subscriber, upsert-subscriber, upsert-blocklist-subscriber
- update-subscriber, update-subscriber-with-lists
- delete-subscribers, delete-blocklisted-subscribers, delete-orphan-subscribers
- blocklist-subscribers
- add-subscribers-to-lists, delete-subscriptions
- confirm-subscription-optin
- unsubscribe-subscribers-from-lists, unsubscribe-by-campaign
- delete-unconfirmed-subscriptions
- export-subscriber-data
- query-subscribers (with dynamic WHERE)
- query-subscribers-count, query-subscribers-count-all
- query-subscribers-for-export
- query-subscribers-template (for bulk operations)
- delete-subscribers-by-query, blocklist-subscribers-by-query
- add-subscribers-to-lists-by-query, delete-subscriptions-by-query
- unsubscribe-subscribers-from-lists-by-query
- has-subscriber-list (permission check)

**Lists** (100+ lines)
- get-lists, query-lists, get-lists-by-optin, get-list-types
- create-list, update-list, update-lists-date, delete-lists

**Campaigns** (500+ lines)
- create-campaign (complex with CTEs for counts and media)
- query-campaigns (with list JSON aggregation)
- get-campaign, get-archived-campaigns
- get-campaign-stats (lazy loading for multiple campaigns)
- get-campaign-for-preview
- get-campaign-status
- get-campaign-queue-stats (queue-based campaigns)
- campaign-has-lists
- next-campaigns (fetch running campaigns with template bodies)
- get-running-campaign
- next-campaign-subscribers (batch subscriber fetching)
- get-one-campaign-subscriber
- update-campaign, update-campaign-counts, update-campaign-status
- update-campaign-archive
- delete-campaign
- register-campaign-view
- get-campaign-analytics-unique-counts, get-campaign-analytics-counts
- get-campaign-bounce-counts, get-campaign-link-counts
- get-campaign-azure-delivery-counts
- get-campaign-unsubscribers
- delete-campaign-views, delete-campaign-link-clicks

**Templates** (40 lines)
- get-templates, create-template, update-template
- set-default-template, delete-template

**Media** (30 lines)
- insert-media, query-media, get-media, delete-media

**Links** (20 lines)
- create-link, register-link-click

**Bounces** (80 lines)
- record-bounce (complex with subscriber status update logic)
- query-bounces, delete-bounces, delete-bounces-by-subscriber
- blocklist-bounced-subscribers

**Queue** (120 lines)
- queue-campaign-emails
- get-queued-email-count
- get-queue-stats
- cancel-campaign-queue
- update-campaign-as-queued
- get-queue-items (with campaign and subscriber joins)
- get-queue-summary-stats
- check-subscriber-smart-sending
- update-subscriber-last-send
- get-next-scheduled-email
- cancel-queue-item, retry-queue-item
- get-smtp-server-capacity
- clear-all-queued-emails, send-all-queued-emails
- get-sent-subscribers-today, requeue-cancelled-emails
- sync-queue-campaign-counts

**Shopify** (60 lines)
- insert-purchase-attribution
- find-recent-link-click, find-recent-email-open
- get-campaign-purchase-stats
- get-subscriber-by-email
- get-campaigns-performance-summary
- get-campaigns-purchase-stats

**Webhook Logs** (50 lines)
- create-webhook-log
- get-webhook-logs, get-webhook-logs-count
- get-all-webhook-logs
- delete-webhook-logs, delete-all-webhook-logs

**Users & Roles** (200+ lines)
- create-user, update-user, delete-users
- get-users, get-user, get-api-tokens
- login-user, update-user-profile, update-user-login
- get-user-roles, get-list-roles
- create-role, update-role, delete-role
- upsert-list-permissions, delete-list-permission

**Dashboard** (2 lines)
- get-dashboard-charts, get-dashboard-counts

**Settings** (20 lines)
- get-settings, update-settings
- get-db-info

### 3.6 Materialized Views

**mat_dashboard_counts**
- Refreshed via cron or on-demand
- Aggregates:
  - Subscriber counts by status
  - Orphaned subscribers (no list memberships)
  - List counts by type and optin
  - Campaign counts by status
  - Total messages sent
- Indexed on updated_at (UNIQUE)

**mat_dashboard_charts**
- Refreshed via cron or on-demand
- Aggregates:
  - Link clicks by date (last 30 days)
  - Campaign views by date (last 30 days)
- Indexed on updated_at (UNIQUE)

**mat_list_subscriber_stats**
- Refreshed via cron or on-demand
- Subscriber counts per list by subscription status
- Plus total subscriber count (list_id = 0)
- Indexed on (list_id, status) (UNIQUE)

**Refresh Mechanism**
- Manual: Via maintenance UI or API
- Automatic: Cron job at configured interval
- Used for performance optimization on large datasets

### 3.7 Migrations

**Migration System**
- Version-specific migration files in internal/migrations/
- Format: v{major}.{minor}.{patch}.go
- Each migration implements up/down functions
- Registered in cmd/upgrade.go migList
- Applied sequentially during --upgrade
- Version tracking in database

**Migration History** (from internal/migrations/)
- v0.4.0, v0.7.0, v0.8.0, v0.9.0
- v1.0.0
- v2.0.0, v2.1.0, v2.2.0, v2.3.0, v2.4.0, v2.5.0
- v3.0.0
- v4.0.0, v4.1.0
- v5.0.0, v5.1.0, v5.2.0
- v6.0.0 (Queue system)
- v6.1.0, v6.2.0, v6.3.0
- v7.0.0, v7.1.0, v7.2.0 (Azure integration, Shopify integration)

---

## 4. ADVANCED FEATURES

### 4.1 Queue System

**Purpose**
- Handle daily sending limits per SMTP server
- Enforce time windows (send only during specific hours)
- Provide capacity-based server selection
- Enable automatic campaign management
- Support account-wide rate limiting
- Implement Smart Sending

**Components**

1. **Database Tables**:
   - email_queue: Stores all queued emails
   - smtp_daily_usage: Tracks daily email counts per server
   - smtp_rate_limit_state: Tracks sliding window per server
   - account_rate_limit_state: Tracks account-wide rate limits
   - subscriber_last_send: Tracks last send time per subscriber (Smart Sending)

2. **Queue Processor** (internal/queue/processor.go):
   - Polls database every 5 seconds
   - Fetches batch of queued emails (configurable batch size)
   - Selects SMTP server based on capacity
   - Sends via specific messenger (email-{name})
   - Updates status and counters
   - Enforces time windows
   - Handles auto-pause/resume
   - Syncs campaign stats

3. **Auto-Pause Scheduler**:
   - Runs every minute
   - Checks if current time is within send window
   - Auto-pauses running campaigns outside window
   - Auto-resumes paused campaigns inside window
   - Requeues cancelled emails on resume

4. **Campaign Stats Sync**:
   - Runs every 5 minutes
   - Syncs campaign.sent counts from email_queue
   - Ensures accurate statistics

**Flow**

1. **Queueing**:
   - User creates campaign with messenger="automatic"
   - User sets campaign status to "running"
   - Backend calls core.QueueCampaignEmails()
   - All campaign recipients inserted into email_queue (status='queued')
   - Campaign marked with use_queue=true, queued_at=NOW()
   - Emails scheduled 2 minutes in future

2. **Processing**:
   - Queue processor polls every 5 seconds
   - Checks if queue is paused (settings.app.queue_paused)
   - Checks if within time window
   - Fetches batch of emails with scheduled_at <= NOW()
   - Gets server capacities (daily limits, sliding window)
   - For each email:
     - Selects server with most capacity
     - Marks email as 'sending' (atomic)
     - Sends via campaign manager's PushCampaignMessageByID()
     - Updates status to 'sent' or 'failed'
     - Increments server usage counters
     - Updates subscriber_last_send for Smart Sending
   - Concurrent sending with semaphore (configurable concurrency)
   - Rate limiting (configurable message rate + account-wide limits)

3. **Completion**:
   - Queue processor checks for campaigns with no queued/sending emails
   - Marks campaign as 'finished'
   - Sets queue_completed_at

**Capacity Management**

- **Daily Limit** (per server):
  - Configured in SMTP settings (daily_limit field)
  - 0 = unlimited
  - Tracked in smtp_daily_usage table (resets daily)
  - Server skipped if limit reached

- **Sliding Window** (per server):
  - Configured globally (app.message_sliding_window_duration, app.message_sliding_window_rate)
  - Tracked in smtp_rate_limit_state table
  - Window resets after duration expires
  - Server skipped if rate exceeded

- **Account-Wide Rate Limits**:
  - Per-minute limit (app.account_rate_limit_per_minute)
  - Per-hour limit (app.account_rate_limit_per_hour)
  - Takes precedence over message rate
  - Prevents Azure rate limit violations (1-hour cooldown)
  - Tracked in account_rate_limit_state table

- **Server Selection**:
  - Prioritize server with most daily remaining capacity
  - Skip servers at capacity (daily or sliding window)
  - If all servers at capacity, skip email for this batch
  - Email remains queued for next poll

**Time Window**

- **Configuration**:
  - app.send_time_start (HH:MM format, e.g., "08:00")
  - app.send_time_end (HH:MM format, e.g., "20:00")
  - app.timezone (e.g., "America/New_York")
  - Empty values = 24/7 sending

- **Enforcement**:
  - Queue processor checks time window before processing
  - Auto-pause scheduler checks every minute
  - Campaigns paused outside window (auto_paused=true)
  - Campaigns resumed inside window
  - Cancelled emails requeued on resume

**Smart Sending**

- **Purpose**: Prevent sending to same subscriber too frequently
- **Configuration**:
  - app.smart_sending_enabled (boolean)
  - app.smart_sending_hours (e.g., 24)
- **Mechanism**:
  - subscriber_last_send table tracks last send time
  - Check performed DURING queueing, not at send time
  - Subscribers within window skipped when queueing
  - After successful send, last_send_at updated
- **Performance**: Filtering at queue time prevents rate limit waste

### 4.2 Multi-SMTP

**Named SMTP Servers**
- Each SMTP server can have a "name" field
- Format: alphanumeric + hyphens only
- Automatically prefixed with "email-" in messenger name
- Example: name="mail2" → messenger="email-mail2"

**Messenger Creation**
- Named servers → dedicated messengers
- Campaigns can select specific messenger
- Unnamed servers → grouped into default "email" messenger
  - Random selection among unnamed servers
  - Maintains backward compatibility

**Configuration Per Server**
- Host, port, credentials
- Connection pool settings (max_conns, timeouts)
- TLS configuration
- From email (per-server sender address)
- Daily limit (0 = unlimited)
- Bounce mailbox UUID (links to specific POP3 mailbox)
- Email headers (custom headers per server)

**Bounce Mailbox Linking**
- Each SMTP server can link to specific bounce mailbox via bounce_mailbox_uuid
- Mailboxes identified by UUID and name
- Multiple mailboxes supported (30+)
- Each mailbox scans at configured interval (minimum 1 minute)
- Each mailbox runs in separate goroutine
- Return-Path matching for bounce detection

**Queue Integration**
- Queue processor selects specific server based on capacity
- Calls campaign manager with server UUID
- Manager looks up server details (name, username, from_email)
- Uses specific messenger (email-{name})
- Ensures correct SMTP credentials and from_email used

### 4.3 Analytics

**View Tracking**
- Transparent 1x1 pixel (px.png) embedded in email
- URL: /campaign/:campUUID/:subUUID/px.png
- Records to campaign_views table
- Individual or anonymous tracking (configurable)
- Timestamp recorded
- Used for open rate calculation

**Click Tracking**
- TrackLink template function wraps URLs
- URL → UUID mapping stored in links table
- Click recorded to link_clicks table
- Redirects to original URL after recording
- Individual or anonymous tracking (configurable)
- Timestamp recorded
- Top links by click count

**Bounce Tracking**
- Multiple sources:
  - POP3 mailbox scanning
  - Webhooks (SES, SendGrid, Postmark, ForwardEmail, Azure)
- Stored in bounces table with:
  - Type (hard, soft, complaint)
  - Source (webhook name or "mailbox")
  - Metadata (JSONB)
  - Campaign and subscriber links
  - Timestamp
- Configurable actions per type:
  - Count threshold
  - Action: none, unsubscribe, blocklist, delete

**Azure Event Grid Integration**

**Delivery Events**:
- Status: Delivered, Bounced, Suppressed, Failed, Quarantined, FilteredSpam
- Tracked in azure_delivery_events table
- Fields: azure_message_id, campaign_id, subscriber_id, status, status_reason, delivery_status_details
- Used for accurate delivery tracking
- Replaces traditional sent count for Azure campaigns

**Engagement Events**:
- Type: open (view), click
- Tracked in azure_engagement_events table
- Fields: azure_message_id, campaign_id, subscriber_id, engagement_type, engagement_context, user_agent
- Provides additional tracking beyond pixel/link methods

**Webhook**:
- Endpoint: /webhooks/azure/eventgrid
- Processes Microsoft.Communication.EmailDeliveryReportReceived
- Processes Microsoft.Communication.EmailEngagementTrackingReportReceived
- Correlates events via azure_message_id
- Maps to campaigns and subscribers via custom message properties

**Shopify Purchase Attribution**

**Integration**:
- Webhook: /webhooks/shopify/orders
- Webhook secret verification
- Processes order creation events

**Attribution Logic**:
- Looks for recent link click OR email open within attribution window
- Attribution window configurable (default: 7 days)
- Finds subscriber by customer email
- Records to purchase_attributions table
- Fields: campaign_id, subscriber_id, order_id, total_price, currency, attributed_via, confidence

**Attribution Methods**:
- Link click: Highest confidence (subscriber clicked link in campaign email)
- Email open: Medium confidence (subscriber opened campaign email)
- Confidence levels: high, medium, low

**Campaign-Level Stats**:
- Total orders attributed to campaign
- Total revenue attributed to campaign
- Average order value
- Currency
- Used in campaign analytics

**Aggregate Performance**:
- Endpoint: /api/campaigns/performance/summary
- Timeframe parameter (days)
- Metrics:
  - Average open rate across campaigns
  - Average click rate across campaigns
  - Total sent (via Azure delivery events)
  - Total orders (via purchase attributions)
  - Total revenue (via purchase attributions)
  - Order rate (orders / sends)
  - Error rate (failed deliveries / sends)

### 4.4 Bounce Handling

**Mailbox Scanning (POP3)**
- Multiple mailboxes supported (30+)
- Configuration per mailbox:
  - Host, port, credentials
  - Return-Path (identifies mailbox)
  - Scan interval (minimum 1 minute)
  - TLS settings
  - UUID and name for identification
- Each mailbox runs in separate goroutine
- Fetches emails via POP3
- Parses bounce emails:
  - Extracts subscriber email
  - Determines bounce type
  - Records to bounces table
- Deletes processed emails from mailbox
- Logs errors and statistics

**Webhooks**

**Amazon SES**:
- SNS topic subscription
- Endpoint: /webhooks/bounce/ses
- Processes Bounce, Complaint, and Permanent Failure notifications
- JSON payload parsing
- Automatic bounce type determination

**SendGrid**:
- Event webhook
- Endpoint: /webhooks/bounce/sendgrid
- Key-based authentication (sendgrid_key in settings)
- Processes bounce, dropped, and spam report events
- Batch event processing

**Postmark**:
- Inbound bounce API
- Endpoint: /webhooks/bounce/postmark
- Basic authentication (username/password in settings)
- Processes bounce notifications
- Automatic type classification

**ForwardEmail**:
- Webhook with key authentication
- Endpoint: /webhooks/bounce/forwardemail
- Processes bounce events
- Key verification

**Azure Event Grid**:
- Endpoint: /webhooks/azure/eventgrid
- Processes EmailDeliveryReportReceived events
- Status mapping: Bounced, Suppressed, Failed, Quarantined, FilteredSpam
- Records to both bounces table and azure_delivery_events table

**Shopify (via Azure Event Grid)**:
- Endpoint: /webhooks/shopify/orders
- Not a bounce handler, but uses similar webhook pattern
- Order creation events
- Purchase attribution

**Bounce Actions**
- Configured per bounce type (soft, hard, complaint)
- Count threshold (e.g., 2 soft bounces, 1 hard bounce)
- Actions:
  - none: Record only
  - unsubscribe: Set subscription status to 'unsubscribed' for all lists
  - blocklist: Set subscriber status to 'blocklisted', unsubscribe from all lists
  - delete: Delete subscriber record (cascades to subscriptions)
- Applied automatically when threshold reached
- Prevents further emails to problematic addresses

**Bounce Management UI**
- View all bounces with filtering
- Filter by type, source, campaign, subscriber
- Search by email
- Delete bounces (single or bulk)
- Blocklist bounced subscribers (bulk)
- Export bounce data

---

## 5. DEPLOYMENT & OPERATIONS

### 5.1 Build & Distribution

**Build Commands**

Backend:
```bash
make build                # Build binary to ./listmonk
make run                  # Dev mode (loads frontend from frontend/dist)
make test                 # Run Go tests
make dist                 # Build distribution with embedded assets
make pack-bin             # Pack static assets into binary using stuffbin
```

Frontend:
```bash
make build-frontend       # Build main frontend + email-builder
make build-email-builder  # Build only email-builder
make run-frontend         # Run frontend dev server
cd frontend && yarn install  # Install dependencies
cd frontend && yarn dev      # Dev server directly
cd frontend && yarn build    # Production build
```

Docker:
```bash
make dev-docker           # Build and run dev environment
make run-backend-docker   # Run backend with docker dev config
make init-dev-docker      # Initialize dev database
make rm-dev-docker        # Tear down docker environment
```

Database:
```bash
./listmonk --install      # Install database schema (first time)
./listmonk --upgrade      # Upgrade existing database (idempotent)
./listmonk --new-config   # Generate config.toml.sample
```

**Asset Embedding**
- Uses `stuffbin` to embed static assets
- Embedded files:
  - Frontend dist (admin UI)
  - Email templates
  - SQL files (schema.sql, queries.sql)
  - Config sample
  - i18n translations
  - Permissions definitions
- Allows single binary distribution
- Falls back to filesystem in dev mode

### 5.2 Configuration Management

**Config File Loading**
1. Command-line flags (`--config` to specify files)
2. Multiple config files merged in order
3. Environment variables override config files (LISTMONK_ prefix)
4. Database settings override environment variables

**Environment Variable Format**
```bash
# Double underscore (__) represents nested keys
LISTMONK_app__address=0.0.0.0:9000
LISTMONK_db__host=postgres
LISTMONK_smtp__0__host=smtp.example.com
```

**Database Settings**
- Stored in settings table
- Editable via Settings UI
- Hot-reload on change (triggers app restart)

### 5.3 Database Management

**Installation**
```bash
./listmonk --install
```
- Creates all tables
- Inserts default settings
- Creates default admin user (if legacy config present)
- Idempotent (can run multiple times with --idempotent flag)

**Upgrades**
```bash
./listmonk --upgrade
```
- Runs pending migrations
- Version tracking
- Idempotent (safe to run multiple times)
- Prompts for confirmation (use --yes to skip)

**Migrations**
- Located in internal/migrations/
- Version-specific (v{major}.{minor}.{patch}.go)
- Up and Down functions
- Applied sequentially
- Registered in cmd/upgrade.go

**Maintenance**
- Materialized view refresh (cron or manual)
- Delete old analytics (campaign views/clicks)
- Delete blocklisted subscribers
- Delete orphan subscribers
- Delete unconfirmed subscriptions
- Database size monitoring

### 5.4 Monitoring & Logging

**Logging**
- Console output with timestamps (EST 12-hour format)
- File info included (Lshortfile)
- Buffered log viewer in UI (last 5000 lines)
- Log level: INFO by default

**Health Check**
- Endpoint: /api/health
- Returns server status
- Used for readiness probes

**Metrics**
- Dashboard counts (subscribers, lists, campaigns, messages)
- Dashboard charts (views, clicks over time)
- Campaign statistics (sent, views, clicks, bounces)
- Queue statistics (queued, sending, sent, failed)
- SMTP server capacity (daily used/remaining)
- Import status

**Events**
- Server-sent events (SSE) for real-time updates
- Event stream at /api/events
- Used for notification delivery

### 5.5 Security

**Authentication**
- Session-based for admin UI
- Token-based for API (Basic Auth)
- OIDC support for SSO
- Password hashing (bcrypt)
- API tokens (user type: api)

**Authorization**
- RBAC with user and list roles
- Permission-based access control
- Route-level permission checking
- Fine-grained permissions:
  - campaigns:get, campaigns:get_all, campaigns:manage
  - subscribers:get, subscribers:get_all, subscribers:manage
  - lists:get, lists:get_all, lists:manage
  - templates:get, templates:manage
  - media:get, media:manage
  - users:get, users:manage
  - settings:get, settings:manage
  - ...and more

**CAPTCHA**
- hCaptcha support (key/secret)
- Altcha support (proof-of-work, complexity configurable)
- Applied to public subscription forms

**CORS**
- Configurable allowed origins
- Used for API access from external domains

**HTTPS**
- Recommended for production
- TLS termination via reverse proxy (nginx, caddy)
- Or direct TLS support in Go

**Secrets Management**
- Passwords encrypted in database
- Webhook secrets verified
- API keys for external services (SendGrid, S3, etc.)

---

## 6. COMMON USE CASES

### 6.1 Newsletter Management

**Workflow**:
1. Create lists (public for subscriptions, private for segmentation)
2. Import or manually add subscribers
3. Create template (or use default)
4. Create campaign:
   - Select lists
   - Compose content (visual editor, HTML, or markdown)
   - Add media attachments (optional)
   - Set from address and subject
   - Configure archiving (optional)
5. Preview and test
6. Schedule or send immediately
7. Monitor analytics (views, clicks, bounces)

**Features Used**:
- List management
- Template editor
- Campaign composer
- Media library
- Analytics dashboard

### 6.2 Transactional Emails

**Workflow**:
1. Create tx template with subject and body
2. Use API to send:
   ```bash
   POST /api/tx
   {
     "subscriber_email": "user@example.com",
     "template_id": 1,
     "data": {
       "name": "John",
       "order_id": "12345"
     }
   }
   ```
3. Template renders with data
4. Sends immediately via configured messenger

**Features Used**:
- TX templates
- Template functions
- API authentication
- Multiple messengers

### 6.3 Multi-Tenant Setup

**Scenario**: Multiple brands or departments, each with their own lists and campaigns

**Approach**:
1. Create list roles for each tenant
2. Assign permissions per tenant list
3. Create users with specific list roles
4. Users see only their permitted lists
5. Campaigns filtered by accessible lists

**Features Used**:
- List roles (RBAC)
- Permission system
- User management
- List filtering

### 6.4 High-Volume Sending

**Scenario**: Sending millions of emails per day

**Approach**:
1. Configure multiple SMTP servers (30+)
2. Set daily limits per server
3. Use queue-based delivery (messenger: automatic)
4. Configure time windows (e.g., 8am-8pm)
5. Set concurrency and message rate appropriately
6. Enable sliding window rate limiting
7. Monitor queue and server capacity

**Features Used**:
- Multi-SMTP support
- Queue system
- Capacity management
- Time windows
- Rate limiting
- Queue monitoring

**Configuration Example**:
```toml
[app]
batch_size = 1000
concurrency = 50
message_rate = 500
message_sliding_window = true
message_sliding_window_duration = "1h"
message_sliding_window_rate = 10000
send_time_start = "08:00"
send_time_end = "20:00"

# 30 SMTP servers with daily limits
[[smtp]]
enabled = true
name = "server1"
host = "smtp1.example.com"
# ... credentials ...
daily_limit = 50000

[[smtp]]
enabled = true
name = "server2"
host = "smtp2.example.com"
# ... credentials ...
daily_limit = 50000

# ... repeat for 30 servers ...
```

### 6.5 E-commerce Integration (Shopify)

**Workflow**:
1. Configure Shopify webhook settings:
   - Enable Shopify integration
   - Set webhook secret
   - Set attribution window (7 days)
2. Shopify sends order webhooks to /webhooks/shopify/orders
3. listmonk attributes orders to campaigns:
   - Finds recent link click or email open
   - Matches subscriber by email
   - Records to purchase_attributions table
4. View campaign-level purchase stats:
   - Total orders
   - Total revenue
   - Average order value
5. View aggregate performance summary:
   - Order rate across all campaigns
   - Total revenue
   - Revenue per campaign

**Features Used**:
- Shopify webhook integration
- Purchase attribution
- Campaign analytics
- Performance dashboard

### 6.6 Deliverability Monitoring (Azure)

**Workflow**:
1. Configure Azure Communication Services with Event Grid
2. Enable Azure integration (bounce.azure.enabled = true)
3. Azure sends webhook events to /webhooks/azure/eventgrid
4. listmonk records:
   - Delivery events (Delivered, Bounced, Failed, etc.)
   - Engagement events (view, click)
5. View campaign analytics:
   - Azure delivery stats by status
   - Engagement events
6. Compare traditional tracking (pixel/link) with Azure engagement events

**Features Used**:
- Azure Event Grid integration
- Webhook logs
- Campaign analytics
- Delivery event tracking

---

## 7. PERFORMANCE CONSIDERATIONS

### 7.1 Database Optimization

**Materialized Views**
- Pre-computed aggregates for dashboard
- Refresh via cron (e.g., 3am daily)
- Significantly faster than live queries on large datasets
- Trade-off: Data may be slightly stale

**Indexes**
- Extensive indexing on frequently queried columns
- Composite indexes for common query patterns
- Date indexes for time-based queries
- Full-text search indexes (tsvector)

**Query Patterns**
- Named queries via goyesql
- Prepared statements for all queries
- Efficient CTEs for complex operations
- JSONB aggregation for list/media associations
- Batch operations for bulk updates

**Connection Pooling**
- Configurable max open/idle connections
- Connection lifetime management
- Unsafe mode for performance (disables prepared statement caching in favor of connection pool)

### 7.2 Campaign Processing

**Batch Fetching**
- Configurable batch size (default: 1000)
- Fetches subscribers in batches
- Reduces database load
- Allows parallel processing

**Concurrency**
- Configurable worker pool (default: 10)
- Each worker processes messages concurrently
- Semaphore for connection limiting
- Rate limiting per worker

**Rate Limiting**
- Message rate (messages per second across all workers)
- Sliding window rate limiting (messages per window per messenger)
- Account-wide rate limiting (per-minute and per-hour)
- Time window enforcement (send only during configured hours)

**Error Handling**
- Max send errors threshold
- Campaign stops if threshold exceeded
- Error counter per campaign
- Prevents wasting resources on broken campaigns

### 7.3 Queue Processing

**Polling Interval**
- Default: 5 seconds (responsive)
- Trade-off: Database load vs. responsiveness
- Configurable via Config

**Batch Size**
- Default: 100 emails per poll
- Larger batches = more throughput
- Smaller batches = less memory usage
- Configurable via Config

**Concurrency**
- Semaphore limits concurrent SMTP connections
- Based on app.concurrency setting
- Prevents overwhelming SMTP servers

**Capacity-Based Selection**
- Selects server with most remaining capacity
- Prevents overloading single server
- Balances load across all servers

**Optimizations**
- Atomic email claiming (prevents duplicate sends in multi-instance setup)
- Pre-fetch server capacities (reduces database queries)
- In-memory tracking of batch usage
- Sliding window checks before send (not per email)

### 7.4 Frontend Performance

**Lazy Loading**
- Route-based code splitting
- Components loaded on-demand
- Reduces initial bundle size

**API Optimizations**
- Request caching in Vuex
- Loading states to prevent duplicate requests
- Pagination for large result sets
- no_body parameter to exclude heavy fields

**UI Optimizations**
- Virtual scrolling for large lists (not implemented, but recommended)
- Debounced search inputs
- Optimistic UI updates
- Skeleton loaders during fetch

### 7.5 Scalability

**Horizontal Scaling**
- Multiple listmonk instances behind load balancer
- Shared PostgreSQL database
- Sticky sessions for admin UI
- Passive mode (campaign processing disabled) for API-only instances

**Campaign Manager**
- Single instance processes campaigns (ScanCampaigns=true)
- Other instances serve API and UI (ScanCampaigns=false)
- Prevents duplicate campaign processing

**Queue Processor**
- Can run in multiple instances
- Atomic email claiming prevents duplicates
- Naturally load-balanced
- No coordination required

**Database Scaling**
- PostgreSQL replication (read replicas)
- Partitioning for large tables (views, clicks, bounces)
- Archive old data
- Materialized views for reporting

**Media Storage**
- Filesystem or S3
- S3 recommended for multi-instance setups
- CDN integration via S3 public URLs

---

## 8. EXTENSIBILITY

### 8.1 Custom Messengers

**Interface** (internal/manager/manager.go)
```go
type Messenger interface {
    Name() string
    Push(models.Message) error
    Flush() error
    Close() error
}
```

**Optional Interface** (for sliding window)
```go
type MessengerWithSlidingWindow interface {
    Messenger
    GetSlidingWindow() (enabled bool, duration string, rate int)
}
```

**Implementation**
1. Create package in internal/messenger/
2. Implement Messenger interface
3. Initialize in cmd/init.go
4. Register with manager.AddMessenger()

**Examples**:
- email: SMTP delivery (internal/messenger/email/)
- postback: HTTP webhook (internal/messenger/postback/)
- automatic: Queue-based (internal/messenger/automatic/)

### 8.2 Custom Bounce Handlers

**Webhook Handler** (internal/bounce/webhooks/)
1. Create handler file (e.g., myservice.go)
2. Implement HTTP handler
3. Parse webhook payload
4. Extract subscriber email, bounce type, metadata
5. Call recordBounce callback
6. Register route in cmd/bounce.go

**Examples**:
- ses.go: Amazon SES SNS notifications
- sendgrid.go: SendGrid Event Webhook
- postmark.go: Postmark Inbound API
- azure.go: Azure Event Grid
- shopify.go: Shopify Orders (similar pattern)

**Mailbox Scanner** (internal/bounce/mailbox/)
1. Implement mailbox protocol (currently only POP3)
2. Parse email format
3. Extract bounce information
4. Call recordBounce callback

### 8.3 Template Functions

**Adding Custom Functions**
1. Modify manager.TemplateFuncs() (internal/manager/manager.go)
2. Add function to returned FuncMap
3. Recompile

**Example**:
```go
funcs := template.FuncMap{
    "MyCustomFunc": func(arg string) string {
        // Implementation
        return result
    },
}
```

**Available Functions**:
- Sprig functions (80+): date, string, math, crypto, etc.
- listmonk functions: TrackLink, UnsubscribeURL, etc.
- Custom functions: Date, L (i18n), Safe (HTML)

### 8.4 Authentication Providers

**OIDC Integration**
- Generic OIDC provider support
- Configurable provider URL, client ID/secret
- Auto-create users option
- Default role assignment

**Custom Auth**
- Modify internal/auth/auth.go
- Implement new provider
- Update cmd/auth.go to initialize

### 8.5 API Extensions

**Adding New Endpoints**
1. Create handler in cmd/ (e.g., cmd/myfeature.go)
2. Implement HTTP handler
3. Add route in cmd/handlers.go (initHTTPHandlers function)
4. Add permission checks
5. Update API client in frontend (frontend/src/api/index.js)

**Pattern**:
```go
func (a *App) MyHandler(c echo.Context) error {
    // Get authenticated user
    user := auth.GetUser(c)

    // Check permissions
    if !user.HasPerm(auth.PermMyFeature) {
        return echo.NewHTTPError(http.StatusForbidden, "permission denied")
    }

    // Parse request
    var req MyRequest
    if err := c.Bind(&req); err != nil {
        return err
    }

    // Call core business logic
    res, err := a.core.DoMyThing(req)
    if err != nil {
        return err  // Returns echo.HTTPError
    }

    // Return response
    return c.JSON(http.StatusOK, okResp{res})
}
```

---

## 9. GOTCHAS & BEST PRACTICES

### 9.1 Common Gotchas

**Frontend Changes**
- Require rebuild (`make build-frontend`) before appearing in binary
- Email-builder must be built before main frontend in dev mode
- Asset version hash forces cache invalidation

**Database Migrations**
- Version-specific and run during --upgrade
- Cannot be rolled back automatically (must write down migration)
- Always backup before upgrading

**SMTP Server Names**
- Automatically prefixed with `email-` and sanitized (alphanumeric + hyphens only)
- Changes to name break existing campaign messenger references
- Use descriptive, stable names

**Bounce Mailbox Intervals**
- Minimum 1 minute scan interval
- Too frequent scanning can overwhelm POP3 server
- Each mailbox runs in separate goroutine

**Campaign Manager vs. Queue**
- Traditional campaigns: Processed by campaign manager
- Queue-based campaigns: Processed by queue processor
- Don't mix: Campaign manager skips queue-based campaigns (use_queue=true)
- Queue processor only handles queue-based campaigns

**PostgreSQL Aggregate Functions**
- Every column in SELECT must be in GROUP BY OR wrapped in aggregate function
- CROSS JOIN with aggregates requires MAX() or add to GROUP BY
- Always use COALESCE with aggregates (NULL → 0)
- sqlx.Get() cannot scan NULL into int fields

**Asset Embedding**
- Binary expects embedded assets by default
- Dev mode uses filesystem (via frontendDir ldflags)
- Static assets loaded from disk if embed fails

**Time Zones**
- Time window enforcement uses configured timezone
- Database timestamps are UTC (TIMESTAMP WITH TIME ZONE)
- Display times converted to user's local timezone in UI

### 9.2 Best Practices

**Configuration**
- Use environment variables for secrets
- Keep config.toml minimal (only non-secrets)
- Use database settings for runtime-changeable values
- Always specify timezone for time windows

**Database**
- Regular backups (pg_dump)
- Materialized view refresh schedule (avoid peak hours)
- Monitor database size and connection pool
- Archive old analytics data

**Campaigns**
- Test with small list before sending to large list
- Use draft status for work-in-progress
- Preview before sending
- Monitor error count during sending

**Queue System**
- Set realistic daily limits per SMTP server
- Configure time windows to match business hours
- Monitor server capacity regularly
- Use queue for large campaigns (>10k recipients)
- Set appropriate batch size and concurrency

**SMTP Servers**
- Use named servers for dedicated purposes
- Link bounce mailboxes to respective servers
- Set from_email per server for consistency
- Test SMTP settings before going live
- Monitor daily usage to prevent hitting limits

**Performance**
- Enable materialized view caching for large databases
- Set appropriate batch size and concurrency
- Use sliding window rate limiting for high-volume
- Monitor campaign send rate and adjust as needed

**Security**
- Use OIDC for SSO in enterprise environments
- Create API users with minimal permissions
- Enable CAPTCHA on public forms
- Use HTTPS in production
- Regularly update listmonk to latest version

**Monitoring**
- Check logs regularly for errors
- Monitor queue statistics
- Track bounce rates
- Review webhook logs for integration issues
- Monitor database size and performance

**Subscribers**
- Use double opt-in for compliance (GDPR, CAN-SPAM)
- Provide easy unsubscribe mechanism
- Honor unsubscribe requests immediately
- Export subscriber data on request (privacy compliance)
- Clean up blocklisted subscribers regularly

**Templates**
- Use base template for consistent branding
- Test templates across email clients
- Use alt body for accessibility
- Minimize template size (faster rendering)
- Use template functions for dynamic content

**Deliverability**
- Configure proper SPF, DKIM, DMARC records
- Use dedicated SMTP servers for bulk sending
- Monitor bounce rates closely
- Handle bounces promptly
- Use clean, verified subscriber lists

---

## 10. CONCLUSION

listmonk is a comprehensive, production-ready newsletter and mailing list management system with:

- **Robust architecture**: Clear separation of concerns (HTTP → core → database)
- **Scalability**: Horizontal scaling, queue-based delivery, multi-SMTP support
- **Flexibility**: Multiple content types, custom templates, extensible design
- **Compliance**: GDPR-friendly, privacy controls, bounce handling
- **Analytics**: Comprehensive tracking (views, clicks, bounces, purchases)
- **Integration**: Webhooks, APIs, OIDC, cloud services (Azure, Shopify)
- **Performance**: Optimized queries, materialized views, efficient processing
- **User-friendly**: Intuitive UI, rich editor, preview functionality
- **Self-hosted**: Full control, no vendor lock-in, single binary distribution

The system is suitable for:
- Small newsletters (few hundred subscribers)
- Medium-scale marketing campaigns (tens of thousands)
- High-volume transactional emails (millions per day)
- Multi-tenant environments
- E-commerce integrations
- Enterprise deployments with SSO

With proper configuration and infrastructure, listmonk can handle millions of subscribers and send millions of emails per day while maintaining deliverability and providing detailed analytics.

---

**End of Specification Document**

**Document Statistics**:
- Backend (Go): ~10,130 lines across 102 files
- Frontend (Vue): ~60 files (estimate: 15,000-20,000 lines)
- Database (SQL): schema.sql (434 lines) + queries.sql (1,809 lines)
- Total API Endpoints: 156+
- Total Database Tables: 30+

This specification is based on the current state of the listmonk codebase as of 2025-11-13. For the most up-to-date information, refer to the source code repository and official documentation.
