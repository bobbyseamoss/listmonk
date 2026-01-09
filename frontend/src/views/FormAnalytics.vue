<template>
  <section class="form-analytics">
    <header class="columns page-header">
      <div class="column is-10">
        <h1 class="title is-4">
          {{ $t('forms.analytics') }}
          <span v-if="form.name" class="has-text-grey-light"> &mdash; {{ form.name }}</span>
        </h1>
      </div>
      <div class="column has-text-right">
        <b-button icon-left="arrow-left" tag="router-link" :to="{ name: 'signupForms' }">
          {{ $t('globals.buttons.back') }}
        </b-button>
      </div>
    </header>

    <b-loading v-if="loading.form" :is-full-page="false" active />

    <div v-else>
      <!-- Summary Cards -->
      <div class="columns is-multiline mb-5">
        <div class="column is-3">
          <div class="box has-text-centered">
            <p class="heading">Total Impressions</p>
            <p class="title is-2">{{ analytics.totalImpressions || 0 }}</p>
          </div>
        </div>
        <div class="column is-3">
          <div class="box has-text-centered">
            <p class="heading">Total Submissions</p>
            <p class="title is-2">{{ analytics.totalSubmissions || 0 }}</p>
          </div>
        </div>
        <div class="column is-3">
          <div class="box has-text-centered">
            <p class="heading">Conversion Rate</p>
            <p class="title is-2">{{ conversionRate }}%</p>
          </div>
        </div>
        <div class="column is-3">
          <div class="box has-text-centered">
            <p class="heading">Avg. Steps Completed</p>
            <p class="title is-2">{{ analytics.avgStepsCompleted || 0 }}</p>
          </div>
        </div>
      </div>

      <!-- Date Range Selector -->
      <div class="box mb-5">
        <div class="columns is-vcentered">
          <div class="column is-narrow">
            <b-field label="Date Range" horizontal>
              <b-select v-model="dateRange" @input="loadAnalytics">
                <option value="7">Last 7 days</option>
                <option value="30">Last 30 days</option>
                <option value="90">Last 90 days</option>
                <option value="365">Last year</option>
              </b-select>
            </b-field>
          </div>
        </div>
      </div>

      <!-- Charts -->
      <div class="columns">
        <div class="column">
          <div class="box">
            <h3 class="title is-5">Impressions & Submissions</h3>
            <div class="chart-container">
              <canvas ref="mainChart" />
            </div>
          </div>
        </div>
      </div>

      <!-- Device & Source Breakdown -->
      <div class="columns">
        <div class="column is-6">
          <div class="box">
            <h3 class="title is-5">Device Types</h3>
            <div class="chart-container chart-small">
              <canvas ref="deviceChart" />
            </div>
          </div>
        </div>
        <div class="column is-6">
          <div class="box">
            <h3 class="title is-5">Traffic Sources</h3>
            <div class="chart-container chart-small">
              <canvas ref="sourceChart" />
            </div>
          </div>
        </div>
      </div>

      <!-- Submissions Table -->
      <div class="box">
        <h3 class="title is-5">Recent Submissions</h3>
        <b-table
          :data="submissions"
          :loading="loading.submissions"
          hoverable
          :paginated="submissions.length > 20"
          per-page="20"
          default-sort="createdAt"
          default-sort-direction="desc"
        >
          <b-table-column v-slot="props" field="email" label="Email" sortable>
            <router-link :to="{ name: 'subscriber', params: { id: props.row.subscriberId } }">
              {{ props.row.email }}
            </router-link>
          </b-table-column>

          <b-table-column v-slot="props" field="pageUrl" label="Page URL" sortable>
            <span class="is-size-7">{{ props.row.pageUrl || '—' }}</span>
          </b-table-column>

          <b-table-column v-slot="props" field="deviceType" label="Device" sortable width="100">
            <b-icon :icon="deviceIcons[props.row.deviceType] || 'help'" size="is-small" />
            {{ props.row.deviceType || 'Unknown' }}
          </b-table-column>

          <b-table-column v-slot="props" field="utmSource" label="UTM Source" sortable width="120">
            {{ props.row.utmSource || '—' }}
          </b-table-column>

          <b-table-column v-slot="props" field="country" label="Country" sortable width="100">
            {{ props.row.country || '—' }}
          </b-table-column>

          <b-table-column v-slot="props" field="couponCode" label="Coupon" sortable width="120">
            <code v-if="props.row.couponCode">{{ props.row.couponCode }}</code>
            <span v-else class="has-text-grey">—</span>
          </b-table-column>

          <b-table-column v-slot="props" field="createdAt" label="Date" sortable width="150">
            {{ $utils.niceDate(props.row.createdAt) }}
          </b-table-column>

          <template #empty>
            <div class="has-text-centered has-text-grey py-5">
              No submissions yet
            </div>
          </template>
        </b-table>
      </div>
    </div>
  </section>
</template>

<script>
import Vue from 'vue';
import Chart from 'chart.js/auto';

export default Vue.extend({
  name: 'FormAnalytics',

  data() {
    return {
      form: {},
      analytics: {},
      submissions: [],
      dateRange: '30',
      loading: {
        form: true,
        submissions: false,
      },
      charts: {
        main: null,
        device: null,
        source: null,
      },
      deviceIcons: {
        desktop: 'monitor',
        mobile: 'cellphone',
        tablet: 'tablet',
      },
    };
  },

  computed: {
    conversionRate() {
      if (!this.analytics.totalImpressions || this.analytics.totalImpressions === 0) {
        return '0.0';
      }
      return ((this.analytics.totalSubmissions / this.analytics.totalImpressions) * 100).toFixed(1);
    },
  },

  mounted() {
    this.loadForm();
    this.loadAnalytics();
    this.loadSubmissions();
  },

  beforeDestroy() {
    // Clean up charts
    Object.values(this.charts).forEach((chart) => {
      if (chart) chart.destroy();
    });
  },

  methods: {
    async loadForm() {
      this.loading.form = true;
      try {
        this.form = await this.$api.getForm(this.$route.params.id);
      } catch (err) {
        this.$utils.toast(err.message, 'is-danger');
        this.$router.push({ name: 'signupForms' });
      } finally {
        this.loading.form = false;
      }
    },

    async loadAnalytics() {
      try {
        this.analytics = await this.$api.getFormAnalytics(this.$route.params.id, {
          days: this.dateRange,
        });
        this.$nextTick(() => {
          this.renderCharts();
        });
      } catch (err) {
        this.$utils.toast(err.message, 'is-danger');
      }
    },

    async loadSubmissions() {
      this.loading.submissions = true;
      try {
        const resp = await this.$api.getFormSubmissions(this.$route.params.id, {
          per_page: 100,
          order_by: 'created_at',
          order: 'desc',
        });
        this.submissions = resp.data || [];
      } catch (err) {
        this.$utils.toast(err.message, 'is-danger');
      } finally {
        this.loading.submissions = false;
      }
    },

    renderCharts() {
      // Main chart (impressions & submissions over time)
      if (this.charts.main) this.charts.main.destroy();
      const mainCtx = this.$refs.mainChart;
      if (mainCtx && this.analytics.dailyStats) {
        this.charts.main = new Chart(mainCtx, {
          type: 'line',
          data: {
            labels: this.analytics.dailyStats.map((d) => d.date),
            datasets: [
              {
                label: 'Impressions',
                data: this.analytics.dailyStats.map((d) => d.impressions),
                borderColor: '#0055d4',
                backgroundColor: 'rgba(0, 85, 212, 0.1)',
                fill: true,
              },
              {
                label: 'Submissions',
                data: this.analytics.dailyStats.map((d) => d.submissions),
                borderColor: '#48c774',
                backgroundColor: 'rgba(72, 199, 116, 0.1)',
                fill: true,
              },
            ],
          },
          options: {
            responsive: true,
            maintainAspectRatio: false,
          },
        });
      }

      // Device chart
      if (this.charts.device) this.charts.device.destroy();
      const deviceCtx = this.$refs.deviceChart;
      if (deviceCtx && this.analytics.deviceBreakdown) {
        this.charts.device = new Chart(deviceCtx, {
          type: 'doughnut',
          data: {
            labels: Object.keys(this.analytics.deviceBreakdown),
            datasets: [{
              data: Object.values(this.analytics.deviceBreakdown),
              backgroundColor: ['#0055d4', '#48c774', '#ffdd57'],
            }],
          },
          options: {
            responsive: true,
            maintainAspectRatio: false,
          },
        });
      }

      // Source chart
      if (this.charts.source) this.charts.source.destroy();
      const sourceCtx = this.$refs.sourceChart;
      if (sourceCtx && this.analytics.sourceBreakdown) {
        this.charts.source = new Chart(sourceCtx, {
          type: 'bar',
          data: {
            labels: Object.keys(this.analytics.sourceBreakdown).slice(0, 10),
            datasets: [{
              label: 'Submissions',
              data: Object.values(this.analytics.sourceBreakdown).slice(0, 10),
              backgroundColor: '#0055d4',
            }],
          },
          options: {
            responsive: true,
            maintainAspectRatio: false,
            indexAxis: 'y',
          },
        });
      }
    },
  },
});
</script>

<style lang="scss" scoped>
.form-analytics {
  .chart-container {
    position: relative;
    height: 300px;

    &.chart-small {
      height: 250px;
    }
  }

  .box {
    margin-bottom: 1rem;
  }

  .heading {
    font-size: 0.75rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: #7a7a7a;
    margin-bottom: 0.5rem;
  }
}
</style>
