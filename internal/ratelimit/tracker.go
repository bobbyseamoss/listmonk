package ratelimit

import (
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

// Common rate limit error patterns for different providers
var (
	// Azure SMTP rate limit patterns
	azureRateLimitPatterns = []string{
		"too many requests",
		"rate limit",
		"throttl",
		"451 4.7.1",     // Azure rate limit SMTP code
		"452 4.3.1",     // Too many recipients
		"421 4.7.0",     // Connection rate limit
		"too many connections",
	}

	// Retry-After header pattern (in seconds or HTTP date)
	retryAfterPattern = regexp.MustCompile(`(?i)retry[-\s]?after[:\s]+(\d+)`)
)

// DisabledServer tracks a temporarily disabled SMTP server
type DisabledServer struct {
	UUID        string
	Name        string
	Reason      string
	DisabledAt  time.Time
	ReenableAt  time.Time
}

// Tracker tracks rate-limited SMTP servers and manages their cooldown periods
type Tracker struct {
	db              *sqlx.DB
	log             *log.Logger
	disabledServers map[string]*DisabledServer
	mu              sync.RWMutex

	// Default cooldown if we can't parse the Retry-After value
	defaultCooldown time.Duration

	// Extra margin to add to cooldown (percentage, e.g., 0.10 = 10%)
	cooldownMargin float64
}

// NewTracker creates a new rate limit tracker
func NewTracker(db *sqlx.DB, log *log.Logger) *Tracker {
	return &Tracker{
		db:              db,
		log:             log,
		disabledServers: make(map[string]*DisabledServer),
		defaultCooldown: 1 * time.Hour,  // Default 1 hour cooldown if not specified
		cooldownMargin:  0.10,           // Add 10% extra margin
	}
}

// IsRateLimitError checks if an error message indicates a rate limit
func (t *Tracker) IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	for _, pattern := range azureRateLimitPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}

// ParseRetryAfter extracts the retry-after duration from an error message or response
// Returns the duration plus the configured margin
func (t *Tracker) ParseRetryAfter(errStr string) time.Duration {
	// Try to find a Retry-After value in seconds
	matches := retryAfterPattern.FindStringSubmatch(errStr)
	if len(matches) >= 2 {
		if seconds, err := strconv.Atoi(matches[1]); err == nil {
			duration := time.Duration(seconds) * time.Second
			// Add margin (e.g., 10% extra)
			margin := time.Duration(float64(duration) * t.cooldownMargin)
			return duration + margin
		}
	}

	// Look for common patterns like "1 hour", "60 minutes", etc.
	lowerErr := strings.ToLower(errStr)
	if strings.Contains(lowerErr, "1 hour") || strings.Contains(lowerErr, "one hour") {
		return time.Hour + time.Duration(float64(time.Hour)*t.cooldownMargin)
	}
	if strings.Contains(lowerErr, "30 minute") {
		return 30*time.Minute + time.Duration(float64(30*time.Minute)*t.cooldownMargin)
	}
	if strings.Contains(lowerErr, "15 minute") {
		return 15*time.Minute + time.Duration(float64(15*time.Minute)*t.cooldownMargin)
	}

	// Default cooldown with margin
	return t.defaultCooldown + time.Duration(float64(t.defaultCooldown)*t.cooldownMargin)
}

// DisableServer temporarily disables an SMTP server due to rate limiting
func (t *Tracker) DisableServer(serverUUID, serverName, reason string, cooldown time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	reenableAt := now.Add(cooldown)

	disabled := &DisabledServer{
		UUID:       serverUUID,
		Name:       serverName,
		Reason:     reason,
		DisabledAt: now,
		ReenableAt: reenableAt,
	}

	t.disabledServers[serverUUID] = disabled

	// Update the database to disable the server
	_, err := t.db.Exec(`
		UPDATE settings
		SET value = jsonb_set(
			value::jsonb,
			array[(
				SELECT (key::int - 1)::text
				FROM jsonb_array_elements(value::jsonb) WITH ORDINALITY arr(elem, key)
				WHERE elem->>'uuid' = $1
				LIMIT 1
			), 'enabled'],
			'false'
		)
		WHERE key = 'smtp'
	`, serverUUID)

	if err != nil {
		t.log.Printf("⚠️ rate limit tracker: failed to disable server %s in database: %v", serverName, err)
	} else {
		t.log.Printf("🚫 rate limit tracker: disabled server '%s' (UUID: %s) due to rate limit. Will re-enable at %s (cooldown: %v)",
			serverName, serverUUID, reenableAt.Format(time.RFC3339), cooldown)
		t.log.Printf("   Reason: %s", reason)
	}
}

// IsServerDisabled checks if a server is currently disabled due to rate limiting
func (t *Tracker) IsServerDisabled(serverUUID string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	disabled, exists := t.disabledServers[serverUUID]
	if !exists {
		return false
	}

	// Check if the cooldown has expired
	if time.Now().After(disabled.ReenableAt) {
		return false
	}

	return true
}

// GetDisabledServers returns a list of currently disabled servers
func (t *Tracker) GetDisabledServers() []*DisabledServer {
	t.mu.RLock()
	defer t.mu.RUnlock()

	servers := make([]*DisabledServer, 0, len(t.disabledServers))
	for _, s := range t.disabledServers {
		if time.Now().Before(s.ReenableAt) {
			servers = append(servers, s)
		}
	}
	return servers
}

// CheckAndReenableServers checks if any disabled servers should be re-enabled
// This should be called periodically (e.g., every minute)
func (t *Tracker) CheckAndReenableServers() {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	for uuid, disabled := range t.disabledServers {
		if now.After(disabled.ReenableAt) {
			// Re-enable the server in the database
			_, err := t.db.Exec(`
				UPDATE settings
				SET value = jsonb_set(
					value::jsonb,
					array[(
						SELECT (key::int - 1)::text
						FROM jsonb_array_elements(value::jsonb) WITH ORDINALITY arr(elem, key)
						WHERE elem->>'uuid' = $1
						LIMIT 1
					), 'enabled'],
					'true'
				)
				WHERE key = 'smtp'
			`, uuid)

			if err != nil {
				t.log.Printf("⚠️ rate limit tracker: failed to re-enable server %s in database: %v", disabled.Name, err)
			} else {
				t.log.Printf("✅ rate limit tracker: re-enabled server '%s' (UUID: %s) after cooldown period",
					disabled.Name, uuid)
			}

			// Remove from disabled list
			delete(t.disabledServers, uuid)
		}
	}
}

// StartReenableScheduler starts a goroutine that periodically checks and re-enables servers
func (t *Tracker) StartReenableScheduler(stopChan <-chan struct{}) {
	t.log.Println("starting rate limit re-enable scheduler")

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.CheckAndReenableServers()
		case <-stopChan:
			t.log.Println("stopping rate limit re-enable scheduler")
			return
		}
	}
}

// HandleRateLimitError processes a rate limit error and disables the server
func (t *Tracker) HandleRateLimitError(serverUUID, serverName string, err error) {
	if !t.IsRateLimitError(err) {
		return
	}

	errStr := err.Error()
	cooldown := t.ParseRetryAfter(errStr)

	t.DisableServer(serverUUID, serverName, errStr, cooldown)
}

// HandleAzureDeliveryFailure processes an Azure webhook delivery failure
// and disables the server if it's a rate limit error
func (t *Tracker) HandleAzureDeliveryFailure(serverUUID, serverName, status string, statusDetails map[string]interface{}) {
	// Check if status indicates rate limiting
	statusLower := strings.ToLower(status)
	if statusLower != "failed" && statusLower != "throttled" {
		return
	}

	// Check status details for rate limit indicators
	var reasonStr string
	if details, ok := statusDetails["statusMessage"].(string); ok {
		reasonStr = details
	}
	if reason, ok := statusDetails["reason"].(string); ok {
		reasonStr += " " + reason
	}

	// Convert statusDetails to string for pattern matching
	detailsLower := strings.ToLower(reasonStr)
	isRateLimit := false
	for _, pattern := range azureRateLimitPatterns {
		if strings.Contains(detailsLower, pattern) {
			isRateLimit = true
			break
		}
	}

	if !isRateLimit {
		return
	}

	cooldown := t.ParseRetryAfter(reasonStr)
	t.DisableServer(serverUUID, serverName, reasonStr, cooldown)
}
