<template>
  <section class="signup-forms">
    <header class="columns page-header">
      <div class="column is-4">
        <h1 class="title is-4">{{ $t('forms.title') }}</h1>
      </div>
      <div class="column has-text-right">
        <b-button type="is-primary" icon-left="plus" tag="router-link" :to="{ name: 'newSignupForm' }">
          {{ $t('globals.buttons.new') }}
        </b-button>
      </div>
    </header>

    <b-table
      :data="forms.results"
      :loading="loading.forms"
      hoverable
      :paginated="forms.results && forms.results.length > 20"
      per-page="20"
      default-sort="createdAt"
      default-sort-direction="desc"
    >
      <b-table-column v-slot="props" field="name" :label="$t('globals.fields.name')" sortable>
        <router-link :to="{ name: 'editSignupForm', params: { id: props.row.id } }">
          <b>{{ props.row.name }}</b>
        </router-link>
        <p v-if="props.row.description" class="is-size-7 has-text-grey">
          {{ props.row.description }}
        </p>
      </b-table-column>

      <b-table-column v-slot="props" field="formType" label="Type" sortable width="100">
        <b-tag :type="typeColors[props.row.formType]">
          {{ props.row.formType }}
        </b-tag>
      </b-table-column>

      <b-table-column v-slot="props" field="status" :label="$t('globals.fields.status')" sortable width="100">
        <b-tag :type="statusColors[props.row.status]">
          {{ props.row.status }}
        </b-tag>
      </b-table-column>

      <b-table-column v-slot="props" field="impressions" label="Impressions" sortable numeric width="120">
        {{ props.row.impressions || 0 }}
      </b-table-column>

      <b-table-column v-slot="props" field="submissions" label="Submissions" sortable numeric width="120">
        {{ props.row.submissions || 0 }}
      </b-table-column>

      <b-table-column v-slot="props" label="Conversion" width="100">
        <span v-if="props.row.impressions > 0">
          {{ ((props.row.submissions || 0) / props.row.impressions * 100).toFixed(1) }}%
        </span>
        <span v-else class="has-text-grey">—</span>
      </b-table-column>

      <b-table-column v-slot="props" field="createdAt" :label="$t('globals.fields.createdAt')" sortable width="150">
        {{ $utils.niceDate(props.row.createdAt) }}
      </b-table-column>

      <b-table-column v-slot="props" width="150" cell-class="has-text-right">
        <!-- Toggle Status -->
        <b-tooltip :label="props.row.status === 'active' ? 'Pause' : 'Activate'">
          <b-button
            size="is-small"
            :type="props.row.status === 'active' ? 'is-warning' : 'is-success'"
            :icon-left="props.row.status === 'active' ? 'pause' : 'play'"
            @click="toggleStatus(props.row)"
          />
        </b-tooltip>

        <!-- Duplicate -->
        <b-tooltip label="Duplicate">
          <b-button size="is-small" icon-left="content-copy" @click="duplicateForm(props.row)" />
        </b-tooltip>

        <!-- Analytics -->
        <b-tooltip label="Analytics">
          <b-button
            size="is-small"
            icon-left="chart-line"
            tag="router-link"
            :to="{ name: 'formAnalytics', params: { id: props.row.id } }"
          />
        </b-tooltip>

        <!-- Delete -->
        <b-tooltip label="Delete">
          <b-button size="is-small" type="is-danger" icon-left="delete-outline" @click="confirmDelete(props.row)" />
        </b-tooltip>
      </b-table-column>

      <template #empty>
        <empty-placeholder>
          <p>{{ $t('forms.empty') }}</p>
          <b-button type="is-primary" icon-left="plus" tag="router-link" :to="{ name: 'newSignupForm' }">
            {{ $t('forms.createNew') }}
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
  name: 'SignupForms',

  components: {
    EmptyPlaceholder,
  },

  data() {
    return {
      typeColors: {
        popup: 'is-info',
        flyout: 'is-link',
        fullpage: 'is-primary',
        embed: 'is-success',
        banner: 'is-warning',
      },
      statusColors: {
        draft: '',
        active: 'is-success',
        paused: 'is-warning',
        archived: 'is-dark',
      },
    };
  },

  computed: {
    ...mapState(['forms', 'loading']),
  },

  mounted() {
    this.loadForms();
  },

  methods: {
    loadForms() {
      this.$api.getForms();
    },

    async toggleStatus(formItem) {
      const newStatus = formItem.status === 'active' ? 'paused' : 'active';
      try {
        await this.$api.updateFormStatus(formItem.id, newStatus);
        this.$utils.toast(this.$t('globals.messages.updated'), 'is-success');
        this.loadForms(); // Reload to get updated data
      } catch (err) {
        this.$utils.toast(err.message, 'is-danger');
      }
    },

    async duplicateForm(form) {
      try {
        const resp = await this.$api.duplicateForm(form.id);
        this.$utils.toast('Form duplicated', 'is-success');
        this.$router.push({ name: 'editSignupForm', params: { id: resp.id } });
      } catch (err) {
        this.$utils.toast(err.message, 'is-danger');
      }
    },

    confirmDelete(form) {
      this.$buefy.dialog.confirm({
        title: this.$t('globals.buttons.delete'),
        message: `Delete form "${form.name}"? This cannot be undone.`,
        confirmText: this.$t('globals.buttons.delete'),
        type: 'is-danger',
        hasIcon: true,
        onConfirm: () => this.deleteForm(form),
      });
    },

    async deleteForm(form) {
      try {
        await this.$api.deleteForm(form.id);
        this.$utils.toast(this.$t('globals.messages.deleted'), 'is-success');
        this.loadForms(); // Reload to reflect deletion
      } catch (err) {
        this.$utils.toast(err.message, 'is-danger');
      }
    },
  },
});
</script>

<style lang="scss" scoped>
.signup-forms {
  .b-table {
    background: #fff;
    border-radius: 4px;
    padding: 1rem;
  }
}
</style>
