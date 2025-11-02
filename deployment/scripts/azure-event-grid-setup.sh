#!/bin/bash
#
# Azure Event Grid Webhook Setup Script for Listmonk
#
# This script creates Event Grid subscriptions for Azure Communication Services
# email events across multiple domains to enable real-time bounce and engagement tracking.
#
# Prerequisites:
# - Azure CLI installed and logged in (az login)
# - Contributor or Event Grid Contributor role on the subscription
# - Azure Communication Services resources already provisioned
#
# Usage:
#   ./azure-event-grid-setup.sh <webhook-endpoint-url> [resource-group]
#
# Examples:
#   ./azure-event-grid-setup.sh https://listmonk.yourdomain.com/webhooks/service/azure
#   ./azure-event-grid-setup.sh https://listmonk.yourdomain.com/webhooks/service/azure rg-listmonk420
#
# What this script does:
# - Finds all Azure Communication Services resources
# - Creates Event Grid subscriptions for each resource
# - Configures delivery and engagement event types
# - Sets up retry policy and event filtering
#

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
WEBHOOK_ENDPOINT="${1:-}"
RESOURCE_GROUP="${2:-}"

# Counters
TOTAL_RESOURCES=0
SUCCESS_COUNT=0
SKIP_COUNT=0
FAIL_COUNT=0

# Print colored message
print_msg() {
    local color=$1
    shift
    echo -e "${color}$@${NC}"
}

# Print section header
print_header() {
    echo ""
    print_msg "$BLUE" "=========================================="
    print_msg "$BLUE" "$1"
    print_msg "$BLUE" "=========================================="
    echo ""
}

# Validate inputs
if [ -z "$WEBHOOK_ENDPOINT" ]; then
    print_msg "$RED" "❌ Error: Webhook endpoint URL required"
    echo ""
    echo "Usage: $0 <webhook-endpoint-url> [resource-group]"
    echo ""
    echo "Examples:"
    echo "  $0 https://listmonk.yourdomain.com/webhooks/service/azure"
    echo "  $0 https://listmonk.yourdomain.com/webhooks/service/azure rg-listmonk420"
    echo ""
    exit 1
fi

# Validate webhook URL format
if [[ ! "$WEBHOOK_ENDPOINT" =~ ^https?:// ]]; then
    print_msg "$RED" "❌ Error: Webhook endpoint must start with http:// or https://"
    print_msg "$YELLOW" "   Note: Azure Event Grid requires HTTPS in production"
    exit 1
fi

print_header "Azure Event Grid Setup for Listmonk"

print_msg "$BLUE" "Webhook Endpoint: $WEBHOOK_ENDPOINT"
if [ -n "$RESOURCE_GROUP" ]; then
    print_msg "$BLUE" "Resource Group Filter: $RESOURCE_GROUP"
else
    print_msg "$BLUE" "Resource Group Filter: (all resource groups)"
fi

# Check if Azure CLI is installed
if ! command -v az &> /dev/null; then
    print_msg "$RED" "❌ Error: Azure CLI is not installed"
    echo ""
    echo "Install Azure CLI:"
    echo "  https://docs.microsoft.com/en-us/cli/azure/install-azure-cli"
    echo ""
    exit 1
fi

# Check if logged in
if ! az account show &> /dev/null; then
    print_msg "$RED" "❌ Error: Not logged in to Azure CLI"
    echo ""
    echo "Please run: az login"
    echo ""
    exit 1
fi

# Get current subscription
SUBSCRIPTION_ID=$(az account show --query id -o tsv)
SUBSCRIPTION_NAME=$(az account show --query name -o tsv)

print_msg "$GREEN" "✓ Logged in to Azure"
print_msg "$BLUE" "  Subscription: $SUBSCRIPTION_NAME"
print_msg "$BLUE" "  Subscription ID: $SUBSCRIPTION_ID"

# Check if Communication extension is installed
if ! az extension list --query "[?name=='communication'].name" -o tsv | grep -q communication; then
    print_msg "$YELLOW" "⚠️  Azure Communication CLI extension not found, installing..."
    az extension add --name communication
    print_msg "$GREEN" "✓ Azure Communication extension installed"
fi

print_header "Finding Azure Communication Services Resources"

# Build the list command
LIST_CMD="az communication list --subscription $SUBSCRIPTION_ID"
if [ -n "$RESOURCE_GROUP" ]; then
    LIST_CMD="$LIST_CMD --resource-group $RESOURCE_GROUP"
fi

# Get all Azure Communication Services resources
print_msg "$BLUE" "Searching for Azure Communication Services..."

ACS_RESOURCES=$(eval "$LIST_CMD --query \"[].{name:name, id:id, resourceGroup:resourceGroup}\" --output json")

if [ "$ACS_RESOURCES" == "[]" ] || [ -z "$ACS_RESOURCES" ]; then
    print_msg "$RED" "❌ No Azure Communication Services resources found"
    if [ -n "$RESOURCE_GROUP" ]; then
        echo "   in resource group: $RESOURCE_GROUP"
    fi
    echo ""
    echo "Please ensure you have:"
    echo "  1. Azure Communication Services resources provisioned"
    echo "  2. Appropriate permissions to list resources"
    if [ -n "$RESOURCE_GROUP" ]; then
        echo "  3. Correct resource group name specified"
    fi
    echo ""
    exit 1
fi

TOTAL_RESOURCES=$(echo "$ACS_RESOURCES" | jq length)
print_msg "$GREEN" "✓ Found $TOTAL_RESOURCES Azure Communication Services resource(s)"

print_header "Creating Event Grid Subscriptions"

# Process each resource
echo "$ACS_RESOURCES" | jq -c '.[]' | while read -r resource; do
    resource_name=$(echo "$resource" | jq -r '.name')
    resource_id=$(echo "$resource" | jq -r '.id')
    resource_group=$(echo "$resource" | jq -r '.resourceGroup')

    subscription_name="listmonk-email-events-${resource_name}"

    echo ""
    print_msg "$BLUE" "Processing: $resource_name"
    print_msg "$BLUE" "  Resource Group: $resource_group"
    print_msg "$BLUE" "  Subscription Name: $subscription_name"

    # Check if subscription already exists
    existing_sub=$(az eventgrid event-subscription list \
        --source-resource-id "$resource_id" \
        --query "[?name=='$subscription_name'].name" \
        -o tsv 2>/dev/null || true)

    if [ -n "$existing_sub" ]; then
        print_msg "$YELLOW" "  ⚠️  Event Grid subscription already exists, skipping"
        ((SKIP_COUNT++))
        continue
    fi

    # Create the event subscription
    print_msg "$BLUE" "  Creating Event Grid subscription..."

    if az eventgrid event-subscription create \
        --name "$subscription_name" \
        --source-resource-id "$resource_id" \
        --endpoint "$WEBHOOK_ENDPOINT" \
        --endpoint-type webhook \
        --included-event-types \
            Microsoft.Communication.EmailDeliveryReportReceived \
            Microsoft.Communication.EmailEngagementTrackingReportReceived \
        --event-delivery-schema eventgridschema \
        --max-delivery-attempts 30 \
        --event-ttl 1440 \
        --labels "managed-by=listmonk" "purpose=email-tracking" "created=$(date +%Y-%m-%d)" \
        --output none 2>&1; then

        print_msg "$GREEN" "  ✓ Successfully created Event Grid subscription"
        ((SUCCESS_COUNT++))
    else
        print_msg "$RED" "  ✗ Failed to create Event Grid subscription"
        print_msg "$YELLOW" "    This might be due to permissions or quota limits"
        ((FAIL_COUNT++))
    fi
done

print_header "Summary"

print_msg "$BLUE" "Total Resources Found: $TOTAL_RESOURCES"
print_msg "$GREEN" "Successfully Created: $SUCCESS_COUNT"
if [ $SKIP_COUNT -gt 0 ]; then
    print_msg "$YELLOW" "Skipped (already exists): $SKIP_COUNT"
fi
if [ $FAIL_COUNT -gt 0 ]; then
    print_msg "$RED" "Failed: $FAIL_COUNT"
fi

echo ""

if [ $SUCCESS_COUNT -gt 0 ]; then
    print_header "Next Steps"
    echo "1. Verify webhook endpoint is accessible:"
    print_msg "$BLUE" "   curl -I $WEBHOOK_ENDPOINT"
    echo ""
    echo "2. Check Event Grid subscriptions in Azure Portal:"
    print_msg "$BLUE" "   Portal > Event Grid Subscriptions"
    echo ""
    echo "3. Enable Azure Event Grid in listmonk:"
    print_msg "$BLUE" "   Settings > Bounces > Enable Azure Event Grid"
    echo ""
    echo "4. Verify bounce.azure.enabled = true in database:"
    print_msg "$BLUE" "   UPDATE settings SET value = '{\"enabled\": true}' WHERE key = 'bounce.azure';"
    echo ""
    echo "5. Monitor listmonk logs for validation events:"
    print_msg "$BLUE" "   tail -f /var/log/listmonk/listmonk.log"
    echo ""

    print_msg "$GREEN" "✓ Azure Event Grid setup complete!"
else
    print_msg "$YELLOW" "⚠️  No new subscriptions were created"
    if [ $SKIP_COUNT -gt 0 ]; then
        echo ""
        echo "All subscriptions already exist. To recreate them:"
        echo "1. Delete existing subscriptions in Azure Portal"
        echo "2. Run this script again"
    fi
fi

echo ""
