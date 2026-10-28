import { test, expect, chromium, type BrowserContext } from '@playwright/test';
import { mkdtemp, rm } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';

const PROFILE_ID = '123e4567-e89b-42d3-a456-426614174000';

interface Launch {
  context: BrowserContext;
  directory: string;
  extensionId: string;
}

async function launch(): Promise<Launch> {
  const directory = await mkdtemp(path.join(os.tmpdir(), 'infoboard-settings-'));
  const context = await chromium.launchPersistentContext(directory, {
    channel: 'chromium',
    headless: false,
    args: [
      `--disable-extensions-except=${process.cwd()}/extension`,
      `--load-extension=${process.cwd()}/extension`,
    ],
  });
  const worker = context.serviceWorkers()[0] ?? await context.waitForEvent('serviceworker', { timeout: 10000 });
  return { context, directory, extensionId: new URL(worker.url()).host };
}

async function close(launch: Launch): Promise<void> {
  await launch.context.close();
  await rm(launch.directory, { recursive: true, force: true, maxRetries: 5, retryDelay: 200 });
}

test('settings options page renders defaults, saves, resets, and uninstalls through the lifecycle client', async () => {
  const launchState = await launch();
  try {
    const page = await launchState.context.newPage();
    const pagesBefore = launchState.context.pages().length;
    await page.addInitScript(({ profileId }: { profileId: string }) => {
      const calls: unknown[] = [];
      const defaults = {
        profile_id: profileId,
        result_limit: 10,
        density: 'comfortable',
        theme: 'system',
        context_labels: true,
        recency_enabled: true,
        protocol_version: 1,
        ranking_model_version: 'rank-1',
      };
      const result = (operation: string, overrides: Record<string, unknown> = {}) => ({
        operation,
        profile_id: profileId,
        status: 'complete',
        retryable: false,
        persistence: 'ready',
        durable: true,
        error_class: '',
        completed_steps: [],
        owned_state_verified: true,
        projection_revision: 0,
        configuration: defaults,
        resync_required: false,
        ...overrides,
      });
      const client = {
        start: async () => ({ ok: true, value: { accepted_protocol: 1, host_version: '0.1.0' } }),
        load: async () => {
          calls.push('load');
          return { ok: true, value: result('load') };
        },
        update: async (configuration: unknown) => {
          calls.push({ update: configuration });
          return { ok: true, value: result('update', { configuration }) };
        },
        reset: async () => {
          calls.push('reset');
          return { ok: true, value: result('reset', { configuration: defaults, resync_required: true }) };
        },
        uninstall: async () => {
          calls.push('uninstall');
          return { ok: true, value: result('uninstall', { completed_steps: ['storage_uninstall', 'paths_removed'] }) };
        },
        close: () => { calls.push('close'); },
      };
      (window as unknown as { __INFOBOARD_SETTINGS_TEST__: unknown }).__INFOBOARD_SETTINGS_TEST__ = {
        profileId,
        client,
        confirm: () => true,
      };
      (window as unknown as { __SETTINGS_TEST_STATE__: unknown[] }).__SETTINGS_TEST_STATE__ = calls;
    }, { profileId: PROFILE_ID });

    await page.goto(`chrome-extension://${launchState.extensionId}/dist/settings/settings-shell.html`);
    await expect(page.locator('[data-settings-status]')).toHaveText('Settings loaded.');
    await expect(page.locator('[data-setting="result_limit"]')).toHaveValue('10');
    await expect(page.locator('[data-setting="density"]')).toHaveValue('comfortable');
    await expect(page.locator('[data-setting="theme"]')).toHaveValue('system');
    await expect(page.locator('[data-setting="context_labels"]')).toBeChecked();
    await expect(page.locator('[data-setting="recency_enabled"]')).toBeChecked();
    await expect(page.locator('dd[data-settings-persistence]')).toHaveText('Ready');
    await expect(page.locator('dd[data-settings-durable]')).toHaveText('Yes');
    await expect(page.locator('dd[data-settings-operation]')).toHaveText('Loaded');
    await expect(page.locator('dd[data-settings-profile]')).toHaveText(PROFILE_ID);

    await page.locator('[data-setting="result_limit"]').fill('25');
    await page.locator('[data-setting="density"]').selectOption('compact');
    await page.locator('[data-setting="theme"]').selectOption('dark');
    await page.locator('[data-setting="context_labels"]').uncheck();
    await page.locator('[data-setting="recency_enabled"]').uncheck();
    await page.locator('[data-settings-save]').click();
    await expect(page.locator('[data-settings-status]')).toHaveText('Settings saved.');
    await expect(page.locator('dd[data-settings-operation]')).toHaveText('Saved');
    await expect(page.locator('[data-setting="result_limit"]')).toHaveValue('25');
    await expect(page.locator('[data-setting="theme"]')).toHaveValue('dark');

    await page.locator('[data-settings-reset]').click();
    await expect(page.locator('[data-settings-status]')).toHaveText('Settings reset to defaults.');
    await expect(page.locator('[data-setting="result_limit"]')).toHaveValue('10');
    await expect(page.locator('[data-setting="density"]')).toHaveValue('comfortable');
    await expect(page.locator('[data-setting="theme"]')).toHaveValue('system');
    await expect(page.locator('[data-setting="context_labels"]')).toBeChecked();
    await expect(page.locator('[data-setting="recency_enabled"]')).toBeChecked();

    await page.locator('[data-settings-uninstall]').click();
    await expect(page.locator('[data-settings-status]')).toHaveText('Uninstall completed.');
    await expect(page.locator('[data-settings-guidance]')).toBeVisible();
    await expect(page.locator('[data-settings-save]')).toBeDisabled();
    await expect(page.locator('[data-settings-reset]')).toBeDisabled();
    await expect(page.locator('[data-settings-uninstall]')).toBeDisabled();

    const calls = await page.evaluate(() => (window as unknown as { __SETTINGS_TEST_STATE__: unknown[] }).__SETTINGS_TEST_STATE__);
    expect(calls[0]).toBe('load');
    expect(calls).toContain('reset');
    expect(calls).toContain('uninstall');
    expect(calls).toContain('close');
    expect(calls.some(call => typeof call === 'object' && call !== null && 'update' in call)).toBe(true);
    expect(launchState.context.pages().length).toBe(pagesBefore);
    expect(page.url()).toContain('/dist/settings/settings-shell.html');
  } finally {
    await close(launchState);
  }
});

test('settings options page renders degraded persistence and safe defaults', async () => {
  const launchState = await launch();
  try {
    const page = await launchState.context.newPage();
    await page.addInitScript(({ profileId }: { profileId: string }) => {
      const configuration = {
        profile_id: profileId,
        result_limit: 10,
        density: 'comfortable',
        theme: 'system',
        context_labels: true,
        recency_enabled: true,
        protocol_version: 1,
        ranking_model_version: 'rank-1',
      };
      const degraded = {
        operation: 'load',
        profile_id: profileId,
        status: 'degraded',
        retryable: true,
        persistence: 'degraded',
        durable: false,
        error_class: 'storage_unavailable',
        completed_steps: [],
        owned_state_verified: false,
        projection_revision: 0,
        configuration,
        resync_required: false,
      };
      const client = {
        start: async () => ({ ok: true, value: { accepted_protocol: 1, host_version: '0.1.0' } }),
        load: async () => ({ ok: true, value: degraded }),
        update: async () => ({ ok: true, value: degraded }),
        reset: async () => ({ ok: true, value: degraded }),
        uninstall: async () => ({ ok: true, value: degraded }),
        close: () => undefined,
      };
      (window as unknown as { __INFOBOARD_SETTINGS_TEST__: unknown }).__INFOBOARD_SETTINGS_TEST__ = {
        profileId,
        client,
        confirm: () => true,
      };
    }, { profileId: PROFILE_ID });

    await page.goto(`chrome-extension://${launchState.extensionId}/dist/settings/settings-shell.html`);
    await expect(page.locator('dd[data-settings-persistence]')).toHaveText('Degraded');
    await expect(page.locator('dd[data-settings-durable]')).toHaveText('No');
    await expect(page.locator('[data-settings-status]')).toContainText('persistence is degraded');
    await expect(page.locator('html')).toHaveAttribute('data-settings-state', 'degraded');
    await expect(page.locator('[data-setting="result_limit"]')).toHaveValue('10');
    await expect(page.locator('[data-setting="recency_enabled"]')).toBeChecked();
  } finally {
    await close(launchState);
  }
});

test('search surface exposes a keyboard-reachable Settings control without changing the tab', async () => {
  const launchState = await launch();
  try {
    const page = await launchState.context.newPage();
    const sourceUrl = page.url();
    await page.addInitScript(({ profileId }: { profileId: string }) => {
      const model = {
        profileId,
        sequence: 0,
        status: 'idle',
        rows: [],
        projectionRevision: null,
      };
      const querySource = {
        view: () => model,
        subscribe: () => () => undefined,
        search: async () => model,
        cancel: () => model,
      };
      const activation = {
        activate: async () => ({
          ok: false,
          confirmed: true,
          browserState: 'unchanged',
          nextAction: 'none',
          errorCode: 'private_context',
          reportState: 'not_configured',
        }),
      };
      (window as unknown as { __INFOBOARD_SURFACE_TEST__: unknown }).__INFOBOARD_SURFACE_TEST__ = {
        context: { browserFamily: 'chrome', profileId, contextKind: 'normal' },
        querySource,
        activation,
        dismiss: () => undefined,
        openSettings: () => { (window as unknown as { __SETTINGS_OPENED__: boolean }).__SETTINGS_OPENED__ = true; },
      };
    }, { profileId: PROFILE_ID });

    await page.goto(`chrome-extension://${launchState.extensionId}/dist/search/surface-shell.html`);
    await expect(page.locator('html')).toHaveAttribute('data-search-ready', 'true');
    const settings = page.getByRole('button', { name: 'Settings' });
    await expect(settings).toBeVisible();
    await settings.focus();
    await expect(settings).toBeFocused();
    await settings.click();
    await expect.poll(() => page.evaluate(() => (window as unknown as { __SETTINGS_OPENED__?: boolean }).__SETTINGS_OPENED__)).toBe(true);
    expect(page.url()).not.toBe(sourceUrl);
    expect(page.url()).toContain('/dist/search/surface-shell.html');
  } finally {
    await close(launchState);
  }
});
