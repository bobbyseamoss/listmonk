# Listmonk Azure Deployment Guide
## Configuration-as-Code with Azure Key Vault

This guide walks through deploying Listmonk to Azure Container Apps with SMTP and bounce mailbox configurations stored securely in Azure Key Vault.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Architecture Overview](#architecture-overview)
3. [Initial Setup (One-Time)](#initial-setup-one-time)
4. [Configuration Export](#configuration-export)
5. [Azure Key Vault Setup](#azure-key-vault-setup)
6. [Container App Configuration](#container-app-configuration)
7. [Deployment](#deployment)
8. [Verification](#verification)
9. [Credential Rotation](#credential-rotation)
10. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Required Tools
- **Azure CLI** (version 2.30.0 or later)
- **Docker** (for building container images)
- **jq** (for JSON processing)
- **PostgreSQL client** (psql)
- **Python 3** (for configuration export)
- **Node.js** and **npm** (for frontend build)
- **Go 1.21+** (for backend build)

### Required Access
- Azure subscription with appropriate permissions
- Access to Azure Container Registry (listmonk420acr)
- Access to Azure Container Apps (listmonk420)
- Access to Azure Database for PostgreSQL (listmonk420-db)

### Verify Installation
```bash
az --version
docker --version
jq --version
psql --version
python3 --version
node --version
go version
```

---

## Architecture Overview

### Components
```
┌───────────────────────────────────────────────────────┐
│ Azure Container Apps                                   │
│ ┌───────────────────────────────────────────────────┐ │
│ │ listmonk container                                │ │
│ │ - Reads config templates                          │ │
│ │ - Fetches credentials from env vars               │ │
│ │ - (Env vars populated from Azure Key Vault)       │ │
│ └───────────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────────┘
                           │
                           │ Secure connection
                           ▼
┌───────────────────────────────────────────────────────┐
│ Azure Database for PostgreSQL (listmonk420-db)        │
│ - Schema initialized via listmonk --install          │
│ - Configuration imported via init_config.sh           │
└───────────────────────────────────────────────────────┘
                           │
                           │ Credentials fetched
                           ▼
┌───────────────────────────────────────────────────────┐
│ Azure Key Vault (listmonk420-kv)                      │
│ - Stores 118 credentials (58 SMTP + 60 bounce)        │
│ - Accessed via managed identity                       │
└───────────────────────────────────────────────────────┘
```

### Security Model
- **Secrets**: Stored in Azure Key Vault (NOT in container images or git)
- **Access**: Container Apps use managed identity to access Key Vault
- **Configuration**: Structure stored in version control, credentials injected at runtime
- **Separation**: Clean separation between code, config, and secrets

---

## Initial Setup (One-Time)

### 1. Login to Azure
```bash
az login
az account set --subscription "YOUR_SUBSCRIPTION_ID"
```

### 2. Verify Resource Group
```bash
az group show --name rg-listmonk420
```

### 3. Verify Database Access
```bash
az postgres flexible-server show \
  --name listmonk420-db \
  --resource-group rg-listmonk420
```

---

## Configuration Export

### Overview
Extract SMTP and bounce configurations from your development database and sanitize them (remove credentials).

### Steps

1. **Ensure Development Environment is Running**
   ```bash
   cd /home/adam/listmonk
   make dev-docker
   ```

2. **Run Configuration Export Script**
   ```bash
   python3 deployment/scripts/export_configs.py
   ```

### Output Files
The script generates these files in `deployment/configs/`:

- **`smtp_config_template.json`**: 29 SMTP servers (sanitized)
- **`bounce_config_template.json`**: 30 bounce mailboxes (sanitized)
- **`app_settings.json`**: Performance settings
- **`credential_map.json`**: ⚠️ **118 PLAINTEXT CREDENTIALS** - Use ONLY for Key Vault upload, then DELETE

### Example Output
```
======================================================================
Export Summary
======================================================================
✓ SMTP servers: 29
✓ Bounce mailboxes: 30
✓ App settings: exported
✓ Credentials: 118

⚠️  WARNING: credential_map.json contains plaintext credentials!
⚠️  DO NOT commit to git! Use only for Key Vault upload, then delete.
```

### Security Checklist
- [ ] Review generated templates for accuracy
- [ ] **DO NOT commit credential_map.json to git**
- [ ] Add `credential_map.json` to `.gitignore`
- [ ] Delete `credential_map.json` after uploading to Key Vault

---

## Azure Key Vault Setup

### Overview
Upload all 118 credentials to Azure Key Vault for secure storage.

### Steps

1. **Review Credential Mapping**
   ```bash
   # Count credentials
   jq 'length' deployment/configs/credential_map.json
   # Should output: 118

   # Preview first credential (DO NOT share output)
   jq 'to_entries | .[0]' deployment/configs/credential_map.json
   ```

2. **Run Upload Script**
   ```bash
   cd deployment/scripts
   ./upload_credentials.sh
   ```

3. **Verify Upload**
   The script will output:
   ```
   ======================================================================
   Upload Summary
   ======================================================================
   ✓ Uploaded: 118
   ○ Skipped (already exists): 0
   ✗ Failed: 0
   ```

4. **Delete Credential File**
   ```bash
   # IMPORTANT: Delete after successful upload
   rm deployment/configs/credential_map.json
   ```

5. **Verify in Azure**
   ```bash
   # List all secrets
   az keyvault secret list \
     --vault-name listmonk420-kv \
     --query "[].{name:name}" \
     -o table

   # Should show 118 secrets:
   # smtp-1-username, smtp-1-password
   # smtp-2-username, smtp-2-password
   # ...
   # bounce-30-username, bounce-30-password
   ```

### Troubleshooting
- **Key Vault doesn't exist**: Script creates it automatically
- **Permission denied**: Ensure you have Key Vault Contributor role
- **Secrets already exist**: Script skips existing secrets (idempotent)

---

## Container App Configuration

### Overview
Configure Azure Container App to access Key Vault secrets and enable auto-initialization.

### Step 1: Enable Managed Identity
```bash
az containerapp identity assign \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --system-assigned
```

### Step 2: Get Identity Principal ID
```bash
IDENTITY_ID=$(az containerapp show \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --query identity.principalId \
  -o tsv)

echo "Managed Identity ID: $IDENTITY_ID"
```

### Step 3: Grant Key Vault Access
```bash
az keyvault set-policy \
  --name listmonk420-kv \
  --object-id $IDENTITY_ID \
  --secret-permissions get list
```

### Step 4: Configure Environment Variables
```bash
# Set AUTO_INSTALL to trigger initialization on first run
az containerapp update \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --set-env-vars \
    "AUTO_INSTALL=true" \
    "LISTMONK_ADMIN_USER=admin" \
    "LISTMONK_ADMIN_PASSWORD=YOUR_SECURE_PASSWORD_HERE"
```

⚠️ **Replace** `YOUR_SECURE_PASSWORD_HERE` with a strong password

### Step 5: Map Credentials from Key Vault to Environment Variables

Due to the large number of credentials (118), we'll create a script to generate the secret mappings:

```bash
# Create temporary script
cat > /tmp/add_secrets.sh <<'EOF'
#!/bin/bash
VAULT_NAME="listmonk420-kv"
APP_NAME="listmonk420"
RG="rg-listmonk420"

# Generate secret references for all 118 credentials
SECRETS=""

# SMTP credentials (1-29)
for i in {1..29}; do
  SECRETS="$SECRETS smtp-${i}-username=keyvaultref:https://${VAULT_NAME}.vault.azure.net/secrets/smtp-${i}-username,identityref:system"
  SECRETS="$SECRETS smtp-${i}-password=keyvaultref:https://${VAULT_NAME}.vault.azure.net/secrets/smtp-${i}-password,identityref:system"
done

# Bounce credentials (1-30)
for i in {1..30}; do
  SECRETS="$SECRETS bounce-${i}-username=keyvaultref:https://${VAULT_NAME}.vault.azure.net/secrets/bounce-${i}-username,identityref:system"
  SECRETS="$SECRETS bounce-${i}-password=keyvaultref:https://${VAULT_NAME}.vault.azure.net/secrets/bounce-${i}-password,identityref:system"
done

# Apply secrets
az containerapp secret set \
  --name $APP_NAME \
  --resource-group $RG \
  --secrets $SECRETS

echo "✓ All 118 credentials configured"
EOF

chmod +x /tmp/add_secrets.sh
/tmp/add_secrets.sh
```

⚠️ **Note**: This step may take 5-10 minutes due to the large number of secrets.

### Verification
```bash
# Check configured secrets (should show 118)
az containerapp secret list \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --query "length(@)"
```

---

## Deployment

### Overview
Build and deploy the application to Azure Container Apps.

### Steps

1. **Run Deployment Script**
   ```bash
   cd /home/adam/listmonk
   ./deploy.sh
   ```

2. **Monitor Deployment**
   The script will:
   - Build email-builder
   - Build frontend
   - Build Go binary
   - Build Docker image
   - Push to Azure Container Registry
   - Update Container App
   - Check if first deployment
   - Provide initialization instructions if needed

3. **First Deployment**
   If this is your first deployment, the script will pause and display:
   ```
   === ACTION REQUIRED ===
   This appears to be your first deployment. Complete these steps:

   1. Upload credentials to Azure Key Vault
   2. Enable AUTO_INSTALL environment variable
   3. Configure Container App secrets from Key Vault
   4. Restart the container to trigger initialization
   ```

4. **Trigger Initialization**
   After configuration, restart the container:
   ```bash
   az containerapp revision restart \
     --name listmonk420 \
     --resource-group rg-listmonk420 \
     --revision $(az containerapp show \
       --name listmonk420 \
       --resource-group rg-listmonk420 \
       --query properties.latestRevisionName \
       -o tsv)
   ```

5. **Monitor Initialization**
   ```bash
   az containerapp logs show \
     --name listmonk420 \
     --resource-group rg-listmonk420 \
     --follow
   ```

   Look for:
   ```
   AUTO_INSTALL enabled - checking database initialization...
   Database not initialized. Running installation...
   ✓ Database initialized
   Running configuration initialization...
   Processing SMTP configuration...
     Found 29 SMTP servers in template
     Injecting credentials...
     Inserting into database...
   ✓ SMTP configuration saved
   Processing bounce configuration...
     Found 30 bounce mailboxes in template
     Injecting credentials...
     Inserting into database...
   ✓ Bounce configuration saved
   ✓ Configuration initialized
   ```

---

## Verification

### 1. Check Application Health
```bash
curl https://list.bobbyseamoss.com/api/health
# Should return: {"data": "OK"}
```

### 2. Access Admin Panel
```
URL: https://list.bobbyseamoss.com/admin
Username: admin
Password: (the password you set in Step 4 of Container App Configuration)
```

### 3. Verify SMTP Configuration
1. Navigate to **Settings > SMTP**
2. Verify 29 SMTP servers are configured
3. Check daily limits and from_email addresses
4. Verify bounce mailbox assignments

### 4. Verify Bounce Configuration
1. Navigate to **Settings > Bounces**
2. Verify 30 bounce mailboxes are configured
3. Check scan intervals and host addresses

### 5. Test Queue System
1. Create a test campaign
2. Select "automatic (queue-based)" messenger
3. Start campaign
4. Navigate to **Queue** page
5. Verify emails are:
   - Assigned to different SMTP servers
   - Scheduled with staggered times
   - Respecting sliding window limits (4 emails per 30 min per server)

---

## Credential Rotation

### Overview
Periodically rotate SMTP and bounce mailbox passwords for security.

### When to Rotate
- Every 90 days (recommended)
- After suspected credential compromise
- When removing team members with credential access

### Steps

1. **Update Credentials in Source Systems**
   - Update passwords in Azure Communication Services
   - Update passwords in bounce mailbox providers (Office 365, etc.)

2. **Update Azure Key Vault**
   ```bash
   # Update a single credential
   az keyvault secret set \
     --vault-name listmonk420-kv \
     --name smtp-1-password \
     --value "NEW_PASSWORD_HERE"

   # Or bulk update using credential_map.json (if regenerated)
   cd deployment/scripts
   ./upload_credentials.sh  # Automatically updates existing secrets
   ```

3. **Trigger Configuration Refresh**
   ```bash
   # Restart container to reload credentials
   az containerapp restart \
     --name listmonk420 \
     --resource-group rg-listmonk420
   ```

4. **Verify**
   Check logs for:
   ```
   ✓ SMTP configuration saved
   ✓ Bounce configuration saved
   ```

### Automated Rotation Script
A future enhancement could use Azure Functions to:
- Monitor Key Vault for credential age
- Trigger automatic rotation
- Update listmonk configuration
- Send notifications

---

## Troubleshooting

### Issue: "Database not initialized" Error

**Symptoms**: Container logs show repeated initialization attempts

**Causes**:
- Database connection failed
- Database credentials incorrect
- Network connectivity issues

**Solution**:
```bash
# Check database connection
az postgres flexible-server show \
  --name listmonk420-db \
  --resource-group rg-listmonk420

# Verify database firewall allows Container App
az postgres flexible-server firewall-rule list \
  --name listmonk420-db \
  --resource-group rg-listmonk420

# Check Container App environment variables
az containerapp show \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --query "properties.template.containers[0].env[]"
```

---

### Issue: "Credentials not found" Warnings

**Symptoms**: Logs show `⚠️ Warning: Credentials not found for SMTP server X`

**Causes**:
- Secret not uploaded to Key Vault
- Secret name mismatch
- Managed identity doesn't have Key Vault access
- Environment variable not mapped

**Solution**:
```bash
# 1. Verify secret exists in Key Vault
az keyvault secret show \
  --vault-name listmonk420-kv \
  --name smtp-1-username

# 2. Verify Container App has secret mapped
az containerapp secret list \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --query "[?contains(name, 'smtp-1')]"

# 3. Check managed identity has access
IDENTITY_ID=$(az containerapp show \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --query identity.principalId \
  -o tsv)

az keyvault show \
  --name listmonk420-kv \
  --query "properties.accessPolicies[?objectId=='$IDENTITY_ID']"

# 4. Re-map secrets if needed
# (Use script from Container App Configuration section)
```

---

### Issue: SMTP Servers Not Sending

**Symptoms**: Emails stuck in "queued" status, no sends happening

**Causes**:
- Credentials incorrect in Key Vault
- SMTP server configuration wrong (host, port, TLS)
- Daily limits reached
- Testing mode enabled

**Solution**:
```bash
# 1. Check queue processor logs
az containerapp logs show \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --follow | grep "queue processor"

# 2. Test SMTP connection from container
az containerapp exec \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --command "sh"

# Inside container:
telnet smtp.azurecomm.net 587

# 3. Verify credentials work
# Use a test SMTP client with credentials from Key Vault

# 4. Check daily usage
# Navigate to Settings > SMTP in admin panel
# View "Daily Used" column
```

---

### Issue: Bounce Mailboxes Not Scanning

**Symptoms**: Bounces not being processed, no scan logs

**Causes**:
- Credentials incorrect
- POP server unreachable
- Scan interval too long
- Mailbox doesn't exist

**Solution**:
```bash
# 1. Check bounce manager logs
az containerapp logs show \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --follow | grep "bounce"

# 2. Test POP connection
az containerapp exec \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --command "sh"

# Inside container:
telnet outlook.office365.com 995

# 3. Manually trigger scan
# Navigate to Settings > Bounces in admin panel
# Click "Test" on a mailbox
```

---

### Issue: Container App Out of Memory

**Symptoms**: Container restarts frequently, OOM errors in logs

**Causes**:
- Too many concurrent connections
- Memory leak
- Insufficient container resources

**Solution**:
```bash
# 1. Check current resource allocation
az containerapp show \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --query "properties.template.containers[0].resources"

# 2. Increase memory allocation
az containerapp update \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --cpu 1.0 \
  --memory 2Gi

# 3. Monitor memory usage
az monitor metrics list \
  --resource $(az containerapp show \
    --name listmonk420 \
    --resource-group rg-listmonk420 \
    --query id -o tsv) \
  --metric "MemoryUsagePercentage"
```

---

## Additional Resources

### Documentation
- [Listmonk Official Docs](https://listmonk.app/docs)
- [Azure Container Apps Docs](https://learn.microsoft.com/en-us/azure/container-apps/)
- [Azure Key Vault Docs](https://learn.microsoft.com/en-us/azure/key-vault/)

### Support
- [Listmonk GitHub Issues](https://github.com/knadh/listmonk/issues)
- [Azure Support](https://azure.microsoft.com/en-us/support/)

### Security
- [Azure Security Baseline for Container Apps](https://learn.microsoft.com/en-us/security/benchmark/azure/baselines/container-apps-security-baseline)
- [Key Vault Security Recommendations](https://learn.microsoft.com/en-us/azure/key-vault/general/security-recommendations)

---

## Changelog

### 2025-10-29 - Initial Version
- Created comprehensive deployment guide
- Documented configuration-as-code approach
- Added Azure Key Vault integration
- Included troubleshooting procedures
- Added credential rotation guide

---

## Contact

For questions or issues with this deployment:
- Review logs: `az containerapp logs show --name listmonk420 --resource-group rg-listmonk420 --follow`
- Check configuration: Review `deployment/configs/*.json` files
- Verify secrets: Use Azure Portal to inspect Key Vault secrets
