import type { FormConfig, ShowOptions } from './types';
import { FormRenderer } from './FormRenderer';
import { evaluateTargeting } from './utils/targeting';
import { setupTrigger, cleanupTriggers } from './utils/triggers';
import {
  getVisitorData,
  shouldShowForm,
  recordImpression,
  recordSubmission,
  recordClose,
  incrementPageViews,
} from './utils/storage';
import { getDeviceType } from './utils/device';

interface ActiveForm {
  renderer: FormRenderer;
  config: FormConfig;
  options: ShowOptions;
}

export class FormController {
  private baseUrl: string;
  private activeForms: Map<string, ActiveForm> = new Map();
  private loadedConfigs: Map<string, FormConfig> = new Map();
  private currentFormPriority: number = 0;

  constructor(baseUrl: string = '') {
    this.baseUrl = baseUrl || this.detectBaseUrl();
    this.init();
  }

  private detectBaseUrl(): string {
    // Try to detect base URL from script tag
    const scripts = document.querySelectorAll('script[src*="lm-forms"]');
    if (scripts.length > 0) {
      const src = (scripts[scripts.length - 1] as HTMLScriptElement).src;
      const url = new URL(src);
      return url.origin;
    }
    return window.location.origin;
  }

  private init(): void {
    // Increment page views
    incrementPageViews();

    // Auto-load forms from script tags
    this.autoLoadFromScripts();

    // Auto-discover active forms from API (for popups, flyouts, etc.)
    this.autoDiscoverForms();

    // Add global styles
    this.addGlobalStyles();
  }

  private autoLoadFromScripts(): void {
    const scripts = document.querySelectorAll('script[data-form]');
    scripts.forEach((script) => {
      const formUuid = script.getAttribute('data-form');
      const formType = script.getAttribute('data-type');
      let target = script.getAttribute('data-target');

      if (formUuid) {
        // For embed forms without a target, auto-create a container div before the script
        if (formType === 'embed' && !target) {
          const containerId = `lm-form-${formUuid}`;
          // Check if container already exists
          let container = document.getElementById(containerId);
          if (!container) {
            container = document.createElement('div');
            container.id = containerId;
            script.parentNode?.insertBefore(container, script);
          }
          target = `#${containerId}`;
        }

        this.load(formUuid, {
          trigger: formType === 'embed' ? 'manual' : 'auto',
          target: target || undefined,
          scriptElement: script as HTMLScriptElement,
          formTypeOverride: formType as ShowOptions['formTypeOverride'],
        });
      }
    });
  }

  private async autoDiscoverForms(): Promise<void> {
    try {
      // Build query params for targeting
      const params = new URLSearchParams();
      params.set('url', window.location.href);

      // Detect Shopify page type if on Shopify
      const shopifyPageType = this.detectShopifyPageType();
      if (shopifyPageType) {
        params.set('shopifyPages', shopifyPageType);
      }

      // Fetch active forms from API
      const response = await fetch(`${this.baseUrl}/api/public/forms/active?${params.toString()}`);
      if (!response.ok) {
        console.log('Failed to fetch active forms');
        return;
      }

      const result = await response.json();
      const forms = result.data as Array<{ uuid: string; formType: string }>;

      if (!forms || forms.length === 0) {
        return;
      }

      // Load non-embed forms (popups, flyouts, etc.) that weren't already loaded via script tags
      for (const form of forms) {
        // Skip embed forms - they need explicit placement via script tags
        if (form.formType === 'embed') {
          continue;
        }

        // Skip if already loaded via script tag
        if (this.loadedConfigs.has(form.uuid)) {
          continue;
        }

        // Load the form with auto trigger
        this.load(form.uuid, { trigger: 'auto' });
      }
    } catch (error) {
      console.error('Error auto-discovering forms:', error);
    }
  }

  private detectShopifyPageType(): string | null {
    // Check for Shopify meta tags or URL patterns
    const shopifyMeta = document.querySelector('meta[name="shopify-checkout-api-token"]');
    const isShopify = shopifyMeta || window.location.hostname.includes('.myshopify.com') ||
      document.querySelector('script[src*="shopify"]');

    if (!isShopify) {
      return null;
    }

    const path = window.location.pathname;

    // Detect page type from URL
    if (path === '/' || path === '') {
      return 'homepage';
    }
    if (path.includes('/products/')) {
      return 'product';
    }
    if (path === '/collections' || path.includes('/collections/')) {
      return 'collection';
    }
    if (path === '/cart') {
      return 'cart';
    }
    if (path.includes('/checkouts/')) {
      return 'checkout';
    }
    if (path === '/pages' || path.includes('/pages/')) {
      return 'page';
    }
    if (path === '/blogs' || path.includes('/blogs/')) {
      return 'blog';
    }
    if (path === '/account' || path.includes('/account/')) {
      return 'account';
    }

    return 'other';
  }

  private addGlobalStyles(): void {
    if (document.getElementById('lm-forms-styles')) return;

    const style = document.createElement('style');
    style.id = 'lm-forms-styles';
    style.textContent = `
      .lm-form-container * {
        box-sizing: border-box;
      }
      .lm-form-wrapper {
        animation: lmFormFadeIn 0.3s ease-out;
      }
      @keyframes lmFormFadeIn {
        from { opacity: 0; }
        to { opacity: 1; }
      }
      @keyframes lmFormSlideIn {
        from { transform: translateY(20px); opacity: 0; }
        to { transform: translateY(0); opacity: 1; }
      }
      @keyframes lmFormZoomIn {
        from { transform: scale(0.9); opacity: 0; }
        to { transform: scale(1); opacity: 1; }
      }
      .lm-form input:focus,
      .lm-form select:focus,
      .lm-form textarea:focus {
        outline: 2px solid #2563eb;
        outline-offset: 1px;
      }
      .lm-form button:hover {
        opacity: 0.9;
      }
      .lm-form-close:hover {
        color: #333;
      }
    `;
    document.head.appendChild(style);
  }

  async load(formUuid: string, options: ShowOptions = {}): Promise<void> {
    try {
      // Fetch form config from API
      const response = await fetch(`${this.baseUrl}/api/public/forms/${formUuid}`);
      if (!response.ok) {
        console.error(`Failed to load form ${formUuid}: ${response.status}`);
        return;
      }

      const config = (await response.json()).data as FormConfig;
      this.loadedConfigs.set(formUuid, config);

      // Check if form should be shown
      if (config.status !== 'active') {
        console.log(`Form ${formUuid} is not active`);
        return;
      }

      // Use formTypeOverride if provided (from data-type attribute), otherwise use config.formType
      const effectiveFormType = options.formTypeOverride || config.formType;

      // Skip targeting/frequency rules for embed forms - they should always show
      // Embed forms are typically placed in footers, sidebars, etc. and should be visible every time
      if (effectiveFormType !== 'embed') {
        // Evaluate targeting rules
        if (!evaluateTargeting(config.targeting)) {
          console.log(`Form ${formUuid} targeting rules not matched`);
          return;
        }

        // Check frequency/suppression rules
        if (!shouldShowForm(formUuid, config.frequency)) {
          console.log(`Form ${formUuid} suppressed by frequency rules`);
          return;
        }

        // Check collision prevention (only one popup/modal form at a time)
        // Embed forms should NOT block popup forms since they occupy different spaces
        const activePopupForms = Array.from(this.activeForms.values()).filter(
          (f) => f.config.formType !== 'embed'
        );
        if (activePopupForms.length > 0 && config.frequency.priority <= this.currentFormPriority) {
          console.log(`Form ${formUuid} blocked by higher priority popup form`);
          return;
        }
      }

      // For embed type forms, handle auto-creation of container if needed
      if (effectiveFormType === 'embed') {
        let target = options.target;

        // If no target but we have a script element, auto-create a container
        if (!target && options.scriptElement) {
          const containerId = `lm-form-${formUuid}`;
          let container = document.getElementById(containerId);
          if (!container) {
            container = document.createElement('div');
            container.id = containerId;
            options.scriptElement.parentNode?.insertBefore(container, options.scriptElement);
          }
          target = `#${containerId}`;
        }

        if (target) {
          this.show(formUuid, { ...options, target });
          return;
        }
      }

      // For manual trigger type, don't set up automatic triggers
      if (config.triggers.type === 'manual' && options.trigger !== 'manual') {
        console.log(`Form ${formUuid} requires manual trigger`);
        return;
      }

      // Check page count trigger
      if (config.triggers.type === 'pages') {
        const visitor = getVisitorData();
        if (visitor.pageViews < config.triggers.pageCount) {
          console.log(`Form ${formUuid} waiting for more page views`);
          return;
        }
      }

      // Setup triggers
      setupTrigger(config.triggers, () => {
        this.show(formUuid, options);
      });
    } catch (error) {
      console.error(`Error loading form ${formUuid}:`, error);
    }
  }

  show(formUuid: string, options: ShowOptions = {}): void {
    const config = this.loadedConfigs.get(formUuid);
    if (!config) {
      console.error(`Form ${formUuid} not loaded. Call load() first.`);
      return;
    }

    // Close any existing forms with lower priority
    if (this.activeForms.size > 0) {
      for (const [uuid, active] of this.activeForms) {
        if (active.config.frequency.priority < config.frequency.priority) {
          this.close(uuid);
        }
      }
    }

    // Create and render form - apply formTypeOverride if present
    const effectiveConfig = options.formTypeOverride
      ? { ...config, formType: options.formTypeOverride }
      : config;
    const renderer = new FormRenderer(effectiveConfig, this.baseUrl);
    const element = renderer.render(options.target);

    // Store active form
    this.activeForms.set(formUuid, {
      renderer,
      config,
      options,
    });

    // Only track priority for popup/modal forms, not embed forms
    if (effectiveConfig.formType !== 'embed') {
      this.currentFormPriority = config.frequency.priority;
    }

    // Record impression
    recordImpression(formUuid);
    this.trackImpression(formUuid);

    // Setup close handlers
    this.setupCloseHandlers(formUuid, element, options);

    // Callback
    if (options.onShow) {
      options.onShow();
    }
  }

  private setupCloseHandlers(formUuid: string, element: HTMLElement, options: ShowOptions): void {
    // Close button handler
    const closeBtn = element.querySelector('.lm-form-close');
    if (closeBtn) {
      closeBtn.addEventListener('click', () => {
        this.close(formUuid);
        if (options.onClose) {
          options.onClose();
        }
      });
    }

    // Escape key handler
    const escHandler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        this.close(formUuid);
        if (options.onClose) {
          options.onClose();
        }
      }
    };
    document.addEventListener('keydown', escHandler);

    // Form submission handler
    const form = element.querySelector('form');
    if (form) {
      form.addEventListener('submit', () => {
        recordSubmission(formUuid);
        if (options.onSubmit) {
          const formData = new FormData(form);
          const data: Record<string, unknown> = {};
          formData.forEach((value, key) => {
            data[key] = value;
          });
          options.onSubmit(data);
        }
      });
    }
  }

  close(formUuid: string): void {
    const active = this.activeForms.get(formUuid);
    if (active) {
      active.renderer.close();
      this.activeForms.delete(formUuid);
      recordClose(formUuid);
      cleanupTriggers();

      // Update current priority (only from non-embed forms)
      const activePopupForms = Array.from(this.activeForms.values()).filter(
        (f) => f.config.formType !== 'embed'
      );
      if (activePopupForms.length === 0) {
        this.currentFormPriority = 0;
      } else {
        this.currentFormPriority = Math.max(
          ...activePopupForms.map((f) => f.config.frequency.priority)
        );
      }
    }
  }

  closeAll(): void {
    for (const uuid of this.activeForms.keys()) {
      this.close(uuid);
    }
  }

  private async trackImpression(formUuid: string): Promise<void> {
    try {
      const visitor = getVisitorData();
      await fetch(`${this.baseUrl}/api/public/forms/${formUuid}/impression`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          session_id: visitor.sessionId,
          visitor_id: visitor.visitorId,
          page_url: window.location.href,
          device_type: getDeviceType(),
        }),
      });
    } catch (error) {
      // Ignore tracking errors
    }
  }

  isFormActive(formUuid: string): boolean {
    return this.activeForms.has(formUuid);
  }

  getActiveFormIds(): string[] {
    return Array.from(this.activeForms.keys());
  }
}
