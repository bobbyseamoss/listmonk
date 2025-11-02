#!/bin/bash
# Run listmonk migration against Azure PostgreSQL

# Database credentials
DB_HOST="${LISTMONK_DB_HOST:-listmonk420-db.postgres.database.azure.com}"
DB_PORT="${LISTMONK_DB_PORT:-5432}"
DB_USER="${LISTMONK_DB_USER:-listmonkadmin}"
DB_PASSWORD="${LISTMONK_DB_PASSWORD}"
DB_NAME="${LISTMONK_DB_DATABASE:-listmonk}"

if [ -z "$DB_PASSWORD" ]; then
    echo "Error: LISTMONK_DB_PASSWORD environment variable not set"
    exit 1
fi

echo "Running migration against: $DB_HOST:$DB_PORT/$DB_NAME"

# Run the migration SQL directly
PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" << 'SQL_EOF'
-- v6.1.0 Migration: Azure Event Grid webhook tracking
DO $$
BEGIN
    -- Check if table already exists
    IF NOT EXISTS (SELECT FROM pg_tables WHERE schemaname = 'public' AND tablename = 'azure_message_tracking') THEN
        -- Create table
        CREATE TABLE azure_message_tracking (
            id                BIGSERIAL PRIMARY KEY,
            azure_message_id  UUID NOT NULL UNIQUE,
            campaign_id       INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
            subscriber_id     INTEGER NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE,
            smtp_server_uuid  UUID,
            sent_at           TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            created_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW()
        );

        -- Create indexes
        CREATE INDEX idx_azure_msg_tracking_msg_id ON azure_message_tracking(azure_message_id);
        CREATE INDEX idx_azure_msg_tracking_campaign ON azure_message_tracking(campaign_id);
        CREATE INDEX idx_azure_msg_tracking_subscriber ON azure_message_tracking(subscriber_id);
        CREATE INDEX idx_azure_msg_tracking_sent_at ON azure_message_tracking(sent_at);

        RAISE NOTICE 'Migration v6.1.0 completed: azure_message_tracking table created';
    ELSE
        RAISE NOTICE 'Migration v6.1.0 already applied: azure_message_tracking table exists';
    END IF;
END $$;
SQL_EOF

echo "Migration completed"
