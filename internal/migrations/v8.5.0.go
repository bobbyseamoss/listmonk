package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V8_5_0 adds Shopify order history tables for complete order tracking.
// Enables segmentation based on purchase history, products, and order data.
func V8_5_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	lo.Println("Creating Shopify order history tables...")

	// Create shopify_orders table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS shopify_orders (
			id                  SERIAL PRIMARY KEY,
			subscriber_id       INTEGER NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE,
			shopify_order_id    TEXT NOT NULL UNIQUE,
			order_number        TEXT,
			order_name          TEXT,
			created_at          TIMESTAMP WITH TIME ZONE NOT NULL,
			processed_at        TIMESTAMP WITH TIME ZONE,
			total_price         NUMERIC(12,2),
			subtotal_price      NUMERIC(12,2),
			total_tax           NUMERIC(12,2),
			total_discounts     NUMERIC(12,2),
			currency            TEXT,
			financial_status    TEXT,
			fulfillment_status  TEXT,
			tags                TEXT[],
			note                TEXT,
			synced_at           TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_shopify_orders_subscriber ON shopify_orders(subscriber_id);
		CREATE INDEX IF NOT EXISTS idx_shopify_orders_created ON shopify_orders(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_shopify_orders_financial_status ON shopify_orders(financial_status);
	`)
	if err != nil {
		return err
	}
	lo.Println("Created shopify_orders table")

	// Create shopify_order_line_items table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS shopify_order_line_items (
			id                   SERIAL PRIMARY KEY,
			order_id             INTEGER NOT NULL REFERENCES shopify_orders(id) ON DELETE CASCADE,
			shopify_line_item_id TEXT NOT NULL,
			product_id           TEXT,
			product_title        TEXT,
			variant_id           TEXT,
			variant_title        TEXT,
			sku                  TEXT,
			quantity             INTEGER NOT NULL DEFAULT 1,
			price                NUMERIC(12,2),
			total_discount       NUMERIC(12,2),
			product_type         TEXT,
			vendor               TEXT,
			properties           JSONB
		);

		CREATE INDEX IF NOT EXISTS idx_shopify_line_items_order ON shopify_order_line_items(order_id);
		CREATE INDEX IF NOT EXISTS idx_shopify_line_items_product ON shopify_order_line_items(product_id);
		CREATE INDEX IF NOT EXISTS idx_shopify_line_items_title ON shopify_order_line_items(product_title);
	`)
	if err != nil {
		return err
	}
	lo.Println("Created shopify_order_line_items table")

	lo.Println("Shopify order history tables created successfully")

	return nil
}
