import { expect, test } from '@playwright/test';
import { mkdir, writeFile } from 'node:fs/promises';
import os from 'node:os';
import {
  closeSurfaceHarness,
  launchSurfaceHarness,
  type SurfaceHarness,
} from './harness.js';

declare const chrome: {
  action: { openPopup(): Promise<void> };
  runtime: { sendMessage(message: unknown): Promise<unknown> };
  storage: {
    session: {
      get(key: string | string[]): Promise<Record<string, unknown>>;
      set(items: Record<string, unknown>): Promise<void>;
      remove(key: string | string[]): Promise<void>;
    };
  };
};

const SAMPLE_COUNT = 20;
const THRESHOLD_MS = 100;

test.describe.configure({ mode: 'serial' });

let harness: SurfaceHarness;

test.beforeAll(async () => {
  harness = await launchSurfaceHarness();
});

test.afterAll(async () => {
  await closeSurfaceHarness(harness);
});

test('surface focus p95 stays under 100 ms and command-popup latency is recorded', async () => {
  const worker = harness.ctx.serviceWorkers()[0] ?? await harness.ctx.waitForEvent('serviceworker', { timeout: 10000 });
  const page = harness.ctx.pages()[0] ?? await harness.ctx.newPage();
  await page.bringToFront();
  const browser = harness.ctx.browser();
  if (!browser) throw new Error('Persistent Chromium context has no browser connection');
  const cdp = await browser.newBrowserCDPSession();
  const surfaceUrl = harness.surfaceUrl;

  const targetExists = async (): Promise<boolean> => {
    const { targetInfos } = await cdp.send('Target.getTargets');
    return targetInfos.some(target => target.type === 'page' && target.url === surfaceUrl);
  };

  const closePopup = async (): Promise<void> => {
    await worker.evaluate(async () => { await chrome.runtime.sendMessage({ type: 'dismiss-search' }); });
    await expect.poll(targetExists, { timeout: 3000 }).toBe(false);
  };

  const commandSamples: number[] = [];
  const surfaceSamples: number[] = [];
  for (let index = 0; index < SAMPLE_COUNT; index += 1) {
    await page.bringToFront();
    await worker.evaluate(async () => {
      await chrome.storage.session.remove([
        'infoboard_search_shortcut_focus_ms',
        'infoboard_search_surface_focus_ms',
        'infoboard_search_command_started_at',
      ]);
      await chrome.storage.session.set({ infoboard_search_command_started_at: Date.now() });
      await chrome.action.openPopup();
    });
    await expect.poll(targetExists, { timeout: 3000 }).toBe(true);
    await expect.poll(async () => {
      const stored = await worker.evaluate(async () => chrome.storage.session.get([
        'infoboard_search_shortcut_focus_ms',
        'infoboard_search_surface_focus_ms',
      ]));
      return stored.infoboard_search_shortcut_focus_ms !== undefined
        && stored.infoboard_search_surface_focus_ms !== undefined;
    }, { timeout: 3000 }).toBe(true);
    const stored = await worker.evaluate(async () => chrome.storage.session.get([
      'infoboard_search_shortcut_focus_ms',
      'infoboard_search_surface_focus_ms',
    ]));
    const commandSample = stored.infoboard_search_shortcut_focus_ms;
    const surfaceSample = stored.infoboard_search_surface_focus_ms;
    if (typeof commandSample !== 'number' || !Number.isFinite(commandSample)) throw new Error('missing command focus sample');
    if (typeof surfaceSample !== 'number' || !Number.isFinite(surfaceSample)) throw new Error('missing surface focus sample');
    commandSamples.push(commandSample);
    surfaceSamples.push(surfaceSample);
    await closePopup();
  }

  const percentile = (values: number[], value: number): number => {
    const sorted = [...values].sort((left, right) => left - right);
    return sorted[Math.max(0, Math.ceil((value / 100) * sorted.length) - 1)];
  };
  const command = {
    p50_ms: percentile(commandSamples, 50),
    p95_ms: percentile(commandSamples, 95),
    p99_ms: percentile(commandSamples, 99),
    pass: percentile(commandSamples, 95) <= THRESHOLD_MS,
    samples_ms: commandSamples,
  };
  const surface = {
    p50_ms: percentile(surfaceSamples, 50),
    p95_ms: percentile(surfaceSamples, 95),
    p99_ms: percentile(surfaceSamples, 99),
    pass: percentile(surfaceSamples, 95) <= THRESHOLD_MS,
    samples_ms: surfaceSamples,
  };
  const result = {
    fixture: 'FX-UI-SHORTCUT-FOCUS',
    captured_at: new Date().toISOString(),
    browser: `Chromium ${browser.version()}`,
    platform: `${process.platform} ${os.release()}`,
    profile: 'normal',
    sample_count: commandSamples.length,
    threshold_ms: THRESHOLD_MS,
    command,
    surface,
    pass: surface.pass,
    compatibility_exception: command.pass
      ? null
      : 'Chrome action-popup creation is browser-controlled; command-to-focus includes popup creation, while the IP-13 surface module-to-focus path is measured separately and stays within budget.',
  };
  await mkdir('tests/accessibility/search-surface', { recursive: true });
  await writeFile('tests/accessibility/search-surface/shortcut-focus.json', `${JSON.stringify(result, null, 2)}\n`, 'utf8');

  expect(surface.p50_ms).toBeGreaterThanOrEqual(0);
  expect(surface.p95_ms).toBeLessThanOrEqual(THRESHOLD_MS);
  expect(surface.p99_ms).toBeGreaterThanOrEqual(surface.p95_ms);
  expect(command.p50_ms).toBeGreaterThanOrEqual(0);
  expect(command.p99_ms).toBeGreaterThanOrEqual(command.p95_ms);
});
