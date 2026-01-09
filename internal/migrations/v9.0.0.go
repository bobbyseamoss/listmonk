package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V9_0_0 adds sign-up forms feature with Klaviyo-like functionality.
// Includes form builder configuration, targeting, triggers, and analytics.
func V9_0_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	lo.Println("Creating sign-up forms tables...")

	// Create form_type enum
	_, err := db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'form_type') THEN
				CREATE TYPE form_type AS ENUM ('popup', 'flyout', 'fullpage', 'embed', 'banner');
			END IF;
		END$$;
	`)
	if err != nil {
		return err
	}
	lo.Println("Created form_type enum")

	// Create form_status enum
	_, err = db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'form_status') THEN
				CREATE TYPE form_status AS ENUM ('draft', 'active', 'paused', 'archived');
			END IF;
		END$$;
	`)
	if err != nil {
		return err
	}
	lo.Println("Created form_status enum")

	// Create sign_up_forms table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sign_up_forms (
			id              SERIAL PRIMARY KEY,
			uuid            UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
			name            TEXT NOT NULL,
			description     TEXT NOT NULL DEFAULT '',
			form_type       form_type NOT NULL DEFAULT 'popup',
			status          form_status NOT NULL DEFAULT 'draft',

			-- Visual configuration (JSONB for flexible block-based structure)
			-- Stores the drag-drop builder configuration
			body_source     JSONB NOT NULL DEFAULT '{}',

			-- Compiled CSS for the form
			custom_css      TEXT NOT NULL DEFAULT '',

			-- Multi-step configuration
			steps           JSONB NOT NULL DEFAULT '[]',

			-- Form behavior settings
			-- Includes: position, animation, overlay, close_button, success_action, etc.
			settings        JSONB NOT NULL DEFAULT '{}',

			-- Targeting configuration
			-- Includes: url_patterns, utm_params, devices, geo_countries, exclude_subscribers
			targeting       JSONB NOT NULL DEFAULT '{}',

			-- Trigger configuration
			-- Includes: type (exit_intent, time_delay, scroll_depth, page_count, manual, immediate)
			triggers        JSONB NOT NULL DEFAULT '{}',

			-- Frequency/suppression settings
			-- Includes: show_again_after_days, suppress_after_submit, max_shows_per_session
			frequency       JSONB NOT NULL DEFAULT '{}',

			-- Associated lists (subscribers added to these lists on form submission)
			list_ids        INTEGER[] NOT NULL DEFAULT '{}',

			-- Coupon configuration
			-- Includes: enabled, type (static/unique), static_code, codes_used
			coupon_config   JSONB NOT NULL DEFAULT '{}',

			created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_sign_up_forms_uuid ON sign_up_forms(uuid);
		CREATE INDEX IF NOT EXISTS idx_sign_up_forms_status ON sign_up_forms(status);
		CREATE INDEX IF NOT EXISTS idx_sign_up_forms_type ON sign_up_forms(form_type);
		CREATE INDEX IF NOT EXISTS idx_sign_up_forms_created_at ON sign_up_forms(created_at);
	`)
	if err != nil {
		return err
	}
	lo.Println("Created sign_up_forms table")

	// Create form_submissions table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS form_submissions (
			id              BIGSERIAL PRIMARY KEY,
			form_id         INTEGER NOT NULL REFERENCES sign_up_forms(id) ON DELETE CASCADE,
			subscriber_id   INTEGER REFERENCES subscribers(id) ON DELETE SET NULL,

			-- Submission data
			email           TEXT NOT NULL,
			data            JSONB NOT NULL DEFAULT '{}',

			-- Attribution/tracking
			page_url        TEXT,
			referrer        TEXT,
			utm_source      TEXT,
			utm_medium      TEXT,
			utm_campaign    TEXT,
			utm_term        TEXT,
			utm_content     TEXT,

			-- Device info
			device_type     TEXT,
			user_agent      TEXT,
			ip_address      TEXT,
			country         TEXT,

			-- Coupon given (if any)
			coupon_code     TEXT,

			created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_form_submissions_form_id ON form_submissions(form_id);
		CREATE INDEX IF NOT EXISTS idx_form_submissions_subscriber_id ON form_submissions(subscriber_id);
		CREATE INDEX IF NOT EXISTS idx_form_submissions_email ON form_submissions(email);
		CREATE INDEX IF NOT EXISTS idx_form_submissions_created_at ON form_submissions(created_at);
	`)
	if err != nil {
		return err
	}
	lo.Println("Created form_submissions table")

	// Create form_impressions table (for analytics)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS form_impressions (
			id              BIGSERIAL PRIMARY KEY,
			form_id         INTEGER NOT NULL REFERENCES sign_up_forms(id) ON DELETE CASCADE,

			-- Session tracking
			session_id      TEXT NOT NULL,
			visitor_id      TEXT,

			-- Context
			page_url        TEXT,
			device_type     TEXT,
			country         TEXT,

			-- What happened
			shown_at        TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			closed_at       TIMESTAMP WITH TIME ZONE,
			submitted_at    TIMESTAMP WITH TIME ZONE,
			step_reached    INTEGER DEFAULT 1
		);

		CREATE INDEX IF NOT EXISTS idx_form_impressions_form_id ON form_impressions(form_id);
		CREATE INDEX IF NOT EXISTS idx_form_impressions_session_id ON form_impressions(session_id);
		CREATE INDEX IF NOT EXISTS idx_form_impressions_shown_at ON form_impressions(shown_at);
	`)
	if err != nil {
		return err
	}
	lo.Println("Created form_impressions table")

	// Create form_coupon_codes table (for unique coupon codes pool)
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS form_coupon_codes (
			id              SERIAL PRIMARY KEY,
			form_id         INTEGER NOT NULL REFERENCES sign_up_forms(id) ON DELETE CASCADE,
			code            TEXT NOT NULL,
			used            BOOLEAN NOT NULL DEFAULT FALSE,
			used_by_email   TEXT,
			used_at         TIMESTAMP WITH TIME ZONE,
			created_at      TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		CREATE UNIQUE INDEX IF NOT EXISTS idx_form_coupon_codes_form_code ON form_coupon_codes(form_id, code);
		CREATE INDEX IF NOT EXISTS idx_form_coupon_codes_available ON form_coupon_codes(form_id, used) WHERE used = FALSE;
	`)
	if err != nil {
		return err
	}
	lo.Println("Created form_coupon_codes table")

	lo.Println("Sign-up forms tables created successfully")

	return nil
}
