#!/bin/bash
# Script to retroactively attribute Shopify purchases to subscribers
# This updates existing webhook logs to create purchase attributions for any subscriber

set -e

# Database connection parameters
DB_HOST="${LISTMONK_DB_HOST:-listmonk420-db.postgres.database.azure.com}"
DB_PORT="${LISTMONK_DB_PORT:-5432}"
DB_USER="${LISTMONK_DB_USER:-listmonkadmin}"
DB_PASSWORD="${LISTMONK_DB_PASSWORD:-T@intshr3dd3r}"
DB_NAME="${LISTMONK_DB_DATABASE:-listmonk}"

echo "Reprocessing Shopify webhook logs to create purchase attributions..."
echo "Using database: $DB_HOST:$DB_PORT/$DB_NAME"
echo ""

# Execute SQL to reprocess webhooks
PGPASSWORD="$DB_PASSWORD" psql \
  -h "$DB_HOST" \
  -p "$DB_PORT" \
  -U "$DB_USER" \
  -d "$DB_NAME" \
  --set=sslmode=require \
  -v ON_ERROR_STOP=1 \
  <<'SQL'

-- Create temporary function to reprocess webhooks
CREATE OR REPLACE FUNCTION reprocess_shopify_purchases() RETURNS TABLE (
  result_order_id TEXT,
  result_customer_email TEXT,
  result_subscriber_id INTEGER,
  result_action TEXT
) AS $$
DECLARE
  webhook_record RECORD;
  order_data JSONB;
  email_var TEXT;
  sub_id INTEGER;
  recent_campaign_id INTEGER;
  order_id_str TEXT;
  order_num TEXT;
  total DECIMAL(10,2);
  curr TEXT;
  existing_count INTEGER;
BEGIN
  -- Loop through all Shopify webhooks that are order events
  FOR webhook_record IN
    SELECT id, request_body::jsonb as body
    FROM webhook_logs
    WHERE webhook_type = 'shopify'
      AND event_type = 'order'
    ORDER BY created_at ASC
  LOOP
    BEGIN
      order_data := webhook_record.body;

      -- Extract order details
      order_id_str := order_data->>'id';
      order_num := order_data->>'order_number';
      email_var := LOWER(order_data->>'contact_email');
      total := COALESCE((order_data->>'total_price')::DECIMAL(10,2), 0);
      curr := COALESCE(order_data->>'currency', 'USD');

      -- Skip if missing required fields
      IF email_var IS NULL OR order_id_str IS NULL THEN
        CONTINUE;
      END IF;

      -- Check if purchase already attributed
      SELECT COUNT(*) INTO existing_count
      FROM purchase_attributions
      WHERE order_id = order_id_str;

      IF existing_count > 0 THEN
        -- Already attributed, skip
        result_order_id := order_id_str;
        result_customer_email := email_var;
        result_subscriber_id := NULL;
        result_action := 'skipped_existing';
        RETURN NEXT;
        CONTINUE;
      END IF;

      -- Check if email is a subscriber
      SELECT id INTO sub_id
      FROM subscribers
      WHERE LOWER(subscribers.email) = email_var
      LIMIT 1;

      IF sub_id IS NOT NULL THEN
        -- Find most recent campaign sent to this subscriber
        recent_campaign_id := NULL;
        BEGIN
          SELECT campaign_id INTO recent_campaign_id
          FROM azure_delivery_events
          WHERE subscriber_id = sub_id
            AND status = 'Delivered'
          ORDER BY event_timestamp DESC
          LIMIT 1;
        EXCEPTION WHEN NO_DATA_FOUND THEN
          -- No delivered campaigns, try to find most recent running campaign for their lists
          BEGIN
            SELECT c.id INTO recent_campaign_id
            FROM campaigns c
            JOIN campaign_lists cl ON cl.campaign_id = c.id
            JOIN subscriber_lists sl ON sl.list_id = cl.list_id
            WHERE sl.subscriber_id = sub_id
              AND c.status = 'running'
            ORDER BY c.started_at DESC
            LIMIT 1;
          EXCEPTION WHEN NO_DATA_FOUND THEN
            recent_campaign_id := NULL;
          END;
        END;

        -- Create purchase attribution
        INSERT INTO purchase_attributions (
          campaign_id,
          subscriber_id,
          order_id,
          order_number,
          customer_email,
          total_price,
          currency,
          attributed_via,
          confidence,
          shopify_data
        ) VALUES (
          recent_campaign_id,  -- Most recent campaign sent to subscriber
          sub_id,
          order_id_str,
          order_num,
          email_var,
          total,
          curr,
          'is_subscriber',
          'high',
          order_data
        );

        result_order_id := order_id_str;
        result_customer_email := email_var;
        result_subscriber_id := sub_id;
        result_action := 'created';
        RETURN NEXT;
      ELSE
        -- Not a subscriber, skip
        result_order_id := order_id_str;
        result_customer_email := email_var;
        result_subscriber_id := NULL;
        result_action := 'not_subscriber';
        RETURN NEXT;
      END IF;

    EXCEPTION WHEN OTHERS THEN
      -- Log error and continue
      result_order_id := order_id_str;
      result_customer_email := email_var;
      result_subscriber_id := NULL;
      result_action := 'error: ' || SQLERRM;
      RETURN NEXT;
      CONTINUE;
    END;
  END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Execute the reprocessing
SELECT * FROM reprocess_shopify_purchases();

-- Show summary
SELECT
  result_action as action,
  COUNT(*) as count
FROM reprocess_shopify_purchases()
GROUP BY result_action
ORDER BY result_action;

-- Drop the temporary function
DROP FUNCTION reprocess_shopify_purchases();

SQL

echo ""
echo "Reprocessing complete!"
echo ""
echo "Summary of purchase attributions:"
PGPASSWORD="$DB_PASSWORD" psql \
  -h "$DB_HOST" \
  -p "$DB_PORT" \
  -U "$DB_USER" \
  -d "$DB_NAME" \
  --set=sslmode=require \
  -c "
SELECT
  COUNT(*) as total_attributions,
  COUNT(campaign_id) as with_campaign,
  COUNT(*) FILTER (WHERE campaign_id IS NULL) as without_campaign,
  COUNT(*) FILTER (WHERE attributed_via = 'is_subscriber') as new_method,
  COUNT(*) FILTER (WHERE attributed_via = 'email_open') as old_method,
  SUM(total_price) as total_revenue
FROM purchase_attributions;
"
