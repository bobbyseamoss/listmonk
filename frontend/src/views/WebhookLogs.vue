<template>
  <section class="webhook-logs">
    <header class="page-header columns">
      <div class="column is-two-thirds">
        <h1 class="title is-4">
          Webhook Logs
          <span v-if="logs.total > 0">({{ logs.total }})</span>
        </h1>
      </div>
      <div class="column has-text-right">
        <b-button @click="exportLogs('csv')" type="is-primary" icon-left="download" :loading="loading.export">
          Export CSV
        </b-button>
        <b-button @click="clearAllLogs" type="is-danger" icon-left="delete" :loading="loading.clearAll" class="ml-2">
          Clear All
        </b-button>
      </div>
    </header>

    <!-- Filters -->
    <div class="columns filters mb-4">
      <div class="column is-3">
        <b-field label="Status Filter">
          <b-select v-model="queryParams.statusFilter" expanded @input="onFilterChange">
            <option value="">All</option>
            <option value="delivered">Delivered</option>
            <option value="failed">Failed (Bounced, Suppressed, etc.)</option>
            <option value="views">Views</option>
            <option value="clicks">Clicks</option>
          </b-select>
        </b-field>
      </div>
      <div class="column is-3">
        <b-field label="Webhook Source">
          <b-select v-model="queryParams.webhookType" expanded @input="onFilterChange">
            <option value="">All</option>
            <option value="azure">Azure</option>
            <option value="ses">SES</option>
            <option value="sendgrid">SendGrid</option>
            <option value="postmark">Postmark</option>
          </b-select>
        </b-field>
      </div>
      <div class="column is-3">
        <b-field label="&nbsp;">
          <b-button v-if="hasActiveFilters" @click="clearFilters" type="is-light" icon-left="close">
            Clear Filters
          </b-button>
        </b-field>
      </div>
    </div>

    <b-table :data="logs.results" :hoverable="true" :loading="loading.logs" default-sort="createdAt" checkable
      @check-all="onTableCheck" @check="onTableCheck" :checked-rows.sync="bulk.checked" detailed show-detail-icon
      paginated backend-pagination pagination-position="both" @page-change="onPageChange"
      :current-page="queryParams.page" :per-page="logs.perPage" :total="logs.total">
      <template #top-left>
        <div class="actions">
          <template v-if="bulk.checked.length > 0">
            <a class="a" href="#" @click.prevent="$utils.confirm(null, () => deleteLogs())" data-cy="btn-delete">
              <b-icon icon="trash-can-outline" size="is-small" /> {{ $t('globals.buttons.delete') }}
            </a>
            <span>
              {{ $t('globals.messages.numSelected', { num: bulk.checked.length }) }}
              <span v-if="!bulk.all && logs.total > logs.perPage">
                &mdash;
                <a href="#" @click.prevent="selectAllLogs">
                  Select all {{ logs.total }}
                </a>
              </span>
            </span>
          </template>
        </div>
      </template>

      <b-table-column v-slot="props" field="webhook_type" label="Webhook Source">
        <b-tag>{{ props.row.webhookType }}</b-tag>
      </b-table-column>

      <b-table-column v-slot="props" field="event_status" label="Status">
        <b-tag :type="getStatusTagType(props.row)">
          {{ getEventStatus(props.row) }}
        </b-tag>
      </b-table-column>

      <b-table-column v-slot="props" field="created_at" label="Created At">
        {{ $utils.niceDate(props.row.createdAt, true) }}
      </b-table-column>

      <b-table-column v-slot="props" cell-class="actions" align="right">
        <div>
          <a href="#" @click.prevent="$utils.confirm(null, () => deleteLog(props.row))"
            data-cy="btn-delete" :aria-label="$t('globals.buttons.delete')">
            <b-tooltip :label="$t('globals.buttons.delete')" type="is-dark">
              <b-icon icon="trash-can-outline" size="is-small" />
            </b-tooltip>
          </a>
        </div>
      </b-table-column>

      <template #detail="props">
        <div class="box">
          <div class="mb-4" v-if="props.row.errorMessage">
            <h4 class="title is-6 has-text-danger">Error</h4>
            <pre class="is-size-7 has-text-danger">{{ props.row.errorMessage }}</pre>
          </div>

          <div class="mb-4" v-if="props.row.requestHeaders && Object.keys(props.row.requestHeaders).length > 0">
            <h4 class="title is-6">Request Headers</h4>
            <pre class="is-size-7">{{ JSON.stringify(props.row.requestHeaders, null, 2) }}</pre>
          </div>

          <div class="mb-4">
            <h4 class="title is-6">Request Body</h4>
            <pre class="is-size-7">{{ formatJSON(props.row.requestBody) }}</pre>
          </div>
        </div>
      </template>

      <template #empty v-if="!loading.logs">
        <empty-placeholder />
      </template>
    </b-table>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import EmptyPlaceholder from '../components/EmptyPlaceholder.vue';

export default Vue.extend({
  components: {
    EmptyPlaceholder,
  },

  data() {
    return {
      logs: {
        results: [],
        total: 0,
        perPage: 20,
      },

      loading: {
        logs: false,
        export: false,
        clearAll: false,
      },

      // Table bulk row selection states.
      bulk: {
        checked: [],
        all: false,
      },

      // Query params to filter the API call.
      queryParams: {
        page: 1,
        webhookType: '',
        eventType: '',
        statusFilter: '',
      },
    };
  },

  methods: {
    formatJSON(str) {
      try {
        return JSON.stringify(JSON.parse(str), null, 2);
      } catch (e) {
        return str;
      }
    },

    // Parse webhook JSON and extract status or engagementType
    getEventStatus(row) {
      // SES webhooks store event type in the database column (camelCase from API)
      if (row.webhookType === 'ses' && row.eventType) {
        return row.eventType;
      }

      try {
        const body = JSON.parse(row.requestBody);

        // Azure webhooks come as an array with eventType
        if (Array.isArray(body) && body.length > 0) {
          const event = body[0];

          // Azure delivery reports
          if (event.eventType === 'Microsoft.Communication.EmailDeliveryReportReceived') {
            return event.data?.status || '-';
          }

          // Azure engagement reports
          if (event.eventType === 'Microsoft.Communication.EmailEngagementTrackingReportReceived') {
            return event.data?.engagementType || '-';
          }

          // SendGrid webhooks - array of events with 'event' field
          if (event.event) {
            // Capitalize first letter for display consistency
            const sgEvent = event.event;
            return sgEvent.charAt(0).toUpperCase() + sgEvent.slice(1);
          }
        }

        // Fallback to HTTP response status
        return row.responseStatus || '-';
      } catch (e) {
        // If parsing fails, show HTTP response status
        return row.responseStatus || '-';
      }
    },

    // Determine tag color based on status
    getStatusTagType(row) {
      const status = this.getEventStatus(row);
      const statusLower = typeof status === 'string' ? status.toLowerCase() : '';

      // Success/Delivered statuses (green)
      if (status === 'Delivered' || statusLower === 'delivered' || statusLower === 'delivery') return 'is-success';
      if (statusLower === 'processed') return 'is-success';

      // Failure statuses (red)
      if (['Bounced', 'Failed', 'Suppressed', 'Quarantined', 'FilteredSpam'].includes(status)) {
        return 'is-danger';
      }
      if (['bounce', 'dropped', 'blocked', 'spamreport', 'unsubscribe', 'complaint', 'reject', 'renderingfailure'].includes(statusLower)) {
        return 'is-danger';
      }

      // Warning/Pending statuses (yellow/warning)
      if (statusLower === 'deferred' || statusLower === 'deliverydelay') return 'is-warning';

      // Send/Accepted status (grey/light) - email accepted but not yet delivered
      if (statusLower === 'send') return 'is-light';

      // Engagement types (blue/info for opens, purple/link for clicks)
      if (status === 'view' || statusLower === 'open') return 'is-info';
      if (status === 'click' || statusLower === 'click') return 'is-link';

      // HTTP status codes
      if (typeof status === 'number') {
        return status >= 200 && status < 300 ? 'is-success' : 'is-danger';
      }

      // Default
      return 'is-light';
    },

    onPageChange(p) {
      this.queryParams.page = p;
      this.getLogs();
    },

    onFilterChange() {
      this.queryParams.page = 1; // Reset to first page when filtering
      this.bulk.checked = [];
      this.bulk.all = false;
      this.getLogs();
    },

    clearFilters() {
      this.queryParams.webhookType = '';
      this.queryParams.eventType = '';
      this.queryParams.statusFilter = '';
      this.queryParams.page = 1;
      this.bulk.checked = [];
      this.bulk.all = false;
      this.getLogs();
    },

    // Mark all logs in the query as selected.
    selectAllLogs() {
      this.bulk.all = true;
    },

    onTableCheck() {
      // Disable bulk.all selection if there are no rows checked in the table.
      if (this.bulk.checked.length !== this.logs.total) {
        this.bulk.all = false;
      }
    },

    getLogs() {
      this.loading.logs = true;

      const params = {
        page: this.queryParams.page,
      };

      if (this.queryParams.webhookType) {
        params.webhook_type = this.queryParams.webhookType;
      }
      if (this.queryParams.eventType) {
        params.event_type = this.queryParams.eventType;
      }
      if (this.queryParams.statusFilter) {
        params.status_filter = this.queryParams.statusFilter;
      }

      this.$api.getWebhookLogs(params).then((data) => {
        this.logs = data;
        this.loading.logs = false;
      });
    },

    deleteLog(log) {
      this.$api.deleteWebhookLogs([log.id]).then(() => {
        this.$utils.toast('Webhook log deleted');
        this.getLogs();
      });
    },

    deleteLogs() {
      const ids = this.bulk.all ? 'all' : this.bulk.checked.map((l) => l.id);
      this.$api.deleteWebhookLogs(ids).then(() => {
        this.$utils.toast('Webhook logs deleted');
        this.bulk.checked = [];
        this.bulk.all = false;
        this.getLogs();
      });
    },

    exportLogs(format = 'csv') {
      this.loading.export = true;

      // Build params including current filters
      const params = { format };
      if (this.queryParams.webhookType) {
        params.webhook_type = this.queryParams.webhookType;
      }
      if (this.queryParams.eventType) {
        params.event_type = this.queryParams.eventType;
      }
      if (this.queryParams.statusFilter) {
        params.status_filter = this.queryParams.statusFilter;
      }

      this.$api.exportWebhookLogs(params).then((blob) => {
        // The blob is already returned from the API
        const url = window.URL.createObjectURL(blob);
        const link = document.createElement('a');
        link.href = url;
        const ext = format === 'csv' ? 'csv' : 'json';
        const filterSuffix = this.queryParams.statusFilter ? `-${this.queryParams.statusFilter}` : '';
        link.download = `webhook-logs${filterSuffix}-${new Date().toISOString().split('T')[0]}.${ext}`;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        window.URL.revokeObjectURL(url);

        this.$utils.toast('Webhook logs exported');
        this.loading.export = false;
      }).catch(() => {
        this.loading.export = false;
      });
    },

    clearAllLogs() {
      this.$utils.confirm(
        'Are you sure you want to delete all webhook logs? This action cannot be undone.',
        () => {
          this.loading.clearAll = true;
          this.$api.deleteWebhookLogs('all').then(() => {
            this.$utils.toast('All webhook logs deleted');
            this.bulk.checked = [];
            this.bulk.all = false;
            this.loading.clearAll = false;
            this.getLogs();
          }).catch(() => {
            this.loading.clearAll = false;
          });
        },
      );
    },
  },

  computed: {
    ...mapState(['loading']),

    hasActiveFilters() {
      return this.queryParams.webhookType !== ''
             || this.queryParams.eventType !== ''
             || this.queryParams.statusFilter !== '';
    },
  },

  mounted() {
    this.getLogs();
  },
});
</script>

<style scoped>
.webhook-logs pre {
  max-height: 400px;
  overflow: auto;
}
</style>
