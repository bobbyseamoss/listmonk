package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V8_0_0 adds onsite tracking tables for Klaviyo-like website visitor tracking.
// This enables tracking of known/identified visitors on the website.
func V8_0_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	lo.Println("Creating onsite tracking tables...")

	// Create site_events table for storing all tracked events
	_, err := db.Exec(`
		-- site_events: All tracked events for known visitors
		CREATE TABLE IF NOT EXISTS site_events (
			id              BIGSERIAL PRIMARY KEY,
			subscriber_id   INTEGER REFERENCES subscribers(id) ON DELETE SET NULL,
			session_id      UUID NOT NULL,
			event_type      TEXT NOT NULL,
			event_name      TEXT,
			page_url        TEXT,
			page_title      TEXT,
			referrer        TEXT,
			properties      JSONB NOT NULL DEFAULT '{}',
			user_agent      TEXT,
			ip_address      TEXT,
			created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		-- Indexes for site_events
		CREATE INDEX IF NOT EXISTS idx_site_events_subscriber_id
			ON site_events(subscriber_id);
		CREATE INDEX IF NOT EXISTS idx_site_events_session_id
			ON site_events(session_id);
		CREATE INDEX IF NOT EXISTS idx_site_events_event_type
			ON site_events(event_type);
		CREATE INDEX IF NOT EXISTS idx_site_events_created_at
			ON site_events(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_site_events_subscriber_created
			ON site_events(subscriber_id, created_at DESC);
	`)
	if err != nil {
		return err
	}

	lo.Println("Created site_events table")

	// Create known_browsers table for linking browser IDs to subscribers
	_, err = db.Exec(`
		-- known_browsers: Links browser IDs to subscribers
		CREATE TABLE IF NOT EXISTS known_browsers (
			id              SERIAL PRIMARY KEY,
			browser_id      TEXT NOT NULL UNIQUE,
			subscriber_id   INTEGER REFERENCES subscribers(id) ON DELETE CASCADE,
			identified_via  TEXT NOT NULL,
			campaign_id     INTEGER REFERENCES campaigns(id) ON DELETE SET NULL,
			created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			last_seen_at    TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		-- Indexes for known_browsers
		CREATE INDEX IF NOT EXISTS idx_known_browsers_subscriber_id
			ON known_browsers(subscriber_id);
		CREATE INDEX IF NOT EXISTS idx_known_browsers_last_seen
			ON known_browsers(last_seen_at DESC);
	`)
	if err != nil {
		return err
	}

	lo.Println("Created known_browsers table")

	// Insert default onsite_tracking settings
	_, err = db.Exec(`
		INSERT INTO settings (key, value) VALUES
		('onsite_tracking', '{"enabled": false, "track_page_views": true, "track_products": true, "allowed_domains": [], "identity_param": "_lmx", "cookie_name": "lm_browser", "cookie_expiry_days": 365, "record_ip_address": false, "retention_days": 90}')
		ON CONFLICT (key) DO NOTHING;
	`)
	if err != nil {
		return err
	}

	lo.Println("Added default onsite_tracking settings")

	lo.Println("Onsite tracking tables created successfully")

	return nil
}
