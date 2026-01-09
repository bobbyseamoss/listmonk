import type { FormConfig, FormBlock, FormStep, FormSettings } from './types';

export class FormRenderer {
  private config: FormConfig;
  private currentStep: number = 0;
  private formData: Record<string, unknown> = {};
  private element: HTMLElement | null = null;

  constructor(config: FormConfig) {
    this.config = config;
  }

  render(targetSelector?: string): HTMLElement {
    const container = document.createElement('div');
    container.className = 'lm-form-container';
    container.setAttribute('data-form-uuid', this.config.uuid);
    container.setAttribute('data-form-type', this.config.formType);

    // Add custom CSS
    if (this.config.customCss) {
      const style = document.createElement('style');
      style.textContent = this.config.customCss;
      container.appendChild(style);
    }

    // Create form wrapper based on type
    const wrapper = this.createWrapper();
    container.appendChild(wrapper);

    // Render current step
    const stepContent = this.renderStep(this.config.steps[this.currentStep]);
    wrapper.querySelector('.lm-form-content')?.appendChild(stepContent);

    this.element = container;

    // For embed forms, insert into target
    if (this.config.formType === 'embed' && targetSelector) {
      const target = document.querySelector(targetSelector);
      if (target) {
        target.appendChild(container);
        return container;
      }
    }

    // For other types, add to body
    document.body.appendChild(container);
    return container;
  }

  private createWrapper(): HTMLElement {
    const settings = this.config.settings;
    const wrapper = document.createElement('div');
    wrapper.className = `lm-form-wrapper lm-form-${this.config.formType}`;

    // Create overlay for popup/flyout/fullpage
    if (['popup', 'flyout', 'fullpage'].includes(this.config.formType) && settings.overlay) {
      const overlay = document.createElement('div');
      overlay.className = 'lm-form-overlay';
      overlay.style.cssText = `
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background-color: ${settings.overlayColor};
        opacity: ${settings.overlayOpacity};
        z-index: 99998;
      `;
      if (settings.closeOnOverlayClick) {
        overlay.addEventListener('click', () => this.close());
      }
      wrapper.appendChild(overlay);
    }

    // Create form content container
    const content = document.createElement('div');
    content.className = 'lm-form-content';
    content.style.cssText = this.getContentStyles();

    // Add close button for closeable forms
    if (['popup', 'flyout', 'fullpage', 'banner'].includes(this.config.formType)) {
      const closeBtn = document.createElement('button');
      closeBtn.className = 'lm-form-close';
      closeBtn.innerHTML = '&times;';
      closeBtn.setAttribute('aria-label', 'Close');
      closeBtn.style.cssText = `
        position: absolute;
        top: 8px;
        right: 8px;
        background: none;
        border: none;
        font-size: 24px;
        cursor: pointer;
        color: #666;
        line-height: 1;
        padding: 4px;
      `;
      closeBtn.addEventListener('click', () => this.close());
      content.appendChild(closeBtn);
    }

    wrapper.appendChild(content);
    return wrapper;
  }

  private getContentStyles(): string {
    const settings = this.config.settings;
    const padding = settings.padding;
    let styles = `
      position: relative;
      width: ${settings.width};
      max-width: ${settings.maxWidth};
      background-color: ${settings.backgroundColor};
      border-radius: ${settings.borderRadius}px;
      padding: ${padding.top}px ${padding.right}px ${padding.bottom}px ${padding.left}px;
      box-sizing: border-box;
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
    `;

    // Position based on form type
    switch (this.config.formType) {
      case 'popup':
        styles += `
          position: fixed;
          top: 50%;
          left: 50%;
          transform: translate(-50%, -50%);
          z-index: 99999;
          box-shadow: 0 10px 40px rgba(0,0,0,0.2);
        `;
        break;

      case 'flyout':
        const position = settings.position || 'right';
        styles += `
          position: fixed;
          z-index: 99999;
          box-shadow: 0 0 20px rgba(0,0,0,0.15);
        `;
        if (position.includes('right')) {
          styles += 'right: 20px;';
        } else {
          styles += 'left: 20px;';
        }
        if (position.includes('top')) {
          styles += 'top: 20px;';
        } else {
          styles += 'bottom: 20px;';
        }
        break;

      case 'fullpage':
        styles += `
          position: fixed;
          top: 0;
          left: 0;
          width: 100%;
          height: 100%;
          max-width: none;
          z-index: 99999;
          display: flex;
          flex-direction: column;
          justify-content: center;
          align-items: center;
        `;
        break;

      case 'banner':
        const bannerPosition = settings.position || 'bottom';
        styles += `
          position: fixed;
          left: 0;
          width: 100%;
          max-width: none;
          z-index: 99999;
          box-shadow: 0 -2px 10px rgba(0,0,0,0.1);
        `;
        if (bannerPosition === 'top') {
          styles += 'top: 0;';
        } else {
          styles += 'bottom: 0;';
        }
        break;

      case 'embed':
        // No special positioning for embedded forms
        break;
    }

    return styles;
  }

  private renderStep(step: FormStep): HTMLElement {
    const container = document.createElement('div');
    container.className = 'lm-form-step';
    container.setAttribute('data-step-id', step.id);

    // Create form element
    const form = document.createElement('form');
    form.className = 'lm-form';
    form.addEventListener('submit', (e) => this.handleSubmit(e));

    // Render blocks
    step.blocks.forEach((block) => {
      const blockEl = this.renderBlock(block);
      form.appendChild(blockEl);
    });

    container.appendChild(form);
    return container;
  }

  private renderBlock(block: FormBlock): HTMLElement {
    const container = document.createElement('div');
    container.className = `lm-block lm-block-${block.type}`;
    container.setAttribute('data-block-id', block.id);

    const props = block.props as Record<string, unknown>;

    switch (block.type) {
      case 'text':
        container.innerHTML = this.renderTextBlock(props);
        break;
      case 'heading':
        container.innerHTML = this.renderHeadingBlock(props);
        break;
      case 'image':
        container.innerHTML = this.renderImageBlock(props);
        break;
      case 'email-input':
        container.innerHTML = this.renderEmailInputBlock(props);
        break;
      case 'text-input':
        container.innerHTML = this.renderTextInputBlock(props);
        break;
      case 'textarea':
        container.innerHTML = this.renderTextareaBlock(props);
        break;
      case 'radio-group':
        container.innerHTML = this.renderRadioGroupBlock(props);
        break;
      case 'checkbox-group':
        container.innerHTML = this.renderCheckboxGroupBlock(props);
        break;
      case 'dropdown':
        container.innerHTML = this.renderDropdownBlock(props);
        break;
      case 'date-picker':
        container.innerHTML = this.renderDatePickerBlock(props);
        break;
      case 'hidden-field':
        container.innerHTML = this.renderHiddenFieldBlock(props);
        break;
      case 'button':
        container.innerHTML = this.renderButtonBlock(props);
        break;
      case 'divider':
        container.innerHTML = this.renderDividerBlock(props);
        break;
      case 'spacer':
        container.innerHTML = this.renderSpacerBlock(props);
        break;
      case 'coupon':
        container.innerHTML = this.renderCouponBlock(props);
        break;
      case 'countdown':
        container.innerHTML = this.renderCountdownBlock(props);
        break;
      case 'signup-counter':
        container.innerHTML = this.renderSignupCounterBlock(props);
        break;
      default:
        container.innerHTML = `<div>Unknown block type: ${block.type}</div>`;
    }

    return container;
  }

  private renderTextBlock(props: Record<string, unknown>): string {
    const padding = props.padding as { top: number; right: number; bottom: number; left: number };
    return `<div style="
      font-size: ${props.fontSize}px;
      font-weight: ${props.fontWeight};
      color: ${props.color};
      text-align: ${props.textAlign};
      padding: ${padding.top}px ${padding.right}px ${padding.bottom}px ${padding.left}px;
    ">${props.content}</div>`;
  }

  private renderHeadingBlock(props: Record<string, unknown>): string {
    const level = props.level || 2;
    const padding = props.padding as { top: number; right: number; bottom: number; left: number };
    return `<h${level} style="
      font-size: ${props.fontSize}px;
      font-weight: ${props.fontWeight};
      color: ${props.color};
      text-align: ${props.textAlign};
      padding: ${padding.top}px ${padding.right}px ${padding.bottom}px ${padding.left}px;
      margin: 0;
    ">${props.content}</h${level}>`;
  }

  private renderImageBlock(props: Record<string, unknown>): string {
    const padding = props.padding as { top: number; right: number; bottom: number; left: number };
    if (!props.src) {
      return '';
    }
    return `<div style="text-align: ${props.alignment}; padding: ${padding.top}px ${padding.right}px ${padding.bottom}px ${padding.left}px;">
      <img src="${props.src}" alt="${props.alt || ''}" style="width: ${props.width}; height: ${props.height}; max-width: 100%;" />
    </div>`;
  }

  private renderEmailInputBlock(props: Record<string, unknown>): string {
    const padding = props.padding as { top: number; right: number; bottom: number; left: number };
    const required = props.required ? 'required' : '';
    return `<div style="padding: ${padding.top}px ${padding.right}px ${padding.bottom}px ${padding.left}px;">
      <label style="display: block; margin-bottom: 4px; font-size: ${props.fontSize}px; color: ${props.labelColor};">
        ${props.label} ${props.required ? '<span style="color: red;">*</span>' : ''}
      </label>
      <input type="email" name="${props.name}" placeholder="${props.placeholder}" ${required}
        style="width: 100%; padding: 10px 12px; font-size: ${props.fontSize}px;
        background-color: ${props.inputBgColor}; border: 1px solid ${props.inputBorderColor};
        border-radius: ${props.borderRadius}px; color: ${props.inputTextColor}; box-sizing: border-box;" />
    </div>`;
  }

  private renderTextInputBlock(props: Record<string, unknown>): string {
    const padding = props.padding as { top: number; right: number; bottom: number; left: number };
    const required = props.required ? 'required' : '';
    return `<div style="padding: ${padding.top}px ${padding.right}px ${padding.bottom}px ${padding.left}px;">
      <label style="display: block; margin-bottom: 4px; font-size: ${props.fontSize}px; color: ${props.labelColor};">
        ${props.label} ${props.required ? '<span style="color: red;">*</span>' : ''}
      </label>
      <input type="text" name="${props.name}" placeholder="${props.placeholder}" ${required}
        style="width: 100%; padding: 10px 12px; font-size: ${props.fontSize}px;
        background-color: ${props.inputBgColor}; border: 1px solid ${props.inputBorderColor};
        border-radius: ${props.borderRadius}px; color: ${props.inputTextColor}; box-sizing: border-box;" />
    </div>`;
  }

  private renderTextareaBlock(props: Record<string, unknown>): string {
    const padding = props.padding as { top: number; right: number; bottom: number; left: number };
    const required = props.required ? 'required' : '';
    return `<div style="padding: ${padding.top}px ${padding.right}px ${padding.bottom}px ${padding.left}px;">
      <label style="display: block; margin-bottom: 4px; font-size: ${props.fontSize}px; color: ${props.labelColor};">
        ${props.label} ${props.required ? '<span style="color: red;">*</span>' : ''}
      </label>
      <textarea name="${props.name}" placeholder="${props.placeholder}" rows="${props.rows}" ${required}
        style="width: 100%; padding: 10px 12px; font-size: ${props.fontSize}px;
        background-color: ${props.inputBgColor}; border: 1px solid ${props.inputBorderColor};
        border-radius: ${props.borderRadius}px; color: ${props.inputTextColor};
        box-sizing: border-box; resize: none;"></textarea>
    </div>`;
  }

  private renderRadioGroupBlock(props: Record<string, unknown>): string {
    const padding = props.padding as { top: number; right: number; bottom: number; left: number };
    const options = props.options as Array<{ label: string; value: string }>;
    const layout = props.layout === 'horizontal' ? 'row' : 'column';

    const optionsHtml = options.map((opt) => `
      <label style="display: flex; align-items: center; gap: 4px; font-size: ${props.fontSize}px;">
        <input type="radio" name="${props.name}" value="${opt.value}" />
        ${opt.label}
      </label>
    `).join('');

    return `<div style="padding: ${padding.top}px ${padding.right}px ${padding.bottom}px ${padding.left}px;">
      <label style="display: block; margin-bottom: 8px; font-size: ${props.fontSize}px; color: ${props.labelColor};">
        ${props.label} ${props.required ? '<span style="color: red;">*</span>' : ''}
      </label>
      <div style="display: flex; flex-direction: ${layout}; gap: 8px;">
        ${optionsHtml}
      </div>
    </div>`;
  }

  private renderCheckboxGroupBlock(props: Record<string, unknown>): string {
    const padding = props.padding as { top: number; right: number; bottom: number; left: number };
    const options = props.options as Array<{ label: string; value: string }>;
    const layout = props.layout === 'horizontal' ? 'row' : 'column';

    const optionsHtml = options.map((opt) => `
      <label style="display: flex; align-items: center; gap: 4px; font-size: ${props.fontSize}px;">
        <input type="checkbox" name="${props.name}" value="${opt.value}" />
        ${opt.label}
      </label>
    `).join('');

    return `<div style="padding: ${padding.top}px ${padding.right}px ${padding.bottom}px ${padding.left}px;">
      <label style="display: block; margin-bottom: 8px; font-size: ${props.fontSize}px; color: ${props.labelColor};">
        ${props.label} ${props.required ? '<span style="color: red;">*</span>' : ''}
      </label>
      <div style="display: flex; flex-direction: ${layout}; gap: 8px;">
        ${optionsHtml}
      </div>
    </div>`;
  }

  private renderDropdownBlock(props: Record<string, unknown>): string {
    const padding = props.padding as { top: number; right: number; bottom: number; left: number };
    const options = props.options as Array<{ label: string; value: string }>;
    const required = props.required ? 'required' : '';

    const optionsHtml = options.map((opt) => `
      <option value="${opt.value}">${opt.label}</option>
    `).join('');

    return `<div style="padding: ${padding.top}px ${padding.right}px ${padding.bottom}px ${padding.left}px;">
      <label style="display: block; margin-bottom: 4px; font-size: ${props.fontSize}px; color: ${props.labelColor};">
        ${props.label} ${props.required ? '<span style="color: red;">*</span>' : ''}
      </label>
      <select name="${props.name}" ${required}
        style="width: 100%; padding: 10px 12px; font-size: ${props.fontSize}px;
        background-color: ${props.inputBgColor}; border: 1px solid ${props.inputBorderColor};
        border-radius: ${props.borderRadius}px; color: ${props.inputTextColor};">
        <option value="">${props.placeholder}</option>
        ${optionsHtml}
      </select>
    </div>`;
  }

  private renderDatePickerBlock(props: Record<string, unknown>): string {
    const padding = props.padding as { top: number; right: number; bottom: number; left: number };
    const required = props.required ? 'required' : '';
    return `<div style="padding: ${padding.top}px ${padding.right}px ${padding.bottom}px ${padding.left}px;">
      <label style="display: block; margin-bottom: 4px; font-size: ${props.fontSize}px; color: ${props.labelColor};">
        ${props.label} ${props.required ? '<span style="color: red;">*</span>' : ''}
      </label>
      <input type="date" name="${props.name}" ${required}
        style="width: 100%; padding: 10px 12px; font-size: ${props.fontSize}px;
        background-color: ${props.inputBgColor}; border: 1px solid ${props.inputBorderColor};
        border-radius: ${props.borderRadius}px; color: ${props.inputTextColor}; box-sizing: border-box;" />
    </div>`;
  }

  private renderHiddenFieldBlock(props: Record<string, unknown>): string {
    let value = props.value as string;

    // Dynamic value based on source
    if (props.source === 'url') {
      const params = new URLSearchParams(window.location.search);
      value = params.get(props.paramName as string) || '';
    } else if (props.source === 'cookie') {
      const match = document.cookie.match(new RegExp(`(^| )${props.paramName}=([^;]+)`));
      value = match ? match[2] : '';
    }

    return `<input type="hidden" name="${props.name}" value="${value}" />`;
  }

  private renderButtonBlock(props: Record<string, unknown>): string {
    const padding = props.padding as { top: number; right: number; bottom: number; left: number };
    const margin = props.margin as { top: number; right: number; bottom: number; left: number };
    const action = props.action as string;

    let type = 'button';
    if (action === 'submit') {
      type = 'submit';
    }

    return `<div style="margin: ${margin.top}px ${margin.right}px ${margin.bottom}px ${margin.left}px;">
      <button type="${type}" data-action="${action}"
        style="width: ${props.fullWidth ? '100%' : 'auto'};
        padding: ${padding.top}px ${padding.right}px ${padding.bottom}px ${padding.left}px;
        font-size: ${props.fontSize}px; font-weight: ${props.fontWeight};
        color: ${props.textColor}; background-color: ${props.bgColor};
        border: none; border-radius: ${props.borderRadius}px; cursor: pointer;">
        ${props.text}
      </button>
    </div>`;
  }

  private renderDividerBlock(props: Record<string, unknown>): string {
    const margin = props.margin as { top: number; bottom: number };
    return `<hr style="
      margin: ${margin.top}px 0 ${margin.bottom}px 0;
      border: none;
      border-top: ${props.thickness}px ${props.style} ${props.color};
    " />`;
  }

  private renderSpacerBlock(props: Record<string, unknown>): string {
    return `<div style="height: ${props.height}px;"></div>`;
  }

  private renderCouponBlock(props: Record<string, unknown>): string {
    const padding = props.padding as { top: number; right: number; bottom: number; left: number };
    return `<div class="lm-coupon-block" style="
      padding: ${padding.top}px ${padding.right}px ${padding.bottom}px ${padding.left}px;
      background-color: ${props.bgColor};
      border: 1px solid ${props.borderColor};
      border-radius: ${props.borderRadius}px;
      text-align: center;
    ">
      <div style="font-size: ${(props.fontSize as number) - 2}px; color: ${props.textColor}; margin-bottom: 4px;">
        ${props.title}
      </div>
      <div class="lm-coupon-code" style="
        font-size: ${(props.fontSize as number) + 4}px;
        font-weight: ${props.fontWeight};
        color: ${props.textColor};
        letter-spacing: 2px;
      ">
        <!-- Coupon code will be filled after submission -->
        <span class="lm-coupon-placeholder">Submit to reveal</span>
      </div>
    </div>`;
  }

  private renderCountdownBlock(props: Record<string, unknown>): string {
    const padding = props.padding as { top: number; right: number; bottom: number; left: number };
    const endDate = new Date(props.endDate as string).getTime();
    const id = `countdown-${Date.now()}`;

    // Start countdown timer
    setTimeout(() => this.startCountdown(id, endDate, props), 0);

    return `<div id="${id}" style="
      padding: ${padding.top}px ${padding.right}px ${padding.bottom}px ${padding.left}px;
      background-color: ${props.bgColor};
      border-radius: ${props.borderRadius}px;
      display: flex;
      justify-content: center;
      gap: 12px;
    ">
      ${props.showDays ? `<div style="text-align: center;"><div class="days" style="font-size: ${props.fontSize}px; font-weight: bold; color: ${props.textColor};">00</div><div style="font-size: ${(props.fontSize as number) * 0.5}px; color: ${props.textColor};">Days</div></div>` : ''}
      ${props.showHours ? `<div style="text-align: center;"><div class="hours" style="font-size: ${props.fontSize}px; font-weight: bold; color: ${props.textColor};">00</div><div style="font-size: ${(props.fontSize as number) * 0.5}px; color: ${props.textColor};">Hours</div></div>` : ''}
      ${props.showMinutes ? `<div style="text-align: center;"><div class="minutes" style="font-size: ${props.fontSize}px; font-weight: bold; color: ${props.textColor};">00</div><div style="font-size: ${(props.fontSize as number) * 0.5}px; color: ${props.textColor};">Min</div></div>` : ''}
      ${props.showSeconds ? `<div style="text-align: center;"><div class="seconds" style="font-size: ${props.fontSize}px; font-weight: bold; color: ${props.textColor};">00</div><div style="font-size: ${(props.fontSize as number) * 0.5}px; color: ${props.textColor};">Sec</div></div>` : ''}
    </div>`;
  }

  private startCountdown(id: string, endDate: number, props: Record<string, unknown>): void {
    const el = document.getElementById(id);
    if (!el) return;

    const update = () => {
      const now = Date.now();
      const diff = Math.max(0, endDate - now);

      const days = Math.floor(diff / (1000 * 60 * 60 * 24));
      const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
      const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
      const seconds = Math.floor((diff % (1000 * 60)) / 1000);

      const daysEl = el.querySelector('.days');
      const hoursEl = el.querySelector('.hours');
      const minutesEl = el.querySelector('.minutes');
      const secondsEl = el.querySelector('.seconds');

      if (daysEl) daysEl.textContent = String(days).padStart(2, '0');
      if (hoursEl) hoursEl.textContent = String(hours).padStart(2, '0');
      if (minutesEl) minutesEl.textContent = String(minutes).padStart(2, '0');
      if (secondsEl) secondsEl.textContent = String(seconds).padStart(2, '0');

      if (diff > 0) {
        requestAnimationFrame(update);
      }
    };

    update();
  }

  private renderSignupCounterBlock(props: Record<string, unknown>): string {
    const padding = props.padding as { top: number; right: number; bottom: number; left: number };
    const text = (props.text as string).replace(
      '{count}',
      `<span style="color: ${props.countColor}; font-weight: bold;">${(props.count as number).toLocaleString()}</span>`
    );
    return `<div style="
      padding: ${padding.top}px ${padding.right}px ${padding.bottom}px ${padding.left}px;
      font-size: ${props.fontSize}px;
      color: ${props.textColor};
      text-align: center;
    ">${text}</div>`;
  }

  private async handleSubmit(e: Event): Promise<void> {
    e.preventDefault();
    const form = e.target as HTMLFormElement;
    const formData = new FormData(form);
    const data: Record<string, unknown> = {};

    formData.forEach((value, key) => {
      if (data[key]) {
        // Handle multiple values (checkboxes)
        if (Array.isArray(data[key])) {
          (data[key] as unknown[]).push(value);
        } else {
          data[key] = [data[key], value];
        }
      } else {
        data[key] = value;
      }
    });

    this.formData = data;

    // Check for next-step button action
    const activeButton = document.activeElement as HTMLButtonElement;
    if (activeButton?.dataset.action === 'next-step') {
      this.goToNextStep();
      return;
    }

    // Submit to API
    try {
      const response = await fetch(`/api/public/forms/${this.config.uuid}/submit`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: data.email,
          data,
          pageUrl: window.location.href,
          referrer: document.referrer,
        }),
      });

      if (!response.ok) {
        throw new Error('Submission failed');
      }

      const result = await response.json();

      // Update coupon code if present
      if (result.couponCode) {
        const couponEl = this.element?.querySelector('.lm-coupon-code');
        if (couponEl) {
          couponEl.innerHTML = result.couponCode;
        }
      }

      // Handle success action
      this.handleSuccess(result);
    } catch (error) {
      console.error('Form submission error:', error);
      // Show error message to user
    }
  }

  private goToNextStep(): void {
    if (this.currentStep < this.config.steps.length - 1) {
      this.currentStep++;
      const content = this.element?.querySelector('.lm-form-content');
      const oldStep = content?.querySelector('.lm-form-step');
      if (oldStep) {
        oldStep.remove();
      }
      const newStep = this.renderStep(this.config.steps[this.currentStep]);
      content?.appendChild(newStep);
    }
  }

  private handleSuccess(result: { couponCode?: string }): void {
    const settings = this.config.settings;

    switch (settings.successAction) {
      case 'message':
        this.showSuccessMessage(settings.successMessage);
        break;
      case 'redirect':
        window.location.href = settings.successRedirectUrl;
        break;
      case 'close':
        this.close();
        break;
    }
  }

  private showSuccessMessage(message: string): void {
    const content = this.element?.querySelector('.lm-form-content');
    if (content) {
      const step = content.querySelector('.lm-form-step');
      if (step) {
        step.innerHTML = `<div class="lm-success-message" style="
          text-align: center;
          padding: 40px 20px;
        ">
          <div style="font-size: 48px; margin-bottom: 16px;">✓</div>
          <div style="font-size: 18px;">${message}</div>
        </div>`;
      }
    }
  }

  close(): void {
    if (this.element) {
      this.element.remove();
      this.element = null;
    }
  }

  getElement(): HTMLElement | null {
    return this.element;
  }
}
