#!/bin/bash
#
# Azure Event Grid Webhook Testing Script for Listmonk
#
# This script sends mock Azure Event Grid events to your listmonk webhook endpoint
# for testing bounce tracking and engagement tracking before deploying to production.
#
# Prerequisites:
# - listmonk running and accessible
# - curl installed
# - jq installed (optional, for pretty output)
#
# Usage:
#   ./test-azure-webhooks.sh [webhook-url]
#
# Examples:
#   ./test-azure-webhooks.sh
#   ./test-azure-webhooks.sh http://localhost:9000/webhooks/service/azure
#   ./test-azure-webhooks.sh https://listmonk.yourdomain.com/webhooks/service/azure
#

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
WEBHOOK_URL="${1:-http://localhost:9000/webhooks/service/azure}"
VERBOSE="${VERBOSE:-0}"

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Print colored message
print_msg() {
    local color=$1
    shift
    echo -e "${color}$@${NC}"
}

# Print test header
print_test() {
    echo ""
    print_msg "$CYAN" "=========================================="
    print_msg "$CYAN" "TEST: $1"
    print_msg "$CYAN" "=========================================="
}

# Print test result
print_result() {
    local status=$1
    local message=$2

    ((TOTAL_TESTS++))

    if [ "$status" == "PASS" ]; then
        print_msg "$GREEN" "✓ PASS: $message"
        ((PASSED_TESTS++))
    else
        print_msg "$RED" "✗ FAIL: $message"
        ((FAILED_TESTS++))
    fi
}

# Send webhook and check response
send_webhook() {
    local test_name=$1
    local event_type=$2
    local payload=$3
    local expected_status=${4:-200}

    print_msg "$BLUE" "Sending $test_name..."
    print_msg "$BLUE" "Event Type: $event_type"

    if [ "$VERBOSE" == "1" ]; then
        print_msg "$YELLOW" "Payload:"
        echo "$payload" | jq '.' 2>/dev/null || echo "$payload"
    fi

    # Send request
    response=$(curl -s -w "\n%{http_code}" -X POST "$WEBHOOK_URL" \
        -H "Content-Type: application/json" \
        -H "aeg-event-type: $event_type" \
        -d "$payload")

    # Extract status code and body
    http_code=$(echo "$response" | tail -n 1)
    response_body=$(echo "$response" | sed '$d')

    print_msg "$BLUE" "HTTP Status: $http_code"

    if [ "$VERBOSE" == "1" ] && [ -n "$response_body" ]; then
        print_msg "$YELLOW" "Response:"
        echo "$response_body" | jq '.' 2>/dev/null || echo "$response_body"
    fi

    # Check status code
    if [ "$http_code" == "$expected_status" ]; then
        print_result "PASS" "$test_name (HTTP $http_code)"
        return 0
    else
        print_result "FAIL" "$test_name (Expected HTTP $expected_status, got $http_code)"
        return 1
    fi
}

print_msg "$BLUE" "=========================================="
print_msg "$BLUE" "Azure Event Grid Webhook Test Suite"
print_msg "$BLUE" "=========================================="
echo ""
print_msg "$BLUE" "Webhook URL: $WEBHOOK_URL"
print_msg "$BLUE" "Timestamp: $(date)"
echo ""

# Check if webhook is reachable
print_msg "$BLUE" "Checking if webhook endpoint is reachable..."
if curl -s -I "$WEBHOOK_URL" > /dev/null 2>&1; then
    print_msg "$GREEN" "✓ Webhook endpoint is reachable"
else
    print_msg "$RED" "✗ Webhook endpoint is not reachable"
    print_msg "$YELLOW" "  Make sure listmonk is running and accessible at: $WEBHOOK_URL"
    exit 1
fi

# =============================================================================
# TEST 1: Subscription Validation
# =============================================================================

print_test "Subscription Validation"

send_webhook \
    "Subscription Validation Event" \
    "SubscriptionValidation" \
    '[{
        "id": "test-validation-001",
        "topic": "/subscriptions/test-sub-id/resourceGroups/test-rg/providers/Microsoft.Communication/CommunicationServices/test-acs",
        "subject": "test@example.com",
        "eventType": "Microsoft.EventGrid.SubscriptionValidationEvent",
        "eventTime": "2025-11-01T12:00:00Z",
        "data": {
            "validationCode": "TEST-VALIDATION-CODE-12345",
            "validationUrl": "https://example.com/validate"
        },
        "dataVersion": "1.0",
        "metadataVersion": "1"
    }]' \
    200

# =============================================================================
# TEST 2: Email Delivery - Bounced
# =============================================================================

print_test "Email Delivery Report - Bounced"

send_webhook \
    "Hard Bounce Event" \
    "Notification" \
    '[{
        "id": "test-delivery-001",
        "topic": "/subscriptions/test-sub-id/resourceGroups/test-rg/providers/Microsoft.Communication/CommunicationServices/test-acs",
        "subject": "noreply@yourdomain.com",
        "eventType": "Microsoft.Communication.EmailDeliveryReportReceived",
        "eventTime": "2025-11-01T12:05:00Z",
        "data": {
            "sender": "noreply@yourdomain.com",
            "recipient": "bounced@invalid-domain.com",
            "messageId": "550e8400-e29b-41d4-a716-446655440001",
            "status": "Bounced",
            "deliveryStatusDetails": {
                "statusMessage": "550 5.1.1 User unknown"
            },
            "deliveryAttemptTimeStamp": "2025-11-01T12:05:00Z",
            "XListmonkCampaign": "test-campaign-uuid-001",
            "XListmonkSubscriber": "test-subscriber-uuid-001"
        },
        "dataVersion": "1.0",
        "metadataVersion": "1"
    }]' \
    200

# =============================================================================
# TEST 3: Email Delivery - Failed
# =============================================================================

print_test "Email Delivery Report - Failed"

send_webhook \
    "Failed Delivery Event" \
    "Notification" \
    '[{
        "id": "test-delivery-002",
        "topic": "/subscriptions/test-sub-id/resourceGroups/test-rg/providers/Microsoft.Communication/CommunicationServices/test-acs",
        "subject": "noreply@yourdomain.com",
        "eventType": "Microsoft.Communication.EmailDeliveryReportReceived",
        "eventTime": "2025-11-01T12:06:00Z",
        "data": {
            "sender": "noreply@yourdomain.com",
            "recipient": "failed@example.com",
            "messageId": "550e8400-e29b-41d4-a716-446655440002",
            "status": "Failed",
            "deliveryStatusDetails": {
                "statusMessage": "SMTP connection failed"
            },
            "deliveryAttemptTimeStamp": "2025-11-01T12:06:00Z",
            "XListmonkCampaign": "test-campaign-uuid-002",
            "XListmonkSubscriber": "test-subscriber-uuid-002"
        },
        "dataVersion": "1.0",
        "metadataVersion": "1"
    }]' \
    200

# =============================================================================
# TEST 4: Email Delivery - Suppressed
# =============================================================================

print_test "Email Delivery Report - Suppressed"

send_webhook \
    "Suppressed Delivery Event" \
    "Notification" \
    '[{
        "id": "test-delivery-003",
        "topic": "/subscriptions/test-sub-id/resourceGroups/test-rg/providers/Microsoft.Communication/CommunicationServices/test-acs",
        "subject": "noreply@yourdomain.com",
        "eventType": "Microsoft.Communication.EmailDeliveryReportReceived",
        "eventTime": "2025-11-01T12:07:00Z",
        "data": {
            "sender": "noreply@yourdomain.com",
            "recipient": "suppressed@example.com",
            "messageId": "550e8400-e29b-41d4-a716-446655440003",
            "status": "Suppressed",
            "deliveryStatusDetails": {
                "statusMessage": "Recipient previously bounced"
            },
            "deliveryAttemptTimeStamp": "2025-11-01T12:07:00Z",
            "XListmonkCampaign": "test-campaign-uuid-003",
            "XListmonkSubscriber": "test-subscriber-uuid-003"
        },
        "dataVersion": "1.0",
        "metadataVersion": "1"
    }]' \
    200

# =============================================================================
# TEST 5: Email Delivery - FilteredSpam
# =============================================================================

print_test "Email Delivery Report - FilteredSpam"

send_webhook \
    "Spam Filtered Event" \
    "Notification" \
    '[{
        "id": "test-delivery-004",
        "topic": "/subscriptions/test-sub-id/resourceGroups/test-rg/providers/Microsoft.Communication/CommunicationServices/test-acs",
        "subject": "noreply@yourdomain.com",
        "eventType": "Microsoft.Communication.EmailDeliveryReportReceived",
        "eventTime": "2025-11-01T12:08:00Z",
        "data": {
            "sender": "noreply@yourdomain.com",
            "recipient": "spamfiltered@example.com",
            "messageId": "550e8400-e29b-41d4-a716-446655440004",
            "status": "FilteredSpam",
            "deliveryStatusDetails": {
                "statusMessage": "Message identified as spam"
            },
            "deliveryAttemptTimeStamp": "2025-11-01T12:08:00Z",
            "XListmonkCampaign": "test-campaign-uuid-004",
            "XListmonkSubscriber": "test-subscriber-uuid-004"
        },
        "dataVersion": "1.0",
        "metadataVersion": "1"
    }]' \
    200

# =============================================================================
# TEST 6: Email Delivery - Quarantined
# =============================================================================

print_test "Email Delivery Report - Quarantined"

send_webhook \
    "Quarantined Delivery Event" \
    "Notification" \
    '[{
        "id": "test-delivery-005",
        "topic": "/subscriptions/test-sub-id/resourceGroups/test-rg/providers/Microsoft.Communication/CommunicationServices/test-acs",
        "subject": "noreply@yourdomain.com",
        "eventType": "Microsoft.Communication.EmailDeliveryReportReceived",
        "eventTime": "2025-11-01T12:09:00Z",
        "data": {
            "sender": "noreply@yourdomain.com",
            "recipient": "quarantined@example.com",
            "messageId": "550e8400-e29b-41d4-a716-446655440005",
            "status": "Quarantined",
            "deliveryStatusDetails": {
                "statusMessage": "Message quarantined for review"
            },
            "deliveryAttemptTimeStamp": "2025-11-01T12:09:00Z",
            "XListmonkCampaign": "test-campaign-uuid-005",
            "XListmonkSubscriber": "test-subscriber-uuid-005"
        },
        "dataVersion": "1.0",
        "metadataVersion": "1"
    }]' \
    200

# =============================================================================
# TEST 7: Email Delivery - Delivered (Success)
# =============================================================================

print_test "Email Delivery Report - Delivered"

send_webhook \
    "Successful Delivery Event" \
    "Notification" \
    '[{
        "id": "test-delivery-006",
        "topic": "/subscriptions/test-sub-id/resourceGroups/test-rg/providers/Microsoft.Communication/CommunicationServices/test-acs",
        "subject": "noreply@yourdomain.com",
        "eventType": "Microsoft.Communication.EmailDeliveryReportReceived",
        "eventTime": "2025-11-01T12:10:00Z",
        "data": {
            "sender": "noreply@yourdomain.com",
            "recipient": "success@example.com",
            "messageId": "550e8400-e29b-41d4-a716-446655440006",
            "status": "Delivered",
            "deliveryStatusDetails": {
                "statusMessage": "250 OK"
            },
            "deliveryAttemptTimeStamp": "2025-11-01T12:10:00Z",
            "XListmonkCampaign": "test-campaign-uuid-006",
            "XListmonkSubscriber": "test-subscriber-uuid-006"
        },
        "dataVersion": "1.0",
        "metadataVersion": "1"
    }]' \
    200

# =============================================================================
# TEST 8: Email Engagement - View/Open
# =============================================================================

print_test "Email Engagement - View"

send_webhook \
    "Email View Event" \
    "Notification" \
    '[{
        "id": "test-engagement-001",
        "topic": "/subscriptions/test-sub-id/resourceGroups/test-rg/providers/Microsoft.Communication/CommunicationServices/test-acs",
        "subject": "noreply@yourdomain.com",
        "eventType": "Microsoft.Communication.EmailEngagementTrackingReportReceived",
        "eventTime": "2025-11-01T12:15:00Z",
        "data": {
            "sender": "noreply@yourdomain.com",
            "recipient": "",
            "messageId": "550e8400-e29b-41d4-a716-446655440007",
            "userActionTimeStamp": "2025-11-01T12:15:00Z",
            "engagementContext": "",
            "userAgent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
            "engagementType": "view",
            "XListmonkCampaign": "test-campaign-uuid-007",
            "XListmonkSubscriber": "test-subscriber-uuid-007"
        },
        "dataVersion": "1.0",
        "metadataVersion": "1"
    }]' \
    200

# =============================================================================
# TEST 9: Email Engagement - Click
# =============================================================================

print_test "Email Engagement - Click"

send_webhook \
    "Email Click Event" \
    "Notification" \
    '[{
        "id": "test-engagement-002",
        "topic": "/subscriptions/test-sub-id/resourceGroups/test-rg/providers/Microsoft.Communication/CommunicationServices/test-acs",
        "subject": "noreply@yourdomain.com",
        "eventType": "Microsoft.Communication.EmailEngagementTrackingReportReceived",
        "eventTime": "2025-11-01T12:16:00Z",
        "data": {
            "sender": "noreply@yourdomain.com",
            "recipient": "",
            "messageId": "550e8400-e29b-41d4-a716-446655440008",
            "userActionTimeStamp": "2025-11-01T12:16:00Z",
            "engagementContext": "https://yourwebsite.com/landing-page",
            "userAgent": "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X) AppleWebKit/605.1.15",
            "engagementType": "click",
            "XListmonkCampaign": "test-campaign-uuid-008",
            "XListmonkSubscriber": "test-subscriber-uuid-008"
        },
        "dataVersion": "1.0",
        "metadataVersion": "1"
    }]' \
    200

# =============================================================================
# TEST 10: Invalid Event Type
# =============================================================================

print_test "Invalid Event Type (Error Case)"

send_webhook \
    "Invalid Event Type" \
    "Notification" \
    '[{
        "id": "test-invalid-001",
        "topic": "/subscriptions/test-sub-id/resourceGroups/test-rg/providers/Microsoft.Communication/CommunicationServices/test-acs",
        "subject": "noreply@yourdomain.com",
        "eventType": "Microsoft.Communication.InvalidEventType",
        "eventTime": "2025-11-01T12:20:00Z",
        "data": {},
        "dataVersion": "1.0",
        "metadataVersion": "1"
    }]' \
    200

# =============================================================================
# Summary
# =============================================================================

echo ""
print_msg "$CYAN" "=========================================="
print_msg "$CYAN" "TEST SUMMARY"
print_msg "$CYAN" "=========================================="
echo ""

print_msg "$BLUE" "Total Tests: $TOTAL_TESTS"
print_msg "$GREEN" "Passed: $PASSED_TESTS"
print_msg "$RED" "Failed: $FAILED_TESTS"

echo ""

if [ $FAILED_TESTS -eq 0 ]; then
    print_msg "$GREEN" "✓ All tests passed!"
    echo ""
    print_msg "$BLUE" "Next steps:"
    echo "  1. Check listmonk logs for processed events"
    echo "  2. Verify bounces table has test bounce records"
    echo "  3. Verify campaign_views table has view records"
    echo "  4. Verify link_clicks table has click records"
    echo ""
    print_msg "$BLUE" "SQL queries to verify:"
    print_msg "$YELLOW" "  -- Check bounces"
    print_msg "$YELLOW" "  SELECT * FROM bounces WHERE source = 'azure' ORDER BY created_at DESC LIMIT 10;"
    echo ""
    print_msg "$YELLOW" "  -- Check views"
    print_msg "$YELLOW" "  SELECT * FROM campaign_views ORDER BY created_at DESC LIMIT 10;"
    echo ""
    print_msg "$YELLOW" "  -- Check clicks"
    print_msg "$YELLOW" "  SELECT * FROM link_clicks ORDER BY created_at DESC LIMIT 10;"
    echo ""
    exit 0
else
    print_msg "$RED" "✗ Some tests failed"
    echo ""
    print_msg "$YELLOW" "Troubleshooting:"
    echo "  1. Check listmonk logs: tail -f /var/log/listmonk/listmonk.log"
    echo "  2. Verify Azure Event Grid is enabled in settings"
    echo "  3. Verify webhook endpoint is accessible"
    echo "  4. Check database connectivity"
    echo ""
    exit 1
fi
