import type { ShowOptions } from './types';
export declare class FormController {
    private baseUrl;
    private activeForms;
    private loadedConfigs;
    private currentFormPriority;
    constructor(baseUrl?: string);
    private detectBaseUrl;
    private init;
    private autoLoadFromScripts;
    private autoDiscoverForms;
    private detectShopifyPageType;
    private addGlobalStyles;
    load(formUuid: string, options?: ShowOptions): Promise<void>;
    show(formUuid: string, options?: ShowOptions): void;
    private setupCloseHandlers;
    close(formUuid: string): void;
    closeAll(): void;
    private trackImpression;
    isFormActive(formUuid: string): boolean;
    getActiveFormIds(): string[];
}
