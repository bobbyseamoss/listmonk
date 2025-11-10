<template>
  <section class="campaigns">
    <header class="columns page-header">
      <div class="column is-10">
        <h1 class="title is-4">
          {{ $t('globals.terms.campaigns') }}
          <span v-if="!isNaN(campaigns.total)">({{ campaigns.total }})</span>
        </h1>
      </div>
      <div class="column has-text-right">
        <b-field v-if="$can('campaigns:manage')" expanded>
          <b-button expanded :to="{ name: 'campaign', params: { id: 'new' } }" tag="router-link" class="btn-new"
            type="is-primary" icon-left="plus" data-cy="btn-new">
            {{ $t('globals.buttons.new') }}
          </b-button>
        </b-field>
      </div>
    </header>

    <!-- Email Performance Summary (Last 30 Days) -->
    <section class="performance-summary" v-if="performanceSummary">
      <div class="box">
        <details open>
          <summary class="title is-6">{{ $t('campaigns.performanceSummary', 'Email performance last 30 days') }}</summary>
          <div class="columns stats-grid">
            <div class="column is-3">
              <div class="stat-item">
                <p class="stat-value">{{ formatPercent(performanceSummary.avg_open_rate) }}</p>
                <p class="stat-label">{{ $t('campaigns.avgOpenRate', 'Average open rate') }}</p>
              </div>
            </div>
            <div class="column is-3">
              <div class="stat-item">
                <p class="stat-value">{{ formatPercent(performanceSummary.avg_click_rate) }}</p>
                <p class="stat-label">{{ $t('campaigns.avgClickRate', 'Average click rate') }}</p>
              </div>
            </div>
            <div class="column is-3">
              <div class="stat-item">
                <p class="stat-value">{{ formatPercent(performanceSummary.order_rate) }}</p>
                <p class="stat-label">{{ $t('campaigns.placedOrder', 'Placed Order') }}</p>
              </div>
            </div>
            <div class="column is-3">
              <div class="stat-item">
                <p class="stat-value">${{ formatCurrency(performanceSummary.revenue_per_recipient) }}</p>
                <p class="stat-label">{{ $t('campaigns.revenuePerRecipient', 'Revenue per recipient') }}</p>
              </div>
            </div>
          </div>
        </details>
      </div>
    </section>

    <b-table :data="campaigns.results" :loading="loading.campaigns" :row-class="highlightedRow" paginated
      backend-pagination pagination-position="both" @page-change="onPageChange" :current-page="queryParams.page"
      :per-page="campaigns.perPage" :total="campaigns.total" hoverable backend-sorting @sort="onSort">
      <template #top-left>
        <div class="columns">
          <div class="column is-6">
            <form @submit.prevent="getCampaigns">
              <div>
                <b-field>
                  <b-input v-model="queryParams.query" name="query" expanded
                    :placeholder="$t('campaigns.queryPlaceholder')" icon="magnify" ref="query" />
                  <p class="controls">
                    <b-button native-type="submit" type="is-primary" icon-left="magnify" />
                  </p>
                </b-field>
              </div>
            </form>
          </div>
        </div>
      </template>

      <b-table-column v-slot="props" cell-class="status" field="status" :label="$t('globals.fields.status')" width="10%"
        sortable :td-attrs="$utils.tdID" header-class="cy-status">
        <div>
          <p>
            <router-link :to="{ name: 'campaign', params: { id: props.row.id } }">
              <b-tag :class="props.row.status">
                {{ $t(`campaigns.status.${props.row.status}`) }}
              </b-tag>
              <span class="spinner is-tiny" v-if="isRunning(props.row.id)">
                <b-loading :is-full-page="false" active />
              </span>
            </router-link>
          </p>
          <p v-if="isSheduled(props.row)">
            <span class="is-size-7 has-text-grey scheduled">
              <b-icon icon="alarm" size="is-small" />
              <span v-if="!isDone(props.row) && !isRunning(props.row)">
                {{ $utils.duration(new Date(), props.row.sendAt, true) }}
                <br />
              </span>
              {{ $utils.niceDate(props.row.sendAt, true) }}
            </span>
          </p>
        </div>
      </b-table-column>
      <b-table-column v-slot="props" field="name" :label="$t('globals.fields.name')" width="25%" sortable
        header-class="cy-name">
        <div>
          <p>
            <b-tag v-if="props.row.type === 'optin'" class="is-small">
              {{ $t('lists.optin') }}
            </b-tag>
            <router-link :to="{ name: 'campaign', params: { id: props.row.id } }">
              {{ props.row.name }}
              <copy-text :text="props.row.name" hide-text />
            </router-link>
          </p>
          <p class="is-size-7 has-text-grey">
            <copy-text :text="props.row.subject" />
          </p>

          <!-- Campaign Progress -->
          <div class="campaign-progress" v-if="showProgress(props.row)">
            <b-progress
              :value="getProgressPercent(getCampaignStats(props.row))"
              :type="props.row.status === 'running' ? 'is-primary' : props.row.status === 'paused' ? 'is-warning' : props.row.status === 'cancelled' ? 'is-danger' : 'is-success'"
              size="is-small"
              show-value
            >
              {{ getProgressText(props.row) }}
            </b-progress>
          </div>

          <b-taglist>
            <b-tag class="is-small" v-for="t in props.row.tags" :key="t">
              {{ t }}
            </b-tag>
          </b-taglist>
        </div>
      </b-table-column>
      <b-table-column v-slot="props" cell-class="lists" field="lists" :label="$t('globals.terms.lists')" width="15%">
        <ul>
          <li v-for="l in props.row.lists" :key="l.id">
            <router-link :to="{ name: 'subscribers_list', params: { listID: l.id } }">
              {{ l.name }}
            </router-link>
          </li>
        </ul>
      </b-table-column>
      <b-table-column v-slot="props" field="created_at" :label="$t('campaigns.startDate', 'Start Date')" width="12%" sortable
        header-class="cy-start-date">
        <div>
          <p v-if="getCampaignStats(props.row).startedAt">
            {{ $utils.niceDate(getCampaignStats(props.row).startedAt, true) }}
          </p>
          <p v-else class="has-text-grey">
            —
          </p>
        </div>
      </b-table-column>

      <b-table-column v-slot="props" field="open_rate" :label="$t('campaigns.openRate', 'Open Rate')" width="10%">
        <div class="fields stats">
          <p>
            <label for="#">{{ calculateOpenRate(getCampaignStats(props.row)) }}</label>
            <span>{{ getCampaignStats(props.row).views || 0 }} {{ $t('campaigns.views', 'views') }}</span>
          </p>
        </div>
      </b-table-column>

      <b-table-column v-slot="props" field="click_rate" :label="$t('campaigns.clickRate', 'Click Rate')" width="10%">
        <div class="fields stats">
          <p>
            <label for="#">{{ calculateClickRate(getCampaignStats(props.row)) }}</label>
            <span>{{ getCampaignStats(props.row).clicks || 0 }} {{ $t('campaigns.clicks', 'clicks') }}</span>
          </p>
        </div>
      </b-table-column>

      <b-table-column v-slot="props" field="purchase_revenue" :label="$t('campaigns.placedOrder', 'Placed Order')" width="12%">
        <div class="fields stats">
          <p v-if="props.row.purchaseOrders > 0">
            <label for="#">${{ formatCurrency(props.row.purchaseRevenue) }}</label>
            <span>{{ props.row.purchaseOrders }} {{ props.row.purchaseOrders === 1 ? $t('campaigns.recipient', 'recipient') : $t('campaigns.recipients', 'recipients') }}</span>
          </p>
          <p v-else>
            <label for="#">$0.00</label>
            <span>0 {{ $t('campaigns.recipients', 'recipients') }}</span>
          </p>
        </div>
      </b-table-column>

      <b-table-column v-slot="props" cell-class="actions" width="15%" align="right">
        <div>
          <!-- start / pause / resume / scheduled -->
          <template v-if="$can('campaigns:manage')">
            <a v-if="canStart(props.row)" href="#"
              @click.prevent="$utils.confirm(null, () => changeCampaignStatus(props.row, 'running'))"
              data-cy="btn-start" :aria-label="$t('campaigns.start')">
              <b-tooltip :label="$t('campaigns.start')" type="is-dark">
                <b-icon icon="play" size="is-small" />
              </b-tooltip>
            </a>

            <a v-if="canPause(props.row)" href="#"
              @click.prevent="$utils.confirm(null, () => changeCampaignStatus(props.row, 'paused'))" data-cy="btn-pause"
              :aria-label="$t('campaigns.pause')">
              <b-tooltip :label="$t('campaigns.pause')" type="is-dark">
                <b-icon icon="pause" size="is-small" />
              </b-tooltip>
            </a>

            <a v-if="canResume(props.row)" href="#"
              @click.prevent="$utils.confirm(null, () => changeCampaignStatus(props.row, 'running'))"
              data-cy="btn-resume" :aria-label="$t('campaigns.send')">
              <b-tooltip :label="$t('campaigns.send')" type="is-dark">
                <b-icon icon="play" size="is-small" />
              </b-tooltip>
            </a>

            <a v-if="canSchedule(props.row)" href="#"
              @click.prevent="$utils.confirm($t('campaigns.confirmSchedule'), () => changeCampaignStatus(props.row, 'scheduled'))"
              data-cy="btn-schedule" :aria-label="$t('campaigns.schedule')">
              <b-tooltip :label="$t('campaigns.schedule')" type="is-dark">
                <b-icon icon="clock-start" size="is-small" />
              </b-tooltip>
            </a>

            <!-- placeholder for finished campaigns -->
            <a v-if="!canCancel(props.row) && !canSchedule(props.row) && !canStart(props.row)" href="#" data-disabled
              aria-label=" ">
              <b-icon icon="play" size="is-small" />
            </a>

            <a v-if="canCancel(props.row)" href="#"
              @click.prevent="$utils.confirm(null, () => changeCampaignStatus(props.row, 'cancelled'))"
              data-cy="btn-cancel" :aria-label="$t('globals.buttons.cancel')">
              <b-tooltip :label="$t('globals.buttons.cancel')" type="is-dark">
                <b-icon icon="cancel" size="is-small" />
              </b-tooltip>
            </a>
            <a v-else href="#" data-disabled aria-label=" ">
              <b-icon icon="cancel" size="is-small" />
            </a>
          </template>

          <a href="#" @click.prevent="previewCampaign(props.row)" data-cy="btn-preview"
            :aria-label="$t('campaigns.preview')">
            <b-tooltip :label="$t('campaigns.preview')" type="is-dark">
              <b-icon icon="file-find-outline" size="is-small" />
            </b-tooltip>
          </a>
          <a v-if="$can('campaigns:manage')" href="#" @click.prevent="$utils.prompt($t('globals.buttons.clone'),
            {
              placeholder: $t('globals.fields.name'),
              value: $t('campaigns.copyOf', { name: props.row.name }),
            },
            (name) => cloneCampaign(name, props.row))" data-cy="btn-clone" :aria-label="$t('globals.buttons.clone')">
            <b-tooltip :label="$t('globals.buttons.clone')" type="is-dark">
              <b-icon icon="file-multiple-outline" size="is-small" />
            </b-tooltip>
          </a>
          <router-link v-if="$can('campaigns:get_analytics')"
            :to="{ name: 'campaignAnalytics', query: { id: props.row.id } }">
            <b-tooltip :label="$t('globals.terms.analytics')" type="is-dark">
              <b-icon icon="chart-bar" size="is-small" />
            </b-tooltip>
          </router-link>
          <a v-if="$can('campaigns:manage')" href="#"
            @click.prevent="$utils.confirm($t('campaigns.confirmDelete', { name: props.row.name }), () => deleteCampaign(props.row))"
            data-cy="btn-delete" :aria-label="$t('globals.buttons.delete')">
            <b-icon icon="trash-can-outline" size="is-small" />
          </a>
        </div>
      </b-table-column>

      <template #empty v-if="!loading.campaigns">
        <empty-placeholder />
      </template>
    </b-table>

    <campaign-preview v-if="previewItem" type="campaign" :id="previewItem.id" :title="previewItem.name"
      @close="closePreview" />
  </section>
</template>

<script>
import dayjs from 'dayjs';
import Vue from 'vue';
import { mapState } from 'vuex';
import CampaignPreview from '../components/CampaignPreview.vue';
import EmptyPlaceholder from '../components/EmptyPlaceholder.vue';
import CopyText from '../components/CopyText.vue';

export default Vue.extend({
  components: {
    CampaignPreview,
    EmptyPlaceholder,
    CopyText,
  },

  data() {
    return {
      previewItem: null,
      performanceSummary: null,
      queryParams: {
        page: 1,
        query: '',
        orderBy: 'created_at',
        order: 'desc',
      },
      pollID: null,
      campaignStatsData: {},
    };
  },

  methods: {
    // Campaign statuses.
    canStart(c) {
      return c.status === 'draft' && !c.sendAt;
    },
    canSchedule(c) {
      return c.status === 'draft' && c.sendAt;
    },
    canPause(c) {
      return c.status === 'running';
    },
    canCancel(c) {
      return c.status === 'running' || c.status === 'paused';
    },
    canResume(c) {
      return c.status === 'paused';
    },
    isSheduled(c) {
      return c.status === 'scheduled' || c.sendAt !== null;
    },
    isDone(c) {
      return c.status === 'finished' || c.status === 'cancelled';
    },

    showProgress(campaign) {
      if (!campaign) {
        return false;
      }
      const stats = this.getCampaignStats(campaign);
      if (!stats) {
        return false;
      }

      // Show progress bar for campaigns that have started running
      // (paused, cancelled, finished, or currently running)
      // Don't show for draft or scheduled campaigns
      if (stats.status === 'running' || stats.status === 'paused'
          || stats.status === 'cancelled' || stats.status === 'finished') {
        return true;
      }

      return false;
    },

    getProgressText(campaign) {
      const stats = this.getCampaignStats(campaign);
      if (!stats) {
        return '0 / 0';
      }

      if (stats.use_queue || stats.useQueue) {
        // Queue-based campaign
        const queueTotal = stats.queue_total || stats.queueTotal || 0;
        const queueSent = stats.queue_sent || stats.queueSent || 0;
        return `${queueSent} / ${queueTotal}`;
      }

      // Regular campaign: Use Azure delivery count if available, otherwise fall back to sent
      const toSend = stats.toSend || 0;
      const azureSent = stats.azure_sent || stats.azureSent || 0;
      const sent = stats.sent || 0;
      const effectiveSent = azureSent > 0 ? azureSent : sent;
      return `${effectiveSent} / ${toSend}`;
    },

    isRunning(id) {
      if (id in this.campaignStatsData) {
        return true;
      }
      return false;
    },

    highlightedRow(data) {
      if (data.status === 'running') {
        return ['running'];
      }
      return '';
    },

    onPageChange(p) {
      this.queryParams.page = p;
      this.getCampaigns();
    },

    onSort(field, direction) {
      this.queryParams.orderBy = field;
      this.queryParams.order = direction;
      this.getCampaigns();
    },

    // Campaign actions.
    previewCampaign(c) {
      this.previewItem = c;
    },

    closePreview() {
      this.previewItem = null;
    },

    getCampaigns() {
      this.$api.getCampaigns({
        page: this.queryParams.page,
        query: this.queryParams.query.replace(/[^\p{L}\p{N}\s]/gu, ' '),
        order_by: this.queryParams.orderBy,
        order: this.queryParams.order,
        no_body: true,
      });
    },

    // Calculate progress percentage for both queue-based and regular campaigns
    getProgressPercent(stats) {
      if (!stats) {
        return 0;
      }

      if (stats.use_queue || stats.useQueue) {
        // Queue-based campaign
        const queueTotal = stats.queue_total || stats.queueTotal || 0;
        if (queueTotal === 0) return 0;
        const queueSent = stats.queue_sent || stats.queueSent || 0;
        return (queueSent / queueTotal) * 100;
      }

      // Regular campaign: Use Azure delivery count if available, otherwise fall back to sent
      const toSend = stats.toSend || 0;
      if (toSend === 0) return 0;
      const azureSent = stats.azure_sent || stats.azureSent || 0;
      const sent = stats.sent || 0;
      const effectiveSent = azureSent > 0 ? azureSent : sent;
      return (effectiveSent / toSend) * 100;
    },

    // Stats returns the campaign object with stats (sent, toSend etc.)
    // if there's live stats available for running campaigns. Otherwise,
    // it returns the incoming campaign object that has the static stats
    // values.
    getCampaignStats(c) {
      if (c.id in this.campaignStatsData) {
        return this.campaignStatsData[c.id];
      }
      return c;
    },

    pollStats() {
      // Clear any running status polls.
      clearInterval(this.pollID);

      // Poll for the status as long as the import is running.
      this.pollID = setInterval(() => {
        this.$api.getCampaignStats().then((data) => {
          // Stop polling. No running campaigns.
          if (data.length === 0) {
            clearInterval(this.pollID);

            // There were running campaigns and stats earlier. Clear them
            // and refetch the campaigns list with up-to-date fields.
            if (Object.keys(this.campaignStatsData).length > 0) {
              this.getCampaigns();
              this.campaignStatsData = {};
            }
          } else {
            // Turn the list of campaigns [{id: 1, ...}, {id: 2, ...}] into
            // a map indexed by the id: {1: {}, 2: {}}.
            this.campaignStatsData = data.reduce((obj, cur) => ({ ...obj, [cur.id]: cur }), {});
          }
        }, () => {
          clearInterval(this.pollID);
        });
      }, 1000);
    },

    changeCampaignStatus(c, status) {
      this.$api.changeCampaignStatus(c.id, status).then(() => {
        this.$utils.toast(this.$t('campaigns.statusChanged', { name: c.name, status }));
        this.getCampaigns();
        this.pollStats();
      });
    },

    async cloneCampaign(name, c) {
      // Fetch the template body from the server.
      let body = '';
      let bodySource = null;
      await this.$api.getCampaign(c.id).then((data) => {
        body = data.body;
        bodySource = data.bodySource;
      });

      const now = this.$utils.getDate();
      const sendLater = !!c.sendAt;
      let sendAt = null;
      if (sendLater) {
        sendAt = dayjs(c.sendAt).isAfter(now) ? c.sendAt : now.add(7, 'day');
      }

      const data = {
        name,
        subject: c.subject,
        lists: c.lists.map((l) => l.id),
        type: c.type,
        from_email: c.fromEmail,
        content_type: c.contentType,
        messenger: c.messenger,
        tags: c.tags,
        template_id: c.templateId,
        body,
        body_source: bodySource,
        altbody: c.altbody,
        headers: c.headers,
        send_later: sendLater,
        send_at: sendAt,
        archive: c.archive,
        archive_template_id: c.archiveTemplateId,
        archive_meta: c.archiveMeta,
        media: c.media.map((m) => m.id),
      };

      if (c.archive) {
        data.archive_slug = `${name.toLowerCase().replace(/[^a-z0-9]/g, '-')}-${Date.now().toString().slice(-4)}`;
      }

      this.$api.createCampaign(data).then((d) => {
        this.$router.push({ name: 'campaign', params: { id: d.id } });
      });
    },

    deleteCampaign(c) {
      this.$api.deleteCampaign(c.id).then(() => {
        this.getCampaigns();
        this.$utils.toast(this.$t('globals.messages.deleted', { name: c.name }));
      });
    },

    getPerformanceSummary() {
      this.$api.getCampaignsPerformanceSummary().then((data) => {
        this.performanceSummary = data;
      }).catch(() => {
        // Silently fail if there's no data
        this.performanceSummary = null;
      });
    },

    formatPercent(value) {
      if (!value || Number.isNaN(value)) return '0.00%';
      return `${value.toFixed(2)}%`;
    },

    formatCurrency(value) {
      if (!value || Number.isNaN(value)) return '0.00';
      return value.toFixed(2);
    },

    calculateOpenRate(stats) {
      if (!stats) return '—';

      const views = stats.views || 0;

      // Determine denominator based on campaign status and type
      let denominator = 0;

      if (stats.use_queue || stats.useQueue) {
        // Queue-based campaign
        const queueSent = stats.queue_sent || stats.queueSent || 0;
        const queueTotal = stats.queue_total || stats.queueTotal || 0;

        // For running queue campaigns, use queue_sent. Otherwise use queue_total.
        if (stats.status === 'running' && queueSent > 0) {
          denominator = queueSent;
        } else {
          denominator = queueTotal;
        }
      } else {
        // Regular campaign: Use Azure delivery count if available
        const azureSent = stats.azure_sent || stats.azureSent || 0;
        const sent = stats.sent || 0;
        const toSend = stats.toSend || 0;

        // Use Azure sent count if available, otherwise use sent/toSend logic
        if (azureSent > 0) {
          denominator = azureSent;
        } else if (stats.status === 'running' && sent > 0) {
          denominator = sent;
        } else {
          denominator = toSend;
        }
      }

      if (denominator === 0) return '—';

      const rate = (views / denominator) * 100;
      return `${rate.toFixed(2)}%`;
    },

    calculateClickRate(stats) {
      if (!stats) return '—';

      const clicks = stats.clicks || 0;

      // Determine denominator based on campaign status and type
      let denominator = 0;

      if (stats.use_queue || stats.useQueue) {
        // Queue-based campaign
        const queueSent = stats.queue_sent || stats.queueSent || 0;
        const queueTotal = stats.queue_total || stats.queueTotal || 0;

        // For running queue campaigns, use queue_sent. Otherwise use queue_total.
        if (stats.status === 'running' && queueSent > 0) {
          denominator = queueSent;
        } else {
          denominator = queueTotal;
        }
      } else {
        // Regular campaign: Use Azure delivery count if available
        const azureSent = stats.azure_sent || stats.azureSent || 0;
        const sent = stats.sent || 0;
        const toSend = stats.toSend || 0;

        // Use Azure sent count if available, otherwise use sent/toSend logic
        if (azureSent > 0) {
          denominator = azureSent;
        } else if (stats.status === 'running' && sent > 0) {
          denominator = sent;
        } else {
          denominator = toSend;
        }
      }

      if (denominator === 0) return '—';

      const rate = (clicks / denominator) * 100;
      return `${rate.toFixed(2)}%`;
    },
  },

  computed: {
    ...mapState(['campaigns', 'loading']),
  },

  mounted() {
    this.getCampaigns();
    this.pollStats();
    this.getPerformanceSummary();
  },

  destroyed() {
    clearInterval(this.pollID);
  },
});
</script>

<style scoped>
.performance-summary {
  margin-bottom: 1.5rem;
}

.performance-summary .box {
  padding: 1rem 1.5rem;
}

.performance-summary summary {
  cursor: pointer;
  margin-bottom: 1rem;
}

.performance-summary .stats-grid {
  margin-top: 0.5rem;
  margin-bottom: 0;
}

.performance-summary .stat-item {
  text-align: center;
}

.performance-summary .stat-value {
  font-size: 2rem;
  font-weight: bold;
  margin-bottom: 0.25rem;
}

.performance-summary .stat-label {
  font-size: 0.875rem;
  color: #4a4a4a;
}

.campaign-progress {
  margin-top: 0.5rem;
  margin-bottom: 0.5rem;
}

.campaign-progress .progress {
  margin-bottom: 0;
}

section.campaigns table tbody td .progress-wrapper .progress.is-small {
  height: 15px;
}

::v-deep .progress-wrapper .progress.is-small + .progress-value {
  font-size: .7rem;
  top: 16%;
}

::v-deep .progress-wrapper .progress.is-small .progress-value {
  font-size: .7rem;
}
</style>
