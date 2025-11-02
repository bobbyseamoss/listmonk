<#
.SYNOPSIS
    Enables user engagement tracking for Azure Email Communication Services domains.

.DESCRIPTION
    This script enables user engagement tracking (open tracking, click tracking)
    for all provisioned domains across all Email Communication Services in the subscription.

.PARAMETER ResourceGroup
    Optional. Filter to only process Email Services in this resource group.

.PARAMETER DryRun
    Optional. Show what would be done without making changes.

.EXAMPLE
    .\Enable-EmailTracking.ps1
    Process all Email Services in the subscription

.EXAMPLE
    .\Enable-EmailTracking.ps1 -ResourceGroup "rg-listmonk420"
    Process only Email Services in the specified resource group

.EXAMPLE
    .\Enable-EmailTracking.ps1 -DryRun
    Preview changes without making them

.NOTES
    Prerequisites:
    - Azure CLI installed (version 2.67.0 or higher)
    - Azure Communication extension installed (auto-installs on first use)
    - Logged in to Azure (az login)
    - Domains must NOT be Azure Managed Domains
    - Domains must have upgraded sending limits (not default limits)

    Author: Azure Deployment Engineer
    Version: 1.0
#>

[CmdletBinding()]
param(
    [Parameter(Mandatory=$false)]
    [string]$ResourceGroup = "",

    [Parameter(Mandatory=$false)]
    [switch]$DryRun
)

# Error handling
$ErrorActionPreference = "Stop"

# Color functions
function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

function Write-Success {
    param([string]$Message)
    Write-ColorOutput "✓ $Message" "Green"
}

function Write-Error {
    param([string]$Message)
    Write-ColorOutput "✗ $Message" "Red"
}

function Write-Warning {
    param([string]$Message)
    Write-ColorOutput "⚠ $Message" "Yellow"
}

function Write-Info {
    param([string]$Message)
    Write-ColorOutput $Message "Cyan"
}

# Header
Write-Host "======================================================================" -ForegroundColor White
Write-Host "Azure Email Communication Services - User Engagement Tracking" -ForegroundColor White
Write-Host "======================================================================" -ForegroundColor White
Write-Host ""

# Check if Azure CLI is installed
Write-Host "Checking prerequisites..."
try {
    $azVersion = az version --query '"azure-cli"' -o tsv 2>$null
    if ($LASTEXITCODE -ne 0) {
        throw "Azure CLI not found"
    }
    Write-Host "Azure CLI version: $azVersion"
} catch {
    Write-Error "Azure CLI is not installed"
    Write-Host "Install from: https://docs.microsoft.com/en-us/cli/azure/install-azure-cli"
    exit 1
}

# Check if logged in to Azure
Write-Host "Checking Azure login status..."
try {
    $account = az account show --output json 2>$null | ConvertFrom-Json
    if ($LASTEXITCODE -ne 0) {
        throw "Not logged in"
    }
    Write-Success "Logged in as: $($account.user.name)"
    Write-Success "Subscription: $($account.name) ($($account.id))"
} catch {
    Write-Error "Not logged in to Azure"
    Write-Host "Please run: az login"
    exit 1
}
Write-Host ""

# Check if communication extension is available
Write-Host "Checking Azure Communication Services extension..."
try {
    $extensions = az extension list --output json | ConvertFrom-Json
    $commExt = $extensions | Where-Object { $_.name -eq "communication" }

    if ($null -eq $commExt) {
        Write-Warning "Installing communication extension..."
        az extension add --name communication --only-show-errors
        Write-Success "Extension installed"
    } else {
        Write-Success "Extension version: $($commExt.version)"
    }
} catch {
    Write-Warning "Could not verify extension status, continuing..."
}
Write-Host ""

if ($DryRun) {
    Write-ColorOutput "=== DRY RUN MODE - No changes will be made ===" "Yellow"
    Write-Host ""
}

# Step 1: List all Email Communication Services
Write-Host "======================================================================" -ForegroundColor White
Write-Host "Step 1: Discovering Email Communication Services" -ForegroundColor White
Write-Host "======================================================================" -ForegroundColor White
Write-Host ""

$listParams = @()
if ($ResourceGroup) {
    Write-Host "Filtering by resource group: $ResourceGroup"
    $listParams += "--resource-group"
    $listParams += $ResourceGroup
}

Write-Host "Fetching Email Communication Services..."
try {
    $emailServicesJson = az communication email list @listParams --output json 2>$null
    if ($LASTEXITCODE -ne 0) {
        $emailServices = @()
    } else {
        $emailServices = $emailServicesJson | ConvertFrom-Json
    }
} catch {
    $emailServices = @()
}

$serviceCount = $emailServices.Count

if ($serviceCount -eq 0) {
    Write-Warning "No Email Communication Services found"
    if ($ResourceGroup) {
        Write-Host "Try running without -ResourceGroup to search all resource groups"
    }
    exit 0
}

Write-Success "Found $serviceCount Email Communication Service(s)"
Write-Host ""

Write-Host "Services:"
foreach ($service in $emailServices) {
    Write-Host "  - $($service.name) (Resource Group: $($service.resourceGroup))"
}
Write-Host ""

# Step 2: Process each Email Service
Write-Host "======================================================================" -ForegroundColor White
Write-Host "Step 2: Processing Domains" -ForegroundColor White
Write-Host "======================================================================" -ForegroundColor White
Write-Host ""

$totalDomains = 0
$enabledCount = 0
$alreadyEnabledCount = 0
$failedCount = 0
$skippedCount = 0
$failedDomains = @()

foreach ($service in $emailServices) {
    Write-Info "Processing Email Service: $($service.name)"
    Write-Host "  Resource Group: $($service.resourceGroup)"

    # List domains for this service
    try {
        $domainsJson = az communication email domain list `
            --email-service-name $service.name `
            --resource-group $service.resourceGroup `
            --output json 2>$null

        if ($LASTEXITCODE -ne 0) {
            $domains = @()
        } else {
            $domains = $domainsJson | ConvertFrom-Json
        }
    } catch {
        $domains = @()
    }

    $domainCount = $domains.Count

    if ($domainCount -eq 0) {
        Write-Warning "  No domains found"
        Write-Host ""
        continue
    }

    Write-Host "  Found $domainCount domain(s)"
    Write-Host ""

    # Process each domain
    foreach ($domain in $domains) {
        $totalDomains++

        $domainName = $domain.name
        $domainManagement = if ($domain.properties.domainManagement) { $domain.properties.domainManagement } else { "Unknown" }
        $currentTracking = if ($domain.properties.userEngagementTracking) { $domain.properties.userEngagementTracking } else { "Unknown" }

        Write-ColorOutput "  Domain: $domainName" "Blue"
        Write-Host "    Management Type: $domainManagement"
        Write-Host "    Current Tracking Status: $currentTracking"

        # Check if Azure Managed Domain
        if ($domainManagement -eq "AzureManaged") {
            Write-Warning "    ⚠️  SKIPPED - Azure Managed Domains do not support user engagement tracking"
            $skippedCount++
            Write-Host ""
            continue
        }

        # Check if already enabled
        if ($currentTracking -eq "Enabled") {
            Write-Success "    Already enabled"
            $alreadyEnabledCount++
            Write-Host ""
            continue
        }

        # Enable tracking
        if ($DryRun) {
            Write-Warning "    [DRY RUN] Would enable user engagement tracking"
            $enabledCount++
        } else {
            Write-Host "    Enabling user engagement tracking... " -NoNewline

            try {
                az communication email domain update `
                    --domain-name $domainName `
                    --email-service-name $service.name `
                    --resource-group $service.resourceGroup `
                    --user-engmnt-tracking "Enabled" `
                    --output none 2>$null

                if ($LASTEXITCODE -eq 0) {
                    Write-ColorOutput "✓ SUCCESS" "Green"
                    $enabledCount++
                } else {
                    throw "Update failed"
                }
            } catch {
                Write-ColorOutput "✗ FAILED" "Red"
                $failedCount++
                $failedDomains += "$domainName (Service: $($service.name), RG: $($service.resourceGroup))"

                Write-Error "    Note: Domains with default sending limits cannot enable tracking."
                Write-Error "    Raise a support ticket to upgrade your service limits."
            }
        }

        Write-Host ""
    }
}

# Summary
Write-Host "======================================================================" -ForegroundColor White
Write-Host "Summary" -ForegroundColor White
Write-Host "======================================================================" -ForegroundColor White
Write-Host ""
Write-Host "Total Email Services: $serviceCount"
Write-Host "Total Domains Processed: $totalDomains"
Write-Host ""

if ($DryRun) {
    Write-ColorOutput "[DRY RUN MODE]" "Yellow"
    Write-ColorOutput "Would enable tracking on: $enabledCount domain(s)" "Yellow"
} else {
    Write-Success "Enabled: $enabledCount"
}

Write-Info "Already Enabled: $alreadyEnabledCount"

if ($skippedCount -gt 0) {
    Write-Warning "Skipped (Azure Managed): $skippedCount"
}

if ($failedCount -gt 0) {
    Write-Error "Failed: $failedCount"
    Write-Host ""
    Write-Host "Failed domains:"
    foreach ($domain in $failedDomains) {
        Write-Error "  - $domain"
    }
}

Write-Host ""

# Additional information
if ($failedCount -gt 0 -or $skippedCount -gt 0) {
    Write-Host "======================================================================" -ForegroundColor White
    Write-Host "Important Notes" -ForegroundColor White
    Write-Host "======================================================================" -ForegroundColor White
    Write-Host ""

    if ($skippedCount -gt 0) {
        Write-ColorOutput "Azure Managed Domains:" "Yellow"
        Write-Host "  User engagement tracking is not available for Azure Managed Domains"
        Write-Host "  (e.g., xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.azurecomm.net)"
        Write-Host "  Use custom domains instead."
        Write-Host ""
    }

    if ($failedCount -gt 0) {
        Write-ColorOutput "Default Sending Limits:" "Red"
        Write-Host "  Domains with default sending limits cannot enable user engagement tracking."
        Write-Host "  To enable tracking:"
        Write-Host "  1. Open a support ticket in Azure Portal"
        Write-Host "  2. Request to upgrade from default sending limits"
        Write-Host "  3. Once approved, run this script again"
        Write-Host ""
    }
}

Write-Host "======================================================================" -ForegroundColor White
Write-Host "User Engagement Tracking Features" -ForegroundColor White
Write-Host "======================================================================" -ForegroundColor White
Write-Host ""
Write-Host "When enabled, the following tracking is available:"
Write-Host "  - Email opens (pixel-based tracking)"
Write-Host "  - Link clicks (URL redirect tracking)"
Write-Host ""
Write-Host "To view engagement data:"
Write-Host "  1. Azure Portal > Email Communication Service > Insights"
Write-Host "  2. Or subscribe to Email User Engagement operational logs"
Write-Host "  3. Or use Azure Monitor to query engagement events"
Write-Host ""
Write-Host "Note: Tracking only works for HTML emails, not plain text."
Write-Host ""

exit 0
