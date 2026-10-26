import { expect, test } from '@playwright/test';
import {
  closeSurfaceHarness,
  driverState,
  launchSurfaceHarness,
  openSurface,
  readUiFixture,
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

test('shortcut focus fixture starts with the query input focused and no stale rows', async () => {
  const fixture = await readUiFixture('UI-SHORTCUT-FOCUS');
  const page = await openSurface(harness, fixture);
  try {
    await expect(page.locator('html')).toHaveAttribute('data-surface-state', 'ready');
    await expect(page.locator('[data-search-input]')).toBeFocused();
    await expect(page.locator('[role="option"]')).toHaveCount(0);
    await expect(page.locator('[data-result-list]')).toBeHidden();
    await expect(page.locator('html')).toHaveAttribute('data-browser-state-confirmed', 'true');
    await expect(page.locator('html')).toHaveAttribute('data-shortcut-focus-ms', /\d+/);
  } finally {
    await page.close();
  }
});

test('no-result, rebuilding, host-down and stale states cannot activate', async () => {
  const cases = [
    ['UI-NO-RESULT', 'no_results', 'No matching open tabs'],
    ['UI-REBUILDING', 'rebuilding', 'Search index is rebuilding. Try again shortly.'],
    ['UI-HOST-DOWN', 'unavailable', 'Search is unavailable. Try again.'],
    ['UI-STALE', 'stale', 'Results are stale. Type again or retry.'],
  ] as const;
  for (const [name, state, message] of cases) {
    const fixture = await readUiFixture(name);
    const page = await openSurface(harness, fixture);
    try {
      await page.locator('[data-search-input]').fill('state');
      await expect(page.locator('html')).toHaveAttribute('data-surface-state', state);
      await expect(page.locator('[data-surface-status]')).toHaveText(message);
      await page.keyboard.press('Enter');
      expect((await driverState(page)).activations).toEqual([]);
      if (state !== 'no_results') {
        await expect(page.locator('html')).toHaveAttribute('data-surface-actions', /retry/);
      }
    } finally {
      await page.close();
    }
  }
});

test('opening/loading keeps focus and later renders the matching result', async () => {
  const fixture = await readUiFixture('UI-OPENING');
  const page = await openSurface(harness, fixture);
  try {
    const input = page.locator('[data-search-input]');
    await input.fill('opening');
    await expect(page.locator('html')).toHaveAttribute('data-surface-state', 'querying');
    await expect(page.locator('[data-surface-status]')).toHaveText('Searching');
    await expect(input).toBeFocused();
    await expect(input).toHaveAttribute('aria-busy', 'true');
    expect((await driverState(page)).pending).toHaveLength(1);

    await page.evaluate(() => {
      (window as unknown as {
        __INFOBOARD_SURFACE_TEST_STATE__: { resolve(index: number, response: unknown): void };
      }).__INFOBOARD_SURFACE_TEST_STATE__.resolve(0, {
        status: 'results',
        projection_revision: 6,
        rows: [{
          result_id: 'opening-one',
          tab_id: 61,
          window_id: 1,
          title: 'Opening result',
          domain: 'opening.example.test',
          explanations: ['title match'],
        }],
      });
    });
    await expect(page.locator('html')).toHaveAttribute('data-surface-state', 'results');
    await expect(page.locator('[data-result-id="opening-one"]')).toHaveCount(1);
    await expect(input).toHaveAttribute('aria-busy', 'false');
  } finally {
    await page.close();
  }
});

test('persistence-degraded rows stay actionable and activation failure is confirmed', async () => {
  const degradedFixture = await readUiFixture('UI-PERSISTENCE-DEGRADED');
  const degraded = await openSurface(harness, degradedFixture);
  try {
    await degraded.locator('[data-search-input]').fill('current');
    await expect(degraded.locator('html')).toHaveAttribute('data-surface-state', 'persistence_degraded');
    await expect(degraded.locator('[data-surface-status]')).toHaveText('1 result. History is unavailable; results are current for this session.');
    await degraded.keyboard.press('Enter');
    await expect(degraded.locator('html')).toHaveAttribute('data-surface-state', 'activation_success');
  } finally {
    await degraded.close();
  }

  const failureFixture = await readUiFixture('UI-ACTIVATION-FAILURE');
  const failure = await openSurface(harness, failureFixture);
  try {
    await failure.locator('[data-search-input]').fill('failure');
    await expect(failure.locator('html')).toHaveAttribute('data-surface-state', 'results');
    await failure.keyboard.press('Enter');
    await expect(failure.locator('html')).toHaveAttribute('data-surface-state', 'activation_failure');
    await expect(failure.locator('[data-surface-status]')).toHaveText('Selected tab is no longer available. Refresh and choose another.');
    await expect(failure.locator('html')).toHaveAttribute('data-browser-state-confirmed', 'true');
    expect(await driverState(failure)).toMatchObject({
      activations: [{ resultId: 'activation-failure-one', tabId: 81, projectionRevision: 12 }],
    });
  } finally {
    await failure.close();
  }
});

test('private-context unavailability disables query and activation actions', async () => {
  const fixture = await readUiFixture('UI-PRIVATE-UNAVAILABLE');
  const page = await openSurface(harness, fixture);
  try {
    await expect(page.locator('html')).toHaveAttribute('data-surface-state', 'private_unavailable');
    await expect(page.locator('[data-surface-status]')).toHaveText('Search is unavailable in private windows.');
    await expect(page.locator('[data-search-input]')).toBeDisabled();
    await expect(page.locator('[data-result-list]')).toBeHidden();
    await expect(page.locator('html')).toHaveAttribute('data-surface-actions', 'dismiss');
    await expect(page.locator('[data-close-search]')).toBeVisible();
  } finally {
    await page.close();
  }
});
