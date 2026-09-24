import { ChromeBrowserAdapter } from '../browser/browser-adapter.js';
import { registerSurfaceDismissal } from './surface-dismissal.js';

const adapter = new ChromeBrowserAdapter();

function reportError(doc: Document, code: 'profile-unavailable' | 'query-input-missing' | 'focus-unavailable'): void {
  doc.documentElement.dataset.searchError = code;
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
  const context = await adapter.currentProfileContext();
  if (!context.ok) { reportError(doc, 'profile-unavailable'); return; }
  const input = doc.querySelector<HTMLInputElement>('[data-search-input]');
  if (!input) { reportError(doc, 'query-input-missing'); return; }
  input.value = '';
  registerSurfaceDismissal(doc);
  doc.documentElement.dataset.searchSession = crypto.randomUUID();
  const announceReady = () => {
    if (doc.activeElement !== input) { reportError(doc, 'focus-unavailable'); return; }
    delete doc.documentElement.dataset.searchError;
    doc.documentElement.dataset.searchReady = 'true';
  };
  requestAnimationFrame(() => {
    try { input.focus({ preventScroll: true }); announceReady(); }
    catch { reportError(doc, 'focus-unavailable'); }
  });
}

if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', () => { void initializeSearchSurface(); }, { once: true });
else void initializeSearchSurface();
