#!/bin/bash

set -e

echo "=== Syncing Azure Database to Local Dev Environment ==="
echo ""

# Configuration
AZURE_HOST="listmonk420-db.postgres.database.azure.com"
AZURE_USER="listmonkadmin"
AZURE_PASS="T@intshr3dd3r"
AZURE_DB="listmonk"

LOCAL_HOST="localhost"
LOCAL_USER="listmonk-dev"
LOCAL_PASS="listmonk-dev"
LOCAL_DB="listmonk-dev"

BACKUP_FILE="azure_db_backup_$(date +%Y%m%d_%H%M%S).sql"

# Step 1: Export from Azure
echo "Step 1: Exporting data from Azure PostgreSQL..."
PGPASSWORD="${AZURE_PASS}" pg_dump \
  -h "${AZURE_HOST}" \
  -U "${AZURE_USER}" \
  -d "${AZURE_DB}" \
  --no-owner \
  --no-acl \
  -f "${BACKUP_FILE}"

echo "✓ Azure data exported to ${BACKUP_FILE}"
echo ""

# Step 2: Stop local dev containers
echo "Step 2: Stopping local dev containers..."
make rm-dev-docker 2>/dev/null || true
echo "✓ Local containers stopped"
echo ""

# Step 3: Start database
echo "Step 3: Starting local database..."
cd dev
docker compose up -d db
cd ..
echo "Waiting for database to be ready..."
sleep 15
echo "✓ Database ready"
echo ""

# Step 4: Import data
echo "Step 4: Importing Azure data to local database..."
PGPASSWORD="${LOCAL_PASS}" psql \
  -h "${LOCAL_HOST}" \
  -p 5432 \
  -U "${LOCAL_USER}" \
  -d "${LOCAL_DB}" \
  -f "${BACKUP_FILE}"

echo "✓ Data imported successfully"
echo ""

# Step 5: Verify import
echo "Step 5: Verifying import..."
SUBSCRIBER_COUNT=$(PGPASSWORD="${LOCAL_PASS}" psql \
  -h "${LOCAL_HOST}" \
  -p 5432 \
  -U "${LOCAL_USER}" \
  -d "${LOCAL_DB}" \
  -t -c "SELECT COUNT(*) FROM subscribers;")

LIST_COUNT=$(PGPASSWORD="${LOCAL_PASS}" psql \
  -h "${LOCAL_HOST}" \
  -p 5432 \
  -U "${LOCAL_USER}" \
  -d "${LOCAL_DB}" \
  -t -c "SELECT COUNT(*) FROM lists;")

echo "✓ Subscribers: ${SUBSCRIBER_COUNT}"
echo "✓ Lists: ${LIST_COUNT}"
echo ""

echo "=== Sync Complete! ==="
echo ""
echo "Next steps:"
echo "  1. Run 'make dev-docker' to start the full dev environment"
echo "  2. Access at http://localhost:8080"
echo "  3. Run 'make dist' to build the complete distribution"
echo ""
echo "Backup file saved as: ${BACKUP_FILE}"
echo "(Keep this file safe as a backup!)"
