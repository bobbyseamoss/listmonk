---
name: migration-deployment-guidelines
description: Migration and deployment best practices for listmonk (database migrations, Docker builds, Azure deployment). CRITICAL skill for any migration, schema change, or deployment work.
---

# Migration & Deployment Guidelines for listmonk

## Purpose

Ensure safe, reliable migrations and deployments with zero data loss and minimal downtime for the listmonk application.

## When to Use This Skill

**CRITICAL - Automatically activates when:**
- Creating or modifying database migrations
- Making schema changes (ALTER TABLE, CREATE TABLE, etc.)
- Building Docker images
- Deploying to Azure Container Apps
- Working with deploy.sh or Dockerfile
- Updating schema.sql or migration files

---

## 🚨 CRITICAL PRE-DEPLOYMENT CHECKLIST

**STOP if you cannot check ALL boxes:**

### Database Migrations
- [ ] Migration is idempotent (uses `IF NOT EXISTS`, `IF EXISTS`)
- [ ] Migration file named `v{major}.{minor}.{patch}.go`
- [ ] Registered in `cmd/upgrade.go` migList array
- [ ] `schema.sql` updated with final schema state
- [ ] Tested with `--upgrade` on dev database
- [ ] No data loss - existing data backfilled if needed
- [ ] Migration can run multiple times safely

### Docker Build
- [ ] Multi-stage build works locally
- [ ] Frontend builds successfully
- [ ] Backend compiles without errors
- [ ] Image size reasonable (<500MB compressed)
- [ ] All static assets copied correctly
- [ ] Health checks functional

### Azure Deployment
- [ ] Database backup taken (production)
- [ ] Migrations run BEFORE container deployment
- [ ] Environment variables verified
- [ ] New revision created with unique suffix
- [ ] Rollback plan documented
- [ ] Team notified of deployment

---

## Database Migration Guide

### Location & Structure
```
internal/migrations/
├── v6.0.0.go    # Queue system tables
├── v6.1.0.go    # Azure message tracking
└── v6.2.0.go    # Delivery events
```

### Migration Template

```go
package migrations

import (
    "github.com/jmoiron/sqlx"
    "github.com/knadh/stuffbin"
)

func v6_X_0(db *sqlx.DB, fs stuffbin.FileSystem, prompt bool) error {
    // ALWAYS use IF NOT EXISTS for idempotency
    if _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS new_table (
            id SERIAL PRIMARY KEY,
            name TEXT NOT NULL,
            created_at TIMESTAMP NOT NULL DEFAULT NOW()
        );

        CREATE INDEX IF NOT EXISTS idx_new_table_name
            ON new_table(name);
    `); err != nil {
        return err
    }

    return nil
}
```

### Register Migration in cmd/upgrade.go

```go
var migList = []migration{
    // ... existing migrations
    {ver: "v6.X.0", upgradeFunc: migrations.V6_X_0},
}
```

### Update schema.sql

After creating migration, update `schema.sql` to reflect final state:
```sql
-- Add your table/column to schema.sql
CREATE TABLE new_table (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### Common Patterns

**Add Table:**
```sql
CREATE TABLE IF NOT EXISTS table_name (
    id SERIAL PRIMARY KEY,
    -- columns
);
```

**Add Column:**
```sql
-- PostgreSQL 11+
ALTER TABLE table_name
ADD COLUMN IF NOT EXISTS column_name TYPE DEFAULT value;
```

**Add Index:**
```sql
CREATE INDEX IF NOT EXISTS idx_name
ON table_name(column);
```

**Backfill Data:**
```sql
-- Update existing rows with new column value
UPDATE table_name
SET new_column = old_column
WHERE new_column IS NULL;
```

### ❌ DANGER ZONE - NEVER DO THIS

- ❌ `DROP TABLE` without user confirmation
- ❌ `ALTER COLUMN TYPE` without data migration plan
- ❌ `DROP COLUMN` before verifying no code references it
- ❌ Non-idempotent migrations (will fail on retry)
- ❌ Migrations that take >30 seconds (block deploys)
- ❌ Schema changes without testing on production-like data

---

## Azure Deployment Environments

This project has multiple Azure deployment environments:

### Environment 1: Bobby Seamoss (listmonk420)
- **Nickname:** Bobby / listmonk420
- **URL:** https://list.bobbyseamoss.com
- **Resource Group:** `rg-listmonk420`
- **Container App:** `listmonk420`
- **Container Registry:** `listmonk420acr.azurecr.io`
- **Database:** `listmonk420-db.postgres.database.azure.com`
- **Database Name:** `listmonk`
- **Database User:** `listmonkadmin`
- **Database Password:** `T@intshr3dd3r`
- **SSL Mode:** `require`
- **SMTP Servers:** 29 Azure Communication Services domains (mail2-mail30.bobbyseamoss.com)
- **Communication Service:** `comma-email-comm`
- **Email Service:** `comma-email-service`

**Connection String (PostgreSQL):**
```bash
PGPASSWORD='T@intshr3dd3r' psql "sslmode=require host=listmonk420-db.postgres.database.azure.com user=listmonkadmin dbname=listmonk"
```

**Known Issues:**
- **Message-ID Lookup Failures:** The `email_queue` table is missing an `azure_message_id` column, preventing proper correlation between sent emails and Azure delivery webhook events. Webhook logs show "Message-ID lookup failed" errors and fall back to recipient email matching.
- **Domain Reputation:** All mail2-mail30.bobbyseamoss.com domains have low Gmail reputation, causing 11-13% failure rate per domain. Requires domain warming strategy (see Bobby Seamoss Failure Analysis report).

### Environment 2: Comma (enjoycomma)
- **Nickname:** Comma
- **URL:** https://list.enjoycomma.com
- **Resource Group:** `comma-rg`
- **Container App:** `listmonk-comma`
- **Container Registry:** `listmonkcommaacr.azurecr.io`
- **Database:** `listmonk420-db.postgres.database.azure.com` (shared with Bobby)
- **Database Name:** `listmonk_comma` (separate database on shared server)
- **Database User:** `listmonkadmin`
- **Location:** East US (container app), Central US (ACR)

**Note:** When deploying to a specific environment, ensure you're using the correct resource names and credentials for that environment.

---

## Docker Build Guide

### Dockerfile.build Structure

```
Stage 1: email-builder    (Node 20 Alpine)
    ↓ builds email-builder TypeScript/Vue app
Stage 2: frontend-builder (Node 20 Alpine)
    ↓ builds main Vue 2.7 frontend + includes email-builder
Stage 3: backend-builder  (Go 1.24 Alpine)
    ↓ compiles Go binary with frontend embedded
Stage 4: final            (Alpine latest)
    → Minimal runtime image with binary + assets
```

### Key Files Copied to Final Image

```
/listmonk/
├── listmonk              # Go binary
├── config.toml           # From config.toml.docker
├── frontend/dist/        # Built frontend assets
├── static/               # Email templates, public assets
├── i18n/                 # Translation files
├── schema.sql            # Database schema
├── queries.sql           # Named queries
├── permissions.json      # RBAC definitions
├── migrations/           # Migration SQL files
└── deployment/scripts/   # Initialization scripts
```

### Build Locally

```bash
# From project root
docker build -f Dockerfile.build -t listmonk:test .

# Test the image
docker run -it listmonk:test /listmonk/listmonk --version
```

### Common Build Issues

**Frontend not found:**
- Check email-builder built first
- Verify COPY --from=email-builder paths
- Ensure yarn build succeeds

**Binary won't run:**
- Check CGO_ENABLED=0 (for Alpine)
- Verify GOOS=linux
- Check file permissions (chmod +x)

**Large image size:**
- Use multi-stage build (discard build tools)
- Only copy necessary files to final stage
- Use Alpine base images

---

## Azure Deployment Process (deploy.sh)

### Step-by-Step Deployment

```bash
# 1. Build Docker image (multi-stage)
docker build -f Dockerfile.build -t listmonk420acr.azurecr.io/listmonk420:latest .

# 2. Push to Azure Container Registry
az acr login --name listmonk420acr
docker push listmonk420acr.azurecr.io/listmonk420:latest

# 3. Run database migrations (BEFORE deploying container)
export LISTMONK_DB_HOST=listmonk420-db.postgres.database.azure.com
export LISTMONK_DB_PORT=5432
export LISTMONK_DB_USER=listmonkadmin
export LISTMONK_DB_PASSWORD=<password>
export LISTMONK_DB_DATABASE=listmonk

# Build temporary migration runner
docker run --rm -v "$(pwd):/app" -w /app golang:1.24-alpine sh -c \
  "CGO_ENABLED=0 go build -o listmonk-migrate cmd/*.go"

# Run migrations
./listmonk-migrate --upgrade

# 4. Deploy to Azure Container Apps
az containerapp update \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --image listmonk420acr.azurecr.io/listmonk420:latest \
  --set-env-vars "DEPLOY_TIME=$(date +%s)" \
  --revision-suffix "deploy-$(date +%Y%m%d-%H%M%S)"
```

### Environment Variables

**Required for deployment:**
```bash
LISTMONK_DB_HOST          # PostgreSQL host
LISTMONK_DB_PORT          # PostgreSQL port (5432)
LISTMONK_DB_USER          # Database user
LISTMONK_DB_PASSWORD      # Database password
LISTMONK_DB_DATABASE      # Database name
```

**Set in Container App:**
```bash
az containerapp update \
  --set-env-vars \
    "LISTMONK_APP__ADMIN_USERNAME=admin" \
    "LISTMONK_APP__ADMIN_PASSWORD=<password>" \
    "AUTO_INSTALL=true"
```

### Verify Deployment

```bash
# Check latest revision
az containerapp show \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --query "properties.latestRevisionName" \
  -o tsv

# View logs
az containerapp logs show \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --follow

# Test health endpoint
curl https://list.bobbyseamoss.com/api/health
```

### Rollback Procedure

```bash
# 1. List revisions
az containerapp revision list \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --query "[].{name:name, active:properties.active, created:properties.createdTime}" \
  -o table

# 2. Activate previous revision
az containerapp revision activate \
  --revision <previous-revision-name> \
  --resource-group rg-listmonk420

# 3. If migration ran, may need database restore
# Contact DBA or restore from backup
```

---

## deploy.sh Review Checklist

**Before running deploy.sh:**

- [ ] All code changes committed
- [ ] Modified files include backend changes? → Check migrations
- [ ] Modified queries.sql? → Verify syntax
- [ ] Modified schema.sql? → Create matching migration
- [ ] Frontend changes? → Test build locally first
- [ ] New dependencies? → Update package.json/go.mod
- [ ] Database backup taken
- [ ] Team notified
- [ ] Rollback plan ready

**After deployment:**

- [ ] Check container logs for errors
- [ ] Verify health endpoint responds
- [ ] Test key functionality (send campaign, view subscribers)
- [ ] Check database migration applied: `SELECT version FROM migrations ORDER BY version DESC LIMIT 5;`
- [ ] Monitor for 15-30 minutes

---

## Common Deployment Scenarios

### Scenario 1: Code-only changes (no schema changes)

```bash
# Quick deployment - no migrations needed
./deploy.sh
```

### Scenario 2: Schema changes (new table/column)

```bash
# 1. Create migration file (internal/migrations/v6.X.0.go)
# 2. Register in cmd/upgrade.go
# 3. Update schema.sql
# 4. Test locally:
./listmonk --upgrade --config config.dev.toml

# 5. Deploy (migrations run automatically in deploy.sh)
./deploy.sh
```

### Scenario 3: Breaking changes (data migration needed)

```bash
# 1. Take database backup
pg_dump -h listmonk420-db.postgres.database.azure.com \
  -U listmonkadmin listmonk > backup.sql

# 2. Create migration with data backfill
# 3. Test on copy of production data
# 4. Deploy during maintenance window
./deploy.sh

# 5. Verify data integrity
psql -h ... -c "SELECT COUNT(*) FROM table_name;"
```

---

## Testing Migrations Locally

```bash
# 1. Create local test database
createdb listmonk_test

# 2. Run install to get base schema
./listmonk --install --config config.test.toml

# 3. Run your new migration
./listmonk --upgrade --config config.test.toml

# 4. Verify schema changes
psql listmonk_test -c "\d table_name"

# 5. Test idempotency (run again)
./listmonk --upgrade --config config.test.toml
# Should succeed without errors
```

---

## Troubleshooting

### Migration fails on production

```bash
# 1. Check error in container logs
az containerapp logs show --name listmonk420 --resource-group rg-listmonk420 --tail 100

# 2. Connect to database and check state
psql -h listmonk420-db.postgres.database.azure.com -U listmonkadmin -d listmonk

# 3. Check migrations table
SELECT * FROM migrations ORDER BY version DESC;

# 4. If partial migration, may need manual cleanup
# Then re-run: ./listmonk --upgrade
```

### Container won't start after deployment

```bash
# 1. Check revision status
az containerapp revision list ...

# 2. View container logs
az containerapp logs show --tail 200

# 3. Common issues:
# - Missing env vars
# - Migration failed
# - Config syntax error
# - Health check failing

# 4. Rollback to previous revision
az containerapp revision activate --revision <previous-name>
```

### Docker build fails

```bash
# Check which stage failed
docker build -f Dockerfile.build -t listmonk:test . --progress=plain

# Common causes:
# - Frontend build errors → Check frontend/yarn.lock
# - Go compile errors → Check go.mod, syntax errors
# - Missing files → Check COPY paths
# - Out of disk space → docker system prune
```

---

## Best Practices Summary

### ✅ DO

- Use idempotent migrations (IF NOT EXISTS)
- Test migrations on dev database first
- Update schema.sql after migrations
- Take database backups before schema changes
- Use unique revision suffixes for deployments
- Monitor logs after deployment
- Document rollback procedures

### ❌ DON'T

- Drop tables/columns without confirmation
- Run migrations manually on production
- Deploy without testing build locally
- Skip updating schema.sql
- Forget to register migrations in migList
- Deploy during peak traffic hours
- Leave commented-out code in migrations

---

## Quick Reference

### Files to Touch

**For database changes:**
- `internal/migrations/v{version}.go` - Create migration
- `cmd/upgrade.go` - Register migration
- `schema.sql` - Update final schema
- `queries.sql` - Add/update queries if needed

**For deployment:**
- `deploy.sh` - Main deployment script
- `Dockerfile.build` - Multi-stage build
- `docker-entrypoint.sh` - Container startup
- `config.toml.docker` - Container config

### Key Commands

```bash
# Local development
./listmonk --install              # Fresh install
./listmonk --upgrade              # Run migrations
make build                        # Build backend
make build-frontend               # Build frontend

# Docker
docker build -f Dockerfile.build -t listmonk:test .
docker run -it listmonk:test sh   # Debug container

# Azure
az acr login --name listmonk420acr
az containerapp logs show --name listmonk420 --resource-group rg-listmonk420 --follow
az containerapp revision list --name listmonk420 --resource-group rg-listmonk420

# Database
psql -h listmonk420-db.postgres.database.azure.com -U listmonkadmin -d listmonk
```

---

## Example: Adding a New Feature with Database Changes

**Scenario:** Add email open tracking

### Step 1: Create Migration

```go
// internal/migrations/v6.3.0.go
package migrations

func v6_3_0(db *sqlx.DB, fs stuffbin.FileSystem, prompt bool) error {
    _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS email_opens (
            id BIGSERIAL PRIMARY KEY,
            campaign_id INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
            subscriber_id INTEGER NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE,
            opened_at TIMESTAMP NOT NULL DEFAULT NOW(),
            ip_address INET,
            user_agent TEXT
        );

        CREATE INDEX IF NOT EXISTS idx_email_opens_campaign
            ON email_opens(campaign_id, opened_at DESC);

        CREATE INDEX IF NOT EXISTS idx_email_opens_subscriber
            ON email_opens(subscriber_id, opened_at DESC);
    `)
    return err
}
```

### Step 2: Register Migration

```go
// cmd/upgrade.go
var migList = []migration{
    // ... existing
    {ver: "v6.3.0", upgradeFunc: migrations.V6_3_0},
}
```

### Step 3: Update schema.sql

```sql
-- schema.sql (add at appropriate location)
CREATE TABLE email_opens (
    id BIGSERIAL PRIMARY KEY,
    campaign_id INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    subscriber_id INTEGER NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE,
    opened_at TIMESTAMP NOT NULL DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT
);

CREATE INDEX idx_email_opens_campaign ON email_opens(campaign_id, opened_at DESC);
CREATE INDEX idx_email_opens_subscriber ON email_opens(subscriber_id, opened_at DESC);
```

### Step 4: Add Queries

```sql
-- queries.sql
-- name: record-email-open
INSERT INTO email_opens (campaign_id, subscriber_id, ip_address, user_agent)
VALUES ($1, $2, $3, $4);

-- name: get-campaign-opens
SELECT COUNT(*) as open_count
FROM email_opens
WHERE campaign_id = $1;
```

### Step 5: Test Locally

```bash
# Test migration
./listmonk --upgrade

# Test idempotency (run twice)
./listmonk --upgrade

# Verify schema
psql listmonk_dev -c "\d email_opens"
```

### Step 6: Deploy

```bash
# Full deployment with migration
./deploy.sh

# Verify migration applied
PGPASSWORD='...' psql -h listmonk420-db.postgres.database.azure.com \
  -U listmonkadmin -d listmonk \
  -c "SELECT version FROM migrations ORDER BY version DESC LIMIT 5;"
```

---

## Additional Resources

- **listmonk docs:** https://listmonk.app/docs
- **PostgreSQL migration guide:** https://www.postgresql.org/docs/current/ddl-alter.html
- **Azure Container Apps:** https://learn.microsoft.com/en-us/azure/container-apps/
- **Docker multi-stage builds:** https://docs.docker.com/build/building/multi-stage/

---

## Support

If deployment fails or you need help:

1. Check container logs first
2. Verify database connection
3. Review recent commits for breaking changes
4. Check Azure Portal for service health
5. Consider rolling back to previous revision

**Remember:** It's better to pause and debug than to rush a deployment that causes downtime.
