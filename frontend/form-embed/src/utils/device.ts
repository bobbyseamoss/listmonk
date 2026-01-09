export type DeviceType = 'desktop' | 'tablet' | 'mobile';

export function getDeviceType(): DeviceType {
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

export function isTouchDevice(): boolean {
  return 'ontouchstart' in window || navigator.maxTouchPoints > 0;
}
