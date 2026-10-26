import { expect, test } from '@playwright/test';
import { readFile, readdir } from 'node:fs/promises';
import path from 'node:path';
import {
  closeSurfaceHarness,
  launchSurfaceHarness,
  openSurface,
  readUiFixture,
  type SurfaceHarness,
} from './harness.js';

declare const chrome: {
  tabs: {
    query(queryInfo: Record<string, unknown>): Promise<Array<{
      id?: number;
      url?: string;
      title?: string;
      pinned: boolean;
      active: boolean;
    }>>;
  };
};

test.describe.configure({ mode: 'serial' });

let harness: SurfaceHarness;

test.beforeAll(async () => {
  harness = await launchSurfaceHarness();
});

test.afterAll(async () => {
  await closeSurfaceHarness(harness);
});

async function readExtensionSources(): Promise<string> {
  const files: string[] = [];
  const walk = async (directory: string): Promise<void> => {
    for (const entry of await readdir(directory, { withFileTypes: true })) {
      const fullPath = path.join(directory, entry.name);
      if (entry.isDirectory()) await walk(fullPath);
      else if (/\.(?:ts|html)$/.test(entry.name)) files.push(fullPath);
    }
  };
  await walk('extension/src');
  return (await Promise.all(files.map(file => readFile(file, 'utf8')))).join('\n');
}

test('manifest keeps the approved permissions and extension sources make no network or browser-state calls', async () => {
  const manifest = JSON.parse(await readFile('extension/manifest.json', 'utf8')) as {
    permissions?: string[];
    host_permissions?: string[];
    optional_permissions?: string[];
  };
  expect([...(manifest.permissions ?? [])].sort()).toEqual([
    'nativeMessaging',
    'storage',
    'tabGroups',
    'tabs',
  ]);
  expect(manifest.host_permissions ?? []).toEqual([]);
  expect(manifest.optional_permissions ?? []).toEqual([]);

  const source = await readExtensionSources();
  expect(source).not.toMatch(/\bfetch\s*\(/);
  expect(source).not.toMatch(/\bnew\s+XMLHttpRequest\b/);
  expect(source).not.toMatch(/\bnew\s+WebSocket\b/);
  expect(source).not.toMatch(/\bnew\s+EventSource\b/);
  expect(source).not.toMatch(/chrome\.tabs\.(?:update|create|remove|move|discard|reload)\s*\(/);
  expect(source).not.toMatch(/chrome\.windows\.(?:create|update|remove)\s*\(/);
  expect(source).not.toMatch(/chrome\.(?:scripting|cookies|history|webRequest|debugger)\b/);

  const entrypoint = await readFile('extension/src/search/surface-entrypoint.ts', 'utf8');
  expect(entrypoint).toContain('activation_unavailable');
  expect(entrypoint).not.toMatch(/chrome\.tabs\b/);
});

test('typing and Escape leave existing browser tabs unchanged and do not activate', async () => {
  const worker = harness.ctx.serviceWorkers()[0] ?? await harness.ctx.waitForEvent('serviceworker', { timeout: 10000 });
  const source = await harness.ctx.newPage();
  await source.goto('data:text/html,<title>Untouched source</title><p>source</p>');
  const sourceUrl = source.url();

  const fixture = await readUiFixture('UI-ESC-DISMISS');
  const surface = await openSurface(harness, fixture);
  try {
    const tabState = async () => worker.evaluate(async () => {
      const tabs = await chrome.tabs.query({});
      return tabs
        .map(tab => ({ id: tab.id, url: tab.url, title: tab.title, pinned: tab.pinned, active: tab.active }))
        .sort((left, right) => (left.id ?? 0) - (right.id ?? 0));
    });

    const before = await tabState();
    await surface.locator('[data-search-input]').fill('escape');
    await expect(surface.locator('html')).toHaveAttribute('data-surface-state', 'results');
    expect(await tabState()).toEqual(before);

    await surface.keyboard.press('Escape');
    await expect(surface.locator('html')).toHaveAttribute('data-dismissed', 'true');
    expect(await tabState()).toEqual(before);
    expect(source.url()).toBe(sourceUrl);
  } finally {
    await surface.close();
    await source.close();
  }
});
