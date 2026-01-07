package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V8_4_0 adds behavioral segment support with performance indexes.
// Enables segmentation by email opens, link clicks, and other engagement behaviors.
func V8_4_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	lo.Println("Adding behavioral segment indexes...")

	// Add composite indexes for behavioral queries
	// These indexes optimize time-bounded behavioral conditions like
	// "opened any email in last 30 days" or "clicked 5+ links in last 90 days"
	_, err := db.Exec(`
		-- Index for email opens with time filtering
		-- Supports: behavior.email_opens conditions
		CREATE INDEX IF NOT EXISTS idx_views_subscriber_created
		ON campaign_views(subscriber_id, created_at DESC)
		WHERE subscriber_id IS NOT NULL;

		-- Index for link clicks with time filtering
		-- Supports: behavior.link_clicks conditions
		CREATE INDEX IF NOT EXISTS idx_clicks_subscriber_created
		ON link_clicks(subscriber_id, created_at DESC)
		WHERE subscriber_id IS NOT NULL;

		-- Index for bounces with time filtering
		-- Supports: behavior.bounces conditions (for future use)
		CREATE INDEX IF NOT EXISTS idx_bounces_subscriber_created
		ON bounces(subscriber_id, created_at DESC);
	`)
	if err != nil {
		return err
	}
	lo.Println("Created behavioral segment indexes")

	lo.Println("Behavioral segment support added successfully")

	return nil
}
