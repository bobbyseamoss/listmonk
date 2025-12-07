package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V7_4_0 adds spintax settings to enable text variation in emails.
// Spintax allows creating multiple variations of text using {option1|option2|option3} syntax.
// Also adds AI-powered spintax generation using Claude API.
func V7_4_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	lo.Println("Adding spintax settings...")

	// Add spintax settings
	if _, err := db.Exec(`
		INSERT INTO settings (key, value) VALUES
			('spintax', '{"enabled": false, "ai": {"enabled": false, "api_key": "", "variation_level": 3}}')
		ON CONFLICT (key) DO NOTHING;
	`); err != nil {
		return err
	}

	lo.Println("Added spintax settings")

	return nil
}
