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

# Step 1: Build Email Builder
echo -e "${YELLOW}[1/7] Building email-builder component...${NC}"
cd frontend/email-builder
npm install
npm run build
cd ../..

# Copy email-builder to frontend public directory
echo -e "${YELLOW}[1/7] Copying email-builder to frontend public directory...${NC}"
mkdir -p frontend/public/static/email-builder
cp -r frontend/email-builder/dist/* frontend/public/static/email-builder/
echo -e "${GREEN}✓ Email-builder built and copied${NC}"
echo ""

# Step 2: Build Frontend
echo -e "${YELLOW}[2/7] Building frontend...${NC}"
cd frontend
npm install
npm run build
cd ..
echo -e "${GREEN}✓ Frontend built${NC}"
echo ""

# Step 3: Build Go Binary
echo -e "${YELLOW}[3/7] Building Go binary...${NC}"
CGO_ENABLED=0 go build -o listmonk -ldflags="-s -w -X 'main.buildString=$(date -u +%Y-%m-%dT%H:%M:%S%z)' -X 'main.versionString=$(git describe --tags --always)'" ./cmd
echo -e "${GREEN}✓ Go binary built${NC}"
echo ""

# Step 4: Build Docker Image
echo -e "${YELLOW}[4/7] Building Docker image...${NC}"
docker build -t ${FULL_IMAGE_NAME} .
echo -e "${GREEN}✓ Docker image built${NC}"
echo ""

# Step 5: Push to Azure Container Registry
echo -e "${YELLOW}[5/7] Pushing image to Azure Container Registry...${NC}"
docker push ${FULL_IMAGE_NAME}
echo -e "${GREEN}✓ Image pushed to ACR${NC}"
echo ""

# Step 6: Deploy to Azure Container Apps
echo -e "${YELLOW}[6/9] Deploying to Azure Container Apps...${NC}"
az containerapp update \
  --name ${CONTAINER_APP_NAME} \
  --resource-group ${RESOURCE_GROUP} \
  --set-env-vars "DEPLOY_TIME=$(date +%s)"
echo -e "${GREEN}✓ Deployed to Azure${NC}"
echo ""

# Step 7: Check if this is first deployment
echo -e "${YELLOW}[7/9] Checking deployment status...${NC}"
FIRST_DEPLOY="false"

# Check if database has settings table (indicates initialization needed)
echo -e "Checking if database initialization is needed..."
if az containerapp exec \
  --name ${CONTAINER_APP_NAME} \
  --resource-group ${RESOURCE_GROUP} \
  --command "./listmonk --check-db" 2>&1 | grep -q "not initialized"; then
  FIRST_DEPLOY="true"
  echo -e "${YELLOW}⚠️  First deployment detected - initialization required${NC}"
else
  echo -e "${GREEN}✓ Database already initialized${NC}"
fi
echo ""

# Step 8: Configuration initialization (if first deploy)
if [ "$FIRST_DEPLOY" = "true" ]; then
  echo -e "${YELLOW}[8/9] First Deployment Setup${NC}"
  echo -e "${RED}=== ACTION REQUIRED ===${NC}"
  echo ""
  echo -e "This appears to be your first deployment. Complete these steps:"
  echo ""
  echo -e "${YELLOW}1. Upload credentials to Azure Key Vault:${NC}"
  echo -e "   cd deployment/scripts"
  echo -e "   ./upload_credentials.sh"
  echo ""
  echo -e "${YELLOW}2. Enable AUTO_INSTALL environment variable:${NC}"
  echo -e "   az containerapp update \\"
  echo -e "     --name ${CONTAINER_APP_NAME} \\"
  echo -e "     --resource-group ${RESOURCE_GROUP} \\"
  echo -e "     --set-env-vars \\"
  echo -e "       \"AUTO_INSTALL=true\" \\"
  echo -e "       \"LISTMONK_ADMIN_USER=admin\" \\"
  echo -e "       \"LISTMONK_ADMIN_PASSWORD=YOUR_SECURE_PASSWORD\""
  echo ""
  echo -e "${YELLOW}3. Configure Container App secrets from Key Vault${NC}"
  echo -e "   (See DEPLOYMENT_GUIDE.md for full instructions)"
  echo ""
  echo -e "${YELLOW}4. Restart the container to trigger initialization:${NC}"
  echo -e "   az containerapp revision restart \\"
  echo -e "     --name ${CONTAINER_APP_NAME} \\"
  echo -e "     --resource-group ${RESOURCE_GROUP} \\"
  echo -e "     --revision \$(az containerapp show --name ${CONTAINER_APP_NAME} --resource-group ${RESOURCE_GROUP} --query properties.latestRevisionName -o tsv)"
  echo ""
  read -p "Press Enter after completing these steps, or Ctrl+C to exit..."
  echo ""
else
  echo -e "${YELLOW}[8/9] Skipping initialization (already configured)${NC}"
  echo ""
fi

# Step 9: Get deployment info
echo -e "${YELLOW}[9/9] Getting deployment information...${NC}"
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
echo ""
echo -e "Check logs with:"
echo -e "  ${YELLOW}az containerapp logs show --name ${CONTAINER_APP_NAME} --resource-group ${RESOURCE_GROUP} --follow${NC}"
echo ""
