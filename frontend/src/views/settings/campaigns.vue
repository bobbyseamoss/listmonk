<template>
  <div class="items">
    <h5 class="title is-5">Smart Sending</h5>
    <p class="help mb-3">
      Smart Sending prevents recipients from receiving too many campaigns within a short time period.
      When enabled, the system tracks the last send time for each recipient and skips sending if they
      received an email within the configured period.
    </p>

    <b-field label="Enable Smart Sending"
      message="When enabled, prevents recipients from receiving multiple campaigns within the Smart Sending Period">
      <b-switch v-model="data['app.smart_sending_enabled']" name="app.smart_sending_enabled" />
    </b-field>

    <div :class="{ disabled: !data['app.smart_sending_enabled'] }">
      <div class="columns">
        <div class="column is-6">
          <b-field label="Smart Sending Period (hours)" label-position="on-border"
            message="Minimum hours between campaign emails to the same recipient">
            <b-numberinput v-model="data['app.smart_sending_period_hours']"
              name="app.smart_sending_period_hours"
              type="is-light"
              placeholder="16"
              min="1"
              max="720"
              :disabled="!data['app.smart_sending_enabled']" />
          </b-field>
        </div>
      </div>
    </div>

    <b-notification v-if="data['app.smart_sending_enabled']" type="is-info" :closable="false" class="mt-3">
      <strong>Smart Sending is Active!</strong>
      Recipients will not receive more than one campaign email every {{ data['app.smart_sending_period_hours'] }} hour(s).
    </b-notification>
  </div>
</template>

<script>
import Vue from 'vue';

export default Vue.extend({
  props: {
    form: {
      type: Object, default: () => { },
    },
  },

  data() {
    return {
      data: this.form,
    };
  },
});
</script>
