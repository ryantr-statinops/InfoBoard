import { isUnknownRecord as isRecord } from './type-guards.js';
import type { ContextKind } from '../../domain/index.js';

export type BrowserEventSource =
  | 'tabs.onCreated'
  | 'tabs.onUpdated'
  | 'tabs.onMoved'
  | 'tabs.onAttached'
  | 'tabs.onDetached'
  | 'tabs.onActivated'
  | 'tabs.onRemoved'
  | 'tabGroups.onUpdated'
  | 'tabGroups.onMoved'
  | 'tabGroups.onRemoved'
  | 'windows.onCreated'
  | 'windows.onRemoved'
  | 'windows.onFocusChanged';

export interface BrowserApiEvent {
  source: BrowserEventSource;
  args: readonly unknown[];
}

export interface BrowserTabSnapshot {
  id?: unknown;
  windowId?: unknown;
  groupId?: unknown;
  title?: unknown;
  url?: unknown;
  pinned?: unknown;
  active?: unknown;
  incognito?: unknown;
  windowLabel?: unknown;
  groupLabel?: unknown;
}

export interface BrowserWindowSnapshot {
  id?: unknown;
  incognito?: unknown;
  focused?: unknown;
  title?: unknown;
}

export interface BrowserGroupSnapshot {
  id?: unknown;
  windowId?: unknown;
  title?: unknown;
}

export interface BrowserTabSnapshotBatch {
  tabs: BrowserTabSnapshot[];
  windows: BrowserWindowSnapshot[];
  groups: BrowserGroupSnapshot[];
  optionalCapabilityErrors: number;
}

export type BrowserEventKind =
  | 'create'
  | 'update'
  | 'move'
  | 'group'
  | 'pin'
  | 'activate'
  | 'window_changed'
  | 'window_removed'
  | 'detach'
  | 'remove';

export interface NormalizedBrowserEvent {
  source: BrowserEventSource;
  kind: BrowserEventKind;
  tabId?: number;
  contextKind?: ContextKind;
  windowId?: number | null;
  groupId?: number;
  patch?: BrowserTabSnapshot;
  changedFields: string[];
  removedGroup?: boolean;
  focused?: boolean;
  requiresSnapshot?: boolean;
}

export type NormalizeBrowserEventResult =
  | { ok: true; event: NormalizedBrowserEvent }
  | { ok: false; reason: 'malformed_event' | 'identity_unavailable' };


function nonNegativeInteger(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isSafeInteger(value) && value >= 0 ? value : undefined;
}

function groupInteger(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isSafeInteger(value) && value >= -1 ? value : undefined;
}

function tabSnapshot(value: unknown): BrowserTabSnapshot | undefined {
  if (!isRecord(value)) return undefined;
  const result: BrowserTabSnapshot = {};
  for (const field of ['id', 'windowId', 'groupId', 'title', 'url', 'pinned', 'active', 'incognito'] as const) {
    if (Object.hasOwn(value, field)) result[field] = value[field];
  }
  return result;
}

function contextKind(tab: BrowserTabSnapshot | undefined): ContextKind | undefined {
  if (tab?.incognito === true) return 'private';
  if (tab?.incognito === false) return 'normal';
  return undefined;
}

function tabEvent(
  source: BrowserEventSource,
  kind: BrowserEventKind,
  tab: BrowserTabSnapshot | undefined,
  patch: BrowserTabSnapshot | undefined,
  changedFields: string[],
): NormalizeBrowserEventResult {
  const tabId = nonNegativeInteger(tab?.id);
  if (tabId === undefined) return { ok: false, reason: 'identity_unavailable' };
  const windowId = nonNegativeInteger(patch?.windowId ?? tab?.windowId);
  const groupIdValue = groupInteger(patch?.groupId ?? tab?.groupId);
  return {
    ok: true,
    event: {
      source,
      kind,
      tabId,
      contextKind: contextKind(patch) ?? contextKind(tab),
      ...(windowId === undefined ? {} : { windowId }),
      ...(groupIdValue === undefined || groupIdValue === -1 ? {} : { groupId: groupIdValue }),
      ...(patch === undefined ? {} : { patch }),
      changedFields,
    },
  };
}

function valueFromChange(change: Record<string, unknown>, tab: BrowserTabSnapshot | undefined, field: keyof BrowserTabSnapshot): unknown {
  if (Object.hasOwn(change, field)) return change[field];
  return tab && Object.hasOwn(tab, field) ? tab[field] : undefined;
}

export function normalizeBrowserEvent(input: BrowserApiEvent): NormalizeBrowserEventResult {
  const args = input.args;
  switch (input.source) {
    case 'tabs.onCreated': {
      const tab = tabSnapshot(args[0]);
      if (!tab) return { ok: false, reason: 'malformed_event' };
      return tabEvent(input.source, 'create', tab, tab, ['id', 'windowId', 'groupId', 'title', 'url', 'pinned', 'active']);
    }
    case 'tabs.onUpdated': {
      const tabId = nonNegativeInteger(args[0]);
      if (tabId === undefined || !isRecord(args[1])) return { ok: false, reason: 'identity_unavailable' };
      const changes = args[1];
      const tab = tabSnapshot(args[2]);
      if (tab && nonNegativeInteger(tab.id) !== tabId) return { ok: false, reason: 'identity_unavailable' };
      const patch: BrowserTabSnapshot = {};
      const changedFields: string[] = [];
      const fields = ['title', 'url', 'windowId', 'groupId', 'pinned', 'active'] as const;
      for (const field of fields) {
        if (Object.hasOwn(changes, field)) {
          patch[field] = valueFromChange(changes, tab, field);
          changedFields.push(field);
        }
      }
      if (tab && Object.hasOwn(tab, 'incognito')) patch.incognito = tab.incognito;
      const kind: BrowserEventKind = changedFields.includes('pinned') ? 'pin' : changedFields.includes('groupId') ? 'group' : changedFields.includes('windowId') ? 'move' : changedFields.includes('active') ? 'activate' : 'update';
      return tabEvent(input.source, kind, { ...(tab ?? {}), id: tabId }, patch, changedFields);
    }
    case 'tabs.onMoved':
    case 'tabs.onAttached': {
      const tabId = nonNegativeInteger(args[0]);
      if (tabId === undefined || !isRecord(args[1])) return { ok: false, reason: 'identity_unavailable' };
      const windowId = nonNegativeInteger(args[1].windowId);
      if (windowId === undefined) return { ok: false, reason: 'identity_unavailable' };
      return { ok: true, event: { source: input.source, kind: 'move', tabId, windowId, changedFields: ['windowId'], requiresSnapshot: true } };
    }
    case 'tabs.onDetached': {
      const tabId = nonNegativeInteger(args[0]);
      const details = args[1];
      if (tabId === undefined || !isRecord(details)) return { ok: false, reason: 'identity_unavailable' };
      const windowId = nonNegativeInteger(details.oldWindowId);
      return { ok: true, event: { source: input.source, kind: 'detach', tabId, ...(windowId === undefined ? {} : { windowId }), changedFields: ['windowId'], requiresSnapshot: true } };
    }
    case 'tabs.onActivated': {
      const info = args[0];
      if (!isRecord(info)) return { ok: false, reason: 'malformed_event' };
      const tabId = nonNegativeInteger(info.tabId), windowId = nonNegativeInteger(info.windowId);
      if (tabId === undefined || windowId === undefined) return { ok: false, reason: 'identity_unavailable' };
      return { ok: true, event: { source: input.source, kind: 'activate', tabId, windowId, changedFields: ['active'] } };
    }
    case 'tabs.onRemoved': {
      const tabId = nonNegativeInteger(args[0]);
      const info = args[1];
      if (tabId === undefined || !isRecord(info)) return { ok: false, reason: 'identity_unavailable' };
      const windowId = nonNegativeInteger(info.windowId);
      return { ok: true, event: { source: input.source, kind: 'remove', tabId, ...(windowId === undefined ? {} : { windowId }), changedFields: ['removed'] } };
    }
    case 'tabGroups.onUpdated': {
      const group = args[0];
      if (!isRecord(group)) return { ok: false, reason: 'malformed_event' };
      const groupId = nonNegativeInteger(group.id), windowId = nonNegativeInteger(group.windowId);
      if (groupId === undefined) return { ok: false, reason: 'identity_unavailable' };
      const patch: BrowserTabSnapshot = {};
      if (Object.hasOwn(group, 'title')) patch.groupLabel = group.title;
      return { ok: true, event: { source: input.source, kind: 'group', groupId, ...(windowId === undefined ? {} : { windowId }), patch, changedFields: Object.keys(patch) } };
    }
    case 'tabGroups.onMoved': {
      const groupId = nonNegativeInteger(args[0]), details = args[1];
      if (groupId === undefined || !isRecord(details)) return { ok: false, reason: 'identity_unavailable' };
      const windowId = nonNegativeInteger(details.windowId);
      return { ok: true, event: { source: input.source, kind: 'group', groupId, ...(windowId === undefined ? {} : { windowId }), changedFields: ['windowId'], requiresSnapshot: true } };
    }
    case 'tabGroups.onRemoved': {
      const group = args[0];
      const groupId = nonNegativeInteger(group);
      if (groupId === undefined) return { ok: false, reason: 'identity_unavailable' };
      return { ok: true, event: { source: input.source, kind: 'group', groupId, changedFields: ['groupId', 'groupLabel'], removedGroup: true } };
    }
    case 'windows.onCreated': {
      const window = args[0];
      if (!isRecord(window)) return { ok: false, reason: 'malformed_event' };
      const windowId = nonNegativeInteger(window.id);
      if (windowId === undefined) return { ok: false, reason: 'identity_unavailable' };
      return { ok: true, event: { source: input.source, kind: 'window_changed', windowId, contextKind: window.incognito === true ? 'private' : window.incognito === false ? 'normal' : undefined, focused: window.focused === true, changedFields: ['window_created'] } };
    }
    case 'windows.onRemoved': {
      const windowId = nonNegativeInteger(args[0]);
      if (windowId === undefined) return { ok: false, reason: 'identity_unavailable' };
      return { ok: true, event: { source: input.source, kind: 'window_removed', windowId, changedFields: ['window_removed'] } };
    }
    case 'windows.onFocusChanged': {
      const rawWindowId = args[0];
      if (rawWindowId === -1) return { ok: true, event: { source: input.source, kind: 'window_changed', windowId: null, focused: false, changedFields: ['focused_window'] } };
      const windowId = nonNegativeInteger(rawWindowId);
      if (windowId === undefined) return { ok: false, reason: 'identity_unavailable' };
      return { ok: true, event: { source: input.source, kind: 'window_changed', windowId, focused: true, changedFields: ['focused_window'] } };
    }
  }
}
