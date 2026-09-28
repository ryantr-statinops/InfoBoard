import { test, expect } from '@playwright/test';
import { parseProfileID } from '../../extension/domain/index.js';
import { ChromeBrowserAdapter, type BrowserApi } from '../../extension/src/browser/browser-adapter.js';
import { TabObserver } from '../../extension/src/browser/tab-observer.js';

const profileId = parseProfileID('123e4567-e89b-42d3-a456-426614174000');
type Listener = (...args: unknown[]) => void;

function fixture(options: { profile?: boolean; privateAccess?: () => Promise<boolean>; tabs?: unknown[] } = {}) {
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
      getAll: async () => [
        { id: 1, incognito: false, focused: true },
        { id: 2, incognito: true, focused: false },
      ],
      ...changeEvents(['windows.onCreated', 'windows.onRemoved', 'windows.onFocusChanged']),
    },
    tabs: {
      query: async () => options.tabs ?? [
        { id: 10, windowId: 1, title: 'Normal tab', url: 'https://normal.example/a', pinned: false, active: true, incognito: false },
        { id: 20, windowId: 2, title: 'Private tab', url: 'https://private.example/b', pinned: false, active: false, incognito: true },
      ],
      ...changeEvents(['tabs.onCreated', 'tabs.onUpdated', 'tabs.onMoved', 'tabs.onAttached', 'tabs.onDetached', 'tabs.onActivated', 'tabs.onRemoved']),
    },
    tabGroups: { query: async () => [], ...changeEvents(['tabGroups.onUpdated', 'tabGroups.onMoved', 'tabGroups.onRemoved']) },
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
