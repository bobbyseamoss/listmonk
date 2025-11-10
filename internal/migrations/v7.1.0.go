package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V7_1_0 adds timezone setting and auto-pause/resume functionality for campaigns.
func V7_1_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	lo.Println("Adding timezone setting and auto-pause/resume functionality...")

	// Add timezone setting with America/New_York as default
	if _, err := db.Exec(`
		INSERT INTO settings (key, value) VALUES
			('app.timezone', '"America/New_York"')
		ON CONFLICT (key) DO NOTHING;
	`); err != nil {
		return err
	}

	lo.Println("Added timezone setting (app.timezone)")

	// Add columns to campaigns table to track auto-pause/resume state
	if _, err := db.Exec(`
		ALTER TABLE campaigns
			ADD COLUMN IF NOT EXISTS auto_paused BOOLEAN NOT NULL DEFAULT FALSE,
			ADD COLUMN IF NOT EXISTS auto_paused_at TIMESTAMP WITH TIME ZONE;

		CREATE INDEX IF NOT EXISTS idx_campaigns_auto_paused ON campaigns(auto_paused);
	`); err != nil {
		return err
	}

	lo.Println("Added auto_paused tracking columns to campaigns table")

	return nil
}
