function reportFocusError(doc: Document, code: 'query-input-missing' | 'focus-unavailable'): void {
  doc.documentElement.dataset.searchError = code;
  const status = doc.querySelector<HTMLElement>('[data-surface-status]');
  if (status) status.textContent = code === 'query-input-missing' ? 'Search is unavailable. Close and reopen the search window.' : 'Search could not receive keyboard focus. Close and reopen the search window.';
}

export function initializeSearchSurface(doc: Document = document): void {
  const input = doc.querySelector<HTMLInputElement>('[data-search-input]');
  if (!input) { reportFocusError(doc, 'query-input-missing'); return; }
  input.value = '';
  doc.documentElement.dataset.searchSession = crypto.randomUUID();
  const announceReady = () => {
    if (doc.activeElement !== input) { reportFocusError(doc, 'focus-unavailable'); return; }
    delete doc.documentElement.dataset.searchError;
    doc.documentElement.dataset.searchReady = 'true';
  };
  requestAnimationFrame(() => {
    try { input.focus({ preventScroll: true }); announceReady(); }
    catch { reportFocusError(doc, 'focus-unavailable'); }
  });
}

if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', () => initializeSearchSurface(), { once: true });
else initializeSearchSurface();
