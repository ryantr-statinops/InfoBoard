// Profile-scoped settings contract shared by the settings surface and the
// native-messaging lifecycle client. This module owns validation and safe
// defaults; it never reads or mutates browser state.

import { parseProfileID, type ProfileID } from '../../domain/index.js';

export type Density = 'compact' | 'comfortable';
export type Theme = 'system' | 'light' | 'dark';
export type PersistenceState = 'ready' | 'degraded' | 'unavailable' | 'unknown';
export type ConfigKey =
  | 'result_limit'
  | 'density'
  | 'theme'
  | 'context_labels'
  | 'recency_enabled';

export const CONFIG_KEYS: readonly ConfigKey[] = Object.freeze([
  'result_limit',
  'density',
  'theme',
  'context_labels',
  'recency_enabled',
]);

export const RESULT_LIMIT_MIN = 1;
export const RESULT_LIMIT_MAX = 50;
export const MAX_RANKING_MODEL_VERSION_BYTES = 128;

export interface ProfileConfig {
  profile_id: ProfileID;
  result_limit: number;
  density: Density;
  theme: Theme;
  context_labels: boolean;
  recency_enabled: boolean;
  protocol_version: number;
  ranking_model_version: string;
}

export interface EffectiveProfileConfig {
  configuration: ProfileConfig;
  persistence: PersistenceState;
  durable: boolean;
}

export const DEFAULT_CONFIG = Object.freeze({
  result_limit: 10,
  density: 'comfortable' as Density,
  theme: 'system' as Theme,
  context_labels: true,
  recency_enabled: true,
});

export type ConfigValidationCode = 'invalid_configuration' | 'profile_mismatch';
export type ConfigValidationField =
  | ConfigKey
  | 'profile_id'
  | 'protocol_version'
  | 'ranking_model_version'
  | 'configuration';

export class ConfigValidationError extends Error {
  constructor(
    readonly field: ConfigValidationField,
    readonly code: ConfigValidationCode = 'invalid_configuration',
  ) {
    super(code);
    this.name = 'ConfigValidationError';
  }
}

function invalid(field: ConfigValidationField): never {
  throw new ConfigValidationError(field);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function boundedString(value: unknown, maxBytes: number, allowEmpty: boolean): value is string {
  return (
    typeof value === 'string' &&
    (allowEmpty || value.length > 0) &&
    new TextEncoder().encode(value).length <= maxBytes
  );
}

export function defaultProfileConfig(
  profileId: ProfileID,
  protocolVersion: number,
  rankingModelVersion: string,
): ProfileConfig {
  return validateProfileConfig({
    profile_id: profileId,
    ...DEFAULT_CONFIG,
    protocol_version: protocolVersion,
    ranking_model_version: rankingModelVersion,
  });
}

export function validateProfileConfig(
  value: unknown,
  expectedProfile?: ProfileID,
): ProfileConfig {
  if (!isRecord(value)) invalid('configuration');

  const allowed = new Set<string>([
    'profile_id',
    'result_limit',
    'density',
    'theme',
    'context_labels',
    'recency_enabled',
    'protocol_version',
    'ranking_model_version',
  ]);
  for (const key of Object.keys(value)) {
    if (!allowed.has(key)) invalid('configuration');
  }

  let profileId: ProfileID;
  try {
    profileId = parseProfileID(value.profile_id);
  } catch {
    return invalid('profile_id');
  }
  if (expectedProfile !== undefined && profileId !== expectedProfile) {
    throw new ConfigValidationError('profile_id', 'profile_mismatch');
  }

  if (
    typeof value.result_limit !== 'number' ||
    !Number.isSafeInteger(value.result_limit) ||
    value.result_limit < RESULT_LIMIT_MIN ||
    value.result_limit > RESULT_LIMIT_MAX
  ) {
    invalid('result_limit');
  }
  if (value.density !== 'compact' && value.density !== 'comfortable') invalid('density');
  if (value.theme !== 'system' && value.theme !== 'light' && value.theme !== 'dark') invalid('theme');
  if (typeof value.context_labels !== 'boolean') invalid('context_labels');
  if (typeof value.recency_enabled !== 'boolean') invalid('recency_enabled');
  if (
    typeof value.protocol_version !== 'number' ||
    !Number.isSafeInteger(value.protocol_version) ||
    value.protocol_version <= 0
  ) {
    invalid('protocol_version');
  }
  if (!boundedString(value.ranking_model_version, MAX_RANKING_MODEL_VERSION_BYTES, true)) {
    invalid('ranking_model_version');
  }

  return {
    profile_id: profileId,
    result_limit: value.result_limit,
    density: value.density,
    theme: value.theme,
    context_labels: value.context_labels,
    recency_enabled: value.recency_enabled,
    protocol_version: value.protocol_version,
    ranking_model_version: value.ranking_model_version,
  };
}

export function serializeProfileConfig(config: ProfileConfig): ProfileConfig {
  return validateProfileConfig(config);
}

export function isConfigKey(value: unknown): value is ConfigKey {
  return typeof value === 'string' && (CONFIG_KEYS as readonly string[]).includes(value);
}
