import type { TargetingRules, UtmRule } from '../types';
import { getDeviceType } from './device';

export function matchesUrlPattern(pattern: string, url: string): boolean {
  // Convert glob pattern to regex
  const regexPattern = pattern
    .replace(/[.+?^${}()|[\]\\]/g, '\\$&') // Escape special regex chars except *
    .replace(/\*/g, '.*'); // Convert * to .*

  try {
    const regex = new RegExp(`^${regexPattern}$`, 'i');
    return regex.test(url);
  } catch (e) {
    return false;
  }
}

export function matchesUtmRule(rule: UtmRule): boolean {
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

export function evaluateTargeting(rules: TargetingRules): boolean {
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

export function getUtmParams(): Record<string, string> {
  const params = new URLSearchParams(window.location.search);
  const utm: Record<string, string> = {};

  for (const key of ['utm_source', 'utm_medium', 'utm_campaign', 'utm_content', 'utm_term']) {
    const value = params.get(key);
    if (value) {
      utm[key] = value;
    }
  }

  return utm;
}
