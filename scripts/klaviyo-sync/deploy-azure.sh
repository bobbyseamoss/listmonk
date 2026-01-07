#!/bin/bash
# Deploy Klaviyo Sync as Azure Container App Job
#
# Prerequisites:
# 1. Azure CLI installed and logged in (az login)
# 2. Docker installed
# 3. KLAVIYO_API_KEY environment variable set
#
# Usage: ./deploy-azure.sh

set -e

# Configuration - matches your existing listmonk setup
RESOURCE_GROUP="rg-listmonk420"
LOCATION="eastus2"
ACR_NAME="listmonk420acr"
JOB_NAME="klaviyo-sync"
ENVIRONMENT="listmonk420-env"
IMAGE_NAME="klaviyo-sync"
IMAGE_TAG="latest"

# Database connection (same as listmonk)
DB_HOST="listmonk420-db.postgres.database.azure.com"
DB_PORT="5432"
DB_NAME="listmonk"
DB_USER="listmonkadmin"
DB_PASSWORD="T@intshr3dd3r"

# Check for Klaviyo API key
if [ -z "$KLAVIYO_API_KEY" ]; then
    echo "Error: KLAVIYO_API_KEY environment variable is required"
    echo "Get your API key from: Klaviyo > Settings > API Keys"
    exit 1
fi

echo "=== Klaviyo Sync Azure Deployment ==="
echo "Resource Group: $RESOURCE_GROUP"
echo "Location: $LOCATION"
echo "Job Name: $JOB_NAME"
echo ""

# Step 1: Create Azure Container Registry if it doesn't exist
echo "Step 1: Checking Azure Container Registry..."
if ! az acr show --name $ACR_NAME --resource-group $RESOURCE_GROUP &>/dev/null; then
    echo "Creating Azure Container Registry: $ACR_NAME"
    az acr create \
        --resource-group $RESOURCE_GROUP \
        --name $ACR_NAME \
        --sku Basic \
        --admin-enabled true
else
    echo "ACR $ACR_NAME already exists"
fi

# Get ACR credentials
ACR_LOGIN_SERVER=$(az acr show --name $ACR_NAME --query loginServer -o tsv)
ACR_USERNAME=$(az acr credential show --name $ACR_NAME --query username -o tsv)
ACR_PASSWORD=$(az acr credential show --name $ACR_NAME --query passwords[0].value -o tsv)

# Step 2: Build and push Docker image
echo ""
echo "Step 2: Building and pushing Docker image..."
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Build the image
docker build -t $ACR_LOGIN_SERVER/$IMAGE_NAME:$IMAGE_TAG .

# Login to ACR
echo $ACR_PASSWORD | docker login $ACR_LOGIN_SERVER -u $ACR_USERNAME --password-stdin

# Push the image
docker push $ACR_LOGIN_SERVER/$IMAGE_NAME:$IMAGE_TAG

# Step 3: Get or create Container Apps environment
echo ""
echo "Step 3: Checking Container Apps environment..."
if ! az containerapp env show --name $ENVIRONMENT --resource-group $RESOURCE_GROUP &>/dev/null; then
    echo "Creating Container Apps environment: $ENVIRONMENT"
    az containerapp env create \
        --name $ENVIRONMENT \
        --resource-group $RESOURCE_GROUP \
        --location $LOCATION
else
    echo "Environment $ENVIRONMENT already exists"
fi

# Step 4: Create or update the Container App Job
echo ""
echo "Step 4: Creating/updating Container App Job..."

# Check if job exists
if az containerapp job show --name $JOB_NAME --resource-group $RESOURCE_GROUP &>/dev/null; then
    echo "Updating existing job: $JOB_NAME"
    az containerapp job update \
        --name $JOB_NAME \
        --resource-group $RESOURCE_GROUP \
        --image "$ACR_LOGIN_SERVER/$IMAGE_NAME:$IMAGE_TAG" \
        --set-env-vars \
            "KLAVIYO_API_KEY=$KLAVIYO_API_KEY" \
            "DB_HOST=$DB_HOST" \
            "DB_PORT=$DB_PORT" \
            "DB_NAME=$DB_NAME" \
            "DB_USER=$DB_USER" \
            "DB_PASSWORD=$DB_PASSWORD" \
            "DB_SSLMODE=require" \
            "LIST_PREFIX=Klaviyo: " \
            "DRY_RUN=false"
else
    echo "Creating new job: $JOB_NAME"
    az containerapp job create \
        --name $JOB_NAME \
        --resource-group $RESOURCE_GROUP \
        --environment $ENVIRONMENT \
        --trigger-type "Schedule" \
        --cron-expression "0 * * * *" \
        --replica-timeout 3600 \
        --replica-retry-limit 1 \
        --parallelism 1 \
        --replica-completion-count 1 \
        --image "$ACR_LOGIN_SERVER/$IMAGE_NAME:$IMAGE_TAG" \
        --cpu 0.5 \
        --memory 1Gi \
        --registry-server $ACR_LOGIN_SERVER \
        --registry-username $ACR_USERNAME \
        --registry-password $ACR_PASSWORD \
        --env-vars \
            "KLAVIYO_API_KEY=$KLAVIYO_API_KEY" \
            "DB_HOST=$DB_HOST" \
            "DB_PORT=$DB_PORT" \
            "DB_NAME=$DB_NAME" \
            "DB_USER=$DB_USER" \
            "DB_PASSWORD=$DB_PASSWORD" \
            "DB_SSLMODE=require" \
            "LIST_PREFIX=Klaviyo: " \
            "DRY_RUN=false"
fi

echo ""
echo "=== Deployment Complete ==="
echo ""
echo "Job Details:"
az containerapp job show --name $JOB_NAME --resource-group $RESOURCE_GROUP \
    --query "{name:name, status:properties.provisioningState, schedule:properties.configuration.scheduleTriggerConfig.cronExpression}" \
    -o table

echo ""
echo "To manually trigger the job:"
echo "  az containerapp job start --name $JOB_NAME --resource-group $RESOURCE_GROUP"
echo ""
echo "To view job execution history:"
echo "  az containerapp job execution list --name $JOB_NAME --resource-group $RESOURCE_GROUP -o table"
echo ""
echo "To view logs:"
echo "  az containerapp job logs show --name $JOB_NAME --resource-group $RESOURCE_GROUP"
