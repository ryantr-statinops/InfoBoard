// IP-17 observable-output redaction.
//
// The extension may show current-scope title/domain/URL context, but diagnostics,
// errors, and exports contain only the explicit allowlist below.

export const DIAGNOSTIC_ALLOWLIST = [
  'schema_version',
  'event_id',
  'event_time',
  'session_id',
  'profile_scope',
  'component',
  'from_state',
  'to_state',
  'state',
  'reason_code',
  'error_class',
  'status',
  'recommended_action',
  'exhausted_action',
  'retryable',
  'duration_ms',
  'projection_revision',
  'indexed_revision',
  'protocol_version',
  'ranking_model_version',
  'host_version',
  'counts',
  'retention_count',
  'retention_bytes',
] as const;

export type DiagnosticField = (typeof DIAGNOSTIC_ALLOWLIST)[number];
const ALLOWED = new Set<string>(DIAGNOSTIC_ALLOWLIST);

export interface SafeErrorOutput {
  readonly error_class: string;
  readonly retryable: boolean;
  readonly recommended_action: string;
}

export function safeError(
  errorClass: string,
  retryable: boolean,
  recommendedAction = 'none',
): SafeErrorOutput {
  return {
    error_class: boundedEnum(errorClass) ? errorClass : 'runtime_recoverable_error',
    retryable: retryable === true,
    recommended_action: boundedEnum(recommendedAction) ? recommendedAction : 'none',
  };
}

function boundedEnum(value: string): boolean {
  return typeof value === 'string' && /^[a-z][a-z0-9_]{0,63}$/.test(value);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function safeValue(key: string, value: unknown): unknown {
  switch (key) {
    case 'schema_version':
    case 'duration_ms':
    case 'projection_revision':
    case 'indexed_revision':
    case 'protocol_version':
    case 'retention_count':
    case 'retention_bytes':
      return typeof value === 'number' && Number.isSafeInteger(value) && value >= 0 ? value : undefined;
    case 'retryable':
      return typeof value === 'boolean' ? value : undefined;
    case 'counts':
      if (!isRecord(value)) return undefined;
      return Object.fromEntries(
        Object.entries(value)
          .filter(([nestedKey, nestedValue]) =>
            ['records', 'dropped', 'retries', 'failures'].includes(nestedKey)
            && typeof nestedValue === 'number'
            && Number.isSafeInteger(nestedValue)
            && nestedValue >= 0)
          .sort(([left], [right]) => left.localeCompare(right)),
      );
    case 'event_id':
    case 'session_id':
      return typeof value === 'string' && /^[a-z0-9][a-z0-9._:-]{0,95}$/.test(value) ? value : undefined;
    case 'profile_scope':
      return typeof value === 'string' && /^local-sha256:[0-9a-f]{16}$/.test(value) ? value : undefined;
    case 'component':
    case 'from_state':
    case 'to_state':
    case 'state':
    case 'reason_code':
    case 'error_class':
    case 'status':
    case 'recommended_action':
    case 'exhausted_action':
    case 'ranking_model_version':
    case 'host_version':
      return typeof value === 'string' && boundedEnum(value) ? value : undefined;
    default:
      return undefined;
  }
}

export function redactDiagnosticRecord(input: Record<string, unknown>): Record<string, unknown> {
  const output: Record<string, unknown> = {};
  for (const key of DIAGNOSTIC_ALLOWLIST) {
    if (!ALLOWED.has(key)) continue;
    const value = safeValue(key, input[key]);
    if (value !== undefined) output[key] = value;
  }
  return output;
}

export function redactDiagnosticRecords(input: readonly Record<string, unknown>[]): Record<string, unknown>[] {
  return input.map(redactDiagnosticRecord);
}

export function displayURL(raw: string): string {
  try {
    const parsed = new URL(raw);
    parsed.username = '';
    parsed.password = '';
    parsed.search = '';
    parsed.hash = '';
    return parsed.toString();
  } catch {
    return '';
  }
}

export async function localDigest(value: string): Promise<string> {
  const bytes = new TextEncoder().encode(value);
  const digest = await crypto.subtle.digest('SHA-256', bytes);
  const hex = [...new Uint8Array(digest)].map((byte) => byte.toString(16).padStart(2, '0')).join('');
  return `local-sha256:${hex.slice(0, 16)}`;
}

export function assertNoSentinels(output: unknown, sentinels: readonly string[]): void {
  const serialized = JSON.stringify(output);
  for (const sentinel of sentinels) {
    if (sentinel.length > 0 && serialized.includes(sentinel)) throw new TypeError('SENSITIVE_OUTPUT');
  }
}
