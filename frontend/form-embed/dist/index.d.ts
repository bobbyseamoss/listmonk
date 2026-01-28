import type { ShowOptions } from './types';
export declare const forms: {
    /**
     * Load a form configuration and set up its triggers
     * @param formUuid - The UUID of the form to load
     * @param options - Show options
     */
    load(formUuid: string, options?: ShowOptions): Promise<void>;
    /**
     * Manually show a form
     * @param formUuid - The UUID of the form to show
     * @param options - Show options
     */
    show(formUuid: string, options?: ShowOptions): void;
    /**
     * Close a specific form
     * @param formUuid - The UUID of the form to close
     */
    close(formUuid: string): void;
    /**
     * Close all active forms
     */
    closeAll(): void;
    /**
     * Check if a form is currently displayed
     * @param formUuid - The UUID of the form to check
     */
    isActive(formUuid: string): boolean;
    /**
     * Get list of currently active form UUIDs
     */
    getActive(): string[];
};
export { FormController } from './FormController';
export { FormRenderer } from './FormRenderer';
export type { FormConfig, ShowOptions, FormBlock, FormStep } from './types';
