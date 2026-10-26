// Fail-closed activation guard for IP-14.
//
// The guard is intentionally pure: it never calls a browser API and never
// returns titles, URLs, query text, or raw browser errors. A caller may only
// proceed to the activation controller when the outcome is `valid`.

import {
  tabIdentityKey,
  type ContextKind,
  type ProfileID,
  type ProjectionEpoch,
  type ProjectionRevision,
  type TabIdentity,
} from '../../domain/index.js';

export const ACTIVATION_GUARD_OUTCOMES = [
  'valid',
  'stale_selection',
  'profile_mismatch',
  'context_mismatch',
  'tab_missing',
  'window_missing',
  'tab_reused',
  'ineligible',
  'projection_rebuilding',
] as const;

export type ActivationGuardOutcome = (typeof ACTIVATION_GUARD_OUTCOMES)[number];
export type ActivationGuardRejection = Exclude<ActivationGuardOutcome, 'valid'>;

export interface ActivationReference {
  readonly profile_id: ProfileID;
  readonly context_kind: ContextKind;
  readonly query_sequence: number;
  readonly result_id: string;
  readonly projection_epoch: ProjectionEpoch;
  readonly projection_revision: ProjectionRevision;
  readonly tab_id: number;
  readonly window_id: number;
  readonly tab_identity: TabIdentity;
}

export interface ActivationSurfaceView {
  readonly profile_id: ProfileID;
  readonly context_kind: ContextKind;
  readonly query_sequence: number;
  readonly selected_result_id: string;
}

// The activation guard only needs the projection fields that participate in
// identity/fence validation. `EligibleTabRecord` is structurally compatible,
// while IP-13's query view can provide the same shape without exposing titles
// or URLs to the guard.
export interface ActivationProjectionRecord {
  readonly tab_identity: TabIdentity;
  readonly window_id: number;
  readonly eligible: boolean;
  readonly projection_epoch: ProjectionEpoch;
  readonly projection_revision: ProjectionRevision;
  readonly result_id?: string;
}

export interface ActivationProjectionView {
  readonly profile_id: ProfileID;
  readonly context_kind: ContextKind;
  readonly epoch: ProjectionEpoch | null;
  readonly revision: ProjectionRevision;
  readonly records: ReadonlyMap<string, ActivationProjectionRecord>;
}

export interface ActivationGuardInput {
  readonly reference: ActivationReference;
  readonly surface: ActivationSurfaceView;
  readonly projection: ActivationProjectionView;
}

export interface ActivationGuardResult {
  readonly outcome: ActivationGuardOutcome;
}

const MAX_RESULT_ID_BYTES = 128;
const MAX_EPOCH_BYTES = 128;

function reject(outcome: ActivationGuardRejection): ActivationGuardResult {
  return { outcome };
}

function isNonNegativeSafeInteger(value: unknown): value is number {
  return typeof value === 'number' && Number.isSafeInteger(value) && value >= 0;
}

function isBoundedOpaque(value: unknown, maxBytes: number): value is string {
  return typeof value === 'string'
    && value.length > 0
    && new TextEncoder().encode(value).length <= maxBytes;
}

function sameIdentity(left: TabIdentity, right: TabIdentity): boolean {
  return left.profile_id === right.profile_id
    && left.context_kind === right.context_kind
    && left.tab_id === right.tab_id;
}

/**
 * Validate the selected result against the current surface and projection.
 *
 * Precedence is deliberate: profile/context fences and projection readiness
 * are checked before query freshness and tab identity, so a stale result from
 * another partition can never be mistaken for a same-profile tab miss.
 */
export function guardActivation(input: ActivationGuardInput): ActivationGuardResult {
  const { reference, surface, projection } = input;
  const identity = reference.tab_identity;

  if (!identity || typeof identity !== 'object') return reject('stale_selection');
  if (!isNonNegativeSafeInteger(reference.tab_id)
    || !isNonNegativeSafeInteger(reference.window_id)
    || !isNonNegativeSafeInteger(reference.query_sequence)
    || !isNonNegativeSafeInteger(Number(reference.projection_revision))
    || !isBoundedOpaque(reference.result_id, MAX_RESULT_ID_BYTES)
    || !isBoundedOpaque(reference.projection_epoch, MAX_EPOCH_BYTES)) {
    return reject('stale_selection');
  }
  if (identity.tab_id !== reference.tab_id) return reject('tab_reused');

  if (reference.profile_id !== surface.profile_id
    || reference.profile_id !== projection.profile_id
    || identity.profile_id !== projection.profile_id) {
    return reject('profile_mismatch');
  }

  if (reference.context_kind !== surface.context_kind
    || reference.context_kind !== projection.context_kind
    || identity.context_kind !== projection.context_kind) {
    return reject('context_mismatch');
  }

  if (projection.epoch === null || !isBoundedOpaque(projection.epoch, MAX_EPOCH_BYTES)) {
    return reject('projection_rebuilding');
  }

  if (reference.query_sequence !== surface.query_sequence
    || reference.result_id !== surface.selected_result_id) {
    return reject('stale_selection');
  }

  if (!isNonNegativeSafeInteger(Number(projection.revision))
    || reference.projection_epoch !== projection.epoch
    || Number(reference.projection_revision) !== Number(projection.revision)) {
    return reject('stale_selection');
  }

  const record = projection.records.get(tabIdentityKey(identity));
  if (!record) {
    const sameTabID = [...projection.records.values()]
      .some(candidate => candidate.tab_identity.tab_id === reference.tab_id);
    return reject(sameTabID ? 'tab_reused' : 'tab_missing');
  }

  if (!sameIdentity(record.tab_identity, identity)) return reject('tab_reused');
  if (record.result_id !== undefined && record.result_id !== reference.result_id) {
    return reject('tab_reused');
  }
  if (record.window_id !== reference.window_id) return reject('window_missing');
  if (record.eligible !== true) return reject('ineligible');
  if (record.projection_epoch !== reference.projection_epoch
    || Number(record.projection_revision) !== Number(reference.projection_revision)) {
    return reject('stale_selection');
  }

  return { outcome: 'valid' };
}

export function isActivationGuardSuccess(result: ActivationGuardResult): boolean {
  return result.outcome === 'valid';
}
