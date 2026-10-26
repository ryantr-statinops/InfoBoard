import { expect, test } from '@playwright/test';
import { safeDomain, safeExplanation } from '../../../extension/src/search/surface-renderer.js';
import {
  closeSurfaceHarness,
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

test('duplicate titles remain distinguishable without exposing raw URLs', async () => {
  const fixture = await readUiFixture('UI-DUPLICATE-CONTEXT');
  const page = await openSurface(harness, fixture);
  try {
    await page.locator('[data-search-input]').fill('project');
    await expect(page.locator('html')).toHaveAttribute('data-surface-state', 'results');
    const rows = page.locator('[role="option"]');
    await expect(rows).toHaveCount(2);
    await expect(rows.nth(0)).toHaveAttribute('data-result-id', 'duplicate-alpha');
    await expect(rows.nth(0)).toHaveAttribute('aria-selected', 'true');
    await expect(rows.nth(0)).toHaveAttribute('aria-label', /Project dashboard\. alpha\.example\.test · Window 1 · Work\. Pinned\. title match/);
    await expect(rows.nth(0).locator('[data-result-pinned]')).toHaveText('Pinned');
    await expect(rows.nth(0).locator('[data-result-context]')).toHaveText('alpha.example.test · Window 1 · Work');
    await expect(rows.nth(1).locator('[data-result-context]')).toHaveText('secret.example.test · Window 2 · Research');
    await expect(rows.nth(1).locator('[data-result-explanation]')).toHaveCount(0);

    const text = await page.locator('[data-result-list]').innerText();
    expect(text).not.toContain('auth=');
    expect(text).not.toContain('token=');
    expect(text).not.toContain('https://');
    expect(await page.locator('[data-search-input]').getAttribute('aria-activedescendant')).toBe('surface-result-0');
  } finally {
    await page.close();
  }
});

test('domain and explanation helpers fail closed on URL-like values', () => {
  expect(safeDomain('https://example.test/path?auth=abc')).toBe('example.test');
  expect(safeDomain('example.test/path?auth=abc')).toBe('example.test');
  expect(safeDomain('')).toBe('');
  expect(safeExplanation('title match')).toBe('title match');
  expect(safeExplanation('https://example.test/path?token=abc')).toBe('');
  expect(safeExplanation('auth=abc')).toBe('');
  expect(safeExplanation('x'.repeat(81))).toBe('');
});
