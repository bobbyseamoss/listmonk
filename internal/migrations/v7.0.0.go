package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V7_0_0 adds Shopify purchase attribution tracking.
func V7_0_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	lo.Println("running migration v7.0.0")

	// Create purchase_attributions table.
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS purchase_attributions (
			id               BIGSERIAL PRIMARY KEY,
			campaign_id      INTEGER NULL REFERENCES campaigns(id) ON DELETE SET NULL,
			subscriber_id    INTEGER NULL REFERENCES subscribers(id) ON DELETE SET NULL,
			order_id         TEXT NOT NULL,
			order_number     TEXT,
			customer_email   TEXT NOT NULL,
			total_price      DECIMAL(10,2),
			currency         TEXT,
			attributed_via   TEXT,
			confidence       TEXT,
			shopify_data     JSONB NOT NULL DEFAULT '{}',
			created_at       TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
	`); err != nil {
		return err
	}

	// Create indexes for performance.
	if _, err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_purchases_campaign_id ON purchase_attributions(campaign_id);
		CREATE INDEX IF NOT EXISTS idx_purchases_subscriber_id ON purchase_attributions(subscriber_id);
		CREATE INDEX IF NOT EXISTS idx_purchases_order_id ON purchase_attributions(order_id);
		CREATE INDEX IF NOT EXISTS idx_purchases_customer_email ON purchase_attributions(customer_email);
		CREATE INDEX IF NOT EXISTS idx_purchases_created_at ON purchase_attributions(created_at);
	`); err != nil {
		return err
	}

	// Add Shopify settings (both nested and flat keys for compatibility).
	if _, err := db.Exec(`
		INSERT INTO settings (key, value) VALUES
		('shopify', '{"enabled": false, "webhook_secret": "", "attribution_window_days": 7}'),
		('shopify.enabled', 'false'),
		('shopify.webhook_secret', '""'),
		('shopify.attribution_window_days', '7')
		ON CONFLICT (key) DO NOTHING;
	`); err != nil {
		return err
	}

	return nil
}
