<template>
  <div class="frequency-panel">
    <!-- Show Again Settings -->
    <div class="box">
      <h3 class="title is-5">Display Frequency</h3>
      <p class="help mb-4">
        Control how often visitors see this form.
      </p>

      <b-field label="If closed, show again after">
        <div class="is-flex is-align-items-center">
          <b-numberinput
            v-model="localValue.showAgainAfterDays"
            min="0"
            max="365"
            step="1"
            controls-position="compact"
            style="width: 120px"
            @input="emitChange"
          />
          <span class="ml-2">days</span>
        </div>
      </b-field>
      <p class="help">Set to 0 to show on every page load (not recommended).</p>

      <hr />

      <b-field label="Maximum impressions per visitor">
        <div class="is-flex is-align-items-center">
          <b-numberinput
            v-model="localValue.maxImpressions"
            min="0"
            max="100"
            step="1"
            controls-position="compact"
            style="width: 120px"
            @input="emitChange"
          />
        </div>
      </b-field>
      <p class="help">Set to 0 for unlimited. Form won't appear after this many views.</p>
    </div>

    <!-- Submission Suppression -->
    <div class="box">
      <h3 class="title is-5">After Submission</h3>
      <p class="help mb-4">
        What happens after a visitor submits the form?
      </p>

      <b-field>
        <b-switch v-model="localValue.suppressAfterSubmission" @input="emitChange">
          Don't show form again after submission
        </b-switch>
      </b-field>

      <div v-if="!localValue.suppressAfterSubmission" class="mt-4">
        <b-field label="If not suppressed, show again after">
          <div class="is-flex is-align-items-center">
            <b-numberinput
              v-model="localValue.suppressDays"
              min="1"
              max="365"
              step="1"
              controls-position="compact"
              style="width: 120px"
              @input="emitChange"
            />
            <span class="ml-2">days</span>
          </div>
        </b-field>
      </div>
    </div>

    <!-- Collision Prevention -->
    <div class="box">
      <h3 class="title is-5">Form Priority</h3>
      <p class="help mb-4">
        When multiple forms could show on the same page, higher priority forms take precedence.
      </p>

      <b-field label="Priority (1-10)">
        <b-slider
          v-model="localValue.priority"
          :min="1"
          :max="10"
          :step="1"
          :tooltip="true"
          @input="emitChange"
        />
      </b-field>
      <p class="help">
        10 = highest priority. Only one form shows at a time.
      </p>
    </div>

    <!-- Cookie Settings -->
    <div class="box">
      <h3 class="title is-5">Cookie Duration</h3>
      <p class="help mb-4">
        How long to remember visitor's suppression preferences.
      </p>

      <b-field label="Cookie expiry">
        <div class="is-flex is-align-items-center">
          <b-numberinput
            v-model="localValue.cookieDays"
            min="1"
            max="365"
            step="1"
            controls-position="compact"
            style="width: 120px"
            @input="emitChange"
          />
          <span class="ml-2">days</span>
        </div>
      </b-field>
      <p class="help">
        After this period, the form may appear again even if previously closed or submitted.
      </p>
    </div>
  </div>
</template>

<script>
import Vue from 'vue';

export default Vue.extend({
  name: 'FormFrequencyPanel',

  props: {
    value: {
      type: Object,
      required: true,
    },
  },

  data() {
    return {
      localValue: {
        showAgainAfterDays: 7,
        maxImpressions: 0,
        suppressAfterSubmission: true,
        suppressDays: 30,
        priority: 5,
        cookieDays: 30,
        ...this.value,
      },
    };
  },

  watch: {
    value: {
      handler(newVal) {
        this.localValue = { ...this.localValue, ...newVal };
      },
      deep: true,
    },
  },

  methods: {
    emitChange() {
      this.$emit('input', { ...this.localValue });
    },
  },
});
</script>

<style lang="scss" scoped>
.frequency-panel {
  .box {
    margin-bottom: 1.5rem;
  }
}
</style>
