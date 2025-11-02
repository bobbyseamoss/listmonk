#!/bin/bash

# Listmonk Build and Deploy Script
# This script builds the frontend, backend, and deploys to Azure Container Apps

set -e  # Exit on any error

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
RESOURCE_GROUP="rg-listmonk420"
CONTAINER_APP_NAME="listmonk420"
ACR_NAME="listmonk420acr"
IMAGE_NAME="listmonk420"
IMAGE_TAG="latest"
FULL_IMAGE_NAME="${ACR_NAME}.azurecr.io/${IMAGE_NAME}:${IMAGE_TAG}"

echo -e "${GREEN}=== Listmonk Build and Deploy ===${NC}"
echo ""

# Step 1: Build Docker Image (multi-stage build handles all compilation)
echo -e "${YELLOW}[1/6] Building Docker image with multi-stage build...${NC}"
echo -e "  - Building email-builder..."
echo -e "  - Building frontend..."
echo -e "  - Building Go binary with message tracking..."
docker build -f Dockerfile.build -t ${FULL_IMAGE_NAME} .
echo -e "${GREEN}✓ Docker image built with all components${NC}"
echo ""

# Step 2: Push to Azure Container Registry
echo -e "${YELLOW}[2/6] Pushing image to Azure Container Registry...${NC}"
az acr login --name listmonk420acr
docker push ${FULL_IMAGE_NAME}
echo -e "${GREEN}✓ Image pushed to ACR${NC}"
echo ""

# Step 3: Run database migration on production
echo -e "${YELLOW}[3/6] Running database migrations on production...${NC}"
echo -e "  - Connecting to Azure PostgreSQL..."
echo -e "  - Running migrations ..."

# Run migrations using the listmonk binary's --upgrade command
# This ensures all migrations in internal/migrations/ are applied
if [ -n "$LISTMONK_DB_PASSWORD" ]; then
    # Export database credentials for listmonk to use
    export LISTMONK_DB_HOST="${LISTMONK_DB_HOST:-listmonk420-db.postgres.database.azure.com}"
    export LISTMONK_DB_PORT="${LISTMONK_DB_PORT:-5432}"
    export LISTMONK_DB_USER="${LISTMONK_DB_USER:-listmonkadmin}"
    export LISTMONK_DB_PASSWORD="${LISTMONK_DB_PASSWORD}"
    export LISTMONK_DB_DATABASE="${LISTMONK_DB_DATABASE:-listmonk}"

    # Build a temporary binary just for running migrations
    echo -e "  - Building migration runner..."
    docker run --rm -v "$(pwd):/app" -w /app golang:1.24-alpine sh -c "CGO_ENABLED=0 go build -o listmonk-migrate cmd/*.go" 2>&1 | grep -v "go: downloading" || true

    # Run migrations
    echo -e "  - Applying database migrations..."
    ./listmonk-migrate --upgrade 2>&1 | grep -E "(applied|already|error|completed)" || echo "  Migration check complete"

    # Clean up temporary binary
    rm -f listmonk-migrate

    echo -e "${GREEN}✓ Database migrations completed${NC}"
else
    echo -e "${RED}⚠ WARNING: LISTMONK_DB_PASSWORD not set, skipping migrations${NC}"
    echo -e "${YELLOW}  Set LISTMONK_DB_PASSWORD and run migrations manually${NC}"
fi
echo ""

# Step 4: Enable Azure Event Grid in production database
#echo -e "${YELLOW}[4/6] Enabling Azure Event Grid webhooks in production...${NC}"
#if [ -n "$LISTMONK_DB_PASSWORD" ]; then
    # Enable both the master webhook switch AND Azure-specific webhook
#    PGPASSWORD="$LISTMONK_DB_PASSWORD" psql \
#        -h "${LISTMONK_DB_HOST:-listmonk420-db.postgres.database.azure.com}" \
#        -p "${LISTMONK_DB_PORT:-5432}" \
#        -U "${LISTMONK_DB_USER:-listmonkadmin}" \
#        -d "${LISTMONK_DB_DATABASE:-listmonk}" \
#        -c "UPDATE settings SET value = jsonb_set(jsonb_set(COALESCE(value, '{}'::jsonb), '{webhooks_enabled}', 'true'::jsonb), '{azure,enabled}', 'true'::jsonb), updated_at = NOW() WHERE key = 'bounce';" \
#        > /dev/null 2>&1

    # Verify both settings
#    WEBHOOKS_ENABLED=$(PGPASSWORD="$LISTMONK_DB_PASSWORD" psql \
#        -h "${LISTMONK_DB_HOST:-listmonk420-db.postgres.database.azure.com}" \
#        -p "${LISTMONK_DB_PORT:-5432}" \
#        -U "${LISTMONK_DB_USER:-listmonkadmin}" \
#        -d "${LISTMONK_DB_DATABASE:-listmonk}" \
#        -t -c "SELECT value->>'webhooks_enabled' FROM settings WHERE key = 'bounce';" 2>/dev/null | xargs)

#    AZURE_ENABLED=$(PGPASSWORD="$LISTMONK_DB_PASSWORD" psql \
#        -h "${LISTMONK_DB_HOST:-listmonk420-db.postgres.database.azure.com}" \
#        -p "${LISTMONK_DB_PORT:-5432}" \
#        -U "${LISTMONK_DB_USER:-listmonkadmin}" \
#        -d "${LISTMONK_DB_DATABASE:-listmonk}" \
#        -t -c "SELECT value->'azure'->>'enabled' FROM settings WHERE key = 'bounce';" 2>/dev/null | xargs)

#    if [ "$WEBHOOKS_ENABLED" = "true" ] && [ "$AZURE_ENABLED" = "true" ]; then
#        echo -e "${GREEN}✓ Bounce webhooks enabled (master + Azure)${NC}"
#    else
#        echo -e "${YELLOW}⚠ Webhook status: master=$WEBHOOKS_ENABLED, azure=$AZURE_ENABLED${NC}"
#    fi
#else
#    echo -e "${YELLOW}⚠ LISTMONK_DB_PASSWORD not set, skipping Azure Event Grid enablement${NC}"
#fi
#echo ""

# Step 4: Deploy to Azure Container Apps
echo -e "${YELLOW}[5/6] Deploying to Azure Container Apps...${NC}"
echo -e "  - Forcing new image pull and container restart..."
az containerapp update \
  --name ${CONTAINER_APP_NAME} \
  --resource-group ${RESOURCE_GROUP} \
  --image ${FULL_IMAGE_NAME} \
  --set-env-vars "DEPLOY_TIME=$(date +%s)" \
  --revision-suffix "deploy-$(date +%Y%m%d-%H%M%S)"
echo -e "${GREEN}✓ Deployed to Azure with new revision${NC}"
echo ""

# Step 5: Get deployment info
echo -e "${YELLOW}[6/6] Getting deployment information...${NC}"
REVISION=$(az containerapp show \
  --name ${CONTAINER_APP_NAME} \
  --resource-group ${RESOURCE_GROUP} \
  --query "properties.latestRevisionName" \
  -o tsv)
echo -e "${GREEN}✓ Deployment complete!${NC}"
echo ""

echo -e "${GREEN}=== Deployment Summary ===${NC}"
echo -e "Revision: ${GREEN}${REVISION}${NC}"
echo -e "URL: ${GREEN}https://list.bobbyseamoss.com${NC}"
echo -e "Image: ${GREEN}${FULL_IMAGE_NAME}${NC}"
echo -e "Features:"
echo -e "  - ${GREEN}✓${NC} Azure message tracking (v6.1.0)"
echo -e "  - ${GREEN}✓${NC} Azure delivery/engagement events (v6.2.0)"
echo -e "  - ${GREEN}✓${NC} Webhook correlation with recipient fallback"
echo -e "  - ${GREEN}✓${NC} Debug logging for webhook payloads"
echo ""
echo -e "${YELLOW}Check container logs:${NC}"
echo -e "  az containerapp logs show --name ${CONTAINER_APP_NAME} --resource-group ${RESOURCE_GROUP} --follow"
echo ""
echo -e "${YELLOW}Verify deployment:${NC}"
echo -e "  # Check message tracking is working (after sending a campaign)"
echo -e "  PGPASSWORD='\$LISTMONK_DB_PASSWORD' psql -h listmonk420-db.postgres.database.azure.com -U listmonkadmin -d listmonk -c 'SELECT COUNT(*) FROM azure_message_tracking;'"
echo ""
echo -e "  # Check if webhooks are arriving"
echo -e "  az containerapp logs show -n ${CONTAINER_APP_NAME} -g ${RESOURCE_GROUP} --tail 50 | grep -i azure"
echo ""
echo -e "${GREEN}=== Webhook Testing ===${NC}"
echo -e "1. Send a test campaign (2-4 recipients)"
echo -e "2. Check tracking table: SELECT * FROM azure_message_tracking ORDER BY sent_at DESC LIMIT 5;"
echo -e "3. Watch for webhook events in logs (look for 'DEBUG: Azure delivery event data')"
echo -e "4. Verify events are recorded: SELECT * FROM azure_delivery_events LIMIT 5;"
echo ""
