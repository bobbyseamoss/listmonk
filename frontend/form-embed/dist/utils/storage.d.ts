import type { VisitorData, FormVisitorData } from '../types';
export declare function getVisitorData(): VisitorData;
export declare function saveVisitorData(data: VisitorData): void;
export declare function getFormData(formUuid: string): FormVisitorData;
export declare function recordImpression(formUuid: string): void;
export declare function recordSubmission(formUuid: string): void;
export declare function recordClose(formUuid: string): void;
export declare function incrementPageViews(): number;
export declare function shouldShowForm(formUuid: string, frequency: {
    showAgainAfterDays: number;
    suppressAfterSubmission: boolean;
    suppressDays: number;
    maxImpressions: number;
}): boolean;
