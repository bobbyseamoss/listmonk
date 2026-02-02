<template>
  <section class="flow-builder">
    <header class="columns page-header">
      <div class="column is-8">
        <h1 class="title is-4">
          {{ isEditing ? 'Edit Flow' : 'New Flow' }}
          <span v-if="flow.name" class="has-text-grey-light"> &mdash; {{ flow.name }}</span>
        </h1>
      </div>
      <div class="column has-text-right">
        <b-button type="is-light" tag="router-link" :to="{ name: 'flows' }" class="mr-2">
          Cancel
        </b-button>
        <b-button type="is-primary" icon-left="content-save-outline" :loading="loading.save" @click="handleSave">
          {{ $t('globals.buttons.save') }}
        </b-button>
      </div>
    </header>

    <b-loading v-if="loading.flow" :is-full-page="false" active />

    <div v-else class="builder-container">
      <div class="columns">
        <!-- Left Panel - Flow Settings -->
        <div class="column is-4">
          <div class="box">
            <h3 class="title is-5">Flow Settings</h3>

            <b-field label="Name">
              <b-input v-model="flow.name" placeholder="Flow name" required />
            </b-field>

            <b-field label="Description">
              <b-input v-model="flow.description" type="textarea" placeholder="Optional description" rows="2" />
            </b-field>

            <b-field label="Status">
              <b-select v-model="flow.status" expanded>
                <option value="draft">Draft</option>
                <option value="active">Active</option>
                <option value="paused">Paused</option>
              </b-select>
            </b-field>

            <b-field label="Trigger">
              <b-select v-model="flow.trigger_type" expanded required>
                <option value="list_subscription">List Subscription</option>
                <option value="segment_entry">Segment Entry</option>
                <option value="shopify_order_created">Shopify: Order Created</option>
                <option value="shopify_order_fulfilled">Shopify: Order Fulfilled</option>
                <option value="shopify_order_cancelled">Shopify: Order Cancelled</option>
                <option value="shopify_checkout_abandoned">Shopify: Abandoned Checkout</option>
                <option value="form_submission">Form Submission</option>
                <option value="date_property">Date Property</option>
              </b-select>
            </b-field>

            <!-- Trigger-specific filters -->
            <div v-if="flow.trigger_type" class="trigger-filters">
              <h4 class="title is-6 mt-4">Trigger Filters (Optional)</h4>
              <p class="help mb-3">Filter which events trigger this flow. Leave empty to trigger on all events of this type.</p>

              <!-- List Subscription Filter -->
              <b-field v-if="flow.trigger_type === 'list_subscription'" label="List">
                <b-select v-model="triggerFilters.list_id" expanded placeholder="Any list">
                  <option :value="null">Any list</option>
                  <option v-for="list in lists.results" :key="list.id" :value="list.id">
                    {{ list.name }}
                  </option>
                </b-select>
              </b-field>

              <!-- Shopify Order Filters -->
              <div v-if="flow.trigger_type.startsWith('shopify_order')">
                <b-field label="Minimum Order Value ($)">
                  <b-input v-model.number="triggerFilters.total_price_gte" type="number" min="0" step="0.01" />
                </b-field>
              </div>
            </div>

            <!-- Stats (only for existing flows) -->
            <div v-if="isEditing && flow.stats" class="stats-box mt-4">
              <h4 class="title is-6">Statistics</h4>
              <div class="columns is-multiline is-mobile">
                <div class="column is-6">
                  <p class="heading">Active Runs</p>
                  <p class="title is-5">{{ flow.stats.active_runs }}</p>
                </div>
                <div class="column is-6">
                  <p class="heading">Completed</p>
                  <p class="title is-5">{{ flow.stats.completed_runs }}</p>
                </div>
                <div class="column is-6">
                  <p class="heading">Emails Sent</p>
                  <p class="title is-5">{{ flow.stats.emails_sent }}</p>
                </div>
                <div class="column is-6">
                  <p class="heading">Failed</p>
                  <p class="title is-5">{{ flow.stats.failed_runs }}</p>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Right Panel - Steps -->
        <div class="column is-8">
          <div class="box">
            <div class="level">
              <div class="level-left">
                <h3 class="title is-5">Flow Steps</h3>
              </div>
              <div class="level-right">
                <b-dropdown position="is-bottom-left">
                  <template #trigger>
                    <b-button type="is-primary" icon-left="plus" size="is-small">
                      Add Step
                    </b-button>
                  </template>

                  <b-dropdown-item @click="addStep('email')">
                    <b-icon icon="email-outline" size="is-small" class="mr-2" />
                    Send Email
                  </b-dropdown-item>
                  <b-dropdown-item @click="addStep('delay')">
                    <b-icon icon="clock-outline" size="is-small" class="mr-2" />
                    Delay
                  </b-dropdown-item>
                  <b-dropdown-item @click="addStep('condition')">
                    <b-icon icon="account_tree" size="is-small" class="mr-2" />
                    Condition Split
                  </b-dropdown-item>
                  <b-dropdown-item @click="addStep('add_to_list')">
                    <b-icon icon="playlist-plus" size="is-small" class="mr-2" />
                    Add to List
                  </b-dropdown-item>
                  <b-dropdown-item @click="addStep('remove_from_list')">
                    <b-icon icon="playlist-minus" size="is-small" class="mr-2" />
                    Remove from List
                  </b-dropdown-item>
                  <b-dropdown-item @click="addStep('update_subscriber')">
                    <b-icon icon="account-edit-outline" size="is-small" class="mr-2" />
                    Update Subscriber
                  </b-dropdown-item>
                </b-dropdown>
              </div>
            </div>

            <!-- Steps List -->
            <div v-if="flow.steps && flow.steps.length > 0" class="steps-list">
              <div
                v-for="(step, index) in flow.steps"
                :key="step.id || `new-${index}`"
                class="step-card"
              >
                <div class="step-header">
                  <span class="step-number">{{ index + 1 }}</span>
                  <b-tag :type="stepTypeColors[step.step_type]" class="mr-2">
                    {{ stepTypeLabels[step.step_type] }}
                  </b-tag>
                  <span class="step-title">{{ getStepSummary(step) }}</span>
                  <div class="step-actions">
                    <b-button type="is-text" size="is-small" @click="editStep(index)">
                      <b-icon icon="pencil-outline" size="is-small" />
                    </b-button>
                    <b-button type="is-text" size="is-small" @click="removeStep(index)">
                      <b-icon icon="trash-can-outline" size="is-small" />
                    </b-button>
                  </div>
                </div>

                <!-- Step Details (expanded when editing) -->
                <div v-if="editingStepIndex === index" class="step-details">
                  <!-- Email Step -->
                  <template v-if="step.step_type === 'email'">
                    <b-field label="Template">
                      <b-select v-model="step.template_id" expanded>
                        <option :value="null">Select a template...</option>
                        <option v-for="tpl in templates" :key="tpl.id" :value="tpl.id">
                          {{ tpl.name }}
                        </option>
                      </b-select>
                    </b-field>
                    <b-field label="Subject (optional, overrides template)">
                      <b-input v-model="step.subject" placeholder="Email subject" />
                    </b-field>
                    <b-field label="From Email (optional)">
                      <b-input v-model="step.from_email" placeholder="sender@example.com" />
                    </b-field>
                  </template>

                  <!-- Delay Step -->
                  <template v-else-if="step.step_type === 'delay'">
                    <div class="columns">
                      <div class="column is-6">
                        <b-field label="Delay">
                          <b-input v-model.number="step.delay_value" type="number" min="1" />
                        </b-field>
                      </div>
                      <div class="column is-6">
                        <b-field label="Unit">
                          <b-select v-model="step.delay_unit" expanded>
                            <option value="minutes">Minutes</option>
                            <option value="hours">Hours</option>
                            <option value="days">Days</option>
                          </b-select>
                        </b-field>
                      </div>
                    </div>
                  </template>

                  <!-- Condition Step -->
                  <template v-else-if="step.step_type === 'condition'">
                    <b-field label="Condition Type">
                      <b-select v-model="step.condition_type" expanded>
                        <option value="trigger_data">Trigger Data</option>
                        <option value="profile_data">Profile Data</option>
                      </b-select>
                    </b-field>
                    <b-field label="Field">
                      <b-input v-model="step.condition_field" placeholder="e.g., total_price or email" />
                    </b-field>
                    <b-field label="Operator">
                      <b-select v-model="step.condition_operator" expanded>
                        <option value="equals">Equals</option>
                        <option value="not_equals">Not Equals</option>
                        <option value="contains">Contains</option>
                        <option value="not_contains">Not Contains</option>
                        <option value="greater_than">Greater Than</option>
                        <option value="less_than">Less Than</option>
                        <option value="exists">Exists</option>
                        <option value="not_exists">Not Exists</option>
                      </b-select>
                    </b-field>
                    <b-field label="Value">
                      <b-input v-model="step.condition_value" placeholder="Value to compare" />
                    </b-field>
                  </template>

                  <!-- List Operations -->
                  <template v-else-if="step.step_type === 'add_to_list' || step.step_type === 'remove_from_list'">
                    <b-field label="List">
                      <b-select v-model="step.list_id" expanded>
                        <option :value="null">Select a list...</option>
                        <option v-for="list in lists.results" :key="list.id" :value="list.id">
                          {{ list.name }}
                        </option>
                      </b-select>
                    </b-field>
                  </template>

                  <!-- Update Subscriber -->
                  <template v-else-if="step.step_type === 'update_subscriber'">
                    <b-field label="Attribute Key">
                      <b-input v-model="step.attrib_key" placeholder="e.g., flow_completed" />
                    </b-field>
                    <b-field label="Value">
                      <b-input v-model="step.attrib_value" placeholder="Value to set" />
                    </b-field>
                  </template>

                  <div class="step-details-actions">
                    <b-button type="is-primary" size="is-small" @click="editingStepIndex = null">
                      Done
                    </b-button>
                  </div>
                </div>

                <!-- Arrow to next step -->
                <div v-if="index < flow.steps.length - 1" class="step-connector">
                  <b-icon icon="arrow-down" />
                </div>
              </div>
            </div>

            <div v-else class="has-text-centered has-text-grey py-6">
              <b-icon icon="account_tree" size="is-large" />
              <p class="mt-3">No steps yet. Add your first step to build the flow.</p>
            </div>
          </div>

          <!-- Test Flow (only for existing flows) -->
          <div v-if="isEditing" class="box mt-4">
            <h3 class="title is-5">Test Flow</h3>
            <p class="help mb-3">Manually trigger this flow for a subscriber to test it.</p>
            <b-field label="Subscriber Email">
              <b-input v-model="testEmail" placeholder="test@example.com" />
            </b-field>
            <b-button
              type="is-info"
              :loading="loading.test"
              :disabled="!testEmail"
              @click="triggerTestFlow"
            >
              Trigger Test Flow
            </b-button>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';

export default Vue.extend({
  name: 'FlowBuilder',

  data() {
    return {
      flow: {
        id: 0,
        uuid: '',
        name: 'New Flow',
        description: '',
        status: 'draft',
        trigger_type: 'list_subscription',
        trigger_filters: {},
        profile_filters: {},
        steps: [],
        stats: null,
      },
      triggerFilters: {},
      editingStepIndex: null,
      testEmail: '',
      loading: {
        flow: false,
        save: false,
        test: false,
      },
      stepTypeColors: {
        email: 'is-info',
        delay: 'is-warning',
        condition: 'is-link',
        add_to_list: 'is-success',
        remove_from_list: 'is-danger',
        update_subscriber: 'is-primary',
      },
      stepTypeLabels: {
        email: 'Send Email',
        delay: 'Delay',
        condition: 'Condition',
        add_to_list: 'Add to List',
        remove_from_list: 'Remove from List',
        update_subscriber: 'Update Subscriber',
      },
    };
  },

  computed: {
    ...mapState(['lists', 'templates']),

    isEditing() {
      return this.$route.params.id !== undefined;
    },
  },

  async mounted() {
    // Load lists and templates
    this.$api.getLists();
    this.$api.getTemplates();

    if (this.isEditing) {
      await this.loadFlow();
    }
  },

  methods: {
    async loadFlow() {
      this.loading.flow = true;
      try {
        const resp = await this.$api.getFlow(this.$route.params.id);
        this.flow = resp;
        // Parse trigger filters
        if (resp.trigger_filters) {
          this.triggerFilters = { ...resp.trigger_filters };
        }
      } catch (err) {
        this.$utils.toast(err.message, 'is-danger');
      } finally {
        this.loading.flow = false;
      }
    },

    addStep(stepType) {
      const newStep = {
        step_order: this.flow.steps.length + 1,
        step_type: stepType,
        template_id: null,
        subject: '',
        body: '',
        from_email: '',
        delay_value: stepType === 'delay' ? 1 : null,
        delay_unit: stepType === 'delay' ? 'hours' : null,
        condition_type: stepType === 'condition' ? 'trigger_data' : null,
        condition_field: '',
        condition_operator: stepType === 'condition' ? 'equals' : null,
        condition_value: '',
        true_next_step: null,
        false_next_step: null,
        list_id: null,
        attrib_key: '',
        attrib_value: '',
      };
      this.flow.steps.push(newStep);
      this.editingStepIndex = this.flow.steps.length - 1;
    },

    editStep(index) {
      this.editingStepIndex = this.editingStepIndex === index ? null : index;
    },

    removeStep(index) {
      this.$buefy.dialog.confirm({
        message: 'Remove this step?',
        confirmText: 'Remove',
        type: 'is-danger',
        onConfirm: () => {
          this.flow.steps.splice(index, 1);
          // Renumber steps
          this.flow.steps = this.flow.steps.map((s, i) => ({
            ...s,
            step_order: i + 1,
          }));
          this.editingStepIndex = null;
        },
      });
    },

    getStepSummary(step) {
      switch (step.step_type) {
        case 'email':
          if (step.subject) return step.subject;
          if (step.template_id && this.templates) {
            const tpl = this.templates.find((t) => t.id === step.template_id);
            return tpl ? tpl.name : 'Send email';
          }
          return 'Send email';
        case 'delay':
          if (step.delay_value && step.delay_unit) {
            return `Wait ${step.delay_value} ${step.delay_unit}`;
          }
          return 'Delay';
        case 'condition':
          if (step.condition_field && step.condition_operator) {
            return `If ${step.condition_field} ${step.condition_operator} ${step.condition_value || '...'}`;
          }
          return 'Condition check';
        case 'add_to_list':
        case 'remove_from_list':
          if (step.list_id && this.lists.results) {
            const list = this.lists.results.find((l) => l.id === step.list_id);
            return list ? list.name : 'List operation';
          }
          return step.step_type === 'add_to_list' ? 'Add to list' : 'Remove from list';
        case 'update_subscriber':
          if (step.attrib_key) {
            return `Set ${step.attrib_key}`;
          }
          return 'Update attribute';
        default:
          return step.step_type;
      }
    },

    async handleSave() {
      if (!this.flow.name) {
        this.$utils.toast('Please enter a flow name', 'is-warning');
        return;
      }
      if (!this.flow.trigger_type) {
        this.$utils.toast('Please select a trigger type', 'is-warning');
        return;
      }

      this.loading.save = true;

      // Build the trigger_filters from the UI
      const triggerFilters = {};
      Object.keys(this.triggerFilters).forEach((key) => {
        if (this.triggerFilters[key] !== null && this.triggerFilters[key] !== '') {
          triggerFilters[key] = this.triggerFilters[key];
        }
      });

      const payload = {
        name: this.flow.name,
        description: this.flow.description,
        status: this.flow.status,
        trigger_type: this.flow.trigger_type,
        trigger_filters: triggerFilters,
        profile_filters: this.flow.profile_filters || {},
      };

      try {
        let savedFlow;
        if (this.isEditing) {
          savedFlow = await this.$api.updateFlow(this.flow.id, payload);
        } else {
          savedFlow = await this.$api.createFlow(payload);
        }

        // Save steps
        if (this.isEditing) {
          // For existing flows, we need to update/create/delete steps
          await this.syncSteps(savedFlow.id);
        } else if (this.flow.steps.length > 0) {
          // For new flows, create all steps sequentially
          await this.flow.steps.reduce(
            (promise, step) => promise.then(() => this.$api.createFlowStep(savedFlow.id, step)),
            Promise.resolve(),
          );
        }

        this.$utils.toast(
          this.isEditing ? 'Flow updated' : 'Flow created',
          'is-success',
        );

        if (!this.isEditing) {
          this.$router.push({ name: 'flow', params: { id: savedFlow.id } });
        } else {
          await this.loadFlow();
        }
      } catch (err) {
        this.$utils.toast(err.message, 'is-danger');
      } finally {
        this.loading.save = false;
      }
    },

    async syncSteps(flowId) {
      // Get existing steps from server
      const existing = await this.$api.getFlowSteps(flowId);
      const existingIds = new Set(existing.map((s) => s.id));
      const currentIds = new Set(this.flow.steps.filter((s) => s.id).map((s) => s.id));

      // Delete removed steps
      const stepsToDelete = existing.filter((s) => !currentIds.has(s.id));
      await Promise.all(stepsToDelete.map((s) => this.$api.deleteFlowStep(flowId, s.id)));

      // Update or create steps sequentially to maintain order
      const stepsWithOrder = this.flow.steps.map((s, i) => ({ ...s, step_order: i + 1 }));
      await stepsWithOrder.reduce((promise, step) => promise.then(() => {
        if (step.id && existingIds.has(step.id)) {
          return this.$api.updateFlowStep(flowId, step.id, step);
        }
        return this.$api.createFlowStep(flowId, step);
      }), Promise.resolve());
    },

    async triggerTestFlow() {
      if (!this.testEmail) return;

      this.loading.test = true;
      try {
        await this.$api.triggerFlow(this.flow.id, null, this.testEmail, {});
        this.$utils.toast('Test flow triggered', 'is-success');
      } catch (err) {
        this.$utils.toast(err.message, 'is-danger');
      } finally {
        this.loading.test = false;
      }
    },
  },
});
</script>

<style lang="scss" scoped>
.flow-builder {
  .builder-container {
    margin-top: 1rem;
  }

  .stats-box {
    padding: 1rem;
    background: #f5f5f5;
    border-radius: 4px;

    .heading {
      font-size: 0.75rem;
      text-transform: uppercase;
      color: #7a7a7a;
      margin-bottom: 0.25rem;
    }

    .title.is-5 {
      margin-bottom: 0;
    }
  }

  .steps-list {
    margin-top: 1rem;
  }

  .step-card {
    border: 1px solid #dbdbdb;
    border-radius: 4px;
    margin-bottom: 0.5rem;
    background: #fff;

    .step-header {
      display: flex;
      align-items: center;
      padding: 0.75rem 1rem;
      border-bottom: 1px solid #f5f5f5;

      .step-number {
        background: #363636;
        color: #fff;
        width: 24px;
        height: 24px;
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 0.75rem;
        margin-right: 0.75rem;
      }

      .step-title {
        flex: 1;
        font-weight: 500;
      }

      .step-actions {
        display: flex;
        gap: 0.5rem;

        a {
          color: #7a7a7a;

          &:hover {
            color: #363636;
          }
        }
      }
    }

    .step-details {
      padding: 1rem;
      background: #fafafa;

      .step-details-actions {
        margin-top: 1rem;
        text-align: right;
      }
    }

    .step-connector {
      text-align: center;
      padding: 0.25rem;
      color: #7a7a7a;
    }
  }
}
</style>
