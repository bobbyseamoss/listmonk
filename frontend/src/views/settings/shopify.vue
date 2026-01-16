<template>
  <div>
    <h3 class="title is-5">{{ $t('settings.shopify.title', 'Shopify Integration') }}</h3>
    <p class="has-text-grey mb-4">
      {{ $t('settings.shopify.description', 'Attribute purchases to email campaigns when customers buy through your Shopify store.') }}
    </p>

    <div class="columns mb-5">
      <div class="column is-3">
        <b-field :label="$t('settings.shopify.enable', 'Enable Shopify')">
          <b-switch v-model="enabled" name="shopify.enabled" />
        </b-field>
      </div>
    </div>

    <div v-if="enabled" class="box">
      <div class="columns mb-4">
        <div class="column">
          <b-field
            :label="$t('settings.shopify.webhookUrl', 'Webhook URL')"
            :message="$t('settings.shopify.webhookUrlHelp', 'Copy this URL and configure it in Shopify Admin → Settings → Notifications → Webhooks')">
            <b-input
              :value="webhookUrl"
              readonly
              type="text" />
            <p class="control">
              <button type="button" class="button is-primary" @click="copyWebhookUrl">
                <b-icon icon="content-copy" size="is-small" />
                <span>{{ $t('globals.buttons.copy', 'Copy') }}</span>
              </button>
            </p>
          </b-field>
        </div>
      </div>

      <div class="columns mb-4">
        <div class="column">
          <b-field
            :label="$t('settings.shopify.webhookSecret', 'Webhook Secret')"
            :message="webhookSecretHelp">
            <b-input
              v-model="webhookSecret"
              type="password"
              name="webhook_secret"
              placeholder="Leave blank to use Client Secret" />
          </b-field>
        </div>
      </div>

      <div class="columns mb-4">
        <div class="column is-4">
          <b-field
            :label="$t('settings.shopify.attributionWindow', 'Attribution Window (Days)')"
            :message="$t('settings.shopify.attributionWindowHelp', 'How many days after a link click should purchases be attributed to a campaign?')">
            <b-select
              v-model="attributionWindowDays"
              name="attribution_window_days"
              expanded>
              <option :value="7">7 {{ $t('settings.shopify.days', 'days') }}</option>
              <option :value="14">14 {{ $t('settings.shopify.days', 'days') }}</option>
              <option :value="30">30 {{ $t('settings.shopify.days', 'days') }}</option>
              <option :value="60">60 {{ $t('settings.shopify.days', 'days') }}</option>
              <option :value="90">90 {{ $t('settings.shopify.days', 'days') }}</option>
            </b-select>
          </b-field>
        </div>
      </div>

      <hr />

      <h4 class="title is-6">Shopify Admin API (for Order Tally)</h4>
      <p class="has-text-grey mb-4">
        Configure your Shopify Admin API credentials to use the Order Tally feature.
        Create a custom app in Shopify Admin → Settings → Apps and sales channels → Develop apps.
      </p>

      <div class="columns mb-4">
        <div class="column">
          <b-field
            label="Store URL"
            message="Your Shopify store URL (e.g., your-store.myshopify.com)">
            <b-input
              v-model="storeUrl"
              type="text"
              name="store_url"
              placeholder="your-store.myshopify.com" />
          </b-field>
        </div>
      </div>

      <div class="columns mb-4">
        <div class="column">
          <b-field
            label="Admin API Access Token"
            message="Your Shopify Admin API access token (starts with shpat_). Leave blank to keep existing.">
            <b-input
              v-model="accessToken"
              type="password"
              name="access_token"
              placeholder="shpat_..." />
          </b-field>
        </div>
      </div>

      <div class="notification is-info is-light">
        <p><strong>{{ $t('settings.shopify.howItWorks', 'How it works:') }}</strong></p>
        <ol>
          <li>{{ $t('settings.shopify.step1', 'Send campaigns with tracked links') }}</li>
          <li>{{ $t('settings.shopify.step2', 'Subscribers click links in your emails') }}</li>
          <li>{{ $t('settings.shopify.step3', 'When they purchase in Shopify, the webhook sends order data to listmonk') }}</li>
          <li>{{ $t('settings.shopify.step4', 'Listmonk attributes the purchase to the campaign if the subscriber clicked within the attribution window') }}</li>
        </ol>
      </div>

      <hr />

      <!-- Customer Sync Section -->
      <h4 class="title is-6">Customer Sync</h4>
      <p class="has-text-grey mb-4">
        Sync customer data from Shopify to subscriber attributes.
        When enabled, customer creates/updates will sync First Name, Last Name, Address, Phone, Tags, and Marketing Consent.
      </p>

      <div class="columns mb-4">
        <div class="column is-3">
          <b-field label="Enable Customer Sync">
            <b-switch v-model="customerSyncEnabled" name="shopify.customer_sync_enabled" />
          </b-field>
        </div>
      </div>

      <div v-if="customerSyncEnabled">
        <div class="columns mb-4">
          <div class="column">
            <b-field
              label="Customer Webhook URL"
              message="Copy this URL and configure it in Shopify Admin for Customer create/update events">
              <b-input
                :value="customerWebhookUrl"
                readonly
                type="text" />
              <p class="control">
                <button type="button" class="button is-primary" @click="copyCustomerWebhookUrl">
                  <b-icon icon="content-copy" size="is-small" />
                  <span>{{ $t('globals.buttons.copy', 'Copy') }}</span>
                </button>
              </p>
            </b-field>
          </div>
        </div>

        <div class="columns mb-4">
          <div class="column is-4">
            <b-field
              label="Add Synced Customers to List (Optional)"
              message="Optionally add newly synced customers to a list">
              <b-select
                v-model="customerSyncListId"
                name="customer_sync_list_id"
                expanded>
                <option :value="0">None</option>
                <option v-for="list in lists" :key="list.id" :value="list.id">
                  {{ list.name }}
                </option>
              </b-select>
            </b-field>
          </div>
        </div>

        <div class="columns mb-4">
          <div class="column">
            <div class="box has-background-light">
              <h5 class="title is-6 mb-3">Bulk Sync</h5>
              <p class="mb-3">
                Sync all existing Shopify customers to matching subscribers.
                Only subscribers with matching emails will be updated.
              </p>

              <div v-if="syncStatus && syncStatus.in_progress" class="mb-3">
                <b-progress
                  type="is-primary"
                  :value="syncProgress"
                  show-value
                  format="percent" />
                <p class="has-text-grey is-size-7 mt-2">
                  {{ syncStatus.synced_count }} synced,
                  {{ syncStatus.skipped_count }} skipped (no matching subscriber),
                  {{ syncStatus.error_count }} errors
                  of {{ syncStatus.total_count }} total
                </p>
                <p v-if="syncStatus.last_error" class="has-text-danger is-size-7 mt-1">
                  Last error: {{ syncStatus.last_error }}
                </p>
              </div>

              <div v-else-if="lastSyncTime" class="mb-3">
                <p class="has-text-grey is-size-7">
                  Last sync: {{ formatDate(lastSyncTime) }}
                </p>
              </div>

              <b-button
                type="is-primary"
                :loading="syncStatus && syncStatus.in_progress"
                :disabled="syncStatus && syncStatus.in_progress"
                @click="startSync">
                <b-icon icon="sync" size="is-small" />
                <span>Sync Now</span>
              </b-button>
            </div>
          </div>
        </div>

        <div class="notification is-light">
          <p><strong>Data Synced:</strong></p>
          <ul class="mt-2">
            <li>First Name &amp; Last Name</li>
            <li>Phone Number</li>
            <li>Default Address (Street, City, Province, Country, ZIP)</li>
            <li>Tags (from Shopify customer tags)</li>
            <li>Order Count &amp; Total Spent</li>
            <li>Birthday (if stored in metafield)</li>
          </ul>
          <p class="mt-3 has-text-grey is-size-7">
            Data is stored in subscriber attributes under <code>shopify.*</code>
          </p>
          <p class="mt-2 has-text-grey is-size-7">
            <strong>Note:</strong> Email marketing consent is NOT synced from Shopify.
            listmonk is the authority on email subscriptions.
          </p>
        </div>
      </div>

      <hr />

      <!-- Storefront Forms Section -->
      <h4 class="title is-6">
        <b-icon icon="form-select" size="is-small" class="mr-1" />
        Storefront Forms
      </h4>
      <p class="has-text-grey mb-4">
        Automatically display listmonk sign-up forms on your Shopify storefront.
        Create forms in the <router-link :to="{ name: 'signupForms' }">Forms</router-link> section
        and configure Shopify page targeting.
      </p>

      <div class="columns mb-4">
        <div class="column is-3">
          <b-field label="Enable Storefront Forms">
            <b-switch v-model="formsEnabled" name="shopify.forms_enabled" :disabled="!storeUrl || !accessToken" />
          </b-field>
        </div>
      </div>

      <div v-if="!storeUrl || !accessToken" class="notification is-warning is-light mb-4">
        <b-icon icon="alert" size="is-small" />
        Configure Store URL and Admin API Access Token above before enabling Storefront Forms.
      </div>

      <div v-else-if="formsEnabled">
        <div v-if="formsStatus && formsStatus.installed" class="notification is-success is-light mb-4">
          <p><strong>ScriptTag Installed</strong></p>
          <p class="is-size-7 mt-1">
            Your Shopify store will automatically load forms based on targeting rules.
          </p>
          <p class="is-size-7 has-text-grey mt-2">
            ScriptTag ID: {{ formsStatus.script_tag_id }}
          </p>
        </div>

        <div v-else class="notification is-warning is-light mb-4">
          <p><strong>ScriptTag Not Installed</strong></p>
          <p class="mb-3 is-size-7">
            Click "Save" to install the form loader script on your Shopify store.
          </p>
        </div>

        <div class="notification is-info is-light">
          <p><strong>How it works:</strong></p>
          <ol class="mt-2">
            <li>Create forms in <router-link :to="{ name: 'signupForms' }">Forms</router-link></li>
            <li>Under <strong>Targeting</strong>, select Shopify page types (Product, Cart, etc.)</li>
            <li>Activate the form</li>
            <li>Forms automatically appear on matching Shopify pages</li>
          </ol>
          <p class="mt-3 is-size-7 has-text-grey">
            <strong>Note:</strong> You may need to re-install the Shopify app if you see permission errors.
            The app now requires the <code>write_script_tags</code> scope.
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import Vue from 'vue';
import {
  getLists,
  startShopifyCustomerSync,
  getShopifyCustomerSyncStatus,
  getShopifyFormsStatus,
} from '../../api';

export default Vue.extend({
  name: 'Shopify',
  props: {
    form: {
      type: Object,
      default: () => ({}),
    },
  },
  data() {
    return {
      lists: [],
      syncStatus: null,
      syncPollInterval: null,
      formsStatus: null,
    };
  },
  computed: {
    webhookUrl() {
      // Get the root URL from app settings
      const rootUrl = this.$store.state.settings['app.root_url'] || window.location.origin;
      return `${rootUrl}/webhooks/shopify/orders`;
    },
    webhookSecretHelp() {
      return this.$t(
        'settings.shopify.webhookSecretHelp',
        'For Partner app webhooks, leave blank to auto-use the Client Secret. '
        + 'For manual webhooks, copy from Shopify Admin → Notifications → Webhooks.',
      );
    },
    customerWebhookUrl() {
      const rootUrl = this.$store.state.settings['app.root_url'] || window.location.origin;
      return `${rootUrl}/webhooks/shopify/customers`;
    },
    enabled: {
      get() {
        return this.form.shopify?.enabled || false;
      },
      set(value) {
        this.$set(this.form.shopify, 'enabled', value);
      },
    },
    webhookSecret: {
      get() {
        return this.form.shopify?.webhook_secret || '';
      },
      set(value) {
        this.$set(this.form.shopify, 'webhook_secret', value);
      },
    },
    attributionWindowDays: {
      get() {
        return this.form.shopify?.attribution_window_days || 14;
      },
      set(value) {
        this.$set(this.form.shopify, 'attribution_window_days', value);
      },
    },
    storeUrl: {
      get() {
        return this.form.shopify?.store_url || '';
      },
      set(value) {
        this.$set(this.form.shopify, 'store_url', value);
      },
    },
    accessToken: {
      get() {
        return this.form.shopify?.access_token || '';
      },
      set(value) {
        this.$set(this.form.shopify, 'access_token', value);
      },
    },
    customerSyncEnabled: {
      get() {
        return this.form.shopify?.customer_sync_enabled || false;
      },
      set(value) {
        this.$set(this.form.shopify, 'customer_sync_enabled', value);
      },
    },
    customerSyncListId: {
      get() {
        return this.form.shopify?.customer_sync_list_id || 0;
      },
      set(value) {
        this.$set(this.form.shopify, 'customer_sync_list_id', value);
      },
    },
    lastSyncTime() {
      return this.form.shopify?.last_customer_sync || null;
    },
    formsEnabled: {
      get() {
        return this.form.shopify?.forms_enabled || false;
      },
      set(value) {
        this.$set(this.form.shopify, 'forms_enabled', value);
      },
    },
    syncProgress() {
      if (!this.syncStatus || this.syncStatus.total_count === 0) return 0;
      return Math.round(((this.syncStatus.synced_count + this.syncStatus.skipped_count + this.syncStatus.error_count) / this.syncStatus.total_count) * 100);
    },
  },
  mounted() {
    this.loadLists();
    this.loadSyncStatus();
    this.loadFormsStatus();
  },
  beforeDestroy() {
    if (this.syncPollInterval) {
      clearInterval(this.syncPollInterval);
    }
  },
  methods: {
    async loadLists() {
      try {
        const resp = await getLists();
        this.lists = resp.data || [];
      } catch (e) {
        // Ignore errors loading lists
      }
    },
    async loadSyncStatus() {
      try {
        const resp = await getShopifyCustomerSyncStatus();
        this.syncStatus = resp.data;

        // If sync is in progress, start polling
        if (this.syncStatus && this.syncStatus.in_progress && !this.syncPollInterval) {
          this.syncPollInterval = setInterval(() => this.loadSyncStatus(), 2000);
        } else if (this.syncStatus && !this.syncStatus.in_progress && this.syncPollInterval) {
          clearInterval(this.syncPollInterval);
          this.syncPollInterval = null;
        }
      } catch (e) {
        // Ignore errors loading sync status
      }
    },
    async startSync() {
      try {
        await startShopifyCustomerSync();
        this.$buefy.toast.open({
          message: 'Customer sync started',
          type: 'is-success',
          duration: 3000,
        });
        // Start polling for status updates
        this.syncPollInterval = setInterval(() => this.loadSyncStatus(), 2000);
        this.loadSyncStatus();
      } catch (e) {
        this.$buefy.toast.open({
          message: e.response?.data?.message || 'Failed to start sync',
          type: 'is-danger',
          duration: 5000,
        });
      }
    },
    copyWebhookUrl() {
      // Copy webhook URL to clipboard
      navigator.clipboard.writeText(this.webhookUrl).then(() => {
        this.$buefy.toast.open({
          message: this.$t('globals.messages.copied', 'Copied to clipboard'),
          type: 'is-success',
          duration: 2000,
        });
      }).catch(() => {
        this.$buefy.toast.open({
          message: this.$t('globals.messages.errorCopying', 'Failed to copy'),
          type: 'is-danger',
          duration: 2000,
        });
      });
    },
    copyCustomerWebhookUrl() {
      navigator.clipboard.writeText(this.customerWebhookUrl).then(() => {
        this.$buefy.toast.open({
          message: this.$t('globals.messages.copied', 'Copied to clipboard'),
          type: 'is-success',
          duration: 2000,
        });
      }).catch(() => {
        this.$buefy.toast.open({
          message: this.$t('globals.messages.errorCopying', 'Failed to copy'),
          type: 'is-danger',
          duration: 2000,
        });
      });
    },
    formatDate(dateStr) {
      if (!dateStr) return '';
      const date = new Date(dateStr);
      return date.toLocaleString();
    },
    async loadFormsStatus() {
      try {
        const resp = await getShopifyFormsStatus();
        this.formsStatus = resp.data;
      } catch (e) {
        // Ignore errors loading forms status
      }
    },
  },
});
</script>
