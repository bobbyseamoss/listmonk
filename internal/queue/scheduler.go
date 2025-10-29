package queue

import (
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/models"
)

// Scheduler handles the initial scheduling of queued emails
type Scheduler struct {
	db  *sqlx.DB
	cfg Config
	log *log.Logger
}

// NewScheduler creates a new queue scheduler
func NewScheduler(db *sqlx.DB, cfg Config, log *log.Logger) *Scheduler {
	return &Scheduler{
		db:  db,
		cfg: cfg,
		log: log,
	}
}

// ScheduleCampaign calculates and assigns scheduled times and SMTP servers for all queued emails in a campaign
func (s *Scheduler) ScheduleCampaign(campaignID int, settings models.Settings) error {
	s.log.Printf("scheduling campaign %d emails across SMTP servers", campaignID)

	// Get all queued emails for this campaign (they all have scheduled_at = NOW() initially)
	var emails []struct {
		ID           int64 `db:"id"`
		SubscriberID int   `db:"subscriber_id"`
	}

	err := s.db.Select(&emails, `
		SELECT id, subscriber_id
		FROM email_queue
		WHERE campaign_id = $1 AND status = 'queued'
		ORDER BY id ASC
	`, campaignID)
	if err != nil {
		return fmt.Errorf("error fetching queued emails: %w", err)
	}

	if len(emails) == 0 {
		s.log.Printf("no emails to schedule for campaign %d", campaignID)
		return nil
	}

	s.log.Printf("found %d emails to schedule for campaign %d", len(emails), campaignID)

	// Get enabled SMTP servers with their capacities and sliding window configs
	type serverInfo struct {
		UUID                  string
		Name                  string
		DailyLimit            int
		CurrentUsage          int
		RemainingCapacity     int
		SlidingWindow         bool
		SlidingWindowDuration time.Duration
		SlidingWindowRate     int
	}
	var servers []serverInfo

	for _, smtp := range settings.SMTP {
		if !smtp.Enabled {
			continue
		}

		// Get current usage
		var usage int
		err := s.db.Get(&usage, `
			SELECT COALESCE(emails_sent, 0)
			FROM smtp_daily_usage
			WHERE smtp_server_uuid = $1 AND usage_date = CURRENT_DATE
		`, smtp.UUID)
		if err != nil {
			usage = 0 // If no row exists, usage is 0
		}

		remaining := smtp.DailyLimit - usage
		if remaining < 0 {
			remaining = 0
		}

		// Parse server's sliding window duration
		var slidingDuration time.Duration
		if smtp.SlidingWindowDuration != "" {
			d, err := time.ParseDuration(smtp.SlidingWindowDuration)
			if err == nil {
				slidingDuration = d
			}
		}

		// Only include servers with remaining capacity (or unlimited)
		if remaining > 0 || smtp.DailyLimit == 0 {
			servers = append(servers, serverInfo{
				UUID:                  smtp.UUID,
				Name:                  smtp.Name,
				DailyLimit:            smtp.DailyLimit,
				CurrentUsage:          usage,
				RemainingCapacity:     remaining,
				SlidingWindow:         smtp.SlidingWindow,
				SlidingWindowDuration: slidingDuration,
				SlidingWindowRate:     smtp.SlidingWindowRate,
			})
		}
	}

	if len(servers) == 0 {
		return fmt.Errorf("no SMTP servers available with remaining capacity")
	}

	s.log.Printf("distributing across %d SMTP servers", len(servers))

	// Calculate total available capacity
	totalCapacity := 0
	for _, srv := range servers {
		if srv.DailyLimit == 0 {
			// Unlimited server - set high capacity
			totalCapacity += 1000000
		} else {
			totalCapacity += srv.RemainingCapacity
		}
	}

	// Get sending time window configuration
	startTime := s.getNextSendingWindow()
	sendingHoursPerDay := s.calculateSendingHours()

	// Calculate send rate (emails per minute)
	// If sliding window is configured, respect it
	sendRatePerMinute := s.calculateSendRate(totalCapacity, sendingHoursPerDay)

	// Check if we should schedule for immediate sending
	// This happens when: no time window is configured AND we're not severely capacity constrained
	immediateMode := (s.cfg.TimeWindowStart == "" || s.cfg.TimeWindowEnd == "") &&
	                  totalCapacity >= len(emails)

	if immediateMode {
		s.log.Printf("immediate mode: scheduling all %d emails for NOW (processor will handle rate limiting)", len(emails))
	} else {
		s.log.Printf("scheduled mode: send rate %d emails/min, starting at %s", sendRatePerMinute, startTime.Format("2006-01-02 15:04:05"))
	}

	// Track per-server sliding window usage
	type serverWindowTracker struct {
		count       int
		windowStart time.Time
	}
	serverTracking := make(map[string]*serverWindowTracker)

	// Initialize tracking for each server with sliding window
	for _, srv := range servers {
		serverTracking[srv.UUID] = &serverWindowTracker{
			count:       0,
			windowStart: startTime,
		}
	}

	// Distribute emails across servers and time
	currentTime := startTime
	serverIndex := 0
	emailsSent := 0

	tx, err := s.db.Beginx()
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback()

	for _, email := range emails {
		// Find a server that can send within its sliding window limit
		var server serverInfo
		var tracker *serverWindowTracker
		foundServer := false

		// Try each server in round-robin order
		for attempts := 0; attempts < len(servers); attempts++ {
			candidateServer := servers[serverIndex%len(servers)]
			candidateTracker := serverTracking[candidateServer.UUID]

			// Check if this server has sliding window limits
			if candidateServer.SlidingWindow && candidateServer.SlidingWindowRate > 0 && candidateServer.SlidingWindowDuration > 0 {
				// Check if window has expired and needs reset
				if currentTime.Sub(candidateTracker.windowStart) >= candidateServer.SlidingWindowDuration {
					candidateTracker.count = 0
					candidateTracker.windowStart = currentTime
				}

				// Check if server has capacity in current window
				if candidateTracker.count < candidateServer.SlidingWindowRate {
					server = candidateServer
					tracker = candidateTracker
					foundServer = true
					break
				}
			} else {
				// Server has no sliding window limit, can always use it
				server = candidateServer
				tracker = candidateTracker
				foundServer = true
				break
			}

			// Try next server
			serverIndex++
		}

		// If no server has capacity, we need to advance time to the next window
		if !foundServer {
			s.log.Printf("all servers at sliding window capacity, advancing time")

			// Find the earliest window reset time across all servers
			var minTimeToReset time.Duration = time.Hour * 24 // Start with a large value
			for _, srv := range servers {
				if srv.SlidingWindow && srv.SlidingWindowRate > 0 && srv.SlidingWindowDuration > 0 {
					t := serverTracking[srv.UUID]
					windowElapsed := currentTime.Sub(t.windowStart)
					timeUntilReset := srv.SlidingWindowDuration - windowElapsed

					if timeUntilReset > 0 && timeUntilReset < minTimeToReset {
						minTimeToReset = timeUntilReset
					}
				}
			}

			// Advance time to next window reset
			if minTimeToReset > 0 && minTimeToReset < time.Hour*24 {
				currentTime = currentTime.Add(minTimeToReset)
				s.log.Printf("advanced time by %v to next window", minTimeToReset)

				// Reset all expired windows
				for _, srv := range servers {
					if srv.SlidingWindow && srv.SlidingWindowDuration > 0 {
						t := serverTracking[srv.UUID]
						if currentTime.Sub(t.windowStart) >= srv.SlidingWindowDuration {
							t.count = 0
							t.windowStart = currentTime
						}
					}
				}

				// Try first server again
				server = servers[serverIndex%len(servers)]
				tracker = serverTracking[server.UUID]
			} else {
				// Fallback: use first server anyway (shouldn't happen)
				server = servers[serverIndex%len(servers)]
				tracker = serverTracking[server.UUID]
			}
		}

		// Update email with scheduled time and assigned server
		_, err := tx.Exec(`
			UPDATE email_queue
			SET scheduled_at = $1,
			    assigned_smtp_server_uuid = $2,
			    updated_at = NOW()
			WHERE id = $3
		`, currentTime, server.UUID, email.ID)
		if err != nil {
			return fmt.Errorf("error updating email schedule: %w", err)
		}

		emailsSent++
		serverIndex++

		// Increment this server's window counter
		tracker.count++

		// Advance time based on mode
		if immediateMode {
			// In immediate mode, keep all emails scheduled for NOW
			// The processor will handle rate limiting in real-time
			currentTime = startTime
		} else {
			// In scheduled mode, spread emails over time based on send rate
			if sendRatePerMinute > 0 {
				secondsPerEmail := 60.0 / float64(sendRatePerMinute)
				currentTime = currentTime.Add(time.Duration(secondsPerEmail * float64(time.Second)))
			} else {
				// If no rate limit, send 1 per second
				currentTime = currentTime.Add(1 * time.Second)
			}
		}

		// Check if we've exceeded the sending window for today (only in scheduled mode)
		if !immediateMode && !s.isWithinSendingWindow(currentTime) {
			// Move to next day's sending window
			currentTime = s.getNextSendingWindow()
			if currentTime.Day() != startTime.Day() {
				// Reset server window tracking for new day
				for _, srv := range servers {
					t := serverTracking[srv.UUID]
					t.count = 0
					t.windowStart = currentTime
				}
				serverIndex = 0
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing schedule updates: %w", err)
	}

	s.log.Printf("successfully scheduled %d emails for campaign %d", emailsSent, campaignID)
	return nil
}

// calculateSendingHours returns how many hours per day emails can be sent
func (s *Scheduler) calculateSendingHours() int {
	if s.cfg.TimeWindowStart == "" || s.cfg.TimeWindowEnd == "" {
		return 24 // No time window configured, can send 24/7
	}

	// Parse time strings (format: "HH:MM")
	startTime, err := time.Parse("15:04", s.cfg.TimeWindowStart)
	if err != nil {
		return 24
	}

	endTime, err := time.Parse("15:04", s.cfg.TimeWindowEnd)
	if err != nil {
		return 24
	}

	// Calculate duration
	duration := endTime.Sub(startTime)
	hours := int(duration.Hours())

	if hours < 0 {
		hours += 24 // Handle overnight windows
	}

	if hours == 0 {
		hours = 24 // Default to 24 hours if calculation fails
	}

	return hours
}

// getNextSendingWindow returns the next time when sending can start
func (s *Scheduler) getNextSendingWindow() time.Time {
	now := time.Now()

	// If no time window configured, can start immediately
	if s.cfg.TimeWindowStart == "" {
		return now
	}

	// Parse start time
	startTime, err := time.Parse("15:04", s.cfg.TimeWindowStart)
	if err != nil {
		return now
	}

	// Create a time for today at the window start
	windowStart := time.Date(now.Year(), now.Month(), now.Day(),
		startTime.Hour(), startTime.Minute(), 0, 0, now.Location())

	// If window has already started today and we're still within it, start now
	if now.After(windowStart) && s.isWithinSendingWindow(now) {
		return now
	}

	// If we're past today's window, move to tomorrow
	if now.After(windowStart) {
		windowStart = windowStart.Add(24 * time.Hour)
	}

	return windowStart
}

// isWithinSendingWindow checks if a given time is within the configured sending window
func (s *Scheduler) isWithinSendingWindow(t time.Time) bool {
	// If no time window is configured, always allow sending
	if s.cfg.TimeWindowStart == "" || s.cfg.TimeWindowEnd == "" {
		return true
	}

	currentTime := t.Format("15:04")

	// Simple string comparison works for HH:MM format
	return currentTime >= s.cfg.TimeWindowStart && currentTime <= s.cfg.TimeWindowEnd
}

// calculateSendRate calculates how many emails can be sent per minute
func (s *Scheduler) calculateSendRate(totalCapacity int, sendingHoursPerDay int) int {
	if totalCapacity == 0 || sendingHoursPerDay == 0 {
		return 1 // Default to 1 email per minute
	}

	// Calculate emails per hour
	emailsPerHour := totalCapacity / sendingHoursPerDay

	// Convert to emails per minute
	emailsPerMinute := emailsPerHour / 60

	if emailsPerMinute < 1 {
		emailsPerMinute = 1
	}

	// Cap at sliding window limit if configured
	if s.cfg.SlidingWindowDuration > 0 && s.cfg.SlidingWindowLimit > 0 {
		windowMinutes := int(s.cfg.SlidingWindowDuration.Minutes())
		maxPerMinute := s.cfg.SlidingWindowLimit / windowMinutes
		if maxPerMinute < emailsPerMinute {
			emailsPerMinute = maxPerMinute
		}
	}

	return emailsPerMinute
}
