-- Enable TLS Skip Verify for all bounce mailboxes
-- This will fix the "certificate is not valid for any names" errors

UPDATE settings
SET value = jsonb_set(
    value,
    '{mailboxes}',
    (
        SELECT jsonb_agg(
            jsonb_set(mailbox, '{tls_skip_verify}', 'true'::jsonb)
        )
        FROM jsonb_array_elements(value->'mailboxes') AS mailbox
    )
)
WHERE key = 'bounce';

-- Verify the update
SELECT
    jsonb_array_elements(value->'mailboxes')->>'name' as mailbox_name,
    jsonb_array_elements(value->'mailboxes')->>'host' as host,
    jsonb_array_elements(value->'mailboxes')->>'tls_skip_verify' as tls_skip_verify
FROM settings
WHERE key = 'bounce'
LIMIT 10;
