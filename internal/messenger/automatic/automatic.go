package automatic

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/models"
)

const (
	// MessengerName is the identifier for the automatic (queue-based) messenger
	MessengerName = "automatic"
)

// Automatic is the queue-based messenger that queues emails instead of sending directly
type Automatic struct {
	db  *sqlx.DB
	log *log.Logger
}

// New returns a new Automatic messenger instance
func New(db *sqlx.DB, log *log.Logger) (*Automatic, error) {
	return &Automatic{
		db:  db,
		log: log,
	}, nil
}

// Name returns the messenger's name
func (a *Automatic) Name() string {
	return MessengerName
}

// Push queues a message for queue-based delivery
// For campaigns using the automatic messenger, messages are already queued in bulk
// via the queue-campaign-emails query when the campaign starts.
// The automatic messenger should NOT be used for transactional emails.
func (a *Automatic) Push(msg models.Message) error {
	// For campaign messages using the automatic messenger, they're already queued in bulk
	// when the campaign status is changed to "running" in UpdateCampaignStatus
	// So we just return success here - no actual sending happens via Push()

	if msg.Campaign == nil {
		// This is a transactional message, which shouldn't use the automatic messenger
		a.log.Printf("WARNING: automatic messenger received transactional email - this messenger should only be used for campaigns")
		return fmt.Errorf("automatic messenger can only be used for campaigns, not transactional emails")
	}

	// For campaign messages, just return success - the queue processor handles actual sending
	return nil
}

// Flush is a no-op for the automatic messenger as there's no buffer to flush
func (a *Automatic) Flush() error {
	return nil
}

// Close closes the messenger
func (a *Automatic) Close() error {
	return nil
}
