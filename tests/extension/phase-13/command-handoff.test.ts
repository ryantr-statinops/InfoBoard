import { expect, test } from '@playwright/test';
import { createCommandHandler } from '../../../extension/src/background/open-search-command.js';
import type { BrowserAdapter, BrowserResult } from '../../../extension/src/browser/browser-adapter.js';
import { parseProfileID } from '../../../extension/domain/index.js';

function adapter(overrides: Partial<BrowserAdapter> = {}): BrowserAdapter {
  return {
    currentProfileContext: async () => ({
      ok: true,
      value: {
        browserFamily: 'chrome',
        profileId: parseProfileID('123e4567-e89b-42d3-a456-426614174000'),
        contextKind: 'normal',
      },
    }),
    openSearchSurface: async (): Promise<BrowserResult<void>> => ({ ok: true, value: undefined }),
    closeSearchSurface: async (): Promise<BrowserResult<void>> => ({ ok: true, value: undefined }),
    privateContextAccess: async () => ({ ok: true, value: false }),
    getOpenTabSnapshot: async () => ({ ok: false, error: { kind: 'unsupported', retryable: false } }),
    registerTabObservationListeners: () => ({ ok: false, error: { kind: 'unsupported', retryable: false } }),
    registerCommandListener() {},
    registerLifecycleListeners() {},
    registerPopupCloseListener() {},
    ...overrides,
  };
}

test('one command open records one start timestamp even for duplicate dispatches', async () => {
  const recorded: number[] = [];
  const opening = Promise.withResolvers<BrowserResult<void>>();
  const started = Promise.withResolvers<void>();
  const handle = createCommandHandler(adapter({
    openSearchSurface: () => {
      started.resolve();
      return opening.promise;
    },
  }), { now: () => 1234, recordCommandStart: at => { recorded.push(at); } });

  const first = handle('open-search');
  const duplicate = handle('open-search');
  await started.promise;
  expect(recorded).toEqual([1234]);
  opening.resolve({ ok: true, value: undefined });
  await Promise.all([first, duplicate]);
  expect(recorded).toEqual([1234]);
});

test('close and unknown commands never record a command-start timestamp', async () => {
  const recorded: number[] = [];
  const handle = createCommandHandler(adapter(), {
    now: () => 99,
    recordCommandStart: at => { recorded.push(at); },
  });
  await handle('close-search');
  await handle('unknown');
  expect(recorded).toEqual([]);
});

test('a timing-store failure does not block the keyboard hot path', async () => {
  let opened = false;
  const handle = createCommandHandler(adapter({
    openSearchSurface: async () => {
      opened = true;
      return { ok: true, value: undefined };
    },
  }), { recordCommandStart: () => { throw new Error('storage unavailable'); } });

  await handle('open-search');
  expect(opened).toBe(true);
});
