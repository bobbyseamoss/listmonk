package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V8_2_0 adds the test email first setting.
// When set, this email will be sent before all others when starting any campaign.
func V8_2_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	lo.Println("Adding test email first setting...")

	// Add test email first setting with default value
	if _, err := db.Exec(`
		INSERT INTO settings (key, value) VALUES
			('app.test_email_first', '"adamgendreau@gmail.com"')
		ON CONFLICT (key) DO NOTHING;
	`); err != nil {
		return err
	}

	lo.Println("Added test email first setting")

	return nil
}
