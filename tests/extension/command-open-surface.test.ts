import { test, expect, chromium } from '@playwright/test';
import { createCommandHandler } from '../../extension/src/background/open-search-command.js';
import { parseProfileID } from '../../extension/domain/index.js';
import type { BrowserAdapter, BrowserResult } from '../../extension/src/browser/browser-adapter.js';
import os from 'node:os';
import { mkdtemp, readFile, rm } from 'node:fs/promises';
import path from 'node:path';

declare const chrome: {
  action: { openPopup(): Promise<void> };
  runtime: { sendMessage(message: unknown): Promise<unknown> };
  storage: { local: { get(key: string): Promise<Record<string, unknown>> } };
};

async function launch(userDataDir: string, viewport: { width: number; height: number }) {
  return chromium.launchPersistentContext(userDataDir, {
    channel: 'chromium',
    headless: false,
    viewport,
    args: [`--disable-extensions-except=${process.cwd()}/extension`, `--load-extension=${process.cwd()}/extension`],
  });
}

test('extension search surface initializes and handles close signals in a 480x600 browser context', async () => {
  const dir = await mkdtemp(path.join(os.tmpdir(), 'infoboard-profile-'));
  const fixture = JSON.parse(await readFile('fixtures/extension/command-open-surface.json', 'utf8'));
  const ctx = await launch(dir, fixture.surface);
  try {
    const source = ctx.pages()[0] ?? await ctx.newPage();
    const externalRequests: string[] = [];
    ctx.on('request', request => { if (/^https?:/.test(request.url())) externalRequests.push(request.url()); });
    const worker = ctx.serviceWorkers()[0] ?? await ctx.waitForEvent('serviceworker', { timeout: 10000 });
    expect(worker.url()).toMatch(/^chrome-extension:\/\/[a-p]{32}\/dist\/src\/background\/service-worker\.js$/);
    await expect.poll(() => worker.evaluate(async () => chrome.storage.local.get('profile_id'))).toHaveProperty('profile_id');
    const extensionId = new URL(worker.url()).host;
    const surfaceUrl = `chrome-extension://${extensionId}/dist/search/surface-shell.html`;
    const browser = ctx.browser();
    if (!browser) throw new Error('Persistent Chromium context has no browser connection');
    const cdp = await browser.newBrowserCDPSession();
    const selectedTabUrl = source.url();
    await worker.evaluate(async () => chrome.action.openPopup());
    await expect.poll(async () => {
      const { targetInfos } = await cdp.send('Target.getTargets');
      return targetInfos.some(target => target.type === 'page' && target.url === surfaceUrl);
    }).toBe(true);
    expect(source.url()).toBe(selectedTabUrl);
    await source.goto(surfaceUrl);
    const originalUrl = source.url();
    await expect.poll(() => source.evaluate(() => ({ focused: document.activeElement === document.querySelector('[data-search-input]'), ready: document.documentElement.dataset.searchReady }))).toEqual({ focused: true, ready: 'true' });
    expect(await source.locator('[data-search-input]').inputValue()).toBe('');
    await expect(source.getByLabel('Search open tabs')).toBeVisible();
    await expect(source.locator('[data-search-input]')).toBeFocused();
    expect(source.url()).toBe(originalUrl);
    const bounds = await source.evaluate(() => ({ width: innerWidth, height: innerHeight, ready: document.documentElement.dataset.searchReady }));
    expect(bounds).toEqual({ width: fixture.surface.width, height: fixture.surface.height, ready: 'true' });
    const stored = await worker.evaluate(async () => chrome.storage.local.get('profile_id'));
    expect(Object.keys(stored)).toEqual(['profile_id']);
    expect(stored.profile_id).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/);

    await source.evaluate(() => { window.close = () => { document.documentElement.dataset.closeRequested = 'true'; }; });
    await source.keyboard.press('Escape');
    await expect(source.locator('html')).toHaveAttribute('data-close-requested', 'true');
    await source.evaluate(() => { delete document.documentElement.dataset.closeRequested; });
    await source.locator('[data-close-search]').click();
    await expect(source.locator('html')).toHaveAttribute('data-close-requested', 'true');
    await source.evaluate(() => { delete document.documentElement.dataset.closeRequested; });
    await worker.evaluate(async () => chrome.runtime.sendMessage({ type: 'dismiss-search' }));
    await expect(source.locator('html')).toHaveAttribute('data-close-requested', 'true');
    expect(source.url()).toBe(originalUrl);
    expect(ctx.pages()).toEqual([source]);
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
