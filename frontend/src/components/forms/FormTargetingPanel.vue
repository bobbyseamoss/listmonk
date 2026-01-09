<template>
  <div class="targeting-panel">
    <!-- URL Targeting -->
    <div class="box">
      <h3 class="title is-5">URL Targeting</h3>
      <p class="help mb-4">
        Show this form only on pages matching these URL patterns.
      </p>

      <b-field label="Show on URLs matching">
        <b-taginput
          v-model="localValue.urlPatterns"
          placeholder="e.g., /products/*, /blog/*"
          @input="emitChange"
        />
      </b-field>

      <b-field label="Exclude URLs matching">
        <b-taginput
          v-model="localValue.excludeUrlPatterns"
          placeholder="e.g., /checkout, /cart"
          @input="emitChange"
        />
      </b-field>
    </div>

    <!-- UTM Parameters -->
    <div class="box">
      <h3 class="title is-5">UTM Parameters</h3>
      <p class="help mb-4">
        Target visitors based on UTM campaign parameters.
      </p>

      <div v-for="(param, index) in localValue.utmParams" :key="index" class="columns mb-2">
        <div class="column is-3">
          <b-select v-model="param.key" expanded @input="emitChange">
            <option value="utm_source">utm_source</option>
            <option value="utm_medium">utm_medium</option>
            <option value="utm_campaign">utm_campaign</option>
            <option value="utm_content">utm_content</option>
            <option value="utm_term">utm_term</option>
          </b-select>
        </div>
        <div class="column is-2">
          <b-select v-model="param.operator" expanded @input="emitChange">
            <option value="equals">equals</option>
            <option value="contains">contains</option>
            <option value="starts_with">starts with</option>
          </b-select>
        </div>
        <div class="column">
          <b-input v-model="param.value" placeholder="Value" @input="emitChange" />
        </div>
        <div class="column is-narrow">
          <b-button type="is-danger" icon-left="delete" @click="removeUtmParam(index)" />
        </div>
      </div>

      <b-button type="is-light" icon-left="plus" size="is-small" @click="addUtmParam">
        Add UTM Rule
      </b-button>
    </div>

    <!-- Device Targeting -->
    <div class="box">
      <h3 class="title is-5">Device Targeting</h3>
      <p class="help mb-4">
        Show this form only on specific device types.
      </p>

      <div class="buttons">
        <b-button
          v-for="device in ['desktop', 'tablet', 'mobile']"
          :key="device"
          :type="localValue.deviceTypes.includes(device) ? 'is-primary' : 'is-light'"
          :icon-left="deviceIcons[device]"
          @click="toggleDevice(device)"
        >
          {{ device.charAt(0).toUpperCase() + device.slice(1) }}
        </b-button>
      </div>
    </div>

    <!-- Geographic Targeting -->
    <div class="box">
      <h3 class="title is-5">Geographic Targeting</h3>
      <p class="help mb-4">
        Target visitors from specific countries (based on IP address).
      </p>

      <b-field label="Show only in countries">
        <b-taginput
          v-model="localValue.countries"
          placeholder="e.g., US, UK, CA"
          @input="emitChange"
        />
      </b-field>
    </div>

    <!-- Subscriber Exclusion -->
    <div class="box">
      <h3 class="title is-5">Subscriber Exclusion</h3>

      <b-field>
        <b-checkbox v-model="localValue.excludeSubscribers" @input="emitChange">
          Don't show to existing subscribers
        </b-checkbox>
      </b-field>
      <p class="help">
        If enabled, visitors who are already subscribed (based on cookies) won't see this form.
      </p>
    </div>
  </div>
</template>

<script>
import Vue from 'vue';

export default Vue.extend({
  name: 'FormTargetingPanel',

  props: {
    value: {
      type: Object,
      required: true,
    },
  },

  data() {
    return {
      localValue: { ...this.value },
      deviceIcons: {
        desktop: 'monitor',
        tablet: 'tablet',
        mobile: 'cellphone',
      },
    };
  },

  watch: {
    value: {
      handler(newVal) {
        this.localValue = { ...newVal };
      },
      deep: true,
    },
  },

  methods: {
    emitChange() {
      this.$emit('input', { ...this.localValue });
    },

    addUtmParam() {
      if (!this.localValue.utmParams) {
        this.localValue.utmParams = [];
      }
      this.localValue.utmParams.push({
        key: 'utm_source',
        operator: 'equals',
        value: '',
      });
      this.emitChange();
    },

    removeUtmParam(index) {
      this.localValue.utmParams.splice(index, 1);
      this.emitChange();
    },

    toggleDevice(device) {
      const idx = this.localValue.deviceTypes.indexOf(device);
      if (idx >= 0) {
        // Don't allow removing all devices
        if (this.localValue.deviceTypes.length > 1) {
          this.localValue.deviceTypes.splice(idx, 1);
        }
      } else {
        this.localValue.deviceTypes.push(device);
      }
      this.emitChange();
    },
  },
});
</script>

<style lang="scss" scoped>
.targeting-panel {
  .box {
    margin-bottom: 1.5rem;
  }
}
</style>
