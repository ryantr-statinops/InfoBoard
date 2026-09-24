import { test, expect } from '@playwright/test';
import { readFile } from 'node:fs/promises';

test('built manifest has only storage/window permissions and open/close action-popup shortcuts', async () => {
  const manifest = JSON.parse(await readFile('extension/manifest.json', 'utf8'));
  expect(manifest.manifest_version).toBe(3);
  expect([...manifest.permissions].sort()).toEqual(['storage', 'windows']);
  expect(manifest.host_permissions).toBeUndefined();
  expect(Object.keys(manifest.commands).sort()).toEqual(['close-search', 'open-search']);
  expect(manifest.commands['open-search'].suggested_key).toEqual({ default: 'Ctrl+Shift+Y', mac: 'Command+Shift+Y' });
  expect(manifest.commands['close-search'].suggested_key).toEqual({ default: 'Ctrl+Shift+X', mac: 'Command+Shift+X' });
  expect(manifest.action.default_popup).toBe('dist/search/surface-shell.html');
  expect(manifest.background.service_worker).toBe('dist/src/background/service-worker.js');
});
