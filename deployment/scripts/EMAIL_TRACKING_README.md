# Azure Email Communication Services - User Engagement Tracking Scripts

This directory contains scripts to enable user engagement tracking (open tracking and click tracking) for all provisioned domains across Azure Email Communication Services.

## Overview

User engagement tracking enables:
- **Email Opens**: Pixel-based tracking to detect when recipients open emails
- **Link Clicks**: URL redirect tracking to monitor which links are clicked

## Scripts

### Bash Script (Linux/macOS/WSL)
**File**: `enable_email_tracking.sh`

### PowerShell Script (Windows/Cross-platform)
**File**: `Enable-EmailTracking.ps1`

Both scripts provide identical functionality with platform-appropriate syntax.

## Prerequisites

### Required
- **Azure CLI**: Version 2.67.0 or higher
  - Install: https://docs.microsoft.com/en-us/cli/azure/install-azure-cli
- **Azure Account**: Must be logged in (`az login`)
- **Azure Communication Extension**: Auto-installs on first use

### Domain Requirements
User engagement tracking has specific limitations:

1. **NOT available for Azure Managed Domains**
   - Azure Managed Domains (e.g., `xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx.azurecomm.net`) do not support tracking
   - Use custom domains instead

2. **Requires upgraded sending limits**
   - Domains with default sending limits cannot enable tracking
   - Must raise a support ticket to upgrade from default limits
   - See "Upgrading Sending Limits" section below

3. **Only works with HTML emails**
   - Tracking does not function with plain text emails
   - Ensure your listmonk campaigns use HTML templates

## Usage

### Bash (Linux/macOS/WSL)

```bash
# Process all Email Services in subscription
./enable_email_tracking.sh

# Process only specific resource group
./enable_email_tracking.sh --resource-group rg-listmonk420

# Dry run (preview changes without making them)
./enable_email_tracking.sh --dry-run

# Combine options
./enable_email_tracking.sh --resource-group rg-listmonk420 --dry-run
```

### PowerShell (Windows)

```powershell
# Process all Email Services in subscription
.\Enable-EmailTracking.ps1

# Process only specific resource group
.\Enable-EmailTracking.ps1 -ResourceGroup "rg-listmonk420"

# Dry run (preview changes without making them)
.\Enable-EmailTracking.ps1 -DryRun

# Combine options
.\Enable-EmailTracking.ps1 -ResourceGroup "rg-listmonk420" -DryRun
```

## What the Scripts Do

1. **Discover Email Services**: Lists all Azure Email Communication Services in the subscription (or filtered by resource group)

2. **List Domains**: For each service, retrieves all provisioned domains

3. **Check Domain Status**:
   - Verifies domain management type (skips Azure Managed)
   - Checks current tracking status (skips already enabled)

4. **Enable Tracking**: Updates domains to enable user engagement tracking

5. **Report Results**: Provides detailed summary of:
   - Total domains processed
   - Successfully enabled
   - Already enabled (no change needed)
   - Skipped (Azure Managed)
   - Failed (default limits)

## Output Example

```
======================================================================
Azure Email Communication Services - User Engagement Tracking
======================================================================

✓ Logged in as: user@example.com
✓ Subscription: My Subscription (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)

======================================================================
Step 1: Discovering Email Communication Services
======================================================================

✓ Found 2 Email Communication Service(s)

Services:
  - listmonk-email-service (Resource Group: rg-listmonk420)
  - backup-email-service (Resource Group: rg-listmonk420)

======================================================================
Step 2: Processing Domains
======================================================================

Processing Email Service: listmonk-email-service
  Found 3 domain(s)

  Domain: mail1.bobbyseamoss.com
    Management Type: CustomerManaged
    Current Tracking Status: Disabled
    Enabling user engagement tracking... ✓ SUCCESS

  Domain: mail2.bobbyseamoss.com
    Management Type: CustomerManaged
    Current Tracking Status: Enabled
    ✓ Already enabled

  Domain: xxxxx.azurecomm.net
    Management Type: AzureManaged
    ⚠️  SKIPPED - Azure Managed Domains do not support tracking

======================================================================
Summary
======================================================================

Total Email Services: 2
Total Domains Processed: 3

✓ Enabled: 1
○ Already Enabled: 1
⚠ Skipped (Azure Managed): 1
```

## Upgrading Sending Limits

If domains fail to enable tracking due to default sending limits:

### Step 1: Open Support Ticket
1. Go to Azure Portal
2. Navigate to your Email Communication Service
3. Click "Support + troubleshooting" → "New support request"

### Step 2: Request Upgrade
**Subject**: Request to upgrade from default sending limits

**Description**:
```
I would like to request an upgrade from default sending limits for my
Email Communication Service to enable user engagement tracking.

Resource: [Your Email Service Name]
Resource Group: [Your Resource Group]
Subscription: [Your Subscription ID]

Use Case: Operating a listmonk newsletter system with multiple SMTP
servers requiring open and click tracking for engagement analytics.
```

### Step 3: Re-run Script
Once the upgrade is approved (typically 1-3 business days):
```bash
./enable_email_tracking.sh
```

## Viewing Engagement Data

After enabling tracking, view engagement data through:

### Azure Portal
1. Navigate to Email Communication Service
2. Click "Insights" in the left menu
3. View open and click metrics

### Azure Monitor
Set up operational logs:
1. Navigate to Email Communication Service
2. Click "Diagnostic settings"
3. Add "Email User Engagement" logs
4. Send to Log Analytics workspace

### Query Example (KQL)
```kusto
ACSEmailUserEngagementOperational
| where TimeGenerated > ago(7d)
| where EngagementType in ("Open", "Click")
| summarize count() by EngagementType, bin(TimeGenerated, 1h)
| render timechart
```

### Programmatic Access
Use Azure Communication Services SDK or REST API to query engagement events.

## Idempotency

Both scripts are **idempotent** and safe to run multiple times:
- Skips domains where tracking is already enabled
- Only updates domains that need changes
- No harm in re-running after failures

## Error Handling

The scripts handle common errors:
- **Azure CLI not installed**: Clear error message with installation link
- **Not logged in**: Prompts to run `az login`
- **No services found**: Suggests removing resource group filter
- **Permission denied**: Indicates insufficient Azure permissions
- **Default limits**: Explains support ticket process

## Integration with Listmonk

This tracking integration enhances listmonk's campaign analytics:

1. **Configure SMTP Servers**: Set up Azure Email Communication Services SMTP servers in listmonk settings

2. **Enable Tracking**: Run this script to enable tracking on all domains

3. **Send HTML Campaigns**: Use HTML templates in listmonk (TinyMCE editor)

4. **Track Engagement**: View open/click data in Azure Portal or integrate with listmonk's webhook system

5. **Smart Sending**: Use engagement data to optimize send times and content

## Azure CLI Commands Reference

### Manual Commands

If you prefer to enable tracking manually for specific domains:

```bash
# List all Email Services
az communication email list --resource-group rg-listmonk420

# List domains for a service
az communication email domain list \
  --email-service-name listmonk-email-service \
  --resource-group rg-listmonk420

# Show domain details
az communication email domain show \
  --domain-name mail1.bobbyseamoss.com \
  --email-service-name listmonk-email-service \
  --resource-group rg-listmonk420

# Enable user engagement tracking
az communication email domain update \
  --domain-name mail1.bobbyseamoss.com \
  --email-service-name listmonk-email-service \
  --resource-group rg-listmonk420 \
  --user-engmnt-tracking Enabled

# Disable user engagement tracking
az communication email domain update \
  --domain-name mail1.bobbyseamoss.com \
  --email-service-name listmonk-email-service \
  --resource-group rg-listmonk420 \
  --user-engmnt-tracking Disabled
```

## Troubleshooting

### Issue: "communication extension not found"
**Solution**: The extension auto-installs on first use. If it fails:
```bash
az extension add --name communication
```

### Issue: "domain update failed - default limits"
**Solution**: Open support ticket to upgrade from default sending limits (see "Upgrading Sending Limits" section)

### Issue: "Azure Managed Domain cannot enable tracking"
**Solution**: Azure Managed Domains don't support tracking. Use custom domains:
1. Add custom domain in Azure Portal
2. Configure DNS records (MX, TXT, DKIM)
3. Wait for verification
4. Run script again

### Issue: "permission denied"
**Solution**: Ensure you have appropriate Azure RBAC roles:
- **Contributor** or **Owner** on the Email Communication Service
- Or custom role with `Microsoft.Communication/EmailServices/Domains/write` permission

### Issue: "tracking enabled but not working"
**Checklist**:
1. Sending HTML emails (not plain text)?
2. Domain has upgraded sending limits?
3. DNS records properly configured?
4. Wait 15-30 minutes for propagation
5. Check Azure Monitor for engagement events

## Security Considerations

### Privacy Compliance
User engagement tracking involves:
- **Open Tracking**: 1x1 transparent pixel in email
- **Click Tracking**: URL redirect through Azure infrastructure

Ensure compliance with:
- **GDPR**: Inform users about tracking in privacy policy
- **CAN-SPAM**: Provide opt-out mechanism (listmonk unsubscribe)
- **CASL**: Obtain consent for tracking

### Best Practices
1. **Transparency**: Disclose tracking in privacy policy
2. **Unsubscribe Honors**: Listmonk automatically handles this
3. **Data Retention**: Configure Azure Monitor retention policies
4. **Access Control**: Limit who can view engagement data

## Cost Implications

User engagement tracking is included in Azure Email Communication Services at no additional cost. However:

- **Azure Monitor**: If sending logs to Log Analytics, standard ingestion and retention charges apply
- **Storage**: Engagement event logs consume storage (typically minimal)
- **Queries**: KQL queries against large datasets may incur costs

Estimate: ~$0.10-1.00/month for typical listmonk deployments (10K-100K emails/month)

## Migration Path

If currently using Azure Managed Domains:

### Step 1: Prepare Custom Domains
```bash
# Your custom domains (already in use based on SMTP config)
- mail1.bobbyseamoss.com
- mail2.bobbyseamoss.com
- mail3.bobbyseamoss.com
# ... etc
```

### Step 2: Add to Azure Email Service
1. Azure Portal → Email Communication Service
2. Click "Provision Domains"
3. Add custom domain
4. Configure DNS records as shown

### Step 3: Verify Domains
Wait for DNS propagation and Azure verification (typically 15-30 minutes)

### Step 4: Enable Tracking
```bash
./enable_email_tracking.sh
```

### Step 5: Update Listmonk SMTP
Update `from_email` addresses in listmonk SMTP settings to use custom domains

## Script Maintenance

Both scripts are designed to be:
- **Self-documenting**: Comprehensive inline comments
- **Future-proof**: Uses stable Azure CLI commands
- **Extensible**: Easy to add features (e.g., bulk disable)

### Suggested Enhancements
- Add `--disable` flag to bulk disable tracking
- Export engagement stats to CSV
- Integration with listmonk API for automatic configuration
- Slack/email notifications on completion

## Support

For issues with:
- **Scripts**: Open issue in listmonk repository
- **Azure Email Service**: Contact Azure Support
- **Listmonk**: See https://listmonk.app/docs

## References

- [Azure Email Communication Services Documentation](https://learn.microsoft.com/en-us/azure/communication-services/concepts/email/email-overview)
- [User Engagement Tracking Quickstart](https://learn.microsoft.com/en-us/azure/communication-services/quickstarts/email/enable-user-engagement-tracking)
- [Azure CLI Communication Commands](https://learn.microsoft.com/en-us/cli/azure/communication/email/domain)
- [Listmonk Multi-SMTP Documentation](../CLAUDE.md#multi-smtp-server-support)

## Version History

- **v1.0** (2025-11-01): Initial release
  - Bash and PowerShell implementations
  - Dry-run mode
  - Resource group filtering
  - Comprehensive error handling
