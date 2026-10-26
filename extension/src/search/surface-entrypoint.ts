import { ChromeBrowserAdapter, type BrowserProfileContext } from '../browser/browser-adapter.js';
import { parseProfileID } from '../../domain/index.js';
import { createNativeSearchQuerySource } from './search-query-source.js';
import {
  SurfaceController,
  type ActivationHandoff,
  type SearchQuerySource,
} from './surface-controller.js';
import { findSurfaceElements } from './surface-renderer.js';
import type { QueryRenderModel } from '../runtime/query-client.js';

const adapter = new ChromeBrowserAdapter();
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
    activate: async () => ({ ok: false, confirmed: true, errorCode: 'activation_unavailable' }),
  };
}

export async function initializeSearchSurface(doc: Document = document): Promise<void> {
  const driver = (globalThis as typeof globalThis & { __INFOBOARD_SURFACE_TEST__?: SurfaceTestDriver }).__INFOBOARD_SURFACE_TEST__;
  if (driver) {
    await initializeWithDriver(doc, driver);
    return;
  }

  const context = await adapter.currentProfileContext();
  if (!context.ok) { reportError(doc, 'profile-unavailable'); return; }
  if (!findSurfaceElements(doc)) { reportError(doc, 'query-input-missing'); return; }

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
    });
    controller.start();
    if (!access.ok || !access.value) controller.showPrivateUnavailable();
    adapter.registerPopupCloseListener(() => controller.dismiss());
    return;
  }

  const runtime = createNativeSearchQuerySource(context.value);
  const controller = new SurfaceController({
    doc,
    querySource: runtime.source,
    activation: noopActivation(),
    dismiss: () => doc.defaultView?.close(),
    commandStartedAt,
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
