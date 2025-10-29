#!/bin/bash
##
## Configuration Initialization Script
## Runs inside the container after --install to configure SMTP and bounce settings
## Reads templates and injects credentials from environment variables
##

set -e  # Exit on error

echo "======================================================================"
echo "Listmonk Configuration Initialization"
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

# Paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_DIR="$SCRIPT_DIR/../configs"
SMTP_TEMPLATE="$CONFIG_DIR/smtp_config_template.json"
BOUNCE_TEMPLATE="$CONFIG_DIR/bounce_config_template.json"
APP_SETTINGS="$CONFIG_DIR/app_settings.json"

# Check if templates exist
if [ ! -f "$SMTP_TEMPLATE" ]; then
    echo "❌ Error: SMTP template not found: $SMTP_TEMPLATE"
    exit 1
fi

if [ ! -f "$BOUNCE_TEMPLATE" ]; then
    echo "❌ Error: Bounce template not found: $BOUNCE_TEMPLATE"
    exit 1
fi

echo "✓ Configuration templates found"
echo ""

# Helper function to inject credentials into config
inject_smtp_credentials() {
    local config="$1"
    local result="$config"

    # Iterate through each SMTP server and inject credentials
    local smtp_count=$(echo "$config" | jq 'length')

    for ((i=0; i<smtp_count; i++)); do
        local username_secret=$(echo "$config" | jq -r ".[$i].username_secret")
        local password_secret=$(echo "$config" | jq -r ".[$i].password_secret")

        # Get credential from environment variable (uppercase, replace - with _)
        local username_var=$(echo "$username_secret" | tr '[:lower:]' '[:upper:]' | tr '-' '_')
        local password_var=$(echo "$password_secret" | tr '[:lower:]' '[:upper:]' | tr '-' '_')

        local username="${!username_var}"
        local password="${!password_var}"

        if [ -z "$username" ] || [ -z "$password" ]; then
            echo "  ⚠️  Warning: Credentials not found for SMTP server $((i+1)) ($username_secret)"
            username=""
            password=""
        fi

        # Inject credentials and remove *_secret fields
        result=$(echo "$result" | jq \
            --arg idx "$i" \
            --arg user "$username" \
            --arg pass "$password" \
            '.[$idx | tonumber] |= (.username = $user | .password = $pass | del(.username_secret, .password_secret))')
    done

    echo "$result"
}

inject_bounce_credentials() {
    local config="$1"
    local result="$config"

    # Iterate through each bounce mailbox and inject credentials
    local bounce_count=$(echo "$config" | jq 'length')

    for ((i=0; i<bounce_count; i++)); do
        local username_secret=$(echo "$config" | jq -r ".[$i].username_secret")
        local password_secret=$(echo "$config" | jq -r ".[$i].password_secret")

        # Get credential from environment variable
        local username_var=$(echo "$username_secret" | tr '[:lower:]' '[:upper:]' | tr '-' '_')
        local password_var=$(echo "$password_secret" | tr '[:lower:]' '[:upper:]' | tr '-' '_')

        local username="${!username_var}"
        local password="${!password_var}"

        if [ -z "$username" ] || [ -z "$password" ]; then
            echo "  ⚠️  Warning: Credentials not found for bounce mailbox $((i+1)) ($username_secret)"
            username=""
            password=""
        fi

        # Inject credentials and remove *_secret fields
        result=$(echo "$result" | jq \
            --arg idx "$i" \
            --arg user "$username" \
            --arg pass "$password" \
            '.[$idx | tonumber] |= (.username = $user | .password = $pass | del(.username_secret, .password_secret))')
    done

    echo "$result"
}

# Process SMTP configuration
echo "Processing SMTP configuration..."
SMTP_CONFIG=$(cat "$SMTP_TEMPLATE")
SMTP_COUNT=$(echo "$SMTP_CONFIG" | jq 'length')
echo "  Found $SMTP_COUNT SMTP servers in template"

echo "  Injecting credentials..."
SMTP_CONFIG=$(inject_smtp_credentials "$SMTP_CONFIG")

# Insert into database
echo "  Inserting into database..."
PGPASSWORD="$DB_PASSWORD" psql \
    -h "$DB_HOST" \
    -p "$DB_PORT" \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    -v ON_ERROR_STOP=1 \
    <<EOF
INSERT INTO settings (key, value)
VALUES ('smtp', '$SMTP_CONFIG'::jsonb)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;
EOF

echo "✓ SMTP configuration saved"
echo ""

# Process Bounce configuration
echo "Processing bounce configuration..."
BOUNCE_CONFIG=$(cat "$BOUNCE_TEMPLATE")
BOUNCE_COUNT=$(echo "$BOUNCE_CONFIG" | jq 'length')
echo "  Found $BOUNCE_COUNT bounce mailboxes in template"

echo "  Injecting credentials..."
BOUNCE_CONFIG=$(inject_bounce_credentials "$BOUNCE_CONFIG")

# Wrap in bounce settings structure
BOUNCE_SETTINGS=$(jq -n \
    --argjson mailboxes "$BOUNCE_CONFIG" \
    '{
        "enabled": true,
        "webhooks_enabled": false,
        "ses_enabled": false,
        "sendgrid_enabled": false,
        "sendgrid_key": "",
        "postmark": {"enabled": false, "username": "", "password": ""},
        "forwardemail": {"enabled": false, "key": ""},
        "mailboxes": $mailboxes
    }')

# Insert into database
echo "  Inserting into database..."
PGPASSWORD="$DB_PASSWORD" psql \
    -h "$DB_HOST" \
    -p "$DB_PORT" \
    -U "$DB_USER" \
    -d "$DB_NAME" \
    -v ON_ERROR_STOP=1 \
    <<EOF
INSERT INTO settings (key, value)
VALUES ('bounce', '$BOUNCE_SETTINGS'::jsonb)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;
EOF

echo "✓ Bounce configuration saved"
echo ""

# Process app settings if available
if [ -f "$APP_SETTINGS" ]; then
    echo "Processing app settings..."
    APP_CONFIG=$(cat "$APP_SETTINGS")

    # Get existing app settings and merge
    EXISTING_APP=$(PGPASSWORD="$DB_PASSWORD" psql \
        -h "$DB_HOST" \
        -p "$DB_PORT" \
        -U "$DB_USER" \
        -d "$DB_NAME" \
        -t -A \
        -c "SELECT value FROM settings WHERE key = 'app';")

    if [ -n "$EXISTING_APP" ]; then
        # Merge with existing
        MERGED_APP=$(echo "$EXISTING_APP" | jq --argjson new "$APP_CONFIG" '. + $new')
    else
        MERGED_APP="$APP_CONFIG"
    fi

    PGPASSWORD="$DB_PASSWORD" psql \
        -h "$DB_HOST" \
        -p "$DB_PORT" \
        -U "$DB_USER" \
        -d "$DB_NAME" \
        -v ON_ERROR_STOP=1 \
        <<EOF
INSERT INTO settings (key, value)
VALUES ('app', '$MERGED_APP'::jsonb)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;
EOF

    echo "✓ App settings saved"
    echo ""
fi

echo "======================================================================"
echo "Configuration Initialization Complete"
echo "======================================================================"
echo "✓ SMTP servers: $SMTP_COUNT"
echo "✓ Bounce mailboxes: $BOUNCE_COUNT"
echo ""
echo "Listmonk is now configured and ready to use!"
echo ""
