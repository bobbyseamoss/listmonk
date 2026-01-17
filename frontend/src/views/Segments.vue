<template>
  <section class="segments">
    <header class="columns page-header">
      <div class="column is-10">
        <h1 class="title is-4">
          {{ $t('globals.terms.segments') }}
          <span v-if="!isNaN(segments.total)">({{ segments.total }})</span>
        </h1>
      </div>
      <div class="column has-text-right">
        <b-field v-if="$can('lists:manage_all')" expanded>
          <b-button expanded type="is-primary" icon-left="plus" class="btn-new" @click="showNewForm" data-cy="btn-new">
            {{ $t('globals.buttons.new') }}
          </b-button>
        </b-field>
      </div>
    </header>

    <b-table :data="segments.results" :loading="loading.segments" hoverable default-sort="createdAt" paginated
      backend-pagination pagination-position="both" @page-change="onPageChange" :current-page="queryParams.page"
      :per-page="segments.perPage" :total="segments.total" backend-sorting @sort="onSort">
      <template #top-left>
        <div class="columns">
          <div class="column is-6">
            <form @submit.prevent="getSegments">
              <div>
                <b-field>
                  <b-input v-model="queryParams.query" name="query" expanded icon="magnify" ref="query"
                    data-cy="query" />
                  <p class="controls">
                    <b-button native-type="submit" type="is-primary" icon-left="magnify" data-cy="btn-query" />
                  </p>
                </b-field>
              </div>
            </form>
          </div>
          <div class="column is-6 has-text-right">
            <b-button size="is-small" icon-left="refresh" :loading="refreshingAll" @click="refreshAllCounts">
              {{ $t('segments.refreshAllCounts') }}
            </b-button>
          </div>
        </div>
      </template>

      <b-table-column v-slot="props" field="name" :label="$t('globals.fields.name')" header-class="cy-name" sortable
        width="25%">
        <div>
          <a :href="`/segments/${props.row.id}`" @click.prevent="showEditForm(props.row)">
            {{ props.row.name }}
          </a>
          <b-taglist v-if="props.row.tags && props.row.tags.length">
            <b-tag class="is-small" v-for="t in props.row.tags" :key="t">
              {{ t }}
            </b-tag>
          </b-taglist>
        </div>
      </b-table-column>

      <b-table-column v-slot="props" field="description" :label="$t('segments.description')"
        header-class="cy-description" width="30%">
        <span class="is-size-7">{{ props.row.description || '-' }}</span>
      </b-table-column>

      <b-table-column v-slot="props" field="conditions" :label="$t('segments.conditions')"
        header-class="cy-conditions" width="15%">
        <span class="is-size-7">
          {{ getConditionSummary(props.row.conditions) }}
        </span>
      </b-table-column>

      <b-table-column v-slot="props" field="cached_count" :label="$t('segments.subscriberCount')"
        header-class="cy-count" numeric centered width="10%">
        <template v-if="props.row.cachedCount !== null">
          <a href="#" @click.prevent="showPreview(props.row)">
            {{ $utils.formatNumber(props.row.cachedCount) }}
            <span class="is-size-7 view">{{ $t('segments.preview') }}</span>
          </a>
        </template>
        <template v-else>
          <a href="#" @click.prevent="refreshCount(props.row)" class="is-size-7">
            {{ $t('segments.refreshCount') }}
          </a>
        </template>
      </b-table-column>

      <b-table-column v-slot="props" field="created_at" :label="$t('globals.fields.createdAt')"
        header-class="cy-created_at" sortable>
        {{ $utils.niceDate(props.row.createdAt) }}
      </b-table-column>

      <b-table-column v-slot="props" cell-class="actions" align="right">
        <div>
          <a v-if="$can('lists:manage_all')" href="#" @click.prevent="showEditForm(props.row)" data-cy="btn-edit"
            :aria-label="$t('globals.buttons.edit')">
            <b-tooltip :label="$t('globals.buttons.edit')" type="is-dark">
              <b-icon icon="pencil-outline" size="is-small" />
            </b-tooltip>
          </a>

          <a href="#" @click.prevent="showPreview(props.row)" data-cy="btn-preview"
            :aria-label="$t('segments.preview')">
            <b-tooltip :label="$t('segments.previewSubscribers')" type="is-dark">
              <b-icon icon="eye-outline" size="is-small" />
            </b-tooltip>
          </a>

          <a href="#" @click.prevent="exportSegment(props.row)" data-cy="btn-export"
            :aria-label="$t('segments.exportSubscribers')">
            <b-tooltip :label="$t('segments.exportSubscribers')" type="is-dark">
              <b-icon icon="cloud-download-outline" size="is-small" />
            </b-tooltip>
          </a>

          <a v-if="$can('lists:manage_all')" href="#" @click.prevent="deleteSegment(props.row)" data-cy="btn-delete"
            :aria-label="$t('globals.buttons.delete')">
            <b-tooltip :label="$t('globals.buttons.delete')" type="is-dark">
              <b-icon icon="trash-can-outline" size="is-small" />
            </b-tooltip>
          </a>
        </div>
      </b-table-column>

      <template #empty v-if="!loading.segments">
        <empty-placeholder />
      </template>
    </b-table>

    <!-- Add / edit form modal -->
    <b-modal scroll="keep" :aria-modal="true" :active.sync="isFormVisible" :width="900" @close="onFormClose">
      <segment-form :data="curItem" :is-editing="isEditing" @finished="formFinished" />
    </b-modal>

    <!-- Preview modal -->
    <b-modal scroll="keep" :aria-modal="true" :active.sync="isPreviewVisible" :width="900">
      <div class="modal-card">
        <header class="modal-card-head">
          <p class="modal-card-title">
            {{ $t('segments.previewSubscribers') }}
            <span v-if="previewSegment.name">: {{ previewSegment.name }}</span>
          </p>
        </header>
        <section class="modal-card-body">
          <b-table :data="previewData.results" :loading="previewLoading" hoverable paginated
            backend-pagination pagination-position="both" @page-change="onPreviewPageChange"
            :current-page="previewPage" :per-page="20" :total="previewData.total">
            <b-table-column v-slot="props" field="email" :label="$t('subscribers.email')">
              <router-link :to="`/subscribers/${props.row.id}`">
                {{ props.row.email }}
              </router-link>
            </b-table-column>
            <b-table-column v-slot="props" field="name" :label="$t('globals.fields.name')">
              {{ props.row.name }}
            </b-table-column>
            <b-table-column v-slot="props" field="status" :label="$t('globals.fields.status')">
              <b-tag :class="props.row.status">
                {{ $t(`subscribers.status.${props.row.status}`) }}
              </b-tag>
            </b-table-column>
            <b-table-column v-slot="props" field="created_at" :label="$t('globals.fields.createdAt')">
              {{ $utils.niceDate(props.row.createdAt) }}
            </b-table-column>
            <template #empty v-if="!previewLoading">
              <empty-placeholder />
            </template>
          </b-table>
        </section>
        <footer class="modal-card-foot">
          <b-button @click="isPreviewVisible = false">{{ $t('globals.buttons.close') }}</b-button>
        </footer>
      </div>
    </b-modal>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import EmptyPlaceholder from '../components/EmptyPlaceholder.vue';
import SegmentForm from './SegmentForm.vue';

export default Vue.extend({
  components: {
    SegmentForm,
    EmptyPlaceholder,
  },

  data() {
    return {
      curItem: null,
      isEditing: false,
      isFormVisible: false,
      isPreviewVisible: false,
      segments: { results: [], total: 0, perPage: 20 },
      queryParams: {
        page: 1,
        query: '',
        orderBy: 'created_at',
        order: 'desc',
      },
      previewSegment: {},
      previewData: { results: [], total: 0 },
      previewLoading: false,
      previewPage: 1,
      refreshingAll: false,
    };
  },

  methods: {
    onPageChange(p) {
      this.queryParams.page = p;
      this.getSegments();
    },

    onSort(field, direction) {
      this.queryParams.orderBy = field;
      this.queryParams.order = direction;
      this.getSegments();
    },

    showEditForm(segment) {
      this.curItem = segment;
      this.isFormVisible = true;
      this.isEditing = true;
    },

    showNewForm() {
      this.curItem = {
        name: '',
        description: '',
        conditions: { operator: 'and', conditions: [] },
        tags: [],
      };
      this.isFormVisible = true;
      this.isEditing = false;
    },

    formFinished() {
      this.isFormVisible = false;
      this.getSegments();
    },

    onFormClose() {
      if (this.$route.params.id) {
        this.$router.push({ name: 'segments' });
      }
    },

    getConditionSummary(conditions) {
      if (!conditions || !conditions.conditions || conditions.conditions.length === 0) {
        return this.$t('segments.noConditions');
      }
      const count = conditions.conditions.length;
      const op = conditions.operator === 'or' ? 'OR' : 'AND';
      return `${count} condition${count > 1 ? 's' : ''} (${op})`;
    },

    getSegments() {
      this.$api.getSegments({
        page: this.queryParams.page,
        query: this.queryParams.query,
        order_by: this.queryParams.orderBy,
        order: this.queryParams.order,
      }).then((resp) => {
        this.segments = resp;
      });
    },

    refreshCount(segment) {
      this.$api.getSegmentCount(segment.id, false).then((resp) => {
        // Update the segment in the list
        const idx = this.segments.results.findIndex((s) => s.id === segment.id);
        if (idx !== -1) {
          this.segments.results[idx].cachedCount = resp.count;
        }
      });
    },

    async refreshAllCounts() {
      this.refreshingAll = true;
      try {
        // Refresh counts for all segments on the current page
        const promises = this.segments.results.map((segment) => this.$api.getSegmentCount(segment.id, false)
          .then((resp) => {
            const idx = this.segments.results.findIndex((s) => s.id === segment.id);
            if (idx !== -1) {
              this.segments.results[idx].cachedCount = resp.count;
            }
          }));
        await Promise.all(promises);
        this.$utils.toast(this.$t('segments.countsRefreshed'));
      } finally {
        this.refreshingAll = false;
      }
    },

    showPreview(segment) {
      this.previewSegment = segment;
      this.previewPage = 1;
      this.isPreviewVisible = true;
      this.loadPreview();
    },

    onPreviewPageChange(p) {
      this.previewPage = p;
      this.loadPreview();
    },

    loadPreview() {
      this.previewLoading = true;
      this.$api.previewSegment(this.previewSegment.id, { page: this.previewPage })
        .then((resp) => {
          this.previewData = resp;
        })
        .finally(() => {
          this.previewLoading = false;
        });
    },

    deleteSegment(segment) {
      this.$utils.confirm(
        this.$t('segments.confirmDelete'),
        () => {
          this.$api.deleteSegment(segment.id).then(() => {
            this.getSegments();
            this.$utils.toast(this.$t('globals.messages.deleted', { name: segment.name }));
          });
        },
      );
    },

    exportSegment(segment) {
      const count = segment.cachedCount !== null ? segment.cachedCount : 0;
      this.$utils.confirm(
        this.$t('segments.confirmExport', { num: this.$utils.formatNumber(count), name: segment.name }),
        () => {
          document.location.href = `/api/segments/${segment.id}/export`;
        },
      );
    },
  },

  computed: {
    ...mapState(['loading']),
  },

  mounted() {
    if (this.$route.params.id) {
      this.$api.getSegment(parseInt(this.$route.params.id, 10)).then((data) => {
        this.showEditForm(data);
      });
    } else {
      this.getSegments();
    }
  },
});
</script>
