import { test, expect } from '@playwright/test';
import { normalizeBrowserEvent, type BrowserApiEvent } from '../../extension/src/browser/event-normalizer.js';

test('create notifications retain profile scope and explicit browser state', () => {
  const result = normalizeBrowserEvent({
    source: 'tabs.onCreated',
    args: [{ id: 10, windowId: 2, groupId: -1, title: 'Tab', url: 'https://example.test/', pinned: false, active: true, incognito: false }],
  });
  expect(result.ok).toBe(true);
  if (!result.ok) return;
  expect(result.event.kind).toBe('create');
  expect(result.event.tabId).toBe(10);
  expect(result.event.windowId).toBe(2);
  expect(result.event.contextKind).toBe('normal');
  expect(result.event.changedFields).toContain('url');
});

test('partial updates include only changed fields while retaining the tab context', () => {
  const input: BrowserApiEvent = {
    source: 'tabs.onUpdated',
    args: [10, { title: 'Renamed' }, { id: 10, windowId: 2, title: 'Renamed', url: 'https://example.test/', pinned: false, active: true, incognito: true }],
  };
  const result = normalizeBrowserEvent(input);
  expect(result.ok).toBe(true);
  if (!result.ok) return;
  expect(result.event.kind).toBe('update');
  expect(result.event.contextKind).toBe('private');
  expect(result.event.patch).toMatchObject({ title: 'Renamed', incognito: true });
  expect(Object.hasOwn(result.event.patch ?? {}, 'url')).toBe(false);
  expect(result.event.changedFields).toEqual(['title']);
});

test('move and detach events require an authoritative snapshot', () => {
  const moved = normalizeBrowserEvent({ source: 'tabs.onMoved', args: [10, { windowId: 3 }] });
  const detached = normalizeBrowserEvent({ source: 'tabs.onDetached', args: [10, { oldWindowId: 2, oldPosition: 0 }] });
  expect(moved.ok && moved.event.requiresSnapshot).toBe(true);
  expect(detached.ok && detached.event.requiresSnapshot).toBe(true);
});

test('window removal preserves its ID as a typed lifecycle event', () => {
  const result = normalizeBrowserEvent({ source: 'windows.onRemoved', args: [3] });
  expect(result.ok).toBe(true);
  if (!result.ok) return;
  expect(result.event.kind).toBe('window_removed');
  expect(result.event.windowId).toBe(3);
});

test('malformed tab updates fail closed instead of inventing identity', () => {
  const result = normalizeBrowserEvent({ source: 'tabs.onUpdated', args: ['10', { title: 'Renamed' }, null] });
  expect(result).toEqual({ ok: false, reason: 'identity_unavailable' });
});

test('Chrome tab attachment reads its newWindowId field and resynchronizes', () => {
  const result = normalizeBrowserEvent({ source: 'tabs.onAttached', args: [10, { newWindowId: 3, newPosition: 0 }] });
  expect(result.ok).toBe(true);
  if (!result.ok) return;
  expect(result.event.kind).toBe('move');
  expect(result.event.windowId).toBe(3);
  expect(result.event.requiresSnapshot).toBe(true);
});

test('group move and removal preserve their snapshot/recovery semantics', () => {
  const moved = normalizeBrowserEvent({ source: 'tabGroups.onMoved', args: [7, { windowId: 3, oldWindowId: 1 }] });
  const removed = normalizeBrowserEvent({ source: 'tabGroups.onRemoved', args: [7] });
  expect(moved.ok && moved.event.kind === 'group' && moved.event.requiresSnapshot).toBe(true);
  expect(removed.ok && removed.event.kind === 'group' && removed.event.removedGroup).toBe(true);
});

test('window creation and focus notifications remain scoped typed events', () => {
  const created = normalizeBrowserEvent({ source: 'windows.onCreated', args: [{ id: 3, incognito: true, focused: false }] });
  const focused = normalizeBrowserEvent({ source: 'windows.onFocusChanged', args: [3] });
  expect(created.ok && created.event.kind === 'window_changed' && created.event.contextKind).toBe('private');
  expect(focused.ok && focused.event.kind).toBe('window_changed');
  expect(focused.ok && focused.event.windowId).toBe(3);
});

test('tab group reassignment remains a tab-level delta including removal from a group', () => {
  const result = normalizeBrowserEvent({
    source: 'tabs.onUpdated',
    args: [10, { groupId: -1 }, { id: 10, windowId: 1, groupId: -1, title: 'Ungrouped', url: 'https://example.test/', pinned: false, active: true, incognito: false }],
  });
  expect(result.ok).toBe(true);
  if (!result.ok) return;
  expect(result.event.kind).toBe('group');
  expect(result.event.tabId).toBe(10);
  expect(result.event.patch?.groupId).toBe(-1);
  expect(result.event.groupId).toBeUndefined();
});
