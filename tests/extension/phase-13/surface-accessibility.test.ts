import { expect, test } from '@playwright/test';
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

test('the combobox, listbox and selected option expose stable accessible names and states', async () => {
  const fixture = await readUiFixture('UI-DUPLICATE-CONTEXT');
  const page = await openSurface(harness, fixture);
  try {
    const input = page.locator('[data-search-input]');
    await input.fill('project');
    await expect(page.locator('html')).toHaveAttribute('data-surface-state', 'results');
    await expect(input).toHaveAttribute('role', 'combobox');
    await expect(input).toHaveAttribute('aria-autocomplete', 'list');
    await expect(input).toHaveAttribute('aria-expanded', 'true');
    await expect(input).toHaveAttribute('aria-controls', 'surface-results');
    await expect(input).toHaveAttribute('aria-activedescendant', 'surface-result-0');
    await expect(input).toHaveAttribute('aria-busy', 'false');

    const list = page.locator('[data-result-list]');
    await expect(list).toHaveAttribute('role', 'listbox');
    await expect(list).toHaveAttribute('aria-label', 'Open tab results');
    await expect(list).toHaveAttribute('aria-busy', 'false');
    await expect(list.locator('[role="option"]')).toHaveCount(2);
    await expect(list.locator('[role="option"]').nth(0)).toHaveAttribute('aria-selected', 'true');
    await expect(list.locator('[role="option"]').nth(1)).toHaveAttribute('aria-selected', 'false');
    await expect(list.locator('[data-result-pinned]')).toHaveText('Pinned');

    const status = page.locator('[data-surface-status]');
    await expect(status).toHaveAttribute('role', 'status');
    await expect(status).toHaveAttribute('aria-live', 'polite');
    await expect(status).toHaveAttribute('aria-atomic', 'true');
    await expect(status).toHaveText('2 results');
  } finally {
    await page.close();
  }
});

test('empty-query announcements are concise and do not echo query or URL text', async () => {
  const fixture = await readUiFixture('UI-EMPTY');
  const page = await openSurface(harness, fixture);
  try {
    await page.locator('[data-search-input]').fill('   ');
    await expect(page.locator('html')).toHaveAttribute('data-surface-state', 'empty_query');
    const status = await page.locator('[data-surface-status]').innerText();
    expect(status).toBe('Recent and related tabs');
    expect(status).not.toContain('auth=');
    expect(status).not.toContain('token=');
    expect(status).not.toContain('http');
    await expect(page.locator('[data-result-list]')).toHaveAttribute('role', 'listbox');
    await expect(page.locator('[data-result-id="recent-1"]')).toHaveAttribute('aria-selected', 'true');
  } finally {
    await page.close();
  }
});
