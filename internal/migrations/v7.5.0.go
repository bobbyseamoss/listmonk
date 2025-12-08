package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V7_5_0 creates SendGrid event tracking tables for delivery and engagement events.
// This mirrors the Azure event tracking tables but for SendGrid webhooks.
func V7_5_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	lo.Println("Creating SendGrid event tracking tables...")

	// Create SendGrid delivery events table
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS sendgrid_delivery_events (
			id BIGSERIAL PRIMARY KEY,
			sg_message_id TEXT,
			campaign_id INTEGER REFERENCES campaigns(id) ON DELETE CASCADE,
			subscriber_id INTEGER REFERENCES subscribers(id) ON DELETE CASCADE,
			event_type VARCHAR(50) NOT NULL,
			email VARCHAR(255),
			reason TEXT,
			bounce_classification VARCHAR(50),
			smtp_id TEXT,
			ip VARCHAR(45),
			event_timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_sendgrid_delivery_campaign ON sendgrid_delivery_events(campaign_id);
		CREATE INDEX IF NOT EXISTS idx_sendgrid_delivery_subscriber ON sendgrid_delivery_events(subscriber_id);
		CREATE INDEX IF NOT EXISTS idx_sendgrid_delivery_event_type ON sendgrid_delivery_events(event_type);
		CREATE INDEX IF NOT EXISTS idx_sendgrid_delivery_timestamp ON sendgrid_delivery_events(event_timestamp);
		CREATE INDEX IF NOT EXISTS idx_sendgrid_delivery_msg_id ON sendgrid_delivery_events(sg_message_id);
	`); err != nil {
		return err
	}

	lo.Println("Created sendgrid_delivery_events table")

	// Create SendGrid engagement events table
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS sendgrid_engagement_events (
			id BIGSERIAL PRIMARY KEY,
			sg_message_id TEXT,
			campaign_id INTEGER REFERENCES campaigns(id) ON DELETE CASCADE,
			subscriber_id INTEGER REFERENCES subscribers(id) ON DELETE CASCADE,
			event_type VARCHAR(20) NOT NULL,
			email VARCHAR(255),
			url TEXT,
			user_agent TEXT,
			ip VARCHAR(45),
			event_timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_sendgrid_engagement_campaign ON sendgrid_engagement_events(campaign_id);
		CREATE INDEX IF NOT EXISTS idx_sendgrid_engagement_subscriber ON sendgrid_engagement_events(subscriber_id);
		CREATE INDEX IF NOT EXISTS idx_sendgrid_engagement_event_type ON sendgrid_engagement_events(event_type);
		CREATE INDEX IF NOT EXISTS idx_sendgrid_engagement_timestamp ON sendgrid_engagement_events(event_timestamp);
		CREATE INDEX IF NOT EXISTS idx_sendgrid_engagement_msg_id ON sendgrid_engagement_events(sg_message_id);
	`); err != nil {
		return err
	}

	lo.Println("Created sendgrid_engagement_events table")

	return nil
}
