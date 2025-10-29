package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_0_0 performs the DB migrations for queue-based email delivery system.
func V6_0_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	lo.Println("IMPORTANT: this upgrade adds queue-based email delivery system. Please be patient ...")

	// Create email_queue table
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS email_queue (
			id BIGSERIAL PRIMARY KEY,
			campaign_id INT NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
			subscriber_id INT NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE,

			-- Queue management
			status VARCHAR(20) NOT NULL DEFAULT 'queued',
			priority INT NOT NULL DEFAULT 0,

			-- Scheduling
			scheduled_at TIMESTAMP WITH TIME ZONE NOT NULL,
			sent_at TIMESTAMP WITH TIME ZONE,

			-- Server assignment
			assigned_smtp_server_uuid VARCHAR(255),

			-- Retry tracking
			retry_count INT NOT NULL DEFAULT 0,
			last_error TEXT,

			-- Timestamps
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

			-- Indexes for performance
			CONSTRAINT email_queue_status_check CHECK (status IN ('queued', 'sending', 'sent', 'failed', 'cancelled'))
		);

		CREATE INDEX IF NOT EXISTS idx_email_queue_status ON email_queue(status);
		CREATE INDEX IF NOT EXISTS idx_email_queue_scheduled_at ON email_queue(scheduled_at);
		CREATE INDEX IF NOT EXISTS idx_email_queue_campaign_id ON email_queue(campaign_id);
		CREATE INDEX IF NOT EXISTS idx_email_queue_assigned_smtp ON email_queue(assigned_smtp_server_uuid);
		CREATE INDEX IF NOT EXISTS idx_email_queue_status_scheduled ON email_queue(status, scheduled_at);
	`); err != nil {
		return err
	}

	// Create smtp_daily_usage table
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS smtp_daily_usage (
			id BIGSERIAL PRIMARY KEY,
			smtp_server_uuid VARCHAR(255) NOT NULL,
			usage_date DATE NOT NULL,
			emails_sent INT NOT NULL DEFAULT 0,

			-- Timestamps
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

			-- Unique constraint: one row per server per day
			CONSTRAINT smtp_daily_usage_unique UNIQUE (smtp_server_uuid, usage_date)
		);

		CREATE INDEX IF NOT EXISTS idx_smtp_daily_usage_uuid_date ON smtp_daily_usage(smtp_server_uuid, usage_date);
	`); err != nil {
		return err
	}

	// Create smtp_rate_limit_state table
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS smtp_rate_limit_state (
			id BIGSERIAL PRIMARY KEY,
			smtp_server_uuid VARCHAR(255) NOT NULL UNIQUE,

			-- Sliding window tracking
			window_start TIMESTAMP WITH TIME ZONE NOT NULL,
			emails_in_window INT NOT NULL DEFAULT 0,

			-- Timestamps
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_smtp_rate_limit_uuid ON smtp_rate_limit_state(smtp_server_uuid);
	`); err != nil {
		return err
	}

	// Modify campaigns table to add queue tracking and testing mode
	if _, err := db.Exec(`
		ALTER TABLE campaigns
			ADD COLUMN IF NOT EXISTS use_queue BOOLEAN NOT NULL DEFAULT FALSE,
			ADD COLUMN IF NOT EXISTS queued_at TIMESTAMP WITH TIME ZONE,
			ADD COLUMN IF NOT EXISTS queue_completed_at TIMESTAMP WITH TIME ZONE,
			ADD COLUMN IF NOT EXISTS testing_mode BOOLEAN NOT NULL DEFAULT FALSE;

		CREATE INDEX IF NOT EXISTS idx_campaigns_use_queue ON campaigns(use_queue);
		CREATE INDEX IF NOT EXISTS idx_campaigns_queued_at ON campaigns(queued_at);
	`); err != nil {
		return err
	}

	lo.Println("Queue system tables and testing mode column created successfully")

	// Add app.testing_mode setting if it doesn't exist
	if _, err := db.Exec(`
		INSERT INTO settings (key, value)
		VALUES ('app.testing_mode', 'false')
		ON CONFLICT (key) DO NOTHING;
	`); err != nil {
		return err
	}

	lo.Println("Added app.testing_mode setting")

	return nil
}
