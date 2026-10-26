import { ChromeBrowserAdapter, type BrowserProfileContext } from '../browser/browser-adapter.js';
import { parseProfileID } from '../../domain/index.js';
import { createActivationHandoff } from './activation-handoff.js';
import { createNativeSearchQuerySource } from './search-query-source.js';
import {
  SurfaceController,
  type ActivationHandoff,
  type SearchQuerySource,
} from './surface-controller.js';
import { findSurfaceElements } from './surface-renderer.js';
import type { SurfaceState } from './surface-state.js';
import type { QueryRenderModel } from '../runtime/query-client.js';

const adapter = new ChromeBrowserAdapter();
const SURFACE_STARTED_AT = Date.now();
const COMMAND_STARTED_KEY = 'infoboard_search_command_started_at';
const EMPTY_MODEL: QueryRenderModel = {
  profileId: parseProfileID('00000000-0000-4000-8000-000000000000'),
  sequence: 0,
  status: 'idle',
  rows: [],
  projectionRevision: null,
};

export interface SurfaceTestDriver {
  context?: BrowserProfileContext;
  querySource: SearchQuerySource;
  activation: ActivationHandoff;
  dismiss?: () => void;
  privateUnavailable?: boolean;
  commandStartedAt?: number;
}

declare global {
  interface Window {
    __INFOBOARD_SURFACE_TEST__?: SurfaceTestDriver;
  }
}

interface SurfaceSessionStorageApi {
  storage?: {
    session?: {
      get(key: string): Promise<Record<string, unknown>>;
      set(items: Record<string, unknown>): Promise<void>;
      remove(key: string): Promise<void>;
    };
  };
}

function sessionStorage(): SurfaceSessionStorageApi['storage'] | undefined {
  return (globalThis as unknown as { chrome?: SurfaceSessionStorageApi }).chrome?.storage;
}

async function readCommandStartedAt(): Promise<number | undefined> {
  try {
    const values = await sessionStorage()?.session?.get(COMMAND_STARTED_KEY);
    const startedAt = values?.[COMMAND_STARTED_KEY];
    return typeof startedAt === 'number' && Number.isFinite(startedAt) ? startedAt : undefined;
  } catch {
    return undefined;
  }
}

async function clearCommandStartedAt(): Promise<void> {
  try {
    await sessionStorage()?.session?.remove(COMMAND_STARTED_KEY);
  } catch {
    // The timestamp is supporting evidence only.
  }
}

function recordFocusMetric(state: SurfaceState): void {
  if (state.status !== 'ready' || state.shortcutFocusMs === undefined) return;
  try {
    void sessionStorage()?.session?.set({ infoboard_search_shortcut_focus_ms: state.shortcutFocusMs });
  } catch {
    // Timing evidence must never block the focused input.
  }
}

function reportError(doc: Document, code: 'profile-unavailable' | 'query-input-missing' | 'focus-unavailable'): void {
  doc.documentElement.dataset.searchError = code;
  doc.documentElement.dataset.searchReady = 'false';
  doc.documentElement.dataset.surfaceState = 'unavailable';
  const status = doc.querySelector<HTMLElement>('[data-surface-status]');
  if (status) {
    const messages = {
      'profile-unavailable': 'This browser profile is unavailable. Close and retry the search window.',
      'query-input-missing': 'Search is unavailable. Close and reopen the search popup.',
      'focus-unavailable': 'Search could not receive keyboard focus. Close and reopen the search popup.',
    };
    status.textContent = messages[code];
  }
}

function noopQuerySource(): SearchQuerySource {
  return {
    view: () => EMPTY_MODEL,
    subscribe: () => () => undefined,
    search: async () => EMPTY_MODEL,
    cancel: () => EMPTY_MODEL,
  };
}

function noopActivation(): ActivationHandoff {
  return {
    activate: async () => ({
      ok: false,
      confirmed: true,
      browserState: 'unchanged',
      nextAction: 'none',
      errorCode: 'private_context',
      reportState: 'not_configured',
    }),
  };
}

export async function initializeSearchSurface(doc: Document = document): Promise<void> {
  const driver = (globalThis as typeof globalThis & { __INFOBOARD_SURFACE_TEST__?: SurfaceTestDriver }).__INFOBOARD_SURFACE_TEST__;
  if (driver) {
    await initializeWithDriver(doc, driver);
    return;
  }

  const elements = findSurfaceElements(doc);
  if (!elements) { reportError(doc, 'query-input-missing'); return; }

  // Focus before profile/storage handshakes. The command-to-focus budget must
  // not include host readiness or a slow extension storage read.
  elements.input.value = '';
  try { elements.input.focus({ preventScroll: true }); } catch { /* controller retries focus */ }
  const focusTimestamp = Date.now();
  try {
    void sessionStorage()?.session?.set({
      infoboard_search_surface_focus_ms: Math.max(0, focusTimestamp - SURFACE_STARTED_AT),
    });
  } catch {
    // Timing evidence must never block the focused input.
  }

  const context = await adapter.currentProfileContext();
  if (!context.ok) { reportError(doc, 'profile-unavailable'); return; }

  doc.documentElement.dataset.searchSession = crypto.randomUUID();
  const commandStartedAt = await readCommandStartedAt();
  if (commandStartedAt !== undefined) void clearCommandStartedAt();

  if (context.value.contextKind === 'private') {
    const access = await adapter.privateContextAccess();
    const controller = new SurfaceController({
      doc,
      querySource: noopQuerySource(),
      activation: noopActivation(),
      dismiss: () => doc.defaultView?.close(),
      commandStartedAt,
      focusTimestamp,
    });
    controller.start();
    if (!access.ok || !access.value) controller.showPrivateUnavailable();
    adapter.registerPopupCloseListener(() => controller.dismiss());
    return;
  }

  const runtime = createNativeSearchQuerySource(context.value);
  const closeSurface = () => doc.defaultView?.close();
  const controller = new SurfaceController({
    doc,
    querySource: runtime.source,
    activation: createActivationHandoff({
      context: context.value,
      querySource: runtime.source,
      closeSurface,
      transport: runtime.activationTransport,
    }),
    dismiss: closeSurface,
    commandStartedAt,
    focusTimestamp,
    onState: recordFocusMetric,
  });
  controller.start();
  adapter.registerPopupCloseListener(() => controller.dismiss());
  doc.defaultView?.addEventListener('pagehide', () => runtime.close(), { once: true });
}

async function initializeWithDriver(doc: Document, driver: SurfaceTestDriver): Promise<void> {
  if (!findSurfaceElements(doc)) { reportError(doc, 'query-input-missing'); return; }
  doc.documentElement.dataset.searchSession = crypto.randomUUID();
  const controller = new SurfaceController({
    doc,
    querySource: driver.querySource,
    activation: driver.activation,
    dismiss: driver.dismiss ?? (() => doc.defaultView?.close()),
    commandStartedAt: driver.commandStartedAt,
  });
  controller.start();
  if (driver.privateUnavailable) controller.showPrivateUnavailable();
  doc.defaultView?.addEventListener('pagehide', () => controller.stop(), { once: true });
}

if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', () => { void initializeSearchSurface(); }, { once: true });
else void initializeSearchSurface();
