// Protocol compatibility note (IP-07): the Chrome/Edge Native Messaging port
// delivers whole JSON objects, so this client never reads or writes the
// host-owned 4-byte length prefix. IP-06 owns framing and the 1 MiB frame cap;
// IP-07's Go codec owns the 256 KiB payload cap and duplicate-key rejection.
// Additive payload fields are tolerated only inside the bounded display,
// counter, and alias-read structures below and only while every decoder stays
// bounded. Unknown envelope fields, unknown message types, unknown error codes,
// and any protocol major other than PROTOCOL_MAJOR fail closed. A breaking
// change increments PROTOCOL_MAJOR and needs a compatibility check on both sides.

import {
  validateTab,
  type ContextKind,
  type EligibleTabRecord,
  type ProfileID,
  type ProjectionEpoch,
  type TabIdentity,
} from '../../domain/index.js';

export const PROTOCOL_MAJOR = 1;

export const PROTOCOL_LIMITS = {
  frameBytes: 1048576,
  payloadBytes: 262144,
  queryScalars: 512,
  snapshotRecords: 10000,
  deltaOperations: 1000,
  results: 50,
  titleScalars: 512,
  domainBytes: 255,
  urlBytes: 2048,
  capabilities: 32,
  typeBytes: 64,
  requestIdBytes: 128,
  profileIdBytes: 128,
  versionBytes: 64,
  resultIdBytes: 128,
  messageKeyBytes: 128,
  displayKeys: 16,
  displayKeyBytes: 64,
  displayValueBytes: 256,
} as const;

export const PROTOCOL_DEADLINES = {
  frameReadMs: 2000,
  handshakeMs: 2000,
  syncAckMs: 1000,
  queryHostBudgetMs: 40,
  queryRequestMs: 100,
  healthMs: 100,
  activationStatusMs: 500,
  shutdownDrainMs: 1000,
} as const;

export const HOST_BOUND_MESSAGE_TYPES = [
  'hello',
  'snapshot',
  'delta',
  'query',
  'health',
  'activation_observed',
  'activation_failed',
] as const;

export const EXTENSION_BOUND_MESSAGE_TYPES = [
  'hello_ack',
  'sync_ack',
  'resync_required',
  'query_result',
  'health_result',
  'activation_ack',
  'error',
] as const;

export const PROTOCOL_ERROR_CODES = [
  'INVALID_FRAME',
  'PROTOCOL_MISMATCH',
  'PROFILE_MISMATCH',
  'PAYLOAD_LIMIT',
  'SNAPSHOT_REQUIRED',
  'REVISION_MISMATCH',
  'INDEX_REBUILDING',
  'QUERY_TIMEOUT',
  'PERSISTENCE_DEGRADED',
  'HOST_SHUTDOWN',
  'INTERNAL_FAILURE',
] as const;

export type HostBoundMessageType = (typeof HOST_BOUND_MESSAGE_TYPES)[number];
export type ExtensionBoundMessageType = (typeof EXTENSION_BOUND_MESSAGE_TYPES)[number];
export type ProtocolErrorCode = (typeof PROTOCOL_ERROR_CODES)[number];

const TERMINAL_ERROR_CODES: ReadonlySet<ProtocolErrorCode> = new Set<ProtocolErrorCode>([
  'INVALID_FRAME',
  'PROTOCOL_MISMATCH',
  'PROFILE_MISMATCH',
]);

const RESYNC_ERROR_CODES: ReadonlySet<ProtocolErrorCode> = new Set<ProtocolErrorCode>([
  'SNAPSHOT_REQUIRED',
  'REVISION_MISMATCH',
]);

const HOST_AVAILABILITY_VALUES: ReadonlySet<string> = new Set(['healthy', 'unavailable', 'recovering']);
const PERSISTENCE_VALUES: ReadonlySet<string> = new Set(['unknown', 'ready', 'degraded', 'disabled']);

export type NativeMessagingClientState =
  | 'Disconnected'
  | 'Connecting'
  | 'Ready'
  | 'Degraded'
  | 'Reconnecting'
  | 'Incompatible'
  | 'Shutdown';

export type ResyncSignalReason =
  | 'revision_mismatch'
  | 'snapshot_required'
  | 'sequence_gap'
  | 'index_rebuilding'
  | 'connection_lost'
  | 'host_shutdown'
  | 'handshake_recovered'
  | 'projection_uncertain';

export type DeltaOperationKind = 'create' | 'update' | 'move' | 'group' | 'pin' | 'activate' | 'remove';

export interface ProtocolEnvelope {
  protocol: number;
  type: string;
  request_id: string;
  profile_id: string;
  projection_revision: number;
  payload: Record<string, unknown>;
}

export interface SnapshotSubmission {
  epoch: ProjectionEpoch;
  records: readonly EligibleTabRecord[];
  sequence: number;
}

export interface DeltaOperation {
  kind: DeltaOperationKind;
  tab_identity: TabIdentity;
  record?: EligibleTabRecord;
}

export interface DeltaSubmission {
  base_revision: number;
  sequence_start: number;
  sequence_end: number;
  operations: readonly DeltaOperation[];
}

export interface QuerySubmission {
  query: string;
  result_limit: number;
  current_window_id: number | null;
}

export interface ActivationSubmission {
  result_id: string;
  tab_id: number;
  projection_revision: number;
  outcome: 'observed' | 'failed';
  error_class?: string;
}

export interface QueryResultReference {
  result_id: string;
  tab_id: number;
  window_id: number;
  display: Record<string, string | number | boolean | null>;
}

export interface QueryResponse {
  results: readonly QueryResultReference[];
  ranking_model_version: string;
  projection_revision: number;
  timing: Record<string, number>;
}

export interface SyncAckResponse {
  accepted_revision: number;
  accepted_sequence: number;
}

export interface HealthResponse {
  host_availability: string;
  host_state: string;
  protocol_state?: string;
  session_state?: string;
  storage_state?: string;
  index_state?: string;
  retryable: boolean;
  projection_revision: number | null;
  projection_sequence: number | null;
  freshness_ms?: number;
  counters?: Record<string, number>;
}

export interface HostProtocolError {
  code: ProtocolErrorCode;
  retryable: boolean;
  message_key: string;
  expected_revision?: number;
  expected_sequence?: number;
}

export type LocalRejectionCode =
  | 'SNAPSHOT_REQUIRED'
  | 'PAYLOAD_LIMIT'
  | 'INVALID_SUBMISSION'
  | 'HOST_UNAVAILABLE'
  | 'NOT_CONNECTED'
  | 'REQUEST_IN_FLIGHT'
  | 'REQUEST_SUPERSEDED'
  | 'DEADLINE_EXCEEDED';

export interface LocalRejection {
  code: LocalRejectionCode;
  retryable: boolean;
}

export type ClientRequestResult<T> =
  | { ok: true; value: T }
  | { ok: false; source: 'host'; error: HostProtocolError }
  | { ok: false; source: 'client'; error: LocalRejection };

export interface NativeMessagingDiagnostics {
  connections_opened: number;
  connections_failed: number;
  frames_sent: number;
  frames_received: number;
  invalid_frames: number;
  uncorrelated_responses: number;
  unsolicited_resync_signals: number;
  requests_sent: number;
  requests_completed: number;
  requests_failed: number;
  requests_timed_out: number;
  resync_signals_emitted: number;
  dropped_results: number;
  errors_by_code: Record<ProtocolErrorCode, number>;
}

export interface NativeMessagingClientView {
  state: NativeMessagingClientState;
  host_available: boolean;
  host_availability: string;
  host_state?: string;
  protocol_state?: string;
  session_state?: string;
  storage_state?: string;
  index_state?: string;
  protocol_version: number;
  host_version?: string;
  extension_version: string;
  ranking_model_version?: string;
  host_acknowledged: boolean;
  acknowledged_revision: number | null;
  acknowledged_sequence: number | null;
  snapshot_required: boolean;
  freshness_ms?: number;
  retryable: boolean;
  resync_reason?: ResyncSignalReason;
  error_code?: ProtocolErrorCode;
  message_key?: string;
  expected_revision?: number;
  expected_sequence?: number;
  diagnostics: NativeMessagingDiagnostics;
}

export type NativeMessagingClientEvent =
  | { kind: 'state'; state: NativeMessagingClientState; view: NativeMessagingClientView }
  | {
      kind: 'resync_required';
      reason: ResyncSignalReason;
      expected_revision?: number;
      expected_sequence?: number;
    }
  | { kind: 'host_error'; code: ProtocolErrorCode; retryable: boolean; message_key: string };

export interface NativePortEvent<T> {
  addListener(listener: (value: T) => void): void;
  removeListener?(listener: (value: T) => void): void;
}

export interface NativeMessagingPort {
  postMessage(message: unknown): void;
  onMessage: NativePortEvent<unknown>;
  onDisconnect: NativePortEvent<unknown>;
  disconnect(): void;
}

export type NativePortFactory = (hostName: string) => NativeMessagingPort;

export interface NativeMessagingClientOptions {
  profileId: ProfileID;
  contextKind: ContextKind;
  hostName: string;
  extensionVersion: string;
  browserFamily: 'chrome' | 'edge';
  browserVersion?: string;
  capabilities?: readonly string[];
  supportedProtocols?: readonly number[];
  createPort?: NativePortFactory;
  limits?: typeof PROTOCOL_LIMITS;
  deadlines?: typeof PROTOCOL_DEADLINES;
  setTimer?: (handler: () => void, timeoutMs: number) => number;
  clearTimer?: (handle: number) => void;
}

type RequestKind = 'hello' | 'snapshot' | 'delta' | 'query' | 'health' | 'activation';

interface PendingRequest {
  requestId: string;
  kind: RequestKind;
  timer: number;
  settle(result: ClientRequestResult<unknown>): void;
}

const ENVELOPE_FIELDS: readonly string[] = [
  'protocol',
  'type',
  'request_id',
  'profile_id',
  'projection_revision',
  'payload',
];

const ALIASES = {
  accepted_protocol: ['accepted_protocol', 'acceptedProtocol', 'protocol'],
  host_availability: ['availability', 'host_state', 'hostAvailability'],
  host_state: ['state', 'lifecycle_state', 'host_state'],
  host_version: ['host_version', 'hostVersion'],
  ranking_model_version: ['ranking_model_version', 'rankingModelVersion'],
  persistence_health: ['persistence_health', 'persistence', 'storage_state'],
  protocol_state: ['protocol_state', 'protocolState'],
  session_state: ['session_state', 'sessionState'],
  storage_state: ['storage_state', 'persistence', 'persistence_health', 'storageState'],
  index_state: ['index_state', 'indexState'],
  retryable: ['retryable'],
  projection_revision: ['projection_revision', 'current_revision', 'revision'],
  projection_sequence: ['projection_sequence', 'current_sequence', 'sequence'],
  freshness_ms: ['freshness_ms', 'freshness_age_ms', 'projection_age_ms', 'freshnessMs'],
  counters: ['counters'],
} as const;

type AliasKey = keyof typeof ALIASES;

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function byteLength(value: string): number {
  return new TextEncoder().encode(value).length;
}

function scalarLength(value: string): number {
  let count = 0;
  for (const character of value) {
    if (character.length > 0) count += 1;
  }
  return count;
}

function isBoundedString(value: unknown, maxBytes: number): value is string {
  return typeof value === 'string' && value.length > 0 && byteLength(value) <= maxBytes;
}

function isNonNegativeInteger(value: unknown): value is number {
  return typeof value === 'number' && Number.isSafeInteger(value) && value >= 0;
}

function readAlias(payload: Record<string, unknown>, key: AliasKey): unknown {
  for (const candidate of ALIASES[key]) {
    if (Object.hasOwn(payload, candidate)) return payload[candidate];
  }
  return undefined;
}

function compact<T extends Record<string, unknown>>(value: T): T {
  const result: Record<string, unknown> = {};
  for (const [key, entry] of Object.entries(value)) {
    if (entry !== undefined) result[key] = entry;
  }
  return result as T;
}

function zeroDiagnostics(): NativeMessagingDiagnostics {
  const errors_by_code = {} as Record<ProtocolErrorCode, number>;
  for (const code of PROTOCOL_ERROR_CODES) errors_by_code[code] = 0;
  return {
    connections_opened: 0,
    connections_failed: 0,
    frames_sent: 0,
    frames_received: 0,
    invalid_frames: 0,
    uncorrelated_responses: 0,
    unsolicited_resync_signals: 0,
    requests_sent: 0,
    requests_completed: 0,
    requests_failed: 0,
    requests_timed_out: 0,
    resync_signals_emitted: 0,
    dropped_results: 0,
    errors_by_code,
  };
}

function defaultBrowserVersion(): string {
  const match = /(?:Edg|Chrome)\/(\d+)/.exec(globalThis.navigator?.userAgent ?? '');
  return match?.[1] ?? '';
}

function defaultPortFactory(): NativePortFactory {
  return hostName => {
    const api = (globalThis as { chrome?: { connectNative?: (name: string) => NativeMessagingPort } }).chrome;
    const connect = api?.connectNative;
    if (!connect) throw new Error('NATIVE_MESSAGING_UNAVAILABLE');
    return connect.call(api, hostName);
  };
}

export function encodeEnvelope(
  type: HostBoundMessageType,
  requestId: string,
  profileId: ProfileID,
  projectionRevision: number,
  payload: Record<string, unknown>,
): ProtocolEnvelope {
  return {
    protocol: PROTOCOL_MAJOR,
    type,
    request_id: requestId,
    profile_id: profileId,
    projection_revision: projectionRevision,
    payload,
  };
}

export type EnvelopeDecodeResult =
  | { ok: true; envelope: ProtocolEnvelope }
  | { ok: false; reason: 'INVALID_FRAME' | 'PROTOCOL_MISMATCH' };

export function decodeEnvelope(value: unknown, limits: typeof PROTOCOL_LIMITS): EnvelopeDecodeResult {
  if (!isRecord(value)) return { ok: false, reason: 'INVALID_FRAME' };
  const keys = Object.keys(value);
  if (keys.length !== ENVELOPE_FIELDS.length) return { ok: false, reason: 'INVALID_FRAME' };
  for (const field of ENVELOPE_FIELDS) {
    if (!Object.hasOwn(value, field)) return { ok: false, reason: 'INVALID_FRAME' };
  }
  if (typeof value.protocol !== 'number' || !Number.isSafeInteger(value.protocol)) {
    return { ok: false, reason: 'INVALID_FRAME' };
  }
  if (value.protocol !== PROTOCOL_MAJOR) return { ok: false, reason: 'PROTOCOL_MISMATCH' };
  if (!isBoundedString(value.type, limits.typeBytes)) return { ok: false, reason: 'INVALID_FRAME' };
  if (!isBoundedString(value.request_id, limits.requestIdBytes)) return { ok: false, reason: 'INVALID_FRAME' };
  if (!isBoundedString(value.profile_id, limits.profileIdBytes)) return { ok: false, reason: 'INVALID_FRAME' };
  if (!isNonNegativeInteger(value.projection_revision)) return { ok: false, reason: 'INVALID_FRAME' };
  if (!isRecord(value.payload)) return { ok: false, reason: 'INVALID_FRAME' };
  return {
    ok: true,
    envelope: {
      protocol: value.protocol,
      type: value.type,
      request_id: value.request_id,
      profile_id: value.profile_id,
      projection_revision: value.projection_revision,
      payload: value.payload,
    },
  };
}

function parseErrorPayload(
  payload: Record<string, unknown>,
  limits: typeof PROTOCOL_LIMITS,
): { ok: true; error: HostProtocolError } | { ok: false } {
  const code = payload.code;
  if (typeof code !== 'string' || !(PROTOCOL_ERROR_CODES as readonly string[]).includes(code)) return { ok: false };
  if (typeof payload.retryable !== 'boolean') return { ok: false };
  if (!isBoundedString(payload.message_key, limits.messageKeyBytes)) return { ok: false };
  const expectedRevision = payload.expected_revision;
  const expectedSequence = payload.expected_sequence;
  if (expectedRevision !== undefined && !isNonNegativeInteger(expectedRevision)) return { ok: false };
  if (expectedSequence !== undefined && !isNonNegativeInteger(expectedSequence)) return { ok: false };
  return {
    ok: true,
    error: {
      code: code as ProtocolErrorCode,
      retryable: payload.retryable,
      message_key: payload.message_key,
      ...(expectedRevision === undefined ? {} : { expected_revision: expectedRevision }),
      ...(expectedSequence === undefined ? {} : { expected_sequence: expectedSequence }),
    },
  };
}

function parseDisplayItem(
  item: Record<string, unknown>,
  limits: typeof PROTOCOL_LIMITS,
): Record<string, string | number | boolean | null> | undefined {
  const display: Record<string, string | number | boolean | null> = {};
  let count = 0;
  for (const [key, value] of Object.entries(item)) {
    if (key === 'result_id' || key === 'tab_id' || key === 'window_id') continue;
    if (count >= limits.displayKeys) return undefined;
    if (byteLength(key) > limits.displayKeyBytes) return undefined;
    if (value === null || typeof value === 'boolean') {
      display[key] = value;
    } else if (typeof value === 'number') {
      if (!Number.isSafeInteger(value)) return undefined;
      display[key] = value;
    } else if (typeof value === 'string') {
      if (byteLength(value) > limits.displayValueBytes) return undefined;
      display[key] = value;
    } else {
      return undefined;
    }
    count += 1;
  }
  return display;
}

function parseQueryResult(
  envelope: ProtocolEnvelope,
  limits: typeof PROTOCOL_LIMITS,
): { ok: true; value: QueryResponse; revision: number } | { ok: false } {
  const payload = envelope.payload;
  const rawResults = payload.results;
  if (!Array.isArray(rawResults) || rawResults.length > limits.results) return { ok: false };
  const declaredRevision = payload.projection_revision;
  if (declaredRevision !== undefined && declaredRevision !== envelope.projection_revision) return { ok: false };
  const revision = envelope.projection_revision;
  const rankingModelVersion = payload.ranking_model_version;
  if (rankingModelVersion !== undefined && !isBoundedString(rankingModelVersion, limits.versionBytes)) {
    return { ok: false };
  }
  const results: QueryResultReference[] = [];
  for (const entry of rawResults) {
    if (!isRecord(entry)) return { ok: false };
    if (!isBoundedString(entry.result_id, limits.resultIdBytes)) return { ok: false };
    if (!isNonNegativeInteger(entry.tab_id) || !isNonNegativeInteger(entry.window_id)) return { ok: false };
    const display = parseDisplayItem(entry, limits);
    if (display === undefined) return { ok: false };
    results.push({ result_id: entry.result_id, tab_id: entry.tab_id, window_id: entry.window_id, display });
  }
  const timing: Record<string, number> = {};
  const rawTiming = payload.timing;
  if (rawTiming !== undefined) {
    if (!isRecord(rawTiming)) return { ok: false };
    for (const [key, value] of Object.entries(rawTiming)) {
      if (!Number.isSafeInteger(value) || (value as number) < 0) return { ok: false };
      timing[key] = value as number;
    }
  }
  return {
    ok: true,
    revision,
    value: {
      results,
      ranking_model_version: rankingModelVersion ?? '',
      projection_revision: revision,
      timing,
    },
  };
}

function optionalBoundedString(payload: Record<string, unknown>, key: AliasKey, maxBytes: number): string | undefined | 'invalid' {
  const value = readAlias(payload, key);
  if (value === undefined) return undefined;
  if (typeof value !== 'string' || byteLength(value) > maxBytes) return 'invalid';
  return value;
}

function parseHealthResult(payload: Record<string, unknown>, limits: typeof PROTOCOL_LIMITS): HealthResponse | undefined {
  const availability = readAlias(payload, 'host_availability');
  const hostState = readAlias(payload, 'host_state');
  if (!isBoundedString(hostState, limits.versionBytes)) return undefined;
  const storageState = readAlias(payload, 'storage_state');
  if (storageState !== undefined && (typeof storageState !== 'string' || !PERSISTENCE_VALUES.has(storageState))) {
    return undefined;
  }
  const retryable = readAlias(payload, 'retryable');
  if (retryable !== undefined && typeof retryable !== 'boolean') return undefined;
  const revision = readAlias(payload, 'projection_revision');
  if (revision !== undefined && !isNonNegativeInteger(revision)) return undefined;
  const sequence = readAlias(payload, 'projection_sequence');
  if (sequence !== undefined && !isNonNegativeInteger(sequence)) return undefined;
  const freshness = readAlias(payload, 'freshness_ms');
  if (freshness !== undefined && !isNonNegativeInteger(freshness)) return undefined;
  const counters: Record<string, number> = {};
  const rawCounters = readAlias(payload, 'counters');
  if (rawCounters !== undefined) {
    if (!isRecord(rawCounters)) return undefined;
    for (const [key, value] of Object.entries(rawCounters)) {
      if (!Number.isSafeInteger(value) || (value as number) < 0) return undefined;
      counters[key] = value as number;
    }
  }
  const protocolState = optionalBoundedString(payload, 'protocol_state', limits.versionBytes);
  const sessionState = optionalBoundedString(payload, 'session_state', limits.versionBytes);
  const indexState = optionalBoundedString(payload, 'index_state', limits.versionBytes);
  if (protocolState === 'invalid' || sessionState === 'invalid' || indexState === 'invalid') return undefined;
  const hostAvailability =
    typeof availability === 'string' && HOST_AVAILABILITY_VALUES.has(availability) ? availability : 'unavailable';
  return compact({
    host_availability: hostAvailability,
    host_state: hostState,
    protocol_state: protocolState,
    session_state: sessionState,
    storage_state: storageState as string | undefined,
    index_state: indexState,
    retryable: retryable === true,
    projection_revision: (revision as number | undefined) ?? null,
    projection_sequence: (sequence as number | undefined) ?? null,
    freshness_ms: freshness as number | undefined,
    counters: Object.keys(counters).length === 0 ? undefined : counters,
  }) as HealthResponse;
}

export class NativeMessagingClient {
  private readonly limits: typeof PROTOCOL_LIMITS;
  private readonly deadlines: typeof PROTOCOL_DEADLINES;
  private readonly createPort: NativePortFactory;
  private readonly setTimer: (handler: () => void, timeoutMs: number) => number;
  private readonly clearTimer: (handle: number) => void;
  private readonly capabilities: readonly string[];
  private readonly supportedProtocols: readonly number[];
  private readonly browserVersion: string;
  private readonly listeners = new Set<(event: NativeMessagingClientEvent) => void>();

  private port: NativeMessagingPort | undefined;
  private portListeners: Array<() => void> = [];
  private state: NativeMessagingClientState = 'Disconnected';
  private hostAcked = false;
  private terminal = false;
  private acknowledgedRevision: number | null = null;
  private acknowledgedSequence: number | null = null;
  private snapshotRequired = true;
  private hostVersion: string | undefined;
  private rankingModelVersion: string | undefined;
  private hostAvailability = 'unavailable';
  private hostState: string | undefined;
  private protocolState: string | undefined;
  private sessionState: string | undefined;
  private storageState: string | undefined;
  private indexState: string | undefined;
  private freshnessMs: number | undefined;
  private resyncReason: ResyncSignalReason | undefined;
  private lastError: HostProtocolError | undefined;
  private activeRequest: PendingRequest | undefined;
  private readonly diagnostics: NativeMessagingDiagnostics = zeroDiagnostics();

  constructor(private readonly options: NativeMessagingClientOptions) {
    this.limits = options.limits ?? PROTOCOL_LIMITS;
    this.deadlines = options.deadlines ?? PROTOCOL_DEADLINES;
    this.createPort = options.createPort ?? defaultPortFactory();
    this.setTimer = options.setTimer ?? ((handler, timeoutMs) => setTimeout(handler, timeoutMs) as unknown as number);
    this.clearTimer = options.clearTimer ?? ((handle: number) => clearTimeout(handle));
    this.capabilities = options.capabilities ?? [];
    this.supportedProtocols = options.supportedProtocols ?? [PROTOCOL_MAJOR];
    this.browserVersion = options.browserVersion ?? defaultBrowserVersion();
  }

  subscribe(listener: (event: NativeMessagingClientEvent) => void): () => void {
    this.listeners.add(listener);
    return () => {
      this.listeners.delete(listener);
    };
  }

  view(): NativeMessagingClientView {
    const retryable = this.lastError?.retryable ?? (this.state === 'Reconnecting' || this.state === 'Degraded');
    return compact({
      state: this.state,
      host_available: this.hostAvailability === 'healthy',
      host_availability: this.hostAvailability,
      host_state: this.hostState,
      protocol_state: this.protocolState,
      session_state: this.sessionState,
      storage_state: this.storageState,
      index_state: this.indexState,
      protocol_version: PROTOCOL_MAJOR,
      host_version: this.hostVersion,
      extension_version: this.options.extensionVersion,
      ranking_model_version: this.rankingModelVersion,
      host_acknowledged: this.hostAcked,
      acknowledged_revision: this.acknowledgedRevision,
      acknowledged_sequence: this.acknowledgedSequence,
      snapshot_required: this.snapshotRequired,
      freshness_ms: this.freshnessMs,
      retryable,
      resync_reason: this.resyncReason,
      error_code: this.lastError?.code,
      message_key: this.lastError?.message_key,
      expected_revision: this.lastError?.expected_revision,
      expected_sequence: this.lastError?.expected_sequence,
      diagnostics: { ...this.diagnostics, errors_by_code: { ...this.diagnostics.errors_by_code } },
    }) as NativeMessagingClientView;
  }

  async start(): Promise<ClientRequestResult<{ accepted_protocol: number; host_version: string }>> {
    if (this.terminal) return rejectLocal('HOST_UNAVAILABLE');
    if (this.port !== undefined) return rejectLocal('REQUEST_IN_FLIGHT');
    this.resetSession();
    this.transition('Connecting');
    let port: NativeMessagingPort;
    try {
      port = this.createPort(this.options.hostName);
    } catch {
      this.diagnostics.connections_failed += 1;
      this.hostAvailability = 'unavailable';
      this.transition('Reconnecting');
      return rejectLocal('HOST_UNAVAILABLE');
    }
    this.port = port;
    this.diagnostics.connections_opened += 1;
    this.attachPort(port);
    return this.sendHello();
  }

  close(): void {
    if (this.state === 'Shutdown') return;
    this.failActive('HOST_SHUTDOWN', true);
    this.teardownPort();
    this.terminal = true;
    this.hostAcked = false;
    this.snapshotRequired = true;
    this.acknowledgedRevision = null;
    this.acknowledgedSequence = null;
    this.transition('Shutdown');
  }

  sendSnapshot(submission: SnapshotSubmission): Promise<ClientRequestResult<SyncAckResponse>> {
    const rejection = this.validateSnapshot(submission);
    if (rejection !== undefined) return Promise.resolve<ClientRequestResult<SyncAckResponse>>(rejection);
    const payload = {
      tabs: submission.records.map(record => wireRecord(record)),
      sequence: submission.sequence,
    };
    return this.dispatch<SyncAckResponse>('snapshot', payload, submission.records.length);
  }

  sendDelta(submission: DeltaSubmission): Promise<ClientRequestResult<SyncAckResponse>> {
    if (!isNonNegativeInteger(submission.base_revision)) return Promise.resolve<ClientRequestResult<SyncAckResponse>>(rejectLocal<SyncAckResponse>('INVALID_SUBMISSION'));
    if (!isNonNegativeInteger(submission.sequence_start)) return Promise.resolve<ClientRequestResult<SyncAckResponse>>(rejectLocal<SyncAckResponse>('INVALID_SUBMISSION'));
    if (!isNonNegativeInteger(submission.sequence_end)) return Promise.resolve<ClientRequestResult<SyncAckResponse>>(rejectLocal<SyncAckResponse>('INVALID_SUBMISSION'));
    if (submission.operations.length > this.limits.deltaOperations) return Promise.resolve<ClientRequestResult<SyncAckResponse>>(rejectLocal<SyncAckResponse>('PAYLOAD_LIMIT'));
    const payload = {
      base_revision: submission.base_revision,
      sequence_start: submission.sequence_start,
      sequence_end: submission.sequence_end,
      operations: submission.operations.map(operation =>
        compact({
          kind: operation.kind,
          tab_identity: { ...operation.tab_identity },
          record: operation.record === undefined ? undefined : wireRecord(operation.record),
        }),
      ),
    };
    const targetRevision = submission.base_revision + submission.operations.length;
    return this.dispatch<SyncAckResponse>('delta', payload, targetRevision);
  }

  query(submission: QuerySubmission): Promise<ClientRequestResult<QueryResponse>> {
    if (typeof submission.query !== 'string') return Promise.resolve<ClientRequestResult<QueryResponse>>(rejectLocal<QueryResponse>('INVALID_SUBMISSION'));
    if (scalarLength(submission.query) > this.limits.queryScalars) return Promise.resolve<ClientRequestResult<QueryResponse>>(rejectLocal<QueryResponse>('PAYLOAD_LIMIT'));
    if (!isNonNegativeInteger(submission.result_limit)) return Promise.resolve<ClientRequestResult<QueryResponse>>(rejectLocal<QueryResponse>('INVALID_SUBMISSION'));
    if (submission.result_limit > this.limits.results) return Promise.resolve<ClientRequestResult<QueryResponse>>(rejectLocal<QueryResponse>('PAYLOAD_LIMIT'));
    if (this.terminal || this.state === 'Incompatible' || this.state === 'Shutdown') {
      return Promise.resolve<ClientRequestResult<QueryResponse>>(rejectLocal<QueryResponse>('HOST_UNAVAILABLE'));
    }
    if (this.port === undefined) return Promise.resolve<ClientRequestResult<QueryResponse>>(rejectLocal<QueryResponse>('NOT_CONNECTED'));
    if (this.acknowledgedRevision === null) return Promise.resolve<ClientRequestResult<QueryResponse>>(rejectLocal<QueryResponse>('SNAPSHOT_REQUIRED'));
    const payload = {
      query: submission.query,
      result_limit: submission.result_limit,
      current_window_id: submission.current_window_id,
    };
    return this.dispatch<QueryResponse>('query', payload, this.acknowledgedRevision);
  }

  health(): Promise<ClientRequestResult<HealthResponse>> {
    return this.dispatch<HealthResponse>('health', {}, this.acknowledgedRevision ?? 0);
  }

  reportActivation(submission: ActivationSubmission): Promise<ClientRequestResult<{ accepted: boolean }>> {
    if (!isBoundedString(submission.result_id, this.limits.resultIdBytes)) {
      return Promise.resolve<ClientRequestResult<{ accepted: boolean }>>(rejectLocal<{ accepted: boolean }>('INVALID_SUBMISSION'));
    }
    if (!isNonNegativeInteger(submission.tab_id)) return Promise.resolve<ClientRequestResult<{ accepted: boolean }>>(rejectLocal<{ accepted: boolean }>('INVALID_SUBMISSION'));
    if (!isNonNegativeInteger(submission.projection_revision)) {
      return Promise.resolve<ClientRequestResult<{ accepted: boolean }>>(rejectLocal<{ accepted: boolean }>('INVALID_SUBMISSION'));
    }
    if (submission.error_class !== undefined && !isBoundedString(submission.error_class, this.limits.versionBytes)) {
      return Promise.resolve<ClientRequestResult<{ accepted: boolean }>>(rejectLocal<{ accepted: boolean }>('PAYLOAD_LIMIT'));
    }
    const payload = {
      result_id: submission.result_id,
      tab_id: submission.tab_id,
      projection_revision: submission.projection_revision,
      outcome: submission.outcome,
      error_class: submission.error_class,
    };
    const type: HostBoundMessageType = submission.outcome === 'observed' ? 'activation_observed' : 'activation_failed';
    return this.dispatch<{ accepted: boolean }>('activation', payload, submission.projection_revision, type);
  }

  private validateSnapshot(submission: SnapshotSubmission): ClientRequestResult<never> | undefined {
    if (submission.records.length > this.limits.snapshotRecords) return rejectLocal('PAYLOAD_LIMIT');
    if (!isNonNegativeInteger(submission.sequence)) return rejectLocal('INVALID_SUBMISSION');
    for (const record of submission.records) {
      if (!validateTab(record, this.options.profileId, this.options.contextKind, submission.epoch)) {
        return rejectLocal('SNAPSHOT_REQUIRED');
      }
    }
    return undefined;
  }

  private resetSession(): void {
    this.hostAcked = false;
    this.acknowledgedRevision = null;
    this.acknowledgedSequence = null;
    this.snapshotRequired = true;
    this.lastError = undefined;
    this.resyncReason = undefined;
    this.hostVersion = undefined;
    this.rankingModelVersion = undefined;
    this.hostAvailability = 'recovering';
    this.hostState = undefined;
    this.protocolState = undefined;
    this.sessionState = undefined;
    this.storageState = undefined;
    this.indexState = undefined;
    this.freshnessMs = undefined;
  }

  private sendHello(): Promise<ClientRequestResult<{ accepted_protocol: number; host_version: string }>> {
    const payload: Record<string, unknown> = {
      extension_version: this.options.extensionVersion,
      browser_family: this.options.browserFamily,
      browser_version: this.browserVersion,
      context_kind: this.options.contextKind,
      capabilities: this.capabilities.slice(0, this.limits.capabilities),
      supported_protocols: this.supportedProtocols.slice(0, this.limits.capabilities),
    };
    return this.dispatch<{ accepted_protocol: number; host_version: string }>('hello', payload, 0, 'hello');
  }

  private deadlineFor(kind: RequestKind): number {
    switch (kind) {
      case 'hello':
        return this.deadlines.handshakeMs;
      case 'snapshot':
      case 'delta':
        return this.deadlines.syncAckMs;
      case 'query':
        return this.deadlines.queryRequestMs;
      case 'health':
        return this.deadlines.healthMs;
      case 'activation':
        return this.deadlines.activationStatusMs;
    }
  }

  private dispatch<T>(
    kind: RequestKind,
    payload: Record<string, unknown>,
    projectionRevision: number,
    type: HostBoundMessageType = kind as HostBoundMessageType,
  ): Promise<ClientRequestResult<T>> {
    if (this.terminal || this.state === 'Incompatible' || this.state === 'Shutdown') {
      return Promise.resolve<ClientRequestResult<T>>(rejectLocal<T>('HOST_UNAVAILABLE'));
    }
    if (this.port === undefined) return Promise.resolve<ClientRequestResult<T>>(rejectLocal<T>('NOT_CONNECTED'));
    if (this.activeRequest !== undefined) {
      if (!(kind === 'query' && this.activeRequest.kind === 'query')) {
        return Promise.resolve<ClientRequestResult<T>>(rejectLocal<T>('REQUEST_IN_FLIGHT'));
      }
      this.settleActive(rejectLocal('REQUEST_SUPERSEDED'));
    }
    const requestId = crypto.randomUUID();
    const envelope = encodeEnvelope(type, requestId, this.options.profileId, projectionRevision, payload);
    if (!this.withinSendBounds(envelope)) {
      return Promise.resolve<ClientRequestResult<T>>({
        ok: false,
        source: 'host',
        error: { code: 'PAYLOAD_LIMIT', retryable: false, message_key: 'PAYLOAD_LIMIT' },
      });
    }
    return new Promise<ClientRequestResult<T>>(resolve => {
      const timer = this.setTimer(() => {
        if (this.activeRequest?.requestId !== requestId) return;
        this.diagnostics.requests_timed_out += 1;
        const failure = rejectLocal('DEADLINE_EXCEEDED');
        this.settleActive(failure);
        this.applyFailure(failure);
      }, this.deadlineFor(kind));
      this.activeRequest = { requestId, kind, timer, settle: result => resolve(result as ClientRequestResult<T>) };
      this.diagnostics.requests_sent += 1;
      try {
        this.port?.postMessage(envelope);
        this.diagnostics.frames_sent += 1;
      } catch {
        this.settleActive(rejectLocal('NOT_CONNECTED'));
      }
    });
  }

  private withinSendBounds(envelope: ProtocolEnvelope): boolean {
    if (this.capabilities.length > this.limits.capabilities) return false;
    if (byteLength(this.options.extensionVersion) > this.limits.versionBytes) return false;
    if (byteLength(this.browserVersion) > this.limits.versionBytes) return false;
    for (const capability of this.capabilities) {
      if (byteLength(capability) > this.limits.versionBytes) return false;
    }
    let payloadBytes: number;
    let frameBytes: number;
    try {
      payloadBytes = byteLength(JSON.stringify(envelope.payload));
      frameBytes = byteLength(JSON.stringify(envelope));
    } catch {
      return false;
    }
    return payloadBytes <= this.limits.payloadBytes && frameBytes <= this.limits.frameBytes;
  }

  private attachPort(port: NativeMessagingPort): void {
    const onMessage = (value: unknown): void => {
      this.handleFrame(value);
    };
    const onDisconnect = (): void => {
      this.handleDisconnect();
    };
    port.onMessage.addListener(onMessage);
    port.onDisconnect.addListener(onDisconnect);
    this.portListeners = [
      () => port.onMessage.removeListener?.(onMessage),
      () => port.onDisconnect.removeListener?.(onDisconnect),
    ];
  }

  private teardownPort(): void {
    for (const remove of this.portListeners) {
      try {
        remove();
      } catch {}
    }
    this.portListeners = [];
    const port = this.port;
    this.port = undefined;
    if (port === undefined) return;
    try {
      port.disconnect();
    } catch {}
  }

  private handleFrame(value: unknown): void {
    this.diagnostics.frames_received += 1;
    const decoded = decodeEnvelope(value, this.limits);
    if (!decoded.ok) {
      this.diagnostics.invalid_frames += 1;
      if (decoded.reason === 'PROTOCOL_MISMATCH') {
        this.enterTerminal('PROTOCOL_MISMATCH', false);
        return;
      }
      return;
    }
    const envelope = decoded.envelope;
    if (envelope.profile_id !== this.options.profileId) {
      this.enterTerminal('PROFILE_MISMATCH', false);
      return;
    }
    if (!(EXTENSION_BOUND_MESSAGE_TYPES as readonly string[]).includes(envelope.type)) {
      this.diagnostics.invalid_frames += 1;
      return;
    }
    if (envelope.type === 'resync_required') {
      this.diagnostics.unsolicited_resync_signals += 1;
      this.handleResyncRequired(envelope.payload);
      return;
    }
    const pending = this.activeRequest;
    if (pending === undefined || pending.requestId !== envelope.request_id) {
      this.diagnostics.uncorrelated_responses += 1;
      return;
    }
    if (!responseMatchesKind(pending.kind, envelope.type)) {
      this.diagnostics.uncorrelated_responses += 1;
      return;
    }
    this.dispatchResponse(pending, envelope);
  }

  private dispatchResponse(pending: PendingRequest, envelope: ProtocolEnvelope): void {
    if (envelope.type === 'error') {
      const parsed = parseErrorPayload(envelope.payload, this.limits);
      if (!parsed.ok) {
        this.diagnostics.invalid_frames += 1;
        this.settleActive({ ok: false, source: 'host', error: { code: 'INVALID_FRAME', retryable: false, message_key: 'INVALID_FRAME' } });
        this.enterTerminal('INVALID_FRAME', false);
        return;
      }
      const failure = { ok: false, source: 'host', error: parsed.error } as const;
      this.settleActive(failure);
      this.applyFailure(failure);
      return;
    }
    if (envelope.type === 'hello_ack') {
      this.settleActive(this.handleHelloAck(envelope.payload));
      return;
    }
    if (envelope.type === 'sync_ack') {
      this.settleActive(this.handleSyncAck(envelope));
      return;
    }
    if (envelope.type === 'query_result') {
      this.settleActive(this.handleQueryResult(envelope));
      return;
    }
    if (envelope.type === 'health_result') {
      this.settleActive(this.handleHealthResult(envelope.payload));
      return;
    }
    this.settleActive({ ok: true, value: { accepted: envelope.payload.accepted === true } });
  }

  private handleHelloAck(
    payload: Record<string, unknown>,
  ): ClientRequestResult<{ accepted_protocol: number; host_version: string }> {
    const acceptedProtocol = readAlias(payload, 'accepted_protocol');
    const hostVersion = readAlias(payload, 'host_version');
    const rankingModelVersion = optionalBoundedString(payload, 'ranking_model_version', this.limits.versionBytes);
    const persistenceHealth = readAlias(payload, 'persistence_health');
    if (!Number.isSafeInteger(acceptedProtocol) || typeof hostVersion !== 'string') {
      return this.failRequest('INVALID_FRAME', false);
    }
    if (byteLength(hostVersion) > this.limits.versionBytes || rankingModelVersion === 'invalid') {
      return this.failRequest('INVALID_FRAME', false);
    }
    if (acceptedProtocol !== PROTOCOL_MAJOR) return this.failRequest('PROTOCOL_MISMATCH', false);
    if (persistenceHealth !== undefined && (typeof persistenceHealth !== 'string' || !PERSISTENCE_VALUES.has(persistenceHealth))) {
      return this.failRequest('INVALID_FRAME', false);
    }
    this.hostAcked = true;
    this.hostVersion = hostVersion;
    this.rankingModelVersion = rankingModelVersion as string | undefined;
    this.snapshotRequired = true;
    this.hostAvailability = 'recovering';
    this.storageState = persistenceHealth as string | undefined;
    const degraded = persistenceHealth === 'degraded' || persistenceHealth === 'disabled';
    this.transition(degraded ? 'Degraded' : 'Connecting');
    return { ok: true, value: { accepted_protocol: acceptedProtocol as number, host_version: hostVersion } };
  }

  private handleSyncAck(envelope: ProtocolEnvelope): ClientRequestResult<SyncAckResponse> {
    const acceptedRevision = envelope.payload.accepted_revision;
    const acceptedSequence = envelope.payload.accepted_sequence;
    if (!isNonNegativeInteger(acceptedRevision) || !isNonNegativeInteger(acceptedSequence)) {
      return this.failRequest('INVALID_FRAME', false);
    }
    if (acceptedRevision !== envelope.projection_revision) {
      const failure = {
        ok: false,
        source: 'host',
        error: { code: 'REVISION_MISMATCH', retryable: true, message_key: 'REVISION_MISMATCH' },
      } as const;
      this.settleActive(failure);
      this.applyFailure(failure);
      return failure;
    }
    this.acknowledgedRevision = acceptedRevision;
    this.acknowledgedSequence = acceptedSequence;
    this.snapshotRequired = false;
    this.resyncReason = undefined;
    this.hostAvailability = 'healthy';
    this.indexState = 'ready';
    const degraded = this.storageState === 'degraded' || this.storageState === 'disabled';
    this.transition(degraded ? 'Degraded' : 'Ready');
    return { ok: true, value: { accepted_revision: acceptedRevision, accepted_sequence: acceptedSequence } };
  }

  private handleQueryResult(envelope: ProtocolEnvelope): ClientRequestResult<QueryResponse> {
    const parsed = parseQueryResult(envelope, this.limits);
    if (!parsed.ok) return this.failRequest('INVALID_FRAME', false);
    if (this.acknowledgedRevision === null || parsed.revision !== this.acknowledgedRevision) {
      this.diagnostics.dropped_results += 1;
      const failure = {
        ok: false,
        source: 'host',
        error: {
          code: 'REVISION_MISMATCH',
          retryable: true,
          message_key: 'REVISION_MISMATCH',
          expected_revision: parsed.revision,
        },
      } as const;
      this.settleActive(failure);
      this.applyFailure(failure);
      return failure;
    }
    return { ok: true, value: parsed.value };
  }

  private handleHealthResult(payload: Record<string, unknown>): ClientRequestResult<HealthResponse> {
    const parsed = parseHealthResult(payload, this.limits);
    if (parsed === undefined) return this.failRequest('INVALID_FRAME', false);
    this.hostAvailability = parsed.host_availability;
    this.hostState = parsed.host_state;
    this.protocolState = parsed.protocol_state;
    this.sessionState = parsed.session_state;
    this.storageState = parsed.storage_state;
    this.indexState = parsed.index_state;
    this.freshnessMs = parsed.freshness_ms;
    if (parsed.host_availability === 'unavailable') {
      this.transition('Degraded');
    } else if (parsed.index_state === 'rebuilding' || parsed.storage_state === 'degraded') {
      this.transition('Degraded');
    }
    return { ok: true, value: parsed };
  }

  private handleResyncRequired(payload: Record<string, unknown>): void {
    const expectedRevision = payload.expected_revision;
    const expectedSequence = payload.expected_sequence;
    this.snapshotRequired = true;
    this.acknowledgedRevision = null;
    this.acknowledgedSequence = null;
    if (expectedRevision !== undefined && isNonNegativeInteger(expectedRevision)) {
      this.lastError = {
        code: 'REVISION_MISMATCH',
        retryable: true,
        message_key: 'REVISION_MISMATCH',
        expected_revision: expectedRevision,
      };
    }
    this.failActive('REVISION_MISMATCH', true);
    this.transition('Reconnecting');
    this.emitResync(mapResyncReason(payload.reason), expectedRevision, expectedSequence);
  }

  private handleDisconnect(): void {
    const wasLive =
      this.state === 'Ready' || this.state === 'Degraded' || this.state === 'Connecting' || this.state === 'Reconnecting';
    this.teardownPort();
    this.hostAcked = false;
    this.snapshotRequired = true;
    this.acknowledgedRevision = null;
    this.acknowledgedSequence = null;
    this.failActive('HOST_SHUTDOWN', true);
    if (this.terminal) return;
    this.hostAvailability = 'unavailable';
    this.hostState = wasLive ? 'unavailable' : this.hostState;
    this.transition(wasLive ? 'Reconnecting' : 'Disconnected');
    this.emitResync('connection_lost');
  }

  private failRequest<T>(code: ProtocolErrorCode, retryable: boolean): ClientRequestResult<T> {
    const failure = { ok: false, source: 'host', error: { code, retryable, message_key: code } } as const;
    this.settleActive(failure);
    this.applyFailure(failure);
    return failure;
  }

  private applyFailure(result: ClientRequestResult<unknown>): void {
    if (result.ok) return;
    if (result.source === 'client') {
      if (result.error.code === 'DEADLINE_EXCEEDED') {
        this.diagnostics.requests_failed += 1;
        this.transition(this.state === 'Ready' ? 'Degraded' : this.state);
      }
      return;
    }
    const { code, retryable, message_key } = result.error;
    this.diagnostics.errors_by_code[code] += 1;
    this.diagnostics.requests_failed += 1;
    this.lastError = result.error;
    this.emit({ kind: 'host_error', code, retryable, message_key });
    if (TERMINAL_ERROR_CODES.has(code)) {
      this.enterTerminal(code, retryable);
      return;
    }
    if (RESYNC_ERROR_CODES.has(code)) {
      this.snapshotRequired = true;
      this.acknowledgedRevision = null;
      this.acknowledgedSequence = null;
      this.transition('Reconnecting');
      this.emitResync(code === 'SNAPSHOT_REQUIRED' ? 'snapshot_required' : 'revision_mismatch', result.error.expected_revision);
      return;
    }
    if (code === 'INDEX_REBUILDING') {
      this.indexState = 'rebuilding';
      this.hostAvailability = 'recovering';
      this.transition('Degraded');
      this.emitResync('index_rebuilding');
      return;
    }
    if (code === 'PERSISTENCE_DEGRADED') {
      this.storageState = 'degraded';
      this.transition('Degraded');
      return;
    }
    if (code === 'QUERY_TIMEOUT') {
      this.transition('Degraded');
      return;
    }
    if (code === 'HOST_SHUTDOWN') {
      this.teardownPort();
      this.hostAcked = false;
      this.snapshotRequired = true;
      this.acknowledgedRevision = null;
      this.acknowledgedSequence = null;
      this.hostAvailability = 'unavailable';
      this.transition('Shutdown');
      this.emitResync('host_shutdown');
      return;
    }
    this.transition('Degraded');
  }

  private enterTerminal(code: ProtocolErrorCode, retryable: boolean): void {
    this.diagnostics.errors_by_code[code] += 1;
    this.lastError = { code, retryable, message_key: code };
    this.terminal = true;
    this.hostAcked = false;
    this.snapshotRequired = true;
    this.acknowledgedRevision = null;
    this.acknowledgedSequence = null;
    this.hostAvailability = 'unavailable';
    this.failActive(code, retryable);
    this.teardownPort();
    this.transition('Incompatible');
    this.emit({ kind: 'host_error', code, retryable, message_key: code });
  }

  private failActive(code: ProtocolErrorCode, retryable: boolean): void {
    if (this.activeRequest === undefined) return;
    this.diagnostics.requests_failed += 1;
    this.settleActive({ ok: false, source: 'host', error: { code, retryable, message_key: code } });
  }

  private settleActive(result: ClientRequestResult<unknown>): void {
    const pending = this.activeRequest;
    if (pending === undefined) return;
    this.activeRequest = undefined;
    this.clearTimer(pending.timer);
    if (result.ok) this.diagnostics.requests_completed += 1;
    pending.settle(result);
  }

  private transition(next: NativeMessagingClientState): void {
    if (this.state === next) return;
    this.state = next;
    this.emit({ kind: 'state', state: next, view: this.view() });
  }

  private emitResync(reason: ResyncSignalReason, expectedRevision?: unknown, expectedSequence?: unknown): void {
    this.resyncReason = reason;
    this.diagnostics.resync_signals_emitted += 1;
    this.emit(
      compact({
        kind: 'resync_required' as const,
        reason,
        expected_revision: isNonNegativeInteger(expectedRevision) ? expectedRevision : undefined,
        expected_sequence: isNonNegativeInteger(expectedSequence) ? expectedSequence : undefined,
      }) as NativeMessagingClientEvent,
    );
  }

  private emit(event: NativeMessagingClientEvent): void {
    for (const listener of this.listeners) {
      try {
        listener(event);
      } catch {}
    }
  }
}

function wireRecord(record: EligibleTabRecord): Record<string, unknown> {
  return { ...record } as unknown as Record<string, unknown>;
}

function rejectLocal<T = never>(code: LocalRejectionCode, retryable = true): ClientRequestResult<T> {
  return { ok: false, source: 'client', error: { code, retryable } };
}

function responseMatchesKind(kind: RequestKind, type: string): boolean {
  switch (kind) {
    case 'hello':
      return type === 'hello_ack';
    case 'snapshot':
    case 'delta':
      return type === 'sync_ack' || type === 'error';
    case 'query':
      return type === 'query_result' || type === 'error';
    case 'health':
      return type === 'health_result' || type === 'error';
    case 'activation':
      return type === 'activation_ack' || type === 'error';
  }
}

function mapResyncReason(value: unknown): ResyncSignalReason {
  if (typeof value === 'string') {
    if (value.includes('sequence')) return 'sequence_gap';
    if (value.includes('revision')) return 'revision_mismatch';
    if (value.includes('snapshot')) return 'snapshot_required';
    if (value.includes('rebuild') || value.includes('index')) return 'index_rebuilding';
  }
  return 'projection_uncertain';
}