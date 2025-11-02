#!/bin/bash
##
## Azure Email Communication Services - Enable User Engagement Tracking
##
## This script enables user engagement tracking (open tracking, click tracking)
## for all provisioned domains across all Email Communication Services in the subscription.
##
## Prerequisites:
## - Azure CLI installed (version 2.67.0 or higher)
## - Azure Communication extension installed (auto-installs on first use)
## - Logged in to Azure (az login)
## - Domains must NOT be Azure Managed Domains
## - Domains must have upgraded sending limits (not default limits)
##
## Usage:
##   ./enable_email_tracking.sh [--resource-group RG_NAME] [--dry-run]
##
## Options:
##   --resource-group RG_NAME    Only process Email Services in this resource group
##   --dry-run                   Show what would be done without making changes
##

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Parse command line arguments
RESOURCE_GROUP=""
DRY_RUN=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --resource-group)
            RESOURCE_GROUP="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            echo "Usage: $0 [--resource-group RG_NAME] [--dry-run]"
            exit 1
            ;;
    esac
done

echo "======================================================================"
echo "Azure Email Communication Services - User Engagement Tracking"
echo "======================================================================"
echo ""

# Check if Azure CLI is installed
if ! command -v az &> /dev/null; then
    echo -e "${RED}❌ Error: Azure CLI is not installed${NC}"
    echo "Install from: https://docs.microsoft.com/en-us/cli/azure/install-azure-cli"
    exit 1
fi

# Check Azure CLI version
AZ_VERSION=$(az version --query '"azure-cli"' -o tsv)
echo "Azure CLI version: $AZ_VERSION"

# Check if logged in to Azure
echo "Checking Azure login status..."
if ! az account show &> /dev/null; then
    echo -e "${RED}❌ Error: Not logged in to Azure${NC}"
    echo "Please run: az login"
    exit 1
fi

ACCOUNT_NAME=$(az account show --query name -o tsv)
SUBSCRIPTION_ID=$(az account show --query id -o tsv)
echo -e "${GREEN}✓${NC} Logged in as: $ACCOUNT_NAME"
echo -e "${GREEN}✓${NC} Subscription: $SUBSCRIPTION_ID"
echo ""

# Check if communication extension is available
echo "Checking Azure Communication Services extension..."
if ! az extension list --query "[?name=='communication'].version" -o tsv &> /dev/null; then
    echo -e "${YELLOW}Installing communication extension...${NC}"
    az extension add --name communication --only-show-errors
    echo -e "${GREEN}✓${NC} Extension installed"
else
    COMM_VERSION=$(az extension list --query "[?name=='communication'].version" -o tsv)
    echo -e "${GREEN}✓${NC} Extension version: $COMM_VERSION"
fi
echo ""

if [ "$DRY_RUN" = true ]; then
    echo -e "${YELLOW}=== DRY RUN MODE - No changes will be made ===${NC}"
    echo ""
fi

# Step 1: List all Email Communication Services
echo "======================================================================"
echo "Step 1: Discovering Email Communication Services"
echo "======================================================================"
echo ""

LIST_PARAMS=()
if [ -n "$RESOURCE_GROUP" ]; then
    echo "Filtering by resource group: $RESOURCE_GROUP"
    LIST_PARAMS+=(--resource-group "$RESOURCE_GROUP")
fi

echo "Fetching Email Communication Services..."
EMAIL_SERVICES=$(az communication email list "${LIST_PARAMS[@]}" --output json 2>/dev/null || echo "[]")

SERVICE_COUNT=$(echo "$EMAIL_SERVICES" | jq -r 'length')

if [ "$SERVICE_COUNT" -eq 0 ]; then
    echo -e "${YELLOW}No Email Communication Services found${NC}"
    if [ -n "$RESOURCE_GROUP" ]; then
        echo "Try running without --resource-group to search all resource groups"
    fi
    exit 0
fi

echo -e "${GREEN}✓${NC} Found $SERVICE_COUNT Email Communication Service(s)"
echo ""

# Display services
echo "Services:"
echo "$EMAIL_SERVICES" | jq -r '.[] | "  - \(.name) (Resource Group: \(.resourceGroup))"'
echo ""

# Step 2: Process each Email Service
echo "======================================================================"
echo "Step 2: Processing Domains"
echo "======================================================================"
echo ""

TOTAL_DOMAINS=0
ENABLED_COUNT=0
ALREADY_ENABLED_COUNT=0
FAILED_COUNT=0
SKIPPED_COUNT=0

declare -a FAILED_DOMAINS

# Process each email service
for row in $(echo "$EMAIL_SERVICES" | jq -r '.[] | @base64'); do
    _jq() {
        echo "$row" | base64 --decode | jq -r "$1"
    }

    SERVICE_NAME=$(_jq '.name')
    SERVICE_RG=$(_jq '.resourceGroup')

    echo -e "${CYAN}Processing Email Service: $SERVICE_NAME${NC}"
    echo "  Resource Group: $SERVICE_RG"

    # List domains for this service
    DOMAINS=$(az communication email domain list \
        --email-service-name "$SERVICE_NAME" \
        --resource-group "$SERVICE_RG" \
        --output json 2>/dev/null || echo "[]")

    DOMAIN_COUNT=$(echo "$DOMAINS" | jq -r 'length')

    if [ "$DOMAIN_COUNT" -eq 0 ]; then
        echo -e "  ${YELLOW}No domains found${NC}"
        echo ""
        continue
    fi

    echo "  Found $DOMAIN_COUNT domain(s)"
    echo ""

    # Process each domain
    for domain_row in $(echo "$DOMAINS" | jq -r '.[] | @base64'); do
        _jq_domain() {
            echo "$domain_row" | base64 --decode | jq -r "$1"
        }

        DOMAIN_NAME=$(_jq_domain '.name')
        DOMAIN_MANAGEMENT=$(_jq_domain '.properties.domainManagement // "Unknown"')
        CURRENT_TRACKING=$(_jq_domain '.properties.userEngagementTracking // "Unknown"')

        TOTAL_DOMAINS=$((TOTAL_DOMAINS + 1))

        echo -e "  ${BLUE}Domain: $DOMAIN_NAME${NC}"
        echo "    Management Type: $DOMAIN_MANAGEMENT"
        echo "    Current Tracking Status: $CURRENT_TRACKING"

        # Check if Azure Managed Domain
        if [ "$DOMAIN_MANAGEMENT" = "AzureManaged" ]; then
            echo -e "    ${YELLOW}⚠️  SKIPPED - Azure Managed Domains do not support user engagement tracking${NC}"
            SKIPPED_COUNT=$((SKIPPED_COUNT + 1))
            echo ""
            continue
        fi

        # Check if already enabled
        if [ "$CURRENT_TRACKING" = "Enabled" ]; then
            echo -e "    ${GREEN}✓ Already enabled${NC}"
            ALREADY_ENABLED_COUNT=$((ALREADY_ENABLED_COUNT + 1))
            echo ""
            continue
        fi

        # Enable tracking
        if [ "$DRY_RUN" = true ]; then
            echo -e "    ${YELLOW}[DRY RUN] Would enable user engagement tracking${NC}"
            ENABLED_COUNT=$((ENABLED_COUNT + 1))
        else
            echo -n "    Enabling user engagement tracking... "

            if az communication email domain update \
                --domain-name "$DOMAIN_NAME" \
                --email-service-name "$SERVICE_NAME" \
                --resource-group "$SERVICE_RG" \
                --user-engmnt-tracking "Enabled" \
                --output none 2>&1; then

                echo -e "${GREEN}✓ SUCCESS${NC}"
                ENABLED_COUNT=$((ENABLED_COUNT + 1))
            else
                echo -e "${RED}✗ FAILED${NC}"
                FAILED_COUNT=$((FAILED_COUNT + 1))
                FAILED_DOMAINS+=("$DOMAIN_NAME (Service: $SERVICE_NAME, RG: $SERVICE_RG)")

                # Common failure reason
                echo -e "    ${RED}Note: Domains with default sending limits cannot enable tracking.${NC}"
                echo -e "    ${RED}Raise a support ticket to upgrade your service limits.${NC}"
            fi
        fi

        echo ""
    done
done

# Summary
echo "======================================================================"
echo "Summary"
echo "======================================================================"
echo ""
echo "Total Email Services: $SERVICE_COUNT"
echo "Total Domains Processed: $TOTAL_DOMAINS"
echo ""

if [ "$DRY_RUN" = true ]; then
    echo -e "${YELLOW}[DRY RUN MODE]${NC}"
    echo -e "Would enable tracking on: ${YELLOW}$ENABLED_COUNT${NC} domain(s)"
else
    echo -e "${GREEN}✓ Enabled: $ENABLED_COUNT${NC}"
fi

echo -e "${CYAN}○ Already Enabled: $ALREADY_ENABLED_COUNT${NC}"

if [ $SKIPPED_COUNT -gt 0 ]; then
    echo -e "${YELLOW}⚠ Skipped (Azure Managed): $SKIPPED_COUNT${NC}"
fi

if [ $FAILED_COUNT -gt 0 ]; then
    echo -e "${RED}✗ Failed: $FAILED_COUNT${NC}"
    echo ""
    echo "Failed domains:"
    for domain in "${FAILED_DOMAINS[@]}"; do
        echo -e "  ${RED}- $domain${NC}"
    done
fi

echo ""

# Additional information
if [ $FAILED_COUNT -gt 0 ] || [ $SKIPPED_COUNT -gt 0 ]; then
    echo "======================================================================"
    echo "Important Notes"
    echo "======================================================================"
    echo ""

    if [ $SKIPPED_COUNT -gt 0 ]; then
        echo -e "${YELLOW}Azure Managed Domains:${NC}"
        echo "  User engagement tracking is not available for Azure Managed Domains"
        echo "  (e.g., xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.azurecomm.net)"
        echo "  Use custom domains instead."
        echo ""
    fi

    if [ $FAILED_COUNT -gt 0 ]; then
        echo -e "${RED}Default Sending Limits:${NC}"
        echo "  Domains with default sending limits cannot enable user engagement tracking."
        echo "  To enable tracking:"
        echo "  1. Open a support ticket in Azure Portal"
        echo "  2. Request to upgrade from default sending limits"
        echo "  3. Once approved, run this script again"
        echo ""
    fi
fi

echo "======================================================================"
echo "User Engagement Tracking Features"
echo "======================================================================"
echo ""
echo "When enabled, the following tracking is available:"
echo "  - Email opens (pixel-based tracking)"
echo "  - Link clicks (URL redirect tracking)"
echo ""
echo "To view engagement data:"
echo "  1. Azure Portal > Email Communication Service > Insights"
echo "  2. Or subscribe to Email User Engagement operational logs"
echo "  3. Or use Azure Monitor to query engagement events"
echo ""
echo "Note: Tracking only works for HTML emails, not plain text."
echo ""

exit 0
