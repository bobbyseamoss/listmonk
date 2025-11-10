package queue

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
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

// SetPushEmailCallback sets the callback function for actually sending emails
func (p *Processor) SetPushEmailCallback(fn func(campaignID int, subID int, serverUUID string) error) {
	p.pushEmail = fn
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

// StartAutoPauseScheduler starts the automatic pause/resume scheduler
// This runs every minute and pauses/resumes campaigns based on the time window
func (p *Processor) StartAutoPauseScheduler() {
	p.log.Println("starting auto-pause scheduler for time window enforcement")

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	// Run immediately on start
	if err := p.autoPauseResumeCampaigns(); err != nil {
		p.log.Printf("error in auto-pause/resume: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := p.autoPauseResumeCampaigns(); err != nil {
				p.log.Printf("error in auto-pause/resume: %v", err)
			}
		case <-p.stopChan:
			p.log.Println("stopping auto-pause scheduler")
			return
		}
	}
}

// autoPauseResumeCampaigns automatically pauses campaigns when outside time window
// and resumes them when inside time window
func (p *Processor) autoPauseResumeCampaigns() error {
	// Check if time window is configured
	if p.cfg.TimeWindowStart == "" || p.cfg.TimeWindowEnd == "" {
		// No time window configured, nothing to do
		return nil
	}

	// Check if we're currently within the time window
	withinWindow := p.isWithinTimeWindow()

	if withinWindow {
		// We're inside the window - resume any auto-paused campaigns
		return p.resumeAutoPausedCampaigns()
	} else {
		// We're outside the window - auto-pause running campaigns
		return p.autoPauseRunningCampaigns()
	}
}

// autoPauseRunningCampaigns pauses all running campaigns when outside time window
func (p *Processor) autoPauseRunningCampaigns() error {
	// Update all running queue-based campaigns to paused status
	// Only pause campaigns that aren't already auto-paused
	res, err := p.db.Exec(`
		UPDATE campaigns
		SET status = 'paused',
		    auto_paused = true,
		    auto_paused_at = NOW(),
		    updated_at = NOW()
		WHERE status = 'running'
		  AND use_queue = true
		  AND auto_paused = false
	`)
	if err != nil {
		return fmt.Errorf("error auto-pausing campaigns: %w", err)
	}

	count, _ := res.RowsAffected()
	if count > 0 {
		p.log.Printf("⏸️  auto-paused %d running campaign(s) - outside time window", count)
	}

	return nil
}

// resumeAutoPausedCampaigns resumes ALL paused queue-based campaigns when inside time window
func (p *Processor) resumeAutoPausedCampaigns() error {
	// First, get all paused queue-based campaign IDs
	var pausedCampaignIDs []int
	err := p.db.Select(&pausedCampaignIDs, `
		SELECT id FROM campaigns
		WHERE status = 'paused' AND use_queue = true
	`)
	if err != nil {
		return fmt.Errorf("error fetching paused campaigns: %w", err)
	}

	if len(pausedCampaignIDs) == 0 {
		return nil // No paused campaigns to resume
	}

	// For each paused campaign, requeue cancelled emails
	for _, campID := range pausedCampaignIDs {
		res, err := p.db.Exec(`
			UPDATE email_queue
			SET status = 'queued', updated_at = NOW()
			WHERE campaign_id = $1
			  AND status = 'cancelled'
			  AND id NOT IN (
				SELECT id FROM email_queue
				WHERE campaign_id = $1 AND status = 'sent'
			)
		`, campID)
		if err != nil {
			p.log.Printf("error requeuing cancelled emails for campaign %d: %v", campID, err)
			continue
		}

		requeuedCount, _ := res.RowsAffected()
		if requeuedCount > 0 {
			p.log.Printf("📬 requeued %d cancelled emails for campaign %d", requeuedCount, campID)
		}
	}

	// Update ALL paused queue-based campaigns back to running status
	// This includes both auto-paused and manually paused campaigns
	res, err := p.db.Exec(`
		UPDATE campaigns
		SET status = 'running',
		    auto_paused = false,
		    auto_paused_at = NULL,
		    updated_at = NOW()
		WHERE status = 'paused'
		  AND use_queue = true
	`)
	if err != nil {
		return fmt.Errorf("error resuming campaigns: %w", err)
	}

	count, _ := res.RowsAffected()
	if count > 0 {
		p.log.Printf("▶️  auto-resumed %d paused campaign(s) - entered time window", count)
	}

	return nil
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

	// Get settings for concurrency and message rate
	settings, err := p.getSettings()
	if err != nil {
		return fmt.Errorf("error getting settings for rate limiting: %w", err)
	}

	// Track in-batch usage to prevent exceeding sliding window limits within a single batch
	batchUsage := make(map[string]int)

	// Use settings-based concurrency for the semaphore
	// This controls how many emails can be sent simultaneously
	maxConcurrent := settings.AppConcurrency
	if maxConcurrent < 1 {
		maxConcurrent = 1 // Default to at least 1
	}
	semaphore := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	// Create a rate limiter based on message rate setting AND account-wide limits
	// CRITICAL: Account-wide limits take precedence over AppMessageRate
	var rateLimiter *time.Ticker
	var delayBetweenMessages time.Duration

	// Start with AppMessageRate delay (if set)
	if settings.AppMessageRate > 0 {
		delayBetweenMessages = time.Second / time.Duration(settings.AppMessageRate)
	}

	// CRITICAL: Check account-wide per-minute limit and use LONGER delay if needed
	if settings.AppAccountRateLimitPerMinute > 0 {
		// Calculate delay: 60 seconds / X messages = Y seconds per message
		minDelayFromMinuteLimit := time.Minute / time.Duration(settings.AppAccountRateLimitPerMinute)
		if delayBetweenMessages == 0 || minDelayFromMinuteLimit > delayBetweenMessages {
			p.log.Printf("⚠️  rate capped by account-wide minute limit: %d/min → 1 message every %v",
				settings.AppAccountRateLimitPerMinute, minDelayFromMinuteLimit)
			delayBetweenMessages = minDelayFromMinuteLimit
		}
	}

	// CRITICAL: Check account-wide per-hour limit and use LONGER delay if needed
	if settings.AppAccountRateLimitPerHour > 0 {
		// Calculate delay: 3600 seconds / X messages = Y seconds per message
		minDelayFromHourLimit := time.Hour / time.Duration(settings.AppAccountRateLimitPerHour)
		if delayBetweenMessages == 0 || minDelayFromHourLimit > delayBetweenMessages {
			p.log.Printf("⚠️  rate capped by account-wide hour limit: %d/hour → 1 message every %v",
				settings.AppAccountRateLimitPerHour, minDelayFromHourLimit)
			delayBetweenMessages = minDelayFromHourLimit
		}
	}

	if delayBetweenMessages > 0 {
		rateLimiter = time.NewTicker(delayBetweenMessages)
		defer rateLimiter.Stop()
		messagesPerSecond := float64(time.Second) / float64(delayBetweenMessages)
		p.log.Printf("✓ rate limiter: %.2f messages/sec (delay: %v between sends)", messagesPerSecond, delayBetweenMessages)
	}

	// Process each email concurrently
	for _, email := range emails {
		// Find a server that can send this email
		serverUUID := p.selectServer(capacities, email)
		if serverUUID == "" {
			// No server available, skip for now
			p.log.Printf("⚠️  no SMTP server available for email %d (campaign %d, subscriber %d) - all servers at capacity",
				email.ID, email.CampaignID, email.SubscriberID)
			continue
		}

		// Log server selection
		if cap, exists := capacities[serverUUID]; exists {
			p.log.Printf("📧 email %d (campaign %d, subscriber %d) → SMTP server '%s' | capacity: %d/%d daily remaining",
				email.ID, email.CampaignID, email.SubscriberID, cap.Name, cap.DailyRemaining, cap.DailyLimit)
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
			// If email is already being processed, silently skip (expected with concurrent polling)
			// Only log actual database errors
			if err.Error() != fmt.Sprintf("email %d already being processed", email.ID) {
				p.log.Printf("error marking email %d as sending: %v", email.ID, err)
			}
			continue
		}

		// Update batch usage counter NOW (before spawning goroutine)
		batchUsage[serverUUID]++

		// Wait for rate limiter to allow next send
		// This ensures we respect the configured message rate (messages per second)
		if rateLimiter != nil {
			<-rateLimiter.C
		}

		// Send the email in a goroutine for concurrency
		wg.Add(1)
		go func(em EmailQueueItem, srv string) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// OPTIMIZATION: Smart Sending check moved to getNextBatch() SQL query
			// Emails fetched from queue have already been filtered for Smart Sending eligibility
			// This dramatically improves performance by:
			// 1. Eliminating N individual database queries (one per email)
			// 2. Preventing rate limit consumption for emails that would be skipped
			// 3. Allowing queue processor to move through Smart Sending-blocked subscribers instantly

			// CRITICAL: Check account-wide rate limit before sending
			// This is the primary rate limit that takes precedence over all others
			canSend, err := p.checkAccountRateLimit()
			if err != nil {
				p.log.Printf("error checking account rate limit: %v", err)
				// On error, be conservative and don't send
				if err := p.markFailed(em.ID, fmt.Sprintf("account rate limit check failed: %v", err)); err != nil {
					p.log.Printf("error marking email %d as failed: %v", em.ID, err)
				}
				return
			}

			if !canSend {
				// We've hit the account-wide rate limit, skip this email for now
				// It will be retried in the next batch
				p.log.Printf("skipping email %d due to account-wide rate limit", em.ID)
				// Mark it back to queued so it gets picked up later
				if _, err := p.db.Exec(`UPDATE email_queue SET status = $1, updated_at = NOW() WHERE id = $2`, StatusQueued, em.ID); err != nil {
					p.log.Printf("error resetting email %d to queued: %v", em.ID, err)
				}
				return
			}

			// Send the email
			if err := p.sendEmail(em, srv); err != nil {
				// Get server name for better error context
				serverName := srv
				if cap, exists := capacities[srv]; exists {
					serverName = cap.Name
				}
				p.log.Printf("✗ error sending email %d (campaign %d, subscriber %d) via SMTP server '%s': %v",
					em.ID, em.CampaignID, em.SubscriberID, serverName, err)
				if err := p.markFailed(em.ID, err.Error()); err != nil {
					p.log.Printf("error marking email %d as failed: %v", em.ID, err)
				}
				return
			}

			// Mark email as sent
			if err := p.markSent(em.ID); err != nil {
				p.log.Printf("error marking email %d as sent: %v", em.ID, err)
				return
			}

			// Log successful delivery
			p.log.Printf("✓ email %d (campaign %d, subscriber %d) delivered successfully via server %s",
				em.ID, em.CampaignID, em.SubscriberID, srv)

			// Update Smart Sending tracking after successful send
			if settings.AppSmartSendingEnabled {
				if _, err := p.db.Exec(
					`INSERT INTO subscriber_last_send (subscriber_id, last_campaign_send_at, updated_at)
					 VALUES ($1, NOW(), NOW())
					 ON CONFLICT (subscriber_id)
					 DO UPDATE SET last_campaign_send_at = NOW(), updated_at = NOW()`,
					em.SubscriberID); err != nil {
					p.log.Printf("error updating subscriber_last_send for subscriber %d: %v", em.SubscriberID, err)
					// Don't return - this is not critical enough to fail the send
				}
			}

			// TODO: Azure Event Grid message tracking
			// For Azure Communication Services, we need to store the Azure message ID for webhook correlation
			// This requires either:
			// 1. Capturing the Message-ID from SMTP response
			// 2. Using custom X-headers that Azure preserves in webhooks
			// 3. Pre-generating UUID and including in Message-ID header
			// Implementation pending proper Message-ID capture mechanism

			// Update server usage counters
			if err := p.incrementServerUsage(srv); err != nil {
				p.log.Printf("error incrementing usage for server %s: %v", srv, err)
			}

			// CRITICAL: Increment account-wide rate limit counters AFTER successful send
			if err := p.incrementAccountRateLimit(); err != nil {
				p.log.Printf("error incrementing account rate limit: %v", err)
			}
		}(email, serverUUID)

		// Update capacity tracking
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

	// Wait for all sends to complete
	wg.Wait()

	// Check if any campaigns are complete and mark them as finished
	if err := p.checkCompletedCampaigns(); err != nil {
		p.log.Printf("error checking completed campaigns: %v", err)
	}

	return nil
}

// checkCompletedCampaigns checks for queue-based campaigns where all emails have been processed
// (sent, failed, or cancelled) and marks them as finished
func (p *Processor) checkCompletedCampaigns() error {
	// Find campaigns that:
	// 1. Are currently running
	// 2. Use the queue (use_queue=true)
	// 3. Have no queued or sending emails left
	query := `
		SELECT DISTINCT c.id, c.name
		FROM campaigns c
		WHERE c.status = 'running'
		  AND c.use_queue = true
		  AND NOT EXISTS (
		    SELECT 1
		    FROM email_queue eq
		    WHERE eq.campaign_id = c.id
		      AND eq.status IN ('queued', 'sending')
		  )
		  AND EXISTS (
		    SELECT 1
		    FROM email_queue eq
		    WHERE eq.campaign_id = c.id
		  )
	`

	type campaignInfo struct {
		ID   int    `db:"id"`
		Name string `db:"name"`
	}

	var campaigns []campaignInfo
	if err := p.db.Select(&campaigns, query); err != nil {
		return fmt.Errorf("error querying completed campaigns: %w", err)
	}

	// Mark each completed campaign as finished
	for _, camp := range campaigns {
		updateQuery := `
			UPDATE campaigns
			SET status = 'finished',
			    queue_completed_at = NOW(),
			    updated_at = NOW()
			WHERE id = $1
		`

		if _, err := p.db.Exec(updateQuery, camp.ID); err != nil {
			p.log.Printf("error marking campaign %d (%s) as finished: %v", camp.ID, camp.Name, err)
			continue
		}

		p.log.Printf("campaign %d (%s) marked as finished - all queued emails processed", camp.ID, camp.Name)
	}

	return nil
}

// isWithinTimeWindow checks if the current time is within the configured sending window
func (p *Processor) isWithinTimeWindow() bool {
	// If no time window is configured, always allow sending
	if p.cfg.TimeWindowStart == "" || p.cfg.TimeWindowEnd == "" {
		return true
	}

	// Get configured timezone from settings
	settings, err := p.getSettings()
	if err != nil {
		p.log.Printf("error getting settings for timezone: %v, using system time", err)
		// Fall back to system time on error
		now := time.Now()
		currentTime := now.Format("15:04")
		return currentTime >= p.cfg.TimeWindowStart && currentTime <= p.cfg.TimeWindowEnd
	}

	// Load the configured timezone
	loc, err := time.LoadLocation(settings.AppTimezone)
	if err != nil {
		p.log.Printf("error loading timezone '%s': %v, using system time", settings.AppTimezone, err)
		// Fall back to system time if timezone is invalid
		now := time.Now()
		currentTime := now.Format("15:04")
		return currentTime >= p.cfg.TimeWindowStart && currentTime <= p.cfg.TimeWindowEnd
	}

	// Get current time in the configured timezone
	now := time.Now().In(loc)
	currentTime := now.Format("15:04")

	// Simple string comparison works for HH:MM format
	return currentTime >= p.cfg.TimeWindowStart && currentTime <= p.cfg.TimeWindowEnd
}

// getNextBatch retrieves the next batch of emails to send from the queue
func (p *Processor) getNextBatch() ([]EmailQueueItem, error) {
	var emails []EmailQueueItem

	// Get settings to check if Smart Sending is enabled
	settings, err := p.getSettings()
	if err != nil {
		return nil, fmt.Errorf("error getting settings: %w", err)
	}

	var query string
	var args []interface{}

	if settings.AppSmartSendingEnabled {
		// OPTIMIZED: Filter Smart Sending subscribers at SQL level
		// This prevents fetching emails that will be skipped, avoiding rate limit waste
		query = `
			SELECT eq.id, eq.campaign_id, eq.subscriber_id, eq.status, eq.priority,
			       eq.scheduled_at, eq.sent_at, eq.assigned_smtp_server_uuid,
			       eq.retry_count, eq.last_error, eq.created_at, eq.updated_at
			FROM email_queue eq
			LEFT JOIN subscriber_last_send sls ON eq.subscriber_id = sls.subscriber_id
			WHERE eq.status = $1
			  AND eq.scheduled_at <= NOW()
			  AND (
			    sls.last_campaign_send_at IS NULL
			    OR sls.last_campaign_send_at <= NOW() - INTERVAL '1 hour' * $2
			  )
			ORDER BY eq.priority DESC, eq.scheduled_at ASC
			LIMIT $3
		`
		args = []interface{}{StatusQueued, settings.AppSmartSendingPeriodHours, p.cfg.BatchSize}
	} else {
		// Smart Sending disabled, fetch all queued emails
		query = `
			SELECT id, campaign_id, subscriber_id, status, priority,
			       scheduled_at, sent_at, assigned_smtp_server_uuid,
			       retry_count, last_error, created_at, updated_at
			FROM email_queue
			WHERE status = $1
			  AND scheduled_at <= NOW()
			ORDER BY priority DESC, scheduled_at ASC
			LIMIT $2
		`
		args = []interface{}{StatusQueued, p.cfg.BatchSize}
	}

	if err := p.db.Select(&emails, query, args...); err != nil {
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
			p.log.Printf("error getting daily usage for SMTP server '%s': %v", smtp.Name, err)
			continue
		}
		capacity.DailyUsed = usage

		// If daily limit is 0 (unlimited), set DailyRemaining to max int
		// Otherwise calculate remaining capacity
		if capacity.DailyLimit == 0 {
			capacity.DailyRemaining = 999999999 // Effectively unlimited
		} else {
			capacity.DailyRemaining = capacity.DailyLimit - capacity.DailyUsed
		}

		// Get sliding window usage
		windowUsage, err := p.getSlidingWindowUsage(smtp.UUID)
		if err != nil {
			p.log.Printf("error getting sliding window usage for SMTP server '%s': %v", smtp.Name, err)
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
	result, err := p.db.Exec(`
		UPDATE email_queue
		SET status = $1, assigned_smtp_server_uuid = $2, updated_at = NOW()
		WHERE id = $3 AND status = $4
	`, StatusSending, serverUUID, emailID, StatusQueued)
	if err != nil {
		return err
	}

	// Check if we actually updated a row
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		// Email was already claimed by another process
		return fmt.Errorf("email %d already being processed", emailID)
	}

	return nil
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

// checkAccountRateLimit checks if sending an email would exceed account-wide rate limits
// Returns true if we can send, false if we need to wait
func (p *Processor) checkAccountRateLimit() (bool, error) {
	settings, err := p.getSettings()
	if err != nil {
		return false, fmt.Errorf("error getting settings: %w", err)
	}

	// If limits are 0 or not set, don't enforce
	if settings.AppAccountRateLimitPerMinute <= 0 && settings.AppAccountRateLimitPerHour <= 0 {
		return true, nil
	}

	// Get current account-wide usage
	var state struct {
		MinuteWindowStart time.Time `db:"minute_window_start"`
		EmailsInMinute    int       `db:"emails_in_minute"`
		HourWindowStart   time.Time `db:"hour_window_start"`
		EmailsInHour      int       `db:"emails_in_hour"`
	}

	err = p.db.Get(&state, `SELECT minute_window_start, emails_in_minute, hour_window_start, emails_in_hour FROM account_rate_limit_state LIMIT 1`)
	if err != nil && err != sql.ErrNoRows {
		return false, fmt.Errorf("error getting account rate limit state: %w", err)
	}

	now := time.Now()

	// Check minute window
	if settings.AppAccountRateLimitPerMinute > 0 {
		// Reset window if expired
		if now.Sub(state.MinuteWindowStart) >= time.Minute {
			// Window has expired, we can send
			return true, nil
		}

		// Check if we've exceeded the limit
		if state.EmailsInMinute >= settings.AppAccountRateLimitPerMinute {
			p.log.Printf("account-wide rate limit: %d emails sent in last minute (limit: %d), waiting...",
				state.EmailsInMinute, settings.AppAccountRateLimitPerMinute)
			return false, nil
		}
	}

	// Check hour window
	if settings.AppAccountRateLimitPerHour > 0 {
		// Reset window if expired
		if now.Sub(state.HourWindowStart) >= time.Hour {
			// Window has expired, we can send
			return true, nil
		}

		// Check if we've exceeded the limit
		if state.EmailsInHour >= settings.AppAccountRateLimitPerHour {
			p.log.Printf("account-wide rate limit: %d emails sent in last hour (limit: %d), waiting...",
				state.EmailsInHour, settings.AppAccountRateLimitPerHour)
			return false, nil
		}
	}

	// We haven't exceeded either limit
	return true, nil
}

// incrementAccountRateLimit increments the account-wide rate limit counters
func (p *Processor) incrementAccountRateLimit() error {
	settings, err := p.getSettings()
	if err != nil {
		return fmt.Errorf("error getting settings: %w", err)
	}

	// If limits are not set, don't track
	if settings.AppAccountRateLimitPerMinute <= 0 && settings.AppAccountRateLimitPerHour <= 0 {
		return nil
	}

	now := time.Now()

	// Update account-wide counters with window reset logic
	_, err = p.db.Exec(`
		UPDATE account_rate_limit_state
		SET
			-- Minute window: reset if more than 1 minute has passed
			minute_window_start = CASE
				WHEN NOW() - minute_window_start > interval '1 minute'
				THEN $1
				ELSE minute_window_start
			END,
			emails_in_minute = CASE
				WHEN NOW() - minute_window_start > interval '1 minute'
				THEN 1
				ELSE emails_in_minute + 1
			END,
			-- Hour window: reset if more than 1 hour has passed
			hour_window_start = CASE
				WHEN NOW() - hour_window_start > interval '1 hour'
				THEN $1
				ELSE hour_window_start
			END,
			emails_in_hour = CASE
				WHEN NOW() - hour_window_start > interval '1 hour'
				THEN 1
				ELSE emails_in_hour + 1
			END,
			updated_at = NOW()
	`, now)

	return err
}
