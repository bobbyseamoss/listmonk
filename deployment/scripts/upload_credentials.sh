#!/bin/bash
##
## Azure Key Vault Credential Upload Script
## Uploads SMTP and bounce mailbox credentials from credential_map.json to Azure Key Vault
##

set -e  # Exit on error

# Configuration
RESOURCE_GROUP="rg-listmonk420"
KEY_VAULT_NAME="listmonk420-kv"
LOCATION="eastus"
CREDENTIAL_FILE="$(dirname "$0")/../configs/credential_map.json"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "======================================================================"
echo "Azure Key Vault Credential Upload"
echo "======================================================================"
echo ""

# Check if Azure CLI is installed
if ! command -v az &> /dev/null; then
    echo -e "${RED}❌ Error: Azure CLI is not installed${NC}"
    echo "Install from: https://docs.microsoft.com/en-us/cli/azure/install-azure-cli"
    exit 1
fi

# Check if logged in to Azure
echo "Checking Azure login status..."
if ! az account show &> /dev/null; then
    echo -e "${RED}❌ Error: Not logged in to Azure${NC}"
    echo "Please run: az login"
    exit 1
fi

ACCOUNT_NAME=$(az account show --query name -o tsv)
echo -e "${GREEN}✓${NC} Logged in as: $ACCOUNT_NAME"
echo ""

# Check if credential file exists
if [ ! -f "$CREDENTIAL_FILE" ]; then
    echo -e "${RED}❌ Error: Credential file not found: $CREDENTIAL_FILE${NC}"
    echo "Please run export_configs.py first to generate the credential map."
    exit 1
fi

echo "Credential file: $CREDENTIAL_FILE"
CREDENTIAL_COUNT=$(jq 'length' "$CREDENTIAL_FILE")
echo "Found $CREDENTIAL_COUNT credentials to upload"
echo ""

# Check if Key Vault exists, create if not
echo "Checking Key Vault: $KEY_VAULT_NAME..."
if az keyvault show --name "$KEY_VAULT_NAME" --resource-group "$RESOURCE_GROUP" &> /dev/null; then
    echo -e "${GREEN}✓${NC} Key Vault exists"
else
    echo -e "${YELLOW}Creating Key Vault: $KEY_VAULT_NAME${NC}"
    az keyvault create \
        --name "$KEY_VAULT_NAME" \
        --resource-group "$RESOURCE_GROUP" \
        --location "$LOCATION" \
        --enable-rbac-authorization false \
        --enabled-for-template-deployment true \
        --enabled-for-deployment true \
        --enabled-for-disk-encryption false

    echo -e "${GREEN}✓${NC} Key Vault created"
fi
echo ""

# Upload credentials
echo "Uploading credentials to Key Vault..."
echo ""

UPLOADED=0
FAILED=0
SKIPPED=0

while IFS="=" read -r secret_name secret_value; do
    # Remove quotes from secret_name and secret_value
    secret_name=$(echo "$secret_name" | tr -d '"')
    secret_value=$(echo "$secret_value" | tr -d '"')

    # Skip if empty
    if [ -z "$secret_name" ] || [ -z "$secret_value" ]; then
        continue
    fi

    echo -n "  Uploading $secret_name... "

    # Check if secret already exists
    if az keyvault secret show --vault-name "$KEY_VAULT_NAME" --name "$secret_name" &> /dev/null; then
        echo -e "${YELLOW}EXISTS (skipped)${NC}"
        ((SKIPPED++))
    else
        # Upload secret
        if az keyvault secret set \
            --vault-name "$KEY_VAULT_NAME" \
            --name "$secret_name" \
            --value "$secret_value" \
            --output none 2>&1; then
            echo -e "${GREEN}✓${NC}"
            ((UPLOADED++))
        else
            echo -e "${RED}FAILED${NC}"
            ((FAILED++))
        fi
    fi

done < <(jq -r 'to_entries | .[] | "\(.key)=\(.value)"' "$CREDENTIAL_FILE")

echo ""
echo "======================================================================"
echo "Upload Summary"
echo "======================================================================"
echo -e "${GREEN}✓ Uploaded: $UPLOADED${NC}"
if [ $SKIPPED -gt 0 ]; then
    echo -e "${YELLOW}○ Skipped (already exists): $SKIPPED${NC}"
fi
if [ $FAILED -gt 0 ]; then
    echo -e "${RED}✗ Failed: $FAILED${NC}"
fi
echo ""

# Security reminder
echo "======================================================================"
echo "Security Reminder"
echo "======================================================================"
echo -e "${YELLOW}⚠️  Important: Delete the credential_map.json file after upload!${NC}"
echo ""
echo "Run: rm $CREDENTIAL_FILE"
echo ""

# Verify access
echo "======================================================================"
echo "Key Vault Access Configuration"
echo "======================================================================"
echo ""
echo "To allow Container Apps to access these secrets:"
echo ""
echo "1. Enable managed identity on your Container App:"
echo "   az containerapp identity assign \\"
echo "     --name listmonk420 \\"
echo "     --resource-group $RESOURCE_GROUP \\"
echo "     --system-assigned"
echo ""
echo "2. Grant Key Vault access to the managed identity:"
echo "   IDENTITY_ID=\$(az containerapp show \\"
echo "     --name listmonk420 \\"
echo "     --resource-group $RESOURCE_GROUP \\"
echo "     --query identity.principalId -o tsv)"
echo ""
echo "   az keyvault set-policy \\"
echo "     --name $KEY_VAULT_NAME \\"
echo "     --object-id \$IDENTITY_ID \\"
echo "     --secret-permissions get list"
echo ""
echo "3. Reference secrets in Container App:"
echo "   az containerapp secret set \\"
echo "     --name listmonk420 \\"
echo "     --resource-group $RESOURCE_GROUP \\"
echo "     --secrets smtp-1-username=keyvaultref:https://$KEY_VAULT_NAME.vault.azure.net/secrets/smtp-1-username,identityref:system"
echo ""

exit 0
