<template>
  <div class="triggers-panel">
    <!-- Primary Trigger -->
    <div class="box">
      <h3 class="title is-5">Primary Trigger</h3>
      <p class="help mb-4">
        When should the form appear?
      </p>

      <b-field>
        <b-radio-button v-model="localValue.type" native-value="immediate" type="is-primary is-light">
          <b-icon icon="flash" />
          <span>Immediately</span>
        </b-radio-button>
        <b-radio-button v-model="localValue.type" native-value="time" type="is-primary is-light">
          <b-icon icon="clock-outline" />
          <span>Time Delay</span>
        </b-radio-button>
        <b-radio-button v-model="localValue.type" native-value="scroll" type="is-primary is-light">
          <b-icon icon="arrow-down" />
          <span>Scroll Depth</span>
        </b-radio-button>
        <b-radio-button v-model="localValue.type" native-value="pages" type="is-primary is-light">
          <b-icon icon="file-multiple" />
          <span>Page Count</span>
        </b-radio-button>
        <b-radio-button v-model="localValue.type" native-value="manual" type="is-primary is-light">
          <b-icon icon="cursor-default-click" />
          <span>Manual</span>
        </b-radio-button>
      </b-field>

      <!-- Time Delay Settings -->
      <div v-if="localValue.type === 'time'" class="mt-4">
        <b-field label="Show after (seconds)">
          <b-numberinput
            v-model="localValue.delay"
            min="0"
            max="300"
            step="1"
            controls-position="compact"
            @input="emitChange"
          />
        </b-field>
      </div>

      <!-- Scroll Depth Settings -->
      <div v-if="localValue.type === 'scroll'" class="mt-4">
        <b-field label="Show when user scrolls past (%)">
          <b-slider
            v-model="localValue.scrollDepth"
            :min="10"
            :max="100"
            :step="5"
            :tooltip="true"
            @input="emitChange"
          />
        </b-field>
      </div>

      <!-- Page Count Settings -->
      <div v-if="localValue.type === 'pages'" class="mt-4">
        <b-field label="Show after viewing this many pages">
          <b-numberinput
            v-model="localValue.pageCount"
            min="1"
            max="20"
            step="1"
            controls-position="compact"
            @input="emitChange"
          />
        </b-field>
      </div>

      <!-- Manual Trigger Info -->
      <div v-if="localValue.type === 'manual'" class="mt-4 notification is-info is-light">
        <p>
          <b-icon icon="information" size="is-small" />
          Form will only appear when triggered via JavaScript API:
        </p>
        <pre class="mt-2">Listmonk.forms.show('{{ formUuid }}')</pre>
      </div>
    </div>

    <!-- Exit Intent -->
    <div class="box">
      <h3 class="title is-5">Exit Intent</h3>
      <p class="help mb-4">
        Show form when user moves cursor to leave the page (desktop only).
      </p>

      <b-field>
        <b-switch v-model="localValue.exitIntentEnabled" @input="emitChange">
          Enable exit intent trigger
        </b-switch>
      </b-field>

      <div v-if="localValue.exitIntentEnabled" class="mt-4 notification is-warning is-light">
        <b-icon icon="alert" size="is-small" />
        Exit intent is an <strong>additional</strong> trigger. The form may appear earlier based on your primary trigger.
      </div>
    </div>

    <!-- Mobile Alternative -->
    <div v-if="localValue.exitIntentEnabled" class="box">
      <h3 class="title is-5">Mobile Exit Intent</h3>
      <p class="help mb-4">
        Exit intent doesn't work on mobile. Choose an alternative trigger.
      </p>

      <b-field>
        <b-select v-model="localValue.mobileExitAlternative" expanded @input="emitChange">
          <option value="none">Don't show exit intent on mobile</option>
          <option value="scroll_up">Show when user scrolls up quickly</option>
          <option value="back_button">Show on back button press</option>
        </b-select>
      </b-field>
    </div>
  </div>
</template>

<script>
import Vue from 'vue';

export default Vue.extend({
  name: 'FormTriggersPanel',

  props: {
    value: {
      type: Object,
      required: true,
    },
    formUuid: {
      type: String,
      default: 'your-form-uuid',
    },
  },

  data() {
    return {
      localValue: { ...this.value },
    };
  },

  watch: {
    value: {
      handler(newVal) {
        this.localValue = { ...newVal };
      },
      deep: true,
    },
    'localValue.type'() {
      this.emitChange();
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
.triggers-panel {
  .box {
    margin-bottom: 1.5rem;
  }

  pre {
    background: #1a1a2e;
    color: #0f0;
    padding: 0.5rem 1rem;
    border-radius: 4px;
    font-size: 0.85rem;
  }
}
</style>
