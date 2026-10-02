import { test, expect } from '@playwright/test';
import { parseProfileID } from '../../extension/domain/index.js';
import { ChromeBrowserAdapter, type BrowserApi } from '../../extension/src/browser/browser-adapter.js';
import { TabObserver, type ObserverHandoff } from '../../extension/src/browser/tab-observer.js';

const profileId = parseProfileID('123e4567-e89b-42d3-a456-426614174000');
type Listener = (...args: unknown[]) => void;

function fixture(options: {
  profile?: boolean;
  privateAccess?: () => Promise<boolean>;
  tabs?: unknown[];
  windows?: unknown[];
  groups?: unknown[];
  queryTabs?: () => Promise<unknown[]>;
  queryWindows?: () => Promise<unknown[]>;
  queryGroups?: () => Promise<unknown[]>;
} = {}) {
  let storedProfile: unknown = options.profile ? profileId : null;
  let installListener: ((details: { reason: string }) => void) | undefined;
  let startupListener: (() => void) | undefined;
  const events = new Map<string, Listener>();
  const changeEvents = (names: string[]) => Object.fromEntries(names.map(name => [name.split('.').at(-1)!, {
    addListener: (listener: Listener) => { events.set(name, listener); },
    removeListener: (listener: Listener) => { if (events.get(name) === listener) events.delete(name); },
  }]));
  const api = {
    runtime: {
      onInstalled: { addListener: (listener: (details: { reason: string }) => void) => { installListener = listener; } },
      onStartup: { addListener: (listener: () => void) => { startupListener = listener; } },
      onMessage: { addListener: () => undefined },
      sendMessage: async () => undefined,
    },
    commands: { onCommand: { addListener: () => undefined } },
    action: { openPopup: async () => undefined },
    windows: {
      getLastFocused: async () => ({ id: 1, incognito: false }),
      getAll: options.queryWindows ?? (async () => options.windows ?? [
        { id: 1, incognito: false, focused: true },
        { id: 2, incognito: true, focused: false },
      ]),
      ...changeEvents(['windows.onCreated', 'windows.onRemoved', 'windows.onFocusChanged']),
    },
    tabs: {
      query: options.queryTabs ?? (async () => options.tabs ?? [
        { id: 10, windowId: 1, title: 'Normal tab', url: 'https://normal.example/a', pinned: false, active: true, incognito: false },
        { id: 20, windowId: 2, title: 'Private tab', url: 'https://private.example/b', pinned: false, active: false, incognito: true },
      ]),
      ...changeEvents(['tabs.onCreated', 'tabs.onUpdated', 'tabs.onMoved', 'tabs.onAttached', 'tabs.onDetached', 'tabs.onActivated', 'tabs.onRemoved']),
    },
    tabGroups: { query: options.queryGroups ?? (async () => options.groups ?? []), ...changeEvents(['tabGroups.onUpdated', 'tabGroups.onMoved', 'tabGroups.onRemoved']) },
    extension: options.privateAccess ? { isAllowedIncognitoAccess: options.privateAccess } : undefined,
    storage: { local: {
      get: async (key: string) => key === 'profile_id' && storedProfile !== null ? { profile_id: storedProfile } : {},
      set: async (values: Record<string, unknown>) => { storedProfile = values.profile_id; },
      remove: async () => { storedProfile = null; },
    } },
  } as unknown as BrowserApi;
  return {
    adapter: new ChromeBrowserAdapter(api), events,
    fireInstall: (reason: string) => installListener?.({ reason }),
    fireStartup: () => startupListener?.(),
    fire: (name: string, ...args: unknown[]) => events.get(name)?.(...args),
    profile: () => storedProfile,
  };
}

async function waitForReady(observer: TabObserver): Promise<void> {
  for (let i = 0; i < 50 && observer.view().status !== 'ready'; i += 1) await Promise.resolve();
}

test('profile surface contract remains available and private access is fail-closed', async () => {
  const denied = fixture({ profile: true, privateAccess: async () => false });
  expect((await denied.adapter.currentProfileContext()).ok).toBe(true);
  expect(await denied.adapter.privateContextAccess()).toEqual({ ok: true, value: false });
  const absent = fixture({ profile: true });
  expect(await absent.adapter.privateContextAccess()).toEqual({ ok: false, error: { kind: 'unsupported', retryable: false } });
  const rejected = fixture({ profile: true, privateAccess: async () => { throw new Error('permission query failed'); } });
  const failure = await rejected.adapter.privateContextAccess();
  expect(failure.ok).toBe(false);
  if (!failure.ok) expect(failure.error.kind).toBe('permission_denied');
});

test('fresh install initializes profile before retrying observer startup and projecting a snapshot', async () => {
  const fake = fixture({ privateAccess: async () => false });
  const observer = new TabObserver(fake.adapter, () => 100);
  const handoffs: Array<{ kind: string; context_kind?: string; records?: unknown[] }> = [];
  observer.subscribe(handoff => handoffs.push(handoff));
  await observer.start();
  expect(observer.view().status).toBe('unavailable');
  fake.adapter.registerLifecycleListeners(() => { void observer.start(); }, () => { void observer.start(); });
  fake.fireInstall('install');
  await waitForReady(observer);
  expect(typeof fake.profile()).toBe('string');
  expect(observer.view().records[0]?.profile_id).toBe(fake.profile());
  expect(observer.view().status).toBe('ready');
  expect(observer.view().records.map(record => record.title_display)).toEqual(['Normal tab']);
  expect(handoffs.findIndex(event => event.kind === 'status')).toBeLessThan(handoffs.findIndex(event => event.kind === 'snapshot'));
});

test('private access denial keeps normal projection and reports private unavailable status', async () => {
  const fake = fixture({ profile: true, privateAccess: async () => false });
  const observer = new TabObserver(fake.adapter, () => 100);
  const handoffs: Array<{ kind: string; context_kind?: string; records?: unknown[]; resync_reason?: string }> = [];
  observer.subscribe(handoff => handoffs.push(handoff));
  await observer.start();
  expect(observer.view().status).toBe('ready');
  expect(observer.view().records.map(record => record.title_display)).toEqual(['Normal tab']);
  expect(handoffs.some(event => event.kind === 'status' && event.context_kind === 'private' && event.resync_reason === 'permission_denied')).toBe(true);
  fake.fire('tabs.onCreated', { id: 21, windowId: 2, title: 'New private tab', url: 'https://private.example/c', pinned: false, active: false, incognito: true });
  for (let i = 0; i < 10; i += 1) await Promise.resolve();
  expect(observer.view().records.map(record => record.context_kind)).toEqual(['normal']);
});

test('granted private access keeps separate projection and clears it when private window ends', async () => {
  const fake = fixture({ profile: true, privateAccess: async () => true });
  const observer = new TabObserver(fake.adapter, () => 100);
  const handoffs: Array<{ kind: string; context_kind?: string }> = [];
  observer.subscribe(event => handoffs.push(event));
  await observer.start();
  expect(observer.view().records.map(record => record.context_kind).sort()).toEqual(['normal', 'private']);
  fake.fire('windows.onRemoved', 2);
  for (let i = 0; i < 10; i += 1) await Promise.resolve();
  expect(observer.view().records.map(record => record.context_kind)).toEqual(['normal']);
  expect(handoffs.some(event => event.kind === 'private_context_ended' && event.context_kind === 'private')).toBe(true);
});

test('authoritative snapshot clears reused-ID quarantine for the new epoch', async () => {
  const snapshotTabs: unknown[] = [{ id: 10, windowId: 1, title: 'Original tab', url: 'https://original.example/a', pinned: false, active: true, incognito: false }];
  const fake = fixture({ profile: true, privateAccess: async () => true, tabs: snapshotTabs });
  const observer = new TabObserver(fake.adapter, () => 100);
  await observer.start();

  const removed = Promise.withResolvers<void>();
  const unsubscribeRemove = observer.subscribe(event => { if (event.kind === 'remove') removed.resolve(); });
  fake.fire('tabs.onRemoved', 10, { windowId: 1, isWindowClosing: false });
  await removed.promise;
  unsubscribeRemove();

  snapshotTabs[0] = { id: 10, windowId: 1, title: 'Reused tab', url: 'https://reused.example/b', pinned: false, active: true, incognito: false };
  const resynced = Promise.withResolvers<ObserverHandoff>();
  const unsubscribeSnapshot = observer.subscribe(event => {
    if (event.kind === 'snapshot' && event.context_kind === 'normal' && event.records?.some(record => record.title_display === 'Reused tab')) resynced.resolve(event);
  });
  fake.fire('tabs.onCreated', snapshotTabs[0]);
  const snapshot = await resynced.promise;
  unsubscribeSnapshot();
  expect(snapshot.kind).toBe('snapshot');

  const processed = Promise.withResolvers<ObserverHandoff>();
  const unsubscribeUpdate = observer.subscribe(event => {
    if (event.kind === 'upsert' || event.kind === 'snapshot') processed.resolve(event);
  });
  fake.fire('tabs.onUpdated', 10, { title: 'Updated after reuse' }, { id: 10, windowId: 1, title: 'Updated after reuse', url: 'https://reused.example/b', pinned: false, active: true, incognito: false });
  const update = await processed.promise;
  unsubscribeUpdate();
  expect(update.kind).toBe('upsert');
  if (update.kind === 'upsert') expect(update.record.title_display).toBe('Updated after reuse');
});

test('a removed normal window cannot leave eligible orphan tabs', async () => {
  const fake = fixture({
    profile: true,
    privateAccess: async () => true,
    windows: [{ id: 1, incognito: false }, { id: 3, incognito: false }],
    tabs: [
      { id: 10, windowId: 1, title: 'Kept tab', url: 'https://kept.example/', pinned: false, active: true, incognito: false },
      { id: 30, windowId: 3, title: 'Removed window tab', url: 'https://removed.example/', pinned: false, active: false, incognito: false },
    ],
  });
  const observer = new TabObserver(fake.adapter, () => 100);
  await observer.start();
  const removed = Promise.withResolvers<ObserverHandoff>();
  const unsubscribe = observer.subscribe(event => {
    if (event.kind === 'remove' && event.tab_identity?.tab_id === 30) removed.resolve(event);
  });
  fake.fire('windows.onRemoved', 3);
  const handoff = await removed.promise;
  unsubscribe();
  expect(handoff.context_kind).toBe('normal');
  expect(observer.view().records.map(record => record.tab_identity.tab_id)).toEqual([10]);
});

test('group-label updates affect only the tabs in the matching group', async () => {
  const fake = fixture({
    profile: true,
    privateAccess: async () => true,
    windows: [{ id: 1, incognito: false }],
    groups: [{ id: 7, windowId: 1, title: 'Old label' }, { id: 8, windowId: 1, title: 'Other label' }],
    tabs: [
      { id: 10, windowId: 1, groupId: 7, title: 'First', url: 'https://first.example/', pinned: false, active: true, incognito: false },
      { id: 11, windowId: 1, groupId: 8, title: 'Second', url: 'https://second.example/', pinned: false, active: false, incognito: false },
    ],
  });
  const observer = new TabObserver(fake.adapter, () => 100);
  await observer.start();
  const changed = Promise.withResolvers<ObserverHandoff>();
  const unsubscribe = observer.subscribe(event => {
    if (event.kind === 'upsert' && event.tab_identity?.tab_id === 10) changed.resolve(event);
  });
  fake.fire('tabGroups.onUpdated', { id: 7, windowId: 1, title: 'New label' });
  const handoff = await changed.promise;
  unsubscribe();
  expect(handoff.kind).toBe('upsert');
  expect(handoff.event_sequence).toBeGreaterThan(0);
  expect(handoff.effective_change).toBe(true);
  expect(handoff.diagnostics.handoffListenerErrors).toBe(0);
  const records = observer.view().records;
  expect(records.find(record => record.tab_identity.tab_id === 10)?.group_label_display).toBe('New label');
  expect(records.find(record => record.tab_identity.tab_id === 11)?.group_label_display).toBe('Other label');
});

test('group updates can restore optional labels after the snapshot omitted group metadata', async () => {
  const fake = fixture({
    profile: true,
    privateAccess: async () => true,
    windows: [{ id: 1, incognito: false }],
    groups: [],
    tabs: [{ id: 10, windowId: 1, groupId: 7, title: 'Grouped tab', url: 'https://grouped.example/', pinned: false, active: true, incognito: false }],
  });
  const observer = new TabObserver(fake.adapter, () => 100);
  await observer.start();
  expect(observer.view().records[0]?.group_label_display).toBeUndefined();
  const upserted = Promise.withResolvers<ObserverHandoff>();
  const unsubscribe = observer.subscribe(event => {
    if (event.kind === 'upsert' && event.tab_identity?.tab_id === 10) upserted.resolve(event);
  });
  fake.fire('tabGroups.onUpdated', { id: 7, windowId: 1, title: 'Arrived later' });
  const handoff = await upserted.promise;
  unsubscribe();
  expect(handoff.kind).toBe('upsert');
  if (handoff.kind === 'upsert') expect(handoff.record.group_label_display).toBe('Arrived later');
});

test('optional group-query failure preserves eligible tabs with an absent group label', async () => {
  const fake = fixture({
    profile: true,
    privateAccess: async () => true,
    windows: [{ id: 1, incognito: false }],
    queryGroups: async () => { throw new Error('optional tab group query unavailable'); },
    tabs: [{ id: 10, windowId: 1, groupId: 7, title: 'Grouped tab', url: 'https://grouped.example/', pinned: false, active: true, incognito: false }],
  });
  const observer = new TabObserver(fake.adapter, () => 100);
  await observer.start();
  expect(observer.view().status).toBe('ready');
  expect(observer.view().records).toHaveLength(1);
  expect(observer.view().records[0]?.group_id).toBe(7);
  expect(observer.view().records[0]?.group_label_display).toBeUndefined();
  expect(observer.view().diagnostics.optionalCapabilityErrors).toBeGreaterThan(0);
});

test('required tab-query permission denial fails closed instead of reporting an empty snapshot', async () => {
  const fake = fixture({
    profile: true,
    privateAccess: async () => true,
    queryTabs: async () => { throw new Error('permission denied querying tabs'); },
  });
  const observer = new TabObserver(fake.adapter, () => 100);
  const statuses: ObserverHandoff[] = [];
  observer.subscribe(event => { if (event.kind === 'status') statuses.push(event); });
  await observer.start();
  expect(observer.view().status).toBe('unavailable');
  expect(observer.view().records).toEqual([]);
  expect(statuses.some(event => event.resync_reason === 'permission_denied')).toBe(true);
});

test('unsupported private-access capability keeps normal observation available', async () => {
  const fake = fixture({ profile: true });
  const observer = new TabObserver(fake.adapter, () => 100);
  const statuses: ObserverHandoff[] = [];
  observer.subscribe(event => { if (event.kind === 'status') statuses.push(event); });
  await observer.start();
  expect(observer.view().status).toBe('ready');
  expect(observer.view().records.map(record => record.context_kind)).toEqual(['normal']);
  expect(statuses.some(event => event.context_kind === 'private' && event.resync_reason === 'unsupported')).toBe(true);
});

test('a successful empty snapshot is ready and distinct from snapshot failure', async () => {
  const fake = fixture({ profile: true, privateAccess: async () => true, windows: [{ id: 1, incognito: false }], tabs: [] });
  const observer = new TabObserver(fake.adapter, () => 100);
  const snapshots: ObserverHandoff[] = [];
  observer.subscribe(event => { if (event.kind === 'snapshot' && event.context_kind === 'normal') snapshots.push(event); });
  await observer.start();
  expect(observer.view().status).toBe('ready');
  expect(observer.view().records).toEqual([]);
  expect(snapshots).toHaveLength(1);
  expect(snapshots[0]?.records).toEqual([]);
});

test('observer diagnostics never repeat raw tab title or URL values', async () => {
  const secretTitle = 'private-title-secret';
  const secretUrl = 'https://safe.example/path?token=private-secret#private-fragment';
  const fake = fixture({
    profile: true,
    privateAccess: async () => true,
    windows: [{ id: 1, incognito: false }],
    tabs: [{ id: 10, windowId: 1, groupId: -1, title: secretTitle, url: secretUrl, pinned: false, active: true, incognito: false }],
  });
  const observer = new TabObserver(fake.adapter, () => 100);
  const diagnostics: string[] = [];
  observer.subscribe(event => { diagnostics.push(JSON.stringify(event.diagnostics)); });
  await observer.start();
  const joined = diagnostics.join('\n');
  expect(joined).not.toContain(secretTitle);
  expect(joined).not.toContain('private-secret');
  expect(joined).not.toContain('private-fragment');
});
