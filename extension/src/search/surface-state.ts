// Pure surface state for the focused search popup. The controller owns DOM and
// browser effects; this module keeps state transitions, permitted actions and
// safe announcements deterministic and easy to test.

import type { QueryClientStatus, QueryRenderModel, QueryRenderRow } from '../runtime/query-client.js';
import type {
  ActivationBrowserState,
  ActivationNextAction,
  ActivationReportState,
} from './surface-controller.js';

export type SurfaceStatus =
  | 'opening'
  | 'ready'
  | 'querying'
  | 'results'
  | 'empty_query'
  | 'no_results'
  | 'rebuilding'
  | 'unavailable'
  | 'stale'
  | 'persistence_degraded'
  | 'activation_success'
  | 'activation_failure'
  | 'private_unavailable'
  | 'invalid';

export type SurfaceAction =
  | 'focus-input'
  | 'select-next'
  | 'select-previous'
  | 'activate'
  | 'dismiss'
  | 'retry'
  | 'clear';

export interface SurfaceActivationState {
  pending: boolean;
  ok?: boolean;
  confirmed: boolean;
  browserState: ActivationBrowserState;
  nextAction: ActivationNextAction;
  reportState: ActivationReportState;
  errorCode?: string;
}

export interface SurfaceState {
  status: SurfaceStatus;
  rows: readonly QueryRenderRow[];
  selectedIndex: number;
  query: string;
  sequence: number;
  projectionRevision: number | null;
  projectionEpoch: string | null;
  persistenceState?: string;
  indexState?: string;
  errorCode?: string;
  errorRetryable?: boolean;
  browserStateConfirmed: boolean;
  activation: SurfaceActivationState;
  dismissed: boolean;
  focusTimestamp?: number;
  shortcutFocusMs?: number;
}

export type SurfaceEvent =
  | { type: 'open' }
  | { type: 'ready'; focusTimestamp?: number; shortcutFocusMs?: number }
  | { type: 'private_unavailable' }
  | { type: 'query'; query: string; sequence: number }
  | { type: 'model'; model: QueryRenderModel }
  | { type: 'select'; index: number }
  | { type: 'activation_started' }
  | {
      type: 'activation_result';
      ok: boolean;
      confirmed: boolean;
      browserState?: ActivationBrowserState;
      nextAction?: ActivationNextAction;
      errorCode?: string;
      reportState?: ActivationReportState;
    }
  | { type: 'error'; code: string; retryable: boolean }
  | { type: 'dismiss' };

const SAFE_ACTIVATION_ERROR = 'activation_failed';

function idleActivation(): SurfaceActivationState {
  return {
    pending: false,
    confirmed: true,
    browserState: 'unchanged',
    nextAction: 'none',
    reportState: 'not_configured',
  };
}

export function initialSurfaceState(): SurfaceState {
  return {
    status: 'opening',
    rows: [],
    selectedIndex: -1,
    query: '',
    sequence: 0,
    projectionRevision: null,
    projectionEpoch: null,
    browserStateConfirmed: true,
    activation: idleActivation(),
    dismissed: false,
  };
}

export function reduceSurfaceState(state: SurfaceState, event: SurfaceEvent): SurfaceState {
  if (state.dismissed && event.type !== 'open') return state;

  switch (event.type) {
    case 'open':
      return initialSurfaceState();
    case 'ready':
      if (state.status !== 'opening' && state.status !== 'ready') return state;
      return {
        ...state,
        status: 'ready',
        focusTimestamp: event.focusTimestamp ?? state.focusTimestamp,
        shortcutFocusMs: event.shortcutFocusMs ?? state.shortcutFocusMs,
      };
    case 'private_unavailable':
      return {
        ...state,
        status: 'private_unavailable',
        rows: [],
        selectedIndex: -1,
        projectionEpoch: null,
        browserStateConfirmed: true,
        activation: idleActivation(),
      };
    case 'query':
      return {
        ...state,
        status: 'querying',
        rows: [],
        selectedIndex: -1,
        query: event.query,
        sequence: event.sequence,
        projectionEpoch: null,
        errorCode: undefined,
        errorRetryable: undefined,
        browserStateConfirmed: true,
        activation: idleActivation(),
      };
    case 'model':
      return reduceModel(state, event.model);
    case 'select': {
      if (state.rows.length === 0) return { ...state, selectedIndex: -1 };
      const selectedIndex = Math.max(0, Math.min(event.index, state.rows.length - 1));
      return { ...state, selectedIndex };
    }
    case 'activation_started':
      if (!canActivate(state)) return state;
      return {
        ...state,
        browserStateConfirmed: false,
        activation: {
          pending: true,
          confirmed: false,
          browserState: 'unchanged',
          nextAction: 'none',
          reportState: 'not_configured',
        },
      };
    case 'activation_result': {
      const errorCode = event.ok ? undefined : event.errorCode ?? SAFE_ACTIVATION_ERROR;
      return {
        ...state,
        status: event.ok ? 'activation_success' : 'activation_failure',
        browserStateConfirmed: event.confirmed,
        activation: {
          pending: false,
          ok: event.ok,
          confirmed: event.confirmed,
          browserState: event.browserState
            ?? (event.ok ? 'tab_active' : event.confirmed ? 'unchanged' : 'unknown'),
          nextAction: event.nextAction ?? (event.ok ? 'dismiss' : 'retry'),
          reportState: event.reportState ?? 'not_configured',
          errorCode,
        },
      };
    }
    case 'error':
      return {
        ...state,
        status: 'unavailable',
        rows: [],
        selectedIndex: -1,
        errorCode: event.code,
        errorRetryable: event.retryable,
        browserStateConfirmed: true,
        activation: idleActivation(),
      };
    case 'dismiss':
      return {
        ...state,
        dismissed: true,
        browserStateConfirmed: true,
        activation: idleActivation(),
      };
  }
}

function reduceModel(state: SurfaceState, model: QueryRenderModel): SurfaceState {
  // A late response from an older local revision must never replace the
  // visible result. QueryClient already rejects stale projection revisions;
  // this guard protects the surface from any injected or duplicated source.
  if (model.sequence < state.sequence) return state;
  if (model.status === 'superseded' || model.status === 'cancelled') return state;

  if (model.status === 'loading') {
    return {
      ...state,
      status: 'querying',
      rows: [],
      selectedIndex: -1,
      projectionRevision: model.projectionRevision,
      projectionEpoch: model.projectionEpoch ?? null,
      browserStateConfirmed: true,
    };
  }

  const status = statusForQuery(model.status);
  const rows = model.rows;
  return {
    ...state,
    status,
    rows,
    selectedIndex: rows.length === 0 ? -1 : Math.min(Math.max(state.selectedIndex, 0), rows.length - 1),
    sequence: model.sequence,
    projectionRevision: model.projectionRevision,
    projectionEpoch: model.projectionEpoch ?? null,
    persistenceState: model.persistenceState,
    indexState: model.indexState,
    errorCode: model.error?.code,
    errorRetryable: model.error?.retryable,
    browserStateConfirmed: true,
    activation: idleActivation(),
  };
}

function statusForQuery(status: QueryClientStatus): SurfaceStatus {
  switch (status) {
    case 'idle':
      return 'ready';
    case 'loading':
      return 'querying';
    case 'results':
      return 'results';
    case 'empty_query':
      return 'empty_query';
    case 'no_results':
      return 'no_results';
    case 'rebuilding':
      return 'rebuilding';
    case 'stale':
    case 'revision_mismatch':
      return 'stale';
    case 'degraded':
      return 'persistence_degraded';
    case 'invalid':
      return 'invalid';
    case 'unavailable':
    case 'timeout':
    case 'snapshot_required':
    case 'error':
      return 'unavailable';
    case 'cancelled':
    case 'superseded':
      return 'stale';
  }
}

export function permittedActions(state: SurfaceState): readonly SurfaceAction[] {
  if (state.dismissed) return [];
  if (state.status === 'private_unavailable') return ['dismiss'];

  const actions: SurfaceAction[] = ['focus-input', 'dismiss'];
  if (state.query !== '') actions.push('clear');
  if (
    state.status === 'unavailable' ||
    state.status === 'rebuilding' ||
    state.status === 'stale' ||
    state.status === 'invalid' ||
    state.status === 'activation_failure'
  ) {
    actions.push('retry');
  }
  if (canActivate(state)) {
    actions.push('select-next', 'select-previous', 'activate');
  }
  return actions;
}

export function canActivate(state: SurfaceState): boolean {
  if (state.activation.pending || state.rows.length === 0 || state.selectedIndex < 0) return false;
  return (
    state.status === 'results' ||
    state.status === 'empty_query' ||
    state.status === 'persistence_degraded'
  );
}

export function selectedRow(state: SurfaceState): QueryRenderRow | undefined {
  if (state.selectedIndex < 0 || state.selectedIndex >= state.rows.length) return undefined;
  return state.rows[state.selectedIndex];
}

export function statusMessage(state: SurfaceState): string {
  switch (state.status) {
    case 'opening':
      return 'Opening search';
    case 'ready':
      return 'Type to search open tabs';
    case 'querying':
      return 'Searching';
    case 'results':
      return resultCountMessage(state.rows.length);
    case 'empty_query':
      return state.rows.length === 0 ? 'No recent tabs available' : 'Recent and related tabs';
    case 'no_results':
      return 'No matching open tabs';
    case 'rebuilding':
      return 'Search index is rebuilding. Try again shortly.';
    case 'unavailable':
      return state.errorRetryable === false
        ? 'Search is unavailable. Close and reopen the search popup.'
        : 'Search is unavailable. Try again.';
    case 'stale':
      return 'Results are stale. Type again or retry.';
    case 'persistence_degraded':
      return state.rows.length === 0
        ? 'History is unavailable. Search results remain available.'
        : `${resultCountMessage(state.rows.length)}. History is unavailable; results are current for this session.`;
    case 'activation_success':
      return 'Opened selected tab';
    case 'activation_failure':
      switch (state.activation.errorCode) {
        case 'stale_selection':
        case 'projection_rebuilding':
          return 'Results changed. Refresh and choose a tab again.';
        case 'tab_missing':
        case 'window_missing':
        case 'tab_reused':
        case 'ineligible':
          return 'Selected tab is no longer available. Refresh and choose another.';
        case 'profile_mismatch':
        case 'context_mismatch':
          return 'This result belongs to another profile. Refresh the search.';
        case 'browser_permission_denied':
        case 'browser_unsupported':
          return 'The browser blocked opening the tab. Check permissions and retry.';
        case 'activation_cancelled':
          return 'Activation was cancelled. Try again.';
        case 'activation_in_progress':
          return 'Activation is already in progress.';
        default:
          return 'Could not open selected tab';
      }
    case 'private_unavailable':
      return 'Search is unavailable in private windows.';
    case 'invalid':
      return 'Search text is invalid or too long.';
  }
}

function resultCountMessage(count: number): string {
  return count === 1 ? '1 result' : `${count} results`;
}
