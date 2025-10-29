-- Migration: Queue-based email delivery system
-- This migration adds tables and schema changes to support:
-- 1. Email queue with priority and scheduling
-- 2. Per-SMTP server daily usage tracking
-- 3. SMTP rate limit state tracking
-- 4. Campaign queue tracking
-- 5. SMTP server from_email and daily_limit settings

-- Create email_queue table
-- Stores all emails waiting to be sent with priority and scheduling information
CREATE TABLE IF NOT EXISTS email_queue (
    id BIGSERIAL PRIMARY KEY,
    campaign_id INT NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    subscriber_id INT NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE,

    -- Queue management
    status VARCHAR(20) NOT NULL DEFAULT 'queued',
        -- queued: waiting to be sent
        -- sending: currently being processed
        -- sent: successfully sent
        -- failed: failed after max retries
        -- cancelled: campaign was cancelled
    priority INT NOT NULL DEFAULT 0,
        -- Higher priority = sent first
        -- Can be used for urgent campaigns

    -- Scheduling
    scheduled_at TIMESTAMP WITH TIME ZONE NOT NULL,
        -- When this email should be sent (respects time windows)
    sent_at TIMESTAMP WITH TIME ZONE,
        -- When email was actually sent

    -- Server assignment
    assigned_smtp_server_uuid VARCHAR(255),
        -- Which SMTP server will/did send this email

    -- Retry tracking
    retry_count INT NOT NULL DEFAULT 0,
    last_error TEXT,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Indexes for performance
    CONSTRAINT email_queue_status_check CHECK (status IN ('queued', 'sending', 'sent', 'failed', 'cancelled'))
);

CREATE INDEX idx_email_queue_status ON email_queue(status);
CREATE INDEX idx_email_queue_scheduled_at ON email_queue(scheduled_at);
CREATE INDEX idx_email_queue_campaign_id ON email_queue(campaign_id);
CREATE INDEX idx_email_queue_assigned_smtp ON email_queue(assigned_smtp_server_uuid);
CREATE INDEX idx_email_queue_status_scheduled ON email_queue(status, scheduled_at);

-- Create smtp_daily_usage table
-- Tracks how many emails each SMTP server has sent per day
CREATE TABLE IF NOT EXISTS smtp_daily_usage (
    id BIGSERIAL PRIMARY KEY,
    smtp_server_uuid VARCHAR(255) NOT NULL,
    usage_date DATE NOT NULL,
    emails_sent INT NOT NULL DEFAULT 0,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Unique constraint: one row per server per day
    CONSTRAINT smtp_daily_usage_unique UNIQUE (smtp_server_uuid, usage_date)
);

CREATE INDEX idx_smtp_daily_usage_uuid_date ON smtp_daily_usage(smtp_server_uuid, usage_date);

-- Create smtp_rate_limit_state table
-- Tracks the sliding window rate limit state for each SMTP server
CREATE TABLE IF NOT EXISTS smtp_rate_limit_state (
    id BIGSERIAL PRIMARY KEY,
    smtp_server_uuid VARCHAR(255) NOT NULL UNIQUE,

    -- Sliding window tracking
    window_start TIMESTAMP WITH TIME ZONE NOT NULL,
    emails_in_window INT NOT NULL DEFAULT 0,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_smtp_rate_limit_uuid ON smtp_rate_limit_state(smtp_server_uuid);

-- Modify campaigns table to add queue tracking
-- These columns track whether a campaign uses the queue system
ALTER TABLE campaigns
    ADD COLUMN IF NOT EXISTS use_queue BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS queued_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS queue_completed_at TIMESTAMP WITH TIME ZONE;

-- Add indexes for queue-related campaign queries
CREATE INDEX IF NOT EXISTS idx_campaigns_use_queue ON campaigns(use_queue);
CREATE INDEX IF NOT EXISTS idx_campaigns_queued_at ON campaigns(queued_at);

-- Note: SMTP server settings (daily_limit and from_email) are stored in the
-- settings table as JSON. They will be added to the Go structs and handled
-- in the application code rather than as separate database columns.

COMMENT ON TABLE email_queue IS 'Queue of all emails waiting to be sent, with priority and scheduling';
COMMENT ON TABLE smtp_daily_usage IS 'Tracks daily email count per SMTP server for enforcing daily limits';
COMMENT ON TABLE smtp_rate_limit_state IS 'Tracks sliding window rate limit state for each SMTP server';
COMMENT ON COLUMN campaigns.use_queue IS 'Whether this campaign uses the queue-based delivery system';
COMMENT ON COLUMN campaigns.queued_at IS 'When the campaign emails were added to the queue';
COMMENT ON COLUMN campaigns.queue_completed_at IS 'When all queued emails for this campaign were sent';
