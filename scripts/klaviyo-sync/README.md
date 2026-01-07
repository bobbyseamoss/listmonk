# Klaviyo → listmonk Sync

One-way sync from Klaviyo to listmonk, running as an Azure Container App Job on an hourly schedule.

## What it does

1. **Syncs Klaviyo Segments → listmonk Lists**
   - Creates new lists prefixed with "Klaviyo: " (e.g., "Klaviyo: Engaged Customers")
   - Upserts subscribers and adds them to the corresponding list

2. **Syncs Klaviyo Suppressions → listmonk Blocklist**
   - Marks suppressed profiles as blocklisted in listmonk

## Prerequisites

1. Azure CLI installed and logged in (`az login`)
2. Docker installed
3. Klaviyo Private API Key

## Getting Your Klaviyo API Key

1. Go to Klaviyo → Settings → API Keys
2. Create a new Private API Key
3. Required scopes: `segments:read`, `profiles:read`

## Deployment

```bash
# Set your Klaviyo API key
export KLAVIYO_API_KEY="pk_xxxxxxxxxxxxxxxx"

# Deploy to Azure
./deploy-azure.sh
```

This will:
1. Create an Azure Container Registry (if needed)
2. Build and push the Docker image
3. Create a Container App Job that runs every hour

## Manual Testing

### Dry Run (local)
```bash
export KLAVIYO_API_KEY="pk_xxxxxxxxxxxxxxxx"
export DB_PASSWORD="T@intshr3dd3r"
export DRY_RUN="true"
python sync.py
```

### Trigger Job Manually
```bash
az containerapp job start --name klaviyo-sync --resource-group rg-listmonk420
```

### View Logs
```bash
az containerapp job logs show --name klaviyo-sync --resource-group rg-listmonk420
```

### View Execution History
```bash
az containerapp job execution list --name klaviyo-sync --resource-group rg-listmonk420 -o table
```

## Configuration

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `KLAVIYO_API_KEY` | Klaviyo Private API Key | Required |
| `DB_HOST` | PostgreSQL host | listmonk420-db.postgres.database.azure.com |
| `DB_PORT` | PostgreSQL port | 5432 |
| `DB_NAME` | Database name | listmonk |
| `DB_USER` | Database user | listmonkadmin |
| `DB_PASSWORD` | Database password | Required |
| `DB_SSLMODE` | SSL mode | require |
| `LIST_PREFIX` | Prefix for created lists | "Klaviyo: " |
| `DRY_RUN` | Log actions without making changes | false |
| `BATCH_SIZE` | Subscribers per batch | 1000 |

## Schedule

The job runs every hour at minute 0 (cron: `0 * * * *`).

To change the schedule:
```bash
az containerapp job update \
    --name klaviyo-sync \
    --resource-group rg-listmonk420 \
    --cron-expression "0 */2 * * *"  # Every 2 hours
```
