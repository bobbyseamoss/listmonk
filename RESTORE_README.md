# Listmonk Server Configuration Restore

Quick restore system for your 30 SMTP and 30 POP bounce servers.

## Files Created

- `restore_servers.sql` - SQL template with all 30 servers
- `restore_config.sh` - Interactive helper script
- `RESTORE_README.md` - This file

## Quick Start

### 1. Customize the Template (First Time Only)

Edit `restore_servers.sql` and replace:

```bash
# Find and replace YOUR_PASSWORD_HERE with actual passwords
sed -i 's/YOUR_PASSWORD_HERE/ActualPassword123/g' restore_servers.sql
```

Or manually edit the file and update:
- `"password": "YOUR_PASSWORD_HERE"` → Your actual SMTP passwords
- `"daily_limit": 1000` → Adjust per server if needed
- `"scan_interval": "15m"` → Adjust bounce checking frequency

### 2. Restore Configuration

**Option A: Interactive Menu (Recommended)**
```bash
./restore_config.sh
# Select option 5: "Restore with backup first"
```

**Option B: Direct SQL Restore**
```bash
PGPASSWORD='Listmonk420Pass!' psql \
  -h listmonk420-db.postgres.database.azure.com \
  -U listmonkadmin \
  -d listmonk \
  -f restore_servers.sql
```

## What Gets Configured

### SMTP Servers (30 total)
- **Servers**: mail2.bobbyseamoss.com through mail31.bobbyseamoss.com
- **From addresses**: adam@mail2.bobbyseamoss.com through adam@mail31.bobbyseamoss.com
- **Daily limit**: 1000 emails per server (30,000 total capacity)
- **Port**: 587 (STARTTLS)
- **Features**: Queue system with daily limits and automatic rotation

### Bounce Mailboxes (30 total)
- **Servers**: mail2.bobbyseamoss.com through mail31.bobbyseamoss.com
- **Addresses**: bounce@mail2.bobbyseamoss.com through bounce@mail31.bobbyseamoss.com
- **Port**: 995 (POP3S)
- **Scan interval**: Every 15 minutes
- **Action**: Blocklist after 2 bounces

## Usage Scenarios

### After Database Wipe
```bash
# 1. Ensure database schema is installed
./listmonk --upgrade

# 2. Restore server configuration
./restore_config.sh
# Choose option 1

# 3. Verify
./restore_config.sh
# Choose option 4
```

### Before Major Changes
```bash
# Create backup before making changes
./restore_config.sh
# Choose option 2

# Creates: backup_YYYYMMDD_HHMMSS.sql
```

### Export Current Config for Review
```bash
./restore_config.sh
# Choose option 3

# Creates: current_config.txt with pretty-printed JSON
```

## Customization Guide

### Change Daily Limits Per Server
Edit `restore_servers.sql` and modify:
```json
"daily_limit": 1000  // Change to desired limit
```

### Disable Specific Servers
Set `"enabled": false` for any server you want to disable.

### Adjust Bounce Scan Interval
```json
"scan_interval": "15m"  // Options: "5m", "10m", "30m", "1h"
```

### Change Bounce Action
```json
"action": "blocklist"  // Options: "blocklist", "delete", "none"
"count": 2             // Bounces before action
```

## Verification Queries

### Check Server Count
```sql
SELECT
  key,
  jsonb_array_length(value) as server_count
FROM settings
WHERE key IN ('smtp', 'bounce');
```

### List All SMTP Servers
```sql
SELECT
  jsonb_array_elements(value)->>'host' as host,
  jsonb_array_elements(value)->>'from_email' as from_email,
  (jsonb_array_elements(value)->>'daily_limit')::int as daily_limit,
  jsonb_array_elements(value)->>'enabled' as enabled
FROM settings
WHERE key = 'smtp';
```

### Check Total Daily Capacity
```sql
SELECT
  SUM((elem->>'daily_limit')::int) as total_daily_capacity
FROM settings, jsonb_array_elements(value) as elem
WHERE key = 'smtp' AND elem->>'enabled' = 'true';
```

## Automated Backups

### Setup Daily Backup (Cron)
```bash
# Add to crontab
crontab -e

# Run backup daily at 2 AM
0 2 * * * cd /home/adam/listmonk && ./restore_config.sh << EOF
2
EOF
```

### Azure Database Automated Backups
```bash
# Enable 7-day retention
az postgres flexible-server update \
  --name listmonk420-db \
  --resource-group rg-listmonk420 \
  --backup-retention 7
```

## Troubleshooting

### "password authentication failed"
Check your connection string in `restore_config.sh`:
```bash
DB_PASSWORD="Listmonk420Pass!"  # Update if password changed
```

### "relation 'settings' does not exist"
Run the database schema migration first:
```bash
./listmonk --upgrade
```

### Settings Not Appearing in UI
After restore, restart the listmonk service:
```bash
# Azure Container Apps
az containerapp revision restart \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --revision $(az containerapp show -n listmonk420 -g rg-listmonk420 --query 'properties.latestRevisionName' -o tsv)
```

### Verify Firewall Access
```bash
# Test database connection
PGPASSWORD='Listmonk420Pass!' psql \
  -h listmonk420-db.postgres.database.azure.com \
  -U listmonkadmin \
  -d listmonk \
  -c "SELECT version();"
```

## Best Practices

1. **Always backup before restoring**: Use option 5 in the script
2. **Keep passwords in a secure location**: Consider using Azure Key Vault
3. **Test with a small campaign first**: After restore, send to 100 recipients
4. **Monitor daily usage**: Check `/api/queue/stats` endpoint
5. **Version control**: Commit restore_servers.sql to git (without passwords)
6. **Regular backups**: Run backup option weekly

## Security Notes

- The `restore_servers.sql` file contains passwords - **DO NOT** commit to public repositories
- Consider using environment variables for sensitive data
- Store backup files in secure Azure Storage with encryption
- Use Azure Key Vault for production environments

## Support

If you encounter issues:
1. Check logs: `./restore_config.sh` option 4 (Verify)
2. Review Azure Container App logs
3. Test database connection manually
4. Verify settings in listmonk UI: Settings → SMTP / Bounces

## Quick Reference

| Command | Purpose |
|---------|---------|
| `./restore_config.sh` | Interactive menu |
| Option 1 | Restore configuration |
| Option 2 | Backup current config |
| Option 3 | Export to readable file |
| Option 4 | Verify configuration |
| Option 5 | Backup + Restore (safest) |
