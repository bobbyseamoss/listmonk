import type { FormConfig } from './types';
export declare class FormRenderer {
    private config;
    private baseUrl;
    private currentStep;
    private formData;
    private element;
    constructor(config: FormConfig, baseUrl?: string);
    render(targetSelector?: string): HTMLElement;
    private createWrapper;
    private getContentStyles;
    private renderStep;
    private renderBlock;
    private renderTextBlock;
    private renderHeadingBlock;
    private renderImageBlock;
    private renderEmailInputBlock;
    private renderTextInputBlock;
    private renderTextareaBlock;
    private renderRadioGroupBlock;
    private renderCheckboxGroupBlock;
    private renderDropdownBlock;
    private renderDatePickerBlock;
    private renderHiddenFieldBlock;
    private renderButtonBlock;
    private renderDividerBlock;
    private renderSpacerBlock;
    private renderCouponBlock;
    private renderCountdownBlock;
    private startCountdown;
    private renderSignupCounterBlock;
    private handleSubmit;
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
    dispatchFormEvent(type: 'open' | 'submit' | 'stepSubmit' | 'close', data?: Record<string, unknown>, step?: number, result?: {
        couponCode?: string;
        subscriber_uuid?: string;
    }): void;
    private goToNextStep;
    private handleSuccess;
    private showSuccessMessage;
    close(): void;
    getElement(): HTMLElement | null;
}
