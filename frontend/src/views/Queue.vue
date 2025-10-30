<template>
  <section class="queue">
    <header class="columns page-header">
      <div class="column is-6">
        <h1 class="title is-4">
          Queue
          <span v-if="!isNaN(queueItems.total)">({{ queueItems.total }})</span>
        </h1>
      </div>
      <div class="column has-text-right">
        <b-button
          @click="togglePause"
          :icon-left="queuePaused ? 'play' : 'pause'"
          :type="queuePaused ? 'is-success' : 'is-warning'"
          :loading="loading.queue"
          class="mr-2"
        >
          {{ queuePaused ? 'Resume' : 'Pause' }} Queue
        </b-button>
        <b-button
          @click="confirmClearQueue"
          icon-left="delete-sweep"
          type="is-danger"
          :loading="loading.queue"
          class="mr-2"
        >
          Clear Queue
        </b-button>
        <b-button
          @click="confirmSendAll"
          icon-left="send"
          type="is-info"
          :loading="loading.queue"
          class="mr-2"
        >
          Send All
        </b-button>
        <b-button @click="refresh" icon-left="refresh" :loading="loading.queue">
          Refresh
        </b-button>
      </div>
    </header>

    <!-- Summary Statistics -->
    <section class="stats-section">
      <div class="columns">
        <div class="column">
          <div class="box has-text-centered">
            <p class="heading">Queued</p>
            <p class="title">{{ stats.queued }}</p>
          </div>
        </div>
        <div class="column">
          <div class="box has-text-centered">
            <p class="heading">Sending</p>
            <p class="title">{{ stats.sending }}</p>
          </div>
        </div>
        <div class="column">
          <div class="box has-text-centered">
            <p class="heading">Sent</p>
            <p class="title">{{ stats.sent }}</p>
          </div>
        </div>
        <div class="column">
          <div class="box has-text-centered">
            <p class="heading">Failed</p>
            <p class="title has-text-danger">{{ stats.failed }}</p>
          </div>
        </div>
        <div class="column">
          <div class="box has-text-centered">
            <p class="heading">Cancelled</p>
            <p class="title has-text-grey">{{ stats.cancelled }}</p>
          </div>
        </div>
      </div>

      <div class="columns">
        <div class="column is-6">
          <div class="box">
            <p class="heading">Next Scheduled Email</p>
            <p class="title is-5" v-if="stats.nextScheduledAt">
              {{ formatDate(stats.nextScheduledAt) }}
              <span class="is-size-6 has-text-grey">
                ({{ countdown(stats.nextScheduledAt) }})
              </span>
            </p>
            <p class="title is-6 has-text-grey" v-else-if="stats.queued > 0">
              Processing queue ({{ stats.queued }} emails ready)
            </p>
            <p class="title is-6 has-text-grey" v-else>No emails scheduled</p>
          </div>
        </div>
        <div class="column is-6">
          <div class="box">
            <p class="heading">Processing Status</p>
            <p class="title is-5">
              <b-tag :type="stats.sending > 0 ? 'is-success' : 'is-light'">
                {{ stats.sending > 0 ? 'Active' : 'Idle' }}
              </b-tag>
              <span v-if="stats.sending > 0" class="is-size-6 has-text-grey ml-2">
                ({{ stats.sending }} emails sending)
              </span>
            </p>
          </div>
        </div>
      </div>
    </section>

    <!-- Filters -->
    <section class="filters-section mb-4">
      <div class="columns">
        <div class="column is-3">
          <b-field label="Campaign">
            <b-input v-model="filters.campaignId" placeholder="Campaign ID" type="number" />
          </b-field>
        </div>
        <div class="column is-3">
          <b-field label="Status">
            <b-select v-model="filters.status" placeholder="All statuses" expanded multiple>
              <option value="queued">Queued</option>
              <option value="sending">Sending</option>
              <option value="sent">Sent</option>
              <option value="failed">Failed</option>
              <option value="cancelled">Cancelled</option>
            </b-select>
          </b-field>
        </div>
        <div class="column is-3">
          <b-field label="Subscriber Email">
            <b-input v-model="filters.subscriber" placeholder="Search email" />
          </b-field>
        </div>
        <div class="column is-3">
          <b-field label="SMTP Server">
            <b-select v-model="filters.smtpServerUuid" placeholder="All servers" expanded>
              <option value="">All servers</option>
              <option v-for="server in smtpServers" :key="server.uuid" :value="server.uuid">
                {{ server.name || server.uuid }}
              </option>
            </b-select>
          </b-field>
        </div>
      </div>
      <div class="columns">
        <div class="column">
          <b-button @click="applyFilters" type="is-primary" icon-left="filter">
            Apply Filters
          </b-button>
          <b-button @click="clearFilters" class="ml-2">
            Clear
          </b-button>
        </div>
      </div>
    </section>

    <!-- Queue Items Table -->
    <b-table :data="queueItems.results" :loading="loading.queue" paginated backend-pagination
      pagination-position="both" @page-change="onPageChange" :current-page="queryParams.page"
      :per-page="queueItems.perPage" :total="queueItems.total" hoverable>

      <b-table-column v-slot="props" field="id" label="ID" width="5%">
        {{ props.row.id }}
      </b-table-column>

      <b-table-column v-slot="props" field="campaign_name" label="Campaign" width="15%">
        <router-link :to="{ name: 'campaign', params: { id: props.row.campaignId } }">
          {{ props.row.campaignName }}
        </router-link>
      </b-table-column>

      <b-table-column v-slot="props" field="subscriber_email" label="Subscriber" width="15%">
        <router-link :to="{ name: 'subscriber', params: { id: props.row.subscriberId } }">
          {{ props.row.subscriberEmail }}
        </router-link>
      </b-table-column>

      <b-table-column v-slot="props" field="status" label="Status" width="10%">
        <b-tag :type="getStatusType(props.row.status)">
          {{ props.row.status }}
        </b-tag>
      </b-table-column>

      <b-table-column v-slot="props" field="priority" label="Priority" width="8%">
        {{ getPriorityLabel(props.row.priority) }}
      </b-table-column>

      <b-table-column v-slot="props" field="scheduled_at" label="Scheduled At" width="12%">
        <span v-if="props.row.scheduledAt">
          {{ formatDate(props.row.scheduledAt) }}
        </span>
        <span v-else class="has-text-grey">-</span>
      </b-table-column>

      <b-table-column v-slot="props" field="sent_at" label="Sent At" width="12%">
        <span v-if="props.row.sentAt">
          {{ formatDate(props.row.sentAt) }}
        </span>
        <span v-else class="has-text-grey">-</span>
      </b-table-column>

      <b-table-column v-slot="props" field="assigned_smtp_server_uuid" label="SMTP Server" width="10%">
        <span v-if="props.row.assignedSmtpServerUuid">
          {{ getServerName(props.row.assignedSmtpServerUuid) }}
        </span>
        <span v-else class="has-text-grey">Not assigned</span>
      </b-table-column>

      <b-table-column v-slot="props" field="retry_count" label="Retries" width="5%">
        {{ props.row.retryCount }}
      </b-table-column>

      <b-table-column v-slot="props" field="error_message" label="Error" width="15%">
        <span v-if="props.row.errorMessage" class="has-text-danger" :title="props.row.errorMessage">
          {{ truncate(props.row.errorMessage, 30) }}
        </span>
        <span v-else class="has-text-grey">-</span>
      </b-table-column>

      <b-table-column v-slot="props" label="Actions" width="8%">
        <div class="actions">
          <a v-if="canCancel(props.row)" href="#"
            @click.prevent="$utils.confirm('Cancel this email?', () => cancelItem(props.row.id))"
            :aria-label="'Cancel'">
            <b-tooltip label="Cancel" type="is-dark">
              <b-icon icon="cancel" size="is-small" />
            </b-tooltip>
          </a>

          <a v-if="canRetry(props.row)" href="#"
            @click.prevent="$utils.confirm('Retry this email?', () => retryItem(props.row.id))"
            :aria-label="'Retry'">
            <b-tooltip label="Retry" type="is-dark">
              <b-icon icon="refresh" size="is-small" />
            </b-tooltip>
          </a>
        </div>
      </b-table-column>

      <template #empty v-if="!loading.queue">
        <empty-placeholder />
      </template>
    </b-table>

    <!-- SMTP Server Capacity Panel -->
    <section class="server-capacity-section mt-5" v-if="smtpServers.length > 0">
      <h2 class="title is-5 mb-3">SMTP Server Capacity</h2>
      <div class="columns is-multiline">
        <div class="column is-6" v-for="server in smtpServers" :key="server.uuid">
          <div class="box">
            <div class="level">
              <div class="level-left">
                <div class="level-item">
                  <div>
                    <p class="heading">{{ server.name || 'Unnamed' }}</p>
                    <p class="is-size-7 has-text-grey">{{ server.fromEmail }}</p>
                  </div>
                </div>
              </div>
              <div class="level-right">
                <div class="level-item">
                  <b-tag :type="server.dailyRemaining > 0 ? 'is-success' : 'is-danger'">
                    {{ server.dailyRemaining }} / {{ server.dailyLimit }} remaining
                  </b-tag>
                </div>
              </div>
            </div>
            <b-progress :value="server.dailyUsed" :max="server.dailyLimit"
              :type="getCapacityColor(server)" show-value>
              {{ server.dailyUsed }} used
            </b-progress>
          </div>
        </div>
      </div>
    </section>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import EmptyPlaceholder from '../components/EmptyPlaceholder.vue';

dayjs.extend(relativeTime);

export default Vue.extend({
  components: {
    EmptyPlaceholder,
  },

  data() {
    return {
      queueItems: {
        results: [],
        total: 0,
        perPage: 20,
      },
      stats: {
        queued: 0,
        sending: 0,
        sent: 0,
        failed: 0,
        cancelled: 0,
        nextScheduledAt: null,
      },
      smtpServers: [],
      queuePaused: false,
      queryParams: {
        page: 1,
      },
      filters: {
        campaignId: '',
        status: ['queued', 'sending'], // Default: only show active items, not cancelled/sent/failed
        subscriber: '',
        smtpServerUuid: '',
      },
      refreshInterval: null,
    };
  },

  computed: {
    ...mapState(['loading']),
  },

  mounted() {
    this.fetchData();
    this.getQueuePauseState();
    // Auto-refresh every 30 seconds
    this.refreshInterval = setInterval(() => {
      this.fetchData();
      this.getQueuePauseState();
    }, 30000);
  },

  beforeDestroy() {
    if (this.refreshInterval) {
      clearInterval(this.refreshInterval);
    }
  },

  methods: {
    async fetchData() {
      await Promise.all([
        this.getQueueItems(),
        this.getQueueStats(),
        this.getSMTPServerCapacity(),
      ]);
    },

    async getQueueItems() {
      const params = {
        page: this.queryParams.page,
        per_page: this.queueItems.perPage,
      };

      if (this.filters.campaignId) {
        params.campaign_id = this.filters.campaignId;
      }
      if (this.filters.status.length > 0) {
        params.status = this.filters.status;
      }
      if (this.filters.subscriber) {
        params.subscriber = this.filters.subscriber;
      }
      if (this.filters.smtpServerUuid) {
        params.smtp_server_uuid = this.filters.smtpServerUuid;
      }

      try {
        const data = await this.$api.getQueueItems(params);
        this.queueItems = data;
      } catch (e) {
        this.$buefy.toast.open({
          message: `Error fetching queue items: ${e.message}`,
          type: 'is-danger',
          queue: false,
        });
      }
    },

    async getQueueStats() {
      try {
        this.stats = await this.$api.getQueueStats();
      } catch (e) {
        this.$buefy.toast.open({
          message: `Error fetching queue stats: ${e.message}`,
          type: 'is-danger',
          queue: false,
        });
      }
    },

    async getSMTPServerCapacity() {
      try {
        this.smtpServers = await this.$api.getSMTPServerCapacity();
      } catch (e) {
        this.$buefy.toast.open({
          message: `Error fetching SMTP server capacity: ${e.message}`,
          type: 'is-danger',
          queue: false,
        });
      }
    },

    async cancelItem(id) {
      try {
        await this.$api.cancelQueueItem(id);
        this.$buefy.toast.open({
          message: 'Queue item cancelled successfully',
          type: 'is-success',
          queue: false,
        });
        this.fetchData();
      } catch (e) {
        this.$buefy.toast.open({
          message: `Error cancelling item: ${e.message}`,
          type: 'is-danger',
          queue: false,
        });
      }
    },

    async retryItem(id) {
      try {
        await this.$api.retryQueueItem(id);
        this.$buefy.toast.open({
          message: 'Queue item queued for retry',
          type: 'is-success',
          queue: false,
        });
        this.fetchData();
      } catch (e) {
        this.$buefy.toast.open({
          message: `Error retrying item: ${e.message}`,
          type: 'is-danger',
          queue: false,
        });
      }
    },

    refresh() {
      this.fetchData();
    },

    applyFilters() {
      this.queryParams.page = 1;
      this.getQueueItems();
    },

    clearFilters() {
      this.filters = {
        campaignId: '',
        status: ['queued', 'sending'], // Default: only show active items
        subscriber: '',
        smtpServerUuid: '',
      };
      this.queryParams.page = 1;
      this.getQueueItems();
    },

    onPageChange(page) {
      this.queryParams.page = page;
      this.getQueueItems();
    },

    formatDate(date) {
      if (!date) return '-';
      return dayjs(date).format('MMM D, YYYY h:mm A');
    },

    countdown(date) {
      if (!date) return '';
      return dayjs(date).fromNow();
    },

    getStatusType(status) {
      const types = {
        queued: 'is-info',
        sending: 'is-warning',
        sent: 'is-success',
        failed: 'is-danger',
        cancelled: 'is-light',
      };
      return types[status] || 'is-light';
    },

    getPriorityLabel(priority) {
      if (priority === 0) return 'Normal';
      if (priority > 0) return `High (${priority})`;
      return `Low (${priority})`;
    },

    getServerName(uuid) {
      const server = this.smtpServers.find((s) => s.uuid === uuid);
      return server ? (server.name || uuid.substring(0, 8)) : uuid.substring(0, 8);
    },

    getCapacityColor(server) {
      const percentage = (server.dailyUsed / server.dailyLimit) * 100;
      if (percentage >= 90) return 'is-danger';
      if (percentage >= 70) return 'is-warning';
      return 'is-success';
    },

    canCancel(item) {
      return item.status === 'queued' || item.status === 'sending';
    },

    canRetry(item) {
      return item.status === 'failed' || item.status === 'cancelled';
    },

    truncate(text, length) {
      if (!text) return '';
      return text.length > length ? `${text.substring(0, length)}...` : text;
    },

    async getQueuePauseState() {
      try {
        const data = await this.$api.getSettings();
        this.queuePaused = data['app.queue_paused'] || false;
      } catch (e) {
        console.error('Error fetching queue pause state:', e);
      }
    },

    async togglePause() {
      const newState = !this.queuePaused;
      const action = newState ? 'pause' : 'resume';

      this.$buefy.dialog.confirm({
        title: `${action.charAt(0).toUpperCase() + action.slice(1)} Queue`,
        message: `Are you sure you want to ${action} the queue processing?`,
        confirmText: action.charAt(0).toUpperCase() + action.slice(1),
        type: newState ? 'is-warning' : 'is-success',
        hasIcon: true,
        onConfirm: async () => {
          try {
            await this.$api.toggleQueuePause(newState);
            this.queuePaused = newState;
            this.$buefy.toast.open({
              message: `Queue ${action}d successfully`,
              type: 'is-success',
            });
          } catch (e) {
            this.$buefy.toast.open({
              message: `Error ${action}ing queue: ${e.message}`,
              type: 'is-danger',
              queue: false,
            });
          }
        },
      });
    },

    confirmClearQueue() {
      this.$buefy.dialog.confirm({
        title: 'Clear Queue',
        message: `
          <p class="has-text-danger"><strong>Warning:</strong> This will cancel ALL queued emails.</p>
          <p>This action cannot be undone.</p>
          <p>Are you sure you want to continue?</p>
        `,
        confirmText: 'Clear Queue',
        type: 'is-danger',
        hasIcon: true,
        onConfirm: async () => {
          try {
            const response = await this.$api.clearAllQueuedEmails();

            // Build success message
            let message = `Successfully cleared ${response.count} queued emails`;

            // Add test mode feedback if applicable
            if (response.test_mode) {
              const extras = [];
              if (response.canceled_count > 0) {
                extras.push(`${response.canceled_count} canceled emails deleted`);
              }
              if (response.sent_count > 0) {
                extras.push(`${response.sent_count} sent emails deleted`);
              }
              if (response.reset_capacity) {
                extras.push('SMTP capacity reset');
              }
              if (response.reset_sliding_window) {
                extras.push('sliding window state reset');
              }
              if (extras.length > 0) {
                message += ` (Test Mode: ${extras.join(', ')})`;
              }
            }

            this.$buefy.toast.open({
              message,
              type: 'is-success',
              duration: 5000,
            });
            this.fetchData();
          } catch (e) {
            this.$buefy.toast.open({
              message: `Error clearing queue: ${e.message}`,
              type: 'is-danger',
              queue: false,
            });
          }
        },
      });
    },

    confirmSendAll() {
      this.$buefy.dialog.confirm({
        title: 'Send All Emails',
        message: `
          <p class="has-text-warning"><strong>Warning:</strong> This will schedule ALL queued emails to be sent immediately.</p>
          <p>Emails will be sent according to server capacity and rate limits.</p>
          <p>Are you sure you want to continue?</p>
        `,
        confirmText: 'Send All',
        type: 'is-info',
        hasIcon: true,
        onConfirm: async () => {
          try {
            const response = await this.$api.sendAllQueuedEmails();
            this.$buefy.toast.open({
              message: `Successfully scheduled ${response.count} emails for immediate sending`,
              type: 'is-success',
            });
            this.fetchData();
          } catch (e) {
            this.$buefy.toast.open({
              message: `Error sending all emails: ${e.message}`,
              type: 'is-danger',
              queue: false,
            });
          }
        },
      });
    },
  },
});
</script>

<style scoped>
.stats-section {
  margin-bottom: 2rem;
}

.filters-section {
  background: #f5f5f5;
  padding: 1.5rem;
  border-radius: 4px;
}

.server-capacity-section {
  border-top: 1px solid #dbdbdb;
  padding-top: 2rem;
}

.actions a {
  margin-right: 0.5rem;
}

.actions a[data-disabled] {
  opacity: 0.3;
  cursor: not-allowed;
}
</style>
