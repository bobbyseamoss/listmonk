#!/bin/bash
##
## Fix Bounce Mailbox Usernames (Local Dev Version)
## For use with local PostgreSQL database
##

set -e

echo "Fixing bounce mailbox usernames in local database..."
echo ""

# Local database connection
DB_HOST="${LISTMONK_DB_HOST:-localhost}"
DB_PORT="${LISTMONK_DB_PORT:-5432}"
DB_USER="${LISTMONK_DB_USER:-listmonk}"
DB_NAME="${LISTMONK_DB_DATABASE:-listmonk}"

echo "Database: $DB_HOST:$DB_PORT/$DB_NAME"
echo "User: $DB_USER"
echo ""

# Use psql to fix all usernames
PGPASSWORD="${LISTMONK_DB_PASSWORD:-listmonk}" psql \
    -h "$DB_HOST" \
    -p "$DB_PORT" \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    -v ON_ERROR_STOP=1 \
    <<'EOF'
-- Fix bounce mailbox usernames by stripping the @hostname part
UPDATE settings
SET value = jsonb_set(
    value,
    '{mailboxes}',
    (
        SELECT jsonb_agg(
            CASE
                WHEN elem->>'username' LIKE '%@%'
                THEN jsonb_set(
                    elem,
                    '{username}',
                    to_jsonb(split_part(elem->>'username', '@', 1))
                )
                ELSE elem
            END
        )
        FROM jsonb_array_elements(value->'mailboxes') elem
    )
)
WHERE key = 'bounce'
AND value->'mailboxes' IS NOT NULL;

-- Show the updated usernames
SELECT
    key,
    elem->>'host' as host,
    elem->>'username' as username
FROM settings,
    jsonb_array_elements(value->'mailboxes') elem
WHERE key = 'bounce';
EOF

echo ""
echo "✓ Usernames fixed!"
echo ""
echo "IMPORTANT: Restart listmonk for changes to take effect."
echo ""
