package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V7_8_0 adds the historical_blocklist table for permanent blocklist storage.
// This table stores emails that should NEVER receive emails again, even if
// the subscriber record is deleted from the subscribers table.
func V7_8_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	lo.Println("Creating historical_blocklist table...")

	_, err := db.Exec(`
		-- Create historical_blocklist table
		CREATE TABLE IF NOT EXISTS historical_blocklist (
			id              SERIAL PRIMARY KEY,
			email           TEXT NOT NULL,
			reason          TEXT NOT NULL DEFAULT 'blocklisted',
			source          TEXT NOT NULL DEFAULT 'manual',
			original_subscriber_id INTEGER,
			created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		-- Create unique index on lowercase email for fast lookups
		CREATE UNIQUE INDEX IF NOT EXISTS idx_historical_blocklist_email
			ON historical_blocklist(LOWER(email));

		-- Create index on created_at for reporting
		CREATE INDEX IF NOT EXISTS idx_historical_blocklist_created_at
			ON historical_blocklist(created_at);
	`)
	if err != nil {
		return err
	}

	lo.Println("Migrating existing blocklisted subscribers to historical_blocklist...")

	// Migrate existing blocklisted subscribers to historical_blocklist
	_, err = db.Exec(`
		INSERT INTO historical_blocklist (email, reason, source, original_subscriber_id, created_at)
		SELECT email, 'blocklisted', 'migration', id, updated_at
		FROM subscribers
		WHERE status = 'blocklisted'
		ON CONFLICT (LOWER(email)) DO NOTHING;
	`)
	if err != nil {
		return err
	}

	lo.Println("Created historical_blocklist table and migrated existing blocklisted subscribers")

	return nil
}
