import type { TriggerConfig } from '../types';
import { isTouchDevice } from './device';

export type TriggerCallback = () => void;

let exitIntentHandler: ((e: MouseEvent) => void) | null = null;
let scrollHandler: (() => void) | null = null;
let scrollUpHandler: (() => void) | null = null;
let timeoutId: ReturnType<typeof setTimeout> | null = null;

export function setupTrigger(config: TriggerConfig, callback: TriggerCallback): void {
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

    case 'manual':
      // Don't set up any automatic trigger
      break;
  }

  // Setup exit intent if enabled
  if (config.exitIntentEnabled) {
    setupExitIntent(callback);
  }
}

function setupScrollTrigger(depth: number, callback: TriggerCallback): void {
  let triggered = false;

  scrollHandler = () => {
    if (triggered) return;

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

function setupExitIntent(callback: TriggerCallback): void {
  if (isTouchDevice()) {
    // For mobile, detect quick scroll up
    setupMobileExitIntent(callback);
    return;
  }

  // For desktop, detect mouse leaving viewport
  let triggered = false;

  exitIntentHandler = (e: MouseEvent) => {
    if (triggered) return;

    // Check if mouse is leaving through top of viewport
    if (e.clientY <= 0) {
      triggered = true;
      callback();
    }
  };

  document.addEventListener('mouseout', exitIntentHandler);
}

function setupMobileExitIntent(callback: TriggerCallback): void {
  let lastScrollTop = 0;
  let scrollUpDistance = 0;
  let triggered = false;

  scrollUpHandler = () => {
    if (triggered) return;

    const scrollTop = window.scrollY || document.documentElement.scrollTop;

    if (scrollTop < lastScrollTop) {
      // Scrolling up
      scrollUpDistance += lastScrollTop - scrollTop;

      // Trigger if user scrolls up more than 100px quickly
      if (scrollUpDistance > 100) {
        triggered = true;
        callback();
      }
    } else {
      // Reset on scroll down
      scrollUpDistance = 0;
    }

    lastScrollTop = scrollTop;
  };

  window.addEventListener('scroll', scrollUpHandler, { passive: true });
}

export function cleanupTriggers(): void {
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
