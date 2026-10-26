// IP-14 surface handoff.
//
// The IP-13 surface supplies only the selected result and its revision-bound
// fields. This module rebuilds the activation reference from the current query
// view, runs the pure guard, then delegates browser mutation to the exact
// single-flight controller. Browser confirmation remains authoritative.

import type { BrowserProfileContext } from '../browser/browser-adapter.js';
import {
  ActivationController,
  ChromeActivationBrowser,
  type ActivationBrowser,
  type ActivationControllerRequest,
  type ActivationControllerResult,
} from '../browser/activation-controller.js';
import {
  guardActivation,
  type ActivationGuardRejection,
  type ActivationProjectionRecord,
  type ActivationReference,
} from '../browser/activation-guard.js';
import {
  parseProjectionEpoch,
  parseRevision,
  tabIdentityKey,
  type ProjectionEpoch,
  type ProjectionRevision,
} from '../../domain/index.js';
import {
  ActivationReporter,
  type ActivationReportTransport,
} from '../runtime/activation-reporter.js';
import type { QueryRenderModel } from '../runtime/query-client.js';
import type {
  ActivationHandoff,
  ActivationNextAction,
  ActivationOutcome,
  ActivationRequest,
} from './surface-controller.js';
import type { SearchQuerySource } from './surface-controller.js';

export interface ActivationHandoffOptions {
  readonly context: BrowserProfileContext;
  readonly querySource: Pick<SearchQuerySource, 'view'>;
  readonly closeSurface: () => void;
  readonly browser?: ActivationBrowser;
  readonly transport?: ActivationReportTransport;
  readonly reporter?: ActivationReporter;
}

export type ActivationGuardBuild =
  | { readonly ok: true; readonly input: ActivationControllerRequest }
  | { readonly ok: false; readonly outcome: ActivationGuardRejection };

function stale(outcome: ActivationGuardRejection): ActivationGuardBuild {
  return { ok: false, outcome };
}

/**
 * Rebuild the guard input from the current model instead of trusting display
 * fields. A missing/moved row, revision change, or changed query sequence is
 * rejected before any browser call.
 */
export function buildActivationGuardInput(
  request: ActivationRequest,
  model: QueryRenderModel,
  context: BrowserProfileContext,
): ActivationGuardBuild {
  if (model.profileId !== context.profileId) return stale('profile_mismatch');
  if (model.status !== 'results'
    && model.status !== 'empty_query'
    && model.status !== 'degraded') {
    return stale('stale_selection');
  }
  if (model.sequence !== request.querySequence) return stale('stale_selection');
  if (model.projectionRevision === null
    || model.projectionEpoch === undefined
    || model.projectionEpoch === null) {
    return stale('projection_rebuilding');
  }
  if (model.projectionRevision !== request.projectionRevision
    || model.projectionEpoch !== request.projectionEpoch) {
    return stale('stale_selection');
  }

  const row = model.rows.find(candidate => candidate.resultId === request.resultId);
  if (row === undefined) return stale('tab_missing');
  if (row.tabId !== request.tabId || row.windowId !== request.windowId) {
    return stale('stale_selection');
  }

  let epoch: ProjectionEpoch;
  let revision: ProjectionRevision;
  try {
    epoch = parseProjectionEpoch(model.projectionEpoch);
    revision = parseRevision(model.projectionRevision);
  } catch {
    return stale('projection_rebuilding');
  }

  const identity = {
    profile_id: context.profileId,
    context_kind: context.contextKind,
    tab_id: row.tabId,
  };
  const reference: ActivationReference = {
    profile_id: context.profileId,
    context_kind: context.contextKind,
    query_sequence: request.querySequence,
    result_id: row.resultId,
    projection_epoch: epoch,
    projection_revision: revision,
    tab_id: row.tabId,
    window_id: row.windowId,
    tab_identity: identity,
  };

  const records = new Map<string, ActivationProjectionRecord>();
  for (const candidate of model.rows) {
    const candidateIdentity = {
      profile_id: context.profileId,
      context_kind: context.contextKind,
      tab_id: candidate.tabId,
    };
    records.set(tabIdentityKey(candidateIdentity), {
      tab_identity: candidateIdentity,
      window_id: candidate.windowId,
      eligible: true,
      projection_epoch: epoch,
      projection_revision: revision,
      result_id: candidate.resultId,
    });
  }

  const input: ActivationControllerRequest = {
    reference,
    surface: {
      profile_id: context.profileId,
      context_kind: context.contextKind,
      query_sequence: request.querySequence,
      selected_result_id: row.resultId,
    },
    projection: {
      profile_id: context.profileId,
      context_kind: context.contextKind,
      epoch,
      revision,
      records,
    },
  };
  const guard = guardActivation(input);
  if (guard.outcome !== 'valid') return stale(guard.outcome);
  return { ok: true, input };
}

function nextActionFor(result: ActivationControllerResult): ActivationNextAction {
  if (result.ok && result.browserState === 'tab_active') return 'dismiss';
  switch (result.errorCode) {
    case 'stale_selection':
    case 'profile_mismatch':
    case 'context_mismatch':
    case 'tab_missing':
    case 'window_missing':
    case 'tab_reused':
    case 'ineligible':
    case 'projection_rebuilding':
      return 'refresh';
    case 'activation_in_progress':
      return 'none';
    default:
      return 'retry';
  }
}

function outcomeFor(
  result: ActivationControllerResult,
  reporterConfigured: boolean,
): ActivationOutcome {
  return {
    ok: result.ok && result.browserState === 'tab_active',
    confirmed: result.confirmed,
    browserState: result.browserState,
    nextAction: nextActionFor(result),
    errorCode: result.errorCode,
    reportState: reporterConfigured ? 'pending' : 'unavailable',
  };
}

function rejectedOutcome(outcome: ActivationGuardRejection): ActivationOutcome {
  return {
    ok: false,
    confirmed: true,
    browserState: 'unchanged',
    nextAction: 'refresh',
    errorCode: outcome,
    reportState: 'unavailable',
  };
}

export function createActivationHandoff(options: ActivationHandoffOptions): ActivationHandoff {
  const reporter = options.reporter
    ?? (options.transport === undefined
      ? undefined
      : new ActivationReporter({ transport: options.transport }));
  let activeRequest: ActivationRequest | null = null;
  const controller = new ActivationController({
    browser: options.browser ?? new ChromeActivationBrowser(),
    revalidate: () => {
      if (activeRequest === null) return null;
      const rebuilt = buildActivationGuardInput(
        activeRequest,
        options.querySource.view(),
        options.context,
      );
      return rebuilt.ok ? rebuilt.input : null;
    },
  });

  return {
    async activate(request: ActivationRequest): Promise<ActivationOutcome> {
      const current = options.querySource.view();
      const built = buildActivationGuardInput(request, current, options.context);
      if (!built.ok) return rejectedOutcome(built.outcome);

      const ownsActiveRequest = !controller.isPending();
      if (ownsActiveRequest) activeRequest = request;
      let result: ActivationControllerResult;
      try {
        result = await controller.activate(built.input);
      } catch {
        return {
          ok: false,
          confirmed: true,
          browserState: 'unknown',
          nextAction: 'retry',
          errorCode: 'browser_operation_failed',
          reportState: reporter === undefined ? 'unavailable' : 'pending',
        };
      } finally {
        if (ownsActiveRequest) activeRequest = null;
      }

      const outcome = outcomeFor(result, reporter !== undefined);
      if (result.ok && result.browserState === 'tab_active') {
        const selected = current.rows.find(row => row.resultId === request.resultId);
        if (reporter !== undefined) {
          // Start serialization before closing the popup; the report is never
          // awaited by the browser-confirmation path.
          void reporter.report({
            reference: built.input.reference,
            outcome: result,
            domain: selected?.domain,
            source: 'keyboard_enter',
          });
        }
        try {
          options.closeSurface();
        } catch {
          // Browser activation is already authoritative; popup teardown is
          // best effort and must not turn success into a raw failure.
        }
        return outcome;
      }

      if (reporter !== undefined
        && result.errorCode !== 'activation_in_progress'
        && result.errorCode !== 'activation_cancelled') {
        void reporter.report({
          reference: built.input.reference,
          outcome: result,
          domain: current.rows.find(row => row.resultId === request.resultId)?.domain,
          source: 'keyboard_enter',
        });
      }
      return outcome;
    },
  };
}
