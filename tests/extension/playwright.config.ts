import { defineConfig } from '@playwright/test';
import { existsSync } from 'node:fs';
const browser = process.env.BROWSER;
if (browser !== 'chromium' && browser !== 'chrome' && browser !== 'edge') {
  throw new Error('BROWSER must be chromium, chrome, or edge');
}
if (browser !== 'chromium') {
  throw new Error(
    `Automated extension tests cannot sideload into branded ${browser} builds (Chrome 137+ removed the required command-line flags). ` +
    'Use BROWSER=chromium for the IP-03 automated suite; branded Chrome/Edge compatibility is covered by the IP-19 browser matrix.',
  );
}
const extensionPath = `${process.cwd()}/extension`;
if (!existsSync(`${extensionPath}/manifest.json`)) throw new Error(`Extension manifest not found: ${extensionPath}/manifest.json`);
export default defineConfig({
  testDir: '../../tests/extension',
  testMatch: /.*\.test\.ts/,
  fullyParallel: false,
  timeout: 45000,
  reporter: 'line',
  use: { browserName: 'chromium', channel: 'chromium', headless: false, launchOptions: { args: [`--disable-extensions-except=${extensionPath}`, `--load-extension=${extensionPath}`] } },
});
