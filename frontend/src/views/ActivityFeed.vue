<template>
  <section class="activity-feed">
    <header class="columns page-header">
      <div class="column is-two-thirds">
        <h1 class="title is-4">
          Activity feed
          <span v-if="feed.total > 0" class="has-text-grey-light">({{ feed.total.toLocaleString() }})</span>
        </h1>
      </div>
    </header>

    <!-- Filters -->
    <div class="columns mb-4">
      <div class="column is-3">
        <b-field label="Showing feed for">
          <b-select v-model="queryParams.eventType" expanded @input="onFilterChange">
            <option value="">All metrics</option>
            <option value="email_open">Email Opens</option>
            <option value="email_click">Email Clicks</option>
            <option value="site_activity">Site Activity</option>
          </b-select>
        </b-field>
      </div>
    </div>

    <b-table
      :data="feed.results"
      :hoverable="true"
      :loading="loading.feed"
      detailed
      show-detail-icon
      detail-key="id"
      paginated
      backend-pagination
      pagination-position="both"
      @page-change="onPageChange"
      :current-page="queryParams.page"
      :per-page="feed.perPage"
      :total="feed.total"
    >
      <b-table-column v-slot="props" field="profile" label="Profile" width="70%">
        <div class="profile-cell">
          <router-link
            v-if="props.row.subscriberId"
            :to="`/subscribers/${props.row.subscriberId}`"
            class="subscriber-name"
          >
            {{ getDisplayName(props.row) }}
          </router-link>
          <span v-else class="subscriber-name anonymous">
            {{ getDisplayName(props.row) }}
          </span>
          <div class="event-description has-text-grey">
            {{ props.row.eventDescription }}
          </div>
        </div>
      </b-table-column>

      <b-table-column v-slot="props" field="time" label="Time" width="25%">
        <span class="has-text-grey">{{ getRelativeTime(props.row.createdAt) }}</span>
      </b-table-column>

      <b-table-column v-slot="props" cell-class="actions" width="5%">
        <b-dropdown v-if="props.row.subscriberId || props.row.campaignId" role="list" position="is-bottom-left">
          <template #trigger>
            <b-icon icon="dots-vertical" class="is-clickable" />
          </template>
          <b-dropdown-item v-if="props.row.subscriberId" role="listitem" @click="viewSubscriber(props.row)">
            <b-icon icon="account" size="is-small" /> View subscriber
          </b-dropdown-item>
          <b-dropdown-item v-if="props.row.campaignId" role="listitem" @click="viewCampaign(props.row)">
            <b-icon icon="email" size="is-small" /> View campaign
          </b-dropdown-item>
        </b-dropdown>
      </b-table-column>

      <template #detail="props">
        <div class="content">
          <table class="table is-narrow">
            <tbody>
              <tr>
                <th>Event Type</th>
                <td><b-tag>{{ props.row.eventType }}</b-tag></td>
              </tr>
              <tr>
                <th>Email</th>
                <td>{{ props.row.subscriberEmail }}</td>
              </tr>
              <tr v-if="props.row.campaignName">
                <th>Campaign</th>
                <td>
                  <router-link :to="`/campaigns/${props.row.campaignId}`">
                    {{ props.row.campaignName }}
                  </router-link>
                </td>
              </tr>
              <tr v-if="props.row.pageUrl">
                <th>URL</th>
                <td>
                  <a :href="props.row.pageUrl" target="_blank" rel="noopener">
                    {{ props.row.pageUrl }}
                  </a>
                </td>
              </tr>
              <tr>
                <th>Timestamp</th>
                <td>{{ $utils.niceDate(props.row.createdAt, true) }}</td>
              </tr>
              <tr v-if="props.row.properties && Object.keys(props.row.properties).length > 0">
                <th>Properties</th>
                <td><pre class="is-size-7">{{ JSON.stringify(props.row.properties, null, 2) }}</pre></td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>

      <template #empty v-if="!loading.feed">
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
      feed: {
        results: [],
        total: 0,
        perPage: 20,
      },

      loading: {
        feed: false,
      },

      queryParams: {
        page: 1,
        eventType: '',
      },
    };
  },

  methods: {
    getDisplayName(row) {
      if (row.subscriberName && row.subscriberName.trim()) {
        return row.subscriberName;
      }
      // Extract name from email if no name
      const email = row.subscriberEmail || '';
      const name = email.split('@')[0];
      return name.charAt(0).toUpperCase() + name.slice(1);
    },

    getRelativeTime(dateStr) {
      const date = new Date(dateStr);
      const now = new Date();
      const diffMs = now - date;
      const diffSecs = Math.floor(diffMs / 1000);
      const diffMins = Math.floor(diffSecs / 60);
      const diffHours = Math.floor(diffMins / 60);
      const diffDays = Math.floor(diffHours / 24);

      if (diffSecs < 60) {
        return `${diffSecs} second${diffSecs !== 1 ? 's' : ''} ago`;
      }
      if (diffMins < 60) {
        return `${diffMins} minute${diffMins !== 1 ? 's' : ''} ago`;
      }
      if (diffHours < 24) {
        return `${diffHours} hour${diffHours !== 1 ? 's' : ''} ago`;
      }
      if (diffDays < 7) {
        return `${diffDays} day${diffDays !== 1 ? 's' : ''} ago`;
      }
      return this.$utils.niceDate(dateStr, false);
    },

    onPageChange(page) {
      this.queryParams.page = page;
      this.getFeed();
    },

    onFilterChange() {
      this.queryParams.page = 1;
      this.getFeed();
    },

    viewSubscriber(row) {
      this.$router.push(`/subscribers/${row.subscriberId}`);
    },

    viewCampaign(row) {
      if (row.campaignId) {
        this.$router.push(`/campaigns/${row.campaignId}`);
      }
    },

    getFeed() {
      this.loading.feed = true;

      const params = {
        page: this.queryParams.page,
      };

      if (this.queryParams.eventType) {
        params.event_type = this.queryParams.eventType;
      }

      this.$api.getActivityFeed(params).then((data) => {
        this.feed = data;
        this.loading.feed = false;
      }).catch(() => {
        this.loading.feed = false;
      });
    },
  },

  computed: {
    ...mapState(['serverConfig']),
  },

  mounted() {
    this.getFeed();
  },
});
</script>

<style scoped>
.profile-cell {
  display: flex;
  flex-direction: column;
}

.subscriber-name {
  font-weight: 500;
  color: #3273dc;
}

.subscriber-name:hover:not(.anonymous) {
  text-decoration: underline;
}

.subscriber-name.anonymous {
  color: #7a7a7a;
  font-style: italic;
}

.event-description {
  font-size: 0.9em;
  margin-top: 2px;
}

.actions {
  text-align: right;
}

pre {
  max-height: 200px;
  overflow: auto;
  background: #f5f5f5;
  padding: 0.5rem;
  border-radius: 4px;
}
</style>
