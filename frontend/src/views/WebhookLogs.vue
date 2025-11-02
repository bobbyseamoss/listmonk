<template>
  <section class="webhook-logs">
    <header class="page-header columns">
      <div class="column is-two-thirds">
        <h1 class="title is-4">
          Webhook Logs
          <span v-if="logs.total > 0">({{ logs.total }})</span>
        </h1>
      </div>
    </header>

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

      <b-table-column v-slot="props" field="webhook_type" label="Webhook Type">
        <b-tag>{{ props.row.webhookType }}</b-tag>
      </b-table-column>

      <b-table-column v-slot="props" field="event_type" label="Event Type">
        <span v-if="props.row.eventType">{{ props.row.eventType }}</span>
        <span v-else class="has-text-grey">-</span>
      </b-table-column>

      <b-table-column v-slot="props" field="response_status" label="Status">
        <b-tag :type="props.row.responseStatus >= 200 && props.row.responseStatus < 300 ? 'is-success' : 'is-danger'">
          {{ props.row.responseStatus }}
        </b-tag>
      </b-table-column>

      <b-table-column v-slot="props" field="processed" label="Processed">
        <b-icon v-if="props.row.processed" icon="check-circle" type="is-success" size="is-small" />
        <b-icon v-else icon="close-circle" type="is-danger" size="is-small" />
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

    onPageChange(p) {
      this.queryParams.page = p;
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

      const params = new URLSearchParams();
      params.append('page', this.queryParams.page);
      if (this.queryParams.webhookType) {
        params.append('webhook_type', this.queryParams.webhookType);
      }
      if (this.queryParams.eventType) {
        params.append('event_type', this.queryParams.eventType);
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
  },

  computed: {
    ...mapState(['loading']),
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
