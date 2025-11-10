#!/bin/bash
##
## Azure Email Domain Verification Script for comma-rg
## Checks verification status of all mail1-mail30.enjoycomma.com domains
##

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
RESOURCE_GROUP="comma-rg"
EMAIL_SERVICE_NAME="comma-email-service"
BASE_DOMAIN="enjoycomma.com"
START_NUM=1
END_NUM=30

echo -e "${GREEN}======================================================================"
echo "Azure Email Domain Verification Status for comma-rg"
echo "======================================================================${NC}"
echo ""

VERIFIED_COUNT=0
PENDING_COUNT=0
FAILED_COUNT=0

for i in $(seq $START_NUM $END_NUM); do
    DOMAIN_NAME="mail${i}.${BASE_DOMAIN}"
    DOMAIN_RESOURCE_NAME="mail${i}-enjoycomma-com"

    echo -e "${YELLOW}Checking: $DOMAIN_NAME${NC}"

    # Get domain status
    DOMAIN_INFO=$(az communication email domain show \
        --name $DOMAIN_RESOURCE_NAME \
        --email-service-name $EMAIL_SERVICE_NAME \
        --resource-group $RESOURCE_GROUP \
        --query "{status: verificationState.domain.status, spf: verificationState.spf.status, dkim: verificationState.dkim.status, dkim2: verificationState.dkim2.status}" \
        -o json 2>/dev/null || echo "null")

    if [ "$DOMAIN_INFO" = "null" ]; then
        echo -e "${RED}  ✗ Domain not found${NC}"
        ((FAILED_COUNT++))
        continue
    fi

    # Parse verification status
    DOMAIN_STATUS=$(echo "$DOMAIN_INFO" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('status', 'Unknown'))")
    SPF_STATUS=$(echo "$DOMAIN_INFO" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('spf', 'Unknown'))")
    DKIM_STATUS=$(echo "$DOMAIN_INFO" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('dkim', 'Unknown'))")
    DKIM2_STATUS=$(echo "$DOMAIN_INFO" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('dkim2', 'Unknown'))")

    # Display status
    if [ "$DOMAIN_STATUS" = "Verified" ] && [ "$SPF_STATUS" = "Verified" ] && [ "$DKIM_STATUS" = "Verified" ] && [ "$DKIM2_STATUS" = "Verified" ]; then
        echo -e "${GREEN}  ✓ VERIFIED - All records verified${NC}"
        ((VERIFIED_COUNT++))
    elif [ "$DOMAIN_STATUS" = "NotStarted" ] || [ "$SPF_STATUS" = "NotStarted" ] || [ "$DKIM_STATUS" = "NotStarted" ] || [ "$DKIM2_STATUS" = "NotStarted" ]; then
        echo -e "${YELLOW}  ⏳ PENDING - Awaiting DNS propagation${NC}"
        echo -e "     Domain: $DOMAIN_STATUS | SPF: $SPF_STATUS | DKIM: $DKIM_STATUS | DKIM2: $DKIM2_STATUS"
        ((PENDING_COUNT++))
    else
        echo -e "${RED}  ⚠ PARTIAL - Some records pending${NC}"
        echo -e "     Domain: $DOMAIN_STATUS | SPF: $SPF_STATUS | DKIM: $DKIM_STATUS | DKIM2: $DKIM2_STATUS"
        ((PENDING_COUNT++))
    fi
    echo ""
done

echo -e "${GREEN}======================================================================"
echo "Verification Summary"
echo "======================================================================${NC}"
echo -e "Total Domains: $((END_NUM - START_NUM + 1))"
echo -e "${GREEN}Verified: $VERIFIED_COUNT${NC}"
echo -e "${YELLOW}Pending: $PENDING_COUNT${NC}"
echo -e "${RED}Failed: $FAILED_COUNT${NC}"
echo ""

if [ $VERIFIED_COUNT -eq $((END_NUM - START_NUM + 1)) ]; then
    echo -e "${GREEN}✓ All domains verified! Ready to get SMTP credentials.${NC}"
    echo -e "${GREEN}  Run: ./get-comma-smtp-credentials.sh${NC}"
elif [ $PENDING_COUNT -gt 0 ]; then
    echo -e "${YELLOW}⏳ DNS propagation in progress.${NC}"
    echo -e "${YELLOW}  Check DNS records: /tmp/comma-dns-records.txt${NC}"
    echo -e "${YELLOW}  DNS propagation can take up to 48 hours${NC}"
    echo -e "${YELLOW}  Re-run this script to check status again${NC}"
fi
echo ""
