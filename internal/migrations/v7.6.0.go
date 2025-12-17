package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V7_6_0 adds email header settings for abuse reporting and Google Postmaster Tools.
// - app.abuse_email: Email address for spam/abuse reports (X-Report-Abuse header)
// - app.feedback_sender_id: Sender ID for Google Postmaster Tools Feedback-ID header
func V7_6_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	lo.Println("Adding email header settings...")

	// Add email header settings
	if _, err := db.Exec(`
		INSERT INTO settings (key, value) VALUES
			('app.abuse_email', '""'),
			('app.feedback_sender_id', '""')
		ON CONFLICT (key) DO NOTHING;
	`); err != nil {
		return err
	}

	lo.Println("Added email header settings (abuse_email, feedback_sender_id)")

	return nil
}
