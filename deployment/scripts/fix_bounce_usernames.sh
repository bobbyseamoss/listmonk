#!/bin/bash
##
## Fix Bounce Mailbox Usernames
## Strips the @hostname part from bounce mailbox usernames
## Changes "adam@mail2.bobbyseamoss.com" to "adam"
##

set -e  # Exit on error

echo "======================================================================"
echo "Bounce Mailbox Username Fix"
echo "======================================================================"
echo ""

# Database connection from environment
DB_HOST="${LISTMONK_DB_HOST:-listmonk420-db.postgres.database.azure.com}"
DB_PORT="${LISTMONK_DB_PORT:-5432}"
DB_USER="${LISTMONK_DB_USER:-listmonkadmin}"
DB_NAME="${LISTMONK_DB_DATABASE:-listmonk}"
DB_PASSWORD="${LISTMONK_DB_PASSWORD}"

if [ -z "$DB_PASSWORD" ]; then
    echo "❌ Error: LISTMONK_DB_PASSWORD not set"
    exit 1
fi

echo "Database: $DB_HOST:$DB_PORT/$DB_NAME"
echo ""

# Get current bounce settings
echo "Fetching current bounce configuration..."
BOUNCE_SETTINGS=$(PGPASSWORD="$DB_PASSWORD" psql \
    -h "$DB_HOST" \
    -p "$DB_PORT" \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    -t -A \
    -c "SELECT value FROM settings WHERE key = 'bounce';")

if [ -z "$BOUNCE_SETTINGS" ]; then
    echo "❌ Error: No bounce settings found in database"
    exit 1
fi

# Extract mailboxes array
MAILBOXES=$(echo "$BOUNCE_SETTINGS" | jq '.mailboxes')
MAILBOX_COUNT=$(echo "$MAILBOXES" | jq 'length')

echo "Found $MAILBOX_COUNT bounce mailboxes"
echo ""

# Process each mailbox and strip @hostname from username
UPDATED_MAILBOXES="$MAILBOXES"
CHANGED=0

for ((i=0; i<MAILBOX_COUNT; i++)); do
    CURRENT_USERNAME=$(echo "$MAILBOXES" | jq -r ".[$i].username")
    HOST=$(echo "$MAILBOXES" | jq -r ".[$i].host")

    # Check if username contains @
    if [[ "$CURRENT_USERNAME" == *"@"* ]]; then
        # Extract local part (everything before @)
        NEW_USERNAME="${CURRENT_USERNAME%%@*}"

        echo "  Mailbox $((i+1)) ($HOST):"
        echo "    Old: $CURRENT_USERNAME"
        echo "    New: $NEW_USERNAME"

        # Update the username in the JSON
        UPDATED_MAILBOXES=$(echo "$UPDATED_MAILBOXES" | jq \
            --arg idx "$i" \
            --arg user "$NEW_USERNAME" \
            '.[$idx | tonumber].username = $user')

        ((CHANGED++))
    else
        echo "  Mailbox $((i+1)) ($HOST): OK (username is already correct: $CURRENT_USERNAME)"
    fi
done

echo ""

if [ $CHANGED -eq 0 ]; then
    echo "✓ All usernames are already correct. No changes needed."
    exit 0
fi

echo "======================================================================"
echo "$CHANGED username(s) will be updated"
echo "======================================================================"
echo ""

# Update the bounce settings with fixed usernames
UPDATED_SETTINGS=$(echo "$BOUNCE_SETTINGS" | jq --argjson mailboxes "$UPDATED_MAILBOXES" '.mailboxes = $mailboxes')

# Write back to database
echo "Updating database..."
PGPASSWORD="$DB_PASSWORD" psql \
    -h "$DB_HOST" \
    -p "$DB_PORT" \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    -v ON_ERROR_STOP=1 \
    <<EOF
UPDATE settings
SET value = '$UPDATED_SETTINGS'::jsonb
WHERE key = 'bounce';
EOF

echo "✓ Database updated successfully"
echo ""
echo "======================================================================"
echo "Fix Complete"
echo "======================================================================"
echo "✓ Updated $CHANGED bounce mailbox username(s)"
echo ""
echo "IMPORTANT: You must restart listmonk for changes to take effect:"
echo "  - If running in Azure: Restart the container app"
echo "  - If running locally: Stop and restart the listmonk process"
echo ""
