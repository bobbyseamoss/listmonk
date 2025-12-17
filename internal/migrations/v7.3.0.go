package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V7_3_0 adds bypass_time_window column to campaigns table.
// This allows individual campaigns to bypass the global sending time window,
// which is useful for running test campaigns outside of normal sending hours.
func V7_3_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	lo.Println("Adding bypass_time_window column to campaigns table...")

	// Add bypass_time_window column to campaigns table
	if _, err := db.Exec(`
		ALTER TABLE campaigns
			ADD COLUMN IF NOT EXISTS bypass_time_window BOOLEAN NOT NULL DEFAULT FALSE;
	`); err != nil {
		return err
	}

	lo.Println("Added bypass_time_window column to campaigns table")

	return nil
}
