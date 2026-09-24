export function registerSurfaceDismissal(doc: Document): void {
  const close = () => { doc.defaultView?.close(); };
  doc.querySelector<HTMLInputElement>('[data-search-input]')?.addEventListener('keydown', event => {
    if (event.key === 'Escape') {
      event.preventDefault();
      close();
    }
  });
  doc.querySelector('[data-close-search]')?.addEventListener('click', close);
}
