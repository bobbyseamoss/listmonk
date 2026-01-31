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
              <b-field label="Start Date">
                <b-datepicker
                  v-model="startDate"
                  placeholder="Select start date"
                  icon="calendar_today"
                  :max-date="endDate || new Date()"
                  trap-focus />
              </b-field>
            </div>
            <div class="column">
              <b-field label="End Date">
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
      <div class="box mb-4">
        <h5 class="title is-5">Summary</h5>
        <p><strong>Period:</strong> {{ results.start_date }} to {{ results.end_date }}</p>
        <p><strong>Orders:</strong> {{ results.order_count }}</p>
        <p><strong>Total Products:</strong> {{ results.grand_total }}</p>
      </div>

      <div v-if="results.effects && results.effects.length > 0">
        <div v-for="effect in results.effects" :key="effect.effect" class="box mb-4">
          <h4 class="title is-5">{{ effect.effect }}</h4>
          <table class="table is-fullwidth is-striped">
            <thead>
              <tr>
                <th>Flavor</th>
                <th class="has-text-right">Count</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="flavor in effect.flavors" :key="flavor.flavor">
                <td>{{ flavor.flavor }}</td>
                <td class="has-text-right">{{ flavor.count }}</td>
              </tr>
            </tbody>
            <tfoot>
              <tr>
                <th>Total</th>
                <th class="has-text-right">{{ effect.total }}</th>
              </tr>
            </tfoot>
          </table>
        </div>
      </div>

      <div v-else class="notification is-warning">
        No product data found for the selected date range.
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
    // Default to last 30 days
    const end = new Date();
    const start = new Date();
    start.setDate(start.getDate() - 30);

    this.startDate = start;
    this.endDate = end;
  },
});
</script>

<style lang="scss" scoped>
.shopify-order-tally {
  .results {
    .box {
      border-left: 4px solid #3273dc;
    }
  }
}
</style>
