import { ChromeBrowserAdapter } from '../browser/browser-adapter.js';
import { registerSurfaceDismissal } from './surface-dismissal.js';

const adapter = new ChromeBrowserAdapter();
const COMMAND_STARTED_KEY = 'infoboard_search_command_started_at';

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

function resetSurfaceState(doc: Document): void {
  delete doc.documentElement.dataset.searchError;
  delete doc.documentElement.dataset.searchReady;
  delete doc.documentElement.dataset.shortcutFocusMs;
  doc.documentElement.dataset.surfaceState = 'opening';
  doc.documentElement.dataset.browserStateConfirmed = 'true';
}

function reportError(doc: Document, code: 'profile-unavailable' | 'query-input-missing' | 'focus-unavailable'): void {
  doc.documentElement.dataset.searchError = code;
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

export async function initializeSearchSurface(doc: Document = document): Promise<void> {
  resetSurfaceState(doc);
  registerSurfaceDismissal(doc);
  const context = await adapter.currentProfileContext();
  if (!context.ok) { reportError(doc, 'profile-unavailable'); return; }
  const input = doc.querySelector<HTMLInputElement>('[data-search-input]');
  if (!input) { reportError(doc, 'query-input-missing'); return; }
  input.value = '';
  doc.documentElement.dataset.searchSession = crypto.randomUUID();
  const commandStartedAt = await readCommandStartedAt();
  const announceReady = () => {
    if (doc.activeElement !== input) { reportError(doc, 'focus-unavailable'); return; }
    delete doc.documentElement.dataset.searchError;
    doc.documentElement.dataset.searchReady = 'true';
    doc.documentElement.dataset.surfaceState = 'ready';
    doc.documentElement.dataset.browserStateConfirmed = 'true';
    if (commandStartedAt !== undefined) {
      const elapsed = Math.max(0, Date.now() - commandStartedAt);
      doc.documentElement.dataset.shortcutFocusMs = String(elapsed);
      void clearCommandStartedAt();
    }
  };
  requestAnimationFrame(() => {
    try { input.focus({ preventScroll: true }); announceReady(); }
    catch { reportError(doc, 'focus-unavailable'); }
  });
}

if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', () => { void initializeSearchSurface(); }, { once: true });
else void initializeSearchSurface();
