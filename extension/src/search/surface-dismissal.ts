declare const chrome: { runtime: { sendMessage(message: unknown): Promise<unknown> } };

export function registerSurfaceDismissal(doc: Document): void {
  const dismiss = () => { void chrome.runtime.sendMessage({ type: 'dismiss-search' }); };
  doc.querySelector<HTMLInputElement>('[data-search-input]')?.addEventListener('keydown', event => {
    if (event.key === 'Escape') {
      event.preventDefault();
      dismiss();
    }
  });
  doc.querySelector('[data-close-search]')?.addEventListener('click', dismiss);
}
