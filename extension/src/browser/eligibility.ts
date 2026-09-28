import {
  tabIdentityKey,
  type ContextKind,
  type EligibleTabRecord,
  type ProfileID,
  type ProjectionEpoch,
  type ProjectionRevision,
  type TabIdentity,
} from '../../domain/index.js';
import {
  type BrowserGroupSnapshot,
  type BrowserTabSnapshot,
  type BrowserWindowSnapshot,
} from './event-normalizer.js';

export type EligibilityReason =
  | 'eligible'
  | 'identity_invalid'
  | 'context_unknown'
  | 'context_mismatch'
  | 'window_unavailable'
  | 'required_field_missing'
  | 'invalid_url'
  | 'unsupported_scheme'
  | 'field_too_large';

export type EligibilityDecision =
  | { decision: 'eligible'; reason: 'eligible'; identity: TabIdentity; identityKey: string; record: EligibleTabRecord; removeOnPrivateContextEnd: boolean }
  | { decision: 'excluded'; reason: Exclude<EligibilityReason, 'eligible' | 'identity_invalid' | 'context_unknown' | 'context_mismatch' | 'window_unavailable' | 'required_field_missing'>; removeOnPrivateContextEnd: boolean }
  | { decision: 'deferred'; reason: Extract<EligibilityReason, 'identity_invalid' | 'context_unknown' | 'context_mismatch' | 'window_unavailable' | 'required_field_missing'>; removeOnPrivateContextEnd: boolean };

export interface EligibilityContext {
  profileId: ProfileID;
  contextKind: ContextKind;
  projectionEpoch: ProjectionEpoch;
  projectionRevision: ProjectionRevision;
  observedAt: number;
  windows: readonly BrowserWindowSnapshot[];
  groups: readonly BrowserGroupSnapshot[];
}

const MAX_TITLE_BYTES = 2048;
const MAX_TITLE_SCALARS = 512;
const MAX_URL_BYTES = 2048;
const MAX_LABEL_BYTES = 2048;

function safeId(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isSafeInteger(value) && value >= 0 ? value : undefined;
}

function withinBounds(value: string, maxBytes: number, maxScalars: number): boolean {
  return new TextEncoder().encode(value).length <= maxBytes && [...value].length <= maxScalars;
}

function deferred(reason: Extract<EligibilityReason, 'identity_invalid' | 'context_unknown' | 'context_mismatch' | 'window_unavailable' | 'required_field_missing'>): EligibilityDecision {
  return { decision: 'deferred', reason, removeOnPrivateContextEnd: false };
}

function excluded(reason: Extract<EligibilityReason, 'invalid_url' | 'unsupported_scheme' | 'field_too_large'>, contextKind: ContextKind): EligibilityDecision {
  return { decision: 'excluded', reason, removeOnPrivateContextEnd: contextKind === 'private' };
}

export function evaluateTabEligibility(tab: BrowserTabSnapshot, context: EligibilityContext): EligibilityDecision {
  const tabId = safeId(tab.id);
  const windowId = safeId(tab.windowId);
  if (tabId === undefined) return deferred('identity_invalid');
  if (tab.incognito !== true && tab.incognito !== false) return deferred('context_unknown');
  const contextKind: ContextKind = tab.incognito ? 'private' : 'normal';
  if (contextKind !== context.contextKind) return deferred('context_mismatch');
  if (windowId === undefined) return deferred('window_unavailable');
  const window = context.windows.find(candidate => safeId(candidate.id) === windowId);
  if (!window) return deferred('window_unavailable');
  if (window.incognito !== true && window.incognito !== false) return deferred('context_unknown');
  if ((window.incognito ? 'private' : 'normal') !== contextKind) return deferred('context_mismatch');
  if (typeof tab.title !== 'string' || typeof tab.url !== 'string' || typeof tab.pinned !== 'boolean' || typeof tab.active !== 'boolean') {
    return deferred('required_field_missing');
  }
  if (!withinBounds(tab.title, MAX_TITLE_BYTES, MAX_TITLE_SCALARS) || !withinBounds(tab.url, MAX_URL_BYTES, MAX_URL_BYTES)) {
    return excluded('field_too_large', contextKind);
  }
  let parsedUrl: URL;
  try {
    parsedUrl = new URL(tab.url);
  } catch {
    return excluded('invalid_url', contextKind);
  }
  if (parsedUrl.protocol !== 'http:' && parsedUrl.protocol !== 'https:') return excluded('unsupported_scheme', contextKind);
  const observedAt = context.observedAt;
  if (!Number.isSafeInteger(observedAt) || observedAt < 0) return deferred('required_field_missing');

  const groupIdValue = tab.groupId;
  let groupId: number | null = null;
  if (groupIdValue !== undefined && groupIdValue !== -1) {
    const validGroupId = safeId(groupIdValue);
    if (validGroupId === undefined) return deferred('required_field_missing');
    groupId = validGroupId;
  }

  const identity: TabIdentity = { profile_id: context.profileId, context_kind: contextKind, tab_id: tabId };
  const group = groupId === null ? undefined : context.groups.find(candidate => safeId(candidate.id) === groupId && safeId(candidate.windowId) === windowId);
  const groupLabel = typeof group?.title === 'string' && withinBounds(group.title, MAX_LABEL_BYTES, MAX_TITLE_SCALARS) ? group.title : undefined;
  const windowLabel = typeof window.title === 'string' && withinBounds(window.title, MAX_LABEL_BYTES, MAX_TITLE_SCALARS) ? window.title : undefined;
  parsedUrl.username = '';
  parsedUrl.password = '';
  const urlDisplay = `${parsedUrl.origin}${parsedUrl.pathname}`;
  const record: EligibleTabRecord = {
    profile_id: context.profileId,
    context_kind: contextKind,
    tab_identity: identity,
    window_id: windowId,
    group_id: groupId,
    title_display: tab.title,
    title_search: tab.title,
    url_search: parsedUrl.href,
    url_display: urlDisplay,
    domain_display: parsedUrl.hostname,
    domain_search: parsedUrl.hostname,
    ...(windowLabel === undefined ? {} : { window_label_display: windowLabel, window_label_search: windowLabel }),
    ...(groupLabel === undefined ? {} : { group_label_display: groupLabel, group_label_search: groupLabel }),
    pinned: tab.pinned,
    active: tab.active,
    eligible: true,
    observed_at: observedAt,
    projection_epoch: context.projectionEpoch,
    projection_revision: context.projectionRevision,
  };
  return {
    decision: 'eligible',
    reason: 'eligible',
    identity,
    identityKey: tabIdentityKey(identity),
    record,
    removeOnPrivateContextEnd: contextKind === 'private',
  };
}
