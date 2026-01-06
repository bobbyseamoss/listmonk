package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V8_1_0 adds Shopify customer sync settings.
func V8_1_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	lo.Println("Adding Shopify customer sync settings...")

	// Update the nested shopify settings object with new fields
	if _, err := db.Exec(`
		UPDATE settings
		SET value = value || '{"customer_sync_enabled": false, "customer_sync_list_id": 0, "last_customer_sync": null, "customer_sync_in_progress": false}'::jsonb
		WHERE key = 'shopify';
	`); err != nil {
		return err
	}

	// Add flat keys for compatibility
	if _, err := db.Exec(`
		INSERT INTO settings (key, value) VALUES
			('shopify.customer_sync_enabled', 'false'),
			('shopify.customer_sync_list_id', '0'),
			('shopify.last_customer_sync', 'null'),
			('shopify.customer_sync_in_progress', 'false')
		ON CONFLICT (key) DO NOTHING;
	`); err != nil {
		return err
	}

	lo.Println("Added Shopify customer sync settings")

	return nil
}
