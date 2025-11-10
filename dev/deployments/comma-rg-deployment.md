# comma-rg Deployment Documentation

**Resource Group**: comma-rg
**Database**: listmonk_comma
**Created**: November 8, 2025
**Status**: Active

## Overview

The comma-rg deployment is a separate instance of listmonk running in its own Azure Resource Group, but sharing the same PostgreSQL server as the main production deployment.

## Infrastructure

### Resource Group: comma-rg
**Location**: East US

### Resources

1. **Container App**: listmonk-comma
   - FQDN: `listmonk-comma.salmonsea-6f7b784e.eastus.azurecontainerapps.io`
   - Admin URL: `https://listmonk-comma.salmonsea-6f7b784e.eastus.azurecontainerapps.io/admin`
   - Environment: listmonk-comma-env

2. **Container Registry**: listmonkcommaacr.azurecr.io
   - Location: Central US
   - SKU: Basic
   - Admin Enabled: true
   - Image: listmonk-comma:latest

3. **PostgreSQL Database**: listmonk_comma
   - Server: listmonk420-db.postgres.database.azure.com (shared with production)
   - Port: 5432
   - User: listmonkadmin
   - Database: listmonk_comma (separate from production "listmonk" database)

4. **Log Analytics Workspace**: workspace-commarg0Tt6
   - For monitoring and application logs

## Database Architecture

**Important**: The comma-rg deployment uses a **multi-tenant architecture**:
- Same PostgreSQL server as production (listmonk420-db)
- Separate database (listmonk_comma) for complete data isolation
- Independent schema versioning via migrations table
- No data sharing between production and comma-rg

**Advantages**:
- Cost-effective (share PostgreSQL infrastructure)
- Independent data (separate databases)
- Easy backup/restore
- Simplified network security

## Environment Variables

```bash
LISTMONK_DB_HOST=listmonk420-db.postgres.database.azure.com
LISTMONK_DB_PORT=5432
LISTMONK_DB_USER=listmonkadmin
LISTMONK_DB_PASSWORD=T@intshr3dd3r
LISTMONK_DB_DATABASE=listmonk_comma
LISTMONK_APP__ADMIN_USERNAME=admin
LISTMONK_APP__ADMIN_PASSWORD=T@intshr3dd3r
AUTO_INSTALL=true
```

## Deployment Process

### Prerequisites

1. **Set environment variable**:
   ```bash
   export LISTMONK_DB_PASSWORD='T@intshr3dd3r'
   ```

2. **Verify Azure CLI is logged in**:
   ```bash
   az account show
   ```

### Deploy to comma-rg

From the project root:

```bash
./deploy-comma.sh
```

The script will:
1. Build Docker image (multi-stage build)
2. Push to listmonkcommaacr registry
3. Check current database schema version
4. Run database migrations (v6.0.0 → v7.0.0)
5. Deploy to listmonk-comma container app
6. Create new revision with timestamp
7. Verify deployment health

### Deployment Script Features

- **Automatic migration detection**: Checks current schema version before deploying
- **Idempotent migrations**: Safe to run multiple times
- **Health checks**: Verifies app is running after deployment
- **Database verification**: Confirms connection and schema version
- **Detailed logging**: Shows all migration steps
- **Color-coded output**: Easy to spot errors or warnings

## Schema Migrations

The comma-rg deployment includes all migrations:

- **v6.0.0**: Queue-based email delivery system
  - email_queue table
  - smtp_daily_usage table
  - smtp_rate_limit_state table

- **v6.1.0**: Azure message tracking
  - azure_message_tracking table

- **v6.2.0**: Azure delivery/engagement events
  - azure_delivery_events table
  - azure_engagement_events table

- **v7.0.0**: Shopify purchase attribution
  - purchase_attributions table

### Verify Migrations

```bash
PGPASSWORD='T@intshr3dd3r' psql \
  -h listmonk420-db.postgres.database.azure.com \
  -U listmonkadmin \
  -d listmonk_comma \
  -c "SELECT * FROM migrations ORDER BY version DESC;"
```

## Features Deployed

All current production features:
- ✅ Queue-based email delivery with daily limits
- ✅ Azure Event Grid webhook integration
- ✅ Message tracking and delivery events
- ✅ Shopify purchase attribution
- ✅ Enhanced campaigns page with metrics
- ✅ Open/Click Rate columns with counts
- ✅ Multi-SMTP server support (30+)
- ✅ Bounce mailbox correlation

## Post-Deployment Verification

### 1. Check Health Endpoint

```bash
curl https://listmonk-comma.salmonsea-6f7b784e.eastus.azurecontainerapps.io/api/health
```

**Expected**: `{"data": "OK"}`

### 2. Test Admin Login

1. Navigate to: https://listmonk-comma.salmonsea-6f7b784e.eastus.azurecontainerapps.io/admin
2. Login with:
   - Username: admin
   - Password: T@intshr3dd3r

### 3. Verify Database Schema

```bash
PGPASSWORD='T@intshr3dd3r' psql \
  -h listmonk420-db.postgres.database.azure.com \
  -U listmonkadmin \
  -d listmonk_comma \
  -c "SELECT version FROM migrations ORDER BY version DESC LIMIT 5;"
```

**Expected**: Should show v7.0.0 as latest

### 4. Check Container Logs

```bash
az containerapp logs show \
  --name listmonk-comma \
  --resource-group comma-rg \
  --follow
```

Look for:
- ✅ "INFO: Starting listmonk..."
- ✅ "INFO: Database connection established"
- ✅ No migration errors
- ❌ Any ERROR or FATAL messages

### 5. Test Email Campaign (Optional)

1. Add test subscriber
2. Create simple campaign
3. Send to test subscriber
4. Verify in azure_message_tracking table:

```bash
PGPASSWORD='T@intshr3dd3r' psql \
  -h listmonk420-db.postgres.database.azure.com \
  -U listmonkadmin \
  -d listmonk_comma \
  -c "SELECT * FROM azure_message_tracking ORDER BY sent_at DESC LIMIT 5;"
```

## Monitoring

### View Container Logs

```bash
# Tail logs in real-time
az containerapp logs show -n listmonk-comma -g comma-rg --follow

# Get last 100 lines
az containerapp logs show -n listmonk-comma -g comma-rg --tail 100

# Filter for errors
az containerapp logs show -n listmonk-comma -g comma-rg --tail 200 | grep -i error
```

### Check Revisions

```bash
az containerapp revision list \
  --name listmonk-comma \
  --resource-group comma-rg \
  --query "[].{name:name, active:properties.active, created:properties.createdTime}" \
  -o table
```

## Rollback Procedure

If deployment fails:

### 1. Activate Previous Revision

```bash
# List revisions
az containerapp revision list \
  --name listmonk-comma \
  --resource-group comma-rg \
  -o table

# Activate previous revision
az containerapp revision activate \
  --revision <previous-revision-name> \
  --resource-group comma-rg
```

### 2. Check for Database Migration Issues

If migrations failed partially, you may need to:

```bash
# Connect to database
PGPASSWORD='T@intshr3dd3r' psql \
  -h listmonk420-db.postgres.database.azure.com \
  -U listmonkadmin \
  -d listmonk_comma

# Check migration status
SELECT * FROM migrations ORDER BY version DESC;

# If migrations table is corrupted, may need manual intervention
# Contact DBA or refer to migration troubleshooting guide
```

## Differences from Production

### Similarities
- Same codebase and features
- Same PostgreSQL server (different database)
- Same deployment process
- Same migration system

### Differences
- **Resource Group**: comma-rg (vs rg-listmonk420)
- **Container App**: listmonk-comma (vs listmonk420)
- **ACR**: listmonkcommaacr (vs listmonk420acr)
- **Database**: listmonk_comma (vs listmonk)
- **URL**: Default Azure FQDN (no custom domain mapped)
- **Purpose**: Testing/staging environment

## Maintenance

### Regular Updates

To deploy new features to comma-rg:

```bash
# 1. Set password
export LISTMONK_DB_PASSWORD='T@intshr3dd3r'

# 2. Run deployment
./deploy-comma.sh

# 3. Verify
curl https://listmonk-comma.salmonsea-6f7b784e.eastus.azurecontainerapps.io/api/health
```

### Database Backups

The listmonk_comma database is backed up automatically as part of the Azure PostgreSQL Flexible Server backup policy.

Manual backup:
```bash
pg_dump -h listmonk420-db.postgres.database.azure.com \
  -U listmonkadmin \
  -d listmonk_comma \
  -F c \
  -f listmonk_comma_backup_$(date +%Y%m%d).dump
```

Restore:
```bash
pg_restore -h listmonk420-db.postgres.database.azure.com \
  -U listmonkadmin \
  -d listmonk_comma_new \
  -c \
  listmonk_comma_backup_20251108.dump
```

## Troubleshooting

### Container Won't Start

1. **Check logs**:
   ```bash
   az containerapp logs show -n listmonk-comma -g comma-rg --tail 200
   ```

2. **Common issues**:
   - Database connection failure → Check env vars
   - Migration error → Check migrations table
   - Image pull error → Verify ACR authentication

### Database Connection Issues

1. **Test connection**:
   ```bash
   PGPASSWORD='T@intshr3dd3r' psql \
     -h listmonk420-db.postgres.database.azure.com \
     -U listmonkadmin \
     -d listmonk_comma \
     -c "SELECT version();"
   ```

2. **Check firewall rules**:
   - Azure Container Apps IP may need to be whitelisted
   - Verify "Allow Azure services" is enabled on PostgreSQL

### Migration Failures

1. **Check migration version**:
   ```bash
   PGPASSWORD='T@intshr3dd3r' psql ... -c "SELECT * FROM migrations ORDER BY version DESC LIMIT 10;"
   ```

2. **Run migrations manually**:
   ```bash
   export LISTMONK_DB_PASSWORD='T@intshr3dd3r'
   export LISTMONK_DB_HOST=listmonk420-db.postgres.database.azure.com
   export LISTMONK_DB_DATABASE=listmonk_comma

   # Build and run migration
   go build -o listmonk cmd/*.go
   ./listmonk --upgrade
   ```

## Cost Optimization

The comma-rg deployment is cost-effective because:
- Shares PostgreSQL server with production (no additional DB cost)
- Uses Basic ACR tier ($5/month)
- Container App scales to zero when idle
- Minimal Log Analytics costs

**Estimated Monthly Cost**: ~$10-15 (mostly Container App runtime)

## Security

### Secrets Management

Current: Environment variables store secrets directly
**Recommended**: Migrate to Azure Key Vault

```bash
# Create Key Vault reference
az containerapp secret set \
  --name listmonk-comma \
  --resource-group comma-rg \
  --secrets db-password=keyvaultref:<vault-uri>
```

### Network Security

- Container App uses HTTPS by default
- PostgreSQL requires SSL connection
- No public IP exposure (fully managed platform)

## Support

For issues with comma-rg deployment:

1. Check this documentation first
2. Review container logs
3. Verify database connectivity
4. Compare with production deployment (rg-listmonk420)
5. Consult /home/adam/listmonk/dev/active/ documentation

---

**Last Updated**: November 8, 2025
**Deployment Script**: `/home/adam/listmonk/deploy-comma.sh`
**Azure Portal**: Search for "comma-rg" resource group
