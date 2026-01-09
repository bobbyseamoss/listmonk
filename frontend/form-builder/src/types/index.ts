// Block types for the form builder
export type BlockType =
  | 'text'
  | 'heading'
  | 'image'
  | 'email-input'
  | 'text-input'
  | 'textarea'
  | 'radio-group'
  | 'checkbox-group'
  | 'dropdown'
  | 'date-picker'
  | 'hidden-field'
  | 'button'
  | 'divider'
  | 'spacer'
  | 'coupon'
  | 'countdown'
  | 'signup-counter';

// Form types
export type FormType = 'popup' | 'flyout' | 'fullpage' | 'embed' | 'banner';

// Form status
export type FormStatus = 'draft' | 'active' | 'paused' | 'archived';

// Base block interface
export interface BaseBlock {
  id: string;
  type: BlockType;
  props: Record<string, unknown>;
}

// Text block
export interface TextBlockProps {
  content: string;
  fontSize: number;
  fontWeight: string;
  color: string;
  textAlign: 'left' | 'center' | 'right';
  padding: { top: number; right: number; bottom: number; left: number };
}

export interface TextBlock extends BaseBlock {
  type: 'text';
  props: TextBlockProps;
}

// Heading block
export interface HeadingBlockProps {
  content: string;
  level: 1 | 2 | 3 | 4 | 5 | 6;
  fontSize: number;
  fontWeight: string;
  color: string;
  textAlign: 'left' | 'center' | 'right';
  padding: { top: number; right: number; bottom: number; left: number };
}

export interface HeadingBlock extends BaseBlock {
  type: 'heading';
  props: HeadingBlockProps;
}

// Image block
export interface ImageBlockProps {
  src: string;
  alt: string;
  width: string;
  height: string;
  alignment: 'left' | 'center' | 'right';
  padding: { top: number; right: number; bottom: number; left: number };
}

export interface ImageBlock extends BaseBlock {
  type: 'image';
  props: ImageBlockProps;
}

// Email input block
export interface EmailInputBlockProps {
  label: string;
  placeholder: string;
  required: boolean;
  fontSize: number;
  labelColor: string;
  inputBgColor: string;
  inputBorderColor: string;
  inputTextColor: string;
  borderRadius: number;
  padding: { top: number; right: number; bottom: number; left: number };
}

export interface EmailInputBlock extends BaseBlock {
  type: 'email-input';
  props: EmailInputBlockProps;
}

// Text input block
export interface TextInputBlockProps {
  name: string;
  label: string;
  placeholder: string;
  required: boolean;
  fontSize: number;
  labelColor: string;
  inputBgColor: string;
  inputBorderColor: string;
  inputTextColor: string;
  borderRadius: number;
  padding: { top: number; right: number; bottom: number; left: number };
}

export interface TextInputBlock extends BaseBlock {
  type: 'text-input';
  props: TextInputBlockProps;
}

// Textarea block
export interface TextareaBlockProps {
  name: string;
  label: string;
  placeholder: string;
  required: boolean;
  rows: number;
  fontSize: number;
  labelColor: string;
  inputBgColor: string;
  inputBorderColor: string;
  inputTextColor: string;
  borderRadius: number;
  padding: { top: number; right: number; bottom: number; left: number };
}

export interface TextareaBlock extends BaseBlock {
  type: 'textarea';
  props: TextareaBlockProps;
}

// Radio group block
export interface RadioGroupBlockProps {
  name: string;
  label: string;
  options: Array<{ value: string; label: string }>;
  required: boolean;
  layout: 'vertical' | 'horizontal';
  fontSize: number;
  labelColor: string;
  padding: { top: number; right: number; bottom: number; left: number };
}

export interface RadioGroupBlock extends BaseBlock {
  type: 'radio-group';
  props: RadioGroupBlockProps;
}

// Checkbox group block
export interface CheckboxGroupBlockProps {
  name: string;
  label: string;
  options: Array<{ value: string; label: string }>;
  required: boolean;
  layout: 'vertical' | 'horizontal';
  fontSize: number;
  labelColor: string;
  padding: { top: number; right: number; bottom: number; left: number };
}

export interface CheckboxGroupBlock extends BaseBlock {
  type: 'checkbox-group';
  props: CheckboxGroupBlockProps;
}

// Dropdown block
export interface DropdownBlockProps {
  name: string;
  label: string;
  placeholder: string;
  options: Array<{ value: string; label: string }>;
  required: boolean;
  fontSize: number;
  labelColor: string;
  inputBgColor: string;
  inputBorderColor: string;
  inputTextColor: string;
  borderRadius: number;
  padding: { top: number; right: number; bottom: number; left: number };
}

export interface DropdownBlock extends BaseBlock {
  type: 'dropdown';
  props: DropdownBlockProps;
}

// Date picker block
export interface DatePickerBlockProps {
  name: string;
  label: string;
  required: boolean;
  fontSize: number;
  labelColor: string;
  inputBgColor: string;
  inputBorderColor: string;
  inputTextColor: string;
  borderRadius: number;
  padding: { top: number; right: number; bottom: number; left: number };
}

export interface DatePickerBlock extends BaseBlock {
  type: 'date-picker';
  props: DatePickerBlockProps;
}

// Hidden field block
export interface HiddenFieldBlockProps {
  name: string;
  value: string;
  source: 'static' | 'url-param' | 'utm';
  paramName?: string;
}

export interface HiddenFieldBlock extends BaseBlock {
  type: 'hidden-field';
  props: HiddenFieldBlockProps;
}

// Button block
export interface ButtonBlockProps {
  text: string;
  action: 'submit' | 'close' | 'next-step' | 'redirect';
  redirectUrl?: string;
  fontSize: number;
  fontWeight: string;
  textColor: string;
  bgColor: string;
  hoverBgColor: string;
  borderRadius: number;
  fullWidth: boolean;
  padding: { top: number; right: number; bottom: number; left: number };
  margin: { top: number; right: number; bottom: number; left: number };
}

export interface ButtonBlock extends BaseBlock {
  type: 'button';
  props: ButtonBlockProps;
}

// Divider block
export interface DividerBlockProps {
  color: string;
  thickness: number;
  style: 'solid' | 'dashed' | 'dotted';
  margin: { top: number; bottom: number };
}

export interface DividerBlock extends BaseBlock {
  type: 'divider';
  props: DividerBlockProps;
}

// Spacer block
export interface SpacerBlockProps {
  height: number;
}

export interface SpacerBlock extends BaseBlock {
  type: 'spacer';
  props: SpacerBlockProps;
}

// Coupon block
export interface CouponBlockProps {
  title: string;
  fontSize: number;
  fontWeight: string;
  textColor: string;
  bgColor: string;
  borderColor: string;
  borderRadius: number;
  padding: { top: number; right: number; bottom: number; left: number };
}

export interface CouponBlock extends BaseBlock {
  type: 'coupon';
  props: CouponBlockProps;
}

// Countdown block
export interface CountdownBlockProps {
  endDate: string;
  showDays: boolean;
  showHours: boolean;
  showMinutes: boolean;
  showSeconds: boolean;
  fontSize: number;
  textColor: string;
  bgColor: string;
  borderRadius: number;
  padding: { top: number; right: number; bottom: number; left: number };
}

export interface CountdownBlock extends BaseBlock {
  type: 'countdown';
  props: CountdownBlockProps;
}

// Signup counter block
export interface SignupCounterBlockProps {
  text: string;
  count: number;
  fontSize: number;
  textColor: string;
  countColor: string;
  padding: { top: number; right: number; bottom: number; left: number };
}

export interface SignupCounterBlock extends BaseBlock {
  type: 'signup-counter';
  props: SignupCounterBlockProps;
}

// Union type for all blocks
export type Block =
  | TextBlock
  | HeadingBlock
  | ImageBlock
  | EmailInputBlock
  | TextInputBlock
  | TextareaBlock
  | RadioGroupBlock
  | CheckboxGroupBlock
  | DropdownBlock
  | DatePickerBlock
  | HiddenFieldBlock
  | ButtonBlock
  | DividerBlock
  | SpacerBlock
  | CouponBlock
  | CountdownBlock
  | SignupCounterBlock;

// Form step
export interface FormStep {
  id: string;
  name: string;
  blocks: Block[];
}

// Form settings
export interface FormSettings {
  position: 'center' | 'top-left' | 'top-right' | 'bottom-left' | 'bottom-right' | 'top' | 'bottom';
  animation: 'fade' | 'slide' | 'zoom' | 'none';
  overlay: boolean;
  overlayColor: string;
  overlayOpacity: number;
  closeButton: boolean;
  closeOnOverlayClick: boolean;
  successAction: 'message' | 'redirect' | 'close';
  successMessage: string;
  successRedirectUrl: string;
  width: string;
  maxWidth: string;
  backgroundColor: string;
  borderRadius: number;
  padding: { top: number; right: number; bottom: number; left: number };
}

// Targeting configuration
export interface FormTargeting {
  urlPatterns: Array<{ type: 'contains' | 'exact' | 'regex'; value: string }>;
  utmParams: Array<{ key: string; value: string }>;
  devices: ('desktop' | 'tablet' | 'mobile')[];
  countries: string[];
  excludeSubscribers: boolean;
}

// Trigger configuration
export interface FormTriggers {
  type: 'immediate' | 'time-delay' | 'scroll-depth' | 'exit-intent' | 'page-count' | 'manual';
  timeDelay?: number;
  scrollDepth?: number;
  pageCount?: number;
}

// Frequency configuration
export interface FormFrequency {
  showAgainAfterDays: number;
  suppressAfterSubmit: boolean;
  maxShowsPerSession: number;
}

// Coupon configuration
export interface CouponConfig {
  enabled: boolean;
  type: 'static' | 'unique';
  staticCode?: string;
}

// Complete form configuration
export interface FormConfig {
  id?: number;
  uuid?: string;
  name: string;
  description: string;
  formType: FormType;
  status: FormStatus;
  steps: FormStep[];
  settings: FormSettings;
  targeting: FormTargeting;
  triggers: FormTriggers;
  frequency: FormFrequency;
  listIds: number[];
  couponConfig: CouponConfig;
  customCss: string;
}

// Block definition for sidebar
export interface BlockDefinition {
  type: BlockType;
  name: string;
  icon: string;
  category: 'content' | 'input' | 'action' | 'layout' | 'special';
  defaultProps: Record<string, unknown>;
}
