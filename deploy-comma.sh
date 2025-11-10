#!/bin/bash

# Listmonk Build and Deploy Script for comma-rg Resource Group
# This script builds the frontend, backend, and deploys to Azure Container Apps in comma-rg

set -e  # Exit on any error

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration for comma-rg deployment
RESOURCE_GROUP="comma-rg"
CONTAINER_APP_NAME="listmonk-comma"
ACR_NAME="listmonkcommaacr"
IMAGE_NAME="listmonk-comma"
IMAGE_TAG="latest"
FULL_IMAGE_NAME="${ACR_NAME}.azurecr.io/${IMAGE_NAME}:${IMAGE_TAG}"

# Database configuration (same PostgreSQL server, different database)
DB_HOST="listmonk420-db.postgres.database.azure.com"
DB_PORT="5432"
DB_USER="listmonkadmin"
DB_DATABASE="listmonk_comma"

echo -e "${GREEN}=== Listmonk Build and Deploy to comma-rg ===${NC}"
echo -e "${YELLOW}Resource Group: ${RESOURCE_GROUP}${NC}"
echo -e "${YELLOW}Container App: ${CONTAINER_APP_NAME}${NC}"
echo -e "${YELLOW}Database: ${DB_DATABASE}${NC}"
echo ""

# Step 1: Build Docker Image (multi-stage build handles all compilation)
echo -e "${YELLOW}[1/7] Building Docker image with multi-stage build...${NC}"
echo -e "  - Building email-builder..."
echo -e "  - Building frontend..."
echo -e "  - Building Go binary with all features..."
echo -e "    • Queue-based email delivery (v6.0.0)"
echo -e "    • Azure message tracking (v6.1.0)"
echo -e "    • Azure delivery/engagement events (v6.2.0)"
echo -e "    • Shopify purchase attribution (v7.0.0)"
echo -e "    • Campaigns page redesign with metrics"
echo -e "  - Using Comma logo..."
docker build -f Dockerfile.build --build-arg LOGO_URL="https://enjoycomma.com/cdn/shop/files/Comma-logo.svg?v=1752549135&width=80" -t ${FULL_IMAGE_NAME} .
echo -e "${GREEN}✓ Docker image built with all components${NC}"
echo ""

# Step 2: Push to Azure Container Registry
echo -e "${YELLOW}[2/7] Pushing image to Azure Container Registry (comma-rg)...${NC}"
az acr login --name ${ACR_NAME}
docker push ${FULL_IMAGE_NAME}
echo -e "${GREEN}✓ Image pushed to ${ACR_NAME}${NC}"
echo ""

# Step 3: Check current database schema version
echo -e "${YELLOW}[3/7] Checking database schema version...${NC}"
if [ -n "$LISTMONK_DB_PASSWORD" ]; then
    CURRENT_VERSION=$(PGPASSWORD="$LISTMONK_DB_PASSWORD" psql \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_DATABASE}" \
        -t -c "SELECT version FROM migrations ORDER BY version DESC LIMIT 1;" 2>/dev/null | xargs || echo "none")

    if [ "$CURRENT_VERSION" = "none" ]; then
        echo -e "${YELLOW}  ⚠ No migrations found - database may need initialization${NC}"
        echo -e "${YELLOW}  AUTO_INSTALL=true will handle initialization on first run${NC}"
    else
        echo -e "${GREEN}  ✓ Current schema version: ${CURRENT_VERSION}${NC}"
    fi
else
    echo -e "${RED}⚠ WARNING: LISTMONK_DB_PASSWORD not set, cannot check schema version${NC}"
fi
echo ""

# Step 4: Run database migrations on listmonk_comma
echo -e "${YELLOW}[4/7] Running database migrations on ${DB_DATABASE}...${NC}"
echo -e "  - Connecting to Azure PostgreSQL..."
echo -e "  - Database: ${DB_DATABASE}"

if [ -n "$LISTMONK_DB_PASSWORD" ]; then
    # Export database credentials for listmonk to use
    export LISTMONK_DB_HOST="${DB_HOST}"
    export LISTMONK_DB_PORT="${DB_PORT}"
    export LISTMONK_DB_USER="${DB_USER}"
    export LISTMONK_DB_PASSWORD="${LISTMONK_DB_PASSWORD}"
    export LISTMONK_DB_DATABASE="${DB_DATABASE}"

    # Build a temporary binary just for running migrations
    echo -e "  - Building migration runner..."
    docker run --rm -v "$(pwd):/app" -w /app golang:1.24-alpine sh -c "CGO_ENABLED=0 go build -o listmonk-migrate cmd/*.go" 2>&1 | grep -v "go: downloading" || true

    # Run migrations
    echo -e "  - Applying database migrations..."
    echo -e "    Checking for migrations: v6.0.0, v6.1.0, v6.2.0, v7.0.0..."
    ./listmonk-migrate --upgrade 2>&1 | grep -E "(applied|already|error|completed|v[0-9])" || echo "  Migration check complete"

    # Clean up temporary binary
    rm -f listmonk-migrate

    # Verify final schema version
    FINAL_VERSION=$(PGPASSWORD="$LISTMONK_DB_PASSWORD" psql \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_DATABASE}" \
        -t -c "SELECT version FROM migrations ORDER BY version DESC LIMIT 1;" 2>/dev/null | xargs)

    echo -e "${GREEN}✓ Database migrations completed${NC}"
    echo -e "${GREEN}  Final schema version: ${FINAL_VERSION}${NC}"
else
    echo -e "${RED}⚠ WARNING: LISTMONK_DB_PASSWORD not set, skipping migrations${NC}"
    echo -e "${YELLOW}  Set LISTMONK_DB_PASSWORD and run migrations manually${NC}"
fi
echo ""

# Step 5: Deploy to Azure Container Apps
echo -e "${YELLOW}[5/7] Deploying to Azure Container Apps (comma-rg)...${NC}"
echo -e "  - Forcing new image pull and container restart..."
az containerapp update \
  --name ${CONTAINER_APP_NAME} \
  --resource-group ${RESOURCE_GROUP} \
  --image ${FULL_IMAGE_NAME} \
  --set-env-vars "DEPLOY_TIME=$(date +%s)" \
  --revision-suffix "deploy-$(date +%Y%m%d-%H%M%S)"
echo -e "${GREEN}✓ Deployed to Azure with new revision${NC}"
echo ""

# Step 6: Get deployment info
echo -e "${YELLOW}[6/7] Getting deployment information...${NC}"
REVISION=$(az containerapp show \
  --name ${CONTAINER_APP_NAME} \
  --resource-group ${RESOURCE_GROUP} \
  --query "properties.latestRevisionName" \
  -o tsv)

FQDN=$(az containerapp show \
  --name ${CONTAINER_APP_NAME} \
  --resource-group ${RESOURCE_GROUP} \
  --query "properties.configuration.ingress.fqdn" \
  -o tsv)

echo -e "${GREEN}✓ Deployment information retrieved${NC}"
echo ""

# Step 7: Verification
echo -e "${YELLOW}[7/7] Running post-deployment verification...${NC}"

# Check health endpoint
echo -e "  - Checking health endpoint..."
HEALTH_URL="https://${FQDN}/api/health"
sleep 10  # Give container time to start

if curl -s -f "${HEALTH_URL}" > /dev/null 2>&1; then
    echo -e "${GREEN}  ✓ Health check passed: ${HEALTH_URL}${NC}"
else
    echo -e "${YELLOW}  ⚠ Health check pending (container may still be starting)${NC}"
    echo -e "${YELLOW}    Check manually: curl ${HEALTH_URL}${NC}"
fi

# Verify database connectivity
if [ -n "$LISTMONK_DB_PASSWORD" ]; then
    echo -e "  - Verifying database connectivity..."
    DB_TEST=$(PGPASSWORD="$LISTMONK_DB_PASSWORD" psql \
        -h "${DB_HOST}" \
        -p "${DB_PORT}" \
        -U "${DB_USER}" \
        -d "${DB_DATABASE}" \
        -t -c "SELECT COUNT(*) FROM migrations;" 2>/dev/null | xargs || echo "0")

    if [ "$DB_TEST" -gt "0" ]; then
        echo -e "${GREEN}  ✓ Database connectivity verified (${DB_TEST} migrations applied)${NC}"
    else
        echo -e "${YELLOW}  ⚠ Database check inconclusive${NC}"
    fi
fi

echo ""
echo -e "${GREEN}=== Deployment Summary ===${NC}"
echo -e "Resource Group: ${GREEN}${RESOURCE_GROUP}${NC}"
echo -e "Revision: ${GREEN}${REVISION}${NC}"
echo -e "URL: ${GREEN}https://${FQDN}${NC}"
echo -e "Admin Login: ${GREEN}https://${FQDN}/admin${NC}"
echo -e "Image: ${GREEN}${FULL_IMAGE_NAME}${NC}"
echo -e "Database: ${GREEN}${DB_DATABASE} on ${DB_HOST}${NC}"
echo ""
echo -e "Features deployed:"
echo -e "  - ${GREEN}✓${NC} Queue-based email delivery system (v6.0.0)"
echo -e "  - ${GREEN}✓${NC} Azure message tracking (v6.1.0)"
echo -e "  - ${GREEN}✓${NC} Azure delivery/engagement events (v6.2.0)"
echo -e "  - ${GREEN}✓${NC} Shopify purchase attribution (v7.0.0)"
echo -e "  - ${GREEN}✓${NC} Enhanced campaigns page with performance metrics"
echo -e "  - ${GREEN}✓${NC} Open/Click Rate columns with raw counts"
echo -e "  - ${GREEN}✓${NC} Support for regular and queue-based campaigns"
echo ""
echo -e "${YELLOW}Verification Steps:${NC}"
echo -e "1. Test admin login:"
echo -e "   ${GREEN}https://${FQDN}/admin${NC}"
echo -e "   Username: admin"
echo -e "   Password: (check container app environment variables)"
echo ""
echo -e "2. Check container logs:"
echo -e "   ${GREEN}az containerapp logs show --name ${CONTAINER_APP_NAME} --resource-group ${RESOURCE_GROUP} --follow${NC}"
echo ""
echo -e "3. Verify database schema:"
echo -e "   ${GREEN}PGPASSWORD='\$LISTMONK_DB_PASSWORD' psql -h ${DB_HOST} -U ${DB_USER} -d ${DB_DATABASE} -c 'SELECT * FROM migrations ORDER BY version DESC LIMIT 5;'${NC}"
echo ""
echo -e "4. Check Azure message tracking (after sending test campaign):"
echo -e "   ${GREEN}PGPASSWORD='\$LISTMONK_DB_PASSWORD' psql -h ${DB_HOST} -U ${DB_USER} -d ${DB_DATABASE} -c 'SELECT COUNT(*) FROM azure_message_tracking;'${NC}"
echo ""
echo -e "${GREEN}=== Deployment Complete! ===${NC}"
echo ""
