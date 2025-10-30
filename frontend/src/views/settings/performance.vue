<template>
  <div class="items">
    <b-field :label="$t('settings.performance.concurrency')" label-position="on-border"
      :message="$t('settings.performance.concurrencyHelp')">
      <b-numberinput v-model="data['app.concurrency']" name="app.concurrency" type="is-light" placeholder="5" min="1"
        max="10000" />
    </b-field>

    <b-field :label="$t('settings.performance.messageRate')" label-position="on-border"
      :message="$t('settings.performance.messageRateHelp')">
      <b-numberinput v-model="data['app.message_rate']" name="app.message_rate" type="is-light" placeholder="5" min="1"
        max="100000" />
    </b-field>

    <b-field :label="$t('settings.performance.batchSize')" label-position="on-border"
      :message="$t('settings.performance.batchSizeHelp')">
      <b-numberinput v-model="data['app.batch_size']" name="app.batch_size" type="is-light" placeholder="1000" min="1"
        max="100000" />
    </b-field>

    <b-field :label="$t('settings.performance.maxErrThreshold')" label-position="on-border"
      :message="$t('settings.performance.maxErrThresholdHelp')">
      <b-numberinput v-model="data['app.max_send_errors']" name="app.max_send_errors" type="is-light" placeholder="1999"
        min="0" max="100000" />
    </b-field>

    <div>
      <div class="columns">
        <div class="column is-6">
          <b-field :label="$t('settings.performance.slidingWindow')"
            :message="$t('settings.performance.slidingWindowHelp')">
            <b-switch v-model="data['app.message_sliding_window']" name="app.message_sliding_window" />
          </b-field>
        </div>

        <div class="column is-3" :class="{ disabled: !data['app.message_sliding_window'] }">
          <b-field :label="$t('settings.performance.slidingWindowRate')" label-position="on-border"
            :message="$t('settings.performance.slidingWindowRateHelp')">
            <b-numberinput v-model="data['app.message_sliding_window_rate']" name="sliding_window_rate" type="is-light"
              controls-position="compact" :disabled="!data['app.message_sliding_window']" placeholder="25" min="1"
              max="10000000" />
          </b-field>
        </div>

        <div class="column is-3" :class="{ disabled: !data['app.message_sliding_window'] }">
          <b-field :label="$t('settings.performance.slidingWindowDuration')" label-position="on-border"
            :message="$t('settings.performance.slidingWindowDurationHelp')">
            <b-input v-model="data['app.message_sliding_window_duration']" name="sliding_window_duration"
              :disabled="!data['app.message_sliding_window']" placeholder="1h" :pattern="regDuration" :maxlength="10" />
          </b-field>
        </div>
      </div>
    </div><!-- sliding window -->

    <div>
      <hr />
      <h5 class="title is-5">Sending Time Window</h5>
      <p class="help mb-3">Restrict email sending to specific hours of the day (useful for avoiding night-time sends)</p>
      <div class="columns">
        <div class="column is-6">
          <b-field label="Send Start Time (24h format)" label-position="on-border"
            message="Time to start sending emails each day (e.g., 08:00). Leave empty for 24/7 sending.">
            <b-input v-model="data['app.send_time_start']" name="app.send_time_start"
              placeholder="08:00" pattern="[0-2][0-9]:[0-5][0-9]" :maxlength="5" />
          </b-field>
        </div>
        <div class="column is-6">
          <b-field label="Send End Time (24h format)" label-position="on-border"
            message="Time to stop sending emails each day (e.g., 20:00). Leave empty for 24/7 sending.">
            <b-input v-model="data['app.send_time_end']" name="app.send_time_end"
              placeholder="20:00" pattern="[0-2][0-9]:[0-5][0-9]" :maxlength="5" />
          </b-field>
        </div>
      </div>
    </div><!-- time window -->

    <div>
      <hr />
      <h5 class="title is-5">Account-Wide Speed Limits</h5>
      <p class="help mb-3">
        <strong>⚠️ Critical:</strong> These limits apply to ALL SMTP servers combined (subscription-wide limit).
        This is the primary rate limit that will be enforced strictly to prevent Azure subscription throttling.
        Set these to match your email service provider's subscription limits.
      </p>
      <div class="columns">
        <div class="column is-6">
          <b-field label="Emails Per Minute (Account-Wide)" label-position="on-border"
            message="Maximum total emails across ALL servers per minute. Azure default: 30">
            <b-numberinput v-model="data['app.account_rate_limit_per_minute']" name="app.account_rate_limit_per_minute"
              type="is-light" placeholder="30" min="1" max="1000" />
          </b-field>
        </div>
        <div class="column is-6">
          <b-field label="Emails Per Hour (Account-Wide)" label-position="on-border"
            message="Maximum total emails across ALL servers per hour. Azure default: 100">
            <b-numberinput v-model="data['app.account_rate_limit_per_hour']" name="app.account_rate_limit_per_hour"
              type="is-light" placeholder="100" min="1" max="100000" />
          </b-field>
        </div>
      </div>
      <b-notification type="is-info" :closable="false" class="mt-2">
        <strong>Note:</strong> These account-wide limits take precedence over per-server sliding window limits.
        The queue processor will send emails sequentially to ensure these limits are never exceeded.
      </b-notification>
    </div><!-- account-wide rate limits -->

    <div>
      <hr />
      <h5 class="title is-5">Testing Mode</h5>
      <b-field label="Enable Testing Mode"
        message="⚠️ When enabled, emails will be simulated but NOT actually sent to recipients. Use this to safely test queue functionality without affecting your mail server reputation.">
        <b-switch v-model="data['app.testing_mode']" name="app.testing_mode" type="is-warning" />
      </b-field>
      <b-notification v-if="data['app.testing_mode']" type="is-warning" :closable="false" class="mt-3">
        <strong>Testing Mode is Active!</strong> No emails will be sent to actual recipients. All sends will be simulated.
      </b-notification>
    </div><!-- testing mode -->

    <div>
      <hr />
      <div class="columns">
        <div class="column is-4">
          <b-field :label="$t('settings.performance.cacheSlowQueries')"
            :message="$t('settings.performance.cacheSlowQueriesHelp')">
            <b-switch v-model="data['app.cache_slow_queries']" name="app.cache_slow_queries" />
          </b-field>
        </div>
        <div class="column is-4" :class="{ disabled: !data['app.cache_slow_queries'] }">
          <b-field :label="$t('settings.maintenance.cron')">
            <b-input v-model="data['app.cache_slow_queries_interval']" :disabled="!data['app.cache_slow_queries']"
              placeholder="0 3 * * *" />
          </b-field>
        </div>
        <div class="column">
          <br /><br />
          <a href="https://listmonk.app/docs/maintenance/performance/" target="_blank" rel="noopener noreferer">
            <b-icon icon="link-variant" /> {{ $t('globals.buttons.learnMore') }}
          </a>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import Vue from 'vue';
import { regDuration } from '../../constants';

export default Vue.extend({
  props: {
    form: {
      type: Object, default: () => { },
    },
  },

  data() {
    return {
      data: this.form,
      regDuration,
    };
  },
});
</script>
