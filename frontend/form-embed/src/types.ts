// Form configuration from API
export interface FormConfig {
  uuid: string;
  name: string;
  formType: 'popup' | 'flyout' | 'fullpage' | 'embed' | 'banner';
  status: 'draft' | 'active' | 'paused' | 'archived';
  steps: FormStep[];
  settings: FormSettings;
  targeting: TargetingRules;
  triggers: TriggerConfig;
  frequency: FrequencyConfig;
  customCss: string;
}

export interface FormStep {
  id: string;
  name: string;
  blocks: FormBlock[];
}

export interface FormBlock {
  id: string;
  type: string;
  props: Record<string, unknown>;
}

export interface FormSettings {
  width: string;
  maxWidth: string;
  backgroundColor: string;
  borderRadius: number;
  padding: { top: number; right: number; bottom: number; left: number };
  overlay: boolean;
  overlayColor: string;
  overlayOpacity: number;
  closeOnOverlayClick: boolean;
  animation: 'none' | 'fade' | 'slide' | 'zoom';
  position?: 'center' | 'top' | 'bottom' | 'left' | 'right' | 'top-left' | 'top-right' | 'bottom-left' | 'bottom-right';
  successAction: 'message' | 'redirect' | 'close';
  successMessage: string;
  successRedirectUrl: string;
}

export interface TargetingRules {
  urlPatterns: string[];
  excludeUrlPatterns: string[];
  utmParams: UtmRule[];
  deviceTypes: ('desktop' | 'tablet' | 'mobile')[];
  countries: string[];
  excludeSubscribers: boolean;
}

export interface UtmRule {
  key: string;
  operator: 'equals' | 'contains' | 'starts_with';
  value: string;
}

export interface TriggerConfig {
  type: 'immediate' | 'time' | 'scroll' | 'pages' | 'manual';
  delay: number;
  scrollDepth: number;
  pageCount: number;
  exitIntentEnabled: boolean;
  mobileExitAlternative: 'none' | 'scroll_up' | 'back_button';
}

export interface FrequencyConfig {
  showAgainAfterDays: number;
  suppressAfterSubmission: boolean;
  suppressDays: number;
  maxImpressions: number;
  priority: number;
  cookieDays: number;
}

export interface ShowOptions {
  trigger?: 'auto' | 'manual';
  target?: string; // CSS selector for embed forms
  scriptElement?: HTMLScriptElement; // Reference to the script tag that loaded this form
  onShow?: () => void;
  onClose?: () => void;
  onSubmit?: (data: Record<string, unknown>) => void;
}

export interface FormState {
  config: FormConfig | null;
  currentStep: number;
  isVisible: boolean;
  isSubmitting: boolean;
  formData: Record<string, unknown>;
  element: HTMLElement | null;
}

export interface VisitorData {
  visitorId: string;
  sessionId: string;
  pageViews: number;
  forms: Record<string, FormVisitorData>;
}

export interface FormVisitorData {
  impressions: number;
  lastShown: number;
  submitted: boolean;
  submittedAt?: number;
  closedAt?: number;
}
