#!/bin/bash
set -e

DB_HOST="listmonk420-db.postgres.database.azure.com"
DB_PORT="5432"
DB_USER="listmonkadmin"
DB_NAME="listmonk"
export PGPASSWORD="T@intshr3dd3r"

echo "=============================================================================="
echo "CLICKS & VIEWS TRACKING CHECK"
echo "=============================================================================="
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
echo "Checking clicks and views tracking..."
echo ""

psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f check_clicks_views.sql

echo ""
echo "=============================================================================="
echo "SUMMARY"
echo "=============================================================================="
echo ""
echo "If counts match between Azure and DB tables:"
echo "  ✓ Clicks and views are being tracked correctly"
echo ""
echo "If counts don't match:"
echo "  - Check webhook_logs for processing errors"
echo "  - Check application logs for insertion errors"
echo "  - Verify subscriber lookups are working"
echo ""
