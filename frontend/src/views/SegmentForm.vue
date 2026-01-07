<template>
  <div class="modal-card segment-form">
    <header class="modal-card-head">
      <p class="modal-card-title">
        {{ isEditing ? data.name : $t('segments.newSegment') }}
      </p>
    </header>

    <section class="modal-card-body">
      <b-field :label="$t('segments.name')" label-position="on-border">
        <b-input v-model="form.name" :maxlength="200" required ref="name" />
      </b-field>

      <b-field :label="$t('segments.description')" label-position="on-border">
        <b-input v-model="form.description" type="textarea" :maxlength="500" />
      </b-field>

      <b-field :label="$t('globals.terms.tags')" label-position="on-border">
        <b-taginput v-model="form.tags" ellipsis icon="tag-outline" :placeholder="$t('globals.buttons.addTag')" />
      </b-field>

      <hr />

      <div class="segment-builder">
        <h4 class="title is-6">{{ $t('segments.conditions') }}</h4>

        <div class="match-type">
          <b-field>
            <b-radio v-model="form.conditions.operator" native-value="and">
              {{ $t('segments.matchAll') }}
            </b-radio>
            <b-radio v-model="form.conditions.operator" native-value="or">
              {{ $t('segments.matchAny') }}
            </b-radio>
          </b-field>
        </div>

        <div class="conditions-list">
          <div v-for="(condition, index) in form.conditions.conditions" :key="index" class="condition-row box">
            <div class="columns is-multiline">
              <!-- Field selector -->
              <div class="column is-3">
                <b-field :label="$t('segments.field')" label-position="on-border">
                  <b-select v-model="condition.field" @input="onFieldChange(index)" expanded>
                    <optgroup label="Subscriber">
                      <option value="email">Email</option>
                      <option value="name">Name</option>
                      <option value="status">Status</option>
                      <option value="created_at">Created At</option>
                      <option value="updated_at">Updated At</option>
                    </optgroup>
                    <optgroup label="Behavior">
                      <option value="behavior.email_opens">Email Opens</option>
                      <option value="behavior.link_clicks">Link Clicks</option>
                      <option value="behavior.opened_campaign">Opened Campaign</option>
                      <option value="behavior.clicked_campaign">Clicked Campaign Link</option>
                    </optgroup>
                    <optgroup label="Shopify">
                      <option value="attribs.shopify.orders_count">Total Orders</option>
                      <option value="attribs.shopify.total_spent">Total Spent ($)</option>
                      <option value="attribs.shopify.purchased_effects">Purchase History</option>
                      <option value="attribs.shopify.first_name">First Name</option>
                      <option value="attribs.shopify.last_name">Last Name</option>
                      <option value="attribs.shopify.tags">Customer Tags</option>
                      <option value="attribs.shopify.currency">Currency</option>
                      <option value="attribs.shopify.synced_at">Last Synced</option>
                    </optgroup>
                    <optgroup label="Order History">
                      <option value="orders.product_title">Purchased Product</option>
                      <option value="orders.product_type">Product Type</option>
                      <option value="orders.total_value">Order Total Value ($)</option>
                      <option value="orders.order_count">Order Count (DB)</option>
                    </optgroup>
                    <optgroup label="Custom Attributes">
                      <option v-for="attr in knownAttribs" :key="attr" :value="`attribs.${attr}`">
                        {{ attr }}
                      </option>
                      <option value="attribs.custom">Custom attribute...</option>
                    </optgroup>
                  </b-select>
                </b-field>
              </div>

              <!-- Custom attribute name -->
              <div v-if="condition.field === 'attribs.custom'" class="column is-2">
                <b-field label="Attribute name" label-position="on-border">
                  <b-input v-model="condition.customField" @input="updateCustomField(index)" placeholder="plan" />
                </b-field>
              </div>

              <!-- Field type selector (hidden for behavioral fields) -->
              <div class="column is-2" v-if="!isBehavioralField(condition.field)">
                <b-field :label="$t('segments.fieldTypes.text')" label-position="on-border">
                  <b-select v-model="condition.field_type" @input="onFieldTypeChange(index)" expanded>
                    <option value="text">{{ $t('segments.fieldTypes.text') }}</option>
                    <option value="number">{{ $t('segments.fieldTypes.number') }}</option>
                    <option value="date">{{ $t('segments.fieldTypes.date') }}</option>
                    <option value="boolean">{{ $t('segments.fieldTypes.boolean') }}</option>
                    <option value="array">{{ $t('segments.fieldTypes.array') }}</option>
                  </b-select>
                </b-field>
              </div>

              <!-- Operator selector -->
              <div class="column is-3">
                <b-field :label="$t('segments.operator')" label-position="on-border">
                  <b-select v-model="condition.operator" expanded>
                    <option v-for="op in getOperatorsForType(condition.field_type)" :key="op.value" :value="op.value">
                      {{ $t(`segments.operators.${op.value}`) }}
                    </option>
                  </b-select>
                </b-field>
              </div>

              <!-- Value input -->
              <div class="column is-3" v-if="showValueInput(condition)">
                <b-field :label="getValueLabel(condition)" label-position="on-border">
                  <!-- Text input -->
                  <b-input v-if="condition.field_type === 'text'" v-model="condition.value" />

                  <!-- Number input (for behavior counts too) -->
                  <b-input v-else-if="condition.field_type === 'number' || behavioralCountOperators.includes(condition.operator)"
                    v-model.number="condition.value" type="number" min="0" />

                  <!-- Date input -->
                  <b-datepicker v-else-if="condition.field_type === 'date' && !daysOperators.includes(condition.operator) && condition.operator !== 'between_dates'"
                    v-model="condition.value" icon="calendar" editable />

                  <!-- Days input for in_last_x_days, more_than_x_days_ago, in_next_x_days -->
                  <b-input v-else-if="daysOperators.includes(condition.operator)" v-model.number="condition.value"
                    type="number" min="1" placeholder="Days" />

                  <!-- Day of week selector -->
                  <b-select v-else-if="condition.operator === 'day_of_week_is'" v-model.number="condition.value" expanded>
                    <option :value="0">Sunday</option>
                    <option :value="1">Monday</option>
                    <option :value="2">Tuesday</option>
                    <option :value="3">Wednesday</option>
                    <option :value="4">Thursday</option>
                    <option :value="5">Friday</option>
                    <option :value="6">Saturday</option>
                  </b-select>

                  <!-- Array input -->
                  <b-taginput v-else-if="condition.field_type === 'array' && arrayValueOperators.includes(condition.operator)"
                    v-model="condition.value" ellipsis />

                  <!-- Campaign selector for campaign-specific behaviors -->
                  <b-select v-else-if="needsCampaignSelector(condition.field)"
                    v-model.number="condition.value" expanded :loading="campaignsLoading">
                    <option v-for="camp in campaigns" :key="camp.id" :value="camp.id">
                      {{ camp.name }}
                    </option>
                  </b-select>

                  <!-- Fallback -->
                  <b-input v-else v-model="condition.value" />
                </b-field>
              </div>

              <!-- Date range inputs for between_dates -->
              <div class="column is-3" v-if="condition.operator === 'between_dates'">
                <b-field label="Start Date" label-position="on-border">
                  <b-datepicker v-model="condition.dateStart" icon="calendar" editable
                    @input="updateBetweenDates(index)" />
                </b-field>
              </div>
              <div class="column is-3" v-if="condition.operator === 'between_dates'">
                <b-field label="End Date" label-position="on-border">
                  <b-datepicker v-model="condition.dateEnd" icon="calendar" editable
                    @input="updateBetweenDates(index)" />
                </b-field>
              </div>

              <!-- Time range selector for behavioral conditions -->
              <div class="column is-2" v-if="showTimeRangeSelector(condition)">
                <b-field label="Time Range" label-position="on-border">
                  <b-select v-model.number="condition.params.time_range_days" expanded>
                    <option :value="7">Last 7 days</option>
                    <option :value="30">Last 30 days</option>
                    <option :value="90">Last 90 days</option>
                    <option :value="180">Last 180 days</option>
                    <option :value="365">Last 365 days</option>
                    <option :value="0">All time</option>
                  </b-select>
                </b-field>
              </div>

              <!-- Remove button -->
              <div class="column is-1">
                <b-button type="is-danger" icon-left="close" size="is-small" @click="removeCondition(index)"
                  :aria-label="$t('segments.removeCondition')" />
              </div>
            </div>
          </div>
        </div>

        <div class="add-condition">
          <b-button type="is-light" icon-left="plus" @click="addCondition">
            {{ $t('segments.addCondition') }}
          </b-button>
        </div>

        <!-- Preview count -->
        <div class="preview-count box" v-if="form.conditions.conditions.length > 0">
          <div class="level">
            <div class="level-left">
              <div class="level-item">
                <span v-if="previewCount !== null">
                  <strong>{{ $utils.formatNumber(previewCount) }}</strong>
                  {{ $tc('segments.count', previewCount) }}
                </span>
                <span v-else class="has-text-grey">
                  {{ $t('segments.refreshCount') }}
                </span>
              </div>
            </div>
            <div class="level-right">
              <div class="level-item">
                <b-button type="is-light" icon-left="refresh" size="is-small" @click="refreshPreviewCount"
                  :loading="countLoading">
                  {{ $t('segments.refreshCount') }}
                </b-button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <footer class="modal-card-foot">
      <b-button @click="$parent.close()">{{ $t('globals.buttons.close') }}</b-button>
      <b-button type="is-primary" @click="save" :loading="loading.segments">
        {{ $t('globals.buttons.save') }}
      </b-button>
    </footer>
  </div>
</template>

<script>
import Vue from 'vue';
import { mapState } from 'vuex';

export default Vue.extend({
  name: 'SegmentForm',

  props: {
    data: {
      type: Object,
      default: () => ({}),
    },
    isEditing: {
      type: Boolean,
      default: false,
    },
  },

  data() {
    return {
      form: {
        name: '',
        description: '',
        tags: [],
        conditions: {
          operator: 'and',
          conditions: [],
        },
      },
      previewCount: null,
      countLoading: false,
      campaigns: [],
      campaignsLoading: false,
      knownAttribs: ['plan', 'country', 'city', 'company', 'source', 'total_orders', 'total_spent'],
      noValueOperators: ['is_set', 'is_not_set', 'is_true', 'is_false', 'is_empty', 'zero', 'greater_than_zero', 'anniversary_is_today'],
      daysOperators: ['in_last_x_days', 'more_than_x_days_ago', 'in_next_x_days'],
      arrayValueOperators: ['includes_any', 'includes_all'],
      behavioralCountOperators: ['at_least', 'at_most', 'equals'],
      behavioralFields: ['behavior.email_opens', 'behavior.link_clicks'],
      campaignBehavioralFields: ['behavior.opened_campaign', 'behavior.clicked_campaign'],
      shopifyFieldTypes: {
        'attribs.shopify.orders_count': 'number',
        'attribs.shopify.total_spent': 'number',
        'attribs.shopify.purchased_effects': 'array',
        'attribs.shopify.first_name': 'text',
        'attribs.shopify.last_name': 'text',
        'attribs.shopify.tags': 'array',
        'attribs.shopify.currency': 'text',
        'attribs.shopify.synced_at': 'date',
      },
      orderFieldTypes: {
        'orders.product_title': 'text',
        'orders.product_type': 'text',
        'orders.total_value': 'number',
        'orders.order_count': 'number',
      },
    };
  },

  computed: {
    ...mapState(['loading']),
  },

  methods: {
    getOperatorsForType(fieldType) {
      const operators = {
        text: [
          { value: 'equals' },
          { value: 'not_equals' },
          { value: 'contains' },
          { value: 'not_contains' },
          { value: 'starts_with' },
          { value: 'ends_with' },
          { value: 'is_set' },
          { value: 'is_not_set' },
        ],
        number: [
          { value: 'equals' },
          { value: 'not_equals' },
          { value: 'greater_than' },
          { value: 'less_than' },
          { value: 'at_least' },
          { value: 'at_most' },
        ],
        date: [
          { value: 'before' },
          { value: 'after' },
          { value: 'on' },
          { value: 'in_last_x_days' },
          { value: 'more_than_x_days_ago' },
          { value: 'in_next_x_days' },
          { value: 'between_dates' },
          { value: 'anniversary_is_today' },
          { value: 'day_of_week_is' },
          { value: 'is_set' },
          { value: 'is_not_set' },
        ],
        boolean: [
          { value: 'is_true' },
          { value: 'is_false' },
        ],
        array: [
          { value: 'includes_any' },
          { value: 'includes_all' },
          { value: 'is_empty' },
          { value: 'has_at_least' },
        ],
        behavior: [
          { value: 'at_least' },
          { value: 'at_most' },
          { value: 'equals' },
          { value: 'zero' },
          { value: 'greater_than_zero' },
        ],
        campaign_behavior: [
          { value: 'is_true' },
          { value: 'is_false' },
        ],
      };
      return operators[fieldType] || operators.text;
    },

    isBehavioralField(field) {
      return field && field.startsWith('behavior.');
    },

    needsCampaignSelector(field) {
      return this.campaignBehavioralFields.includes(field);
    },

    showTimeRangeSelector(condition) {
      return this.behavioralFields.includes(condition.field);
    },

    showValueInput(condition) {
      // No value for certain operators
      if (this.noValueOperators.includes(condition.operator)) {
        return false;
      }
      // between_dates has separate date pickers
      if (condition.operator === 'between_dates') {
        return false;
      }
      return true;
    },

    getValueLabel(condition) {
      if (this.behavioralCountOperators.includes(condition.operator)) {
        return 'Count';
      }
      if (this.needsCampaignSelector(condition.field)) {
        return 'Campaign';
      }
      if (condition.operator === 'day_of_week_is') {
        return 'Day';
      }
      return this.$t('segments.value');
    },

    updateBetweenDates(index) {
      const condition = this.form.conditions.conditions[index];
      if (condition.dateStart && condition.dateEnd) {
        condition.value = {
          start: condition.dateStart.toISOString(),
          end: condition.dateEnd.toISOString(),
        };
      }
    },

    loadCampaigns() {
      this.campaignsLoading = true;
      this.$api.getCampaignsMinimal()
        .then((resp) => {
          this.campaigns = resp;
        })
        .finally(() => {
          this.campaignsLoading = false;
        });
    },

    onFieldChange(index) {
      const condition = this.form.conditions.conditions[index];
      // Set default field type based on field
      if (['created_at', 'updated_at'].includes(condition.field)) {
        condition.field_type = 'date';
      } else if (condition.field === 'status') {
        condition.field_type = 'text';
      } else if (this.behavioralFields.includes(condition.field)) {
        condition.field_type = 'behavior';
        // Initialize params for behavioral conditions
        if (!condition.params) {
          condition.params = { time_range_days: 30 };
        }
      } else if (this.campaignBehavioralFields.includes(condition.field)) {
        condition.field_type = 'campaign_behavior';
        // Load campaigns if not already loaded
        if (this.campaigns.length === 0) {
          this.loadCampaigns();
        }
      } else if (this.shopifyFieldTypes[condition.field]) {
        // Shopify fields have predefined types
        condition.field_type = this.shopifyFieldTypes[condition.field];
      } else if (this.orderFieldTypes[condition.field]) {
        // Order history fields have predefined types
        condition.field_type = this.orderFieldTypes[condition.field];
      } else {
        condition.field_type = 'text';
      }
      this.onFieldTypeChange(index);
    },

    onFieldTypeChange(index) {
      const condition = this.form.conditions.conditions[index];
      const operators = this.getOperatorsForType(condition.field_type);
      if (!operators.find((op) => op.value === condition.operator)) {
        condition.operator = operators[0].value;
      }
      // Reset value
      if (condition.field_type === 'array' && this.arrayValueOperators.includes(condition.operator)) {
        condition.value = [];
      } else if (condition.field_type === 'date' && !this.daysOperators.includes(condition.operator)) {
        condition.value = null;
      } else {
        condition.value = '';
      }
    },

    updateCustomField(index) {
      const condition = this.form.conditions.conditions[index];
      if (condition.customField) {
        condition.field = `attribs.${condition.customField}`;
      }
    },

    addCondition() {
      this.form.conditions.conditions.push({
        field: 'email',
        field_type: 'text',
        operator: 'contains',
        value: '',
        params: {},
      });
    },

    removeCondition(index) {
      this.form.conditions.conditions.splice(index, 1);
    },

    refreshPreviewCount() {
      this.countLoading = true;
      this.$api.previewSegmentConditions(this.form.conditions, { per_page: 1 })
        .then((resp) => {
          this.previewCount = resp.total;
        })
        .finally(() => {
          this.countLoading = false;
        });
    },

    save() {
      if (!this.form.name) {
        this.$utils.toast(this.$t('segments.invalidName'), 'is-danger');
        return;
      }

      // Clean up conditions for saving
      const conditions = {
        operator: this.form.conditions.operator,
        conditions: this.form.conditions.conditions.map((c) => {
          const cond = {
            field: c.field,
            field_type: c.field_type,
            operator: c.operator,
          };
          // Only include value if operator requires it
          if (!this.noValueOperators.includes(c.operator)) {
            cond.value = c.value;
          }
          // Include params if present and not empty
          if (c.params && Object.keys(c.params).length > 0) {
            cond.params = c.params;
          }
          return cond;
        }),
      };

      const data = {
        ...this.form,
        conditions,
      };

      if (this.isEditing) {
        this.$api.updateSegment({ id: this.data.id, ...data }).then(() => {
          this.$utils.toast(this.$t('globals.messages.updated', { name: data.name }));
          this.$emit('finished');
        });
      } else {
        this.$api.createSegment(data).then(() => {
          this.$utils.toast(this.$t('globals.messages.created', { name: data.name }));
          this.$emit('finished');
        });
      }
    },
  },

  mounted() {
    if (this.isEditing && this.data) {
      this.form = {
        name: this.data.name || '',
        description: this.data.description || '',
        tags: this.data.tags || [],
        conditions: this.data.conditions || { operator: 'and', conditions: [] },
      };

      // Initialize params for existing conditions if not present
      let needsCampaigns = false;
      const { conditions } = this.form.conditions;
      for (let i = 0; i < conditions.length; i += 1) {
        if (!conditions[i].params) {
          conditions[i].params = {};
        }
        // Check if any condition needs campaign selector
        if (this.campaignBehavioralFields.includes(conditions[i].field)) {
          needsCampaigns = true;
        }
        // Initialize time range for behavioral fields
        if (this.behavioralFields.includes(conditions[i].field) && !conditions[i].params.time_range_days) {
          conditions[i].params.time_range_days = 30;
        }
      }

      // Load campaigns if needed
      if (needsCampaigns) {
        this.loadCampaigns();
      }

      if (this.data.cachedCount !== null) {
        this.previewCount = this.data.cachedCount;
      }
    }
    this.$nextTick(() => {
      this.$refs.name?.focus();
    });
  },
});
</script>

<style lang="scss" scoped>
.segment-form {
  width: 800px;
  max-width: calc(100vw - 40px);

  .segment-builder {
    .match-type {
      margin-bottom: 1rem;
    }

    .conditions-list {
      .condition-row {
        margin-bottom: 0.5rem;
        padding: 0.75rem;
        background: #f9f9f9;

        .columns {
          align-items: flex-end;
        }
      }
    }

    .add-condition {
      margin-top: 1rem;
    }

    .preview-count {
      margin-top: 1.5rem;
      background: #f0f7ff;
    }
  }
}
</style>
