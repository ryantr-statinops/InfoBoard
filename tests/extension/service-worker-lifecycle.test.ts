import { test, expect } from '@playwright/test';
import { readFile } from 'node:fs/promises';
import { parseProfileID } from '../../extension/domain/index.js';
import { ChromeBrowserAdapter, resetProfileAfterDisconnect, type BrowserApi } from '../../extension/src/browser/browser-adapter.js';

type PopupMessageListener = (message: unknown, sender: unknown, sendResponse: (response: unknown) => void) => boolean | void;

function browserApi(readValue: unknown | null, recordWindowRead: () => void, calls: string[]): BrowserApi {
  let popupMessageListener: PopupMessageListener | undefined;
  return {
    runtime: {
      onInstalled: { addListener: () => undefined },
      onStartup: { addListener: () => undefined },
      onMessage: { addListener: listener => { popupMessageListener = listener; } },
      sendMessage: async message => {
        if (!popupMessageListener) throw new Error('No popup message listener');
        let response: unknown;
        popupMessageListener(message, {}, value => { response = value; });
        return response;
      },
    },
    commands: { onCommand: { addListener: () => undefined } },
    action: { openPopup: async options => { calls.push(`open-popup:${options?.windowId ?? 'current'}`); } },
    windows: { getLastFocused: async () => { recordWindowRead(); return { id: 1, incognito: false }; } },
    storage: { local: {
      get: async key => { calls.push(`read:${key}`); return readValue === null ? {} : { [key]: readValue }; },
      set: async values => { calls.push(`write:${Object.keys(values).join(',')}`); },
      remove: async key => { calls.push(`remove:${key}`); },
    } },
  };
}

test('missing and malformed profile identities fail closed before browser-window work', async () => {
  for (const value of ['not-an-id', null]) {
    let windowReads = 0;
    const calls: string[] = [];
    const adapter = new ChromeBrowserAdapter(browserApi(value, () => { windowReads += 1; }, calls));
    const result = await adapter.currentProfileContext();
    expect(result.ok).toBe(false);
    expect(windowReads).toBe(0);
    expect(calls).toEqual(['read:profile_id']);
  }
});

test('the action popup opens in the focused browser window without querying page tabs', async () => {
  let windowReads = 0;
  const calls: string[] = [];
  const profile = parseProfileID('123e4567-e89b-42d3-a456-426614174000');
  const adapter = new ChromeBrowserAdapter(browserApi(profile, () => { windowReads += 1; }, calls));
  const context = await adapter.currentProfileContext();
  expect(context.ok).toBe(true);
  const opened = await adapter.openSearchSurface();
  expect(opened).toEqual({ ok: true, value: undefined });
  expect(windowReads).toBe(1);
  expect(calls).toEqual(['read:profile_id', 'open-popup:current']);
});

test('close-search is forwarded only to the popup close listener', async () => {
  const calls: string[] = [];
  const adapter = new ChromeBrowserAdapter(browserApi(null, () => undefined, calls));
  let closed = false;
  adapter.registerPopupCloseListener(() => { closed = true; });
  const result = await adapter.closeSearchSurface();
  expect(result).toEqual({ ok: true, value: undefined });
  expect(closed).toBe(true);
});

test('explicit identity reset disconnects before removing the sole approved key', async () => {
  const calls: string[] = [];
  const adapter = new ChromeBrowserAdapter(browserApi(null, () => undefined, calls));
  await adapter.write(parseProfileID('123e4567-e89b-42d3-a456-426614174000'));
  await resetProfileAfterDisconnect(adapter, async () => { calls.push('disconnect'); });
  expect(calls).toEqual(['write:profile_id', 'disconnect', 'remove:profile_id']);
});

test('surface and manifest exclude page storage, network permissions, and injected scripts', async () => {
  const html = await readFile('extension/src/search/surface-shell.html', 'utf8');
  const adapter = await readFile('extension/src/browser/browser-adapter.ts', 'utf8');
  const manifest = JSON.parse(await readFile('extension/manifest.json', 'utf8'));
  expect(html).not.toMatch(/https?:\/\//);
  expect(html).not.toContain('localStorage');
  expect(html).not.toContain('sessionStorage');
  expect(adapter).not.toContain('storage.sync');
  expect(manifest.permissions).toEqual(['storage', 'windows']);
  expect(manifest.host_permissions).toBeUndefined();
  expect(manifest.action.default_popup).toBe('dist/search/surface-shell.html');
});
