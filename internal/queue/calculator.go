package queue

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/listmonk/models"
)

// Calculator estimates when campaigns will complete based on capacity constraints
type Calculator struct {
	db  *sqlx.DB
	cfg Config
}

// NewCalculator creates a new delivery calculator
func NewCalculator(db *sqlx.DB, cfg Config) *Calculator {
	return &Calculator{
		db:  db,
		cfg: cfg,
	}
}

// EstimateCampaignDelivery calculates when a campaign will be delivered
func (c *Calculator) EstimateCampaignDelivery(totalEmails int, settings models.Settings) (DeliveryEstimate, error) {
	estimate := DeliveryEstimate{
		TotalEmails:    totalEmails,
		EmailsPerServer: make(map[string]int),
	}

	// Get available SMTP servers
	var availableServers []models.Settings
	for _, smtp := range settings.SMTP {
		if smtp.Enabled && smtp.DailyLimit > 0 {
			availableServers = append(availableServers, settings)
		}
	}

	if len(availableServers) == 0 {
		return estimate, fmt.Errorf("no SMTP servers with daily limits configured")
	}

	// Calculate total daily capacity across all servers
	totalDailyCapacity := 0
	for _, smtp := range settings.SMTP {
		if smtp.Enabled && smtp.DailyLimit > 0 {
			totalDailyCapacity += smtp.DailyLimit
		}
	}

	if totalDailyCapacity == 0 {
		return estimate, fmt.Errorf("total daily capacity is zero")
	}

	// Calculate sending hours per day based on time window
	sendingHoursPerDay := c.calculateSendingHours()

	// Calculate sliding window capacity per hour
	slidingWindowPerHour := 0
	if c.cfg.SlidingWindowDuration > 0 && c.cfg.SlidingWindowLimit > 0 {
		windowsPerHour := float64(time.Hour) / float64(c.cfg.SlidingWindowDuration)
		slidingWindowPerHour = int(float64(c.cfg.SlidingWindowLimit) * windowsPerHour)
	}

	// Determine the limiting factor (daily limit vs sliding window)
	hourlyCapacity := slidingWindowPerHour
	if hourlyCapacity == 0 || sendingHoursPerDay * hourlyCapacity > totalDailyCapacity {
		// Daily limit is the constraint
		hourlyCapacity = totalDailyCapacity / sendingHoursPerDay
	}

	// Calculate how many days needed
	emailsPerDay := hourlyCapacity * sendingHoursPerDay
	if emailsPerDay > totalDailyCapacity {
		emailsPerDay = totalDailyCapacity
	}

	daysNeeded := (totalEmails + emailsPerDay - 1) / emailsPerDay // Ceiling division
	estimate.EstimatedDays = daysNeeded

	// Distribute emails across servers proportionally to their daily limits
	emailsPerServer := make(map[string]int)
	remainingEmails := totalEmails

	for _, smtp := range settings.SMTP {
		if !smtp.Enabled || smtp.DailyLimit == 0 {
			continue
		}

		// Calculate this server's share based on its proportion of total capacity
		proportion := float64(smtp.DailyLimit) / float64(totalDailyCapacity)
		serverEmails := int(float64(totalEmails) * proportion)

		// Don't exceed the server's capacity over the days needed
		maxServerCapacity := smtp.DailyLimit * daysNeeded
		if serverEmails > maxServerCapacity {
			serverEmails = maxServerCapacity
		}

		if serverEmails > remainingEmails {
			serverEmails = remainingEmails
		}

		emailsPerServer[smtp.Name] = serverEmails
		remainingEmails -= serverEmails

		if remainingEmails <= 0 {
			break
		}
	}

	estimate.EmailsPerServer = emailsPerServer
	estimate.ServersToUse = len(emailsPerServer)

	// Calculate timeline
	startTime := c.getNextSendingWindow()
	estimate.EstimatedStartTime = startTime

	// Calculate end time
	currentTime := startTime
	emailsSent := 0
	dailyBreakdown := []DailyBreakdown{}

	for day := 0; day < daysNeeded && emailsSent < totalEmails; day++ {
		dailyEmails := emailsPerDay
		if emailsSent+dailyEmails > totalEmails {
			dailyEmails = totalEmails - emailsSent
		}

		breakdown := DailyBreakdown{
			Date:         currentTime,
			EmailsToSend: dailyEmails,
			ServersUsed:  estimate.ServersToUse,
		}
		dailyBreakdown = append(dailyBreakdown, breakdown)

		emailsSent += dailyEmails

		// Calculate time to send these emails
		hoursNeeded := dailyEmails / hourlyCapacity
		if dailyEmails%hourlyCapacity > 0 {
			hoursNeeded++
		}

		// Move to next day's sending window
		currentTime = currentTime.Add(24 * time.Hour)
		currentTime = c.alignToSendingWindow(currentTime)
	}

	estimate.DailyBreakdown = dailyBreakdown
	estimate.EstimatedEndTime = currentTime
	estimate.EstimatedDuration = estimate.EstimatedEndTime.Sub(estimate.EstimatedStartTime)
	estimate.WithinSingleDay = daysNeeded == 1

	return estimate, nil
}

// calculateSendingHours returns how many hours per day emails can be sent
func (c *Calculator) calculateSendingHours() int {
	if c.cfg.TimeWindowStart == "" || c.cfg.TimeWindowEnd == "" {
		return 24 // No time window configured, can send 24/7
	}

	// Parse time strings (format: "HH:MM")
	startTime, err := time.Parse("15:04", c.cfg.TimeWindowStart)
	if err != nil {
		return 24
	}

	endTime, err := time.Parse("15:04", c.cfg.TimeWindowEnd)
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
func (c *Calculator) getNextSendingWindow() time.Time {
	now := time.Now()

	// If no time window configured, can start immediately
	if c.cfg.TimeWindowStart == "" {
		return now
	}

	// Parse start time
	startTime, err := time.Parse("15:04", c.cfg.TimeWindowStart)
	if err != nil {
		return now
	}

	// Create a time for today at the window start
	windowStart := time.Date(now.Year(), now.Month(), now.Day(),
		startTime.Hour(), startTime.Minute(), 0, 0, now.Location())

	// If window has already started today and we're still within it, start now
	if now.After(windowStart) {
		if c.cfg.TimeWindowEnd != "" {
			endTime, err := time.Parse("15:04", c.cfg.TimeWindowEnd)
			if err == nil {
				windowEnd := time.Date(now.Year(), now.Month(), now.Day(),
					endTime.Hour(), endTime.Minute(), 0, 0, now.Location())
				if now.Before(windowEnd) {
					return now
				}
			}
		}
	}

	// If we're past today's window, move to tomorrow
	if now.After(windowStart) {
		windowStart = windowStart.Add(24 * time.Hour)
	}

	return windowStart
}

// alignToSendingWindow aligns a time to the next sending window
func (c *Calculator) alignToSendingWindow(t time.Time) time.Time {
	if c.cfg.TimeWindowStart == "" {
		return t
	}

	startTime, err := time.Parse("15:04", c.cfg.TimeWindowStart)
	if err != nil {
		return t
	}

	return time.Date(t.Year(), t.Month(), t.Day(),
		startTime.Hour(), startTime.Minute(), 0, 0, t.Location())
}

// GetServerCapacitySummary returns a summary of all SMTP server capacities
func (c *Calculator) GetServerCapacitySummary(settings models.Settings) ([]ServerCapacity, error) {
	var capacities []ServerCapacity

	for _, smtp := range settings.SMTP {
		if !smtp.Enabled {
			continue
		}

		capacity := ServerCapacity{
			UUID:       smtp.UUID,
			Name:       smtp.Name,
			DailyLimit: smtp.DailyLimit,
		}

		// Get today's usage from database
		var usage int
		err := c.db.Get(&usage, `
			SELECT COALESCE(emails_sent, 0)
			FROM smtp_daily_usage
			WHERE smtp_server_uuid = $1 AND usage_date = CURRENT_DATE
		`, smtp.UUID)

		if err != nil && err.Error() != "sql: no rows in result set" {
			return nil, err
		}

		capacity.DailyUsed = usage
		capacity.DailyRemaining = capacity.DailyLimit - capacity.DailyUsed
		if capacity.DailyRemaining < 0 {
			capacity.DailyRemaining = 0
		}

		capacity.CanSendNow = capacity.DailyRemaining > 0 || capacity.DailyLimit == 0

		capacities = append(capacities, capacity)
	}

	return capacities, nil
}
