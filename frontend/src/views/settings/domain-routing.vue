<template>
  <div class="items">
    <h5 class="title is-5">Domain Routing</h5>
    <p class="help mb-3">
      Configure how emails are routed for each domain. Select Azure to send through Azure Communication Services,
      SendGrid to send through SendGrid, or Disabled to block sending to this domain.
    </p>

    <div class="domain-routing-list">
      <table class="table is-fullwidth is-striped">
        <thead>
          <tr>
            <th>Domain</th>
            <th class="has-text-centered">Azure</th>
            <th class="has-text-centered">SendGrid</th>
            <th class="has-text-centered">Disabled</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(item, index) in domainList" :key="item.domain">
            <td>
              <strong>{{ item.domain }}</strong>
            </td>
            <td class="has-text-centered">
              <input
                type="radio"
                :checked="item.routing === 'azure'"
                :name="'routing_' + index"
                :aria-label="'Azure routing for ' + item.domain"
                @change="setRouting(index, 'azure')" />
            </td>
            <td class="has-text-centered">
              <input
                type="radio"
                :checked="item.routing === 'sendgrid'"
                :name="'routing_' + index"
                :aria-label="'SendGrid routing for ' + item.domain"
                @change="setRouting(index, 'sendgrid')" />
            </td>
            <td class="has-text-centered">
              <input
                type="radio"
                :checked="item.routing === 'disabled'"
                :name="'routing_' + index"
                :aria-label="'Disabled routing for ' + item.domain"
                @change="setRouting(index, 'disabled')" />
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <b-notification type="is-info" :closable="false" class="mt-4">
      <strong>How Domain Routing Works:</strong>
      <ul class="mt-2">
        <li><b-icon icon="arrow-right" size="is-small" /> <strong>Azure:</strong> Emails sent through Azure Communication Services (smtp.azurecomm.net)</li>
        <li><b-icon icon="arrow-right" size="is-small" /> <strong>SendGrid:</strong> Emails sent through SendGrid (smtp.sendgrid.net)</li>
        <li><b-icon icon="arrow-right" size="is-small" /> <strong>Disabled:</strong> Emails to this domain are blocked from sending</li>
        <li><b-icon icon="arrow-right" size="is-small" /> When multiple servers of the same type exist, round-robin is used</li>
      </ul>
    </b-notification>

    <div class="buttons mt-4">
      <b-button type="is-light" size="is-small" @click="setAllAzure">Set All to Azure</b-button>
      <b-button type="is-light" size="is-small" @click="setAllSendgrid">Set All to SendGrid</b-button>
      <b-button type="is-light" size="is-small" @click="resetDefaults">Reset to Defaults</b-button>
    </div>
  </div>
</template>

<script>
import Vue from 'vue';

// Top email domains (US-based, sorted by subscriber count)
const TOP_DOMAINS = [
  'gmail.com',
  'yahoo.com',
  'hotmail.com',
  'icloud.com',
  'aol.com',
  'outlook.com',
  'comcast.net',
  'live.com',
  'msn.com',
  'me.com',
  'ymail.com',
  'att.net',
  'sbcglobal.net',
  'verizon.net',
  'bellsouth.net',
  'mac.com',
  'rocketmail.com',
  'cox.net',
  'charter.net',
  'mail.com',
  'aim.com',
  'gmx.com',
  'netscape.net',
  'myyahoo.com',
  'googlemail.com',
  'earthlink.net',
];

// Default routing: gmail.com -> sendgrid, all others -> azure
function getDefaultRouting(domain) {
  return domain === 'gmail.com' ? 'sendgrid' : 'azure';
}

export default Vue.extend({
  props: {
    form: {
      type: Object, default: () => ({}),
    },
  },

  data() {
    return {
      data: this.form,
      domainList: [],
    };
  },

  watch: {
    domainList: {
      deep: true,
      handler() {
        this.syncToForm();
      },
    },
  },

  mounted() {
    this.initializeFromSettings();
  },

  methods: {
    initializeFromSettings() {
      const savedRouting = this.data.domain_routing || {};
      const list = [];

      TOP_DOMAINS.forEach((domain) => {
        let routing = getDefaultRouting(domain);
        if (savedRouting[domain] && typeof savedRouting[domain] === 'string') {
          routing = savedRouting[domain];
        }
        list.push({ domain, routing });
      });

      this.domainList = list;
    },

    setRouting(index, value) {
      Vue.set(this.domainList, index, {
        ...this.domainList[index],
        routing: value,
      });
    },

    syncToForm() {
      const routingObj = {};
      this.domainList.forEach((item) => {
        routingObj[item.domain] = item.routing;
      });
      this.$set(this.data, 'domain_routing', routingObj);
    },

    setAllAzure() {
      this.domainList = this.domainList.map((item) => ({
        ...item,
        routing: 'azure',
      }));
    },

    setAllSendgrid() {
      this.domainList = this.domainList.map((item) => ({
        ...item,
        routing: 'sendgrid',
      }));
    },

    resetDefaults() {
      this.domainList = this.domainList.map((item) => ({
        ...item,
        routing: getDefaultRouting(item.domain),
      }));
    },
  },
});
</script>

<style scoped>
.domain-routing-list .table {
  border: 1px solid #dbdbdb;
}

.domain-routing-list .table th {
  background-color: #f5f5f5;
}

.domain-routing-list .table td {
  vertical-align: middle;
}

.domain-routing-list input[type="radio"] {
  width: 18px;
  height: 18px;
  cursor: pointer;
}

.buttons {
  gap: 0.5rem;
}
</style>
