# Listmonk Deployment Guide

## Quick Deploy

To build and deploy both frontend and backend to Azure:

```bash
./deploy.sh
```

## What the Script Does

The `deploy.sh` script automates the complete build and deployment process:

1. **Build Email-Builder Component** - Compiles the visual email editor
2. **Build Frontend** - Compiles the Vue.js frontend with Vite
3. **Build Go Binary** - Compiles the listmonk backend
4. **Build Docker Image** - Creates the container image
5. **Push to Azure Container Registry** - Uploads the image to ACR
6. **Deploy to Azure Container Apps** - Creates a new revision with the updated image
7. **Show Deployment Info** - Displays the new revision name and URL

## Prerequisites

- Node.js and npm installed
- Go installed
- Docker installed and running
- Azure CLI installed and authenticated (`az login`)
- Access to Azure Container Registry (listmonk420acr.azurecr.io)

## Manual Steps

If you need to run individual steps:

### Build Email-Builder
```bash
cd frontend/email-builder
npm install
npm run build
mkdir -p ../public/static/email-builder
cp -r dist/* ../public/static/email-builder/
cd ../..
```

### Build Frontend
```bash
cd frontend
npm install
npm run build
cd ..
```

### Build Go Binary
```bash
go build -o listmonk -ldflags="-s -w -X 'main.buildString=$(date -u +%Y-%m-%dT%H:%M:%S%z)' -X 'main.versionString=$(git describe --tags --always)'" ./cmd
```

### Build Docker Image
```bash
docker build -t listmonk420acr.azurecr.io/listmonk420:latest .
```

### Push to ACR
```bash
docker push listmonk420acr.azurecr.io/listmonk420:latest
```

### Deploy to Azure
```bash
az containerapp update \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --set-env-vars "DEPLOY_TIME=$(date +%s)"
```

## Configuration

The script uses these Azure resources:
- **Resource Group**: rg-listmonk420
- **Container App**: listmonk420
- **Container Registry**: listmonk420acr.azurecr.io
- **Image Name**: listmonk420:latest

To modify these, edit the configuration variables at the top of `deploy.sh`.

## Checking Logs

After deployment, check the logs:
```bash
az containerapp logs show \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --follow
```

## Troubleshooting

### Docker Build Fails
- Ensure Go binary is built first
- Check that frontend/dist directory exists and has files

### Azure Push Fails
- Run `az login` to authenticate
- Check ACR credentials: `az acr login --name listmonk420acr`

### Deployment Doesn't Update
- The script forces a new revision by updating the DEPLOY_TIME environment variable
- Check revision name with: `az containerapp revision list --name listmonk420 --resource-group rg-listmonk420`

### Email-Builder Not Loading
- Ensure email-builder was built before frontend build
- Verify files exist in `frontend/dist/static/email-builder/`
- Check browser console for MIME type errors

## Azure Resources

- **URL**: https://list.bobbyseamoss.com
- **Database**: listmonk420-db.postgres.database.azure.com
- **Container App Environment**: listmonk420-env
