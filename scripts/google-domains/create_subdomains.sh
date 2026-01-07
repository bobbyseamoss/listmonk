#!/bin/bash
# ========================================================================
# Create Google Workspace Subdomains and Users
# ========================================================================
# This script:
#   1. Adds 30 subdomains (mail1-mail30.bobbyseamoss.com) to Google Workspace
#   2. Creates adam@mail{N}.bobbyseamoss.com user for each
#   3. Outputs DNS verification records
# ========================================================================

GAM="/home/adam/bin/gamadv-xtd3/gam"
BASE_DOMAIN="bobbyseamoss.com"
PASSWORD="ChangeMe123!"  # Change this or generate random passwords

echo "=========================================================================="
echo "Creating Google Workspace Subdomains"
echo "=========================================================================="
echo ""

# Arrays to track results
declare -a VERIFICATION_RECORDS

# Step 1: Add subdomains
echo "Step 1: Adding subdomains to Google Workspace..."
echo ""

for i in $(seq 1 30); do
    SUBDOMAIN="mail${i}.${BASE_DOMAIN}"
    echo -n "  Adding ${SUBDOMAIN}... "

    # Add subdomain
    RESULT=$($GAM create domain ${SUBDOMAIN} 2>&1)

    if echo "$RESULT" | grep -q "already exists"; then
        echo "Already exists"
    elif echo "$RESULT" | grep -q "Created"; then
        echo "Created"

        # Get verification record
        VERIFY=$($GAM info domain ${SUBDOMAIN} 2>&1 | grep -A1 "Verification")
        if [ -n "$VERIFY" ]; then
            VERIFICATION_RECORDS+=("${SUBDOMAIN}: ${VERIFY}")
        fi
    else
        echo "Failed: $RESULT"
    fi
done

echo ""
echo "Step 2: Creating users..."
echo ""

for i in $(seq 1 30); do
    SUBDOMAIN="mail${i}.${BASE_DOMAIN}"
    EMAIL="adam@${SUBDOMAIN}"
    echo -n "  Creating ${EMAIL}... "

    # Create user
    RESULT=$($GAM create user ${EMAIL} firstname Adam lastname Schwartz password "${PASSWORD}" 2>&1)

    if echo "$RESULT" | grep -q "already exists"; then
        echo "Already exists"
    elif echo "$RESULT" | grep -q "Created"; then
        echo "Created"
    else
        echo "Failed: $RESULT"
    fi
done

echo ""
echo "=========================================================================="
echo "DNS Verification Records"
echo "=========================================================================="
echo ""
echo "Add these TXT records to your DNS:"
echo ""

for record in "${VERIFICATION_RECORDS[@]}"; do
    echo "  $record"
done

echo ""
echo "=========================================================================="
echo "Complete!"
echo "=========================================================================="
echo ""
echo "Next steps:"
echo "  1. Add the DNS records from google_subdomain_dns.txt to your DNS provider"
echo "  2. Verify each domain in Google Admin Console"
echo "  3. Wait for DNS propagation (up to 48 hours)"
echo ""
echo "To list all domains:"
echo "  $GAM print domains"
echo ""
echo "To list all users:"
echo "  $GAM print users"
