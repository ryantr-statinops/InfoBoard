import assert from 'node:assert/strict';
import { readFile, readdir } from 'node:fs/promises';
import test from 'node:test';

const SETTINGS_ROOT = 'extension/src/settings';

const FORBIDDEN = [
  /\bchrome\.(?:tabs|history|windows)\b/,
  /\bbrowser\.(?:tabs|history|windows)\b/,
  /\btabs\.(?:query|update|remove|create|reload|goBack|goForward)\b/,
  /\bwindows\.(?:update|create|remove)\b/,
  /\bhistory\.(?:pushState|replaceState|deleteUrl|deleteRange|addUrl|search)\b/,
  /\blocation\.(?:assign|replace|reload)\b/,
];

test('settings lifecycle sources never call tab, history, or window mutation APIs', async () => {
  const files = (await readdir(SETTINGS_ROOT)).filter(name => name.endsWith('.ts') || name.endsWith('.html'));
  assert.ok(files.includes('settings-controller.ts'));
  assert.ok(files.includes('settings-shell.html'));
  for (const file of files) {
    const source = await readFile(`${SETTINGS_ROOT}/${file}`, 'utf8');
    for (const pattern of FORBIDDEN) {
      assert.doesNotMatch(source, pattern, `${file} matched forbidden browser mutation ${pattern}`);
    }
  }
});

test('the settings lifecycle client does not import a tab or history adapter', async () => {
  const source = await readFile(`${SETTINGS_ROOT}/lifecycle-client.ts`, 'utf8');
  assert.doesNotMatch(source, /browser-adapter|tab-observer|activation-reporter|query-client/);
});
