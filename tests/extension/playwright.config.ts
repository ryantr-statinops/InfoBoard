import { defineConfig } from '@playwright/test';
const browser = process.env.BROWSER;
if (browser !== 'chrome' && browser !== 'edge') throw new Error('BROWSER must be chrome or edge');
export default defineConfig({
  testDir: '../../tests/extension',
  testMatch: /.*\.test\.ts/,
  fullyParallel: false,
  timeout: 45000,
  reporter: 'line',
  use: { browserName: 'chromium', channel: browser === 'chrome' ? 'chrome' : 'msedge', headless: false, launchOptions: { ignoreDefaultArgs: ['--disable-extensions'], args: [`--disable-extensions-except=${process.cwd()}/extension`, `--load-extension=${process.cwd()}/extension`] } },
});
