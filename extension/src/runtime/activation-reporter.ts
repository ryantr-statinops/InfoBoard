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

export interface ActivationReportInput {
  readonly reference: ActivationReference;
  readonly outcome: ActivationControllerResult;
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
  private readonly onState?: (status: ActivationReportStatus) => void;
  private readonly pending = new Set<AbortController>();
  private current: ActivationReportStatus = INITIAL_STATUS;
  private disposed = false;

  constructor(options: ActivationReporterOptions) {
    this.transport = options.transport;
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
    const promise = this.send(submission, input.outcome, controller);
    return promise.finally(() => {
      this.pending.delete(controller);
    });
  }

  /** Cancel pending report work without touching browser state. */
  cancel(): void {
    for (const controller of this.pending) controller.abort();
  }

  /** Reset/uninstall teardown: cancel pending work and emit no later updates. */
  dispose(): void {
    this.disposed = true;
    this.cancel();
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
    try {
      this.onState?.(status);
    } catch {
      // Status listeners are diagnostics; they cannot change browser authority.
    }
    return status;
  }
}
