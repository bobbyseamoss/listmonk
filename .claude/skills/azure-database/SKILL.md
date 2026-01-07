---
name: azure-database
description: Azure PostgreSQL database connection credentials and commands. Triggers on database, psql, PostgreSQL, Azure DB, query, SQL, check database, connect database, database credentials, listmonk420-db, run query, execute SQL.
---

# Azure Database Connection

## Purpose

Provides connection credentials and commands for accessing the Azure PostgreSQL database used by the listmonk420 deployment.

## When to Use

- Querying the Azure PostgreSQL database
- Checking campaign status, email queue, or subscriber data
- Running SQL commands against the production database
- Debugging database-related issues
- Viewing database statistics or counts

## Connection Details

### Azure PostgreSQL Server

| Property | Value |
|----------|-------|
| **Host** | `listmonk420-db.postgres.database.azure.com` |
| **Port** | `5432` |
| **Database** | `listmonk` |
| **Username** | `listmonkadmin` |
| **Password** | `T@intshr3dd3r` |
| **SSL Mode** | `require` |

### Container App Details

| Property | Value |
|----------|-------|
| **App Name** | `listmonk420` |
| **Resource Group** | `rg-listmonk420` |
| **FQDN** | `listmonk420.agreeablerock-9bb0f280.eastus2.azurecontainerapps.io` |
| **Region** | East US 2 |

## Connection Commands

### Direct psql Connection

```bash
PGPASSWORD='T@intshr3dd3r' psql "host=listmonk420-db.postgres.database.azure.com port=5432 dbname=listmonk user=listmonkadmin sslmode=require"
```

### One-liner Query Pattern

```bash
PGPASSWORD='T@intshr3dd3r' psql "host=listmonk420-db.postgres.database.azure.com port=5432 dbname=listmonk user=listmonkadmin sslmode=require" -c "YOUR SQL HERE"
```

## Common Queries

### Check Email Queue Status

```bash
PGPASSWORD='T@intshr3dd3r' psql "host=listmonk420-db.postgres.database.azure.com port=5432 dbname=listmonk user=listmonkadmin sslmode=require" -c "
SELECT status, COUNT(*) as count
FROM email_queue
GROUP BY status
ORDER BY status;"
```

### Check Campaign Status

```bash
PGPASSWORD='T@intshr3dd3r' psql "host=listmonk420-db.postgres.database.azure.com port=5432 dbname=listmonk user=listmonkadmin sslmode=require" -c "
SELECT id, name, status, sent, to_send, use_queue, messenger
FROM campaigns
WHERE status = 'running'
ORDER BY id DESC;"
```

### Check SMTP Server Usage

```bash
PGPASSWORD='T@intshr3dd3r' psql "host=listmonk420-db.postgres.database.azure.com port=5432 dbname=listmonk user=listmonkadmin sslmode=require" -c "
SELECT smtp_server_uuid, usage_date, emails_sent
FROM smtp_daily_usage
WHERE usage_date = CURRENT_DATE
ORDER BY emails_sent DESC;"
```

### Check Subscriber Count by Status

```bash
PGPASSWORD='T@intshr3dd3r' psql "host=listmonk420-db.postgres.database.azure.com port=5432 dbname=listmonk user=listmonkadmin sslmode=require" -c "
SELECT status, COUNT(*) as count
FROM subscribers
GROUP BY status;"
```

### Check Recent Bounces

```bash
PGPASSWORD='T@intshr3dd3r' psql "host=listmonk420-db.postgres.database.azure.com port=5432 dbname=listmonk user=listmonkadmin sslmode=require" -c "
SELECT email, type, source, created_at
FROM bounces
ORDER BY created_at DESC
LIMIT 20;"
```

### Check Queue for Specific Campaign

```bash
PGPASSWORD='T@intshr3dd3r' psql "host=listmonk420-db.postgres.database.azure.com port=5432 dbname=listmonk user=listmonkadmin sslmode=require" -c "
SELECT status, COUNT(*) as count
FROM email_queue
WHERE campaign_id = CAMPAIGN_ID_HERE
GROUP BY status;"
```

## Azure CLI Commands

### View Container App Logs

```bash
az containerapp logs show --name listmonk420 --resource-group rg-listmonk420 --type console --tail 100
```

### Check Container App Status

```bash
az containerapp show --name listmonk420 --resource-group rg-listmonk420 --query "{name:name, status:properties.runningStatus, latestRevision:properties.latestRevisionName}"
```

### View Environment Variables

```bash
az containerapp show --name listmonk420 --resource-group rg-listmonk420 | jq -r '.properties.template.containers[0].env[] | "\(.name)=\(.value // "SECRET")"'
```

## Key Tables

| Table | Purpose |
|-------|---------|
| `campaigns` | Campaign definitions and status |
| `email_queue` | Queued emails for sending |
| `subscribers` | Subscriber records |
| `subscriber_lists` | List memberships |
| `bounces` | Bounce records |
| `smtp_daily_usage` | Daily email counts per SMTP |
| `smtp_rate_limit_state` | Rate limiting state |
| `settings` | Application settings (JSON) |

## Important Notes

1. **SSL Required**: Azure PostgreSQL requires SSL. Always use `sslmode=require`
2. **No Local DB**: The production deployment uses only Azure database, not localhost
3. **Credentials**: Username is `listmonkadmin`, not `listmonk`
4. **Resource Group**: Container app is in `rg-listmonk420`, not `mail.bobbyseamoss.com`
