#!/bin/bash

# Test script for Shopify webhook integration
# This simulates a Shopify order webhook with proper HMAC-SHA256 signature

set -e

# Configuration
WEBHOOK_URL="https://list.bobbyseamoss.com/webhooks/shopify/orders"
WEBHOOK_SECRET="test-secret-123"  # Change this to your actual webhook secret
SUBSCRIBER_EMAIL="test@example.com"  # Change this to an actual subscriber email in your database

# Sample Shopify order payload
ORDER_PAYLOAD=$(cat <<'EOF'
{
  "id": 1234567890,
  "order_number": 1001,
  "email": "SUBSCRIBER_EMAIL_PLACEHOLDER",
  "total_price": "99.99",
  "currency": "USD",
  "created_at": "2025-11-08T14:00:00Z",
  "line_items": [
    {
      "id": 123,
      "title": "Test Product",
      "quantity": 1,
      "price": "99.99"
    }
  ]
}
EOF
)

# Replace placeholder with actual email
ORDER_PAYLOAD="${ORDER_PAYLOAD//SUBSCRIBER_EMAIL_PLACEHOLDER/$SUBSCRIBER_EMAIL}"

echo "=== Shopify Webhook Test ==="
echo ""
echo "1. Configuration:"
echo "   Webhook URL: $WEBHOOK_URL"
echo "   Subscriber Email: $SUBSCRIBER_EMAIL"
echo "   Order ID: 1234567890"
echo "   Total: \$99.99 USD"
echo ""

# Generate HMAC-SHA256 signature (base64 encoded)
HMAC_SIGNATURE=$(echo -n "$ORDER_PAYLOAD" | openssl dgst -sha256 -hmac "$WEBHOOK_SECRET" -binary | base64)

echo "2. Generated HMAC Signature:"
echo "   $HMAC_SIGNATURE"
echo ""

# Send the webhook
echo "3. Sending webhook request..."
echo ""

RESPONSE=$(curl -s -w "\nHTTP_CODE:%{http_code}" -X POST "$WEBHOOK_URL" \
  -H "Content-Type: application/json" \
  -H "X-Shopify-Hmac-Sha256: $HMAC_SIGNATURE" \
  -H "X-Shopify-Topic: orders/create" \
  -H "X-Shopify-Shop-Domain: test-shop.myshopify.com" \
  -d "$ORDER_PAYLOAD")

HTTP_CODE=$(echo "$RESPONSE" | grep "HTTP_CODE:" | cut -d: -f2)
RESPONSE_BODY=$(echo "$RESPONSE" | sed '/HTTP_CODE:/d')

echo "4. Response:"
echo "   HTTP Status: $HTTP_CODE"
echo "   Response Body: $RESPONSE_BODY"
echo ""

if [ "$HTTP_CODE" = "200" ]; then
    echo "✅ Webhook accepted!"
    echo ""
    echo "5. Next steps:"
    echo "   - Check webhook logs in database:"
    echo "     SELECT * FROM webhook_logs ORDER BY created_at DESC LIMIT 1;"
    echo ""
    echo "   - Check if purchase attribution was created:"
    echo "     SELECT * FROM purchase_attributions WHERE order_id = '1234567890';"
    echo ""
    echo "   - Check application logs:"
    echo "     az containerapp logs show --name listmonk420 --resource-group rg-listmonk420 --tail 50 | grep -i shopify"
else
    echo "❌ Webhook failed!"
    echo ""
    echo "Common issues:"
    echo "   - 401 Unauthorized: HMAC signature mismatch (check webhook_secret)"
    echo "   - 400 Bad Request: Invalid JSON payload"
    echo "   - Check application logs for details"
fi
