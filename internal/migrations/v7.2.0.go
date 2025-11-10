package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V7_2_0 adds internet_message_id column to Azure tracking tables for proper engagement attribution.
// Azure uses different message IDs for delivery vs engagement events, but internetMessageId is consistent.
func V7_2_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	lo.Println("Adding internet_message_id columns for proper Azure engagement attribution...")

	// Add internet_message_id to azure_message_tracking table
	if _, err := db.Exec(`
		ALTER TABLE azure_message_tracking
			ADD COLUMN IF NOT EXISTS internet_message_id TEXT;

		-- Create index for fast lookups by internet_message_id
		CREATE INDEX IF NOT EXISTS idx_azure_msg_tracking_internet_msg_id
			ON azure_message_tracking(internet_message_id);
	`); err != nil {
		return err
	}

	lo.Println("Added internet_message_id column to azure_message_tracking table")

	// Add internet_message_id to azure_engagement_events table
	if _, err := db.Exec(`
		ALTER TABLE azure_engagement_events
			ADD COLUMN IF NOT EXISTS internet_message_id TEXT;

		-- Create index for fast lookups by internet_message_id
		CREATE INDEX IF NOT EXISTS idx_azure_engagement_internet_msg_id
			ON azure_engagement_events(internet_message_id);
	`); err != nil {
		return err
	}

	lo.Println("Added internet_message_id column to azure_engagement_events table")

	return nil
}
