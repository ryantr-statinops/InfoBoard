import type { BrowserAdapter } from '../browser/browser-adapter.js';

export function createCommandHandler(adapter: BrowserAdapter): (command: string) => Promise<void> {
  let inFlight: Promise<void> | undefined;
  return async command => {
    if (command !== 'open-search') return;
    if (inFlight) return inFlight;
    inFlight = adapter.openSearchSurface().then(() => undefined);
    try { await inFlight; } finally { inFlight = undefined; }
  };
}
