<template>
  <section class="azure-analytics">
    <b-loading :active="loading" />

    <div v-if="!loading && analytics">
      <!-- Summary Stats -->
      <div class="columns">
        <div class="column">
          <h4 class="title is-5">Azure Event Grid Analytics</h4>
        </div>
      </div>

      <!-- Delivery Status Summary -->
      <div class="columns">
        <div class="column is-6">
          <div class="box">
            <h5 class="subtitle is-6">Delivery Status</h5>
            <div v-if="analytics.delivery_stats && analytics.delivery_stats.length > 0">
              <div v-for="stat in analytics.delivery_stats" :key="stat.status" class="level is-mobile">
                <div class="level-left">
                  <span class="has-text-weight-semibold">{{ stat.status }}:</span>
                </div>
                <div class="level-right">
                  <span>{{ stat.count }}</span>
                </div>
              </div>
            </div>
            <p v-else class="has-text-grey-light">No delivery events yet</p>
          </div>
        </div>

        <div class="column is-6">
          <div class="box">
            <h5 class="subtitle is-6">Engagement</h5>
            <div v-if="analytics.engagement_stats && analytics.engagement_stats.length > 0">
              <div v-for="stat in analytics.engagement_stats" :key="stat.type" class="level is-mobile">
                <div class="level-left">
                  <span class="has-text-weight-semibold">{{ stat.type }}:</span>
                </div>
                <div class="level-right">
                  <span>{{ stat.count }}</span>
                </div>
              </div>
            </div>
            <p v-else class="has-text-grey-light">No engagement events yet</p>
          </div>
        </div>
      </div>

      <!-- Delivery Events Table -->
      <div class="box mt-4">
        <h5 class="subtitle is-6">Recent Delivery Events</h5>
        <b-table
          :data="deliveryEvents"
          :loading="loadingDelivery"
          :hoverable="true"
          :paginated="deliveryEvents.length > 10"
          :per-page="10"
          default-sort="event_timestamp"
          default-sort-direction="desc"
        >
          <b-table-column field="status" label="Status" sortable v-slot="props">
            <b-tag :type="getStatusType(props.row.status)">
              {{ props.row.status }}
            </b-tag>
          </b-table-column>

          <b-table-column field="status_reason" label="Reason" v-slot="props">
            <span class="is-size-7">{{ props.row.status_reason || '-' }}</span>
          </b-table-column>

          <b-table-column field="event_timestamp" label="Time" sortable v-slot="props">
            {{ $utils.niceDate(props.row.event_timestamp, true) }}
          </b-table-column>

          <template #empty>
            <section class="section">
              <div class="content has-text-grey has-text-centered">
                <p>No delivery events found</p>
              </div>
            </section>
          </template>
        </b-table>
      </div>

      <!-- Engagement Events Table -->
      <div class="box mt-4">
        <h5 class="subtitle is-6">Recent Engagement Events</h5>
        <b-table
          :data="engagementEvents"
          :loading="loadingEngagement"
          :hoverable="true"
          :paginated="engagementEvents.length > 10"
          :per-page="10"
          default-sort="event_timestamp"
          default-sort-direction="desc"
        >
          <b-table-column field="engagement_type" label="Type" sortable v-slot="props">
            <b-tag :type="props.row.engagement_type === 'view' ? 'is-info' : 'is-success'">
              {{ props.row.engagement_type }}
            </b-tag>
          </b-table-column>

          <b-table-column field="engagement_context" label="Context" v-slot="props">
            <span class="is-size-7" style="word-break: break-all;">
              {{ props.row.engagement_context || '-' }}
            </span>
          </b-table-column>

          <b-table-column field="event_timestamp" label="Time" sortable v-slot="props">
            {{ $utils.niceDate(props.row.event_timestamp, true) }}
          </b-table-column>

          <template #empty>
            <section class="section">
              <div class="content has-text-grey has-text-centered">
                <p>No engagement events found</p>
              </div>
            </section>
          </template>
        </b-table>
      </div>
    </div>
  </section>
</template>

<script>
export default {
  name: 'CampaignAzureAnalytics',

  props: {
    campaignId: {
      type: Number,
      required: true,
    },
  },

  data() {
    return {
      loading: false,
      loadingDelivery: false,
      loadingEngagement: false,
      analytics: null,
      deliveryEvents: [],
      engagementEvents: [],
    };
  },

  mounted() {
    this.loadData();
  },

  methods: {
    async loadData() {
      this.loading = true;
      try {
        // Load summary analytics
        const analyticsData = await this.$api.getCampaignAzureAnalytics(this.campaignId);
        this.analytics = (analyticsData && analyticsData.data) || {};

        // Load recent delivery events
        this.loadingDelivery = true;
        const deliveryData = await this.$api.getCampaignAzureDeliveryEvents(
          this.campaignId,
          { per_page: 50 },
        );
        this.deliveryEvents = (deliveryData && deliveryData.data && deliveryData.data.results) || [];
        this.loadingDelivery = false;

        // Load recent engagement events
        this.loadingEngagement = true;
        const engagementData = await this.$api.getCampaignAzureEngagementEvents(
          this.campaignId,
          { per_page: 50 },
        );
        this.engagementEvents = (engagementData && engagementData.data && engagementData.data.results) || [];
        this.loadingEngagement = false;
      } catch (error) {
        this.$utils.toast(error.toString(), 'is-danger');
      } finally {
        this.loading = false;
      }
    },

    getStatusType(status) {
      switch (status) {
        case 'Delivered':
          return 'is-success';
        case 'Bounced':
        case 'Failed':
          return 'is-danger';
        case 'Quarantined':
        case 'FilteredSpam':
          return 'is-warning';
        case 'Suppressed':
          return 'is-light';
        default:
          return 'is-info';
      }
    },
  },
};
</script>

<style scoped>
.azure-analytics {
  padding: 1rem;
}
</style>
