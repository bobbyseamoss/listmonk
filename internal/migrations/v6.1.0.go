package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_1_0 performs the DB migrations for Azure Event Grid webhook integration.
func V6_1_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	lo.Println("running migration v6.1.0: Azure Event Grid webhook tracking")

	// Create azure_message_tracking table for correlating Azure events with listmonk campaigns/subscribers
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS azure_message_tracking (
			id                BIGSERIAL PRIMARY KEY,
			azure_message_id  UUID NOT NULL UNIQUE,
			campaign_id       INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
			subscriber_id     INTEGER NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE,
			smtp_server_uuid  UUID,
			sent_at           TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			created_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`); err != nil {
		return err
	}

	// Create indexes for efficient lookups
	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_azure_msg_tracking_msg_id ON azure_message_tracking(azure_message_id);
		CREATE INDEX IF NOT EXISTS idx_azure_msg_tracking_campaign ON azure_message_tracking(campaign_id);
		CREATE INDEX IF NOT EXISTS idx_azure_msg_tracking_subscriber ON azure_message_tracking(subscriber_id);
		CREATE INDEX IF NOT EXISTS idx_azure_msg_tracking_sent_at ON azure_message_tracking(sent_at);
	`); err != nil {
		return err
	}

	lo.Println("migration v6.1.0 completed successfully")
	return nil
}
