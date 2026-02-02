<template>
  <section class="flows">
    <header class="columns page-header">
      <div class="column is-4">
        <h1 class="title is-4">Flows</h1>
      </div>
      <div class="column has-text-right">
        <b-button type="is-primary" icon-left="plus" tag="router-link" :to="{ name: 'newFlow' }">
          {{ $t('globals.buttons.new') }}
        </b-button>
      </div>
    </header>

    <!-- Status filter tabs -->
    <b-tabs v-model="activeTab" type="is-boxed" class="mb-4">
      <b-tab-item label="All" value="" />
      <b-tab-item label="Active" value="active" />
      <b-tab-item label="Draft" value="draft" />
      <b-tab-item label="Paused" value="paused" />
    </b-tabs>

    <b-table
      :data="filteredFlows"
      :loading="loading.flows"
      hoverable
      :paginated="filteredFlows && filteredFlows.length > 20"
      per-page="20"
      default-sort="createdAt"
      default-sort-direction="desc"
    >
      <b-table-column v-slot="props" field="name" label="Name" sortable>
        <router-link :to="{ name: 'flow', params: { id: props.row.id } }">
          <b>{{ props.row.name }}</b>
        </router-link>
        <p v-if="props.row.description" class="is-size-7 has-text-grey">
          {{ props.row.description }}
        </p>
      </b-table-column>

      <b-table-column v-slot="props" field="trigger_type" label="Trigger" sortable width="180">
        <b-tag :type="triggerColors[props.row.trigger_type] || 'is-light'">
          {{ formatTriggerType(props.row.trigger_type) }}
        </b-tag>
      </b-table-column>

      <b-table-column v-slot="props" field="status" :label="$t('globals.fields.status')" sortable width="100">
        <b-tag :type="statusColors[props.row.status]">
          {{ props.row.status }}
        </b-tag>
      </b-table-column>

      <b-table-column v-slot="props" label="Active Runs" width="120" numeric>
        <span v-if="props.row.stats">{{ props.row.stats.active_runs || 0 }}</span>
        <span v-else class="has-text-grey">—</span>
      </b-table-column>

      <b-table-column v-slot="props" label="Completed" width="120" numeric>
        <span v-if="props.row.stats">{{ props.row.stats.completed_runs || 0 }}</span>
        <span v-else class="has-text-grey">—</span>
      </b-table-column>

      <b-table-column v-slot="props" label="Emails Sent" width="120" numeric>
        <span v-if="props.row.stats">{{ props.row.stats.emails_sent || 0 }}</span>
        <span v-else class="has-text-grey">—</span>
      </b-table-column>

      <b-table-column v-slot="props" field="created_at" :label="$t('globals.fields.createdAt')" sortable width="150">
        {{ $utils.niceDate(props.row.created_at) }}
      </b-table-column>

      <b-table-column v-slot="props" cell-class="actions" width="120" align="right">
        <div>
          <!-- Toggle Status -->
          <a href="#" @click.prevent="toggleStatus(props.row)"
            :aria-label="props.row.status === 'active' ? 'Pause' : 'Activate'">
            <b-tooltip :label="props.row.status === 'active' ? 'Pause' : 'Activate'" type="is-dark">
              <b-icon :icon="props.row.status === 'active' ? 'pause' : 'play'" size="is-small" />
            </b-tooltip>
          </a>

          <!-- Duplicate -->
          <a href="#" @click.prevent="duplicateFlow(props.row)" aria-label="Duplicate">
            <b-tooltip label="Duplicate" type="is-dark">
              <b-icon icon="file-multiple-outline" size="is-small" />
            </b-tooltip>
          </a>

          <!-- Edit -->
          <router-link :to="{ name: 'flow', params: { id: props.row.id } }" aria-label="Edit">
            <b-tooltip label="Edit" type="is-dark">
              <b-icon icon="pencil-outline" size="is-small" />
            </b-tooltip>
          </router-link>

          <!-- Delete -->
          <a href="#" @click.prevent="confirmDelete(props.row)" aria-label="Delete">
            <b-icon icon="trash-can-outline" size="is-small" />
          </a>
        </div>
      </b-table-column>

      <template #empty>
        <empty-placeholder>
          <p>No flows yet. Create your first automation flow.</p>
          <b-button type="is-primary" icon-left="plus" tag="router-link" :to="{ name: 'newFlow' }">
            Create Flow
          </b-button>
        </empty-placeholder>
      </template>
    </b-table>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import EmptyPlaceholder from '../components/EmptyPlaceholder.vue';

export default Vue.extend({
  name: 'Flows',

  components: {
    EmptyPlaceholder,
  },

  data() {
    return {
      activeTab: '',
      triggerColors: {
        list_subscription: 'is-info',
        segment_entry: 'is-link',
        shopify_order_created: 'is-success',
        shopify_order_fulfilled: 'is-primary',
        shopify_order_cancelled: 'is-warning',
        shopify_checkout_abandoned: 'is-danger',
        form_submission: 'is-info',
        date_property: 'is-dark',
      },
      statusColors: {
        draft: '',
        active: 'is-success',
        paused: 'is-warning',
        archived: 'is-dark',
      },
      triggerLabels: {
        list_subscription: 'List Subscription',
        segment_entry: 'Segment Entry',
        shopify_order_created: 'Order Created',
        shopify_order_fulfilled: 'Order Fulfilled',
        shopify_order_cancelled: 'Order Cancelled',
        shopify_checkout_abandoned: 'Abandoned Checkout',
        form_submission: 'Form Submission',
        date_property: 'Date Property',
      },
    };
  },

  computed: {
    ...mapState(['flows', 'loading']),

    filteredFlows() {
      if (!this.flows || !Array.isArray(this.flows)) return [];
      if (!this.activeTab) return this.flows;
      return this.flows.filter((f) => f.status === this.activeTab);
    },
  },

  mounted() {
    this.loadFlows();
  },

  methods: {
    loadFlows() {
      this.$api.getFlows({ include_stats: 'true' });
    },

    formatTriggerType(type) {
      return this.triggerLabels[type] || type;
    },

    async toggleStatus(flow) {
      const newStatus = flow.status === 'active' ? 'paused' : 'active';
      try {
        await this.$api.updateFlowStatus(flow.id, newStatus);
        this.$utils.toast(this.$t('globals.messages.updated', { name: flow.name }), 'is-success');
        this.loadFlows();
      } catch (err) {
        this.$utils.toast(err.message, 'is-danger');
      }
    },

    async duplicateFlow(flow) {
      try {
        const resp = await this.$api.duplicateFlow(flow.id);
        this.$utils.toast('Flow duplicated', 'is-success');
        this.$router.push({ name: 'flow', params: { id: resp.id } });
      } catch (err) {
        this.$utils.toast(err.message, 'is-danger');
      }
    },

    confirmDelete(flow) {
      this.$buefy.dialog.confirm({
        title: this.$t('globals.buttons.delete'),
        message: `Delete flow "${flow.name}"? This will cancel all active runs and cannot be undone.`,
        confirmText: this.$t('globals.buttons.delete'),
        type: 'is-danger',
        hasIcon: true,
        onConfirm: () => this.deleteFlow(flow),
      });
    },

    async deleteFlow(flow) {
      try {
        await this.$api.deleteFlow(flow.id);
        this.$utils.toast(this.$t('globals.messages.deleted'), 'is-success');
        this.loadFlows();
      } catch (err) {
        this.$utils.toast(err.message, 'is-danger');
      }
    },
  },
});
</script>

<style lang="scss" scoped>
.flows {
  .b-table {
    background: #fff;
    border-radius: 4px;
    padding: 1rem;
  }

  .actions a {
    margin-left: 0.5rem;
  }
}
</style>
