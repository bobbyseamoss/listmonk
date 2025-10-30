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
echo -e "${YELLOW}[1/4] Building Docker image with multi-stage build...${NC}"
echo -e "  - Building email-builder..."
echo -e "  - Building frontend..."
echo -e "  - Building Go binary..."
docker build -f Dockerfile.build -t ${FULL_IMAGE_NAME} .
echo -e "${GREEN}✓ Docker image built with all components${NC}"
echo ""

# Step 2: Push to Azure Container Registry
echo -e "${YELLOW}[2/4] Pushing image to Azure Container Registry...${NC}"
docker push ${FULL_IMAGE_NAME}
echo -e "${GREEN}✓ Image pushed to ACR${NC}"
echo ""

# Step 3: Deploy to Azure Container Apps
echo -e "${YELLOW}[3/4] Deploying to Azure Container Apps...${NC}"
az containerapp update \
  --name ${CONTAINER_APP_NAME} \
  --resource-group ${RESOURCE_GROUP} \
  --set-env-vars "DEPLOY_TIME=$(date +%s)"
echo -e "${GREEN}✓ Deployed to Azure${NC}"
echo ""

# Step 4: Get deployment info
echo -e "${YELLOW}[4/4] Getting deployment information...${NC}"
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
