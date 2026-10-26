import type { BrowserAdapter } from '../browser/browser-adapter.js';

export interface CommandHandlerOptions {
  now?: () => number;
  recordCommandStart?: (startedAt: number) => void | Promise<void>;
}

export function createCommandHandler(
  adapter: BrowserAdapter,
  options: CommandHandlerOptions = {},
): (command: string) => Promise<void> {
  const now = options.now ?? (() => Date.now());
  let inFlight: Promise<void> | undefined;
  return async command => {
    if (command === 'close-search') { await adapter.closeSearchSurface(); return; }
    if (command !== 'open-search') return;
    if (inFlight) return inFlight;
    inFlight = (async () => {
      const startedAt = now();
      try {
        await options.recordCommandStart?.(startedAt);
      } catch {
        // Timing is supporting evidence; a storage failure must not block the
        // keyboard hot path.
      }
      await adapter.openSearchSurface();
    })();
    try { await inFlight; } finally { inFlight = undefined; }
  };
}
