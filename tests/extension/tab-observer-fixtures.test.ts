import { test, expect } from '@playwright/test';
import { readFileSync } from 'node:fs';
import { parseProfileID } from '../../extension/domain/index.js';
import { ChromeBrowserAdapter, type BrowserApi } from '../../extension/src/browser/browser-adapter.js';
import { TabObserver, type ObserverHandoff } from '../../extension/src/browser/tab-observer.js';

const profileId = parseProfileID('123e4567-e89b-42d3-a456-426614174000');
const corpus = JSON.parse(readFileSync('fixtures/browser/phase-04/tab-observation.json', 'utf8')) as Phase04Corpus;
type Listener = (...args: unknown[]) => void;

interface Phase04Scenario {
  fixture_id: string;
  private_access: boolean;
  initial: { windows: unknown[]; groups: unknown[]; tabs: unknown[] };
  events: Array<{ source?: string; args?: unknown[]; snapshot_after?: unknown[] }>;
  expected: {
    tab_ids: number[];
    status: string;
    records?: Array<{ tab_id: number; title?: string; group_label?: string; window_id?: number; url_display?: string; pinned?: boolean; active?: boolean }>;
    minimum_duplicate_events?: number;
    minimum_excluded?: number;
    minimum_deferred?: number;
    private_context_ended?: boolean;
    private_unavailable_reason?: string;
  };
}
interface Phase04Corpus { schema_version: number; phase: string; fixtures: Phase04Scenario[] }

function fakeBrowser(scenario: Phase04Scenario, edgeCompatible = false) {
  let storedProfile: unknown = profileId;
  const listeners = new Map<string, Listener>();
  const changes = (sources: string[]) => Object.fromEntries(sources.map(source => [source.split('.').at(-1)!, {
    addListener: (listener: Listener) => { listeners.set(source, listener); },
    removeListener: (listener: Listener) => { if (listeners.get(source) === listener) listeners.delete(source); },
  }]));
  const tabs = [...scenario.initial.tabs];
  const api = {
    runtime: { onInstalled: { addListener: () => undefined }, onStartup: { addListener: () => undefined }, onMessage: { addListener: () => undefined }, sendMessage: async () => undefined },
    commands: { onCommand: { addListener: () => undefined } },
    action: { openPopup: async () => undefined },
    windows: {
      getLastFocused: async () => ({ id: 1, incognito: false }),
      getAll: async () => scenario.initial.windows,
      ...changes(['windows.onCreated', 'windows.onRemoved', 'windows.onFocusChanged']),
    },
    tabs: {
      query: async () => tabs,
      ...changes(['tabs.onCreated', 'tabs.onUpdated', 'tabs.onMoved', 'tabs.onAttached', 'tabs.onDetached', 'tabs.onActivated', 'tabs.onRemoved']),
    },
    tabGroups: { query: async () => scenario.initial.groups, ...changes(edgeCompatible ? ['tabGroups.onUpdated'] : ['tabGroups.onUpdated', 'tabGroups.onMoved', 'tabGroups.onRemoved']) },
    extension: { isAllowedIncognitoAccess: async () => scenario.private_access },
    storage: { local: {
      get: async (key: string) => key === 'profile_id' ? { profile_id: storedProfile } : {},
      set: async (values: Record<string, unknown>) => { storedProfile = values.profile_id; },
      remove: async () => { storedProfile = null; },
    } },
  } as unknown as BrowserApi;
  return {
    adapter: new ChromeBrowserAdapter(api),
    fire: (source: string, ...args: unknown[]) => listeners.get(source)?.(...args),
    replaceSnapshot: (next: unknown[]) => { tabs.splice(0, tabs.length, ...next); },
  };
}

async function flushQueuedEvents(): Promise<void> {
  for (let i = 0; i < 20; i += 1) await Promise.resolve();
}

test('the eight IP-04 fixture scenarios produce their declared observable projections', async () => {
  expect(corpus.schema_version).toBe(1);
  expect(corpus.phase).toBe('IP-04');
  expect(corpus.fixtures.map(item => item.fixture_id).sort()).toEqual([
    'FX-PERMISSION-DENIED', 'FX-PRIVATE-CONTEXT', 'FX-TAB-ELIGIBILITY', 'FX-TAB-EVENT-COALESCE',
    'FX-TAB-EVENTS', 'FX-TAB-ID-REUSE', 'FX-TAB-SNAPSHOT-CONVERGENCE', 'FX-TAB-WINDOW-LIFECYCLE',
  ]);

  for (const scenario of corpus.fixtures) {
    const fake = fakeBrowser(scenario);
    const observer = new TabObserver(fake.adapter, () => 100);
    const handoffs: ObserverHandoff[] = [];
    observer.subscribe(event => handoffs.push(event));
    await observer.start();
    await flushQueuedEvents();

    for (const event of scenario.events) {
      if (event.snapshot_after) fake.replaceSnapshot(event.snapshot_after);
      if (event.source) fake.fire(event.source, ...(event.args ?? []));
      await flushQueuedEvents();
    }

    const view = observer.view();
    expect(view.status, scenario.fixture_id).toBe(scenario.expected.status);
    const records = view.records;
    expect(records.map(record => record.tab_identity.tab_id).sort((a, b) => a - b), scenario.fixture_id).toEqual([...scenario.expected.tab_ids].sort((a, b) => a - b));
    expect(new Set(records.map(record => `${record.profile_id}:${record.context_kind}:${record.tab_identity.tab_id}`)).size, scenario.fixture_id).toBe(records.length);

    for (const expected of scenario.expected.records ?? []) {
      const record = records.find(candidate => candidate.tab_identity.tab_id === expected.tab_id);
      expect(record, `${scenario.fixture_id}: record ${expected.tab_id}`).toBeDefined();
      if (!record) continue;
      if (expected.title !== undefined) expect(record.title_display, scenario.fixture_id).toBe(expected.title);
      if (expected.group_label !== undefined) expect(record.group_label_display, scenario.fixture_id).toBe(expected.group_label);
      if (expected.window_id !== undefined) expect(record.window_id, scenario.fixture_id).toBe(expected.window_id);
      if (expected.url_display !== undefined) expect(record.url_display, scenario.fixture_id).toBe(expected.url_display);
      if (expected.pinned !== undefined) expect(record.pinned, scenario.fixture_id).toBe(expected.pinned);
      if (expected.active !== undefined) expect(record.active, scenario.fixture_id).toBe(expected.active);
    }
    if (scenario.expected.minimum_duplicate_events !== undefined) expect(view.diagnostics.duplicateEvents, scenario.fixture_id).toBeGreaterThanOrEqual(scenario.expected.minimum_duplicate_events);
    if (scenario.expected.minimum_excluded !== undefined) expect(view.diagnostics.excludedTabs, scenario.fixture_id).toBeGreaterThanOrEqual(scenario.expected.minimum_excluded);
    if (scenario.expected.minimum_deferred !== undefined) expect(view.diagnostics.deferredTabs, scenario.fixture_id).toBeGreaterThanOrEqual(scenario.expected.minimum_deferred);
    if (scenario.expected.private_context_ended) expect(handoffs.some(event => event.kind === 'private_context_ended' && event.context_kind === 'private'), scenario.fixture_id).toBe(true);
    if (scenario.expected.private_unavailable_reason) expect(handoffs.some(event => event.kind === 'status' && event.context_kind === 'private' && event.resync_reason === scenario.expected.private_unavailable_reason), scenario.fixture_id).toBe(true);
  }
});

async function projectionFor(scenario: Phase04Scenario, edgeCompatible: boolean) {
  const fake = fakeBrowser(scenario, edgeCompatible);
  const observer = new TabObserver(fake.adapter, () => 100);
  await observer.start();
  await flushQueuedEvents();
  for (const event of scenario.events) {
    if (event.snapshot_after) fake.replaceSnapshot(event.snapshot_after);
    if (event.source) fake.fire(event.source, ...(event.args ?? []));
    await flushQueuedEvents();
  }
  const view = observer.view();
  const records = view.records.map(record => ({
    profile_id: record.profile_id,
    context_kind: record.context_kind,
    tab_id: record.tab_identity.tab_id,
    window_id: record.window_id,
    group_id: record.group_id,
    title_display: record.title_display,
    url_display: record.url_display,
    domain_display: record.domain_display,
    group_label_display: record.group_label_display,
    window_label_display: record.window_label_display,
    pinned: record.pinned,
    active: record.active,
  })).sort((left, right) => `${left.context_kind}:${left.tab_id}`.localeCompare(`${right.context_kind}:${right.tab_id}`));
  return { status: view.status, resync_required: view.resync_required, records };
}

test('Chrome-complete and Edge-optional fake APIs converge over the same fixture corpus', async () => {
  for (const scenario of corpus.fixtures) {
    const chromeProjection = await projectionFor(scenario, false);
    const edgeProjection = await projectionFor(scenario, true);
    expect(edgeProjection, scenario.fixture_id).toEqual(chromeProjection);
  }
});
