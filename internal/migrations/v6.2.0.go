package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_2_0 performs the DB migrations for Azure Event Grid delivery and engagement tracking.
func V6_2_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	lo.Println("running migration v6.2.0: Azure Event Grid delivery and engagement tracking")

	// Create azure_delivery_events table for storing delivery status events
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS azure_delivery_events (
			id                BIGSERIAL PRIMARY KEY,
			azure_message_id  UUID NOT NULL,
			campaign_id       INTEGER REFERENCES campaigns(id) ON DELETE CASCADE,
			subscriber_id     INTEGER REFERENCES subscribers(id) ON DELETE CASCADE,
			status            VARCHAR(50) NOT NULL,
			status_reason     TEXT,
			delivery_status_details TEXT,
			event_timestamp   TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`); err != nil {
		return err
	}

	// Create indexes for efficient delivery event lookups
	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_azure_delivery_msg_id ON azure_delivery_events(azure_message_id);
		CREATE INDEX IF NOT EXISTS idx_azure_delivery_campaign ON azure_delivery_events(campaign_id);
		CREATE INDEX IF NOT EXISTS idx_azure_delivery_subscriber ON azure_delivery_events(subscriber_id);
		CREATE INDEX IF NOT EXISTS idx_azure_delivery_status ON azure_delivery_events(status);
		CREATE INDEX IF NOT EXISTS idx_azure_delivery_timestamp ON azure_delivery_events(event_timestamp);
	`); err != nil {
		return err
	}

	// Create azure_engagement_events table for storing open/click events
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS azure_engagement_events (
			id                BIGSERIAL PRIMARY KEY,
			azure_message_id  UUID NOT NULL,
			campaign_id       INTEGER REFERENCES campaigns(id) ON DELETE CASCADE,
			subscriber_id     INTEGER REFERENCES subscribers(id) ON DELETE CASCADE,
			engagement_type   VARCHAR(20) NOT NULL,
			engagement_context TEXT,
			user_agent        TEXT,
			event_timestamp   TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`); err != nil {
		return err
	}

	// Create indexes for efficient engagement event lookups
	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_azure_engagement_msg_id ON azure_engagement_events(azure_message_id);
		CREATE INDEX IF NOT EXISTS idx_azure_engagement_campaign ON azure_engagement_events(campaign_id);
		CREATE INDEX IF NOT EXISTS idx_azure_engagement_subscriber ON azure_engagement_events(subscriber_id);
		CREATE INDEX IF NOT EXISTS idx_azure_engagement_type ON azure_engagement_events(engagement_type);
		CREATE INDEX IF NOT EXISTS idx_azure_engagement_timestamp ON azure_engagement_events(event_timestamp);
	`); err != nil {
		return err
	}

	lo.Println("migration v6.2.0 completed successfully")
	return nil
}
