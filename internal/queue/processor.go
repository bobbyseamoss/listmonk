package queue

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/models"
	"github.com/lib/pq"
)

// RateLimitHandler is a callback for handling rate limit errors
type RateLimitHandler func(serverUUID, serverName string, err error)

// Processor handles the queue-based email delivery system
type Processor struct {
	db  *sqlx.DB
	cfg Config
	log *log.Logger

	// Campaign and messenger access
	getCampaign func(int) (*models.Campaign, error)
	pushEmail   func(campaignID int, subID int, serverUUID string) error

	// Rate limit handler callback
	rateLimitHandler RateLimitHandler

	// Control channels
	stopChan chan struct{}
	doneChan chan struct{}

	// Round-robin counter for distributing emails across servers with equal capacity
	rrCounter uint64
	rrMutex   sync.Mutex
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

// SetRateLimitHandler sets the callback function for handling rate limit errors
func (p *Processor) SetRateLimitHandler(fn RateLimitHandler) {
	p.rateLimitHandler = fn
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

// StartCampaignStatsSync starts the periodic campaign statistics sync
// This runs every 5 minutes to sync campaign.sent counts from email_queue
func (p *Processor) StartCampaignStatsSync() {
	p.log.Println("starting campaign stats sync (every 5 minutes)")

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	// Run immediately on start
	if err := p.syncRunningCampaignCounts(); err != nil {
		p.log.Printf("error in initial campaign stats sync: %v", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := p.syncRunningCampaignCounts(); err != nil {
				p.log.Printf("error in campaign stats sync: %v", err)
			}
		case <-p.stopChan:
			p.log.Println("stopping campaign stats sync")
			return
		}
	}
}

// syncRunningCampaignCounts syncs campaign.sent counts from email_queue for running campaigns
func (p *Processor) syncRunningCampaignCounts() error {
	// Get all running queue-based campaign IDs
	var runningCampaignIDs []int
	err := p.db.Select(&runningCampaignIDs, `
		SELECT id FROM campaigns
		WHERE status = 'running'
		  AND use_queue = true
	`)
	if err != nil {
		return fmt.Errorf("error fetching running campaigns: %w", err)
	}

	if len(runningCampaignIDs) == 0 {
		return nil // No running campaigns to sync
	}

	// Sync campaign.sent counts from email_queue
	result, err := p.db.Exec(`
		UPDATE campaigns c
		SET
			sent = (
				SELECT COUNT(*)
				FROM email_queue
				WHERE campaign_id = c.id AND status = 'sent'
			),
			updated_at = NOW()
		WHERE c.id = ANY($1)
		  AND c.use_queue = true
	`, pq.Array(runningCampaignIDs))
	if err != nil {
		return fmt.Errorf("error syncing campaign counts: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		p.log.Printf("📊 synced sent counts for %d running campaign(s)", rowsAffected)
	}

	return nil
}

// autoPauseResumeCampaigns automatically pauses campaigns when outside time window
// and resumes them when inside time window
func (p *Processor) autoPauseResumeCampaigns() error {
	// Fetch current settings dynamically from database
	// This ensures we always use the latest time window values even if changed in UI
	settings, err := p.getSettings()
	if err != nil {
		return fmt.Errorf("error fetching settings for auto-pause: %w", err)
	}

	// Check if time window is configured
	if settings.AppSendTimeStart == "" || settings.AppSendTimeEnd == "" {
		// No time window configured, nothing to do
		return nil
	}

	// Check if we're currently within the time window using dynamic settings
	withinWindow := p.isWithinTimeWindowDynamic(settings)

	if withinWindow {
		// We're inside the window - resume any auto-paused campaigns
		return p.resumeAutoPausedCampaigns()
	} else {
		// We're outside the window - auto-pause running campaigns
		return p.autoPauseRunningCampaigns()
	}
}

// autoPauseRunningCampaigns pauses all running campaigns when outside time window
// except for campaigns with bypass_time_window=true
func (p *Processor) autoPauseRunningCampaigns() error {
	// First, get list of running queue-based campaign IDs to sync
	// Exclude campaigns with bypass_time_window=true - they should keep running
	// Check for both use_queue=true AND messenger='automatic' to catch campaigns
	// that were started before use_queue was properly set
	var runningCampaignIDs []int
	err := p.db.Select(&runningCampaignIDs, `
		SELECT id FROM campaigns
		WHERE status = 'running'
		  AND (use_queue = true OR messenger = 'automatic')
		  AND auto_paused = false
		  AND bypass_time_window = false
	`)
	if err != nil {
		return fmt.Errorf("error fetching running campaigns: %w", err)
	}


	// Sync campaign.sent counts from email_queue before pausing
	// This ensures the paused campaign shows accurate sent counts
	if len(runningCampaignIDs) > 0 {
		_, err = p.db.Exec(`
			UPDATE campaigns c
			SET
				sent = (
					SELECT COUNT(*)
					FROM email_queue
					WHERE campaign_id = c.id AND status = 'sent'
				),
				updated_at = NOW()
			WHERE c.id = ANY($1)
			  AND c.use_queue = true
		`, runningCampaignIDs)
		if err != nil {
			p.log.Printf("warning: error syncing campaign counts before pause: %v", err)
			// Continue with pause even if sync fails
		} else {
			p.log.Printf("✓ synced sent counts for %d campaign(s) before pausing", len(runningCampaignIDs))
		}
	}

	// Update all running queue-based campaigns to paused status
	// Only pause campaigns that aren't already auto-paused
	// IMPORTANT: Skip campaigns with bypass_time_window=true - they should keep running
	// Check for both use_queue=true AND messenger='automatic' to catch campaigns
	// that were started before use_queue was properly set
	res, err := p.db.Exec(`
		UPDATE campaigns
		SET status = 'paused',
		    auto_paused = true,
		    auto_paused_at = NOW(),
		    updated_at = NOW()
		WHERE status = 'running'
		  AND (use_queue = true OR messenger = 'automatic')
		  AND auto_paused = false
		  AND bypass_time_window = false
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
	// Check for both use_queue=true AND messenger='automatic' to catch campaigns
	// that were started before use_queue was properly set
	var pausedCampaignIDs []int
	err := p.db.Select(&pausedCampaignIDs, `
		SELECT id FROM campaigns
		WHERE status = 'paused' AND (use_queue = true OR messenger = 'automatic')
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
	// Check for both use_queue=true AND messenger='automatic' to catch campaigns
	// that were started before use_queue was properly set
	res, err := p.db.Exec(`
		UPDATE campaigns
		SET status = 'running',
		    auto_paused = false,
		    auto_paused_at = NULL,
		    updated_at = NOW()
		WHERE status = 'paused'
		  AND (use_queue = true OR messenger = 'automatic')
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
	withinWindow := p.isWithinTimeWindow()

	// If outside time window, only process campaigns that have bypass_time_window=true
	bypassOnly := !withinWindow

	// If outside window and no bypass campaigns exist, skip processing entirely
	if bypassOnly {
		hasbypassCampaigns, err := p.hasBypassTimeWindowCampaigns()
		if err != nil {
			p.log.Printf("error checking bypass campaigns: %v", err)
			return nil
		}
		if !hasbypassCampaigns {
			return nil
		}
	}

	// Get the next batch of emails to send (filtered to bypass campaigns if outside window)
	emails, err := p.getNextBatch(bypassOnly)
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
		serverUUID, isDisabled := p.selectServer(capacities, email)
		if isDisabled {
			// Domain is disabled - mark as failed and skip
			_ = p.markFailed(email.ID, "domain routing disabled - sending blocked")
			continue
		}
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

		// CRITICAL: Enforce account-wide rate limit BEFORE spawning goroutine
		// This prevents Azure rate limit violations that result in 1-hour cooldowns
		// The ticker controls send rate, semaphore controls concurrency
		if rateLimiter != nil {
			<-rateLimiter.C
		}

		// Send the email in a goroutine for concurrency
		wg.Add(1)
		go func(em EmailQueueItem, srv string) {
			defer wg.Done()

			// Acquire semaphore to limit concurrent SMTP connections
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// OPTIMIZATION: Smart Sending check moved to getNextBatch() SQL query
			// Emails fetched from queue have already been filtered for Smart Sending eligibility
			// This dramatically improves performance by:
			// 1. Eliminating N individual database queries (one per email)
			// 2. Preventing rate limit consumption for emails that would be skipped
			// 3. Allowing queue processor to move through Smart Sending-blocked subscribers instantly

			// Rate limiting is enforced BEFORE goroutine spawn (line 470-472)
			// This prevents Azure rate limit violations without database contention
			// Ticker controls send rate, semaphore controls concurrency

			// Send the email
			if err := p.sendEmail(em, srv); err != nil {
				// Get server name for better error context
				serverName := srv
				if cap, exists := capacities[srv]; exists {
					serverName = cap.Name
				}
				p.log.Printf("✗ error sending email %d (campaign %d, subscriber %d) via SMTP server '%s': %v",
					em.ID, em.CampaignID, em.SubscriberID, serverName, err)

				// Check if this is a rate limit error and notify the handler
				if p.rateLimitHandler != nil {
					p.rateLimitHandler(srv, serverName, err)
				}

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
// Uses dynamically-fetched settings to ensure UI changes take effect immediately
func (p *Processor) isWithinTimeWindow() bool {
	// Get settings dynamically from database to pick up any UI changes
	settings, err := p.getSettings()
	if err != nil {
		p.log.Printf("error getting settings for time window check: %v", err)
		return true // Default to allow sending on error
	}

	// Use the dynamic isWithinTimeWindowDynamic function
	return p.isWithinTimeWindowDynamic(settings)
}

// isWithinTimeWindowDynamic checks if the current time is within the sending window
// using dynamically-fetched settings instead of cached config values
func (p *Processor) isWithinTimeWindowDynamic(settings models.Settings) bool {
	// If no time window is configured, always allow sending
	if settings.AppSendTimeStart == "" || settings.AppSendTimeEnd == "" {
		return true
	}

	// Load the configured timezone
	loc, err := time.LoadLocation(settings.AppTimezone)
	if err != nil {
		p.log.Printf("error loading timezone '%s': %v, using system time", settings.AppTimezone, err)
		// Fall back to system time if timezone is invalid
		now := time.Now()
		currentTime := now.Format("15:04")
		return currentTime >= settings.AppSendTimeStart && currentTime <= settings.AppSendTimeEnd
	}

	// Get current time in the configured timezone
	now := time.Now().In(loc)
	currentTime := now.Format("15:04")

	// Simple string comparison works for HH:MM format
	return currentTime >= settings.AppSendTimeStart && currentTime <= settings.AppSendTimeEnd
}

// hasBypassTimeWindowCampaigns checks if there are any running campaigns with bypass_time_window=true
// that have queued emails waiting to be sent
func (p *Processor) hasBypassTimeWindowCampaigns() (bool, error) {
	var count int
	err := p.db.Get(&count, `
		SELECT COUNT(DISTINCT eq.campaign_id)
		FROM email_queue eq
		INNER JOIN campaigns c ON eq.campaign_id = c.id
		WHERE eq.status = $1
		  AND eq.scheduled_at <= NOW()
		  AND c.bypass_time_window = true
		  AND c.status = 'running'
	`, StatusQueued)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// getNextBatch retrieves the next batch of emails to send from the queue
// If bypassOnly is true, only returns emails from campaigns with bypass_time_window=true
func (p *Processor) getNextBatch(bypassOnly bool) ([]EmailQueueItem, error) {
	var emails []EmailQueueItem

	// STUCK EMAIL RECOVERY: Reset emails that have been in "sending" status for too long
	// This prevents emails from being permanently stuck if the send operation fails,
	// times out, or the container crashes mid-send.
	// Threshold: 5 minutes (adjust if needed based on typical send times)
	recoveryResult, err := p.db.Exec(`
		UPDATE email_queue
		SET status = $1,
		    updated_at = NOW(),
		    last_error = CASE
		        WHEN last_error IS NULL THEN 'Recovered from stuck sending state'
		        ELSE last_error || ' | Recovered from stuck sending state'
		    END
		WHERE status = $2
		  AND updated_at < NOW() - INTERVAL '5 minutes'
	`, StatusQueued, StatusSending)

	if err != nil {
		// Log error but don't fail the batch - we can still process queued emails
		p.log.Printf("warning: error recovering stuck emails: %v", err)
	} else {
		recoveredCount, _ := recoveryResult.RowsAffected()
		if recoveredCount > 0 {
			p.log.Printf("🔄 recovered %d email(s) stuck in 'sending' status for >5 minutes", recoveredCount)
		}
	}

	// CRITICAL FIX: Do NOT apply Smart Sending filter here!
	// Smart Sending should be enforced when emails are QUEUED, not when fetched.
	// Once an email is in email_queue, it was intentionally queued and should be sent.
	// Applying Smart Sending filter here causes massive slowdowns:
	//   - Campaign 65 had 78,000 emails queued
	//   - Smart Sending filtered out 99%+ of them at fetch time
	//   - Only 1-2 emails per batch instead of 100
	//   - Send rate dropped from 2000/hour to 60/hour
	//
	// Fetch queued emails that are ready to send
	// If bypassOnly=true, only get emails from campaigns with bypass_time_window=true
	// JOIN with subscribers to get email address for domain-based SMTP routing
	var query string
	if bypassOnly {
		query = `
			SELECT eq.id, eq.campaign_id, eq.subscriber_id, s.email as subscriber_email,
			       eq.status, eq.priority, eq.scheduled_at, eq.sent_at,
			       eq.assigned_smtp_server_uuid, eq.retry_count, eq.last_error,
			       eq.created_at, eq.updated_at
			FROM email_queue eq
			INNER JOIN campaigns c ON eq.campaign_id = c.id
			INNER JOIN subscribers s ON eq.subscriber_id = s.id
			WHERE eq.status = $1
			  AND eq.scheduled_at <= NOW()
			  AND c.bypass_time_window = true
			ORDER BY eq.priority DESC, eq.scheduled_at ASC
			LIMIT $2
		`
	} else {
		query = `
			SELECT eq.id, eq.campaign_id, eq.subscriber_id, s.email as subscriber_email,
			       eq.status, eq.priority, eq.scheduled_at, eq.sent_at,
			       eq.assigned_smtp_server_uuid, eq.retry_count, eq.last_error,
			       eq.created_at, eq.updated_at
			FROM email_queue eq
			INNER JOIN subscribers s ON eq.subscriber_id = s.id
			WHERE eq.status = $1
			  AND eq.scheduled_at <= NOW()
			ORDER BY eq.priority DESC, eq.scheduled_at ASC
			LIMIT $2
		`
	}

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

	// Log how many SMTP servers are configured
	enabledCount := 0
	for _, smtp := range settings.SMTP {
		if smtp.Enabled {
			enabledCount++
		}
	}
	p.log.Printf("📊 SMTP server config: %d total servers, %d enabled", len(settings.SMTP), enabledCount)

	for _, smtp := range settings.SMTP {
		if !smtp.Enabled {
			p.log.Printf("⏭️  SMTP server '%s' (UUID: %s) is DISABLED, skipping", smtp.Name, smtp.UUID)
			continue
		}

		capacity := &ServerCapacity{
			UUID:       smtp.UUID,
			Name:       smtp.Name,
			Host:       smtp.Host,
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
		p.log.Printf("✅ SMTP server '%s' (UUID: %s) added to capacities: DailyLimit=%d, DailyRemaining=%d, CanSendNow=%v",
			smtp.Name, smtp.UUID, capacity.DailyLimit, capacity.DailyRemaining, capacity.CanSendNow)
	}

	p.log.Printf("📊 Final capacities map has %d servers available for sending", len(capacities))
	return capacities, nil
}

// selectServer selects the best SMTP server to use for an email
// Uses dynamic domain-based routing from settings, then falls back to round-robin
// Returns empty string with isDisabled=true if the domain is disabled (should not send)
func (p *Processor) selectServer(capacities map[string]*ServerCapacity, email EmailQueueItem) (string, bool) {
	// DOMAIN-BASED ROUTING FIRST - takes priority over pre-assigned servers
	// This allows UI-configured domain routing to override any previous assignments
	if email.SubscriberEmail != "" {
		domain := extractDomain(email.SubscriberEmail)

		// Get domain routing rules from settings
		settings, err := p.getSettings()
		if err == nil && settings.DomainRouting != nil && len(settings.DomainRouting) > 0 {
			// Check for exact domain match
			if routingType, exists := settings.DomainRouting[domain]; exists && routingType != "" {
				// Handle "disabled" - domain is blocked from sending
				if routingType == "disabled" {
					p.log.Printf("🚫 domain routing: %s is DISABLED, skipping email to %s", domain, email.SubscriberEmail)
					return "", true // Return empty with isDisabled=true
				}

				// Handle "azure" or "sendgrid" - select from matching servers
				selectedUUID := p.selectFromServerType(capacities, routingType, domain, email.SubscriberEmail)
				if selectedUUID != "" {
					return selectedUUID, false
				}
				// All matching servers unavailable, fall through to round-robin
				p.log.Printf("⚠️  domain routing: all %s servers unavailable for %s, using round-robin fallback", routingType, domain)
			}
		}
	}

	// If no domain routing matched, check if email has a pre-assigned server
	if email.AssignedSMTPServerUUID.Valid && email.AssignedSMTPServerUUID.String != "" {
		if cap, exists := capacities[email.AssignedSMTPServerUUID.String]; exists && cap.CanSendNow {
			return email.AssignedSMTPServerUUID.String, false
		}
	}

	// Collect all servers that can send and group by capacity
	type serverEntry struct {
		uuid     string
		capacity *ServerCapacity
	}

	var availableServers []serverEntry
	var bestCapacity int = -1

	// First pass: find the best capacity and collect servers with that capacity
	for uuid, cap := range capacities {
		if !cap.CanSendNow {
			continue
		}

		if cap.DailyRemaining > bestCapacity {
			// Found a better capacity, start fresh
			bestCapacity = cap.DailyRemaining
			availableServers = []serverEntry{{uuid: uuid, capacity: cap}}
		} else if cap.DailyRemaining == bestCapacity {
			// Same capacity, add to list for round-robin
			availableServers = append(availableServers, serverEntry{uuid: uuid, capacity: cap})
		}
	}

	if len(availableServers) == 0 {
		return "", false
	}

	// If only one server, return it
	if len(availableServers) == 1 {
		return availableServers[0].uuid, false
	}

	// Multiple servers with equal capacity - use round-robin
	// Sort by UUID to ensure consistent ordering across instances
	for i := 0; i < len(availableServers)-1; i++ {
		for j := i + 1; j < len(availableServers); j++ {
			if availableServers[i].uuid > availableServers[j].uuid {
				availableServers[i], availableServers[j] = availableServers[j], availableServers[i]
			}
		}
	}

	// Use atomic counter for round-robin selection
	p.rrMutex.Lock()
	index := p.rrCounter % uint64(len(availableServers))
	p.rrCounter++
	p.rrMutex.Unlock()

	selected := availableServers[index]
	p.log.Printf("🔄 round-robin selection: %d servers with equal capacity (%d), selected '%s' (index %d)",
		len(availableServers), bestCapacity, selected.capacity.Name, index)

	return selected.uuid, false
}

// extractDomain extracts the domain from an email address
func extractDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return strings.ToLower(parts[1])
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

// selectFromServerType selects a server based on routing type (azure or sendgrid) using round-robin
// Returns empty string if no matching servers are available
func (p *Processor) selectFromServerType(capacities map[string]*ServerCapacity, routingType, domain, email string) string {
	// Filter capacities to only include servers matching the routing type
	type serverEntry struct {
		uuid     string
		capacity *ServerCapacity
	}
	var availableServers []serverEntry

	for uuid, cap := range capacities {
		if !cap.CanSendNow {
			continue // Server can't send right now
		}

		// Check if server matches the routing type based on host
		isMatch := false
		switch routingType {
		case "azure":
			isMatch = strings.Contains(strings.ToLower(cap.Host), "azurecomm")
		case "sendgrid":
			isMatch = strings.Contains(strings.ToLower(cap.Host), "sendgrid")
		}

		if isMatch {
			availableServers = append(availableServers, serverEntry{uuid: uuid, capacity: cap})
		}
	}

	if len(availableServers) == 0 {
		return "" // No matching servers available
	}

	// If only one server, return it
	if len(availableServers) == 1 {
		p.log.Printf("🎯 domain routing: %s → %s (%s for %s)",
			email, availableServers[0].capacity.Name, routingType, domain)
		return availableServers[0].uuid
	}

	// Multiple servers - sort by UUID for consistent ordering, then round-robin
	for i := 0; i < len(availableServers)-1; i++ {
		for j := i + 1; j < len(availableServers); j++ {
			if availableServers[i].uuid > availableServers[j].uuid {
				availableServers[i], availableServers[j] = availableServers[j], availableServers[i]
			}
		}
	}

	// Use atomic counter for round-robin selection
	p.rrMutex.Lock()
	index := p.rrCounter % uint64(len(availableServers))
	p.rrCounter++
	p.rrMutex.Unlock()

	selected := availableServers[index]
	p.log.Printf("🎯 domain routing: %s → %s (%s for %s, %d servers available, index %d)",
		email, selected.capacity.Name, routingType, domain, len(availableServers), index)

	return selected.uuid
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
