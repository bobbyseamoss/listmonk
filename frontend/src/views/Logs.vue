<template>
  <section class="logs content relative">
    <h1 class="title is-4">
      {{ $t('logs.title') }}
    </h1>
    <hr />
    <log-view :loading="loading.logs" :lines="lines" />
    <div class="log-export-footer">
      <b-button type="is-primary" icon-left="download" @click="exportLogs">
        Export Logs
      </b-button>
    </div>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import LogView from '../components/LogView.vue';

export default Vue.extend({
  components: {
    LogView,
  },

  data() {
    return {
      lines: [],
      pollId: null,
    };
  },

  methods: {
    getLogs() {
      this.$api.getLogs().then((data) => {
        this.lines = data;
      });
    },

    exportLogs() {
      // Create log content with the same formatting as displayed on the page
      const logContent = this.lines.join('\n');

      // Create a blob with the log content
      const blob = new Blob([logContent], { type: 'text/plain;charset=utf-8' });

      // Create a download link
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;

      // Generate filename with timestamp
      const now = new Date();
      const timestamp = now.toISOString().replace(/[:.]/g, '-').substring(0, 19);
      link.download = `listmonk-logs-${timestamp}.txt`;

      // Trigger download
      document.body.appendChild(link);
      link.click();

      // Cleanup
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);

      // Show success notification
      this.$buefy.toast.open({
        message: 'Logs exported successfully',
        type: 'is-success',
        duration: 3000,
      });
    },
  },

  computed: {
    ...mapState(['logs', 'loading']),
  },

  mounted() {
    this.getLogs();

    // Update the logs every 10 seconds.
    this.pollId = setInterval(() => this.getLogs(), 10000);
  },

  destroyed() {
    clearInterval(this.pollId);
  },
});
</script>

<style scoped>
.log-export-footer {
  margin-top: 1rem;
  text-align: left;
}
</style>
