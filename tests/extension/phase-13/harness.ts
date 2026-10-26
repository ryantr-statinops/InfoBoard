import { chromium, type BrowserContext, type Page } from '@playwright/test';
import { mkdtemp, readFile, rm } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';

export interface UiRow {
  result_id: string;
  tab_id: number;
  window_id: number;
  group_id?: number;
  title: string;
  domain: string;
  window_label?: string;
  group_label?: string;
  pinned?: boolean;
  score?: number;
  best_class?: string;
  explanations?: string[];
}

export interface UiResponse {
  status: string;
  rows?: UiRow[];
  projection_revision?: number;
  projection_epoch?: string;
  persistence_state?: string;
  index_state?: string;
  defer?: boolean;
  error?: { source: 'host' | 'client'; code: string; retryable: boolean };
}

export interface UiFixture {
  fixture_id: string;
  profile_id: string;
  context_kind: 'normal' | 'private';
  surface: { width: number; height: number };
  projection_revision?: number;
  projection_epoch?: string;
  command_started_at?: number;
  private_unavailable?: boolean;
  defer_responses?: boolean;
  responses?: UiResponse[];
  activation?: {
    ok: boolean;
    confirmed: boolean;
    error_code?: string;
    browser_state?: 'unchanged' | 'window_focused' | 'tab_active' | 'unknown';
    next_action?: 'dismiss' | 'retry' | 'refresh' | 'none';
  };
}

export interface SurfaceHarness {
  ctx: BrowserContext;
  extensionId: string;
  surfaceUrl: string;
  dir: string;
}

export async function launchSurfaceHarness(viewport = { width: 480, height: 600 }): Promise<SurfaceHarness> {
  const dir = await mkdtemp(path.join(os.tmpdir(), 'infoboard-surface-'));
  const ctx = await chromium.launchPersistentContext(dir, {
    channel: 'chromium',
    headless: false,
    viewport,
    args: [`--disable-extensions-except=${process.cwd()}/extension`, `--load-extension=${process.cwd()}/extension`],
  });
  const worker = ctx.serviceWorkers()[0] ?? await ctx.waitForEvent('serviceworker', { timeout: 10000 });
  const extensionId = new URL(worker.url()).host;
  return {
    ctx,
    extensionId,
    surfaceUrl: `chrome-extension://${extensionId}/dist/search/surface-shell.html`,
    dir,
  };
}

export async function closeSurfaceHarness(harness: SurfaceHarness): Promise<void> {
  await harness.ctx.close();
  await rm(harness.dir, { recursive: true, force: true, maxRetries: 5, retryDelay: 200 });
}

export async function readUiFixture(name: string): Promise<UiFixture> {
  return JSON.parse(await readFile(`fixtures/ui/phase-13/${name}.json`, 'utf8')) as UiFixture;
}

export async function openSurface(harness: SurfaceHarness, fixture: UiFixture): Promise<Page> {
  const page = await harness.ctx.newPage();
  await page.addInitScript((value: UiFixture) => {
    type DriverRow = {
      resultId: string;
      tabId: number;
      windowId: number;
      groupId?: number;
      title: string;
      domain: string;
      windowLabel?: string;
      groupLabel?: string;
      pinned: boolean;
      score?: number;
      bestClass?: string;
      explanations: string[];
    };
    type DriverModel = {
      profileId: string;
      sequence: number;
      status: string;
      rows: DriverRow[];
      projectionRevision: number | null;
      projectionEpoch?: string | null;
      persistenceState?: string;
      indexState?: string;
      error?: { source: 'host' | 'client'; code: string; retryable: boolean };
    };
    type Pending = {
      sequence: number;
      response: UiResponse;
      resolve: (model: DriverModel) => void;
    };
    const listeners = new Set<(model: DriverModel) => void>();
    const state = {
      queries: [] as Array<{ query: string }>,
      activations: [] as Array<{
        resultId: string;
        tabId: number;
        windowId: number;
        projectionRevision: number;
        projectionEpoch: string;
        querySequence: number;
      }>,
      dismisses: 0,
      sequence: 0,
      pending: [] as Pending[],
    };
    const baseModel = (sequence: number, status: string, rows: DriverRow[] = [], response: UiResponse = { status }): DriverModel => ({
      profileId: value.profile_id,
      sequence,
      status,
      rows,
      projectionRevision: response.projection_revision ?? value.projection_revision ?? 1,
      projectionEpoch: response.projection_epoch
        ?? value.projection_epoch
        ?? '123e4567-e89b-42d3-a456-426614174099',
      persistenceState: response.persistence_state,
      indexState: response.index_state,
      error: response.error,
    });
    const rowsFor = (response: UiResponse): DriverRow[] => (response.rows ?? []).map(row => ({
      resultId: row.result_id,
      tabId: row.tab_id,
      windowId: row.window_id,
      groupId: row.group_id,
      title: row.title,
      domain: row.domain,
      windowLabel: row.window_label,
      groupLabel: row.group_label,
      pinned: row.pinned === true,
      score: row.score,
      bestClass: row.best_class,
      explanations: row.explanations ?? [],
    }));
    let model = baseModel(0, 'idle');
    const emit = (next: DriverModel): void => {
      model = next;
      for (const listener of listeners) listener(next);
    };
    const resultModel = (sequence: number, response: UiResponse): DriverModel =>
      baseModel(sequence, response.status, rowsFor(response), response);
    const source = {
      view: () => model,
      subscribe(listener: (next: DriverModel) => void) {
        listeners.add(listener);
        return () => { listeners.delete(listener); };
      },
      search(input: { query: string }) {
        state.queries.push({ query: input.query });
        const sequence = ++state.sequence;
        emit(baseModel(sequence, 'loading'));
        const response = value.responses?.[state.queries.length - 1]
          ?? value.responses?.[value.responses.length - 1]
          ?? { status: 'no_results' };
        if (value.defer_responses === true || response.defer === true) {
          return new Promise<DriverModel>(resolve => {
            state.pending.push({ sequence, response, resolve });
          });
        }
        const result = resultModel(sequence, response);
        emit(result);
        return Promise.resolve(result);
      },
      cancel() {
        const sequence = ++state.sequence;
        const cancelled = baseModel(sequence, 'cancelled');
        emit(cancelled);
        return cancelled;
      },
    };
    const activation = {
      activate: async (request: {
        resultId: string;
        tabId: number;
        windowId: number;
        projectionRevision: number;
        projectionEpoch: string;
        querySequence: number;
      }) => {
        state.activations.push(request);
        const configured = value.activation;
        const ok = configured?.ok ?? true;
        const confirmed = configured?.confirmed ?? true;
        return {
          ok,
          confirmed,
          browserState: configured?.browser_state
            ?? (ok && confirmed ? 'tab_active' : confirmed ? 'unchanged' : 'unknown'),
          nextAction: configured?.next_action ?? (ok ? 'dismiss' : 'retry'),
          errorCode: configured?.error_code,
          reportState: 'not_configured' as const,
        };
      },
    };
    const dismiss = () => {
      state.dismisses += 1;
      document.documentElement.dataset.dismissed = 'true';
    };
    const testState = state as typeof state & {
      resolve(index: number, response: UiResponse): void;
    };
    testState.resolve = (index: number, response: UiResponse) => {
      const pending = state.pending[index];
      if (pending === undefined) return;
      const result = resultModel(pending.sequence, response);
      emit(result);
      pending.resolve(result);
      state.pending.splice(index, 1);
    };
    (window as unknown as {
      __INFOBOARD_SURFACE_TEST__: {
        context: { browserFamily: 'chrome'; profileId: string; contextKind: 'normal' | 'private' };
        querySource: typeof source;
        activation: typeof activation;
        dismiss: () => void;
        privateUnavailable?: boolean;
        commandStartedAt?: number;
      };
      __INFOBOARD_SURFACE_TEST_STATE__: typeof testState;
    }).__INFOBOARD_SURFACE_TEST__ = {
      context: {
        browserFamily: 'chrome',
        profileId: value.profile_id,
        contextKind: value.context_kind,
      },
      querySource: source,
      activation,
      dismiss,
      privateUnavailable: value.private_unavailable,
      commandStartedAt: value.command_started_at,
    };
    (window as unknown as { __INFOBOARD_SURFACE_TEST_STATE__: typeof testState }).__INFOBOARD_SURFACE_TEST_STATE__ = testState;
  }, fixture);
  await page.goto(harness.surfaceUrl);
  await page.waitForFunction(() => document.documentElement.dataset.surfaceState !== 'opening');
  return page;
}

export async function surfaceState(page: Page): Promise<string | undefined> {
  return page.evaluate(() => document.documentElement.dataset.surfaceState);
}

export async function driverState(page: Page): Promise<{
  queries: Array<{ query: string }>;
  activations: Array<{
    resultId: string;
    tabId: number;
    windowId: number;
    projectionRevision: number;
    projectionEpoch: string;
    querySequence: number;
  }>;
  dismisses: number;
  pending: unknown[];
}> {
  return page.evaluate(() => {
    const state = (window as unknown as {
      __INFOBOARD_SURFACE_TEST_STATE__: {
        queries: Array<{ query: string }>;
        activations: Array<{
          resultId: string;
          tabId: number;
          windowId: number;
          projectionRevision: number;
          projectionEpoch: string;
          querySequence: number;
        }>;
        dismisses: number;
        pending: unknown[];
      };
    }).__INFOBOARD_SURFACE_TEST_STATE__;
    return {
      queries: state.queries,
      activations: state.activations,
      dismisses: state.dismisses,
      pending: state.pending,
    };
  });
}
