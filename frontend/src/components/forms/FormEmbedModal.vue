<template>
  <div class="embed-panel">
    <div class="box">
      <h3 class="title is-5">Embed Code</h3>
      <p class="help mb-4">
        Add this code to your website to display the form.
      </p>

      <b-tabs v-model="activeTab" type="is-boxed">
        <!-- Auto Load Tab -->
        <b-tab-item label="Auto Load" icon="flash">
          <p class="mb-4">
            The form will automatically load and display based on its trigger and targeting rules.
          </p>
          <div class="code-block">
            <pre><code>{{ autoLoadCode }}</code></pre>
            <b-button
              type="is-light"
              size="is-small"
              icon-left="content-copy"
              class="copy-btn"
              @click="copyToClipboard(autoLoadCode)"
            >
              Copy
            </b-button>
          </div>
        </b-tab-item>

        <!-- Manual Trigger Tab -->
        <b-tab-item label="Manual Trigger" icon="cursor-default-click">
          <p class="mb-4">
            Trigger the form programmatically using JavaScript.
          </p>
          <div class="code-block">
            <pre><code>{{ manualTriggerCode }}</code></pre>
            <b-button
              type="is-light"
              size="is-small"
              icon-left="content-copy"
              class="copy-btn"
              @click="copyToClipboard(manualTriggerCode)"
            >
              Copy
            </b-button>
          </div>

          <h4 class="title is-6 mt-4">Button Example</h4>
          <div class="code-block">
            <pre><code>{{ buttonExample }}</code></pre>
            <b-button
              type="is-light"
              size="is-small"
              icon-left="content-copy"
              class="copy-btn"
              @click="copyToClipboard(buttonExample)"
            >
              Copy
            </b-button>
          </div>
        </b-tab-item>

        <!-- Embed Tab (for embed forms) -->
        <b-tab-item v-if="form.formType === 'embed'" label="Inline Embed" icon="code-tags">
          <p class="mb-4">
            Place this code where you want the form to appear on your page.
          </p>
          <div class="code-block">
            <pre><code>{{ inlineEmbedCode }}</code></pre>
            <b-button
              type="is-light"
              size="is-small"
              icon-left="content-copy"
              class="copy-btn"
              @click="copyToClipboard(inlineEmbedCode)"
            >
              Copy
            </b-button>
          </div>
        </b-tab-item>
      </b-tabs>
    </div>

    <!-- Form URL -->
    <div class="box">
      <h3 class="title is-5">Direct Link</h3>
      <p class="help mb-4">
        Share this URL to show the form directly.
      </p>

      <b-field>
        <b-input :value="directUrl" readonly expanded />
        <p class="control">
          <b-button type="is-primary" icon-left="content-copy" @click="copyToClipboard(directUrl)">
            Copy
          </b-button>
        </p>
      </b-field>
    </div>

    <!-- API Endpoint -->
    <div class="box">
      <h3 class="title is-5">API Information</h3>

      <b-field label="Form UUID">
        <b-input :value="form.uuid" readonly />
      </b-field>

      <b-field label="Submit Endpoint">
        <b-input :value="submitEndpoint" readonly />
      </b-field>
    </div>
  </div>
</template>

<script>
import Vue from 'vue';

export default Vue.extend({
  name: 'FormEmbedModal',

  props: {
    form: {
      type: Object,
      required: true,
    },
  },

  data() {
    return {
      activeTab: 0,
    };
  },

  computed: {
    baseUrl() {
      return window.location.origin;
    },

    autoLoadCode() {
      const endScript = '<' + '/script>';
      return `<!-- Listmonk Form -->
<script src="${this.baseUrl}/public/lm-forms.js" data-form="${this.form.uuid}">${endScript}`;
    },

    manualTriggerCode() {
      const endScript = '<' + '/script>';
      return `// Include the script first
<script src="${this.baseUrl}/public/lm-forms.js">${endScript}

// Then trigger the form
<script>
  Listmonk.forms.show('${this.form.uuid}');
${endScript}`;
    },

    buttonExample() {
      return `<button onclick="Listmonk.forms.show('${this.form.uuid}')">
  Subscribe
</button>`;
    },

    inlineEmbedCode() {
      const endScript = '<' + '/script>';
      return `<!-- Listmonk Embedded Form -->
<div id="lm-form-${this.form.uuid}"></div>
<script src="${this.baseUrl}/public/lm-forms.js"
  data-form="${this.form.uuid}"
  data-type="embed"
  data-target="#lm-form-${this.form.uuid}">${endScript}`;
    },

    directUrl() {
      return `${this.baseUrl}/subscription/form/${this.form.uuid}`;
    },

    submitEndpoint() {
      return `POST ${this.baseUrl}/api/public/forms/${this.form.uuid}/submit`;
    },
  },

  methods: {
    copyToClipboard(text) {
      navigator.clipboard.writeText(text).then(() => {
        this.$buefy.toast.open({
          message: 'Copied to clipboard',
          type: 'is-success',
        });
      }).catch(() => {
        this.$buefy.toast.open({
          message: 'Failed to copy',
          type: 'is-danger',
        });
      });
    },
  },
});
</script>

<style lang="scss" scoped>
.embed-panel {
  .box {
    margin-bottom: 1.5rem;
  }

  .code-block {
    position: relative;
    background: #1a1a2e;
    border-radius: 4px;
    padding: 1rem;
    margin-bottom: 1rem;

    pre {
      margin: 0;
      overflow-x: auto;
    }

    code {
      color: #e0e0e0;
      font-size: 0.85rem;
      white-space: pre-wrap;
      word-break: break-all;
    }

    .copy-btn {
      position: absolute;
      top: 0.5rem;
      right: 0.5rem;
    }
  }
}
</style>
