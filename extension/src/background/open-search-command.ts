import type { BrowserAdapter } from '../browser/browser-adapter.js';

const SURFACE_PATH = 'dist/search/surface-shell.html';
const BOUNDS = { width: 480, height: 600 } as const;

export function createCommandHandler(adapter: BrowserAdapter): (command: string) => Promise<void> {
  let inFlight: Promise<void> | undefined;
  return async command => {
    if (command !== 'open-search') return;
    if (inFlight) return inFlight;
    inFlight = (async () => {
      const context = await adapter.currentProfileContext();
      if (!context.ok) return;
      const surfaces = await adapter.listSearchSurfaces();
      if (!surfaces.ok) return;
      const existing = surfaces.value[0];
      if (existing) { await adapter.focusSearchSurface(existing.windowId, existing.tabId); return; }
      await adapter.createSearchSurface(adapter.extensionUrl(SURFACE_PATH), BOUNDS);
    })();
    try { await inFlight; } finally { inFlight = undefined; }
  };
}
