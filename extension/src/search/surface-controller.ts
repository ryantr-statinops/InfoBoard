import type { ContextKind, ProfileID } from '../../domain/index.js';
import type { QueryInput, QueryRenderModel, QueryRenderRow } from '../runtime/query-client.js';
import {
  canActivate,
  initialSurfaceState,
  permittedActions,
  reduceSurfaceState,
  selectedRow,
  type SurfaceState,
} from './surface-state.js';
import { findSurfaceElements, renderSurface, type SurfaceElements } from './surface-renderer.js';

export interface SearchQuerySource {
  view(): QueryRenderModel;
  subscribe(listener: (model: QueryRenderModel) => void): () => void;
  search(input: QueryInput): Promise<QueryRenderModel>;
  cancel(): QueryRenderModel;
}

export type ActivationBrowserState = 'unchanged' | 'window_focused' | 'tab_active' | 'unknown';
export type ActivationNextAction = 'dismiss' | 'retry' | 'refresh' | 'none';
export type ActivationReportState =
  | 'not_configured'
  | 'pending'
  | 'acknowledged'
  | 'degraded'
  | 'unavailable'
  | 'cancelled';

export interface ActivationRequest {
  resultId: string;
  tabId: number;
  windowId: number;
  projectionRevision: number;
  projectionEpoch: string;
  querySequence: number;
}

export interface ActivationOutcome {
  ok: boolean;
  confirmed: boolean;
  browserState: ActivationBrowserState;
  nextAction: ActivationNextAction;
  errorCode?: string;
  reportState?: ActivationReportState;
}

export type ActivationTeardownReason = 'reset' | 'uninstall' | 'browser_shutdown';
export type ActivationCleanupState = 'not_configured' | 'complete' | 'degraded';

export interface ActivationCleanupContext {
  reason: ActivationTeardownReason;
  profileId: ProfileID;
  contextKind: ContextKind;
}

export interface ActivationTeardownResult {
  reason: ActivationTeardownReason;
  cancelled: boolean;
  reportsDiscarded: boolean;
  cleanup: ActivationCleanupState;
}

export interface ActivationHandoff {
  activate(request: ActivationRequest): Promise<ActivationOutcome>;
  /** Reset/uninstall teardown; never performs a tab or window mutation. */
  teardown?(reason?: ActivationTeardownReason): Promise<ActivationTeardownResult>;
}

export interface SurfaceControllerOptions {
  doc: Document;
  querySource: SearchQuerySource;
  activation: ActivationHandoff;
  dismiss: () => void;
  now?: () => number;
  commandStartedAt?: number;
  focusTimestamp?: number;
  onState?: (state: SurfaceState) => void;
}

export class SurfaceController {
  private readonly doc: Document;
  private readonly querySource: SearchQuerySource;
  private readonly activation: ActivationHandoff;
  private readonly dismissSurface: () => void;
  private readonly now: () => number;
  private readonly commandStartedAt?: number;
  private readonly focusTimestamp?: number;
  private readonly onState?: (state: SurfaceState) => void;
  private readonly elements: SurfaceElements;
  private readonly cleanups: Array<() => void> = [];
  private state: SurfaceState = initialSurfaceState();
  private requestSequence = 0;
  private lastQuery: string | undefined;

  constructor(options: SurfaceControllerOptions) {
    this.doc = options.doc;
    this.querySource = options.querySource;
    this.activation = options.activation;
    this.dismissSurface = options.dismiss;
    this.now = options.now ?? (() => Date.now());
    this.commandStartedAt = options.commandStartedAt;
    this.focusTimestamp = options.focusTimestamp;
    this.onState = options.onState;
    const elements = findSurfaceElements(this.doc);
    if (!elements) throw new Error('search surface markup is missing required elements');
    this.elements = elements;
  }

  start(): void {
    this.state = reduceSurfaceState(this.state, { type: 'open' });
    this.bind();
    this.render();
    this.elements.input.value = '';
    this.cleanups.push(this.querySource.subscribe(model => this.applyModel(model)));
    this.doc.defaultView?.requestAnimationFrame(() => {
      this.focusInput();
      const focusTimestamp = this.focusTimestamp ?? this.now();
      const shortcutFocusMs = this.commandStartedAt === undefined
        ? undefined
        : Math.max(0, focusTimestamp - this.commandStartedAt);
      this.dispatch({ type: 'ready', focusTimestamp, shortcutFocusMs });
    });
  }

  stop(): void {
    for (const cleanup of this.cleanups.splice(0)) cleanup();
  }

  view(): SurfaceState {
    return this.state;
  }

  showPrivateUnavailable(): void {
    this.dispatch({ type: 'private_unavailable' });
  }

  focusInput(): void {
    try {
      this.elements.input.focus({ preventScroll: true });
    } catch {
      // The surface remains dismissible even if focus is temporarily denied.
    }
  }

  submit(value: string): void {
    const query = normalizeQuery(value);
    if (query === this.lastQuery) return;
    this.lastQuery = query;
    const sequence = ++this.requestSequence;
    this.dispatch({ type: 'query', query, sequence });
    void this.querySource.search({ query }).then(model => this.applyModel(model)).catch(error => {
      this.dispatch({ type: 'error', code: safeErrorCode(error), retryable: true });
    });
  }

  retry(): void {
    this.lastQuery = undefined;
    this.submit(this.elements.input.value);
  }

  dismiss(): void {
    if (!this.state.dismissed) {
      try { this.querySource.cancel(); } catch { /* dismissal must still succeed */ }
      this.dispatch({ type: 'dismiss' });
    }
    this.dismissSurface();
  }

  selectNext(): void {
    if (this.state.rows.length === 0) return;
    const next = this.state.selectedIndex < 0 ? 0 : this.state.selectedIndex + 1;
    this.dispatch({ type: 'select', index: Math.min(next, this.state.rows.length - 1) });
  }

  selectPrevious(): void {
    if (this.state.rows.length === 0) return;
    const previous = this.state.selectedIndex < 0 ? 0 : this.state.selectedIndex - 1;
    this.dispatch({ type: 'select', index: Math.max(previous, 0) });
  }

  selectFirst(): void {
    if (this.state.rows.length > 0) this.dispatch({ type: 'select', index: 0 });
  }

  selectLast(): void {
    if (this.state.rows.length > 0) this.dispatch({ type: 'select', index: this.state.rows.length - 1 });
  }

  async activateSelected(): Promise<void> {
    if (!canActivate(this.state)) return;
    const row = selectedRow(this.state);
    const revision = this.state.projectionRevision;
    const epoch = this.state.projectionEpoch;
    if (row === undefined || revision === null || epoch === null) return;
    this.dispatch({ type: 'activation_started' });
    try {
      const outcome = await this.activation.activate({
        resultId: row.resultId,
        tabId: row.tabId,
        windowId: row.windowId,
        projectionRevision: revision,
        projectionEpoch: epoch,
        querySequence: this.state.sequence,
      });
      this.dispatch({
        type: 'activation_result',
        ok: outcome.ok,
        confirmed: outcome.confirmed,
        browserState: outcome.browserState,
        nextAction: outcome.nextAction,
        errorCode: outcome.errorCode,
        reportState: outcome.reportState,
      });
    } catch {
      this.dispatch({
        type: 'activation_result',
        ok: false,
        confirmed: true,
        browserState: 'unknown',
        nextAction: 'retry',
        errorCode: 'activation_failed',
        reportState: 'unavailable',
      });
    }
  }

  private bind(): void {
    const onInput = () => this.submit(this.elements.input.value);
    const onKeyDown = (event: KeyboardEvent) => this.onKeyDown(event);
    this.elements.input.addEventListener('input', onInput);
    this.elements.input.addEventListener('keydown', onKeyDown);
    this.cleanups.push(() => this.elements.input.removeEventListener('input', onInput));
    this.cleanups.push(() => this.elements.input.removeEventListener('keydown', onKeyDown));
    if (this.elements.close) {
      const onClose = () => this.dismiss();
      this.elements.close.addEventListener('click', onClose);
      this.cleanups.push(() => this.elements.close?.removeEventListener('click', onClose));
    }
  }

  private onKeyDown(event: KeyboardEvent): void {
    if (event.key === 'Escape') {
      event.preventDefault();
      this.dismiss();
      return;
    }
    if (event.key === 'ArrowDown') {
      event.preventDefault();
      this.selectNext();
      return;
    }
    if (event.key === 'ArrowUp') {
      event.preventDefault();
      this.selectPrevious();
      return;
    }
    if (event.key === 'Home') {
      event.preventDefault();
      this.selectFirst();
      return;
    }
    if (event.key === 'End') {
      event.preventDefault();
      this.selectLast();
      return;
    }
    if (event.key === 'Enter') {
      event.preventDefault();
      void this.activateSelected();
    }
  }

  private applyModel(model: QueryRenderModel): void {
    if (model.sequence < this.state.sequence) return;
    this.dispatch({ type: 'model', model });
  }

  private dispatch(event: Parameters<typeof reduceSurfaceState>[1]): void {
    this.state = reduceSurfaceState(this.state, event);
    this.render();
    this.onState?.(this.state);
  }

  private render(): void {
    renderSurface(this.doc, this.state);
    const actions = permittedActions(this.state);
    this.doc.documentElement.dataset.surfaceActions = actions.join(' ');
  }
}

function normalizeQuery(value: string): string {
  return value.trim();
}

function safeErrorCode(error: unknown): string {
  if (typeof error === 'object' && error !== null && 'code' in error && typeof error.code === 'string') {
    return error.code.slice(0, 64);
  }
  return 'QUERY_FAILED';
}

export type { QueryRenderRow };
