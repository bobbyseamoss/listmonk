<template>
  <div class="items" v-if="data.onsite_tracking">
    <b-field label="Enable Onsite Tracking"
      message="Track identified subscribers on your website. Only visitors who click links in emails or are explicitly identified will be tracked.">
      <b-switch v-model="data.onsite_tracking.enabled" name="onsite_tracking.enabled" />
    </b-field>

    <div v-if="data.onsite_tracking.enabled">
      <hr />
      <h4 class="title is-5">Tracking Options</h4>

      <b-field label="Track Page Views"
        message="Automatically track page views for identified visitors.">
        <b-switch v-model="data.onsite_tracking.track_page_views" name="onsite_tracking.track_page_views" />
      </b-field>

      <b-field label="Track Product Views"
        message="Automatically detect and track product views on Shopify stores.">
        <b-switch v-model="data.onsite_tracking.track_products" name="onsite_tracking.track_products" />
      </b-field>

      <b-field label="Record IP Address"
        message="Store visitor IP addresses with tracking events. Disable for enhanced privacy.">
        <b-switch v-model="data.onsite_tracking.record_ip_address" name="onsite_tracking.record_ip_address" />
      </b-field>

      <hr />
      <h4 class="title is-5">Domain Restrictions</h4>

      <b-field label="Allowed Domains"
        message="Comma-separated list of domains where tracking is allowed. Leave empty to allow all domains.">
        <b-input v-model="allowedDomainsStr" name="onsite_tracking.allowed_domains"
          placeholder="e.g., example.com, shop.example.com" />
      </b-field>

      <hr />
      <h4 class="title is-5">Cookie Settings</h4>

      <b-field label="Identity Parameter"
        message="URL parameter name for identity tokens in email links.">
        <b-input v-model="data.onsite_tracking.identity_param" name="onsite_tracking.identity_param"
          placeholder="_lmx" />
      </b-field>

      <b-field label="Cookie Name"
        message="Name of the browser cookie used to store visitor identity.">
        <b-input v-model="data.onsite_tracking.cookie_name" name="onsite_tracking.cookie_name"
          placeholder="lm_browser" />
      </b-field>

      <b-field label="Cookie Expiry (days)"
        message="Number of days before the tracking cookie expires.">
        <b-numberinput v-model="data.onsite_tracking.cookie_expiry_days" min="1" max="365" />
      </b-field>

      <hr />
      <h4 class="title is-5">Data Retention</h4>

      <b-field label="Retention Period (days)"
        message="Automatically delete tracking events older than this many days. Set to 0 to keep forever.">
        <b-numberinput v-model="data.onsite_tracking.retention_days" min="0" max="3650" />
      </b-field>

      <hr />
      <h4 class="title is-5">Embed Code</h4>

      <b-field message="Add this script to your website's &lt;head&gt; section to enable tracking.">
        <div class="embed-code">
          <b-input type="textarea" :value="embedCode" readonly />
          <b-button type="is-primary" icon-left="content-copy" @click="copyEmbedCode">
            Copy
          </b-button>
        </div>
      </b-field>
    </div>
  </div>
</template>

<script>
import Vue from 'vue';

export default Vue.extend({
  props: {
    form: {
      type: Object, default: () => { },
    },
  },

  data() {
    return {
      data: this.form,
    };
  },

  computed: {
    allowedDomainsStr: {
      get() {
        return (this.data.onsite_tracking?.allowed_domains || []).join(', ');
      },
      set(val) {
        if (this.data.onsite_tracking) {
          this.data.onsite_tracking.allowed_domains = val
            .split(',')
            .map((d) => d.trim())
            .filter((d) => d !== '');
        }
      },
    },

    embedCode() {
      const rootURL = this.$root.serverConfig?.rootURL || window.location.origin;
      // eslint-disable-next-line no-useless-escape
      return `<script async src="${rootURL}/public/lm.js"><\/script>`;
    },
  },

  methods: {
    copyEmbedCode() {
      navigator.clipboard.writeText(this.embedCode).then(() => {
        this.$utils.toast('Embed code copied to clipboard');
      }).catch(() => {
        this.$utils.toast('Failed to copy embed code', 'is-danger');
      });
    },
  },

  mounted() {
    // Initialize defaults if not set
    if (!this.data.onsite_tracking) {
      this.$set(this.data, 'onsite_tracking', {
        enabled: false,
        track_page_views: true,
        track_products: true,
        allowed_domains: [],
        identity_param: '_lmx',
        cookie_name: 'lm_browser',
        cookie_expiry_days: 365,
        record_ip_address: false,
        retention_days: 90,
      });
    }
  },
});
</script>

<style scoped>
.embed-code {
  display: flex;
  gap: 0.5rem;
  align-items: flex-start;
}
.embed-code .input {
  font-family: monospace;
  font-size: 0.85rem;
}
</style>
