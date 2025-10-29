# Quick Command Reference

## Deployment

```bash
# Full build and deploy
./deploy.sh

# Watch deployment progress
az containerapp revision list \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  -o table
```

## Logs

```bash
# View live logs
az containerapp logs show \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --follow

# View last 50 lines
az containerapp logs show \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --tail 50
```

## Container Management

```bash
# Restart container (force new revision)
az containerapp revision restart \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --revision $(az containerapp show --name listmonk420 --resource-group rg-listmonk420 --query properties.latestRevisionName -o tsv)

# List all revisions
az containerapp revision list \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  -o table

# Deactivate old revisions
az containerapp revision deactivate \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --revision <revision-name>

# Scale container
az containerapp update \
  --name listmonk420 \
  --resource-group rg-listmonk420 \
  --min-replicas 1 \
  --max-replicas 5
```

## Database

```bash
# Connect to database
PGPASSWORD='T@intshr3dd3r' psql \
  -h listmonk420-db.postgres.database.azure.com \
  -U listmonkadmin \
  -d listmonk

# Run query from command line
PGPASSWORD='T@intshr3dd3r' psql \
  -h listmonk420-db.postgres.database.azure.com \
  -U listmonkadmin \
  -d listmonk \
  -c "SELECT COUNT(*) FROM subscribers;"

# Backup database
pg_dump -h listmonk420-db.postgres.database.azure.com \
  -U listmonkadmin \
  -d listmonk \
  --no-password \
  > listmonk_backup_$(date +%Y%m%d).sql
```

## Docker

```bash
# Login to Azure Container Registry
az acr login --name listmonk420acr

# List images
az acr repository list --name listmonk420acr -o table

# List tags for image
az acr repository show-tags \
  --name listmonk420acr \
  --repository listmonk420 \
  -o table

# Remove old images
az acr repository delete \
  --name listmonk420acr \
  --image listmonk420:<old-tag> \
  --yes
```

## Local Development

```bash
# Run frontend dev server
cd frontend
npm run dev

# Build frontend only
cd frontend
npm run build

# Build Go binary only
go build -o listmonk ./cmd

# Run locally with local database
./listmonk --config config.toml

# Run tests
go test ./...
```

## Git

```bash
# View changes
git status
git diff

# Commit changes
git add .
git commit -m "Description of changes"
git push

# Create new branch
git checkout -b feature/new-feature

# View commit history
git log --oneline -10
```

## Azure Resources Info

```bash
# Show container app details
az containerapp show \
  --name listmonk420 \
  --resource-group rg-listmonk420

# Show database details
az postgres flexible-server show \
  --name listmonk420-db \
  --resource-group rg-listmonk420

# Show container environment details
az containerapp env show \
  --name listmonk420-env \
  --resource-group rg-listmonk420

# List all resources in resource group
az resource list \
  --resource-group rg-listmonk420 \
  -o table
```

## Monitoring

```bash
# Check container health
curl -I https://list.bobbyseamoss.com/

# Check specific endpoint
curl https://list.bobbyseamoss.com/health

# View metrics
az monitor metrics list \
  --resource $(az containerapp show --name listmonk420 --resource-group rg-listmonk420 --query id -o tsv) \
  --metric-names Requests \
  --start-time 2025-10-28T00:00:00Z \
  --end-time 2025-10-28T23:59:59Z
```

## Troubleshooting

```bash
# Check if email-builder exists
curl -I https://list.bobbyseamoss.com/admin/static/email-builder/email-builder.umd.js

# Verify frontend build
ls -lh frontend/dist/static/email-builder/

# Check Go binary
./listmonk --version

# Verify Docker image
docker images | grep listmonk420

# Check Azure login
az account show

# Test database connection
PGPASSWORD='T@intshr3dd3r' psql \
  -h listmonk420-db.postgres.database.azure.com \
  -U listmonkadmin \
  -d listmonk \
  -c "SELECT 1;"
```
