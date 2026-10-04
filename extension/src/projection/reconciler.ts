// Projection boundary for the extension producer (IP-05-T01).
// IP-02 owns ProfileID, ContextKind, TabIdentity, ProjectionEpoch,
// ProjectionRevision, EligibleTabRecord, and ProjectionState. IP-04 owns the
// allocation of projection_epoch, previous_revision, projection_revision, and
// event_sequence on every ObserverHandoff value. This module maps those two
// vocabularies onto the per-profile/per-context projection partitions and
// forwards the IP-04-allocated fence values unchanged. It never manufactures an
// identity, epoch, revision, or sequence and keeps no second counter: a handoff
// that does not carry all mandatory partition and fence fields is rejected with
// a bounded reason instead of being completed by a default. Snapshot authority,
// canonical ordering, delta reduction, atomic commit, and recovery are later
// IP-05 tasks and are intentionally not implemented here.

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
] as const;

export type ProjectionRejectReason = (typeof PROJECTION_REJECT_REASONS)[number];
export type ProjectionSubmissionKind = (typeof PROJECTION_ADMISSION_KINDS)[number];

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

export interface ProjectionBoundaryDiagnostics {
  handoffs_admitted: number;
  handoffs_rejected: number;
  rejections_by_reason: Record<ProjectionRejectReason, number>;
}

export interface ProjectionBoundaryView {
  readonly profile_id: ProfileID;
  readonly partitions: readonly ProjectionPartitionView[];
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

interface ForwardedFence {
  projection_epoch: ProjectionEpoch;
  projection_revision: ProjectionRevision;
  event_sequence: number;
}

function zeroDiagnostics(): ProjectionBoundaryDiagnostics {
  const rejections_by_reason = {} as Record<ProjectionRejectReason, number>;
  for (const reason of PROJECTION_REJECT_REASONS) rejections_by_reason[reason] = 0;
  return { handoffs_admitted: 0, handoffs_rejected: 0, rejections_by_reason };
}

export class ProjectionReconciler {
  private readonly states: Record<ContextKind, ProjectionState>;
  private readonly forwarded: Record<ContextKind, ForwardedFence | undefined> = { normal: undefined, private: undefined };
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
      diagnostics: { ...this.diagnostics, rejections_by_reason: { ...this.diagnostics.rejections_by_reason } },
    };
  }
}