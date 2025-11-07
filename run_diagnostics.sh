#!/bin/bash
# ==============================================================================
# LISTMONK DIAGNOSTICS RUNNER
# ==============================================================================
# This script helps you run the diagnostic SQL files against your Azure database
# ==============================================================================

set -e

# Database connection details
DB_HOST="listmonk420-db.postgres.database.azure.com"
DB_PORT="5432"
DB_USER="listmonkadmin"
DB_NAME="listmonk"
export PGPASSWORD="T@intshr3dd3r"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=============================================================================="
echo "LISTMONK DIAGNOSTICS"
echo "=============================================================================="
echo ""
echo "This script will run diagnostics on your listmonk database."
echo ""
echo "Database: $DB_HOST"
echo "User: $DB_USER"
echo ""

# Function to test database connection
test_connection() {
    echo -n "Testing database connection... "
    if timeout 10 psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" > /dev/null 2>&1; then
        echo -e "${GREEN}OK${NC}"
        return 0
    else
        echo -e "${RED}FAILED${NC}"
        echo ""
        echo "Unable to connect to the database. Please check:"
        echo "  1. Firewall rules allow your IP address"
        echo "  2. Database credentials are correct"
        echo "  3. Database is running"
        echo ""
        return 1
    fi
}

# Test connection first
if ! test_connection; then
    exit 1
fi

echo ""
echo "=============================================================================="
echo "RUNNING DIAGNOSTICS"
echo "=============================================================================="
echo ""

# Run the diagnostic script
echo "Running diagnose_bounces.sql..."
echo ""

psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f diagnose_bounces.sql

echo ""
echo "=============================================================================="
echo "DIAGNOSTIC COMPLETE"
echo "=============================================================================="
echo ""
echo "Review the output above to identify issues."
echo ""
echo "Next steps:"
echo ""
echo -e "${YELLOW}1. Fix bounce configuration:${NC}"
echo "   ./run_diagnostics.sh fix-config"
echo ""
echo -e "${YELLOW}2. Blocklist bounced subscribers:${NC}"
echo "   ./run_diagnostics.sh blocklist"
echo ""
echo -e "${YELLOW}3. Check specific issue:${NC}"
echo "   - If bounce.actions is wrong → run 'fix-config'"
echo "   - If subscribers should be blocklisted → run 'blocklist'"
echo "   - If message tracking is broken → check Azure webhook setup"
echo ""

# Handle command-line arguments
case "${1:-}" in
    fix-config)
        echo "=============================================================================="
        echo "FIXING BOUNCE CONFIGURATION"
        echo "=============================================================================="
        echo ""
        echo -e "${YELLOW}WARNING: This will update bounce.actions in the database.${NC}"
        echo "After running this, you MUST restart listmonk to apply changes!"
        echo ""
        read -p "Continue? (yes/no): " confirm
        if [ "$confirm" = "yes" ]; then
            psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f fix_bounce_config.sql
            echo ""
            echo -e "${GREEN}Configuration updated!${NC}"
            echo ""
            echo -e "${RED}IMPORTANT: Restart listmonk now!${NC}"
            echo "  az containerapp restart --name listmonk420 --resource-group <your-rg>"
        else
            echo "Aborted."
        fi
        ;;

    fix-soft-bounce)
        echo "=============================================================================="
        echo "FIXING SOFT BOUNCE CONFIGURATION"
        echo "=============================================================================="
        echo ""
        echo -e "${RED}CRITICAL: Your soft bounces are set to blocklist!${NC}"
        echo "Soft bounces are temporary issues and should NOT blocklist subscribers."
        echo ""
        echo "This will:"
        echo "  1. Change soft bounce action from 'blocklist' to 'none'"
        echo "  2. Show subscribers who were incorrectly blocklisted"
        echo ""
        read -p "Continue? (yes/no): " confirm
        if [ "$confirm" = "yes" ]; then
            psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f fix_soft_bounce_config.sql
            echo ""
            echo -e "${GREEN}Configuration updated!${NC}"
            echo ""
            echo -e "${RED}IMPORTANT: Restart listmonk now!${NC}"
            echo "  az containerapp restart --name listmonk420 --resource-group <your-rg>"
            echo ""
            echo "Next step: Review the list of incorrectly blocklisted subscribers"
            echo "Then run: ./run_diagnostics.sh unblocklist-soft"
        else
            echo "Aborted."
        fi
        ;;

    unblocklist-soft)
        echo "=============================================================================="
        echo "UN-BLOCKLIST SOFT BOUNCE SUBSCRIBERS"
        echo "=============================================================================="
        echo ""
        psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f unblocklist_soft_bounces.sql
        echo ""
        echo "Review the list above. To actually un-blocklist, edit unblocklist_soft_bounces.sql"
        echo "and uncomment the UPDATE statement, then run this command again."
        ;;

    blocklist)
        echo "=============================================================================="
        echo "BLOCKLISTING BOUNCED SUBSCRIBERS"
        echo "=============================================================================="
        echo ""
        psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f blocklist_bounced_subscribers.sql
        echo ""
        echo "Review the list above. To actually blocklist, edit blocklist_bounced_subscribers.sql"
        echo "and uncomment the UPDATE statement, then run this command again."
        ;;

    help|*)
        echo "Usage: ./run_diagnostics.sh [command]"
        echo ""
        echo "Commands:"
        echo "  (none)              - Run diagnostics only"
        echo "  fix-config          - Fix bounce.actions configuration"
        echo "  fix-soft-bounce     - Fix soft bounce config (RECOMMENDED)"
        echo "  unblocklist-soft    - Un-blocklist subscribers with only soft bounces"
        echo "  blocklist           - Preview/blocklist bounced subscribers"
        echo "  help                - Show this help"
        ;;
esac
