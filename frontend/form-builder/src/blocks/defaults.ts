import type { BlockType, BlockDefinition } from '@/types';

const defaultPadding = { top: 8, right: 8, bottom: 8, left: 8 };
const defaultMargin = { top: 0, right: 0, bottom: 0, left: 0 };

export const blockDefinitions: BlockDefinition[] = [
  // Content blocks
  {
    type: 'heading',
    name: 'Heading',
    icon: 'title',
    category: 'content',
    defaultProps: {
      content: 'Sign up for our newsletter',
      level: 2,
      fontSize: 24,
      fontWeight: 'bold',
      color: '#1a1a1a',
      textAlign: 'center',
      padding: defaultPadding,
    },
  },
  {
    type: 'text',
    name: 'Text',
    icon: 'text_fields',
    category: 'content',
    defaultProps: {
      content: 'Get the latest updates and exclusive offers.',
      fontSize: 14,
      fontWeight: 'normal',
      color: '#666666',
      textAlign: 'center',
      padding: defaultPadding,
    },
  },
  {
    type: 'image',
    name: 'Image',
    icon: 'image',
    category: 'content',
    defaultProps: {
      src: '',
      alt: 'Image',
      width: 200,
      height: 'auto',
      alignment: 'center',
      padding: defaultPadding,
    },
  },

  // Input blocks
  {
    type: 'email-input',
    name: 'Email Input',
    icon: 'email',
    category: 'input',
    defaultProps: {
      name: 'email',
      label: 'Email address',
      placeholder: 'Enter your email',
      required: true,
      showLabel: true,
      fontSize: 14,
      labelColor: '#333333',
      inputBgColor: '#ffffff',
      inputBorderColor: '#cccccc',
      inputTextColor: '#333333',
      borderRadius: 4,
      padding: defaultPadding,
      width: '100%',
    },
  },
  {
    type: 'text-input',
    name: 'Text Input',
    icon: 'text_format',
    category: 'input',
    defaultProps: {
      name: 'name',
      label: 'Name',
      placeholder: 'Enter your name',
      required: false,
      subscriberField: 'name',
      customFieldName: '',
      fontSize: 14,
      labelColor: '#333333',
      inputBgColor: '#ffffff',
      inputBorderColor: '#cccccc',
      inputTextColor: '#333333',
      borderRadius: 4,
      padding: defaultPadding,
      width: '100%',
    },
  },
  {
    type: 'textarea',
    name: 'Textarea',
    icon: 'notes',
    category: 'input',
    defaultProps: {
      name: 'message',
      label: 'Message',
      placeholder: 'Enter your message',
      required: false,
      rows: 4,
      fontSize: 14,
      labelColor: '#333333',
      inputBgColor: '#ffffff',
      inputBorderColor: '#cccccc',
      inputTextColor: '#333333',
      borderRadius: 4,
      padding: defaultPadding,
    },
  },
  {
    type: 'radio-group',
    name: 'Radio Group',
    icon: 'radio_button_checked',
    category: 'input',
    defaultProps: {
      name: 'preference',
      label: 'Select an option',
      options: [
        { value: 'option1', label: 'Option 1' },
        { value: 'option2', label: 'Option 2' },
      ],
      required: false,
      layout: 'vertical',
      fontSize: 14,
      labelColor: '#333333',
      padding: defaultPadding,
    },
  },
  {
    type: 'checkbox-group',
    name: 'Checkbox Group',
    icon: 'check_box',
    category: 'input',
    defaultProps: {
      name: 'interests',
      label: 'Select all that apply',
      options: [
        { value: 'option1', label: 'Option 1' },
        { value: 'option2', label: 'Option 2' },
      ],
      required: false,
      layout: 'vertical',
      fontSize: 14,
      labelColor: '#333333',
      padding: defaultPadding,
    },
  },
  {
    type: 'dropdown',
    name: 'Dropdown',
    icon: 'arrow_drop_down',
    category: 'input',
    defaultProps: {
      name: 'country',
      label: 'Country',
      placeholder: 'Select a country',
      options: [
        { value: 'us', label: 'United States' },
        { value: 'uk', label: 'United Kingdom' },
        { value: 'ca', label: 'Canada' },
      ],
      required: false,
      fontSize: 14,
      labelColor: '#333333',
      inputBgColor: '#ffffff',
      inputBorderColor: '#cccccc',
      inputTextColor: '#333333',
      borderRadius: 4,
      padding: defaultPadding,
      width: '100%',
    },
  },
  {
    type: 'date-picker',
    name: 'Date Picker',
    icon: 'calendar_today',
    category: 'input',
    defaultProps: {
      name: 'birthdate',
      label: 'Date of birth',
      required: false,
      fontSize: 14,
      labelColor: '#333333',
      inputBgColor: '#ffffff',
      inputBorderColor: '#cccccc',
      inputTextColor: '#333333',
      borderRadius: 4,
      padding: defaultPadding,
      width: '100%',
    },
  },
  {
    type: 'hidden-field',
    name: 'Hidden Field',
    icon: 'visibility_off',
    category: 'input',
    defaultProps: {
      name: 'source',
      value: '',
      source: 'static',
    },
  },

  // Action blocks
  {
    type: 'button',
    name: 'Button',
    icon: 'smart_button',
    category: 'action',
    defaultProps: {
      text: 'Subscribe',
      action: 'submit',
      fontSize: 16,
      fontWeight: 'bold',
      textColor: '#ffffff',
      bgColor: '#2563eb',
      hoverBgColor: '#1d4ed8',
      borderRadius: 6,
      fullWidth: true,
      width: '100%',
      padding: { top: 12, right: 24, bottom: 12, left: 24 },
      margin: { top: 0, right: 0, bottom: 0, left: 0 },
    },
  },

  // Layout blocks
  {
    type: 'divider',
    name: 'Divider',
    icon: 'horizontal_rule',
    category: 'layout',
    defaultProps: {
      color: '#e5e7eb',
      thickness: 1,
      style: 'solid',
      margin: { top: 16, bottom: 16 },
    },
  },
  {
    type: 'spacer',
    name: 'Spacer',
    icon: 'expand',
    category: 'layout',
    defaultProps: {
      height: 24,
    },
  },

  // Special blocks
  {
    type: 'coupon',
    name: 'Coupon',
    icon: 'local_offer',
    category: 'special',
    defaultProps: {
      title: 'Your discount code:',
      fontSize: 14,
      fontWeight: 'bold',
      textColor: '#1a1a1a',
      bgColor: '#f3f4f6',
      borderColor: '#d1d5db',
      borderRadius: 4,
      padding: { top: 12, right: 16, bottom: 12, left: 16 },
    },
  },
  {
    type: 'countdown',
    name: 'Countdown',
    icon: 'timer',
    category: 'special',
    defaultProps: {
      endDate: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
      showDays: true,
      showHours: true,
      showMinutes: true,
      showSeconds: true,
      fontSize: 24,
      textColor: '#1a1a1a',
      bgColor: '#f3f4f6',
      borderRadius: 8,
      padding: { top: 16, right: 16, bottom: 16, left: 16 },
    },
  },
  {
    type: 'signup-counter',
    name: 'Signup Counter',
    icon: 'group',
    category: 'special',
    defaultProps: {
      text: 'Join {count}+ subscribers',
      count: 1000,
      fontSize: 14,
      textColor: '#666666',
      countColor: '#2563eb',
      padding: { top: 8, right: 8, bottom: 8, left: 8 },
    },
  },
];

export function getDefaultBlockProps(type: BlockType): Record<string, unknown> {
  const definition = blockDefinitions.find((d) => d.type === type);
  if (!definition) {
    console.warn(`No default props found for block type: ${type}`);
    return {};
  }
  return { ...definition.defaultProps };
}

export function getBlocksByCategory(category: BlockDefinition['category']): BlockDefinition[] {
  return blockDefinitions.filter((d) => d.category === category);
}
