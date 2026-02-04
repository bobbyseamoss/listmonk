<template>
  <section class="shopify-order-tally">
    <header class="columns page-header">
      <div class="column is-half">
        <h1 class="title is-4">Shopify Order Tally</h1>
      </div>
    </header>
    <hr />

    <div class="columns">
      <div class="column is-6">
        <div class="box">
          <h5 class="title is-5">Select Date Range</h5>
          <div class="columns">
            <div class="column">
              <b-field label="From">
                <b-datepicker
                  v-model="startDate"
                  placeholder="Select start date"
                  icon="calendar_today"
                  :max-date="endDate || new Date()"
                  trap-focus />
              </b-field>
            </div>
            <div class="column">
              <b-field label="To">
                <b-datepicker
                  v-model="endDate"
                  placeholder="Select end date"
                  icon="calendar_today"
                  :min-date="startDate"
                  :max-date="new Date()"
                  trap-focus />
              </b-field>
            </div>
          </div>
          <b-button
            type="is-primary"
            icon-left="magnify"
            :loading="loading"
            :disabled="!startDate || !endDate"
            @click="fetchOrderTally">
            Generate Report
          </b-button>
        </div>
      </div>
    </div>

    <div v-if="error" class="notification is-danger">
      {{ error }}
    </div>

    <div v-if="results" class="results mt-5">
      <div class="tally-output">
        <div class="tally-header">
          {{ results.start_date }} to {{ results.end_date }}
          Orders: {{ results.order_count }} | Total: {{ results.grand_total }}
        </div>

        <div v-if="results.effects && results.effects.length > 0">
          <div v-for="effect in results.effects" :key="effect.effect" class="tally-section">
            <div class="tally-effect">{{ effect.effect }}</div>
            <div v-for="flavor in effect.flavors" :key="flavor.flavor" class="tally-line">
              {{ flavor.flavor }} {{ flavor.count }}
            </div>
            <div class="tally-total">Total {{ effect.total }}</div>
          </div>
        </div>

        <div v-else class="tally-line">
          No product data found for the selected date range.
        </div>
      </div>
    </div>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import * as api from '../api';

export default Vue.extend({
  name: 'ShopifyOrderTally',

  data() {
    return {
      startDate: null,
      endDate: null,
      loading: false,
      error: null,
      results: null,
    };
  },

  computed: {
    ...mapState(['settings']),
  },

  methods: {
    formatDate(date) {
      if (!date) return '';
      const d = new Date(date);
      const year = d.getFullYear();
      const month = String(d.getMonth() + 1).padStart(2, '0');
      const day = String(d.getDate()).padStart(2, '0');
      return `${year}-${month}-${day}`;
    },

    async fetchOrderTally() {
      if (!this.startDate || !this.endDate) {
        this.error = 'Please select both start and end dates';
        return;
      }

      this.loading = true;
      this.error = null;
      this.results = null;

      try {
        const params = {
          start_date: this.formatDate(this.startDate),
          end_date: this.formatDate(this.endDate),
        };
        this.results = await api.getShopifyOrderTally(params);
      } catch (err) {
        this.error = err.response?.data?.message || err.message || 'Error fetching order tally';
      } finally {
        this.loading = false;
      }
    },
  },

  mounted() {
    // Default: From yesterday To today
    const today = new Date();
    const yesterday = new Date();
    yesterday.setDate(yesterday.getDate() - 1);

    this.startDate = yesterday;
    this.endDate = today;
  },
});
</script>

<style lang="scss" scoped>
.shopify-order-tally {
  .tally-output {
    font-family: monospace;
    font-size: 14px;
    line-height: 1.6;
    background: #fafafa;
    border: 1px solid #ddd;
    padding: 20px;
    white-space: pre-wrap;
  }

  .tally-header {
    margin-bottom: 20px;
    padding-bottom: 10px;
    border-bottom: 1px solid #ccc;
    color: #666;
  }

  .tally-section {
    margin-bottom: 20px;
  }

  .tally-effect {
    font-weight: bold;
    text-transform: uppercase;
    margin-bottom: 4px;
  }

  .tally-line {
    padding-left: 0;
  }

  .tally-total {
    font-weight: bold;
    margin-top: 4px;
  }
}
</style>
