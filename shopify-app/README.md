# Listmonk Shopify Integration App

This directory contains the configuration for the dedicated Shopify app that integrates with listmonk for email marketing.

## Current Issue

Your existing token only has `read_customers` and `read_orders` scopes.
**Missing:** `read_products` (required to access variant/product data)

## Required Permissions (Access Scopes)

| Scope | Purpose |
|-------|---------|
| `read_customers` | Sync customer data to listmonk subscribers |
| `read_orders` | Sync order history for segmentation and purchase attribution |
| `read_products` | Access product/variant details in orders |

## Webhooks

| Topic | Endpoint | Purpose |
|-------|----------|---------|
| `orders/create` | `/webhooks/shopify/orders` | Purchase attribution, real-time order sync |
| `customers/create` | `/webhooks/shopify/customers` | Auto-subscribe new customers |
| `customers/update` | `/webhooks/shopify/customers` | Keep subscriber data in sync |

---

## Quick Setup: Create Custom App in Shopify Admin (RECOMMENDED)

The easiest way to get the correct scopes is to create a **Custom App** directly in your Shopify Admin.

### Step 1: Create Custom App

1. Go to **Shopify Admin** → https://b999e5-b7.myshopify.com/admin
2. Navigate to **Settings** (bottom left) → **Apps and sales channels**
3. Click **Develop apps** (top right)
4. If prompted, click **Allow custom app development**
5. Click **Create an app**
6. Name: `Listmonk Integration`
7. Click **Create app**

### Step 2: Configure Admin API Scopes

1. In your new app, click **Configure Admin API scopes**
2. Search and enable these scopes:
   - ✅ `read_customers`
   - ✅ `read_orders`
   - ✅ `read_products`
3. Click **Save**

### Step 3: Install and Get Token

1. Click the **API credentials** tab
2. Under "Access tokens", click **Install app**
3. Click **Install** in the popup
4. Click **Reveal token once** to see your Admin API access token
5. **Copy and save the token** - it's only shown once!

### Step 4: Update listmonk Settings

In listmonk database, update the access token:

```sql
-- Update Shopify access token
UPDATE settings
SET value = jsonb_set(value::jsonb, '{access_token}', '"YOUR_NEW_TOKEN"')
WHERE key = 'shopify';
```

Or update via API/UI if available.

### Step 5: Set Up Webhooks

In your Custom App:
1. Click **Webhooks** in the left menu
2. Click **Add webhook** for each:

| Event | Format | URL |
|-------|--------|-----|
| Customer creation | JSON | `https://list.bobbyseamoss.com/webhooks/shopify/customers` |
| Customer update | JSON | `https://list.bobbyseamoss.com/webhooks/shopify/customers` |
| Order creation | JSON | `https://list.bobbyseamoss.com/webhooks/shopify/orders` |

3. Copy the **Webhook signing secret** (shown at bottom)
4. Update in listmonk settings

---

## Alternative: Shopify Partner App Setup

### Step 1: Create App in Shopify Partner Dashboard

1. Go to [Shopify Partners](https://partners.shopify.com/)
2. Navigate to **Apps** > **Create app**
3. Choose **Create app manually**
4. Enter app name: `Listmonk Email Integration`

### Step 2: Configure App Settings

In the Shopify Partner Dashboard for your app:

#### API Access Scopes
Add these scopes under **Configuration** > **Access Scopes**:
- `read_customers`
- `read_orders`
- `read_products`

#### URLs
- **App URL**: `https://list.bobbyseamoss.com`
- **Allowed redirection URL(s)**: `https://list.bobbyseamoss.com/webhooks/shopify/auth/callback`

### Step 3: Get API Credentials

From the app's **API credentials** page, copy:
- **API key** (Client ID)
- **API secret key** (Client Secret)

### Step 4: Install App on Your Store

1. From the Partner Dashboard, click **Test your app**
2. Select your store (e.g., `b999e5-b7.myshopify.com`)
3. Click **Install app**
4. Authorize the requested permissions

### Step 5: Get Access Token

After installation, you'll receive an **Admin API access token**. This is the token to use in listmonk settings.

### Step 6: Configure listmonk

Update the Shopify settings in listmonk database or UI.

### Step 7: Set Up Webhooks in Shopify

In Shopify Admin > **Settings** > **Notifications** > **Webhooks**:

Or use the Shopify CLI:

```bash
# Navigate to this directory
cd shopify-app

# Link to your app
shopify app config link

# Deploy webhooks
shopify app deploy
```

#### Manual Webhook Setup (Alternative)

In Shopify Admin:

1. Go to **Settings** > **Notifications** > **Webhooks**
2. Click **Create webhook**
3. Add these webhooks:

| Event | Format | URL |
|-------|--------|-----|
| Order creation | JSON | `https://list.bobbyseamoss.com/webhooks/shopify/orders` |
| Customer creation | JSON | `https://list.bobbyseamoss.com/webhooks/shopify/customers` |
| Customer update | JSON | `https://list.bobbyseamoss.com/webhooks/shopify/customers` |

4. Copy the **Webhook signing secret** and add to listmonk settings

## Using Shopify CLI

### Prerequisites

```bash
# Install Shopify CLI (if not installed)
npm install -g @shopify/cli @shopify/app

# Login to Shopify Partners
shopify auth login
```

### Link to Existing App

```bash
cd shopify-app
shopify app config link
```

### Deploy Webhooks

```bash
shopify app deploy
```

### Check App Info

```bash
shopify app info
```

## Verifying Integration

### Test Customer Sync

```bash
# In listmonk, trigger manual sync
curl -X POST https://list.bobbyseamoss.com/api/shopify/customers/sync \
  -H "Authorization: Bearer YOUR_API_TOKEN"
```

### Test Webhook Delivery

1. Create a test order in Shopify
2. Check listmonk logs for webhook receipt:
   ```bash
   az containerapp logs show --name listmonk420 --resource-group rg-listmonk420 --tail 50 | grep -i shopify
   ```

### Verify Data in Database

```sql
-- Check synced orders
SELECT * FROM shopify_orders ORDER BY created_at DESC LIMIT 5;

-- Check synced line items
SELECT * FROM shopify_order_line_items ORDER BY id DESC LIMIT 10;
```

## Troubleshooting

### "Access denied for variant field"

This error means the app doesn't have `read_products` scope. Reinstall the app with updated permissions.

### Webhooks Not Arriving

1. Check webhook URL is correct
2. Verify HMAC secret matches
3. Check Shopify's webhook logs in Admin > Settings > Notifications > Webhooks

### Customer Sync Failing

1. Verify access token is valid
2. Check `read_customers` scope is granted
3. Test API access:
   ```bash
   curl -X GET "https://bobbyseamoss.myshopify.com/admin/api/2024-10/customers.json?limit=1" \
     -H "X-Shopify-Access-Token: YOUR_TOKEN"
   ```

## API Version

This integration uses Shopify Admin API version **2024-10**.

Update the version in:
- `shopify.app.toml` - webhooks.api_version
- `cmd/shopify_customers.go` - GraphQL endpoint URL
