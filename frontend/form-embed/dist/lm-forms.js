var Listmonk = (function (exports) {
    'use strict';

    class FormRenderer {
        constructor(config, baseUrl = '') {
            this.currentStep = 0;
            this.formData = {};
            this.element = null;
            this.config = config;
            this.baseUrl = baseUrl;
        }
        render(targetSelector) {
            var _a;
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
            (_a = wrapper.querySelector('.lm-form-content')) === null || _a === void 0 ? void 0 : _a.appendChild(stepContent);
            this.element = container;
            // For embed forms, insert into target
            if (this.config.formType === 'embed' && targetSelector) {
                const target = document.querySelector(targetSelector);
                if (target) {
                    target.appendChild(container);
                    // Dispatch 'open' event for GTM/analytics integration
                    this.dispatchFormEvent('open');
                    return container;
                }
            }
            // For other types, add to body
            document.body.appendChild(container);
            // Dispatch 'open' event for GTM/analytics integration
            this.dispatchFormEvent('open');
            return container;
        }
        createWrapper() {
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
        getContentStyles() {
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
                    }
                    else {
                        styles += 'left: 20px;';
                    }
                    if (position.includes('top')) {
                        styles += 'top: 20px;';
                    }
                    else {
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
                    }
                    else {
                        styles += 'bottom: 0;';
                    }
                    break;
            }
            return styles;
        }
        renderStep(step) {
            const container = document.createElement('div');
            container.className = 'lm-form-step';
            container.setAttribute('data-step-id', step.id);
            // Create form element
            const form = document.createElement('form');
            form.className = 'lm-form';
            // Use flexbox to allow side-by-side blocks with width < 100%
            // align-items: flex-end ensures button aligns with input (not label)
            form.style.cssText = 'display: flex; flex-wrap: wrap; align-items: flex-end;';
            form.addEventListener('submit', (e) => this.handleSubmit(e));
            // Render blocks
            step.blocks.forEach((block) => {
                const blockEl = this.renderBlock(block);
                form.appendChild(blockEl);
            });
            container.appendChild(form);
            return container;
        }
        renderBlock(block) {
            const container = document.createElement('div');
            container.className = `lm-block lm-block-${block.type}`;
            container.setAttribute('data-block-id', block.id);
            const props = block.props;
            // Apply width if specified (for side-by-side layout)
            // Note: For image blocks, props.width is the IMAGE width, not the block width
            // Image blocks should always be full width so the image can be centered within
            if (block.type === 'image') {
                container.style.width = '100%';
            }
            else {
                const blockWidth = props.width;
                if (blockWidth && blockWidth !== '100%') {
                    container.style.width = blockWidth;
                    container.style.boxSizing = 'border-box';
                }
                else {
                    container.style.width = '100%';
                }
            }
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
        renderTextBlock(props) {
            const padding = props.padding;
            return `<div style="
      font-size: ${props.fontSize}px;
      font-weight: ${props.fontWeight};
      color: ${props.color};
      text-align: ${props.textAlign};
      padding: ${padding.top}px ${padding.right}px ${padding.bottom}px ${padding.left}px;
    ">${props.content}</div>`;
        }
        renderHeadingBlock(props) {
            const level = props.level || 2;
            const padding = props.padding;
            return `<h${level} style="
      font-size: ${props.fontSize}px;
      font-weight: ${props.fontWeight};
      color: ${props.color};
      text-align: ${props.textAlign};
      padding: ${padding.top}px ${padding.right}px ${padding.bottom}px ${padding.left}px;
      margin: 0;
    ">${props.content}</h${level}>`;
        }
        renderImageBlock(props) {
            const padding = props.padding;
            if (!props.src) {
                return '';
            }
            // Convert numeric width to px string (form builder stores width as number, e.g., 200)
            const imageWidth = typeof props.width === 'number' ? `${props.width}px` : (props.width || 'auto');
            const imageHeight = props.height || 'auto';
            const alignment = props.alignment || 'center';
            return `<div style="text-align: ${alignment}; padding: ${padding.top}px ${padding.right}px ${padding.bottom}px ${padding.left}px;">
      <img src="${props.src}" alt="${props.alt || ''}" style="width: ${imageWidth}; height: ${imageHeight}; max-width: 100%;" />
    </div>`;
        }
        renderEmailInputBlock(props) {
            const padding = props.padding;
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
        renderTextInputBlock(props) {
            const padding = props.padding;
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
        renderTextareaBlock(props) {
            const padding = props.padding;
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
        renderRadioGroupBlock(props) {
            const padding = props.padding;
            const options = props.options;
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
        renderCheckboxGroupBlock(props) {
            const padding = props.padding;
            const options = props.options;
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
        renderDropdownBlock(props) {
            const padding = props.padding;
            const options = props.options;
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
        renderDatePickerBlock(props) {
            const padding = props.padding;
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
        renderHiddenFieldBlock(props) {
            let value = props.value;
            // Dynamic value based on source
            if (props.source === 'url') {
                const params = new URLSearchParams(window.location.search);
                value = params.get(props.paramName) || '';
            }
            else if (props.source === 'cookie') {
                const match = document.cookie.match(new RegExp(`(^| )${props.paramName}=([^;]+)`));
                value = match ? match[2] : '';
            }
            return `<input type="hidden" name="${props.name}" value="${value}" />`;
        }
        renderButtonBlock(props) {
            const padding = props.padding;
            const margin = props.margin;
            const action = props.action;
            let type = 'button';
            if (action === 'submit') {
                type = 'submit';
            }
            return `<div style="padding: 8px; margin: ${margin.top}px ${margin.right}px ${margin.bottom}px ${margin.left}px;">
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
        renderDividerBlock(props) {
            const margin = props.margin;
            return `<hr style="
      margin: ${margin.top}px 0 ${margin.bottom}px 0;
      border: none;
      border-top: ${props.thickness}px ${props.style} ${props.color};
    " />`;
        }
        renderSpacerBlock(props) {
            return `<div style="height: ${props.height}px;"></div>`;
        }
        renderCouponBlock(props) {
            const padding = props.padding;
            return `<div class="lm-coupon-block" style="
      padding: ${padding.top}px ${padding.right}px ${padding.bottom}px ${padding.left}px;
      background-color: ${props.bgColor};
      border: 1px solid ${props.borderColor};
      border-radius: ${props.borderRadius}px;
      text-align: center;
    ">
      <div style="font-size: ${props.fontSize - 2}px; color: ${props.textColor}; margin-bottom: 4px;">
        ${props.title}
      </div>
      <div class="lm-coupon-code" style="
        font-size: ${props.fontSize + 4}px;
        font-weight: ${props.fontWeight};
        color: ${props.textColor};
        letter-spacing: 2px;
      ">
        <!-- Coupon code will be filled after submission -->
        <span class="lm-coupon-placeholder">Submit to reveal</span>
      </div>
    </div>`;
        }
        renderCountdownBlock(props) {
            const padding = props.padding;
            const endDate = new Date(props.endDate).getTime();
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
      ${props.showDays ? `<div style="text-align: center;"><div class="days" style="font-size: ${props.fontSize}px; font-weight: bold; color: ${props.textColor};">00</div><div style="font-size: ${props.fontSize * 0.5}px; color: ${props.textColor};">Days</div></div>` : ''}
      ${props.showHours ? `<div style="text-align: center;"><div class="hours" style="font-size: ${props.fontSize}px; font-weight: bold; color: ${props.textColor};">00</div><div style="font-size: ${props.fontSize * 0.5}px; color: ${props.textColor};">Hours</div></div>` : ''}
      ${props.showMinutes ? `<div style="text-align: center;"><div class="minutes" style="font-size: ${props.fontSize}px; font-weight: bold; color: ${props.textColor};">00</div><div style="font-size: ${props.fontSize * 0.5}px; color: ${props.textColor};">Min</div></div>` : ''}
      ${props.showSeconds ? `<div style="text-align: center;"><div class="seconds" style="font-size: ${props.fontSize}px; font-weight: bold; color: ${props.textColor};">00</div><div style="font-size: ${props.fontSize * 0.5}px; color: ${props.textColor};">Sec</div></div>` : ''}
    </div>`;
        }
        startCountdown(id, endDate, props) {
            const el = document.getElementById(id);
            if (!el)
                return;
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
                if (daysEl)
                    daysEl.textContent = String(days).padStart(2, '0');
                if (hoursEl)
                    hoursEl.textContent = String(hours).padStart(2, '0');
                if (minutesEl)
                    minutesEl.textContent = String(minutes).padStart(2, '0');
                if (secondsEl)
                    secondsEl.textContent = String(seconds).padStart(2, '0');
                if (diff > 0) {
                    requestAnimationFrame(update);
                }
            };
            update();
        }
        renderSignupCounterBlock(props) {
            const padding = props.padding;
            const text = props.text.replace('{count}', `<span style="color: ${props.countColor}; font-weight: bold;">${props.count.toLocaleString()}</span>`);
            return `<div style="
      padding: ${padding.top}px ${padding.right}px ${padding.bottom}px ${padding.left}px;
      font-size: ${props.fontSize}px;
      color: ${props.textColor};
      text-align: center;
    ">${text}</div>`;
        }
        async handleSubmit(e) {
            var _a;
            e.preventDefault();
            const form = e.target;
            const formData = new FormData(form);
            const data = {};
            formData.forEach((value, key) => {
                if (data[key]) {
                    // Handle multiple values (checkboxes)
                    if (Array.isArray(data[key])) {
                        data[key].push(value);
                    }
                    else {
                        data[key] = [data[key], value];
                    }
                }
                else {
                    data[key] = value;
                }
            });
            this.formData = data;
            // Check for next-step button action
            const activeButton = document.activeElement;
            if ((activeButton === null || activeButton === void 0 ? void 0 : activeButton.dataset.action) === 'next-step') {
                // Dispatch stepSubmit event for multi-step forms
                this.dispatchFormEvent('stepSubmit', data, this.currentStep);
                this.goToNextStep();
                return;
            }
            // Submit to API
            try {
                const response = await fetch(`${this.baseUrl}/api/public/forms/${this.config.uuid}/submit`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        email: data.email,
                        data,
                        page_url: window.location.href,
                        referrer: document.referrer,
                    }),
                });
                if (!response.ok) {
                    throw new Error('Submission failed');
                }
                const result = await response.json();
                // Store subscriber UUID in cookie for profile lookup API (365 days)
                if (result.subscriber_uuid) {
                    const maxAge = 365 * 24 * 60 * 60; // 365 days in seconds
                    document.cookie = `lm_subscriber=${result.subscriber_uuid}; path=/; max-age=${maxAge}; SameSite=Lax`;
                }
                // Update coupon code if present
                if (result.couponCode) {
                    const couponEl = (_a = this.element) === null || _a === void 0 ? void 0 : _a.querySelector('.lm-coupon-code');
                    if (couponEl) {
                        couponEl.innerHTML = result.couponCode;
                    }
                }
                // Dispatch submit event for GTM/analytics integration (like Klaviyo's klaviyoForms event)
                this.dispatchFormEvent('submit', data, this.currentStep, result);
                // Handle success action
                this.handleSuccess(result);
            }
            catch (error) {
                console.error('Form submission error:', error);
                // Show error message to user
            }
        }
        /**
         * Dispatch a custom window event for GTM/analytics integration.
         * Similar to Klaviyo's 'klaviyoForms' event pattern.
         *
         * Event types:
         * - 'open': Form was displayed to user
         * - 'submit': Form was submitted successfully
         * - 'stepSubmit': A step in a multi-step form was submitted
         * - 'close': Form was closed
         *
         * Usage in GTM:
         * window.addEventListener('listmonkForms', function(e) {
         *   if (e.detail.type === 'submit') {
         *     dataLayer.push({ event: 'listmonk_form_submit', formData: e.detail });
         *   }
         * });
         */
        dispatchFormEvent(type, data, step, result) {
            const eventDetail = {
                type,
                formUUID: this.config.uuid,
                formName: this.config.name,
                formType: this.config.formType,
                subscriberUUID: result === null || result === void 0 ? void 0 : result.subscriber_uuid,
                metaData: {
                    $email: data === null || data === void 0 ? void 0 : data.email,
                    $first_name: (data === null || data === void 0 ? void 0 : data.first_name) || (data === null || data === void 0 ? void 0 : data.firstName) || (data === null || data === void 0 ? void 0 : data.name),
                    $last_name: (data === null || data === void 0 ? void 0 : data.last_name) || (data === null || data === void 0 ? void 0 : data.lastName),
                    ...data,
                },
                step,
                couponCode: result === null || result === void 0 ? void 0 : result.couponCode,
                pageUrl: window.location.href,
                referrer: document.referrer,
                timestamp: new Date().toISOString(),
            };
            const event = new CustomEvent('listmonkForms', {
                detail: eventDetail,
                bubbles: true,
                cancelable: false,
            });
            window.dispatchEvent(event);
        }
        goToNextStep() {
            var _a;
            if (this.currentStep < this.config.steps.length - 1) {
                this.currentStep++;
                const content = (_a = this.element) === null || _a === void 0 ? void 0 : _a.querySelector('.lm-form-content');
                const oldStep = content === null || content === void 0 ? void 0 : content.querySelector('.lm-form-step');
                if (oldStep) {
                    oldStep.remove();
                }
                const newStep = this.renderStep(this.config.steps[this.currentStep]);
                content === null || content === void 0 ? void 0 : content.appendChild(newStep);
            }
        }
        handleSuccess(result) {
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
        showSuccessMessage(message) {
            var _a;
            const content = (_a = this.element) === null || _a === void 0 ? void 0 : _a.querySelector('.lm-form-content');
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
        close() {
            if (this.element) {
                // Dispatch 'close' event for GTM/analytics integration
                this.dispatchFormEvent('close');
                this.element.remove();
                this.element = null;
            }
        }
        getElement() {
            return this.element;
        }
    }

    function getDeviceType() {
        const ua = navigator.userAgent.toLowerCase();
        // Check for tablets first (iPads, Android tablets)
        if (/(ipad|tablet|(android(?!.*mobile))|(windows(?!.*phone)(.*touch))|kindle|playbook|silk|(puffin(?!.*(IP|AP|WP))))/.test(ua)) {
            return 'tablet';
        }
        // Check for mobile devices
        if (/(mobi|ipod|phone|blackberry|opera mini|fennec|minimo|symbian|psp|nintendo ds|archos|skyfire|puffin|blazer|bolt|gobrowser|iris|maemo|semc|teashark|uzard)/.test(ua)) {
            return 'mobile';
        }
        // Check screen width as fallback
        if (window.innerWidth < 768) {
            return 'mobile';
        }
        if (window.innerWidth < 1024) {
            return 'tablet';
        }
        return 'desktop';
    }
    function isTouchDevice() {
        return 'ontouchstart' in window || navigator.maxTouchPoints > 0;
    }

    function matchesUrlPattern(pattern, url) {
        // Convert glob pattern to regex
        const regexPattern = pattern
            .replace(/[.+?^${}()|[\]\\]/g, '\\$&') // Escape special regex chars except *
            .replace(/\*/g, '.*'); // Convert * to .*
        try {
            const regex = new RegExp(`^${regexPattern}$`, 'i');
            return regex.test(url);
        }
        catch (e) {
            return false;
        }
    }
    function matchesUtmRule(rule) {
        const params = new URLSearchParams(window.location.search);
        const value = params.get(rule.key);
        if (!value) {
            return false;
        }
        switch (rule.operator) {
            case 'equals':
                return value.toLowerCase() === rule.value.toLowerCase();
            case 'contains':
                return value.toLowerCase().includes(rule.value.toLowerCase());
            case 'starts_with':
                return value.toLowerCase().startsWith(rule.value.toLowerCase());
            default:
                return false;
        }
    }
    function evaluateTargeting(rules) {
        const currentUrl = window.location.pathname + window.location.search;
        // Check URL patterns (if specified, at least one must match)
        if (rules.urlPatterns.length > 0) {
            const matchesUrl = rules.urlPatterns.some((pattern) => matchesUrlPattern(pattern, currentUrl));
            if (!matchesUrl) {
                return false;
            }
        }
        // Check exclude URL patterns (if any match, don't show)
        if (rules.excludeUrlPatterns.length > 0) {
            const matchesExclude = rules.excludeUrlPatterns.some((pattern) => matchesUrlPattern(pattern, currentUrl));
            if (matchesExclude) {
                return false;
            }
        }
        // Check UTM parameters (all must match if specified)
        if (rules.utmParams.length > 0) {
            const matchesAllUtm = rules.utmParams.every((rule) => matchesUtmRule(rule));
            if (!matchesAllUtm) {
                return false;
            }
        }
        // Check device type
        if (rules.deviceTypes.length > 0) {
            const currentDevice = getDeviceType();
            if (!rules.deviceTypes.includes(currentDevice)) {
                return false;
            }
        }
        // Country targeting would require IP geolocation (handled server-side)
        // For now, skip client-side country checks
        return true;
    }

    let exitIntentHandler = null;
    let scrollHandler = null;
    let scrollUpHandler = null;
    let timeoutId = null;
    function setupTrigger(config, callback) {
        cleanupTriggers();
        switch (config.type) {
            case 'immediate':
                callback();
                break;
            case 'time':
                timeoutId = setTimeout(callback, config.delay * 1000);
                break;
            case 'scroll':
                setupScrollTrigger(config.scrollDepth, callback);
                break;
            case 'pages':
                // Page count is handled by storage, already evaluated before this
                callback();
                break;
        }
        // Setup exit intent if enabled
        if (config.exitIntentEnabled) {
            setupExitIntent(callback);
        }
    }
    function setupScrollTrigger(depth, callback) {
        let triggered = false;
        scrollHandler = () => {
            if (triggered)
                return;
            const scrollTop = window.scrollY || document.documentElement.scrollTop;
            const scrollHeight = document.documentElement.scrollHeight - window.innerHeight;
            const scrollPercent = (scrollTop / scrollHeight) * 100;
            if (scrollPercent >= depth) {
                triggered = true;
                callback();
            }
        };
        window.addEventListener('scroll', scrollHandler, { passive: true });
    }
    function setupExitIntent(callback) {
        if (isTouchDevice()) {
            // For mobile, detect quick scroll up
            setupMobileExitIntent(callback);
            return;
        }
        // For desktop, detect mouse leaving viewport
        let triggered = false;
        exitIntentHandler = (e) => {
            if (triggered)
                return;
            // Check if mouse is leaving through top of viewport
            if (e.clientY <= 0) {
                triggered = true;
                callback();
            }
        };
        document.addEventListener('mouseout', exitIntentHandler);
    }
    function setupMobileExitIntent(callback) {
        let lastScrollTop = 0;
        let scrollUpDistance = 0;
        let triggered = false;
        scrollUpHandler = () => {
            if (triggered)
                return;
            const scrollTop = window.scrollY || document.documentElement.scrollTop;
            if (scrollTop < lastScrollTop) {
                // Scrolling up
                scrollUpDistance += lastScrollTop - scrollTop;
                // Trigger if user scrolls up more than 100px quickly
                if (scrollUpDistance > 100) {
                    triggered = true;
                    callback();
                }
            }
            else {
                // Reset on scroll down
                scrollUpDistance = 0;
            }
            lastScrollTop = scrollTop;
        };
        window.addEventListener('scroll', scrollUpHandler, { passive: true });
    }
    function cleanupTriggers() {
        if (exitIntentHandler) {
            document.removeEventListener('mouseout', exitIntentHandler);
            exitIntentHandler = null;
        }
        if (scrollHandler) {
            window.removeEventListener('scroll', scrollHandler);
            scrollHandler = null;
        }
        if (scrollUpHandler) {
            window.removeEventListener('scroll', scrollUpHandler);
            scrollUpHandler = null;
        }
        if (timeoutId) {
            clearTimeout(timeoutId);
            timeoutId = null;
        }
    }

    const STORAGE_KEY = 'lm_forms';
    const VISITOR_ID_KEY = 'lm_visitor_id';
    function generateId() {
        return 'xxxxxxxxxxxx4xxxyxxxxxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
            const r = (Math.random() * 16) | 0;
            const v = c === 'x' ? r : (r & 0x3) | 0x8;
            return v.toString(16);
        });
    }
    function getSessionId() {
        let sessionId = sessionStorage.getItem('lm_session_id');
        if (!sessionId) {
            sessionId = generateId();
            sessionStorage.setItem('lm_session_id', sessionId);
        }
        return sessionId;
    }
    function getVisitorId() {
        let visitorId = localStorage.getItem(VISITOR_ID_KEY);
        if (!visitorId) {
            visitorId = generateId();
            localStorage.setItem(VISITOR_ID_KEY, visitorId);
        }
        return visitorId;
    }
    function getVisitorData() {
        try {
            const stored = localStorage.getItem(STORAGE_KEY);
            if (stored) {
                const data = JSON.parse(stored);
                data.sessionId = getSessionId();
                return data;
            }
        }
        catch (e) {
            // Storage not available or corrupted
        }
        return {
            visitorId: getVisitorId(),
            sessionId: getSessionId(),
            pageViews: 0,
            forms: {},
        };
    }
    function saveVisitorData(data) {
        try {
            localStorage.setItem(STORAGE_KEY, JSON.stringify(data));
        }
        catch (e) {
            // Storage not available
        }
    }
    function getFormData(formUuid) {
        const visitor = getVisitorData();
        return visitor.forms[formUuid] || {
            impressions: 0,
            lastShown: 0,
            submitted: false,
        };
    }
    function recordImpression(formUuid) {
        const visitor = getVisitorData();
        if (!visitor.forms[formUuid]) {
            visitor.forms[formUuid] = {
                impressions: 0,
                lastShown: 0,
                submitted: false,
            };
        }
        visitor.forms[formUuid].impressions++;
        visitor.forms[formUuid].lastShown = Date.now();
        saveVisitorData(visitor);
    }
    function recordSubmission(formUuid) {
        const visitor = getVisitorData();
        if (!visitor.forms[formUuid]) {
            visitor.forms[formUuid] = {
                impressions: 1,
                lastShown: Date.now(),
                submitted: false,
            };
        }
        visitor.forms[formUuid].submitted = true;
        visitor.forms[formUuid].submittedAt = Date.now();
        saveVisitorData(visitor);
    }
    function recordClose(formUuid) {
        const visitor = getVisitorData();
        if (!visitor.forms[formUuid]) {
            visitor.forms[formUuid] = {
                impressions: 1,
                lastShown: Date.now(),
                submitted: false,
            };
        }
        visitor.forms[formUuid].closedAt = Date.now();
        saveVisitorData(visitor);
    }
    function incrementPageViews() {
        const visitor = getVisitorData();
        visitor.pageViews++;
        saveVisitorData(visitor);
        return visitor.pageViews;
    }
    function shouldShowForm(formUuid, frequency) {
        const formData = getFormData(formUuid);
        const now = Date.now();
        const dayMs = 24 * 60 * 60 * 1000;
        // Check if submitted and should be suppressed
        if (formData.submitted) {
            if (frequency.suppressAfterSubmission) {
                return false;
            }
            if (formData.submittedAt) {
                const daysSinceSubmission = (now - formData.submittedAt) / dayMs;
                if (daysSinceSubmission < frequency.suppressDays) {
                    return false;
                }
            }
        }
        // Check max impressions
        if (frequency.maxImpressions > 0 && formData.impressions >= frequency.maxImpressions) {
            return false;
        }
        // Check show again after days
        if (formData.closedAt) {
            const daysSinceClosed = (now - formData.closedAt) / dayMs;
            if (daysSinceClosed < frequency.showAgainAfterDays) {
                return false;
            }
        }
        return true;
    }

    class FormController {
        constructor(baseUrl = '') {
            this.activeForms = new Map();
            this.loadedConfigs = new Map();
            this.currentFormPriority = 0;
            this.baseUrl = baseUrl || this.detectBaseUrl();
            this.init();
        }
        detectBaseUrl() {
            // Try to detect base URL from script tag
            const scripts = document.querySelectorAll('script[src*="lm-forms"]');
            if (scripts.length > 0) {
                const src = scripts[scripts.length - 1].src;
                const url = new URL(src);
                return url.origin;
            }
            return window.location.origin;
        }
        init() {
            // Increment page views
            incrementPageViews();
            // Auto-load forms from script tags
            this.autoLoadFromScripts();
            // Auto-discover active forms from API (for popups, flyouts, etc.)
            this.autoDiscoverForms();
            // Add global styles
            this.addGlobalStyles();
        }
        autoLoadFromScripts() {
            const scripts = document.querySelectorAll('script[data-form]');
            scripts.forEach((script) => {
                var _a;
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
                            (_a = script.parentNode) === null || _a === void 0 ? void 0 : _a.insertBefore(container, script);
                        }
                        target = `#${containerId}`;
                    }
                    this.load(formUuid, {
                        trigger: formType === 'embed' ? 'manual' : 'auto',
                        target: target || undefined,
                        scriptElement: script,
                        formTypeOverride: formType,
                    });
                }
            });
        }
        async autoDiscoverForms() {
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
                const forms = result.data;
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
            }
            catch (error) {
                console.error('Error auto-discovering forms:', error);
            }
        }
        detectShopifyPageType() {
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
        addGlobalStyles() {
            if (document.getElementById('lm-forms-styles'))
                return;
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
        async load(formUuid, options = {}) {
            var _a;
            try {
                // Fetch form config from API
                const response = await fetch(`${this.baseUrl}/api/public/forms/${formUuid}`);
                if (!response.ok) {
                    console.error(`Failed to load form ${formUuid}: ${response.status}`);
                    return;
                }
                const config = (await response.json()).data;
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
                    const activePopupForms = Array.from(this.activeForms.values()).filter((f) => f.config.formType !== 'embed');
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
                            (_a = options.scriptElement.parentNode) === null || _a === void 0 ? void 0 : _a.insertBefore(container, options.scriptElement);
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
            }
            catch (error) {
                console.error(`Error loading form ${formUuid}:`, error);
            }
        }
        show(formUuid, options = {}) {
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
        setupCloseHandlers(formUuid, element, options) {
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
            const escHandler = (e) => {
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
                        const data = {};
                        formData.forEach((value, key) => {
                            data[key] = value;
                        });
                        options.onSubmit(data);
                    }
                });
            }
        }
        close(formUuid) {
            const active = this.activeForms.get(formUuid);
            if (active) {
                active.renderer.close();
                this.activeForms.delete(formUuid);
                recordClose(formUuid);
                cleanupTriggers();
                // Update current priority (only from non-embed forms)
                const activePopupForms = Array.from(this.activeForms.values()).filter((f) => f.config.formType !== 'embed');
                if (activePopupForms.length === 0) {
                    this.currentFormPriority = 0;
                }
                else {
                    this.currentFormPriority = Math.max(...activePopupForms.map((f) => f.config.frequency.priority));
                }
            }
        }
        closeAll() {
            for (const uuid of this.activeForms.keys()) {
                this.close(uuid);
            }
        }
        async trackImpression(formUuid) {
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
            }
            catch (error) {
                // Ignore tracking errors
            }
        }
        isFormActive(formUuid) {
            return this.activeForms.has(formUuid);
        }
        getActiveFormIds() {
            return Array.from(this.activeForms.keys());
        }
    }

    // Global instance
    let formController = null;
    // Initialize on load
    function init() {
        if (formController)
            return;
        formController = new FormController();
    }
    // Auto-init when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    }
    else {
        init();
    }
    // Public API
    const forms = {
        /**
         * Load a form configuration and set up its triggers
         * @param formUuid - The UUID of the form to load
         * @param options - Show options
         */
        load(formUuid, options) {
            init();
            return formController.load(formUuid, options);
        },
        /**
         * Manually show a form
         * @param formUuid - The UUID of the form to show
         * @param options - Show options
         */
        show(formUuid, options) {
            init();
            // If form not loaded, load and show
            formController.load(formUuid, { ...options, trigger: 'manual' });
        },
        /**
         * Close a specific form
         * @param formUuid - The UUID of the form to close
         */
        close(formUuid) {
            formController === null || formController === void 0 ? void 0 : formController.close(formUuid);
        },
        /**
         * Close all active forms
         */
        closeAll() {
            formController === null || formController === void 0 ? void 0 : formController.closeAll();
        },
        /**
         * Check if a form is currently displayed
         * @param formUuid - The UUID of the form to check
         */
        isActive(formUuid) {
            var _a;
            return (_a = formController === null || formController === void 0 ? void 0 : formController.isFormActive(formUuid)) !== null && _a !== void 0 ? _a : false;
        },
        /**
         * Get list of currently active form UUIDs
         */
        getActive() {
            var _a;
            return (_a = formController === null || formController === void 0 ? void 0 : formController.getActiveFormIds()) !== null && _a !== void 0 ? _a : [];
        },
    };
    // Attach to window for global access
    if (typeof window !== 'undefined') {
        window.Frigid = { forms };
    }

    exports.FormController = FormController;
    exports.FormRenderer = FormRenderer;
    exports.forms = forms;

    return exports;

})({});
//# sourceMappingURL=lm-forms.js.map
