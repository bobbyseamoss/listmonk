#!/bin/bash
#
# Enable Azure Event Grid in Listmonk (Container-Friendly)
#
# This script enables Azure Event Grid webhooks in listmonk settings
# using whatever database connection method is available.
#
# Usage:
#   ./enable-azure.sh
#
# The script will try multiple methods:
#   1. Use environment variables if set (LISTMONK_DB_*)
#   2. Use database password from environment if available
#   3. Prompt for credentials interactively
#   4. Try common defaults
#

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_msg() {
    local color=$1
    shift
    echo -e "${color}$@${NC}"
}

print_header() {
    echo ""
    print_msg "$BLUE" "=========================================="
    print_msg "$BLUE" "$1"
    print_msg "$BLUE" "=========================================="
    echo ""
}

# Detect database credentials
detect_db_credentials() {
    # Try environment variables first
    DB_HOST="${LISTMONK_DB_HOST:-${DB_HOST:-listmonk420-db.postgres.database.azure.com}}"
    DB_PORT="${LISTMONK_DB_PORT:-${DB_PORT:-5432}}"
    DB_USER="${LISTMONK_DB_USER:-${DB_USER:-listmonkadmin}}"
    DB_DATABASE="${LISTMONK_DB_DATABASE:-${DB_DATABASE:-listmonk}}"

    # Check for password in environment
    if [ -n "${LISTMONK_DB_PASSWORD:-}" ]; then
        DB_PASSWORD="$LISTMONK_DB_PASSWORD"
    elif [ -n "${DB_PASSWORD:-}" ]; then
        DB_PASSWORD="$DB_PASSWORD"
    else
        DB_PASSWORD=""
    fi
}

# Method 1: Try psql directly with PGPASSWORD
try_psql() {
    print_msg "$BLUE" "Method 1: Trying psql..."

    if ! command -v psql &> /dev/null; then
        print_msg "$YELLOW" "  ⚠️  psql not found, skipping"
        return 1
    fi

    if [ -z "$DB_PASSWORD" ]; then
        print_msg "$YELLOW" "  ⚠️  No password available, skipping"
        return 1
    fi

    print_msg "$BLUE" "  Connecting to: $DB_USER@$DB_HOST:$DB_PORT/$DB_DATABASE"

    if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_DATABASE" \
        -f /home/adam/listmonk/deployment/scripts/enable_azure_event_grid.sql 2>&1; then
        print_msg "$GREEN" "  ✓ Successfully enabled via psql"
        return 0
    else
        print_msg "$YELLOW" "  ⚠️  psql failed"
        return 1
    fi
}

# Method 2: Try Python script
try_python() {
    print_msg "$BLUE" "Method 2: Trying Python script..."

    if ! command -v python3 &> /dev/null; then
        print_msg "$YELLOW" "  ⚠️  python3 not found, skipping"
        return 1
    fi

    # Check for psycopg2
    if ! python3 -c "import psycopg2" 2>/dev/null; then
        print_msg "$YELLOW" "  ⚠️  psycopg2 not installed"
        print_msg "$BLUE" "  Installing psycopg2-binary..."
        if pip3 install psycopg2-binary --quiet 2>&1; then
            print_msg "$GREEN" "  ✓ psycopg2 installed"
        else
            print_msg "$YELLOW" "  ⚠️  Could not install psycopg2, skipping"
            return 1
        fi
    fi

    if [ -z "$DB_PASSWORD" ]; then
        print_msg "$YELLOW" "  ⚠️  No password available, skipping"
        return 1
    fi

    if python3 /home/adam/listmonk/deployment/scripts/enable_azure_event_grid.py \
        --host "$DB_HOST" \
        --port "$DB_PORT" \
        --user "$DB_USER" \
        --password "$DB_PASSWORD" \
        --database "$DB_DATABASE" 2>&1; then
        print_msg "$GREEN" "  ✓ Successfully enabled via Python"
        return 0
    else
        print_msg "$YELLOW" "  ⚠️  Python script failed"
        return 1
    fi
}

# Method 3: Manual instructions
show_manual_instructions() {
    print_header "Manual Setup Required"

    print_msg "$YELLOW" "Unable to connect to database automatically."
    echo ""
    print_msg "$BLUE" "Please run ONE of the following manually:"
    echo ""

    print_msg "$BLUE" "Option 1: Using psql"
    print_msg "$YELLOW" "  PGPASSWORD='your-password' psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_DATABASE \\"
    print_msg "$YELLOW" "    -f /home/adam/listmonk/deployment/scripts/enable_azure_event_grid.sql"
    echo ""

    print_msg "$BLUE" "Option 2: Using Python"
    print_msg "$YELLOW" "  python3 /home/adam/listmonk/deployment/scripts/enable_azure_event_grid.py \\"
    print_msg "$YELLOW" "    --host $DB_HOST --port $DB_PORT --user $DB_USER \\"
    print_msg "$YELLOW" "    --password 'your-password' --database $DB_DATABASE"
    echo ""

    print_msg "$BLUE" "Option 3: Copy SQL to your database client"
    print_msg "$YELLOW" "  1. Open: /home/adam/listmonk/deployment/scripts/enable_azure_event_grid.sql"
    print_msg "$YELLOW" "  2. Copy the SQL content"
    print_msg "$YELLOW" "  3. Paste into Azure Portal Query Editor or your SQL client"
    print_msg "$YELLOW" "  4. Execute"
    echo ""

    print_msg "$BLUE" "Option 4: Update via Listmonk API (when running)"
    print_msg "$YELLOW" "  curl -u 'admin:password' -X PUT \\"
    print_msg "$YELLOW" "    'http://localhost:9000/api/settings' \\"
    print_msg "$YELLOW" "    -H 'Content-Type: application/json' \\"
    print_msg "$YELLOW" "    -d '{\"bounce.azure.enabled\": true}'"
    echo ""
}

# Main script
print_header "Enable Azure Event Grid for Listmonk"

# Detect credentials
detect_db_credentials

print_msg "$BLUE" "Database Configuration:"
print_msg "$BLUE" "  Host: $DB_HOST"
print_msg "$BLUE" "  Port: $DB_PORT"
print_msg "$BLUE" "  User: $DB_USER"
print_msg "$BLUE" "  Database: $DB_DATABASE"
if [ -n "$DB_PASSWORD" ]; then
    print_msg "$GREEN" "  Password: ✓ Found"
else
    print_msg "$YELLOW" "  Password: ⚠️  Not found"
fi
echo ""

# Try different methods
if try_psql; then
    exit 0
elif try_python; then
    exit 0
else
    show_manual_instructions
    exit 1
fi
