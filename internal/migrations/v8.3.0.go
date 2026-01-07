package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V8_3_0 adds Klaviyo-style Segments feature.
// Segments are dynamic subscriber groups defined by conditions that are evaluated in real-time.
func V8_3_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	lo.Println("Adding Segments feature...")

	// Create segments table
	_, err := db.Exec(`
		-- segments: Dynamic subscriber groups based on conditions
		CREATE TABLE IF NOT EXISTS segments (
			id              SERIAL PRIMARY KEY,
			uuid            UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
			name            TEXT NOT NULL,
			description     TEXT NOT NULL DEFAULT '',

			-- Segment conditions stored as JSONB
			-- Structure: { "operator": "and|or", "conditions": [...] }
			conditions      JSONB NOT NULL DEFAULT '{"operator": "and", "conditions": []}',

			-- Cache control for subscriber count
			cached_count    INT,
			cached_at       TIMESTAMP WITH TIME ZONE,

			-- Metadata
			tags            VARCHAR(100)[],
			created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		-- Indexes for segments
		CREATE INDEX IF NOT EXISTS idx_segments_uuid ON segments(uuid);
		CREATE INDEX IF NOT EXISTS idx_segments_name ON segments(name);
		CREATE INDEX IF NOT EXISTS idx_segments_created_at ON segments(created_at);
	`)
	if err != nil {
		return err
	}
	lo.Println("Created segments table")

	// Create campaign_segments junction table
	_, err = db.Exec(`
		-- campaign_segments: Links campaigns to segments (like campaign_lists)
		CREATE TABLE IF NOT EXISTS campaign_segments (
			id              BIGSERIAL PRIMARY KEY,
			campaign_id     INTEGER NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE ON UPDATE CASCADE,

			-- Segments may be deleted, so segment_id is nullable
			-- and a copy of the original segment name is maintained here.
			segment_id      INTEGER NULL REFERENCES segments(id) ON DELETE SET NULL ON UPDATE CASCADE,
			segment_name    TEXT NOT NULL DEFAULT ''
		);

		-- Indexes for campaign_segments
		CREATE UNIQUE INDEX IF NOT EXISTS idx_camp_segments_unique ON campaign_segments (campaign_id, segment_id);
		CREATE INDEX IF NOT EXISTS idx_camp_segments_camp_id ON campaign_segments(campaign_id);
		CREATE INDEX IF NOT EXISTS idx_camp_segments_segment_id ON campaign_segments(segment_id);
	`)
	if err != nil {
		return err
	}
	lo.Println("Created campaign_segments table")

	// Add target_type column to campaigns
	_, err = db.Exec(`
		-- Add target_type to campaigns: 'lists' (default) or 'segments'
		ALTER TABLE campaigns
			ADD COLUMN IF NOT EXISTS target_type VARCHAR(20) NOT NULL DEFAULT 'lists';
	`)
	if err != nil {
		return err
	}
	lo.Println("Added target_type column to campaigns")

	// Add GIN index on subscribers.attribs for efficient JSONB queries
	_, err = db.Exec(`
		-- GIN index for efficient JSONB queries on subscriber attributes
		CREATE INDEX IF NOT EXISTS idx_subs_attribs_gin ON subscribers USING GIN (attribs);
	`)
	if err != nil {
		return err
	}
	lo.Println("Added GIN index on subscribers.attribs")

	lo.Println("Segments feature added successfully")

	return nil
}
