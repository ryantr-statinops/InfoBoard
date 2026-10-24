// Query client for the focused search surface (IP-12). It owns the local input
// sequence, refuses to render a response that no longer matches the latest
// input/profile/revision, and exposes a bounded render model to IP-13.

import type { ProfileID } from '../../domain/index.js';
import type {
  ClientRequestResult,
  HostProtocolError,
  QueryResponse,
  QuerySubmission,
} from './native-messaging-client.js';

export type QueryClientStatus =
  | 'idle'
  | 'loading'
  | 'results'
  | 'empty_query'
  | 'no_results'
  | 'cancelled'
  | 'superseded'
  | 'stale'
  | 'invalid'
  | 'revision_mismatch'
  | 'rebuilding'
  | 'snapshot_required'
  | 'unavailable'
  | 'timeout'
  | 'degraded'
  | 'error';

export interface QueryRenderRow {
  resultId: string;
  tabId: number;
  windowId: number;
  groupId?: number;
  title: string;
  domain: string;
  windowLabel?: string;
  groupLabel?: string;
  pinned: boolean;
  score?: number;
  bestClass?: string;
  explanations: readonly string[];
}

export interface QueryRenderError {
  source: 'host' | 'client';
  code: string;
  retryable: boolean;
  expectedRevision?: number;
  expectedSequence?: number;
}

export interface QueryRenderModel {
  profileId: ProfileID;
  sequence: number;
  status: QueryClientStatus;
  rows: readonly QueryRenderRow[];
  projectionRevision: number | null;
  rankingModelVersion?: string;
  acceptedLimit?: number;
  indexState?: string;
  persistenceState?: string;
  freshnessAgeMs?: number;
  timing?: Readonly<Record<string, number>>;
  error?: QueryRenderError;
}

export interface QueryInput {
  query: string;
  resultLimit?: number;
  currentWindowId?: number | null;
  currentGroupId?: number | null;
}

export interface QueryTransportView {
  acknowledged_revision: number | null;
}

export interface QueryTransport {
  view(): QueryTransportView;
  query(submission: QuerySubmission): Promise<ClientRequestResult<QueryResponse>>;
}

export interface QueryClientOptions {
  profileId: ProfileID;
  transport: QueryTransport;
}

export type QueryClientListener = (model: QueryRenderModel) => void;

export class QueryClient {
  private sequence = 0;
  private latestSequence = 0;
  private state: QueryRenderModel;
  private readonly listeners = new Set<QueryClientListener>();

  constructor(private readonly options: QueryClientOptions) {
    this.state = this.emptyModel('idle', 0, options.transport.view().acknowledged_revision);
  }

  subscribe(listener: QueryClientListener): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }

  view(): QueryRenderModel {
    return {
      ...this.state,
      rows: this.state.rows.map(row => ({ ...row, explanations: [...row.explanations] })),
      timing: this.state.timing === undefined ? undefined : { ...this.state.timing },
      error: this.state.error === undefined ? undefined : { ...this.state.error },
    };
  }

  cancel(): QueryRenderModel {
    this.sequence += 1;
    this.latestSequence = this.sequence;
    this.state = this.emptyModel('cancelled', this.sequence, this.options.transport.view().acknowledged_revision);
    this.emit();
    return this.view();
  }

  async search(input: QueryInput): Promise<QueryRenderModel> {
    const invalid = this.validate(input);
    if (invalid !== undefined) return invalid;

    const sequence = ++this.sequence;
    this.latestSequence = sequence;
    const requestedRevision = this.options.transport.view().acknowledged_revision;
    this.state = this.emptyModel('loading', sequence, requestedRevision);
    this.emit();

    const result = await this.options.transport.query({
      query: input.query,
      result_limit: input.resultLimit,
      current_window_id: input.currentWindowId ?? null,
      current_group_id: input.currentGroupId,
      query_sequence: sequence,
    });

    if (sequence !== this.latestSequence) {
      return this.supersededModel(sequence, requestedRevision);
    }

    if (!result.ok) {
      return this.applyFailure(sequence, requestedRevision, result);
    }

    const currentRevision = this.options.transport.view().acknowledged_revision;
    if (
      requestedRevision === null ||
      result.value.projection_revision !== requestedRevision ||
      currentRevision !== requestedRevision ||
      result.value.query_sequence !== sequence
    ) {
      this.state = this.emptyModel('stale', sequence, currentRevision);
      this.emit();
      return this.view();
    }

    const rows = result.value.results.map(row => renderRow(row));
    const status: QueryClientStatus = result.value.status;
    this.state = {
      profileId: this.options.profileId,
      sequence,
      status,
      rows,
      projectionRevision: result.value.projection_revision,
      rankingModelVersion: result.value.ranking_model_version,
      acceptedLimit: result.value.accepted_limit,
      indexState: result.value.index_state,
      persistenceState: result.value.persistence_state,
      freshnessAgeMs: result.value.freshness_age_ms,
      timing: { ...result.value.timing },
    };
    this.emit();
    return this.view();
  }

  private validate(input: QueryInput): QueryRenderModel | undefined {
    if (typeof input.query !== 'string') {
      return this.invalidModel('INVALID_SUBMISSION');
    }
    if (input.resultLimit !== undefined && !isNonNegativeInteger(input.resultLimit)) {
      return this.invalidModel('INVALID_SUBMISSION');
    }
    if (input.currentWindowId !== undefined && input.currentWindowId !== null && !isNonNegativeInteger(input.currentWindowId)) {
      return this.invalidModel('INVALID_SUBMISSION');
    }
    if (input.currentGroupId !== undefined && input.currentGroupId !== null && !isNonNegativeInteger(input.currentGroupId)) {
      return this.invalidModel('INVALID_SUBMISSION');
    }
    return undefined;
  }

  private applyFailure(
    sequence: number,
    projectionRevision: number | null,
    result: Exclude<ClientRequestResult<QueryResponse>, { ok: true }>,
  ): QueryRenderModel {
    const status = result.source === 'client' ? clientFailureStatus(result.error.code) : hostFailureStatus(result.error);
    const model: QueryRenderModel = {
      ...this.emptyModel(status, sequence, projectionRevision),
      error: result.source === 'client'
        ? { source: 'client', code: result.error.code, retryable: result.error.retryable }
        : {
            source: 'host',
            code: result.error.code,
            retryable: result.error.retryable,
            expectedRevision: result.error.expected_revision,
            expectedSequence: result.error.expected_sequence,
          },
    };
    this.state = model;
    this.emit();
    return this.view();
  }

  private invalidModel(code: string): QueryRenderModel {
    return {
      ...this.emptyModel('invalid', this.sequence, this.options.transport.view().acknowledged_revision),
      error: { source: 'client', code, retryable: false },
    };
  }

  private supersededModel(sequence: number, projectionRevision: number | null): QueryRenderModel {
    return this.emptyModel('superseded', sequence, projectionRevision);
  }

  private emptyModel(status: QueryClientStatus, sequence: number, projectionRevision: number | null): QueryRenderModel {
    return {
      profileId: this.options.profileId,
      sequence,
      status,
      rows: [],
      projectionRevision,
    };
  }

  private emit(): void {
    const model = this.view();
    for (const listener of this.listeners) listener(model);
  }
}

function renderRow(row: QueryResponse['results'][number]): QueryRenderRow {
  const display = row.display;
  const groupId = numberValue(display.group_id);
  const score = numberValue(display.score);
  return {
    resultId: row.result_id,
    tabId: row.tab_id,
    windowId: row.window_id,
    groupId,
    title: stringValue(display.title_display),
    domain: stringValue(display.domain_display),
    windowLabel: optionalStringValue(display.window_label_display),
    groupLabel: optionalStringValue(display.group_label_display),
    pinned: display.pinned === true,
    score,
    bestClass: optionalStringValue(display.best_class),
    explanations: [...row.explanations],
  };
}

function stringValue(value: string | number | boolean | null | undefined): string {
  return typeof value === 'string' ? value : '';
}

function optionalStringValue(value: string | number | boolean | null | undefined): string | undefined {
  return typeof value === 'string' && value !== '' ? value : undefined;
}

function numberValue(value: string | number | boolean | null | undefined): number | undefined {
  return typeof value === 'number' && Number.isSafeInteger(value) && value >= 0 ? value : undefined;
}

function isNonNegativeInteger(value: unknown): value is number {
  return typeof value === 'number' && Number.isSafeInteger(value) && value >= 0;
}

function clientFailureStatus(code: string): QueryClientStatus {
  switch (code) {
    case 'REQUEST_SUPERSEDED':
      return 'superseded';
    case 'DEADLINE_EXCEEDED':
      return 'timeout';
    case 'HOST_UNAVAILABLE':
    case 'NOT_CONNECTED':
      return 'unavailable';
    case 'SNAPSHOT_REQUIRED':
      return 'snapshot_required';
    case 'PAYLOAD_LIMIT':
    case 'INVALID_SUBMISSION':
      return 'invalid';
    default:
      return 'error';
  }
}

function hostFailureStatus(error: HostProtocolError): QueryClientStatus {
  switch (error.code) {
    case 'REVISION_MISMATCH':
      return 'revision_mismatch';
    case 'INDEX_REBUILDING':
      return 'rebuilding';
    case 'QUERY_TIMEOUT':
      return 'timeout';
    case 'HOST_SHUTDOWN':
      return 'unavailable';
    case 'PERSISTENCE_DEGRADED':
      return 'degraded';
    case 'SNAPSHOT_REQUIRED':
      return 'snapshot_required';
    case 'INVALID_FRAME':
    case 'PROFILE_MISMATCH':
    case 'PAYLOAD_LIMIT':
      return 'invalid';
    default:
      return 'error';
  }
}
