import { FormController } from './FormController';
import type { ShowOptions } from './types';

// Global instance
let formController: FormController | null = null;

// Initialize on load
function init(): void {
  if (formController) return;
  formController = new FormController();
}

// Auto-init when DOM is ready
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}

// Public API
export const forms = {
  /**
   * Load a form configuration and set up its triggers
   * @param formUuid - The UUID of the form to load
   * @param options - Show options
   */
  load(formUuid: string, options?: ShowOptions): Promise<void> {
    init();
    return formController!.load(formUuid, options);
  },

  /**
   * Manually show a form
   * @param formUuid - The UUID of the form to show
   * @param options - Show options
   */
  show(formUuid: string, options?: ShowOptions): void {
    init();
    // If form not loaded, load and show
    formController!.load(formUuid, { ...options, trigger: 'manual' });
  },

  /**
   * Close a specific form
   * @param formUuid - The UUID of the form to close
   */
  close(formUuid: string): void {
    formController?.close(formUuid);
  },

  /**
   * Close all active forms
   */
  closeAll(): void {
    formController?.closeAll();
  },

  /**
   * Check if a form is currently displayed
   * @param formUuid - The UUID of the form to check
   */
  isActive(formUuid: string): boolean {
    return formController?.isFormActive(formUuid) ?? false;
  },

  /**
   * Get list of currently active form UUIDs
   */
  getActive(): string[] {
    return formController?.getActiveFormIds() ?? [];
  },
};

// Export for module usage
export { FormController } from './FormController';
export { FormRenderer } from './FormRenderer';
export type { FormConfig, ShowOptions, FormBlock, FormStep } from './types';

// Attach to window for global access
if (typeof window !== 'undefined') {
  (window as Window & { Listmonk?: { forms: typeof forms } }).Listmonk = { forms };
}
