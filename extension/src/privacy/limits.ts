// IP-17 input limits and secret-shape policy for extension-side inputs.
//
// The numeric values come from IP-07's PROTOCOL_LIMITS; this module adds the
// product operating envelope and the 1..50 result-limit rule.

import { PROTOCOL_LIMITS } from '../runtime/native-messaging-client.js';

export const OPERATING_TAB_LIMIT = 1000;
export const MIN_RESULT_LIMIT = 1;
export const MAX_RESULT_LIMIT = PROTOCOL_LIMITS.results;

export type LimitCode = 'PAYLOAD_LIMIT' | 'INVALID_FRAME';

export type LimitDecision<T> =
  | { readonly ok: true; readonly value: T }
  | { readonly ok: false; readonly code: LimitCode; readonly bound?: string };

export interface QueryInputLike {
  readonly query: string;
  readonly resultLimit?: number;
  readonly currentWindowId?: number | null;
  readonly currentGroupId?: number | null;
}

export type SecretKind =
  | 'auth_header'
  | 'cookie'
  | 'token'
  | 'private_key'
  | 'credential'
  | 'environment_secret';

export interface SecretFinding {
  readonly field: string;
  readonly kind: SecretKind;
}

const SECRET_FIELDS: Record<string, SecretKind> = {
  authorization: 'auth_header',
  auth_header: 'auth_header',
  'proxy-authorization': 'auth_header',
  cookie: 'cookie',
  'set-cookie': 'cookie',
  token: 'token',
  access_token: 'token',
  refresh_token: 'token',
  session_token: 'token',
  secret: 'credential',
  password: 'credential',
  credential: 'credential',
  api_key: 'credential',
  apikey: 'credential',
  private_key: 'private_key',
  environment_secret: 'environment_secret',
  env_secret: 'environment_secret',
};

const AUTH_HEADER = /\b(?:bearer|basic)\s+[A-Za-z0-9._~+/=-]{8,}/i;
const JWT = /\beyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\b/;
const KNOWN_TOKEN = /\b(?:ghp_|github_pat_|sk-|AKIA)[A-Za-z0-9_-]{8,}\b/;
const PRIVATE_KEY = /-----BEGIN [A-Z ]*PRIVATE KEY-----/;
const PASSWORD_PAIR = /\b(?:password|passwd|token|secret)=[^&\s]+/i;

function safeInteger(value: unknown): value is number {
  return typeof value === 'number' && Number.isSafeInteger(value);
}

export function validateFrameBytes(size: number): LimitDecision<number> {
  if (!safeInteger(size) || size < 0 || size > PROTOCOL_LIMITS.frameBytes) {
    return { ok: false, code: 'PAYLOAD_LIMIT', bound: `frame_bytes<=${PROTOCOL_LIMITS.frameBytes}` };
  }
  return { ok: true, value: size };
}

export function validatePayloadBytes(size: number): LimitDecision<number> {
  if (!safeInteger(size) || size < 0 || size > PROTOCOL_LIMITS.payloadBytes) {
    return { ok: false, code: 'PAYLOAD_LIMIT', bound: `payload_bytes<=${PROTOCOL_LIMITS.payloadBytes}` };
  }
  return { ok: true, value: size };
}

export function validateSnapshotCount(count: number): LimitDecision<number> {
  if (!safeInteger(count) || count < 0 || count > OPERATING_TAB_LIMIT) {
    return { ok: false, code: 'PAYLOAD_LIMIT', bound: `operating_tabs<=${OPERATING_TAB_LIMIT}` };
  }
  return { ok: true, value: count };
}

export function validateDeltaCount(count: number): LimitDecision<number> {
  if (!safeInteger(count) || count < 0 || count > PROTOCOL_LIMITS.deltaOperations) {
    return { ok: false, code: 'PAYLOAD_LIMIT', bound: `delta_operations<=${PROTOCOL_LIMITS.deltaOperations}` };
  }
  return { ok: true, value: count };
}

export function validateHTTPURL(raw: string): LimitDecision<string> {
  try {
    const parsed = new URL(raw);
    if ((parsed.protocol !== 'http:' && parsed.protocol !== 'https:') || parsed.hostname === '') {
      return { ok: false, code: 'INVALID_FRAME' };
    }
    return { ok: true, value: raw };
  } catch {
    return { ok: false, code: 'INVALID_FRAME' };
  }
}

export function validateQueryInput(input: QueryInputLike): LimitDecision<number> {
  if (typeof input.query !== 'string') return { ok: false, code: 'INVALID_FRAME' };
  if ([...input.query].length > PROTOCOL_LIMITS.queryScalars) {
    return { ok: false, code: 'PAYLOAD_LIMIT', bound: `query_scalars<=${PROTOCOL_LIMITS.queryScalars}` };
  }
  if (/[\u0000-\u001f\u007f]/.test(input.query)) return { ok: false, code: 'INVALID_FRAME' };
  for (const value of [input.currentWindowId, input.currentGroupId]) {
    if (value !== undefined && value !== null && (!safeInteger(value) || value < 0)) {
      return { ok: false, code: 'INVALID_FRAME' };
    }
  }
  const resultLimit = input.resultLimit ?? 10;
  if (!safeInteger(resultLimit) || resultLimit < MIN_RESULT_LIMIT || resultLimit > MAX_RESULT_LIMIT) {
    return { ok: false, code: 'PAYLOAD_LIMIT', bound: `result_limit<=${MAX_RESULT_LIMIT}` };
  }
  return { ok: true, value: resultLimit };
}

export function containsSecretShapedValue(value: string): boolean {
  return AUTH_HEADER.test(value)
    || JWT.test(value)
    || KNOWN_TOKEN.test(value)
    || PRIVATE_KEY.test(value)
    || PASSWORD_PAIR.test(value);
}

export function detectSecret(field: string, value: unknown): SecretFinding[] {
  const findings: SecretFinding[] = [];
  const kind = SECRET_FIELDS[field.toLowerCase()];
  if (kind) findings.push({ field, kind });
  if (typeof value !== 'string') return findings;
  if (AUTH_HEADER.test(value)) findings.push({ field, kind: 'auth_header' });
  else if (JWT.test(value) || KNOWN_TOKEN.test(value)) findings.push({ field, kind: 'token' });
  else if (PRIVATE_KEY.test(value)) findings.push({ field, kind: 'private_key' });
  else if (PASSWORD_PAIR.test(value)) findings.push({ field, kind: 'credential' });
  return findings;
}

export function rejectSecrets(fields: Readonly<Record<string, unknown>>): SecretFinding[] {
  const findings: SecretFinding[] = [];
  for (const [field, value] of Object.entries(fields)) findings.push(...detectSecret(field, value));
  return findings;
}
