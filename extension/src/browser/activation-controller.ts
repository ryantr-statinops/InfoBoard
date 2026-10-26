// Browser-owned activation controller for IP-14.
//
// The controller accepts only an already-guarded activation reference, performs
// one exact tab/window operation at a time, and claims success only after the
// browser confirms the same tab ID, window ID, and active state. It never
// searches for a replacement tab or falls back to another result.

import type { BrowserResult } from './browser-adapter.js';
import {
  guardActivation,
  type ActivationGuardInput,
  type ActivationGuardOutcome,
  type ActivationReference,
} from './activation-guard.js';

export type ActivationBrowserState = 'unchanged' | 'window_focused' | 'tab_active' | 'unknown';

export type ActivationErrorCode =
  | ActivationGuardOutcome
  | 'activation_in_progress'
  | 'activation_cancelled'
  | 'browser_permission_denied'
  | 'browser_unsupported'
  | 'browser_operation_failed'
  | 'verification_failed';

export interface ActivationBrowserTab {
  readonly id: number;
  readonly windowId: number;
  readonly active: boolean;
}

export interface ActivationBrowser {
  getTab(tabId: number): Promise<BrowserResult<ActivationBrowserTab>>;
  focusWindow(windowId: number): Promise<BrowserResult<void>>;
  activateTab(tabId: number): Promise<BrowserResult<ActivationBrowserTab>>;
}

export interface ActivationControllerResult {
  readonly requestId: string;
  readonly ok: boolean;
  readonly confirmed: boolean;
  readonly browserState: ActivationBrowserState;
  readonly errorCode?: ActivationErrorCode;
  readonly durationMs: number;
}

export interface ActivationControllerOptions {
  readonly browser: ActivationBrowser;
  readonly now?: () => number;
  readonly onConfirmed?: () => void;
}

export interface ActivationControllerRequest extends ActivationGuardInput {}

interface InFlightActivation {
  readonly key: string;
  readonly promise: Promise<ActivationControllerResult>;
}

const CHROME_ERROR_MAP = {
  permission_denied: 'browser_permission_denied',
  unsupported: 'browser_unsupported',
  operation_failed: 'browser_operation_failed',
  cancelled: 'activation_cancelled',
  not_found: 'browser_operation_failed',
} as const;

function requestKey(reference: ActivationReference): string {
  return [
    reference.profile_id,
    reference.context_kind,
    reference.query_sequence,
    reference.result_id,
    reference.projection_epoch,
    reference.projection_revision,
    reference.tab_id,
    reference.window_id,
  ].join('\u0000');
}

function monotonicNow(): number {
  const performanceNow = globalThis.performance?.now;
  return typeof performanceNow === 'function' ? performanceNow.call(globalThis.performance) : Date.now();
}

function safeDuration(startedAt: number, now: () => number): number {
  const duration = now() - startedAt;
  return Number.isFinite(duration) && duration >= 0 ? duration : 0;
}

function errorCodeForFailure(
  error: Extract<BrowserResult<unknown>, { ok: false }>['error'],
  stage: 'get_tab' | 'focus_window' | 'activate_tab',
): ActivationErrorCode {
  if (error.kind === 'not_found') {
    return stage === 'focus_window' ? 'window_missing' : 'tab_missing';
  }
  if (error.kind === 'cancelled') return 'activation_cancelled';
  return CHROME_ERROR_MAP[error.kind];
}

export class ActivationController {
  private readonly browser: ActivationBrowser;
  private readonly now: () => number;
  private readonly onConfirmed?: () => void;
  private inFlight: InFlightActivation | null = null;
  private abortController: AbortController | null = null;
  private requestCounter = 0;

  constructor(options: ActivationControllerOptions) {
    this.browser = options.browser;
    this.now = options.now ?? monotonicNow;
    this.onConfirmed = options.onConfirmed;
  }

  isPending(): boolean {
    return this.inFlight !== null;
  }

  activate(request: ActivationControllerRequest): Promise<ActivationControllerResult> {
    const key = requestKey(request.reference);
    if (this.inFlight) {
      if (this.inFlight.key === key) return this.inFlight.promise;
      return Promise.resolve(this.result(
        `activation-${++this.requestCounter}`,
        false,
        true,
        'unchanged',
        'activation_in_progress',
        0,
      ));
    }

    const requestId = `activation-${++this.requestCounter}`;
    const abortController = new AbortController();
    this.abortController = abortController;
    let promise: Promise<ActivationControllerResult>;
    promise = this.run(requestId, request, abortController.signal)
      .finally(() => {
        if (this.inFlight?.promise === promise) this.inFlight = null;
        if (this.abortController === abortController) this.abortController = null;
      });
    this.inFlight = { key, promise };
    return promise;
  }

  /** Cancel the current request before its next browser mutation. */
  cancel(): void {
    this.abortController?.abort();
  }

  /** Reset/uninstall teardown entry point. */
  cancelAll(): void {
    this.cancel();
  }

  private async run(
    requestId: string,
    request: ActivationControllerRequest,
    signal: AbortSignal,
  ): Promise<ActivationControllerResult> {
    const startedAt = this.now();
    const reference = request.reference;
    const guard = guardActivation(request);
    if (guard.outcome !== 'valid') {
      return this.result(requestId, false, true, 'unchanged', guard.outcome, safeDuration(startedAt, this.now));
    }
    if (signal.aborted) {
      return this.result(requestId, false, true, 'unchanged', 'activation_cancelled', safeDuration(startedAt, this.now));
    }

    const tabResult = await this.browser.getTab(reference.tab_id);
    if (!tabResult.ok) {
      return this.result(
        requestId,
        false,
        true,
        'unchanged',
        errorCodeForFailure(tabResult.error, 'get_tab'),
        safeDuration(startedAt, this.now),
      );
    }
    if (tabResult.value.id !== reference.tab_id) {
      return this.result(requestId, false, true, 'unchanged', 'tab_reused', safeDuration(startedAt, this.now));
    }
    if (tabResult.value.windowId !== reference.window_id) {
      return this.result(requestId, false, true, 'unchanged', 'window_missing', safeDuration(startedAt, this.now));
    }
    if (signal.aborted) {
      return this.result(requestId, false, true, 'unchanged', 'activation_cancelled', safeDuration(startedAt, this.now));
    }

    const focusResult = await this.browser.focusWindow(reference.window_id);
    if (!focusResult.ok) {
      return this.result(
        requestId,
        false,
        true,
        'unchanged',
        errorCodeForFailure(focusResult.error, 'focus_window'),
        safeDuration(startedAt, this.now),
      );
    }
    if (signal.aborted) {
      return this.result(requestId, false, true, 'window_focused', 'activation_cancelled', safeDuration(startedAt, this.now));
    }

    const activateResult = await this.browser.activateTab(reference.tab_id);
    if (!activateResult.ok) {
      return this.result(
        requestId,
        false,
        true,
        'window_focused',
        errorCodeForFailure(activateResult.error, 'activate_tab'),
        safeDuration(startedAt, this.now),
      );
    }
    if (activateResult.value.id !== reference.tab_id
      || activateResult.value.windowId !== reference.window_id
      || activateResult.value.active !== true) {
      return this.result(requestId, false, true, 'window_focused', 'verification_failed', safeDuration(startedAt, this.now));
    }

    try {
      this.onConfirmed?.();
    } catch {
      // Browser activation is already authoritative; surface cleanup is best effort.
    }
    return this.result(requestId, true, true, 'tab_active', undefined, safeDuration(startedAt, this.now));
  }

  private result(
    requestId: string,
    ok: boolean,
    confirmed: boolean,
    browserState: ActivationBrowserState,
    errorCode: ActivationErrorCode | undefined,
    durationMs: number,
  ): ActivationControllerResult {
    return { requestId, ok, confirmed, browserState, errorCode, durationMs };
  }
}

interface ChromeActivationApi {
  readonly tabs?: {
    get(tabId: number): Promise<unknown>;
    update(tabId: number, updateProperties: { active: boolean }): Promise<unknown>;
  };
  readonly windows?: {
    update(windowId: number, updateInfo: { focused: boolean }): Promise<unknown>;
  };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

function failure(error: unknown): BrowserResult<never> {
  const message = error instanceof Error ? error.message.toLowerCase() : '';
  const kind = message.includes('permission')
    ? 'permission_denied'
    : message.includes('not found')
      ? 'not_found'
      : message.includes('unsupported')
        ? 'unsupported'
        : message.includes('cancel')
          ? 'cancelled'
          : 'operation_failed';
  return { ok: false, error: { kind, retryable: kind === 'operation_failed' } };
}

function tabFromApi(value: unknown): ActivationBrowserTab | null {
  if (!isRecord(value)
    || !Number.isSafeInteger(value.id)
    || !Number.isSafeInteger(value.windowId)
    || typeof value.active !== 'boolean') {
    return null;
  }
  return { id: value.id as number, windowId: value.windowId as number, active: value.active };
}

export class ChromeActivationBrowser implements ActivationBrowser {
  private readonly api: ChromeActivationApi;

  constructor(api?: ChromeActivationApi) {
    this.api = api ?? (globalThis as typeof globalThis & { chrome: ChromeActivationApi }).chrome;
  }

  async getTab(tabId: number): Promise<BrowserResult<ActivationBrowserTab>> {
    if (!this.api.tabs) return { ok: false, error: { kind: 'unsupported', retryable: false } };
    try {
      const tab = tabFromApi(await this.api.tabs.get(tabId));
      if (tab === null) return { ok: false, error: { kind: 'operation_failed', retryable: true } };
      return { ok: true, value: tab };
    } catch (error) {
      return failure(error);
    }
  }

  async focusWindow(windowId: number): Promise<BrowserResult<void>> {
    if (!this.api.windows) return { ok: false, error: { kind: 'unsupported', retryable: false } };
    try {
      const window = await this.api.windows.update(windowId, { focused: true });
      if (isRecord(window) && window.id !== undefined && window.id !== windowId) {
        return { ok: false, error: { kind: 'operation_failed', retryable: true } };
      }
      if (isRecord(window) && window.focused === false) {
        return { ok: false, error: { kind: 'operation_failed', retryable: true } };
      }
      return { ok: true, value: undefined };
    } catch (error) {
      return failure(error);
    }
  }

  async activateTab(tabId: number): Promise<BrowserResult<ActivationBrowserTab>> {
    if (!this.api.tabs) return { ok: false, error: { kind: 'unsupported', retryable: false } };
    try {
      const tab = tabFromApi(await this.api.tabs.update(tabId, { active: true }));
      if (tab === null) return { ok: false, error: { kind: 'operation_failed', retryable: true } };
      return { ok: true, value: tab };
    } catch (error) {
      return failure(error);
    }
  }
}
