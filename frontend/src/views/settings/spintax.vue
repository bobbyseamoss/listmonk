<template>
  <div>
    <h3 class="title is-5">{{ $t('settings.spintax.title', 'Spintax') }}</h3>
    <p class="has-text-grey mb-4">
      {{ $t('settings.spintax.description', 'Enable spintax text variations in email templates. Spintax syntax: {option1|option2|option3} randomly selects one option for each recipient.') }}
    </p>

    <div class="columns mb-5">
      <div class="column is-3">
        <b-field :label="$t('settings.spintax.enable', 'Enable Spintax')">
          <b-switch v-model="enabled" name="spintax.enabled" />
        </b-field>
      </div>
    </div>

    <div v-if="enabled" class="box">
      <h4 class="title is-6 mb-4">{{ $t('settings.spintax.aiTitle', 'AI-Powered Spintax Generation') }}</h4>
      <p class="has-text-grey mb-4">
        {{ $t('settings.spintax.aiDescription', 'Use Claude AI to automatically generate spintax variations for your email content.') }}
      </p>

      <div class="columns mb-4">
        <div class="column is-3">
          <b-field :label="$t('settings.spintax.aiEnable', 'Enable AI Generation')">
            <b-switch v-model="aiEnabled" name="spintax.ai.enabled" />
          </b-field>
        </div>
      </div>

      <div v-if="aiEnabled">
        <div class="columns mb-4">
          <div class="column">
            <b-field
              :label="$t('settings.spintax.apiKey', 'Claude API Key')"
              :message="$t('settings.spintax.apiKeyHelp', 'Your Anthropic API key for Claude. Leave blank to keep existing key.')">
              <b-input
                v-model="apiKey"
                type="password"
                name="api_key"
                placeholder="sk-ant-..." />
            </b-field>
          </div>
        </div>

        <div class="columns mb-4">
          <div class="column is-4">
            <b-field
              :label="$t('settings.spintax.variationLevel', 'Variation Level')"
              :message="$t('settings.spintax.variationLevelHelp', 'How much variety should the AI generate? Higher levels create more variations.')">
              <b-select
                v-model="variationLevel"
                name="variation_level"
                expanded>
                <option :value="1">1 - Minimal</option>
                <option :value="2">2 - Light</option>
                <option :value="3">3 - Medium</option>
                <option :value="4">4 - High</option>
                <option :value="5">5 - Maximum</option>
              </b-select>
            </b-field>
          </div>
        </div>
      </div>

      <div class="notification is-info is-light">
        <p><strong>{{ $t('settings.spintax.syntaxTitle', 'Spintax Syntax:') }}</strong></p>
        <ul>
          <li><code>{Hello|Hi|Hey}</code> - {{ $t('settings.spintax.syntaxBasic', 'Randomly picks one greeting') }}</li>
          <li><code>{Great|Awesome} {news|update}!</code> - {{ $t('settings.spintax.syntaxMultiple', 'Multiple spintax blocks combine') }}</li>
          <li><code>{Hello {friend|there}|Hi}</code> - {{ $t('settings.spintax.syntaxNested', 'Nested spintax is supported') }}</li>
        </ul>
      </div>

      <div class="notification is-warning is-light mt-4">
        <p><strong>{{ $t('settings.spintax.note', 'Note:') }}</strong></p>
        <p>{{ $t('settings.spintax.noteText', 'Spintax is processed at send time. Each recipient receives a unique variation. Preview shows one possible variation.') }}</p>
      </div>
    </div>
  </div>
</template>

<script>
import Vue from 'vue';

export default Vue.extend({
  name: 'Spintax',
  props: {
    form: {
      type: Object,
      default: () => ({}),
    },
  },
  computed: {
    enabled: {
      get() {
        return this.form.spintax?.enabled || false;
      },
      set(value) {
        if (!this.form.spintax) {
          this.$set(this.form, 'spintax', { enabled: false, ai: { enabled: false, api_key: '', variation_level: 3 } });
        }
        this.$set(this.form.spintax, 'enabled', value);
      },
    },
    aiEnabled: {
      get() {
        return this.form.spintax?.ai?.enabled || false;
      },
      set(value) {
        if (!this.form.spintax) {
          this.$set(this.form, 'spintax', { enabled: false, ai: { enabled: false, api_key: '', variation_level: 3 } });
        }
        if (!this.form.spintax.ai) {
          this.$set(this.form.spintax, 'ai', { enabled: false, api_key: '', variation_level: 3 });
        }
        this.$set(this.form.spintax.ai, 'enabled', value);
      },
    },
    apiKey: {
      get() {
        return this.form.spintax?.ai?.api_key || '';
      },
      set(value) {
        if (!this.form.spintax?.ai) {
          this.$set(this.form.spintax, 'ai', { enabled: false, api_key: '', variation_level: 3 });
        }
        this.$set(this.form.spintax.ai, 'api_key', value);
      },
    },
    variationLevel: {
      get() {
        return this.form.spintax?.ai?.variation_level || 3;
      },
      set(value) {
        if (!this.form.spintax?.ai) {
          this.$set(this.form.spintax, 'ai', { enabled: false, api_key: '', variation_level: 3 });
        }
        this.$set(this.form.spintax.ai, 'variation_level', value);
      },
    },
  },
});
</script>
