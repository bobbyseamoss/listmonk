#!/bin/bash
# ========================================================================
# GAM Setup Script for Google Workspace
# ========================================================================
# Run this script ONCE to authorize GAM with your Google Workspace
# This requires interactive browser-based OAuth
# ========================================================================

GAM="/home/adam/bin/gamadv-xtd3/gam"

echo "=========================================================================="
echo "GAM Setup for Google Workspace"
echo "=========================================================================="
echo ""
echo "This will guide you through authorizing GAM with your Google Workspace."
echo "You will need:"
echo "  1. Super Admin access to your Google Workspace"
echo "  2. A web browser for OAuth authorization"
echo ""
echo "Press Enter to continue or Ctrl+C to cancel..."
read

# Step 1: Create project and enable APIs
echo ""
echo "Step 1: Creating Google Cloud project..."
$GAM create project

# Step 2: Authorize admin access
echo ""
echo "Step 2: Authorizing admin access..."
$GAM oauth create

# Step 3: Test connection
echo ""
echo "Step 3: Testing connection..."
$GAM info domain

echo ""
echo "=========================================================================="
echo "GAM Setup Complete!"
echo "=========================================================================="
echo ""
echo "You can now run: ./create_subdomains.sh"
