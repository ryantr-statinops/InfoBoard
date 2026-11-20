import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile, stat } from 'node:fs/promises';
import { resolve } from 'node:path';

test('manifest.json conforms to Pure MV3 requirements', async () => {
  const manifestRaw = await readFile('extension/manifest.json', 'utf8');
  const manifest = JSON.parse(manifestRaw);

  assert.equal(manifest.manifest_version, 3);
  assert.equal(manifest.name, 'InfoBoard');

  // Verify permissions are strictly trimmed to the essential 3
  assert.deepEqual(manifest.permissions.sort(), ['storage', 'tabGroups', 'tabs']);
  assert.equal(manifest.permissions.includes('nativeMessaging'), false);

  // Background service worker
  assert.equal(manifest.background.service_worker, 'dist/src/background/service-worker.js');
  assert.equal(manifest.background.type, 'module');

  // Action popup
  assert.equal(manifest.action.default_popup, 'dist/search/surface-shell.html');

  // Keyboard command shortcut
  assert.ok(manifest.commands._execute_action, '_execute_action command must be configured');
  assert.equal(manifest.commands._execute_action.suggested_key.default, 'Alt+W');
  assert.equal(manifest.commands._execute_action.suggested_key.mac, 'Command+Shift+K');
});

test('build artifacts exist in extension/dist and are ultra-lightweight (< 50KB)', async () => {
  const artifacts = [
    'extension/dist/search/surface-shell.html',
    'extension/dist/search/surface-shell.css',
    'extension/dist/src/background/service-worker.js',
    'extension/dist/src/background/mru-manager.js',
    'extension/dist/src/search/surface-entrypoint.js',
    'extension/dist/src/search/scorer.js',
    'extension/dist/src/search/types.js'
  ];

  let totalBytes = 0;
  for (const file of artifacts) {
    const fileStat = await stat(file);
    assert.ok(fileStat.size > 0, `${file} should not be empty`);
    totalBytes += fileStat.size;
  }

  // Entire extension code bundle should be under 50KB uncompressed!
  assert.ok(
    totalBytes < 50 * 1024,
    `Total bundle size (${(totalBytes / 1024).toFixed(2)} KB) must be under 50KB`
  );
});

test('surface-shell.html contains critical accessible search elements', async () => {
  const html = await readFile('extension/dist/search/surface-shell.html', 'utf8');
  assert.ok(html.includes('data-search-input'), 'Must contain search input');
  assert.ok(html.includes('id="tab-results"'), 'Must contain tab results list container');
  assert.ok(html.includes('role="listbox"'), 'Tab results must have role="listbox" for accessibility');
  assert.ok(html.includes('surface-shell.css'), 'Must link to surface-shell.css');
  assert.ok(html.includes('surface-entrypoint.js'), 'Must load surface-entrypoint.js module');
});
