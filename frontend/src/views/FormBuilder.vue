<template>
  <section class="form-builder">
    <header class="columns page-header">
      <div class="column is-10">
        <h1 class="title is-4">
          {{ isEditing ? 'Edit Form' : 'New Form' }}
          <span v-if="form.name" class="has-text-grey-light"> &mdash; {{ form.name }}</span>
        </h1>
      </div>
      <div class="column has-text-right">
        <b-button type="is-primary" icon-left="content-save-outline" :loading="loading.save" @click="handleSave">
          {{ $t('globals.buttons.save') }}
        </b-button>
      </div>
    </header>

    <b-loading v-if="loading.form" :is-full-page="false" active />

    <div v-else class="builder-container">
      <b-tabs v-model="activeTab" type="is-boxed">
        <!-- Design Tab -->
        <b-tab-item label="Design" icon="pencil">
          <div class="iframe-container">
            <iframe
              ref="builderIframe"
              :src="builderUrl"
              frameborder="0"
              title="Form Builder"
              @load="onIframeLoad"
            />
          </div>
        </b-tab-item>

        <!-- Targeting Tab -->
        <b-tab-item label="Targeting" icon="target">
          <form-targeting-panel v-model="form.targeting" />
        </b-tab-item>

        <!-- Triggers Tab -->
        <b-tab-item label="Triggers" icon="lightning-bolt">
          <form-triggers-panel v-model="form.triggers" />
        </b-tab-item>

        <!-- Behavior Tab -->
        <b-tab-item label="Behavior" icon="cog">
          <form-frequency-panel v-model="form.frequency" />
        </b-tab-item>

        <!-- Lists Tab -->
        <b-tab-item label="Lists" icon="format-list-bulleted">
          <div class="box">
            <h3 class="title is-5">Subscribe to Lists</h3>
            <p class="help mb-4">
              Select which lists subscribers will be added to when they submit this form.
            </p>
            <list-selector
              :selected="selectedLists"
              :all="allLists"
              label="Lists"
              placeholder="Select lists..."
              @input="onListsChange"
            />
          </div>
        </b-tab-item>

        <!-- Embed Tab -->
        <b-tab-item v-if="isEditing" label="Embed" icon="code-tags">
          <form-embed-modal :form="form" />
        </b-tab-item>
      </b-tabs>
    </div>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';
import ListSelector from '../components/ListSelector.vue';
import FormTargetingPanel from '../components/forms/FormTargetingPanel.vue';
import FormTriggersPanel from '../components/forms/FormTriggersPanel.vue';
import FormFrequencyPanel from '../components/forms/FormFrequencyPanel.vue';
import FormEmbedModal from '../components/forms/FormEmbedModal.vue';

export default Vue.extend({
  name: 'FormBuilder',

  components: {
    ListSelector,
    FormTargetingPanel,
    FormTriggersPanel,
    FormFrequencyPanel,
    FormEmbedModal,
  },

  data() {
    return {
      activeTab: 0,
      form: {
        id: 0,
        uuid: '',
        name: 'New Form',
        description: '',
        formType: 'popup',
        status: 'draft',
        bodySource: {},
        customCss: '',
        steps: [],
        settings: {},
        targeting: {
          urlPatterns: [],
          excludeUrlPatterns: [],
          utmParams: [],
          deviceTypes: ['desktop', 'tablet', 'mobile'],
          countries: [],
          excludeSubscribers: true,
        },
        triggers: {
          type: 'time',
          delay: 3,
          scrollDepth: 50,
          exitIntentEnabled: false,
          pageCount: 1,
        },
        frequency: {
          showAgainAfterDays: 7,
          suppressAfterSubmission: true,
          suppressDays: 30,
          maxImpressions: 0,
          priority: 1,
        },
        listIds: [],
        couponConfig: {},
      },
      loading: {
        form: false,
        save: false,
        lists: false,
      },
      iframeReady: false,
      formDataLoaded: false,
    };
  },

  computed: {
    ...mapState(['lists']),

    isEditing() {
      return this.$route.params.id !== undefined;
    },

    builderUrl() {
      // In development, form-builder runs on a separate port
      if (process.env.NODE_ENV === 'development') {
        return 'http://localhost:5174';
      }
      return '/admin/static/form-builder/index.html';
    },

    // Get all lists as array for ListSelector
    allLists() {
      return this.lists && this.lists.results ? this.lists.results : [];
    },

    // Convert form.listIds (array of IDs) to array of list objects for ListSelector
    selectedLists() {
      if (!this.form.listIds || !this.allLists.length) {
        return [];
      }
      return this.allLists.filter((list) => this.form.listIds.includes(list.id));
    },
  },

  mounted() {
    // Listen for messages from the form builder iframe
    window.addEventListener('message', this.handleIframeMessage);

    // Load lists
    this.loadLists();

    // Load form if editing
    if (this.isEditing) {
      this.loadForm();
    }
  },

  beforeDestroy() {
    window.removeEventListener('message', this.handleIframeMessage);
  },

  methods: {
    async loadLists() {
      this.loading.lists = true;
      try {
        await this.$store.dispatch('lists/get');
      } catch (err) {
        this.$utils.toast(err.message, 'is-danger');
      } finally {
        this.loading.lists = false;
      }
    },

    async loadForm() {
      this.loading.form = true;
      try {
        const resp = await this.$api.getForm(this.$route.params.id);
        // Map snake_case backend response to camelCase frontend properties
        this.form = {
          ...this.form,
          id: resp.id,
          uuid: resp.uuid,
          name: resp.name,
          description: resp.description,
          formType: resp.formType || resp.form_type || this.form.formType,
          status: resp.status,
          bodySource: resp.bodySource || resp.body_source || {},
          customCss: resp.customCss || resp.custom_css || '',
          steps: resp.steps || [],
          settings: resp.settings || {},
          targeting: resp.targeting || this.form.targeting,
          triggers: resp.triggers || this.form.triggers,
          frequency: resp.frequency || this.form.frequency,
          listIds: resp.listIds || resp.list_ids || [],
          couponConfig: resp.couponConfig || resp.coupon_config || {},
        };
        this.formDataLoaded = true;
        // If iframe is already ready, send the data now
        this.trySendFormToBuilder();
      } catch (err) {
        this.$utils.toast(err.message, 'is-danger');
        this.$router.push({ name: 'signupForms' });
      } finally {
        this.loading.form = false;
      }
    },

    onIframeLoad() {
      // Iframe loaded, but wait for FORM_BUILDER_READY message from React app
    },

    handleIframeMessage(event) {
      if (!event.data || !event.data.type) return;

      switch (event.data.type) {
        case 'FORM_BUILDER_READY':
          // Builder is ready, mark iframe as ready and try to send form data
          this.iframeReady = true;
          // For new forms (not editing), mark data as loaded since we use defaults
          if (!this.isEditing) {
            this.formDataLoaded = true;
          }
          this.trySendFormToBuilder();
          break;

        case 'FORM_BUILDER_SAVE':
          // Builder sent updated form data - sync all fields
          this.form.bodySource = event.data.form;
          // Also sync individual fields from the form builder
          if (event.data.form) {
            this.form.name = event.data.form.name || this.form.name;
            this.form.formType = event.data.form.formType || this.form.formType;
            this.form.status = event.data.form.status || this.form.status;
            this.form.steps = event.data.form.steps || this.form.steps;
            this.form.settings = event.data.form.settings || this.form.settings;
            this.form.customCss = event.data.form.customCss || this.form.customCss;
          }
          break;

        case 'FORM_BUILDER_CANCEL':
          // User cancelled in builder
          this.$router.push({ name: 'signupForms' });
          break;

        default:
          // Unknown message type, ignore
          break;
      }
    },

    trySendFormToBuilder() {
      // Only send data when both iframe is ready AND form data is loaded (or it's a new form)
      if (!this.iframeReady || !this.formDataLoaded) {
        return;
      }
      this.sendFormToBuilder();
    },

    sendFormToBuilder() {
      const iframe = this.$refs.builderIframe;
      if (iframe && iframe.contentWindow) {
        iframe.contentWindow.postMessage({
          type: 'FORM_BUILDER_INIT',
          form: this.form.bodySource,
        }, '*');
      }
    },

    async handleSave() {
      this.loading.save = true;

      try {
        // Wait for the form builder to send its current state
        const builderFormData = await this.requestFormBuilderData();

        // Use form builder data directly for the payload
        const payload = {
          name: builderFormData.name || this.form.name || 'New Form',
          description: this.form.description,
          form_type: builderFormData.formType || this.form.formType || 'popup',
          status: builderFormData.status || this.form.status || 'draft',
          body_source: builderFormData,
          custom_css: builderFormData.customCss || this.form.customCss || '',
          steps: builderFormData.steps || this.form.steps || [],
          settings: builderFormData.settings || this.form.settings || {},
          targeting: this.form.targeting,
          triggers: this.form.triggers,
          frequency: this.form.frequency,
          list_ids: this.form.listIds,
          coupon_config: builderFormData.couponConfig || this.form.couponConfig || {},
        };

        if (this.isEditing) {
          await this.$api.updateForm(this.form.id, payload);
          this.$utils.toast(this.$t('globals.messages.updated', { name: payload.name }), 'is-success');
        } else {
          const resp = await this.$api.createForm(payload);
          this.$utils.toast(this.$t('globals.messages.created', { name: payload.name }), 'is-success');
          this.$router.push({ name: 'editSignupForm', params: { id: resp.id } });
        }
      } catch (err) {
        this.$utils.toast(err.message, 'is-danger');
      } finally {
        this.loading.save = false;
      }
    },

    requestFormBuilderData() {
      return new Promise((resolve) => {
        let handler = null;
        const timeout = setTimeout(() => {
          window.removeEventListener('message', handler);
          // Fall back to current form data if no response
          resolve(this.form.bodySource || {});
        }, 2000);

        handler = (event) => {
          if (event.data && event.data.type === 'FORM_BUILDER_SAVE') {
            clearTimeout(timeout);
            window.removeEventListener('message', handler);
            resolve(event.data.form || {});
          }
        };

        window.addEventListener('message', handler);

        // Request the form data from the builder
        const iframe = this.$refs.builderIframe;
        if (iframe && iframe.contentWindow) {
          iframe.contentWindow.postMessage({
            type: 'FORM_BUILDER_REQUEST_SAVE',
          }, '*');
        } else {
          clearTimeout(timeout);
          window.removeEventListener('message', handler);
          resolve(this.form.bodySource || {});
        }
      });
    },

    // Handle list selection changes - convert list objects to IDs
    onListsChange(selectedListObjects) {
      this.form.listIds = selectedListObjects.map((list) => list.id);
    },
  },
});
</script>

<style lang="scss" scoped>
.form-builder {
  .builder-container {
    background: #fff;
    border-radius: 4px;
    padding: 1rem;
  }

  .iframe-container {
    height: calc(100vh - 280px);
    min-height: 500px;

    iframe {
      width: 100%;
      height: 100%;
      border: 1px solid #e5e7eb;
      border-radius: 4px;
    }
  }

  .box {
    margin-bottom: 1rem;
  }
}
</style>
