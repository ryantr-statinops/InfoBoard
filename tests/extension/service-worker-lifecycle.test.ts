import { test, expect } from '@playwright/test';
import { readFile } from 'node:fs/promises';
import { ChromeBrowserAdapter, resetProfileAfterDisconnect } from '../../extension/src/browser/browser-adapter.js';

test('worker inputs fail closed on malformed or unavailable profile identity', async () => {
  for (const value of ['not-an-id', null]) {
    let windowReads = 0;
    const api = {
      runtime: { getURL: (path: string) => `chrome-extension://test/${path}`, onInstalled: { addListener() {} }, onStartup: { addListener() {} }, onMessage: { addListener() {} } },
      commands: { onCommand: { addListener() {} } },
      tabs: { query: async () => { throw new Error('must not inspect tabs'); }, update: async () => undefined },
      windows: { getLastFocused: async () => { windowReads++; return { id: 1, incognito: false }; }, create: async () => ({ id: 1 }), update: async () => undefined, remove: async () => undefined },
      storage: { local: { get: async (key: string) => { expect(key).toBe('profile_id'); return value === null ? {} : { profile_id: value }; }, set: async () => undefined, remove: async () => undefined } },
    };
    const result = await new ChromeBrowserAdapter(api as any).currentProfileContext();
    expect(result.ok).toBe(false);
    expect(windowReads).toBe(0);
  }
});

test('explicit identity reset disconnects before removing the sole approved key', async () => {
  const calls: string[] = [];
  const api = {
    runtime: { getURL: (path: string) => path, onInstalled: { addListener() {} }, onStartup: { addListener() {} }, onMessage: { addListener() {} } },
    commands: { onCommand: { addListener() {} } }, tabs: { query: async () => [], update: async () => undefined },
    windows: { getLastFocused: async () => ({ id: 1 }), create: async () => ({ id: 1 }), update: async () => undefined, remove: async () => undefined },
    storage: { local: { get: async (key: string) => { calls.push(`read:${key}`); return {}; }, set: async (values: Record<string, unknown>) => { calls.push(`write:${Object.keys(values).join(',')}`); }, remove: async (key: string) => { calls.push(`remove:${key}`); } } },
  };
  const adapter = new ChromeBrowserAdapter(api as any);
  await adapter.write('123e4567-e89b-42d3-a456-426614174000' as any);
  await resetProfileAfterDisconnect(adapter, async () => { calls.push('disconnect'); });
  expect(calls).toEqual(['write:profile_id', 'disconnect', 'remove:profile_id']);
});

test('surface sources and generated manifest exclude network and page storage', async () => {
  const html = await readFile('extension/src/search/surface-shell.html', 'utf8');
  const adapter = await readFile('extension/src/browser/browser-adapter.ts', 'utf8');
  const manifest = JSON.parse(await readFile('extension/manifest.json', 'utf8'));
  expect(html).not.toMatch(/https?:\/\//);
  expect(html).not.toContain('localStorage');
  expect(html).not.toContain('sessionStorage');
  expect(adapter).not.toContain('storage.sync');
  expect(manifest.permissions).toEqual(['storage', 'tabs', 'windows']);
});
