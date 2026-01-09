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

    // Add global styles
    this.addGlobalStyles();
  }

  private autoLoadFromScripts(): void {
    const scripts = document.querySelectorAll('script[data-form]');
    scripts.forEach((script) => {
      const formUuid = script.getAttribute('data-form');
      const formType = script.getAttribute('data-type');
      const target = script.getAttribute('data-target');

      if (formUuid) {
        this.load(formUuid, {
          trigger: formType === 'embed' ? 'manual' : 'auto',
          target: target || undefined,
        });
      }
    });
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

      // Check collision prevention (only one form at a time)
      if (this.activeForms.size > 0 && config.frequency.priority <= this.currentFormPriority) {
        console.log(`Form ${formUuid} blocked by higher priority form`);
        return;
      }

      // For embed type, show immediately at target
      if (config.formType === 'embed' && options.target) {
        this.show(formUuid, options);
        return;
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

    // Create and render form
    const renderer = new FormRenderer(config);
    const element = renderer.render(options.target);

    // Store active form
    this.activeForms.set(formUuid, {
      renderer,
      config,
      options,
    });
    this.currentFormPriority = config.frequency.priority;

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

      // Update current priority
      if (this.activeForms.size === 0) {
        this.currentFormPriority = 0;
      } else {
        this.currentFormPriority = Math.max(
          ...Array.from(this.activeForms.values()).map((f) => f.config.frequency.priority)
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
          sessionId: visitor.sessionId,
          visitorId: visitor.visitorId,
          pageUrl: window.location.href,
          deviceType: getDeviceType(),
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
