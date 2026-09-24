import { test, expect, chromium } from '@playwright/test';
import { createCommandHandler } from '../../extension/src/background/open-search-command.js';
import type { BrowserAdapter, BrowserResult } from '../../extension/src/browser/browser-adapter.js';
import os from 'node:os';
import { mkdtemp, readFile, rm } from 'node:fs/promises';
import path from 'node:path';

async function launch(userDataDir: string) {
  const channel = process.env.BROWSER === 'edge' ? 'msedge' : 'chrome';
  return chromium.launchPersistentContext(userDataDir, {
    channel,
    headless: false,
    ignoreDefaultArgs: ['--disable-extensions'],
    args: [`--disable-extensions-except=${process.cwd()}/extension`, `--load-extension=${process.cwd()}/extension`],
  });
}

test('open-search opens the browser action popup, validates profile, and preserves the active tab', async () => {
  const dir = await mkdtemp(path.join(os.tmpdir(), 'infoboard-profile-'));
  const fixture = JSON.parse(await readFile('fixtures/extension/command-open-surface.json', 'utf8'));
  const ctx = await launch(dir);
  try {
    const source = ctx.pages()[0] ?? await ctx.newPage();
    const originalUrl = source.url();
    const externalRequests: string[] = [];
    ctx.on('request', request => { if (/^https?:/.test(request.url())) externalRequests.push(request.url()); });
    const worker = ctx.serviceWorkers()[0] ?? await ctx.waitForEvent('serviceworker', { timeout: 10000 });

    const firstPopup = ctx.waitForEvent('page', { timeout: 10000 });
    await source.bringToFront();
    await source.keyboard.press('Control+Shift+Y');
    let popup = await firstPopup;
    await expect(popup.locator('[data-search-input]')).toBeFocused({ timeout: 10000 });
    expect(await popup.locator('[data-search-input]').inputValue()).toBe('');
    expect(source.url()).toBe(originalUrl);
    const bounds = await popup.evaluate(() => ({ width: innerWidth, height: innerHeight, ready: document.documentElement.dataset.searchReady }));
    expect(bounds).toEqual({ width: fixture.surface.width, height: fixture.surface.height, ready: 'true' });
    const stored = await worker.evaluate(async () => (globalThis as typeof globalThis & { chrome: { storage: { local: { get(key: string): Promise<Record<string, unknown>> } } } }).chrome.storage.local.get('profile_id'));
    expect(Object.keys(stored)).toEqual(['profile_id']);
    expect(stored.profile_id).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);

    await popup.keyboard.press('Escape');
    await expect.poll(() => ctx.pages().includes(popup)).toBe(false);
    expect(source.url()).toBe(originalUrl);

    const reopenedPopup = ctx.waitForEvent('page', { timeout: 10000 });
    await source.bringToFront();
    await source.keyboard.press('Control+Shift+Y');
    popup = await reopenedPopup;
    await expect(popup.locator('[data-search-input]')).toBeFocused({ timeout: 10000 });
    expect(await popup.locator('[data-search-input]').inputValue()).toBe('');
    await popup.locator('[data-close-search]').click();
    await expect.poll(() => ctx.pages().includes(popup)).toBe(false);
    expect(source.url()).toBe(originalUrl);
    expect(externalRequests).toEqual([]);
  } finally {
    await ctx.close();
    await rm(dir, { recursive: true, force: true, maxRetries: 5, retryDelay: 200 });
  }
});

test('single-flight command opens the action popup only once', async () => {
  const calls: string[] = [];
  const opening = Promise.withResolvers<BrowserResult<void>>();
  const started = Promise.withResolvers<void>();
  const adapter: BrowserAdapter = {
    currentProfileContext: async () => { throw new Error('profile is checked by the popup before readiness'); },
    openSearchSurface: () => { calls.push('open'); started.resolve(); return opening.promise; },
    registerCommandListener() {},
    registerLifecycleListeners() {},
  };
  const handle = createCommandHandler(adapter);
  const first = handle('open-search');
  const duplicate = handle('open-search');
  await started.promise;
  expect(calls).toEqual(['open']);
  opening.resolve({ ok: true, value: undefined });
  await Promise.all([first, duplicate]);
  await handle('unknown');
  expect(calls).toEqual(['open']);
});
