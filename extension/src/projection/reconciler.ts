// Projection boundary for the extension producer (IP-05-T01, IP-05-T02).
// IP-02 owns ProfileID, ContextKind, TabIdentity, ProjectionEpoch,
// ProjectionRevision, EligibleTabRecord, and ProjectionState. IP-04 owns the
// allocation of projection_epoch, previous_revision, projection_revision, and
// event_sequence on every ObserverHandoff value. This module maps those two
// vocabularies onto the per-profile/per-context projection partitions and
// forwards the IP-04-allocated fence values unchanged. It never manufactures an
// identity, epoch, revision, or sequence and keeps no second counter: a handoff
// that does not carry all mandatory partition and fence fields is rejected with
// a bounded reason instead of being completed by a default.
//
// IP-05-T02 adds snapshot authority on top of that boundary: a handoff is the
// authoritative browser read for a partition only when it is a kind=snapshot
// handoff that cleared the T01 fence, carries the explicit records array IP-04
// read, states a present boolean resync_required of false, and stays inside the
// IP-02/IP-07 record bound. An empty records array from such a handoff is an
// intentional successful empty read and is authoritative; a status, unavailable,
// private-denied, partial, resync-required, or read-outcome-undeclared handoff is
// never turned into an empty authoritative snapshot and never clears the
// previously acquired snapshot of the partition. The producer hands the IP-07 seam
// a typed result instead of a map, so an uncertain read cannot be published as a
// converged projection. Those gates run on the raw handoff before any record is read
// or staged, so a status, partial, resync-required, over-bound, or undeclared
// handoff fails closed without materializing the payload it carries.
//
// Canonical ordering, lineage/ordering rules, delta reduction, atomic commit,
// and restart recovery are later IP-05 tasks and are intentionally not
// implemented here: the authoritative snapshot is retained as an acquisition
// result only and never committed into a live projection map.

import {
  emptyProjection,
  parseProfileID,
  parseProjectionEpoch,
  parseRevision,
  tabIdentityKey,
  validateTab,
  type ContextKind,
  type EligibleTabRecord,
  type ProfileID,
  type ProjectionEpoch,
  type ProjectionRevision,
  type ProjectionState,
  type TabIdentity,
} from '../../domain/index.js';
import type { ObserverHandoff, ResyncReason } from '../browser/tab-observer.js';

export const PROJECTION_PARTITION_FIELDS = ['profile_id', 'context_kind'] as const;

export const PROJECTION_FENCE_FIELDS = [
  'projection_epoch',
  'previous_revision',
  'projection_revision',
  'event_sequence',
] as const;

export const PROJECTION_ADMISSION_KINDS = [
  'snapshot',
  'upsert',
  'remove',
  'window',
  'status',
  'private_context_ended',
] as const;

// The one bound every snapshot authority decision shares: IP-02 acceptSnapshot and
// the IP-07 snapshot_records limit both cap a full browser read at 10000 eligible
// records, so a larger payload is refused instead of being staged.
export const PROJECTION_MAX_SNAPSHOT_RECORDS = 10000;

// The typed outcomes an acquisition can report to the IP-07 seam. AUTHORITATIVE is
// the only value that may carry records; the two failure values are the IP-07 wire
// error codes an incomplete or contradicting handoff maps to, so this module never
// invents a private error vocabulary beside the protocol's.
export const PROJECTION_SNAPSHOT_OUTCOMES = [
  'AUTHORITATIVE',
  'SNAPSHOT_REQUIRED',
  'PROFILE_MISMATCH',
] as const;

export const PROJECTION_REJECT_REASONS = [
  'unpartitioned_handoff',
  'unknown_handoff_kind',
  'missing_profile_id',
  'invalid_profile_id',
  'partition_profile_mismatch',
  'missing_context_kind',
  'invalid_context_kind',
  'missing_projection_epoch',
  'invalid_projection_epoch',
  'missing_previous_revision',
  'invalid_previous_revision',
  'missing_projection_revision',
  'invalid_projection_revision',
  'missing_event_sequence',
  'invalid_event_sequence',
  'missing_records',
  'missing_record',
  'missing_tab_identity',
  'invalid_tab_identity',
  'identity_partition_mismatch',
  'invalid_record',
  'tab_identity_mismatch',
  'non_snapshot_handoff',
  'missing_resync_marker',
  'invalid_resync_marker',
  'snapshot_resync_required',
  'snapshot_bounds_exceeded',
] as const;

export type ProjectionRejectReason = (typeof PROJECTION_REJECT_REASONS)[number];
export type ProjectionSubmissionKind = (typeof PROJECTION_ADMISSION_KINDS)[number];
export type ProjectionSnapshotOutcome = (typeof PROJECTION_SNAPSHOT_OUTCOMES)[number];
export type ProjectionSnapshotFailure = Exclude<ProjectionSnapshotOutcome, 'AUTHORITATIVE'>;

export interface ProjectionPartition {
  readonly profile_id: ProfileID;
  readonly context_kind: ContextKind;
}

export interface ProjectionFence extends ProjectionPartition {
  readonly projection_epoch: ProjectionEpoch;
  readonly previous_revision: ProjectionRevision;
  readonly projection_revision: ProjectionRevision;
  readonly event_sequence: number;
}

export interface ProjectionSubmission {
  readonly kind: ProjectionSubmissionKind;
  readonly fence: ProjectionFence;
  readonly effective_change: boolean;
  readonly resync_required: boolean;
  readonly resync_reason?: ResyncReason;
  readonly records: readonly EligibleTabRecord[];
  readonly record?: EligibleTabRecord;
  readonly tab_identity?: TabIdentity;
  readonly identity_keys: readonly string[];
}

export interface ProjectionPartitionView {
  readonly context_kind: ContextKind;
  readonly last_observed_epoch: ProjectionEpoch | null;
  readonly last_observed_revision: ProjectionRevision | null;
  readonly last_observed_sequence: number | null;
  readonly record_count: number;
}

// One authoritative browser read of a partition, exactly as IP-04 declared it. The
// records are the validated IP-02 eligible records of the declared fence: the
// authoritative list is the snapshot IP-04 read, not a projection this module
// assembled. explicitly_empty states that the browser read completed and reported
// no eligible tab, which is a converged projection and not a failed read.
export interface ProjectionSnapshotAuthority {
  readonly fence: ProjectionFence;
  readonly records: readonly EligibleTabRecord[];
  readonly record_count: number;
  readonly explicitly_empty: boolean;
}

export type SnapshotAcquisition =
  | {
      readonly ok: true;
      readonly authoritative: true;
      readonly outcome: 'AUTHORITATIVE';
      readonly snapshot: ProjectionSnapshotAuthority;
    }
  | {
      readonly ok: false;
      readonly authoritative: false;
      readonly outcome: ProjectionSnapshotFailure;
      readonly reason: ProjectionRejectReason;
      readonly resync_required: true;
      readonly fence?: ProjectionFence;
      readonly resync_reason?: ResyncReason;
    };

// The observable acquisition state of one partition. epoch, projection_revision,
// event_sequence, record_count, and explicitly_empty always describe the retained
// authoritative snapshot, so a refused or partial read is observable as an outcome
// while the previously acquired data stays intact.
export interface ProjectionSnapshotPartitionView {
  readonly context_kind: ContextKind;
  readonly last_outcome: ProjectionSnapshotOutcome | 'UNINITIALIZED';
  readonly retained_authoritative: boolean;
  readonly epoch: ProjectionEpoch | null;
  readonly projection_revision: ProjectionRevision | null;
  readonly event_sequence: number | null;
  readonly record_count: number;
  readonly explicitly_empty: boolean;
}

export interface ProjectionBoundaryDiagnostics {
  handoffs_admitted: number;
  handoffs_rejected: number;
  rejections_by_reason: Record<ProjectionRejectReason, number>;
  snapshots_authoritative: number;
  snapshots_unauthoritative: number;
  snapshot_refusals_by_reason: Record<ProjectionRejectReason, number>;
}

export interface ProjectionBoundaryView {
  readonly profile_id: ProfileID;
  readonly partitions: readonly ProjectionPartitionView[];
  readonly snapshots: readonly ProjectionSnapshotPartitionView[];
  readonly diagnostics: ProjectionBoundaryDiagnostics;
}

export type PartitionAdmission =
  | { ok: true; partition: ProjectionPartition }
  | { ok: false; reason: ProjectionRejectReason };

export type FenceAdmission =
  | { ok: true; fence: ProjectionFence }
  | { ok: false; reason: ProjectionRejectReason };

export type IdentityAdmission =
  | { ok: true; identity: TabIdentity; key: string }
  | { ok: false; reason: ProjectionRejectReason };

export type RecordAdmission =
  | { ok: true; record: EligibleTabRecord }
  | { ok: false; reason: ProjectionRejectReason };

export type ProjectionAdmission =
  | { ok: true; submission: ProjectionSubmission }
  | { ok: false; reason: ProjectionRejectReason };

const CONTEXT_KINDS: readonly string[] = ['normal', 'private'];
const EMPTY_RECORDS: readonly EligibleTabRecord[] = [];

function rejected(reason: ProjectionRejectReason): { ok: false; reason: ProjectionRejectReason } {
  return { ok: false, reason };
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function isAbsent(value: unknown): boolean {
  return value === undefined || value === null;
}

function compact<T extends Record<string, unknown>>(value: T): T {
  const result: Record<string, unknown> = {};
  for (const [key, entry] of Object.entries(value)) {
    if (entry !== undefined) result[key] = entry;
  }
  return result as T;
}

function tryParse<T>(parse: (value: unknown) => T, value: unknown): T | undefined {
  try {
    return parse(value);
  } catch {
    return undefined;
  }
}

function readContextKind(value: unknown): ContextKind | undefined {
  return typeof value === 'string' && CONTEXT_KINDS.includes(value) ? (value as ContextKind) : undefined;
}

function isAdmissionKind(kind: unknown): kind is ProjectionSubmissionKind {
  return typeof kind === 'string' && (PROJECTION_ADMISSION_KINDS as readonly string[]).includes(kind);
}

export function readProjectionPartition(value: unknown): PartitionAdmission {
  if (!isRecord(value)) return rejected('unpartitioned_handoff');
  if (isAbsent(value.profile_id)) return rejected('missing_profile_id');
  const profile_id = tryParse(parseProfileID, value.profile_id);
  if (profile_id === undefined) return rejected('invalid_profile_id');
  if (isAbsent(value.context_kind)) return rejected('missing_context_kind');
  const context_kind = readContextKind(value.context_kind);
  if (context_kind === undefined) return rejected('invalid_context_kind');
  return { ok: true, partition: { profile_id, context_kind } };
}

export function readProjectionFence(value: unknown, boundProfile: ProfileID): FenceAdmission {
  const partition = readProjectionPartition(value);
  if (!partition.ok) return partition;
  if (partition.partition.profile_id !== boundProfile) return rejected('partition_profile_mismatch');
  if (!isRecord(value)) return rejected('unpartitioned_handoff');
  if (isAbsent(value.projection_epoch)) return rejected('missing_projection_epoch');
  const projection_epoch = tryParse(parseProjectionEpoch, value.projection_epoch);
  if (projection_epoch === undefined) return rejected('invalid_projection_epoch');
  const previous_revision = readRevisionField(value.previous_revision, 'previous_revision');
  if (!previous_revision.ok) return previous_revision;
  const projection_revision = readRevisionField(value.projection_revision, 'projection_revision');
  if (!projection_revision.ok) return projection_revision;
  const event_sequence = readSequenceField(value.event_sequence);
  if (!event_sequence.ok) return event_sequence;
  return {
    ok: true,
    fence: {
      profile_id: partition.partition.profile_id,
      context_kind: partition.partition.context_kind,
      projection_epoch,
      previous_revision: previous_revision.value,
      projection_revision: projection_revision.value,
      event_sequence: event_sequence.value,
    },
  };
}

export function readTabIdentity(value: unknown): IdentityAdmission {
  if (isAbsent(value)) return rejected('missing_tab_identity');
  if (!isRecord(value)) return rejected('invalid_tab_identity');
  const profile_id = tryParse(parseProfileID, value.profile_id);
  const context_kind = readContextKind(value.context_kind);
  const tab_id = value.tab_id;
  if (profile_id === undefined || context_kind === undefined) return rejected('invalid_tab_identity');
  if (typeof tab_id !== 'number' || !Number.isSafeInteger(tab_id) || tab_id < 0) return rejected('invalid_tab_identity');
  const identity: TabIdentity = { profile_id, context_kind, tab_id };
  return { ok: true, identity, key: tabIdentityKey(identity) };
}

export function readProjectionRecord(value: unknown, fence: ProjectionFence, expectedKey?: string): RecordAdmission {
  if (isAbsent(value)) return rejected('missing_record');
  if (!isRecord(value)) return rejected('invalid_record');
  const record = value as unknown as EligibleTabRecord;
  if (!validateTab(record, fence.profile_id, fence.context_kind, fence.projection_epoch)) return rejected('invalid_record');
  if (expectedKey !== undefined && tabIdentityKey(record.tab_identity) !== expectedKey) return rejected('tab_identity_mismatch');
  return { ok: true, record };
}

export function admitHandoff(boundProfile: ProfileID, handoff: ObserverHandoff): ProjectionAdmission {
  if (handoff.kind === 'disposed') return rejected('unpartitioned_handoff');
  if (!isAdmissionKind(handoff.kind)) return rejected('unknown_handoff_kind');
  const fence = readProjectionFence(handoff, boundProfile);
  if (!fence.ok) return fence;
  let identity: TabIdentity | undefined;
  let identityKey: string | undefined;
  if (handoff.kind === 'upsert' || handoff.kind === 'remove') {
    const read = readTabIdentity(handoff.tab_identity);
    if (!read.ok) return read;
    if (read.identity.profile_id !== fence.fence.profile_id || read.identity.context_kind !== fence.fence.context_kind) {
      return rejected('identity_partition_mismatch');
    }
    identity = read.identity;
    identityKey = read.key;
  }
  let records: readonly EligibleTabRecord[] = EMPTY_RECORDS;
  if (handoff.kind === 'snapshot') {
    if (!Array.isArray(handoff.records)) return rejected('missing_records');
    const staged: EligibleTabRecord[] = [];
    for (const row of handoff.records) {
      const read = readProjectionRecord(row, fence.fence, identityKey);
      if (!read.ok) return read;
      staged.push(read.record);
    }
    records = staged;
  }
  let record: EligibleTabRecord | undefined;
  if (handoff.kind === 'upsert') {
    const read = readProjectionRecord(handoff.record, fence.fence, identityKey);
    if (!read.ok) return read;
    record = read.record;
  }
  return {
    ok: true,
    submission: compact({
      kind: handoff.kind,
      fence: fence.fence,
      effective_change: handoff.effective_change === true,
      resync_required: handoff.resync_required === true,
      resync_reason: handoff.resync_reason,
      records,
      record,
      tab_identity: identity,
      identity_keys: identityKey === undefined ? [] : [identityKey],
    }) as ProjectionSubmission,
  };
}

function readRevisionField(
  value: unknown,
  field: 'previous_revision' | 'projection_revision',
): { ok: true; value: ProjectionRevision } | { ok: false; reason: ProjectionRejectReason } {
  if (isAbsent(value)) return rejected(field === 'previous_revision' ? 'missing_previous_revision' : 'missing_projection_revision');
  const revision = tryParse(parseRevision, value);
  if (revision === undefined) return rejected(field === 'previous_revision' ? 'invalid_previous_revision' : 'invalid_projection_revision');
  return { ok: true, value: revision };
}

function readSequenceField(value: unknown): { ok: true; value: number } | { ok: false; reason: ProjectionRejectReason } {
  if (isAbsent(value)) return rejected('missing_event_sequence');
  if (typeof value !== 'number' || !Number.isSafeInteger(value) || value < 0) return rejected('invalid_event_sequence');
  return { ok: true, value };
}

// IP-05-T02: snapshot authority.

// snapshotOutcome maps one bounded boundary refusal onto the IP-07 code the
// extension reports when it cannot present an authoritative snapshot. A handoff
// that names another partition contradicts the bound profile rather than merely
// lacking fields, so it maps to the non-retryable PROFILE_MISMATCH; every other
// refusal is repaired by a complete authoritative read and stays the retryable
// SNAPSHOT_REQUIRED. This is a mapping onto the protocol's own literals, not a
// second error vocabulary.
function snapshotOutcome(reason: ProjectionRejectReason): ProjectionSnapshotFailure {
  return reason === 'partition_profile_mismatch' || reason === 'identity_partition_mismatch'
    ? 'PROFILE_MISMATCH'
    : 'SNAPSHOT_REQUIRED';
}

// unauthoritative builds the typed failure of one handoff that cannot stand as the
// authoritative browser read of its partition. It carries no records at all, so a
// downstream consumer can never read it as a converged empty projection, and it
// always states that a fresh authoritative snapshot is required. fence is present
// only when the handoff did state a valid one, and resync_reason is the IP-04
// marker explaining the failed read; neither carries a title, URL, query, token, or
// page value.
function unauthoritative(
  reason: ProjectionRejectReason,
  handoff: ObserverHandoff,
  fence?: ProjectionFence,
): SnapshotAcquisition {
  return compact({
    ok: false,
    authoritative: false,
    outcome: snapshotOutcome(reason),
    reason,
    resync_required: true,
    fence,
    resync_reason: handoff.resync_reason,
  }) as SnapshotAcquisition;
}

// acquireSnapshot decides whether one IP-04 handoff is the authoritative browser
// read of the bound profile's partition. The checks are ordered so that no record is
// read, copied, or staged before the handoff has proven it is a complete read worth
// materializing: the raw kind, fence, marker, record-list, and record-count gates run
// first, and only a handoff that clears all of them reaches the T01 record walk.
//
// 1. The kind is an admitted handoff kind at all: a disposed session is unpartitioned
//    and an unknown kind is unknown, neither of which can name a read.
// 2. The handoff states a usable fence: the bound profile, a known context, and the
//    IP-04 allocated epoch, previous revision, revision, and sequence. Reading it
//    costs a few scalar fields, refuses another profile or an unstated fence field,
//    and gives every later refusal the partition it belongs to.
// 3. Its kind is snapshot. A status, window, private_context_ended, upsert, or
//    remove handoff reports an observation or an event, never a complete read, so
//    it is refused instead of being completed into an empty snapshot.
// 4. It states a usable read outcome. IP-04 declares resync_required as a
//    mandatory boolean and sets it for an unavailable or permission-denied read,
//    an incomplete or partial read, and an unreadable private context, so such a
//    handoff is an uncertain read even when it arrives with an empty records
//    array. The raw handoff field is read here instead of the admitted submission:
//    the T01 boundary normalizes that marker with the strict comparison
//    handoff.resync_required === true, which coerces an absent or non-boolean
//    marker to false and would report an undeclared read as a completed one,
//    letting it stand as an authoritative empty snapshot. Authority requires
//    resync_required to be present and boolean, and false.
// 5. It declares its record list explicitly and keeps it inside the IP-02/IP-07
//    snapshot bound, so an oversized payload is refused by its length alone rather
//    than after every record in it was copied.
// 6. Every record passes the IP-02 bounds and identity checks against that fence,
//    which is the only step that materializes records and the only one T01 owns.
//
// A snapshot that passes all six is authoritative, including one whose explicit
// records array is empty: an empty array from a completed read means the browser
// reports no eligible tab, and publishing it as authoritative is what lets the host
// converge on an empty projection instead of waiting forever for data that will
// never arrive. The fence and records are forwarded exactly as IP-04 declared them:
// this function classifies a handoff and never repairs, renumbers, or allocates one.
export function acquireSnapshot(boundProfile: ProfileID, handoff: ObserverHandoff): SnapshotAcquisition {
  if (handoff.kind === 'disposed') return unauthoritative('unpartitioned_handoff', handoff);
  if (!isAdmissionKind(handoff.kind)) return unauthoritative('unknown_handoff_kind', handoff);
  // The fence is resolved before anything can allocate, so an unpartitioned, foreign,
  // or unstated-fence handoff is refused on scalar fields alone.
  const fence = readProjectionFence(handoff, boundProfile);
  if (!fence.ok) return unauthoritative(fence.reason, handoff);
  if (handoff.kind !== 'snapshot') return unauthoritative('non_snapshot_handoff', handoff, fence.fence);
  if (isAbsent(handoff.resync_required)) return unauthoritative('missing_resync_marker', handoff, fence.fence);
  if (typeof handoff.resync_required !== 'boolean') return unauthoritative('invalid_resync_marker', handoff, fence.fence);
  if (handoff.resync_required) return unauthoritative('snapshot_resync_required', handoff, fence.fence);
  if (!Array.isArray(handoff.records)) return unauthoritative('missing_records', handoff, fence.fence);
  if (handoff.records.length > PROJECTION_MAX_SNAPSHOT_RECORDS) return unauthoritative('snapshot_bounds_exceeded', handoff, fence.fence);
  // Only a handoff that may be authority now reaches the T01 boundary, which owns
  // the per-record identity and bounds validation.
  const admission = admitHandoff(boundProfile, handoff);
  if (!admission.ok) return unauthoritative(admission.reason, handoff, fence.fence);
  const submission = admission.submission;
  return {
    ok: true,
    authoritative: true,
    outcome: 'AUTHORITATIVE',
    snapshot: {
      fence: submission.fence,
      records: submission.records,
      record_count: submission.records.length,
      explicitly_empty: submission.records.length === 0,
    },
  };
}

interface ForwardedFence {
  projection_epoch: ProjectionEpoch;
  projection_revision: ProjectionRevision;
  event_sequence: number;
}

interface RetainedSnapshot {
  readonly fence: ProjectionFence;
  readonly records: readonly EligibleTabRecord[];
}

// The acquisition state of one partition. The retained snapshot is only ever
// replaced by another authoritative read, while last_outcome records the newest
// attempt separately so a refused read is observable without disturbing the data.
interface PartitionSnapshotState {
  retained: RetainedSnapshot | undefined;
  last_outcome: ProjectionSnapshotOutcome | 'UNINITIALIZED';
}

function zeroCounters(): Record<ProjectionRejectReason, number> {
  const counters = {} as Record<ProjectionRejectReason, number>;
  for (const reason of PROJECTION_REJECT_REASONS) counters[reason] = 0;
  return counters;
}

function zeroDiagnostics(): ProjectionBoundaryDiagnostics {
  return {
    handoffs_admitted: 0,
    handoffs_rejected: 0,
    rejections_by_reason: zeroCounters(),
    snapshots_authoritative: 0,
    snapshots_unauthoritative: 0,
    snapshot_refusals_by_reason: zeroCounters(),
  };
}

export class ProjectionReconciler {
  private readonly states: Record<ContextKind, ProjectionState>;
  private readonly forwarded: Record<ContextKind, ForwardedFence | undefined> = { normal: undefined, private: undefined };
  private readonly snapshots: Record<ContextKind, PartitionSnapshotState> = {
    normal: { retained: undefined, last_outcome: 'UNINITIALIZED' },
    private: { retained: undefined, last_outcome: 'UNINITIALIZED' },
  };
  private readonly diagnostics: ProjectionBoundaryDiagnostics = zeroDiagnostics();

  constructor(private readonly boundProfile: ProfileID) {
    this.states = {
      normal: emptyProjection(boundProfile, 'normal'),
      private: emptyProjection(boundProfile, 'private'),
    };
  }

  admit(handoff: ObserverHandoff): ProjectionAdmission {
    const admission = admitHandoff(this.boundProfile, handoff);
    if (!admission.ok) {
      this.diagnostics.handoffs_rejected += 1;
      this.diagnostics.rejections_by_reason[admission.reason] += 1;
      return admission;
    }
    this.diagnostics.handoffs_admitted += 1;
    const { fence } = admission.submission;
    this.forwarded[fence.context_kind] = {
      projection_epoch: fence.projection_epoch,
      projection_revision: fence.projection_revision,
      event_sequence: fence.event_sequence,
    };
    return admission;
  }

  // acquire classifies one IP-04 handoff as the authoritative browser read of its
  // partition and retains it as this partition's acquired snapshot. A refused,
  // partial, unavailable, or resync-required handoff changes no retained snapshot:
  // the previously acquired read of that partition stays exactly as it was, so a
  // failed read can never be observed as an empty projection and never erases valid
  // state. Acquisition is not commit: the retained snapshot is an input for the
  // later atomic replacement task and is never written into the live projection map.
  acquire(handoff: ObserverHandoff): SnapshotAcquisition {
    const acquisition = acquireSnapshot(this.boundProfile, handoff);
    if (!acquisition.ok) {
      this.diagnostics.snapshots_unauthoritative += 1;
      this.diagnostics.snapshot_refusals_by_reason[acquisition.reason] += 1;
      // A handoff that stated no usable partition cannot be attributed to one, so
      // only its bounded counter records it; its retained snapshot is untouched
      // either way.
      if (acquisition.fence !== undefined) this.snapshots[acquisition.fence.context_kind].last_outcome = acquisition.outcome;
      return acquisition;
    }
    this.diagnostics.snapshots_authoritative += 1;
    const { fence, records } = acquisition.snapshot;
    const partition = this.snapshots[fence.context_kind];
    partition.retained = { fence, records };
    partition.last_outcome = acquisition.outcome;
    return acquisition;
  }

  // acquiredSnapshot returns the retained authoritative snapshot of a partition, or
  // undefined when no authoritative read has been acquired for it yet. The record
  // list is copied out so a caller cannot mutate the retained state through it, and
  // no other path changes this value: acquiring a newer snapshot is the only way.
  acquiredSnapshot(context_kind: ContextKind): ProjectionSnapshotAuthority | undefined {
    const retained = this.snapshots[context_kind].retained;
    if (retained === undefined) return undefined;
    return {
      fence: retained.fence,
      records: [...retained.records],
      record_count: retained.records.length,
      explicitly_empty: retained.records.length === 0,
    };
  }

  livePartition(context_kind: ContextKind): ProjectionState {
    return this.states[context_kind];
  }

  view(): ProjectionBoundaryView {
    return {
      profile_id: this.boundProfile,
      partitions: (['normal', 'private'] as const).map(context_kind => {
        const observed = this.forwarded[context_kind];
        return {
          context_kind,
          last_observed_epoch: observed?.projection_epoch ?? null,
          last_observed_revision: observed?.projection_revision ?? null,
          last_observed_sequence: observed?.event_sequence ?? null,
          record_count: this.states[context_kind].records.size,
        };
      }),
      snapshots: (['normal', 'private'] as const).map(context_kind => {
        const partition = this.snapshots[context_kind];
        const retained = partition.retained;
        return {
          context_kind,
          last_outcome: partition.last_outcome,
          retained_authoritative: retained !== undefined,
          epoch: retained?.fence.projection_epoch ?? null,
          projection_revision: retained?.fence.projection_revision ?? null,
          event_sequence: retained?.fence.event_sequence ?? null,
          record_count: retained?.records.length ?? 0,
          explicitly_empty: retained !== undefined && retained.records.length === 0,
        };
      }),
      diagnostics: {
        ...this.diagnostics,
        rejections_by_reason: { ...this.diagnostics.rejections_by_reason },
        snapshot_refusals_by_reason: { ...this.diagnostics.snapshot_refusals_by_reason },
      },
    };
  }
}