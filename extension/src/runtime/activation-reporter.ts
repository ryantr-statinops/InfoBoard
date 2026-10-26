// Non-blocking activation status reporter for IP-14.
//
// Browser confirmation is authoritative. A host disconnect, acknowledgement
// timeout, or persistence failure may downgrade reporter diagnostics/recency,
// but it must never rewrite a confirmed activation as a browser failure or
// trigger a compensating browser operation.

import type {
  ActivationBrowserState,
  ActivationControllerResult,
  ActivationErrorCode,
} from '../browser/activation-controller.js';
import type { ActivationReference } from '../browser/activation-guard.js';
import type { ActivationMetadata } from '../../domain/index.js';
import type {
  ActivationSubmission,
  ClientRequestResult,
  ProtocolErrorCode,
} from './native-messaging-client.js';

export type ActivationReporterState =
  | 'idle'
  | 'reporting'
  | 'acknowledged'
  | 'degraded'
  | 'unavailable'
  | 'cancelled';

export type ActivationRecencySource = 'keyboard_enter' | 'surface_select';

export interface ActivationReportInput {
  readonly reference: ActivationReference;
  readonly outcome: ActivationControllerResult;
  readonly domain?: string | null;
  readonly source?: ActivationRecencySource;
  readonly activatedAt?: number;
  readonly storageSequence?: number;
}

export type ActivationRecencyRejection =
  | 'not_confirmed'
  | 'private_context'
  | 'profile_mismatch'
  | 'invalid_identity'
  | 'invalid_domain'
  | 'invalid_timestamp'
  | 'invalid_source'
  | 'invalid_sequence';

export type ActivationRecencyState =
  | 'appended'
  | 'disabled'
  | 'sink_failed'
  | 'cancelled'
  | ActivationRecencyRejection;

export interface ActivationRecencyInput {
  readonly reference: ActivationReference;
  readonly outcome: ActivationControllerResult;
  readonly domain?: string | null;
  readonly source?: ActivationRecencySource;
  readonly activatedAt?: number;
  readonly storageSequence?: number;
}

export interface ActivationRecencyBuild {
  readonly state: 'valid' | ActivationRecencyRejection;
  readonly record?: ActivationMetadata;
}

export interface ActivationRecencyResult {
  readonly state: ActivationRecencyState;
  readonly record?: ActivationMetadata;
}

export interface ActivationRecencySink {
  append(record: ActivationMetadata): Promise<void> | void;
}

export interface ActivationRecencyOptions {
  readonly sink?: ActivationRecencySink;
  readonly now?: () => number;
}

export interface ActivationReportStatus {
  readonly state: ActivationReporterState;
  readonly accepted: boolean;
  readonly activationOk: boolean;
  readonly browserConfirmed: boolean;
  readonly browserState: ActivationBrowserState;
  readonly errorCode?: ActivationErrorCode;
  readonly hostErrorCode?: ProtocolErrorCode;
}

export interface ActivationReportTransport {
  reportActivation(submission: ActivationSubmission): Promise<ClientRequestResult<{ accepted: boolean }>>;
}

export interface ActivationReporterOptions {
  readonly transport: ActivationReportTransport;
  readonly recency?: ActivationRecencyHandoff;
  readonly onState?: (status: ActivationReportStatus) => void;
}

const INITIAL_STATUS: ActivationReportStatus = {
  state: 'idle',
  accepted: false,
  activationOk: false,
  browserConfirmed: true,
  browserState: 'unchanged',
};

export function isConfirmedActivation(outcome: ActivationControllerResult): boolean {
  return outcome.ok
    && outcome.confirmed
    && outcome.browserState === 'tab_active'
    && outcome.errorCode === undefined;
}

/**
 * Map an extension-owned activation failure onto the host's closed error enum.
 * The host never performs browser work; this code is diagnostic only.
 */
export function mapActivationErrorToWire(
  errorCode: ActivationErrorCode | undefined,
): ProtocolErrorCode {
  switch (errorCode) {
    case 'profile_mismatch':
    case 'context_mismatch':
      return 'PROFILE_MISMATCH';
    case 'stale_selection':
    case 'tab_missing':
    case 'window_missing':
    case 'tab_reused':
    case 'ineligible':
      return 'REVISION_MISMATCH';
    case 'projection_rebuilding':
      return 'INDEX_REBUILDING';
    case 'activation_cancelled':
      return 'HOST_SHUTDOWN';
    default:
      return 'INTERNAL_FAILURE';
  }
}

/**
 * Serialize only opaque reference/identity fields. Titles, URLs, query text,
 * display strings, and raw browser exceptions are deliberately absent.
 */
export function buildActivationSubmission(
  reference: ActivationReference,
  outcome: ActivationControllerResult,
): ActivationSubmission {
  const confirmed = isConfirmedActivation(outcome);
  return {
    result_reference: {
      tab_id: reference.tab_id,
      projection_epoch: reference.projection_epoch,
      projection_revision: Number(reference.projection_revision),
      result_id: reference.result_id,
    },
    tab_id: reference.tab_id,
    projection_revision: Number(reference.projection_revision),
    outcome: confirmed ? 'activated' : 'failed',
    error_code: confirmed ? undefined : mapActivationErrorToWire(outcome.errorCode),
  };
}

/**
 * Normalize a display domain into a bounded hostname-like value. URLs, query
 * strings, fragments, credentials, whitespace, and control characters are
 * rejected rather than truncated into a different domain.
 */
export function normalizeActivationDomain(value: unknown): string | null {
  if (typeof value !== 'string') return null;
  const normalized = value.trim().toLowerCase().replace(/\.$/, '');
  if (normalized.length === 0 || normalized.length > 255) return null;
  if (/[\s/?#@:%]/.test(normalized) || normalized.includes('://')) return null;
  if (new TextEncoder().encode(normalized).length > 255) return null;
  return normalized;
}

export function buildActivationMetadata(
  input: ActivationRecencyInput,
  now: number,
): ActivationRecencyBuild {
  if (!isConfirmedActivation(input.outcome)) return { state: 'not_confirmed' };

  const reference = input.reference;
  if (reference.context_kind !== 'normal'
    || reference.tab_identity.context_kind !== 'normal') {
    return { state: 'private_context' };
  }
  if (reference.tab_identity.profile_id !== reference.profile_id) {
    return { state: 'profile_mismatch' };
  }
  if (reference.tab_identity.tab_id !== reference.tab_id
    || !Number.isSafeInteger(reference.tab_id)
    || reference.tab_id < 0) {
    return { state: 'invalid_identity' };
  }

  const domain = normalizeActivationDomain(input.domain);
  if (domain === null) return { state: 'invalid_domain' };

  const source = input.source ?? 'keyboard_enter';
  if (source !== 'keyboard_enter' && source !== 'surface_select') {
    return { state: 'invalid_source' };
  }

  const activatedAt = input.activatedAt ?? now;
  if (!Number.isSafeInteger(activatedAt)
    || activatedAt < 0
    || activatedAt > now + 60_000
    || activatedAt < now - 30 * 86_400_000) {
    return { state: 'invalid_timestamp' };
  }

  const storageSequence = input.storageSequence ?? 0;
  if (!Number.isSafeInteger(storageSequence) || storageSequence < 0) {
    return { state: 'invalid_sequence' };
  }

  return {
    state: 'valid',
    record: {
      profile_id: reference.profile_id,
      context_kind: 'normal',
      tab_identity: { ...reference.tab_identity },
      domain,
      activated_at: activatedAt,
      source,
      storage_sequence: storageSequence,
    },
  };
}

export class ActivationRecencyHandoff {
  private readonly sink?: ActivationRecencySink;
  private readonly now: () => number;
  private readonly pending = new Set<AbortController>();
  private current: ActivationRecencyResult = { state: 'disabled' };
  private disposed = false;

  constructor(options: ActivationRecencyOptions = {}) {
    this.sink = options.sink;
    this.now = options.now ?? Date.now;
  }

  view(): ActivationRecencyResult {
    return this.current;
  }

  append(input: ActivationRecencyInput): Promise<ActivationRecencyResult> {
    if (this.disposed) return Promise.resolve(this.finish({ state: 'cancelled' }));
    const build = buildActivationMetadata(input, this.now());
    if (build.state !== 'valid') {
      return Promise.resolve(this.finish({ state: build.state }));
    }
    if (build.record === undefined) {
      return Promise.resolve(this.finish({ state: 'sink_failed' }));
    }
    if (this.sink === undefined) {
      return Promise.resolve(this.finish({ state: 'disabled', record: build.record }));
    }

    const controller = new AbortController();
    this.pending.add(controller);
    const promise = this.send(build.record, controller);
    return promise.finally(() => {
      this.pending.delete(controller);
    });
  }

  cancel(): void {
    for (const controller of this.pending) controller.abort();
  }

  dispose(): void {
    this.disposed = true;
    this.cancel();
    this.current = { state: 'cancelled' };
  }

  private async send(
    record: ActivationMetadata,
    controller: AbortController,
  ): Promise<ActivationRecencyResult> {
    try {
      await this.sink?.append(record);
      if (controller.signal.aborted || this.disposed) {
        return this.finish({ state: 'cancelled', record });
      }
      return this.finish({ state: 'appended', record });
    } catch {
      return this.finish({ state: 'sink_failed', record });
    }
  }

  private finish(result: ActivationRecencyResult): ActivationRecencyResult {
    this.current = result;
    return result;
  }
}

function statusFor(
  state: ActivationReporterState,
  outcome: ActivationControllerResult,
  accepted: boolean,
  hostErrorCode?: ProtocolErrorCode,
): ActivationReportStatus {
  return {
    state,
    accepted,
    activationOk: outcome.ok,
    browserConfirmed: outcome.confirmed,
    browserState: outcome.browserState,
    errorCode: outcome.errorCode,
    hostErrorCode,
  };
}

export class ActivationReporter {
  private readonly transport: ActivationReportTransport;
  private readonly recency?: ActivationRecencyHandoff;
  private readonly onState?: (status: ActivationReportStatus) => void;
  private readonly pending = new Set<AbortController>();
  private current: ActivationReportStatus = INITIAL_STATUS;
  private disposed = false;

  constructor(options: ActivationReporterOptions) {
    this.transport = options.transport;
    this.recency = options.recency;
    this.onState = options.onState;
  }

  view(): ActivationReportStatus {
    return this.current;
  }

  isPending(): boolean {
    return this.pending.size > 0;
  }

  report(input: ActivationReportInput): Promise<ActivationReportStatus> {
    if (this.disposed) {
      return Promise.resolve(this.finish('cancelled', input.outcome, false));
    }

    const controller = new AbortController();
    this.pending.add(controller);
    this.transition(statusFor('reporting', input.outcome, false));
    const submission = buildActivationSubmission(input.reference, input.outcome);
    const recency = this.recency?.append({
      reference: input.reference,
      outcome: input.outcome,
      domain: input.domain,
      source: input.source,
      activatedAt: input.activatedAt,
      storageSequence: input.storageSequence,
    }) ?? Promise.resolve(undefined);
    const promise = Promise.all([
      this.send(submission, input.outcome, controller),
      recency,
    ]).then(([status]) => status);
    return promise.finally(() => {
      this.pending.delete(controller);
    });
  }

  /** Cancel pending report work without touching browser state. */
  cancel(): void {
    for (const controller of this.pending) controller.abort();
    this.recency?.cancel();
  }

  /** Reset/uninstall teardown: discard pending work and emit no later updates. */
  dispose(): void {
    this.disposed = true;
    this.cancel();
    this.pending.clear();
    this.recency?.dispose();
    this.current = {
      ...this.current,
      state: 'cancelled',
      accepted: false,
    };
  }

  private async send(
    submission: ActivationSubmission,
    outcome: ActivationControllerResult,
    controller: AbortController,
  ): Promise<ActivationReportStatus> {
    try {
      const result = await this.transport.reportActivation(submission);
      if (controller.signal.aborted || this.disposed) {
        return this.finish('cancelled', outcome, false);
      }
      if (!result.ok) {
        const hostErrorCode = result.source === 'host' ? result.error.code : undefined;
        const state: ActivationReporterState = hostErrorCode === 'PERSISTENCE_DEGRADED'
          ? 'degraded'
          : 'unavailable';
        return this.finish(state, outcome, false, hostErrorCode);
      }
      if (result.value.accepted !== true) {
        return this.finish('degraded', outcome, false);
      }
      return this.finish('acknowledged', outcome, true);
    } catch {
      return this.finish('unavailable', outcome, false);
    }
  }

  private finish(
    state: ActivationReporterState,
    outcome: ActivationControllerResult,
    accepted: boolean,
    hostErrorCode?: ProtocolErrorCode,
  ): ActivationReportStatus {
    const status = statusFor(state, outcome, accepted, hostErrorCode);
    this.transition(status);
    return status;
  }

  private transition(status: ActivationReportStatus): ActivationReportStatus {
    this.current = status;
    if (this.disposed) return status;
    try {
      this.onState?.(status);
    } catch {
      // Status listeners are diagnostics; they cannot change browser authority.
    }
    return status;
  }
}
