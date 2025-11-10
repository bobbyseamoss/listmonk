#!/bin/bash
##
## Azure Email SMTP Credentials Script for comma-rg
## Retrieves SMTP credentials for all mail1-mail30.enjoycomma.com domains
##

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
RESOURCE_GROUP="comma-rg"
COMMUNICATION_SERVICE_NAME="comma-email-comm"
EMAIL_SERVICE_NAME="comma-email-service"
BASE_DOMAIN="enjoycomma.com"
START_NUM=1
END_NUM=30

echo -e "${GREEN}======================================================================"
echo "Azure Email SMTP Credentials for comma-rg"
echo "======================================================================${NC}"
echo ""

# Get Communication Service connection string (contains access keys)
echo -e "${YELLOW}Getting Communication Service connection string...${NC}"
CONN_STRING=$(az communication list-key \
    --name $COMMUNICATION_SERVICE_NAME \
    --resource-group $RESOURCE_GROUP \
    --query "primaryConnectionString" \
    -o tsv)

# Extract endpoint from connection string
ENDPOINT=$(echo "$CONN_STRING" | grep -oP 'endpoint=\K[^;]+')
ACCESS_KEY=$(echo "$CONN_STRING" | grep -oP 'accesskey=\K[^;]+')

echo -e "${GREEN}✓ Connection string retrieved${NC}"
echo ""

# Output file
OUTPUT_FILE="/tmp/comma-smtp-credentials.txt"
> $OUTPUT_FILE

echo -e "${YELLOW}Extracting SMTP credentials for all domains...${NC}"
echo ""

# SMTP Configuration (same for all Azure Communication Services)
SMTP_HOST="smtp.azurecomm.net"
SMTP_PORT="587"
SMTP_SECURITY="STARTTLS"

echo "======================================================================" | tee -a $OUTPUT_FILE
echo "Azure Email Communication Services - SMTP Configuration" | tee -a $OUTPUT_FILE
echo "Resource Group: comma-rg" | tee -a $OUTPUT_FILE
echo "Generated: $(date)" | tee -a $OUTPUT_FILE
echo "======================================================================" | tee -a $OUTPUT_FILE
echo "" | tee -a $OUTPUT_FILE

for i in $(seq $START_NUM $END_NUM); do
    DOMAIN_NAME="mail${i}.${BASE_DOMAIN}"
    DOMAIN_RESOURCE_NAME="mail${i}-enjoycomma-com"

    # Get domain verification status
    DOMAIN_STATUS=$(az communication email domain show \
        --name $DOMAIN_RESOURCE_NAME \
        --email-service-name $EMAIL_SERVICE_NAME \
        --resource-group $RESOURCE_GROUP \
        --query "verificationState.domain.status" \
        -o tsv 2>/dev/null || echo "NotFound")

    echo "-------------------------------------------------------------------" | tee -a $OUTPUT_FILE
    echo "Domain: $DOMAIN_NAME" | tee -a $OUTPUT_FILE
    echo "-------------------------------------------------------------------" | tee -a $OUTPUT_FILE

    if [ "$DOMAIN_STATUS" = "Verified" ]; then
        echo -e "${GREEN}Status: VERIFIED ✓${NC}" | tee -a $OUTPUT_FILE
        echo "" | tee -a $OUTPUT_FILE
        echo "SMTP Configuration:" | tee -a $OUTPUT_FILE
        echo "  Host: $SMTP_HOST" | tee -a $OUTPUT_FILE
        echo "  Port: $SMTP_PORT" | tee -a $OUTPUT_FILE
        echo "  Security: $SMTP_SECURITY" | tee -a $OUTPUT_FILE
        echo "  Username: $COMMUNICATION_SERVICE_NAME" | tee -a $OUTPUT_FILE
        echo "  Password: $ACCESS_KEY" | tee -a $OUTPUT_FILE
        echo "  From Address: noreply@$DOMAIN_NAME" | tee -a $OUTPUT_FILE
        echo "" | tee -a $OUTPUT_FILE

        # Listmonk SMTP configuration
        echo "Listmonk Configuration:" | tee -a $OUTPUT_FILE
        echo "  Name: mail${i}" | tee -a $OUTPUT_FILE
        echo "  Host: $SMTP_HOST" | tee -a $OUTPUT_FILE
        echo "  Port: $SMTP_PORT" | tee -a $OUTPUT_FILE
        echo "  Auth Protocol: login" | tee -a $OUTPUT_FILE
        echo "  Username: $COMMUNICATION_SERVICE_NAME" | tee -a $OUTPUT_FILE
        echo "  Password: [same access key as above]" | tee -a $OUTPUT_FILE
        echo "  From Email: adam@$DOMAIN_NAME" | tee -a $OUTPUT_FILE
        echo "  TLS: STARTTLS (587)" | tee -a $OUTPUT_FILE
        echo "  Max Connections: 10" | tee -a $OUTPUT_FILE
        echo "  Daily Limit: 1000 (adjust as needed)" | tee -a $OUTPUT_FILE
        echo "" | tee -a $OUTPUT_FILE
    elif [ "$DOMAIN_STATUS" = "NotFound" ]; then
        echo -e "${RED}Status: NOT FOUND ✗${NC}" | tee -a $OUTPUT_FILE
        echo "" | tee -a $OUTPUT_FILE
    else
        echo -e "${YELLOW}Status: NOT VERIFIED (${DOMAIN_STATUS})${NC}" | tee -a $OUTPUT_FILE
        echo "  Run ./verify-comma-email-domains.sh to check verification status" | tee -a $OUTPUT_FILE
        echo "" | tee -a $OUTPUT_FILE
    fi
done

echo "======================================================================" | tee -a $OUTPUT_FILE
echo "" | tee -a $OUTPUT_FILE
echo "IMPORTANT NOTES:" | tee -a $OUTPUT_FILE
echo "1. All domains use the SAME username and password (Communication Service credentials)" | tee -a $OUTPUT_FILE
echo "2. The 'From Address' can be any valid address at the verified domain" | tee -a $OUTPUT_FILE
echo "3. Azure Communication Services enforces sender domain matching" | tee -a $OUTPUT_FILE
echo "4. SMTP credentials saved to: $OUTPUT_FILE" | tee -a $OUTPUT_FILE
echo "" | tee -a $OUTPUT_FILE

# Create SQL insert script for listmonk_comma database
SQL_FILE="/tmp/comma-smtp-insert.sql"
> $SQL_FILE

echo "-- SMTP Server Configuration for listmonk_comma database" > $SQL_FILE
echo "-- Generated: $(date)" >> $SQL_FILE
echo "-- Run with: PGPASSWORD='T@intshr3dd3r' psql -h listmonk420-db.postgres.database.azure.com -U listmonkadmin -d listmonk_comma -f $SQL_FILE" >> $SQL_FILE
echo "" >> $SQL_FILE

# Get current settings
CURRENT_SMTP=$(PGPASSWORD='T@intshr3dd3r' psql \
    -h listmonk420-db.postgres.database.azure.com \
    -U listmonkadmin \
    -d listmonk_comma \
    --set=sslmode=require \
    -t -c "SELECT value FROM settings WHERE key = 'smtp';" 2>/dev/null | xargs || echo "[]")

# Build SMTP array JSON
SMTP_JSON="["

for i in $(seq $START_NUM $END_NUM); do
    DOMAIN_NAME="mail${i}.${BASE_DOMAIN}"
    DOMAIN_RESOURCE_NAME="mail${i}-enjoycomma-com"

    DOMAIN_STATUS=$(az communication email domain show \
        --name $DOMAIN_RESOURCE_NAME \
        --email-service-name $EMAIL_SERVICE_NAME \
        --resource-group $RESOURCE_GROUP \
        --query "verificationState.domain.status" \
        -o tsv 2>/dev/null || echo "NotFound")

    if [ "$DOMAIN_STATUS" = "Verified" ]; then
        if [ $i -gt $START_NUM ]; then
            SMTP_JSON="$SMTP_JSON,"
        fi

        SMTP_JSON="$SMTP_JSON
  {
    \"uuid\": \"$(uuidgen | tr '[:upper:]' '[:lower:]')\",
    \"enabled\": true,
    \"name\": \"mail${i}\",
    \"host\": \"$SMTP_HOST\",
    \"port\": $SMTP_PORT,
    \"auth_protocol\": \"login\",
    \"username\": \"$COMMUNICATION_SERVICE_NAME\",
    \"password\": \"$ACCESS_KEY\",
    \"from_email\": \"adam@$DOMAIN_NAME\",
    \"hello_hostname\": \"\",
    \"tls_enabled\": true,
    \"tls_skip_verify\": false,
    \"max_conns\": 10,
    \"max_msg_retries\": 2,
    \"idle_timeout\": \"15s\",
    \"wait_timeout\": \"5s\",
    \"max_msg_rate\": 0,
    \"daily_limit\": 1000,
    \"email_headers\": [],
    \"bounce_mailbox_uuid\": null
  }"
    fi
done

SMTP_JSON="$SMTP_JSON
]"

# Create SQL UPDATE statement
cat >> $SQL_FILE <<EOF

-- Update SMTP settings
UPDATE settings
SET value = '$SMTP_JSON'::jsonb
WHERE key = 'smtp';

-- Verify the update
SELECT key, jsonb_array_length(value) as smtp_server_count
FROM settings
WHERE key = 'smtp';

EOF

echo -e "${GREEN}======================================================================"
echo "SMTP Credentials Summary"
echo "======================================================================${NC}"
echo -e "Credentials saved to: ${GREEN}$OUTPUT_FILE${NC}"
echo -e "SQL script saved to: ${GREEN}$SQL_FILE${NC}"
echo ""
echo -e "${YELLOW}Next Steps:${NC}"
echo "1. Review credentials: cat $OUTPUT_FILE"
echo "2. Import SMTP settings to database: "
echo "   PGPASSWORD='T@intshr3dd3r' psql -h listmonk420-db.postgres.database.azure.com -U listmonkadmin -d listmonk_comma -f $SQL_FILE"
echo "3. Restart listmonk-comma container app"
echo "4. Test sending from each SMTP server in listmonk admin"
echo ""
