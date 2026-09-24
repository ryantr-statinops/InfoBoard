import { test, expect, chromium } from '@playwright/test';
import type { BrowserContext, Page } from '@playwright/test';
import { createCommandHandler } from '../../extension/src/background/open-search-command.js';
import { parseProfileID } from '../../extension/domain/index.js';
import type { BrowserAdapter, BrowserResult } from '../../extension/src/browser/browser-adapter.js';
import os from 'node:os';
import { mkdtemp, readFile, rm } from 'node:fs/promises';
import path from 'node:path';

declare const chrome: { storage: { local: { get(key: string): Promise<Record<string, unknown>> } } };

async function launch(userDataDir: string) {
  const channel = process.env.BROWSER === 'edge' ? 'msedge' : 'chrome';
  return chromium.launchPersistentContext(userDataDir, {
    channel,
    headless: false,
    ignoreDefaultArgs: ['--disable-extensions'],
    args: [`--disable-extensions-except=${process.cwd()}/extension`, `--load-extension=${process.cwd()}/extension`],
  });
}

async function openPopupFromShortcut(page: Page, context: BrowserContext) {
  const popupEvent = context.waitForEvent('page', { timeout: 10000 });
  await page.bringToFront();
  await page.keyboard.press('Control+Shift+Y');
  return popupEvent;
}

test('open-search popup closes from its shortcut, Escape, or the close control', async () => {
  const dir = await mkdtemp(path.join(os.tmpdir(), 'infoboard-profile-'));
  const fixture = JSON.parse(await readFile('fixtures/extension/command-open-surface.json', 'utf8'));
  const ctx = await launch(dir);
  try {
    const source = ctx.pages()[0] ?? await ctx.newPage();
    const originalUrl = source.url();
    const externalRequests: string[] = [];
    ctx.on('request', request => { if (/^https?:/.test(request.url())) externalRequests.push(request.url()); });
    const worker = ctx.serviceWorkers()[0] ?? await ctx.waitForEvent('serviceworker', { timeout: 10000 });

    let popup = await openPopupFromShortcut(source, ctx);
    await expect(popup.locator('[data-search-input]')).toBeFocused({ timeout: 10000 });
    expect(await popup.locator('[data-search-input]').inputValue()).toBe('');
    expect(source.url()).toBe(originalUrl);
    const bounds = await popup.evaluate(() => ({ width: innerWidth, height: innerHeight, ready: document.documentElement.dataset.searchReady }));
    expect(bounds).toEqual({ width: fixture.surface.width, height: fixture.surface.height, ready: 'true' });
    const stored = await worker.evaluate(async () => chrome.storage.local.get('profile_id'));
    expect(Object.keys(stored)).toEqual(['profile_id']);
    expect(stored.profile_id).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);

    await popup.keyboard.press('Control+Shift+X');
    await expect.poll(() => ctx.pages().includes(popup)).toBe(false);
    expect(source.url()).toBe(originalUrl);

    popup = await openPopupFromShortcut(source, ctx);
    await expect(popup.locator('[data-search-input]')).toBeFocused({ timeout: 10000 });
    await popup.keyboard.press('Escape');
    await expect.poll(() => ctx.pages().includes(popup)).toBe(false);

    popup = await openPopupFromShortcut(source, ctx);
    await popup.locator('[data-close-search]').click();
    await expect.poll(() => ctx.pages().includes(popup)).toBe(false);
    expect(source.url()).toBe(originalUrl);
    expect(externalRequests).toEqual([]);
  } finally {
    await ctx.close();
    await rm(dir, { recursive: true, force: true, maxRetries: 5, retryDelay: 200 });
  }
});

test('single-flight command opens and closes the action popup only once', async () => {
  const calls: string[] = [];
  const opening = Promise.withResolvers<BrowserResult<void>>();
  const started = Promise.withResolvers<void>();
  const adapter: BrowserAdapter = {
    currentProfileContext: async () => ({ ok: true, value: { browserFamily: 'chrome', profileId: parseProfileID('123e4567-e89b-42d3-a456-426614174000'), contextKind: 'normal' } }),
    openSearchSurface: () => { calls.push('open'); started.resolve(); return opening.promise; },
    closeSearchSurface: async () => { calls.push('close'); return { ok: true, value: undefined }; },
    registerCommandListener() {},
    registerLifecycleListeners() {},
    registerPopupCloseListener() {},
  };
  const handle = createCommandHandler(adapter);
  const first = handle('open-search');
  const duplicate = handle('open-search');
  await started.promise;
  expect(calls).toEqual(['open']);
  opening.resolve({ ok: true, value: undefined });
  await Promise.all([first, duplicate]);
  await handle('close-search');
  expect(calls).toEqual(['open', 'close']);
  await handle('unknown');
  expect(calls).toEqual(['open', 'close']);
});
