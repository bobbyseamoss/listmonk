package queue

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/models"
)

// Processor handles the queue-based email delivery system
type Processor struct {
	db  *sqlx.DB
	cfg Config
	log *log.Logger

	// Campaign and messenger access
	getCampaign func(int) (*models.Campaign, error)
	pushEmail   func(campaignID int, subID int, serverUUID string) error

	// Control channels
	stopChan chan struct{}
	doneChan chan struct{}
}

// Config holds the queue processor configuration
type Config struct {
	// PollInterval is how often to check for new emails to send
	PollInterval time.Duration

	// BatchSize is how many emails to process in one batch
	BatchSize int

	// TimeWindowStart is the time of day to start sending (e.g., "08:00")
	TimeWindowStart string

	// TimeWindowEnd is the time of day to stop sending (e.g., "20:00")
	TimeWindowEnd string

	// SlidingWindowDuration is the duration of the sliding window (e.g., 30m)
	SlidingWindowDuration time.Duration

	// SlidingWindowLimit is the max emails per window across all servers
	SlidingWindowLimit int
}

// New creates a new queue processor
func New(db *sqlx.DB, cfg Config, log *log.Logger) *Processor {
	return &Processor{
		db:       db,
		cfg:      cfg,
		log:      log,
		stopChan: make(chan struct{}),
		doneChan: make(chan struct{}),
	}
}

// Start begins processing the queue
func (p *Processor) Start() {
	p.log.Println("starting queue processor")

	ticker := time.NewTicker(p.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := p.processQueue(); err != nil {
				p.log.Printf("error processing queue: %v", err)
			}
		case <-p.stopChan:
			p.log.Println("stopping queue processor")
			close(p.doneChan)
			return
		}
	}
}

// Stop gracefully stops the queue processor
func (p *Processor) Stop() {
	close(p.stopChan)
	<-p.doneChan
}

// isPaused checks if the queue processing is paused
func (p *Processor) isPaused() bool {
	settings, err := p.getSettings()
	if err != nil {
		p.log.Printf("error getting settings for pause check: %v", err)
		return false // Default to not paused on error
	}
	return settings.AppQueuePaused
}

// processQueue processes a batch of emails from the queue
func (p *Processor) processQueue() error {
	// Check if the queue is paused
	if p.isPaused() {
		return nil
	}

	// Check if we're within the time window
	if !p.isWithinTimeWindow() {
		return nil
	}

	// Get the next batch of emails to send
	emails, err := p.getNextBatch()
	if err != nil {
		return fmt.Errorf("error getting next batch: %w", err)
	}

	if len(emails) == 0 {
		return nil
	}

	// Get server capacities
	capacities, err := p.getServerCapacities()
	if err != nil {
		return fmt.Errorf("error getting server capacities: %w", err)
	}

	// Track in-batch usage to prevent exceeding sliding window limits within a single batch
	batchUsage := make(map[string]int)

	// Process each email
	for _, email := range emails {
		// Find a server that can send this email
		serverUUID := p.selectServer(capacities, email)
		if serverUUID == "" {
			// No server available, skip for now
			continue
		}

		// Check if this server has sliding window limits and would exceed them in this batch
		if cap, exists := capacities[serverUUID]; exists && cap.SlidingWindowLimit > 0 {
			// Check current batch usage
			if batchUsage[serverUUID] >= cap.SlidingWindowLimit {
				// This server has reached its limit for this batch, skip this email
				continue
			}
		}

		// Mark email as sending
		if err := p.markSending(email.ID, serverUUID); err != nil {
			p.log.Printf("error marking email %d as sending: %v", email.ID, err)
			continue
		}

		// Send the email (this will be implemented when we integrate with campaigns)
		if err := p.sendEmail(email, serverUUID); err != nil {
			p.log.Printf("error sending email %d: %v", email.ID, err)
			if err := p.markFailed(email.ID, err.Error()); err != nil {
				p.log.Printf("error marking email %d as failed: %v", email.ID, err)
			}
			continue
		}

		// Mark email as sent
		if err := p.markSent(email.ID); err != nil {
			p.log.Printf("error marking email %d as sent: %v", email.ID, err)
			continue
		}

		// Update server usage counters
		if err := p.incrementServerUsage(serverUUID); err != nil {
			p.log.Printf("error incrementing usage for server %s: %v", serverUUID, err)
		}

		// Update batch usage counter and capacity
		batchUsage[serverUUID]++
		if cap, exists := capacities[serverUUID]; exists {
			// Decrement daily remaining capacity
			if cap.DailyLimit > 0 {
				cap.DailyRemaining--
				if cap.DailyRemaining <= 0 {
					cap.CanSendNow = false
				}
			}
		}
	}

	return nil
}

// isWithinTimeWindow checks if the current time is within the configured sending window
func (p *Processor) isWithinTimeWindow() bool {
	// If no time window is configured, always allow sending
	if p.cfg.TimeWindowStart == "" || p.cfg.TimeWindowEnd == "" {
		return true
	}

	now := time.Now()
	currentTime := now.Format("15:04")

	// Simple string comparison works for HH:MM format
	return currentTime >= p.cfg.TimeWindowStart && currentTime <= p.cfg.TimeWindowEnd
}

// getNextBatch retrieves the next batch of emails to send from the queue
func (p *Processor) getNextBatch() ([]EmailQueueItem, error) {
	var emails []EmailQueueItem

	query := `
		SELECT id, campaign_id, subscriber_id, status, priority,
		       scheduled_at, sent_at, assigned_smtp_server_uuid,
		       retry_count, last_error, created_at, updated_at
		FROM email_queue
		WHERE status = $1
		  AND scheduled_at <= NOW()
		ORDER BY priority DESC, scheduled_at ASC
		LIMIT $2
	`

	if err := p.db.Select(&emails, query, StatusQueued, p.cfg.BatchSize); err != nil {
		return nil, err
	}

	return emails, nil
}

// getServerCapacities gets the current capacity for all SMTP servers
func (p *Processor) getServerCapacities() (map[string]*ServerCapacity, error) {
	// Get settings to find all SMTP servers
	settings, err := p.getSettings()
	if err != nil {
		return nil, err
	}

	capacities := make(map[string]*ServerCapacity)

	for _, smtp := range settings.SMTP {
		if !smtp.Enabled {
			continue
		}

		capacity := &ServerCapacity{
			UUID:       smtp.UUID,
			Name:       smtp.Name,
			DailyLimit: smtp.DailyLimit,
		}

		// Get daily usage
		usage, err := p.getDailyUsage(smtp.UUID)
		if err != nil {
			p.log.Printf("error getting daily usage for %s: %v", smtp.UUID, err)
			continue
		}
		capacity.DailyUsed = usage
		capacity.DailyRemaining = capacity.DailyLimit - capacity.DailyUsed

		// Get sliding window usage
		windowUsage, err := p.getSlidingWindowUsage(smtp.UUID)
		if err != nil {
			p.log.Printf("error getting sliding window usage for %s: %v", smtp.UUID, err)
			continue
		}
		capacity.SlidingWindowUsed = windowUsage
		capacity.SlidingWindowLimit = p.cfg.SlidingWindowLimit
		capacity.SlidingWindowPeriod = p.cfg.SlidingWindowDuration

		// Determine if server can send now
		capacity.CanSendNow = (capacity.DailyRemaining > 0 || capacity.DailyLimit == 0) &&
			(capacity.SlidingWindowUsed < capacity.SlidingWindowLimit || capacity.SlidingWindowLimit == 0)

		capacities[smtp.UUID] = capacity
	}

	return capacities, nil
}

// selectServer selects the best SMTP server to use for an email
func (p *Processor) selectServer(capacities map[string]*ServerCapacity, email EmailQueueItem) string {
	// If email already has a server assigned, try to use that
	if email.AssignedSMTPServerUUID.Valid && email.AssignedSMTPServerUUID.String != "" {
		if cap, exists := capacities[email.AssignedSMTPServerUUID.String]; exists && cap.CanSendNow {
			return email.AssignedSMTPServerUUID.String
		}
	}

	// Find the server with the most remaining capacity
	var bestServer string
	var bestCapacity *ServerCapacity

	for uuid, cap := range capacities {
		if !cap.CanSendNow {
			continue
		}

		if bestCapacity == nil || cap.DailyRemaining > bestCapacity.DailyRemaining {
			bestServer = uuid
			bestCapacity = cap
		}
	}

	return bestServer
}

// Database operations
func (p *Processor) markSending(emailID int64, serverUUID string) error {
	_, err := p.db.Exec(`
		UPDATE email_queue
		SET status = $1, assigned_smtp_server_uuid = $2, updated_at = NOW()
		WHERE id = $3
	`, StatusSending, serverUUID, emailID)
	return err
}

func (p *Processor) markSent(emailID int64) error {
	_, err := p.db.Exec(`
		UPDATE email_queue
		SET status = $1, sent_at = NOW(), updated_at = NOW()
		WHERE id = $2
	`, StatusSent, emailID)
	return err
}

func (p *Processor) markFailed(emailID int64, errMsg string) error {
	_, err := p.db.Exec(`
		UPDATE email_queue
		SET status = $1, last_error = $2, retry_count = retry_count + 1, updated_at = NOW()
		WHERE id = $3
	`, StatusFailed, sql.NullString{String: errMsg, Valid: true}, emailID)
	return err
}

func (p *Processor) getDailyUsage(serverUUID string) (int, error) {
	var usage int
	err := p.db.Get(&usage, `
		SELECT COALESCE(emails_sent, 0)
		FROM smtp_daily_usage
		WHERE smtp_server_uuid = $1 AND usage_date = CURRENT_DATE
	`, serverUUID)

	if err == sql.ErrNoRows {
		return 0, nil
	}
	return usage, err
}

func (p *Processor) getSlidingWindowUsage(serverUUID string) (int, error) {
	var state SMTPRateLimitState
	err := p.db.Get(&state, `
		SELECT id, smtp_server_uuid, window_start, emails_in_window, created_at, updated_at
		FROM smtp_rate_limit_state
		WHERE smtp_server_uuid = $1
	`, serverUUID)

	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	// Check if window has expired
	if time.Since(state.WindowStart) > p.cfg.SlidingWindowDuration {
		return 0, nil
	}

	return state.EmailsInWindow, nil
}

func (p *Processor) incrementServerUsage(serverUUID string) error {
	tx, err := p.db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Increment daily usage
	_, err = tx.Exec(`
		INSERT INTO smtp_daily_usage (smtp_server_uuid, usage_date, emails_sent, created_at, updated_at)
		VALUES ($1, CURRENT_DATE, 1, NOW(), NOW())
		ON CONFLICT (smtp_server_uuid, usage_date)
		DO UPDATE SET emails_sent = smtp_daily_usage.emails_sent + 1, updated_at = NOW()
	`, serverUUID)
	if err != nil {
		return err
	}

	// Update sliding window (only if configured)
	if p.cfg.SlidingWindowDuration > 0 {
		now := time.Now()
		// Convert Go duration to PostgreSQL interval format (e.g., "30m" -> "30 minutes")
		intervalStr := p.cfg.SlidingWindowDuration.String()

		_, err = tx.Exec(`
			INSERT INTO smtp_rate_limit_state (smtp_server_uuid, window_start, emails_in_window, created_at, updated_at)
			VALUES ($1, $2, 1, NOW(), NOW())
			ON CONFLICT (smtp_server_uuid)
			DO UPDATE SET
				window_start = CASE
					WHEN NOW() - smtp_rate_limit_state.window_start > $3::interval
					THEN $2
					ELSE smtp_rate_limit_state.window_start
				END,
				emails_in_window = CASE
					WHEN NOW() - smtp_rate_limit_state.window_start > $3::interval
					THEN 1
					ELSE smtp_rate_limit_state.emails_in_window + 1
				END,
				updated_at = NOW()
		`, serverUUID, now, intervalStr)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (p *Processor) getSettings() (models.Settings, error) {
	var (
		b   []byte
		out models.Settings
	)

	// Query aggregated settings from database (same query as get-settings in queries.sql)
	if err := p.db.Get(&b, `SELECT JSON_OBJECT_AGG(key, value) AS settings FROM (SELECT * FROM settings ORDER BY key) t`); err != nil {
		return out, fmt.Errorf("error fetching settings: %w", err)
	}

	// Parse settings from JSON
	if err := json.Unmarshal(b, &out); err != nil {
		return out, fmt.Errorf("error unmarshaling settings: %w", err)
	}

	return out, nil
}

func (p *Processor) sendEmail(email EmailQueueItem, serverUUID string) error {
	// Check if this is a callback-based send (integrated with campaign manager)
	if p.pushEmail != nil {
		// Use the campaign manager's push logic
		return p.pushEmail(email.CampaignID, email.SubscriberID, serverUUID)
	}

	// Fallback: In testing mode or when no pushEmail callback is set,
	// just return success (email will be marked as sent by the caller)
	p.log.Printf("sendEmail called for campaign %d, subscriber %d, server %s (no push callback, assuming testing mode)",
		email.CampaignID, email.SubscriberID, serverUUID)
	return nil
}

// GetQueueStats returns statistics about the email queue
func (p *Processor) GetQueueStats() (QueueStats, error) {
	var stats QueueStats

	// Get counts by status
	err := p.db.Get(&stats.TotalQueued, `SELECT COUNT(*) FROM email_queue WHERE status = $1`, StatusQueued)
	if err != nil {
		return stats, err
	}

	err = p.db.Get(&stats.TotalSending, `SELECT COUNT(*) FROM email_queue WHERE status = $1`, StatusSending)
	if err != nil {
		return stats, err
	}

	err = p.db.Get(&stats.TotalSent, `SELECT COUNT(*) FROM email_queue WHERE status = $1`, StatusSent)
	if err != nil {
		return stats, err
	}

	err = p.db.Get(&stats.TotalFailed, `SELECT COUNT(*) FROM email_queue WHERE status = $1`, StatusFailed)
	if err != nil {
		return stats, err
	}

	// Get oldest and newest scheduled times
	type TimeRange struct {
		Oldest *time.Time `db:"oldest"`
		Newest *time.Time `db:"newest"`
	}
	var tr TimeRange
	err = p.db.Get(&tr, `
		SELECT
			MIN(scheduled_at) as oldest,
			MAX(scheduled_at) as newest
		FROM email_queue
		WHERE status IN ($1, $2)
	`, StatusQueued, StatusSending)
	if err != nil && err != sql.ErrNoRows {
		return stats, err
	}
	stats.OldestScheduled = tr.Oldest
	stats.NewestScheduled = tr.Newest

	return stats, nil
}
