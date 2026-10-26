import { expect, test } from '@playwright/test';
import {
  closeSurfaceHarness,
  driverState,
  launchSurfaceHarness,
  openSurface,
  readUiFixture,
  surfaceState,
  type SurfaceHarness,
} from './harness.js';

test.describe.configure({ mode: 'serial' });

let harness: SurfaceHarness;

test.beforeAll(async () => {
  harness = await launchSurfaceHarness();
});

test.afterAll(async () => {
  await closeSurfaceHarness(harness);
});

test('arrow keys move selection and Enter hands off the explicit result revision', async () => {
  const fixture = await readUiFixture('UI-ARROW-ENTER');
  const page = await openSurface(harness, fixture);
  try {
    const input = page.locator('[data-search-input]');
    await input.fill('docs');
    await expect(page.locator('html')).toHaveAttribute('data-surface-state', 'results');
    await expect(input).toBeFocused();
    await expect(input).toHaveAttribute('aria-activedescendant', 'surface-result-0');

    await page.keyboard.press('ArrowDown');
    await expect(input).toHaveAttribute('aria-activedescendant', 'surface-result-1');
    await expect(page.locator('[role="option"]').nth(1)).toHaveAttribute('aria-selected', 'true');
    await expect(page.locator('[role="option"]').nth(0)).toHaveAttribute('aria-selected', 'false');

    await page.keyboard.press('Enter');
    await expect(page.locator('html')).toHaveAttribute('data-surface-state', 'activation_success');
    await expect(page.locator('[data-surface-status]')).toHaveText('Opened selected tab');
    expect(await driverState(page)).toMatchObject({
      activations: [{ resultId: 'arrow-second', tabId: 32, projectionRevision: 11 }],
      dismisses: 0,
    });
    await expect(page.locator('html')).toHaveAttribute('data-browser-state-confirmed', 'true');
  } finally {
    await page.close();
  }
});

test('Escape dismisses without activation and leaves browser state confirmed unchanged', async () => {
  const fixture = await readUiFixture('UI-ESC-DISMISS');
  const page = await openSurface(harness, fixture);
  try {
    await page.locator('[data-search-input]').fill('escape');
    await expect(page.locator('html')).toHaveAttribute('data-surface-state', 'results');
    await page.keyboard.press('Escape');
    await expect(page.locator('html')).toHaveAttribute('data-dismissed', 'true');
    expect(await surfaceState(page)).toBe('results');
    expect(await driverState(page)).toMatchObject({ activations: [], dismisses: 1 });
    await expect(page.locator('html')).toHaveAttribute('data-browser-state-confirmed', 'true');
    await expect(page.locator('[data-search-input]')).toBeFocused();
  } finally {
    await page.close();
  }
});

test('the explicit close control is keyboard reachable and dismisses', async () => {
  const fixture = await readUiFixture('UI-ESC-DISMISS');
  const page = await openSurface(harness, fixture);
  try {
    await page.locator('[data-close-search]').focus();
    await expect(page.locator('[data-close-search]')).toBeFocused();
    await page.keyboard.press('Enter');
    await expect(page.locator('html')).toHaveAttribute('data-dismissed', 'true');
    expect(await driverState(page)).toMatchObject({ activations: [], dismisses: 1 });
  } finally {
    await page.close();
  }
});
