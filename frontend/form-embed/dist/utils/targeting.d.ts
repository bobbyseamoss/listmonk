import type { TargetingRules, UtmRule } from '../types';
export declare function matchesUrlPattern(pattern: string, url: string): boolean;
export declare function matchesUtmRule(rule: UtmRule): boolean;
export declare function evaluateTargeting(rules: TargetingRules): boolean;
export declare function getUtmParams(): Record<string, string>;
