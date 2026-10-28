// Accessible settings surface controller. It owns form state and safe status
// copy only; durable settings and uninstall cleanup stay behind the lifecycle
// client. Reset and uninstall never call tab, window, or history APIs.

import type { ProfileID } from '../../domain/index.js';
import type {
  ClientRequestResult,
  LifecycleOperation,
  LifecycleResponse,
} from '../runtime/native-messaging-client.js';
import {
  ConfigValidationError,
  validateProfileConfig,
  type ConfigKey,
  type ProfileConfig,
} from './contract.js';
import type { SettingsLifecycleClient } from './lifecycle-client.js';

export interface SettingsControllerOptions {
  doc: Document;
  profileId: ProfileID;
  client: SettingsLifecycleClient;
  confirm?: (message: string) => boolean;
  close?: () => void;
}

interface SettingsElements {
  form: HTMLFormElement;
  limit: HTMLInputElement;
  density: HTMLSelectElement;
  theme: HTMLSelectElement;
  contextLabels: HTMLInputElement;
  recency: HTMLInputElement;
  status: HTMLElement;
  persistence: HTMLElement;
  durable: HTMLElement;
  operation: HTMLElement;
  profile: HTMLElement;
  guidance: HTMLElement;
  save: HTMLButtonElement;
  reset: HTMLButtonElement;
  uninstall: HTMLButtonElement;
}

export class SettingsController {
  private readonly doc: Document;
  private readonly profileId: ProfileID;
  private readonly client: SettingsLifecycleClient;
  private readonly confirmAction: (message: string) => boolean;
  private readonly closePage: (() => void) | undefined;
  private readonly elements: SettingsElements;
  private current: ProfileConfig | undefined;
  private busy = false;
  private closed = false;
  private readonly cleanups: Array<() => void> = [];

  constructor(options: SettingsControllerOptions) {
    this.doc = options.doc;
    this.profileId = options.profileId;
    this.client = options.client;
    this.confirmAction = options.confirm ?? (message => globalThis.confirm(message));
    this.closePage = options.close;
    this.elements = findSettingsElements(this.doc);
  }

  async start(): Promise<void> {
    this.bind();
    this.setBusy(true);
    this.setStatus('Connecting to InfoBoard…');
    const started = await this.client.start();
    if (!started.ok) {
      this.setBusy(false);
      this.showFailure(started);
      return;
    }
    await this.load();
  }

  stop(): void {
    for (const cleanup of this.cleanups.splice(0)) cleanup();
    this.client.close();
  }

  private async load(): Promise<void> {
    this.setBusy(true);
    this.setStatus('Loading settings…');
    const result = await this.client.load();
    this.setBusy(false);
    if (!result.ok) {
      this.showFailure(result);
      return;
    }
    this.render(result.value);
  }

  private bind(): void {
    const onSubmit = (event: SubmitEvent) => {
      event.preventDefault();
      void this.save();
    };
    const onReset = () => { void this.reset(); };
    const onUninstall = () => { void this.uninstall(); };
    this.elements.form.addEventListener('submit', onSubmit);
    this.elements.reset.addEventListener('click', onReset);
    this.elements.uninstall.addEventListener('click', onUninstall);
    this.cleanups.push(() => this.elements.form.removeEventListener('submit', onSubmit));
    this.cleanups.push(() => this.elements.reset.removeEventListener('click', onReset));
    this.cleanups.push(() => this.elements.uninstall.removeEventListener('click', onUninstall));
  }

  private async save(): Promise<void> {
    if (this.busy || this.closed) return;
    const configuration = this.readConfiguration();
    if (configuration === undefined) return;
    this.setBusy(true);
    this.setStatus('Saving settings…');
    const result = await this.client.update(configuration);
    this.setBusy(false);
    if (!result.ok) {
      this.showFailure(result);
      return;
    }
    this.render(result.value);
  }

  private async reset(): Promise<void> {
    if (this.busy || this.closed) return;
    if (!this.confirmAction('Reset InfoBoard settings and local search data for this browser profile?')) return;
    this.setBusy(true);
    this.setStatus('Resetting settings…');
    const result = await this.client.reset();
    this.setBusy(false);
    if (!result.ok) {
      this.showFailure(result);
      return;
    }
    this.render(result.value);
  }

  private async uninstall(): Promise<void> {
    if (this.busy || this.closed) return;
    if (!this.confirmAction('Uninstall InfoBoard local data for this browser profile? This removes the local settings database and search index. It does not change your tabs or history.')) return;
    this.setBusy(true);
    this.setStatus('Uninstalling local data…');
    const result = await this.client.uninstall();
    this.setBusy(false);
    if (!result.ok) {
      this.showFailure(result);
      return;
    }
    this.render(result.value);
  }

  private readConfiguration(): ProfileConfig | undefined {
    if (this.current === undefined) {
      this.setStatus('Settings are not loaded yet. Reopen this page and try again.');
      return undefined;
    }
    const candidate = {
      ...this.current,
      result_limit: Number(this.elements.limit.value),
      density: this.elements.density.value,
      theme: this.elements.theme.value,
      context_labels: this.elements.contextLabels.checked,
      recency_enabled: this.elements.recency.checked,
    };
    try {
      return validateProfileConfig(candidate, this.profileId);
    } catch (error) {
      const field = error instanceof ConfigValidationError ? error.field : 'configuration';
      this.markInvalid(field);
      this.setStatus('Check the highlighted settings and try again.');
      return undefined;
    }
  }

  private render(result: LifecycleResponse): void {
    let configuration: ProfileConfig;
    try {
      configuration = validateProfileConfig(result.configuration, this.profileId);
    } catch {
      this.showFailure({
        ok: false,
        source: 'host',
        error: { code: 'INVALID_FRAME', retryable: false, message_key: 'INVALID_FRAME' },
      });
      return;
    }
    this.current = configuration;
    this.applyConfiguration(configuration);
    this.clearInvalid();
    this.elements.persistence.textContent = persistenceLabel(result.persistence);
    this.elements.durable.textContent = result.durable ? 'Yes' : 'No';
    this.elements.operation.textContent = operationLabel(result.operation);
    this.elements.profile.textContent = result.profile_id;
    this.doc.documentElement.dataset.settingsState = result.status;
    this.doc.documentElement.dataset.settingsPersistence = result.persistence;
    this.doc.documentElement.dataset.settingsOperation = result.operation;
    this.doc.documentElement.dataset.settingsResync = String(result.resync_required);
    this.elements.guidance.hidden = true;
    this.elements.guidance.textContent = '';

    if (result.operation === 'uninstall' && result.status === 'complete') {
      this.closed = true;
      this.setButtonsDisabled(true);
      this.elements.guidance.hidden = false;
      this.elements.guidance.textContent = 'Local InfoBoard data was removed for this profile. Close this page and reopen the extension to set it up again.';
      this.setStatus('Uninstall completed.');
      this.client.close();
      this.closePage?.();
      return;
    }
    if (result.operation === 'reset' && result.status === 'complete') {
      this.elements.guidance.hidden = false;
      this.elements.guidance.textContent = 'Settings were restored to the product defaults. Search refreshes on the next request.';
    }
    this.setStatus(resultMessage(result));
  }

  private applyConfiguration(configuration: ProfileConfig): void {
    this.elements.limit.value = String(configuration.result_limit);
    this.elements.density.value = configuration.density;
    this.elements.theme.value = configuration.theme;
    this.elements.contextLabels.checked = configuration.context_labels;
    this.elements.recency.checked = configuration.recency_enabled;
  }

  private showFailure(result: ClientRequestResult<unknown>): void {
    if (result.ok) return;
    const message = result.source === 'client'
      ? clientFailureMessage(result.error.code)
      : hostFailureMessage(result.error.code, result.error.retryable);
    this.doc.documentElement.dataset.settingsState = 'error';
    this.setStatus(message);
  }

  private setStatus(message: string): void {
    this.elements.status.textContent = message;
  }

  private setBusy(busy: boolean): void {
    this.busy = busy;
    this.elements.form.setAttribute('aria-busy', String(busy));
    this.setButtonsDisabled(busy || this.closed);
  }

  private setButtonsDisabled(disabled: boolean): void {
    this.elements.save.disabled = disabled;
    this.elements.reset.disabled = disabled;
    this.elements.uninstall.disabled = disabled;
  }

  private markInvalid(field: string): void {
    this.clearInvalid();
    const key = field === 'profile_id' ? 'profile' : field;
    const target = this.doc.querySelector<HTMLElement>(`[data-setting="${key}"]`);
    if (target) {
      target.setAttribute('aria-invalid', 'true');
      target.focus();
    }
  }

  private clearInvalid(): void {
    for (const element of this.doc.querySelectorAll<HTMLElement>('[data-setting][aria-invalid]')) {
      element.removeAttribute('aria-invalid');
    }
  }
}

function findSettingsElements(doc: Document): SettingsElements {
  const form = doc.querySelector<HTMLFormElement>('[data-settings-form]');
  const limit = doc.querySelector<HTMLInputElement>('[data-setting="result_limit"]');
  const density = doc.querySelector<HTMLSelectElement>('[data-setting="density"]');
  const theme = doc.querySelector<HTMLSelectElement>('[data-setting="theme"]');
  const contextLabels = doc.querySelector<HTMLInputElement>('[data-setting="context_labels"]');
  const recency = doc.querySelector<HTMLInputElement>('[data-setting="recency_enabled"]');
  const status = doc.querySelector<HTMLElement>('[data-settings-status]');
  const persistence = doc.querySelector<HTMLElement>('[data-settings-persistence]');
  const durable = doc.querySelector<HTMLElement>('[data-settings-durable]');
  const operation = doc.querySelector<HTMLElement>('[data-settings-operation]');
  const profile = doc.querySelector<HTMLElement>('[data-settings-profile]');
  const guidance = doc.querySelector<HTMLElement>('[data-settings-guidance]');
  const save = doc.querySelector<HTMLButtonElement>('[data-settings-save]');
  const reset = doc.querySelector<HTMLButtonElement>('[data-settings-reset]');
  const uninstall = doc.querySelector<HTMLButtonElement>('[data-settings-uninstall]');
  if (!form || !limit || !density || !theme || !contextLabels || !recency || !status || !persistence
    || !durable || !operation || !profile || !guidance || !save || !reset || !uninstall) {
    throw new Error('settings markup is missing required elements');
  }
  return { form, limit, density, theme, contextLabels, recency, status, persistence, durable, operation, profile, guidance, save, reset, uninstall };
}

function persistenceLabel(persistence: LifecycleResponse['persistence']): string {
  switch (persistence) {
    case 'ready':
      return 'Ready';
    case 'degraded':
      return 'Degraded';
    case 'unavailable':
      return 'Unavailable';
    case 'unknown':
      return 'Unknown';
  }
}

function operationLabel(operation: LifecycleOperation): string {
  switch (operation) {
    case 'load':
      return 'Loaded';
    case 'update':
      return 'Saved';
    case 'reset':
      return 'Reset';
    case 'uninstall':
      return 'Uninstall';
  }
}

function resultMessage(result: LifecycleResponse): string {
  if (result.status === 'complete') {
    if (result.operation === 'load') return 'Settings loaded.';
    if (result.operation === 'update') return 'Settings saved.';
    if (result.operation === 'reset') return 'Settings reset to defaults.';
    return 'Uninstall completed.';
  }
  if (result.status === 'degraded') {
    return `Settings are active but persistence is degraded${result.error_class ? ` (${result.error_class})` : ''}. Changes may not survive a restart.`;
  }
  if (result.status === 'retryable') {
    return `The host is temporarily unavailable${result.error_class ? ` (${result.error_class})` : ''}. Try again.`;
  }
  return `The operation was incomplete${result.error_class ? ` (${result.error_class})` : ''}. Retry after the host is available.`;
}

function clientFailureMessage(code: string): string {
  switch (code) {
    case 'HOST_UNAVAILABLE':
    case 'NOT_CONNECTED':
      return 'The InfoBoard host is unavailable. Reopen this page after the host is installed.';
    case 'DEADLINE_EXCEEDED':
      return 'The host did not respond in time. Try again.';
    case 'REQUEST_IN_FLIGHT':
    case 'REQUEST_SUPERSEDED':
      return 'Another settings request is still running. Wait a moment and try again.';
    case 'INVALID_SUBMISSION':
      return 'Check the highlighted settings and try again.';
    case 'SNAPSHOT_REQUIRED':
      return 'The host needs to refresh its local index. Try again.';
    default:
      return 'Settings could not be completed. Try again.';
  }
}

function hostFailureMessage(code: string, retryable: boolean): string {
  const suffix = retryable ? ' Try again.' : ' Reopen the extension after checking the host.';
  return `The host rejected the request (${code}).${suffix}`;
}
