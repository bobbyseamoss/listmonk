package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V6_3_0 performs the DB migrations for webhook logging.
func V6_3_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	lo.Println("running migration v6.3.0: Webhook logging")

	// Create webhook_logs table for storing all incoming webhook data
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS webhook_logs (
			id                BIGSERIAL PRIMARY KEY,
			webhook_type      VARCHAR(50) NOT NULL,
			event_type        VARCHAR(100),
			request_headers   JSONB,
			request_body      TEXT NOT NULL,
			response_status   INTEGER NOT NULL,
			response_body     TEXT,
			processed         BOOLEAN NOT NULL DEFAULT false,
			error_message     TEXT,
			created_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`); err != nil {
		return err
	}

	// Create indexes for efficient webhook log lookups
	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_webhook_logs_type ON webhook_logs(webhook_type);
		CREATE INDEX IF NOT EXISTS idx_webhook_logs_event_type ON webhook_logs(event_type);
		CREATE INDEX IF NOT EXISTS idx_webhook_logs_processed ON webhook_logs(processed);
		CREATE INDEX IF NOT EXISTS idx_webhook_logs_created_at ON webhook_logs(created_at DESC);
	`); err != nil {
		return err
	}

	lo.Println("migration v6.3.0 completed successfully")
	return nil
}
