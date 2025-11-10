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
            :message="$t('settings.shopify.webhookSecretHelp', 'Copy this from your Shopify webhook configuration. Leave blank to keep existing value.')">
            <b-input
              v-model="webhookSecret"
              type="password"
              name="webhook_secret"
              placeholder="Leave blank to keep existing" />
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

      <div class="notification is-info is-light">
        <p><strong>{{ $t('settings.shopify.howItWorks', 'How it works:') }}</strong></p>
        <ol>
          <li>{{ $t('settings.shopify.step1', 'Send campaigns with tracked links') }}</li>
          <li>{{ $t('settings.shopify.step2', 'Subscribers click links in your emails') }}</li>
          <li>{{ $t('settings.shopify.step3', 'When they purchase in Shopify, the webhook sends order data to listmonk') }}</li>
          <li>{{ $t('settings.shopify.step4', 'Listmonk attributes the purchase to the campaign if the subscriber clicked within the attribution window') }}</li>
        </ol>
      </div>
    </div>
  </div>
</template>

<script>
import Vue from 'vue';

export default Vue.extend({
  name: 'Shopify',
  props: {
    form: {
      type: Object,
      default: () => ({}),
    },
  },
  computed: {
    webhookUrl() {
      // Get the root URL from app settings
      const rootUrl = this.$store.state.settings['app.root_url'] || window.location.origin;
      return `${rootUrl}/webhooks/shopify/orders`;
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
  },
  methods: {
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
  },
});
</script>
