import type { TriggerConfig } from '../types';
export type TriggerCallback = () => void;
export declare function setupTrigger(config: TriggerConfig, callback: TriggerCallback): void;
export declare function cleanupTriggers(): void;
