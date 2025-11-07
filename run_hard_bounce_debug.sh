#!/bin/bash
set -e

DB_HOST="listmonk420-db.postgres.database.azure.com"
DB_PORT="5432"
DB_USER="listmonkadmin"
DB_NAME="listmonk"
export PGPASSWORD="T@intshr3dd3r"

echo "=============================================================================="
echo "HARD BOUNCE DEBUGGING"
echo "=============================================================================="
echo ""
echo "Running comprehensive hard bounce diagnostics..."
echo ""

# Test connection
echo -n "Testing database connection... "
if timeout 10 psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" > /dev/null 2>&1; then
    echo "OK"
else
    echo "FAILED"
    exit 1
fi

echo ""
echo "Running diagnostics..."
echo ""

psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f debug_hard_bounce_failure.sql

echo ""
echo "=============================================================================="
echo "NEXT STEPS"
echo "=============================================================================="
echo ""
echo "Based on the results above:"
echo "  - If subscriber_id is NULL → Run: ./fix_subscriber_lookup.sh"
echo "  - If status != blocklisted → Check application logs for errors"
echo "  - If lookup failed → Email mismatch issue"
echo ""
