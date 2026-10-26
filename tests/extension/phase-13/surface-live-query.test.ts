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

test('rapid revisions render only the newest accepted response', async () => {
  const fixture = await readUiFixture('UI-LIVE-QUERY');
  const page = await openSurface(harness, fixture);
  try {
    const input = page.locator('[data-search-input]');
    await input.fill('first');
    await expect(page.locator('html')).toHaveAttribute('data-surface-state', 'querying');
    await input.fill('second');
    await expect.poll(async () => (await driverState(page)).pending.length).toBe(2);

    await page.evaluate(() => {
      (window as unknown as {
        __INFOBOARD_SURFACE_TEST_STATE__: {
          resolve(index: number, response: unknown): void;
        };
      }).__INFOBOARD_SURFACE_TEST_STATE__.resolve(1, {
        status: 'results',
        projection_revision: 4,
        rows: [{
          result_id: 'live-2',
          tab_id: 2,
          window_id: 1,
          title: 'Second accepted revision',
          domain: 'second.example.test',
          explanations: ['title match'],
        }],
      });
    });
    await expect(page.locator('html')).toHaveAttribute('data-surface-state', 'results');
    await expect(page.locator('[data-result-id="live-2"]')).toHaveCount(1);
    await expect(page.locator('[data-surface-status]')).toHaveText('1 result');
    await expect(page.locator('[data-search-input]')).toHaveAttribute('aria-activedescendant', 'surface-result-0');

    await page.evaluate(() => {
      (window as unknown as {
        __INFOBOARD_SURFACE_TEST_STATE__: {
          resolve(index: number, response: unknown): void;
        };
      }).__INFOBOARD_SURFACE_TEST_STATE__.resolve(0, {
        status: 'results',
        projection_revision: 4,
        rows: [{
          result_id: 'live-1',
          tab_id: 1,
          window_id: 1,
          title: 'First accepted revision',
          domain: 'first.example.test',
          explanations: ['title match'],
        }],
      });
    });
    await expect(page.locator('[data-result-id="live-2"]')).toHaveCount(1);
    await expect(page.locator('[data-result-id="live-1"]')).toHaveCount(0);
    expect((await driverState(page)).queries.map(query => query.query)).toEqual(['first', 'second']);
  } finally {
    await page.close();
  }
});

test('empty and whitespace input requests the bounded context list', async () => {
  const fixture = await readUiFixture('UI-EMPTY');
  const page = await openSurface(harness, fixture);
  try {
    await page.locator('[data-search-input]').fill('   ');
    await expect(page.locator('html')).toHaveAttribute('data-surface-state', 'empty_query');
    await expect(page.locator('[data-surface-status]')).toHaveText('Recent and related tabs');
    await expect(page.locator('[data-result-id="recent-1"]')).toHaveCount(1);
    expect((await driverState(page)).queries).toEqual([{ query: '' }]);
  } finally {
    await page.close();
  }
});
