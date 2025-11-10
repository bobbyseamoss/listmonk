#!/bin/bash
##
## Azure Email Communication Services Setup for comma-rg
## Creates 30 Email Communication Services for mail1-mail30.enjoycomma.com
##

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
RESOURCE_GROUP="comma-rg"
LOCATION="eastus"
BASE_DOMAIN="enjoycomma.com"
COMMUNICATION_SERVICE_NAME="comma-email-comm"
EMAIL_SERVICE_NAME="comma-email-service"

# Number of mail servers to create
START_NUM=1
END_NUM=30

echo -e "${GREEN}======================================================================"
echo "Azure Email Communication Services Setup for comma-rg"
echo "======================================================================${NC}"
echo ""
echo "Resource Group: $RESOURCE_GROUP"
echo "Location: $LOCATION"
echo "Base Domain: $BASE_DOMAIN"
echo "Mail Servers: mail${START_NUM} through mail${END_NUM}"
echo ""

# Step 1: Create Communication Service
echo -e "${YELLOW}[1/5] Creating Azure Communication Service...${NC}"
if az communication list --resource-group $RESOURCE_GROUP --query "[?name=='$COMMUNICATION_SERVICE_NAME']" -o tsv | grep -q "$COMMUNICATION_SERVICE_NAME"; then
    echo "  ✓ Communication Service already exists: $COMMUNICATION_SERVICE_NAME"
else
    az communication create \
        --name $COMMUNICATION_SERVICE_NAME \
        --resource-group $RESOURCE_GROUP \
        --location "global" \
        --data-location "United States"
    echo -e "${GREEN}  ✓ Communication Service created: $COMMUNICATION_SERVICE_NAME${NC}"
fi
echo ""

# Step 2: Create Email Communication Service
echo -e "${YELLOW}[2/5] Creating Email Communication Service...${NC}"
if az communication email list --resource-group $RESOURCE_GROUP --query "[?name=='$EMAIL_SERVICE_NAME']" -o tsv | grep -q "$EMAIL_SERVICE_NAME"; then
    echo "  ✓ Email Service already exists: $EMAIL_SERVICE_NAME"
else
    az communication email create \
        --name $EMAIL_SERVICE_NAME \
        --resource-group $RESOURCE_GROUP \
        --location "global" \
        --data-location "United States"
    echo -e "${GREEN}  ✓ Email Service created: $EMAIL_SERVICE_NAME${NC}"
fi
echo ""

# Step 3: Create custom domains for mail1-mail30
echo -e "${YELLOW}[3/5] Creating custom email domains...${NC}"
for i in $(seq $START_NUM $END_NUM); do
    DOMAIN_NAME="mail${i}.${BASE_DOMAIN}"
    DOMAIN_RESOURCE_NAME="mail${i}-enjoycomma-com"

    echo "  Creating domain: $DOMAIN_NAME"

    # Check if domain already exists
    if az communication email domain list \
        --email-service-name $EMAIL_SERVICE_NAME \
        --resource-group $RESOURCE_GROUP \
        --query "[?name=='$DOMAIN_RESOURCE_NAME']" -o tsv | grep -q "$DOMAIN_RESOURCE_NAME"; then
        echo "    ✓ Domain already exists: $DOMAIN_NAME"
    else
        az communication email domain create \
            --name $DOMAIN_RESOURCE_NAME \
            --email-service-name $EMAIL_SERVICE_NAME \
            --resource-group $RESOURCE_GROUP \
            --location "global" \
            --domain-name $DOMAIN_NAME \
            --domain-management CustomerManaged \
            --user-engmnt-tracking Disabled

        echo -e "${GREEN}    ✓ Created domain: $DOMAIN_NAME${NC}"
    fi
done
echo ""

# Step 4: Get DNS verification records
echo -e "${YELLOW}[4/5] Getting DNS verification records...${NC}"
echo "  You need to add the following DNS records to verify domain ownership:"
echo ""

DNS_RECORDS_FILE="/tmp/comma-dns-records.txt"
> $DNS_RECORDS_FILE

for i in $(seq $START_NUM $END_NUM); do
    DOMAIN_NAME="mail${i}.${BASE_DOMAIN}"
    DOMAIN_RESOURCE_NAME="mail${i}-enjoycomma-com"

    echo "  Domain: $DOMAIN_NAME"

    # Get verification records
    VERIFICATION_RECORDS=$(az communication email domain show \
        --name $DOMAIN_RESOURCE_NAME \
        --email-service-name $EMAIL_SERVICE_NAME \
        --resource-group $RESOURCE_GROUP \
        --query "verificationRecords" -o json 2>/dev/null || echo "{}")

    if [ "$VERIFICATION_RECORDS" != "{}" ] && [ "$VERIFICATION_RECORDS" != "null" ]; then
        echo "$VERIFICATION_RECORDS" | python3 -c "
import sys, json
records = json.load(sys.stdin)
print(f'    TXT Record:')
if 'domain' in records and records['domain']:
    print(f'      Name: {records[\"domain\"].get(\"name\", \"N/A\")}')
    print(f'      Value: {records[\"domain\"].get(\"value\", \"N/A\")}')
if 'spf' in records and records['spf']:
    print(f'    SPF Record:')
    print(f'      Name: {records[\"spf\"].get(\"name\", \"N/A\")}')
    print(f'      Value: {records[\"spf\"].get(\"value\", \"N/A\")}')
if 'dkim' in records and records['dkim']:
    print(f'    DKIM Record:')
    print(f'      Name: {records[\"dkim\"].get(\"name\", \"N/A\")}')
    print(f'      Value: {records[\"dkim\"].get(\"value\", \"N/A\")}')
if 'dkim2' in records and records['dkim2']:
    print(f'    DKIM2 Record:')
    print(f'      Name: {records[\"dkim2\"].get(\"name\", \"N/A\")}')
    print(f'      Value: {records[\"dkim2\"].get(\"value\", \"N/A\")}')
"

        # Save to file for later reference
        echo "=== $DOMAIN_NAME ===" >> $DNS_RECORDS_FILE
        echo "$VERIFICATION_RECORDS" | python3 -m json.tool >> $DNS_RECORDS_FILE
        echo "" >> $DNS_RECORDS_FILE
    fi
    echo ""
done

echo -e "${GREEN}  ✓ DNS records saved to: $DNS_RECORDS_FILE${NC}"
echo ""

# Step 5: Link Communication Service to Email Service
echo -e "${YELLOW}[5/5] Linking Communication Service to Email Service...${NC}"
COMM_SERVICE_ID=$(az communication show \
    --name $COMMUNICATION_SERVICE_NAME \
    --resource-group $RESOURCE_GROUP \
    --query "id" -o tsv)

echo "  Communication Service ID: $COMM_SERVICE_ID"
echo -e "${GREEN}  ✓ Services configured${NC}"
echo ""

echo -e "${GREEN}======================================================================"
echo "Email Communication Services Setup Complete!"
echo "======================================================================${NC}"
echo ""
echo -e "${YELLOW}IMPORTANT NEXT STEPS:${NC}"
echo "1. Add DNS records from: $DNS_RECORDS_FILE"
echo "2. Wait for DNS propagation (can take up to 48 hours)"
echo "3. Verify domains using: ./verify-comma-email-domains.sh"
echo "4. Get SMTP credentials using: ./get-comma-smtp-credentials.sh"
echo "5. Configure listmonk_comma database with SMTP settings"
echo ""
