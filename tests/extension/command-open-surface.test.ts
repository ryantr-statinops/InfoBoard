import { test, expect, chromium } from '@playwright/test';
import { createCommandHandler } from '../../extension/src/background/open-search-command.js';
import type { BrowserAdapter } from '../../extension/src/browser/browser-adapter.js';
import os from 'node:os';
import { mkdtemp, readFile, rm } from 'node:fs/promises';
import path from 'node:path';

async function launch(userDataDir: string) {
  const channel = process.env.BROWSER === 'edge' ? 'msedge' : 'chrome';
  return chromium.launchPersistentContext(userDataDir, {
    channel, headless: false,
    ignoreDefaultArgs: ['--disable-extensions'],
    args: [`--disable-extensions-except=${process.cwd()}/extension`, `--load-extension=${process.cwd()}/extension`],
  });
}

test('open-search creates and reuses a focused bounded surface, preserves source tab, and retains only profile_id', async () => {
  const dir = await mkdtemp(path.join(os.tmpdir(), 'infoboard-profile-'));
  const fixture = JSON.parse(await readFile('fixtures/extension/command-open-surface.json', 'utf8'));
  let ctx = await launch(dir);
  try {
    const source = ctx.pages()[0] ?? await ctx.newPage();
    const originalUrl = source.url();
    const externalRequests: string[] = [];
    ctx.on('request', request => { if (/^https?:/.test(request.url())) externalRequests.push(request.url()); });
    let worker = ctx.serviceWorkers()[0] ?? await ctx.waitForEvent('serviceworker');
    const opened = ctx.waitForEvent('page', { timeout: 10000 });
    await source.bringToFront();
    await source.keyboard.press('Control+Shift+Y');
    const surface = await opened;
    await expect(surface.locator('[data-search-input]')).toBeFocused({ timeout: 10000 });
    expect(await surface.locator('[data-search-input]').inputValue()).toBe('');
    expect(source.url()).toBe(originalUrl);
    const bounds = await surface.evaluate(async () => {
      const api = (globalThis as typeof globalThis & { chrome: any }).chrome;
      const win = await api.windows.getCurrent();
      return { width: win.width, height: win.height, ready: document.documentElement.dataset.searchReady };
    });
    expect(bounds).toEqual({ width: fixture.surface.width, height: fixture.surface.height, ready: 'true' });
    const state = await worker.evaluate(async () => (globalThis as typeof globalThis & { chrome: any }).chrome.storage.local.get(null));
    expect(Object.keys(state)).toEqual(['profile_id']);
    expect(state.profile_id).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);
    const beforePages = ctx.pages().length;
    await source.bringToFront();
    await source.keyboard.press('Control+Shift+Y');
    await expect.poll(() => ctx.pages().length).toBe(beforePages);
    await surface.keyboard.press('Escape');
    await expect.poll(() => ctx.pages().includes(surface)).toBe(false);
    expect(source.url()).toBe(originalUrl);
    expect(externalRequests).toEqual([]);
  } finally { await ctx.close(); await rm(dir, { recursive: true, force: true, maxRetries: 5, retryDelay: 200 }); }
});

test('single-flight dispatch creates once and returns to existing-surface focus', async () => {
  const calls: string[] = [];
  let release!: (result: any) => void;
  const adapter = {
    extensionUrl: (value: string) => `extension://id/${value}`,
    currentProfileContext: async () => { calls.push('context'); return { ok: true as const, value: { browserFamily: 'chrome' as const, profileId: '123e4567-e89b-42d3-a456-426614174000' as any, contextKind: 'normal' as const } }; },
    listSearchSurfaces: async () => { calls.push('list'); return { ok: true as const, value: [] }; },
    createSearchSurface: async (url: string, bounds: { width: 480; height: 600 }) => { calls.push(`create:${url}:${bounds.width}x${bounds.height}`); return new Promise(resolve => { release = resolve; }); },
    focusSearchSurface: async (windowId: number, tabId: number) => { calls.push(`focus:${windowId}:${tabId}`); return { ok: true as const, value: { windowId, tabId } }; },
    closeSearchSurface: async () => ({ ok: true as const, value: undefined }),
    registerCommandListener() {}, registerLifecycleListeners() {}, registerMessageListener() {},
  } as BrowserAdapter;
  const handle = createCommandHandler(adapter);
  const first = handle('open-search');
  const duplicate = handle('open-search');
  await new Promise(resolve => setTimeout(resolve, 0));
  expect(calls).toEqual(['context', 'list', 'create:extension://id/dist/search/surface-shell.html:480x600']);
  release({ ok: true, value: { windowId: 4, tabId: 9 } });
  await Promise.all([first, duplicate]);
  expect(calls.filter(value => value.startsWith('create:'))).toHaveLength(1);
  const existing = { ...adapter, listSearchSurfaces: async () => ({ ok: true as const, value: [{ windowId: 7, tabId: 11 }] }), createSearchSurface: async () => { throw new Error('unexpected create'); } } as BrowserAdapter;
  await createCommandHandler(existing)('open-search');
  expect(calls.at(-1)).toBe('focus:7:11');
  await handle('unknown');
  expect(calls.at(-1)).toBe('focus:7:11');
});
