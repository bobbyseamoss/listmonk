package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V7_7_0 adds domain routing settings for configuring which SMTP servers
// handle specific email domains (e.g., Gmail, Yahoo, iCloud).
func V7_7_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	lo.Println("Adding domain routing settings...")

	// Add domain_routing settings - empty object means all domains use round-robin
	if _, err := db.Exec(`
		INSERT INTO settings (key, value) VALUES
			('domain_routing', '{}')
		ON CONFLICT (key) DO NOTHING;
	`); err != nil {
		return err
	}

	lo.Println("Added domain routing settings")

	return nil
}
