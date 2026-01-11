import { create } from 'zustand';
import { v4 as uuidv4 } from 'uuid';
import type { Block, BlockType, FormConfig, FormStep, FormSettings, FormTargeting, FormTriggers, FormFrequency, CouponConfig } from '@/types';
import { getDefaultBlockProps } from '@/blocks/defaults';

type PreviewDevice = 'desktop' | 'tablet' | 'mobile';

interface FormBuilderState {
  // Form configuration
  form: FormConfig;

  // UI state
  selectedBlockId: string | null;
  selectedStepIndex: number;
  isDirty: boolean;
  previewDevice: PreviewDevice;

  // Actions
  setForm: (form: FormConfig) => void;
  updateFormField: <K extends keyof FormConfig>(field: K, value: FormConfig[K]) => void;
  setPreviewDevice: (device: PreviewDevice) => void;

  // Step actions
  addStep: () => void;
  removeStep: (index: number) => void;
  updateStepName: (index: number, name: string) => void;
  selectStep: (index: number) => void;

  // Block actions
  addBlock: (type: BlockType, stepIndex?: number) => void;
  removeBlock: (blockId: string) => void;
  updateBlock: (blockId: string, props: Partial<Block['props']>) => void;
  moveBlock: (blockId: string, newIndex: number) => void;
  selectBlock: (blockId: string | null) => void;
  duplicateBlock: (blockId: string) => void;

  // Settings actions
  updateSettings: (settings: Partial<FormSettings>) => void;
  updateTargeting: (targeting: Partial<FormTargeting>) => void;
  updateTriggers: (triggers: Partial<FormTriggers>) => void;
  updateFrequency: (frequency: Partial<FormFrequency>) => void;
  updateCouponConfig: (config: Partial<CouponConfig>) => void;

  // Reset
  reset: () => void;
}

const defaultSettings: FormSettings = {
  position: 'center',
  animation: 'fade',
  overlay: true,
  overlayColor: '#000000',
  overlayOpacity: 0.5,
  closeButton: true,
  closeOnOverlayClick: true,
  successAction: 'message',
  successMessage: 'Thank you for subscribing!',
  successRedirectUrl: '',
  width: '400px',
  maxWidth: '90vw',
  backgroundColor: '#ffffff',
  borderRadius: 8,
  padding: { top: 24, right: 24, bottom: 24, left: 24 },
};

const defaultTargeting: FormTargeting = {
  urlPatterns: [],
  utmParams: [],
  devices: ['desktop', 'tablet', 'mobile'],
  countries: [],
  excludeSubscribers: false,
  shopifyPages: [],
};

const defaultTriggers: FormTriggers = {
  type: 'time-delay',
  timeDelay: 5,
};

const defaultFrequency: FormFrequency = {
  showAgainAfterDays: 7,
  suppressAfterSubmit: true,
  maxShowsPerSession: 1,
};

const defaultCouponConfig: CouponConfig = {
  enabled: false,
  type: 'static',
};

const createDefaultForm = (): FormConfig => ({
  name: 'New Form',
  description: '',
  formType: 'popup',
  status: 'draft',
  steps: [
    {
      id: uuidv4(),
      name: 'Step 1',
      blocks: [],
    },
  ],
  settings: { ...defaultSettings },
  targeting: { ...defaultTargeting },
  triggers: { ...defaultTriggers },
  frequency: { ...defaultFrequency },
  listIds: [],
  couponConfig: { ...defaultCouponConfig },
  customCss: '',
});

export const useFormBuilderStore = create<FormBuilderState>((set, get) => ({
  form: createDefaultForm(),
  selectedBlockId: null,
  selectedStepIndex: 0,
  isDirty: false,
  previewDevice: 'desktop',

  setPreviewDevice: (device) => set({ previewDevice: device }),

  setForm: (form) => {
    // If form is empty/invalid, use default form
    if (!form || !form.steps || !Array.isArray(form.steps) || form.steps.length === 0) {
      set({ form: createDefaultForm(), isDirty: false });
      return;
    }
    // Merge with defaults to ensure all required fields exist
    const mergedForm: FormConfig = {
      name: form.name || 'New Form',
      description: form.description || '',
      formType: form.formType || 'popup',
      status: form.status || 'draft',
      steps: form.steps,
      settings: { ...defaultSettings, ...(form.settings || {}) },
      targeting: { ...defaultTargeting, ...(form.targeting || {}) },
      triggers: { ...defaultTriggers, ...(form.triggers || {}) },
      frequency: { ...defaultFrequency, ...(form.frequency || {}) },
      listIds: form.listIds || [],
      couponConfig: { ...defaultCouponConfig, ...(form.couponConfig || {}) },
      customCss: form.customCss || '',
    };
    set({ form: mergedForm, isDirty: false });
  },

  updateFormField: (field, value) =>
    set((state) => ({
      form: { ...state.form, [field]: value },
      isDirty: true,
    })),

  addStep: () =>
    set((state) => ({
      form: {
        ...state.form,
        steps: [
          ...state.form.steps,
          {
            id: uuidv4(),
            name: `Step ${state.form.steps.length + 1}`,
            blocks: [],
          },
        ],
      },
      isDirty: true,
    })),

  removeStep: (index) =>
    set((state) => {
      if (state.form.steps.length <= 1) return state;
      const newSteps = state.form.steps.filter((_, i) => i !== index);
      return {
        form: { ...state.form, steps: newSteps },
        selectedStepIndex: Math.min(state.selectedStepIndex, newSteps.length - 1),
        isDirty: true,
      };
    }),

  updateStepName: (index, name) =>
    set((state) => ({
      form: {
        ...state.form,
        steps: state.form.steps.map((step, i) =>
          i === index ? { ...step, name } : step
        ),
      },
      isDirty: true,
    })),

  selectStep: (index) => set({ selectedStepIndex: index, selectedBlockId: null }),

  addBlock: (type, stepIndex) =>
    set((state) => {
      const targetIndex = stepIndex ?? state.selectedStepIndex;
      const newBlock: Block = {
        id: uuidv4(),
        type,
        props: getDefaultBlockProps(type),
      } as Block;

      return {
        form: {
          ...state.form,
          steps: state.form.steps.map((step, i) =>
            i === targetIndex
              ? { ...step, blocks: [...step.blocks, newBlock] }
              : step
          ),
        },
        selectedBlockId: newBlock.id,
        isDirty: true,
      };
    }),

  removeBlock: (blockId) =>
    set((state) => ({
      form: {
        ...state.form,
        steps: state.form.steps.map((step) => ({
          ...step,
          blocks: step.blocks.filter((block) => block.id !== blockId),
        })),
      },
      selectedBlockId: state.selectedBlockId === blockId ? null : state.selectedBlockId,
      isDirty: true,
    })),

  updateBlock: (blockId, props) =>
    set((state) => ({
      form: {
        ...state.form,
        steps: state.form.steps.map((step) => ({
          ...step,
          blocks: step.blocks.map((block) =>
            block.id === blockId
              ? { ...block, props: { ...block.props, ...props } }
              : block
          ),
        })),
      },
      isDirty: true,
    })),

  moveBlock: (blockId, newIndex) =>
    set((state) => {
      const stepIndex = state.selectedStepIndex;
      const step = state.form.steps[stepIndex];
      const blockIndex = step.blocks.findIndex((b) => b.id === blockId);

      if (blockIndex === -1 || blockIndex === newIndex) return state;

      const newBlocks = [...step.blocks];
      const [removed] = newBlocks.splice(blockIndex, 1);
      newBlocks.splice(newIndex, 0, removed);

      return {
        form: {
          ...state.form,
          steps: state.form.steps.map((s, i) =>
            i === stepIndex ? { ...s, blocks: newBlocks } : s
          ),
        },
        isDirty: true,
      };
    }),

  selectBlock: (blockId) => set({ selectedBlockId: blockId }),

  duplicateBlock: (blockId) =>
    set((state) => {
      const stepIndex = state.selectedStepIndex;
      const step = state.form.steps[stepIndex];
      const blockIndex = step.blocks.findIndex((b) => b.id === blockId);

      if (blockIndex === -1) return state;

      const originalBlock = step.blocks[blockIndex];
      const newBlock = {
        ...originalBlock,
        id: uuidv4(),
        props: { ...originalBlock.props },
      };

      const newBlocks = [...step.blocks];
      newBlocks.splice(blockIndex + 1, 0, newBlock);

      return {
        form: {
          ...state.form,
          steps: state.form.steps.map((s, i) =>
            i === stepIndex ? { ...s, blocks: newBlocks } : s
          ),
        },
        selectedBlockId: newBlock.id,
        isDirty: true,
      };
    }),

  updateSettings: (settings) =>
    set((state) => ({
      form: {
        ...state.form,
        settings: { ...state.form.settings, ...settings },
      },
      isDirty: true,
    })),

  updateTargeting: (targeting) =>
    set((state) => ({
      form: {
        ...state.form,
        targeting: { ...state.form.targeting, ...targeting },
      },
      isDirty: true,
    })),

  updateTriggers: (triggers) =>
    set((state) => ({
      form: {
        ...state.form,
        triggers: { ...state.form.triggers, ...triggers },
      },
      isDirty: true,
    })),

  updateFrequency: (frequency) =>
    set((state) => ({
      form: {
        ...state.form,
        frequency: { ...state.form.frequency, ...frequency },
      },
      isDirty: true,
    })),

  updateCouponConfig: (config) =>
    set((state) => ({
      form: {
        ...state.form,
        couponConfig: { ...state.form.couponConfig, ...config },
      },
      isDirty: true,
    })),

  reset: () => set({ form: createDefaultForm(), selectedBlockId: null, selectedStepIndex: 0, isDirty: false, previewDevice: 'desktop' }),
}));
