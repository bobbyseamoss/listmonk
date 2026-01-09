import type { VisitorData, FormVisitorData } from '../types';

const STORAGE_KEY = 'lm_forms';
const VISITOR_ID_KEY = 'lm_visitor_id';

function generateId(): string {
  return 'xxxxxxxxxxxx4xxxyxxxxxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

function getSessionId(): string {
  let sessionId = sessionStorage.getItem('lm_session_id');
  if (!sessionId) {
    sessionId = generateId();
    sessionStorage.setItem('lm_session_id', sessionId);
  }
  return sessionId;
}

function getVisitorId(): string {
  let visitorId = localStorage.getItem(VISITOR_ID_KEY);
  if (!visitorId) {
    visitorId = generateId();
    localStorage.setItem(VISITOR_ID_KEY, visitorId);
  }
  return visitorId;
}

export function getVisitorData(): VisitorData {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      const data = JSON.parse(stored) as VisitorData;
      data.sessionId = getSessionId();
      return data;
    }
  } catch (e) {
    // Storage not available or corrupted
  }

  return {
    visitorId: getVisitorId(),
    sessionId: getSessionId(),
    pageViews: 0,
    forms: {},
  };
}

export function saveVisitorData(data: VisitorData): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(data));
  } catch (e) {
    // Storage not available
  }
}

export function getFormData(formUuid: string): FormVisitorData {
  const visitor = getVisitorData();
  return visitor.forms[formUuid] || {
    impressions: 0,
    lastShown: 0,
    submitted: false,
  };
}

export function recordImpression(formUuid: string): void {
  const visitor = getVisitorData();
  if (!visitor.forms[formUuid]) {
    visitor.forms[formUuid] = {
      impressions: 0,
      lastShown: 0,
      submitted: false,
    };
  }
  visitor.forms[formUuid].impressions++;
  visitor.forms[formUuid].lastShown = Date.now();
  saveVisitorData(visitor);
}

export function recordSubmission(formUuid: string): void {
  const visitor = getVisitorData();
  if (!visitor.forms[formUuid]) {
    visitor.forms[formUuid] = {
      impressions: 1,
      lastShown: Date.now(),
      submitted: false,
    };
  }
  visitor.forms[formUuid].submitted = true;
  visitor.forms[formUuid].submittedAt = Date.now();
  saveVisitorData(visitor);
}

export function recordClose(formUuid: string): void {
  const visitor = getVisitorData();
  if (!visitor.forms[formUuid]) {
    visitor.forms[formUuid] = {
      impressions: 1,
      lastShown: Date.now(),
      submitted: false,
    };
  }
  visitor.forms[formUuid].closedAt = Date.now();
  saveVisitorData(visitor);
}

export function incrementPageViews(): number {
  const visitor = getVisitorData();
  visitor.pageViews++;
  saveVisitorData(visitor);
  return visitor.pageViews;
}

export function shouldShowForm(formUuid: string, frequency: {
  showAgainAfterDays: number;
  suppressAfterSubmission: boolean;
  suppressDays: number;
  maxImpressions: number;
}): boolean {
  const formData = getFormData(formUuid);
  const now = Date.now();
  const dayMs = 24 * 60 * 60 * 1000;

  // Check if submitted and should be suppressed
  if (formData.submitted) {
    if (frequency.suppressAfterSubmission) {
      return false;
    }
    if (formData.submittedAt) {
      const daysSinceSubmission = (now - formData.submittedAt) / dayMs;
      if (daysSinceSubmission < frequency.suppressDays) {
        return false;
      }
    }
  }

  // Check max impressions
  if (frequency.maxImpressions > 0 && formData.impressions >= frequency.maxImpressions) {
    return false;
  }

  // Check show again after days
  if (formData.closedAt) {
    const daysSinceClosed = (now - formData.closedAt) / dayMs;
    if (daysSinceClosed < frequency.showAgainAfterDays) {
      return false;
    }
  }

  return true;
}
