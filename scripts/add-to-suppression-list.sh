#!/bin/bash
#
# Add email addresses to Azure Communication Services domain suppression list
#
# Usage:
#   ./add-to-suppression-list.sh <email-service> <resource-group> <domain> <suppression-list-name> <csv-file>
#   ./add-to-suppression-list.sh <email-service> <resource-group> <domain> <suppression-list-name> --email <email>
#
# Examples:
#   # Add single email
#   ./add-to-suppression-list.sh Bobby-Seamoss-Mail mail.bobbyseamoss.com mail2.bobbyseamoss.com donotreply --email user@example.com
#
#   # Add from CSV file (one email per line, or with columns: email,first_name,last_name,notes)
#   ./add-to-suppression-list.sh Bobby-Seamoss-Mail mail.bobbyseamoss.com mail2.bobbyseamoss.com donotreply emails.csv
#

# Don't exit on error - we handle errors ourselves
set +e

API_VERSION="2023-06-01-preview"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

error() {
    echo -e "${RED}Error: $1${NC}" >&2
}

success() {
    echo -e "${GREEN}$1${NC}"
}

warn() {
    echo -e "${YELLOW}$1${NC}"
}

if [ "$#" -lt 5 ]; then
    echo "Usage: $0 <email-service> <resource-group> <domain> <suppression-list-name> <csv-file|--email email>"
    echo ""
    echo "Examples:"
    echo "  $0 Bobby-Seamoss-Mail mail.bobbyseamoss.com mail2.bobbyseamoss.com donotreply --email user@example.com"
    echo "  $0 Bobby-Seamoss-Mail mail.bobbyseamoss.com mail2.bobbyseamoss.com donotreply emails.csv"
    exit 1
fi

EMAIL_SERVICE="$1"
RESOURCE_GROUP="$2"
DOMAIN="$3"
SUPPRESSION_LIST="$4"
INPUT="$5"

# Get subscription ID
echo "Authenticating with Azure..."
SUBSCRIPTION_ID=$(az account show --query id -o tsv 2>&1)
if [ $? -ne 0 ]; then
    error "Failed to get subscription ID. Are you logged in to Azure CLI?"
    echo "Run: az login"
    exit 1
fi
echo "Using subscription: $SUBSCRIPTION_ID"

# Verify the suppression list exists
echo "Verifying suppression list exists..."
LIST_CHECK_URL="https://management.azure.com/subscriptions/${SUBSCRIPTION_ID}/resourceGroups/${RESOURCE_GROUP}/providers/Microsoft.Communication/emailServices/${EMAIL_SERVICE}/domains/${DOMAIN}/suppressionLists/${SUPPRESSION_LIST}?api-version=${API_VERSION}"

list_response=$(az rest --method GET --url "$LIST_CHECK_URL" 2>&1)
list_exit_code=$?

if [ $list_exit_code -ne 0 ]; then
    error "Suppression list '$SUPPRESSION_LIST' not found or access denied"
    echo ""
    echo "Details: $list_response"
    echo ""
    echo "Available suppression lists for $DOMAIN:"
    az rest --method GET \
        --url "https://management.azure.com/subscriptions/${SUBSCRIPTION_ID}/resourceGroups/${RESOURCE_GROUP}/providers/Microsoft.Communication/emailServices/${EMAIL_SERVICE}/domains/${DOMAIN}/suppressionLists?api-version=${API_VERSION}" \
        2>/dev/null | jq -r '.value[].name' 2>/dev/null || echo "  (unable to list)"
    exit 1
fi
success "Suppression list '$SUPPRESSION_LIST' found"

BASE_URL="https://management.azure.com/subscriptions/${SUBSCRIPTION_ID}/resourceGroups/${RESOURCE_GROUP}/providers/Microsoft.Communication/emailServices/${EMAIL_SERVICE}/domains/${DOMAIN}/suppressionLists/${SUPPRESSION_LIST}/suppressionListAddresses"

# Track failed emails for summary
declare -a FAILED_EMAILS=()

add_email() {
    local email="$1"
    local first_name="${2:-}"
    local last_name="${3:-}"
    local notes="${4:-}"

    # Validate email format (basic check)
    if [[ ! "$email" =~ ^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$ ]]; then
        echo -e "Adding $email... ${YELLOW}⚠ Skipped (invalid email format)${NC}"
        return 2
    fi

    # Generate a unique address ID (use email hash for idempotency)
    local address_id=$(echo -n "$email" | md5sum | cut -c1-32)

    local url="${BASE_URL}/${address_id}?api-version=${API_VERSION}"

    # Build JSON body
    local body=$(jq -n \
        --arg email "$email" \
        --arg first "$first_name" \
        --arg last "$last_name" \
        --arg notes "$notes" \
        '{
            properties: {
                email: $email,
                firstName: (if $first == "" then null else $first end),
                lastName: (if $last == "" then null else $last end),
                notes: (if $notes == "" then null else $notes end)
            }
        }')

    echo -n "Adding $email... "

    # Capture both stdout and stderr
    local response
    local exit_code
    response=$(az rest --method PUT --url "$url" --body "$body" 2>&1)
    exit_code=$?

    if [ $exit_code -eq 0 ]; then
        success "✓"
        return 0
    else
        echo -e "${RED}✗ Failed${NC}"

        # Parse error message if possible
        local error_msg=$(echo "$response" | jq -r '.error.message // empty' 2>/dev/null)
        if [ -n "$error_msg" ]; then
            echo -e "  ${RED}→ $error_msg${NC}"
        else
            # Show raw error if can't parse JSON
            local short_error=$(echo "$response" | head -c 200)
            echo -e "  ${RED}→ $short_error${NC}"
        fi

        FAILED_EMAILS+=("$email: ${error_msg:-$response}")
        return 1
    fi
}

# Check if adding single email
if [ "$INPUT" == "--email" ]; then
    if [ -z "$6" ]; then
        error "--email requires an email address"
        exit 1
    fi
    echo ""
    add_email "$6"
    exit_code=$?
    if [ $exit_code -eq 0 ]; then
        exit 0
    elif [ $exit_code -eq 2 ]; then
        exit 1
    else
        exit 1
    fi
fi

# Process CSV file
if [ ! -f "$INPUT" ]; then
    error "File not found: $INPUT"
    exit 1
fi

# Count lines in file (excluding header)
line_count=$(wc -l < "$INPUT")
echo ""
echo "Adding emails from $INPUT to suppression list '$SUPPRESSION_LIST' on domain '$DOMAIN'..."
echo "File contains approximately $line_count lines"
echo ""

total=0
success_count=0
failed=0
skipped=0

while IFS=, read -r email first_name last_name notes || [ -n "$email" ]; do
    # Skip header row
    if [ "$email" == "email" ] || [ "$email" == "Email" ] || [ "$email" == "EMAIL" ]; then
        continue
    fi

    # Skip empty lines
    [ -z "$email" ] && continue

    # Skip lines that start with # (comments)
    [[ "$email" =~ ^# ]] && continue

    # Trim whitespace and remove quotes
    email=$(echo "$email" | xargs | tr -d '"' | tr -d "'")
    first_name=$(echo "$first_name" | xargs | tr -d '"' | tr -d "'" 2>/dev/null || echo "")
    last_name=$(echo "$last_name" | xargs | tr -d '"' | tr -d "'" 2>/dev/null || echo "")
    notes=$(echo "$notes" | xargs | tr -d '"' | tr -d "'" 2>/dev/null || echo "")

    ((total++)) || true

    add_email "$email" "$first_name" "$last_name" "$notes"
    result=$?

    if [ $result -eq 0 ]; then
        ((success_count++)) || true
    elif [ $result -eq 2 ]; then
        ((skipped++)) || true
    else
        ((failed++)) || true
    fi
done < "$INPUT"

echo ""
echo "========================================"
echo "Summary"
echo "========================================"
echo -e "Total processed: $total"
echo -e "${GREEN}Success:         $success_count${NC}"
echo -e "${YELLOW}Skipped:         $skipped${NC}"
echo -e "${RED}Failed:          $failed${NC}"

if [ ${#FAILED_EMAILS[@]} -gt 0 ]; then
    echo ""
    echo "Failed emails:"
    for fail in "${FAILED_EMAILS[@]}"; do
        echo -e "  ${RED}• $fail${NC}"
    done
fi

# Exit with error if any failed
if [ $failed -gt 0 ]; then
    exit 1
fi
exit 0
