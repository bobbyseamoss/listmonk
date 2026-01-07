#!/bin/bash

# Shopify Webhook Setup Script for Listmonk
# This script registers webhooks using the Shopify Admin API

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration - Update these values
SHOP_DOMAIN="${SHOPIFY_SHOP_DOMAIN:-bobbyseamoss.myshopify.com}"
ACCESS_TOKEN="${SHOPIFY_ACCESS_TOKEN:-}"
LISTMONK_URL="${LISTMONK_URL:-https://list.bobbyseamoss.com}"
API_VERSION="2024-10"

# Validate required env vars
if [ -z "$ACCESS_TOKEN" ]; then
    echo -e "${RED}Error: SHOPIFY_ACCESS_TOKEN environment variable is required${NC}"
    echo "Usage: SHOPIFY_ACCESS_TOKEN=your_token ./setup-webhooks.sh"
    exit 1
fi

SHOPIFY_API="https://${SHOP_DOMAIN}/admin/api/${API_VERSION}"

echo -e "${GREEN}=== Shopify Webhook Setup for Listmonk ===${NC}"
echo -e "Shop: ${SHOP_DOMAIN}"
echo -e "Listmonk URL: ${LISTMONK_URL}"
echo ""

# Function to create webhook
create_webhook() {
    local topic=$1
    local address=$2

    echo -e "${YELLOW}Creating webhook: ${topic}${NC}"

    response=$(curl -s -X POST "${SHOPIFY_API}/webhooks.json" \
        -H "X-Shopify-Access-Token: ${ACCESS_TOKEN}" \
        -H "Content-Type: application/json" \
        -d "{
            \"webhook\": {
                \"topic\": \"${topic}\",
                \"address\": \"${address}\",
                \"format\": \"json\"
            }
        }")

    # Check for errors
    if echo "$response" | grep -q '"errors"'; then
        error=$(echo "$response" | jq -r '.errors // .error // "Unknown error"')
        echo -e "${RED}  Error: ${error}${NC}"
    else
        webhook_id=$(echo "$response" | jq -r '.webhook.id // "unknown"')
        echo -e "${GREEN}  Created webhook ID: ${webhook_id}${NC}"
    fi
}

# Function to list existing webhooks
list_webhooks() {
    echo -e "${YELLOW}Existing webhooks:${NC}"

    response=$(curl -s -X GET "${SHOPIFY_API}/webhooks.json" \
        -H "X-Shopify-Access-Token: ${ACCESS_TOKEN}")

    echo "$response" | jq -r '.webhooks[] | "  - \(.topic): \(.address) (ID: \(.id))"' 2>/dev/null || echo "  No webhooks found"
    echo ""
}

# Function to delete all webhooks
delete_all_webhooks() {
    echo -e "${YELLOW}Deleting existing webhooks...${NC}"

    response=$(curl -s -X GET "${SHOPIFY_API}/webhooks.json" \
        -H "X-Shopify-Access-Token: ${ACCESS_TOKEN}")

    webhook_ids=$(echo "$response" | jq -r '.webhooks[].id' 2>/dev/null)

    for id in $webhook_ids; do
        echo -e "  Deleting webhook ID: ${id}"
        curl -s -X DELETE "${SHOPIFY_API}/webhooks/${id}.json" \
            -H "X-Shopify-Access-Token: ${ACCESS_TOKEN}" > /dev/null
    done
    echo ""
}

# Check current access scopes
check_scopes() {
    echo -e "${YELLOW}Checking access scopes...${NC}"

    response=$(curl -s -X GET "${SHOPIFY_API}/access_scopes.json" \
        -H "X-Shopify-Access-Token: ${ACCESS_TOKEN}")

    scopes=$(echo "$response" | jq -r '.access_scopes[].handle' 2>/dev/null | tr '\n' ', ' | sed 's/,$//')
    echo -e "  Current scopes: ${GREEN}${scopes}${NC}"

    # Check for required scopes
    required=("read_customers" "read_orders" "read_products")
    missing=()

    for scope in "${required[@]}"; do
        if ! echo "$scopes" | grep -q "$scope"; then
            missing+=("$scope")
        fi
    done

    if [ ${#missing[@]} -gt 0 ]; then
        echo -e "${RED}  Missing required scopes: ${missing[*]}${NC}"
        echo -e "${YELLOW}  Please reinstall the app with updated permissions${NC}"
        return 1
    else
        echo -e "${GREEN}  All required scopes are present!${NC}"
    fi
    echo ""
}

# Main execution
case "${1:-setup}" in
    "list")
        list_webhooks
        ;;
    "delete")
        delete_all_webhooks
        ;;
    "scopes")
        check_scopes
        ;;
    "setup"|*)
        # Check scopes first
        check_scopes || exit 1

        # List existing webhooks
        list_webhooks

        # Ask to delete existing
        read -p "Delete existing webhooks before creating new ones? (y/N) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            delete_all_webhooks
        fi

        # Create webhooks
        echo -e "${GREEN}Creating webhooks...${NC}"
        create_webhook "orders/create" "${LISTMONK_URL}/webhooks/shopify/orders"
        create_webhook "customers/create" "${LISTMONK_URL}/webhooks/shopify/customers"
        create_webhook "customers/update" "${LISTMONK_URL}/webhooks/shopify/customers"

        echo ""
        echo -e "${GREEN}=== Setup Complete ===${NC}"
        echo ""
        list_webhooks

        echo -e "${YELLOW}Next steps:${NC}"
        echo "1. Copy the webhook signing secret from Shopify Admin > Settings > Notifications > Webhooks"
        echo "2. Update LISTMONK_SHOPIFY_WEBHOOK_SECRET in your listmonk configuration"
        echo "3. Test by creating a test order in Shopify"
        ;;
esac
