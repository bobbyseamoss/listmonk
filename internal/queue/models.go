package queue

import (
	"database/sql"
	"time"
)

// EmailQueueItem represents a single email in the queue
type EmailQueueItem struct {
	ID                    int64     `db:"id" json:"id"`
	CampaignID            int       `db:"campaign_id" json:"campaign_id"`
	SubscriberID          int       `db:"subscriber_id" json:"subscriber_id"`
	Status                string    `db:"status" json:"status"`
	Priority              int       `db:"priority" json:"priority"`
	ScheduledAt           time.Time `db:"scheduled_at" json:"scheduled_at"`
	SentAt                 *time.Time       `db:"sent_at" json:"sent_at,omitempty"`
	AssignedSMTPServerUUID sql.NullString  `db:"assigned_smtp_server_uuid" json:"assigned_smtp_server_uuid"`
	RetryCount             int             `db:"retry_count" json:"retry_count"`
	LastError              sql.NullString  `db:"last_error" json:"last_error,omitempty"`
	CreatedAt             time.Time `db:"created_at" json:"created_at"`
	UpdatedAt             time.Time `db:"updated_at" json:"updated_at"`
}

// Queue item statuses
const (
	StatusQueued    = "queued"
	StatusSending   = "sending"
	StatusSent      = "sent"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
)

// SMTPDailyUsage tracks how many emails an SMTP server has sent today
type SMTPDailyUsage struct {
	ID               int64     `db:"id" json:"id"`
	SMTPServerUUID   string    `db:"smtp_server_uuid" json:"smtp_server_uuid"`
	UsageDate        time.Time `db:"usage_date" json:"usage_date"`
	EmailsSent       int       `db:"emails_sent" json:"emails_sent"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time `db:"updated_at" json:"updated_at"`
}

// SMTPRateLimitState tracks the sliding window rate limit for an SMTP server
type SMTPRateLimitState struct {
	ID               int64     `db:"id" json:"id"`
	SMTPServerUUID   string    `db:"smtp_server_uuid" json:"smtp_server_uuid"`
	WindowStart      time.Time `db:"window_start" json:"window_start"`
	EmailsInWindow   int       `db:"emails_in_window" json:"emails_in_window"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time `db:"updated_at" json:"updated_at"`
}

// ServerCapacity represents the available capacity for an SMTP server
type ServerCapacity struct {
	UUID                string
	Name                string
	DailyLimit          int
	DailyUsed           int
	DailyRemaining      int
	SlidingWindowLimit  int
	SlidingWindowUsed   int
	SlidingWindowPeriod time.Duration
	CanSendNow          bool
	NextAvailableAt     time.Time
}

// DeliveryEstimate represents an estimate of when a campaign will complete
type DeliveryEstimate struct {
	TotalEmails         int           `json:"total_emails"`
	EstimatedStartTime  time.Time     `json:"estimated_start_time"`
	EstimatedEndTime    time.Time     `json:"estimated_end_time"`
	EstimatedDuration   time.Duration `json:"estimated_duration"`
	EstimatedDays       int           `json:"estimated_days"`
	ServersToUse        int           `json:"servers_to_use"`
	EmailsPerServer     map[string]int `json:"emails_per_server"`
	DailyBreakdown      []DailyBreakdown `json:"daily_breakdown"`
	WithinSingleDay     bool          `json:"within_single_day"`
}

// DailyBreakdown shows how many emails will be sent each day
type DailyBreakdown struct {
	Date           time.Time `json:"date"`
	EmailsToSend   int       `json:"emails_to_send"`
	ServersUsed    int       `json:"servers_used"`
}

// QueueStats provides statistics about the email queue
type QueueStats struct {
	TotalQueued     int            `json:"total_queued"`
	TotalSending    int            `json:"total_sending"`
	TotalSent       int            `json:"total_sent"`
	TotalFailed     int            `json:"total_failed"`
	ByCampaign      map[int]int    `json:"by_campaign"`
	OldestScheduled *time.Time     `json:"oldest_scheduled,omitempty"`
	NewestScheduled *time.Time     `json:"newest_scheduled,omitempty"`
}
