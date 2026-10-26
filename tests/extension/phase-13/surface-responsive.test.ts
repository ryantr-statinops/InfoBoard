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

test('narrow viewport and 200% text keep query, close and selected result operable', async () => {
  const fixture = await readUiFixture('UI-NARROW-TEXT-SCALE');
  const page = await openSurface(harness, fixture);
  try {
    await page.locator('[data-search-input]').fill('long');
    await expect(page.locator('html')).toHaveAttribute('data-surface-state', 'results');
    await page.setViewportSize({ width: 320, height: 480 });
    await page.evaluate(() => { document.documentElement.style.fontSize = '200%'; });

    const metrics = await page.evaluate(() => {
      const input = document.querySelector<HTMLInputElement>('[data-search-input]')!;
      const close = document.querySelector<HTMLButtonElement>('[data-close-search]')!;
      const selected = document.querySelector<HTMLElement>('[role="option"][aria-selected="true"]')!;
      const list = document.querySelector<HTMLElement>('[data-result-list]')!;
      const inputRect = input.getBoundingClientRect();
      const closeRect = close.getBoundingClientRect();
      const selectedRect = selected.getBoundingClientRect();
      const listRect = list.getBoundingClientRect();
      return {
        innerWidth,
        innerHeight,
        scrollWidth: document.documentElement.scrollWidth,
        bodyScrollWidth: document.body.scrollWidth,
        inputFont: getComputedStyle(input).fontSize,
        inputOutline: getComputedStyle(input).outlineWidth,
        inputVisible: inputRect.top >= 0 && inputRect.bottom <= innerHeight && inputRect.width > 0,
        closeVisible: closeRect.top >= 0 && closeRect.bottom <= innerHeight && closeRect.width > 0,
        selectedVisible: selectedRect.top >= listRect.top - 1 && selectedRect.top < listRect.bottom,
      };
    });

    expect(metrics.inputFont).toBe('32px');
    expect(metrics.inputVisible).toBe(true);
    expect(metrics.closeVisible).toBe(true);
    expect(metrics.selectedVisible).toBe(true);
    expect(metrics.scrollWidth).toBeLessThanOrEqual(metrics.innerWidth);
    // Chromium can round a 100%-width body by one physical pixel on Windows;
    // the document itself must not introduce horizontal scrolling.
    expect(metrics.bodyScrollWidth).toBeLessThanOrEqual(metrics.innerWidth + 1);
    expect(parseFloat(metrics.inputOutline)).toBeGreaterThanOrEqual(2);
    await expect(page.locator('[data-result-id="narrow-1"]')).toHaveAttribute('aria-selected', 'true');
  } finally {
    await page.close();
  }
});

test('reduced motion removes the nonessential row transition', async () => {
  const fixture = await readUiFixture('UI-NARROW-TEXT-SCALE');
  const page = await openSurface(harness, fixture);
  try {
    await page.locator('[data-search-input]').fill('long');
    await expect(page.locator('html')).toHaveAttribute('data-surface-state', 'results');
    const row = page.locator('[role="option"]').first();

    await page.emulateMedia({ reducedMotion: 'no-preference' });
    await expect.poll(() => row.evaluate(element => getComputedStyle(element).transitionDuration)).toContain('0.12s');
    await page.emulateMedia({ reducedMotion: 'reduce' });
    await expect.poll(() => row.evaluate(element => getComputedStyle(element).transitionDuration)).toBe('0s');
  } finally {
    await page.close();
  }
});
