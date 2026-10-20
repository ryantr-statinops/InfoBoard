import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';

import { acceptSnapshot, emptyProjection, parseProfileID, parseProjectionEpoch } from '../../../extension/dist/domain/index.js';
import { ProjectionReconciler, acquireSnapshot, admitHandoff, reduceProjectionBatch } from '../../../extension/dist/src/projection/reconciler.js';

const corpus = JSON.parse(await readFile('fixtures/projection/phase-05/snapshot-authority.json', 'utf8'));
const populated = corpus.cases.find((item) => item.case_id === 'complete_populated_snapshot');
const profile = parseProfileID(corpus.input.profile_id);

test('IP-05-T03 canonicalizes identity order and preserves optional-field absence', () => {
  const records = structuredClone(populated.handoff.records.slice(0, 2));
  records[1].tab_identity.tab_id = 2;
  const handoff = { ...populated.handoff, records: [records[0], records[1]] };
  const result = acquireSnapshot(profile, handoff);
  assert.equal(result.ok, true, 'the complete browser snapshot remains authoritative');
  assert.deepEqual(result.snapshot.records.map((record) => record.tab_identity.tab_id), [2, 10]);
  assert.equal(
    Object.hasOwn(result.snapshot.records[0], 'window_label_display'),
    false,
    'canonicalization must not synthesize absent optional labels',
  );

  const permuted = acquireSnapshot(profile, { ...handoff, records: [...records].reverse() });
  assert.equal(permuted.ok, true);
  assert.deepEqual(permuted.snapshot.records, result.snapshot.records, 'input order must not affect canonical output');
});

test('IP-05-T03 rejects duplicate identities before producing canonical output', () => {
  const record = structuredClone(populated.handoff.records[0]);
  const duplicate = acquireSnapshot(profile, {
    ...populated.handoff,
    records: [record, structuredClone(record)],
  });
  assert.equal(duplicate.ok, false);
  assert.equal(duplicate.reason, 'duplicate_tab_identity');
  assert.equal(duplicate.snapshot, undefined);
});
test('IP-05-T04 accepts only a revision-1 snapshot with predecessor revision zero', () => {
  const handoff = populated.handoff;
  for (const invalid of [
    { ...handoff, projection_revision: 2, previous_revision: 1 },
    { ...handoff, previous_revision: 1 },
  ]) {
    const result = acquireSnapshot(profile, invalid);
    assert.equal(result.ok, false, 'a snapshot from a non-initial lineage was authoritative');
    assert.equal(result.reason, 'invalid_snapshot_lineage');
  }
  assert.equal(acquireSnapshot(profile, handoff).ok, true, 'the initial revision-1 snapshot must remain authoritative');
});
function submissionOf(handoff) {
  const admission = admitHandoff(profile, handoff);
  assert.ok(admission.ok, "fixture handoff must be admitted");
  return admission.submission;
}

test('IP-05-T05 reduces no-op and effective events atomically with independent sequence and revision', () => {
  const epoch = parseProjectionEpoch(populated.handoff.projection_epoch);
  const rows = structuredClone(populated.handoff.records);
  const state = emptyProjection(profile, 'normal');
  assert.equal(acceptSnapshot(state, epoch, rows), 'APPLIED');
  const first = rows.find((row) => row.tab_identity.tab_id === 10);
  const second = rows.find((row) => row.tab_identity.tab_id === 11);
  assert.ok(first && second);

  const noOp = { ...populated.handoff, kind: 'upsert', previous_revision: 1, projection_revision: 1, event_sequence: 2, effective_change: false, tab_identity: first.tab_identity, record: first };
  const updated = { ...second, title_display: 'Beta updated', title_search: 'beta updated', projection_revision: 2 };
  const update = { ...populated.handoff, kind: 'upsert', previous_revision: 1, projection_revision: 2, event_sequence: 3, effective_change: true, tab_identity: second.tab_identity, record: updated };
  const remove = { ...populated.handoff, kind: 'remove', previous_revision: 2, projection_revision: 3, event_sequence: 4, effective_change: true, tab_identity: first.tab_identity };
  const batch = [noOp, update, remove].map(submissionOf);
  const result = reduceProjectionBatch(state, 1, batch);
  assert.equal(result.outcome, 'APPLIED');
  assert.equal(Number(state.revision), 3, 'the no-op sequence must not advance projection revision');
  assert.equal(result.event_sequence, 4, 'every accepted handoff must advance event sequence');
  assert.equal(state.records.size, 2, 'the reducer should not drop unrelated records');
  assert.equal(state.records.has(JSON.stringify([profile, 'normal', 10])), false);
  assert.equal(state.records.get(JSON.stringify([profile, 'normal', 11])).title_display, 'Beta updated');
  const beforeReplay = [...state.records];
  const duplicate = reduceProjectionBatch(state, 4, batch);
  assert.equal(duplicate.outcome, 'DUPLICATE');
  assert.equal(duplicate.resync_required, false);
  assert.equal(Number(state.revision), 3);
  assert.deepEqual([...state.records], beforeReplay);

  const changedRecord = { ...batch[1].record, title_display: 'Conflicting beta', title_search: 'conflicting beta' };
  const conflict = reduceProjectionBatch(state, 4, [{ ...batch[1], record: changedRecord }]);
  assert.equal(conflict.outcome, 'REVISION_MISMATCH');
  assert.equal(conflict.resync_required, true);
  assert.deepEqual([...state.records], beforeReplay);

  const stale = reduceProjectionBatch(state, 4, [{ ...batch[0], fence: { ...batch[0].fence, event_sequence: 1 } }]);
  assert.equal(stale.outcome, 'SNAPSHOT_REQUIRED');
  assert.equal(stale.resync_required, true);
  assert.deepEqual([...state.records], beforeReplay);
});

test('IP-05-T05 rejects a failed batch without publishing earlier operations', () => {
  const epoch = parseProjectionEpoch(populated.handoff.projection_epoch);
  const rows = structuredClone(populated.handoff.records);
  const state = emptyProjection(profile, 'normal');
  assert.equal(acceptSnapshot(state, epoch, rows), 'APPLIED');
  const beforeRecords = [...state.records];
  const first = rows.find((row) => row.tab_identity.tab_id === 10);
  const second = rows.find((row) => row.tab_identity.tab_id === 11);
  assert.ok(first && second);
  const updated = { ...second, title_display: 'Beta updated', title_search: 'beta updated', projection_revision: 2 };
  const valid = { ...populated.handoff, kind: 'upsert', previous_revision: 1, projection_revision: 2, event_sequence: 2, effective_change: true, tab_identity: second.tab_identity, record: updated };
  const invalid = { ...populated.handoff, kind: 'remove', previous_revision: 2, projection_revision: 3, event_sequence: 3, effective_change: true, tab_identity: { ...first.tab_identity, tab_id: 99 } };
  const result = reduceProjectionBatch(state, 1, [valid, invalid].map(submissionOf));
  assert.equal(result.outcome, 'SNAPSHOT_REQUIRED');
  assert.equal(Number(state.revision), 1);
  assert.deepEqual([...state.records], beforeRecords);
});
test('IP-05-T06 acknowledges exact replay and rejects conflicting or stale sequence keys', () => {
  const epoch = parseProjectionEpoch(populated.handoff.projection_epoch);
  const rows = structuredClone(populated.handoff.records);
  const state = emptyProjection(profile, 'normal');
  assert.equal(acceptSnapshot(state, epoch, rows), 'APPLIED');
  const second = rows.find((row) => row.tab_identity.tab_id === 11);
  assert.ok(second);
  const updated = { ...second, title_display: 'Beta changed', title_search: 'beta changed', projection_revision: 2 };
  const handoff = { ...populated.handoff, kind: 'upsert', previous_revision: 1, projection_revision: 2, event_sequence: 2, effective_change: true, tab_identity: second.tab_identity, record: updated };
  const submission = submissionOf(handoff);
  assert.equal(reduceProjectionBatch(state, 1, [submission]).outcome, 'APPLIED');

  const beforeReplay = [...state.records];
  assert.equal(reduceProjectionBatch(state, 2, [submission]).outcome, 'DUPLICATE');
  assert.deepEqual([...state.records], beforeReplay);
  assert.equal(Number(state.revision), 2);

  const changed = { ...submission, record: { ...updated, title_display: 'Conflicting title' } };
  assert.equal(reduceProjectionBatch(state, 2, [changed]).outcome, 'REVISION_MISMATCH');
  const stale = { ...submission, fence: { ...submission.fence, event_sequence: 1 } };
  assert.equal(reduceProjectionBatch(state, 2, [stale]).outcome, 'SNAPSHOT_REQUIRED');
  assert.deepEqual([...state.records], beforeReplay);
  assert.equal(Number(state.revision), 2);
});
test('IP-05-T07 commits snapshots atomically, detaches records, and replaces old identities', () => {
  const reconciler = new ProjectionReconciler(profile);
  const initialHandoff = structuredClone(populated.handoff);
  assert.equal(reconciler.acquire(initialHandoff).ok, true);
  const acquiredCopy = reconciler.acquiredSnapshot('normal');
  assert.ok(acquiredCopy);
  initialHandoff.records[0].title_display = 'mutated source';
  acquiredCopy.records[0].tab_identity.tab_id = 99;

  const committed = reconciler.commitSnapshot('normal');
  assert.equal(committed.outcome, 'COMMITTED');
  assert.equal(committed.resync_required, false);
  const initialState = reconciler.livePartition('normal');
  const first = initialState.records.get(JSON.stringify([profile, 'normal', 10]));
  assert.equal(first?.title_display, 'Alpha', 'the committed map must not alias acquisition input');
  assert.equal(first?.tab_identity.tab_id, 10, 'mutating an acquired view must not alter the committed identity');
  assert.equal(initialState.records.size, populated.handoff.records.length);

  assert.equal(reconciler.commitSnapshot('normal').outcome, 'DUPLICATE');
  const nextEpoch = '123e4567-e89b-42d3-a456-426614174003';
  const replacementRecord = structuredClone(populated.handoff.records.find((record) => record.tab_identity.tab_id === 11));
  replacementRecord.projection_epoch = nextEpoch;
  const replacement = {
    ...populated.handoff,
    projection_epoch: nextEpoch,
    previous_revision: 0,
    projection_revision: 1,
    event_sequence: 3,
    records: [replacementRecord],
  };
  assert.equal(reconciler.acquire(replacement).ok, true);
  const replaced = reconciler.commitSnapshot('normal');
  assert.equal(replaced.outcome, 'COMMITTED');
  const next = reconciler.livePartition('normal');
  assert.equal(next.epoch, nextEpoch);
  assert.equal(Number(next.revision), 1);
  assert.equal(next.records.size, 1);
  assert.equal(next.records.has(JSON.stringify([profile, 'normal', 10])), false);
  assert.equal(next.records.has(JSON.stringify([profile, 'normal', 11])), true);
});
test('IP-05-T08 requires a fresh snapshot after reconnect before deltas resume', () => {
  const reconciler = new ProjectionReconciler(profile);
  assert.equal(reconciler.acquire(structuredClone(populated.handoff)).ok, true);
  assert.equal(reconciler.commitSnapshot('normal').outcome, 'COMMITTED');

  reconciler.requireSnapshot('normal');
  const quarantined = reconciler.livePartition('normal');
  assert.equal(quarantined.records.size, 0, 'old session references must be cleared');
  assert.equal(Number(quarantined.revision), 0, 'old revision must not survive reconnect');
  assert.equal(reconciler.acquiredSnapshot('normal'), undefined, 'old acquired snapshot cannot be replayed');

  const previous = populated.handoff.records.find((record) => record.tab_identity.tab_id === 11);
  assert.ok(previous);
  const stale = {
    ...populated.handoff,
    kind: 'upsert',
    previous_revision: 1,
    projection_revision: 2,
    event_sequence: 2,
    effective_change: true,
    tab_identity: previous.tab_identity,
    record: { ...previous, title_display: 'stale session', title_search: 'stale session', projection_revision: 2 },
  };
  const rejected = reconciler.reduceDeltas('normal', [submissionOf(stale)]);
  assert.equal(rejected.outcome, 'SNAPSHOT_REQUIRED');
  assert.equal(rejected.resync_required, true);
  assert.equal(reconciler.livePartition('normal').records.size, 0);

  const nextEpoch = '123e4567-e89b-42d3-a456-426614174004';
  const freshRecord = structuredClone(previous);
  freshRecord.projection_epoch = nextEpoch;
  const fresh = {
    ...populated.handoff,
    projection_epoch: nextEpoch,
    previous_revision: 0,
    projection_revision: 1,
    event_sequence: 1,
    records: [freshRecord],
  };
  assert.equal(reconciler.acquire(fresh).ok, true);
  assert.equal(reconciler.commitSnapshot('normal').outcome, 'COMMITTED');
  const recovered = reconciler.livePartition('normal');
  assert.equal(recovered.epoch, nextEpoch);
  assert.equal(Number(recovered.revision), 1);
  assert.equal(recovered.records.size, 1);
  assert.equal(recovered.records.has(JSON.stringify([profile, 'normal', 10])), false);
});
function handoffFor(profileId, contextKind) {
  const handoff = structuredClone(populated.handoff);
  handoff.profile_id = profileId;
  handoff.context_kind = contextKind;
  for (const record of handoff.records) {
    record.profile_id = profileId;
    record.context_kind = contextKind;
    record.tab_identity.profile_id = profileId;
    record.tab_identity.context_kind = contextKind;
  }
  return handoff;
}

test('IP-05-T09 isolates profile and context maps and disposes private references only', () => {
  const otherProfile = parseProfileID('123e4567-e89b-42d3-a456-426614174099');
  const first = new ProjectionReconciler(profile);
  const second = new ProjectionReconciler(otherProfile);
  for (const contextKind of ['normal', 'private']) {
    const handoff = handoffFor(profile, contextKind);
    assert.equal(first.acquire(handoff).ok, true);
    assert.equal(first.commitSnapshot(contextKind).outcome, 'COMMITTED');
  }
  const otherProfileHandoff = handoffFor(otherProfile, 'normal');
  assert.equal(second.acquire(otherProfileHandoff).ok, true);
  assert.equal(second.commitSnapshot('normal').outcome, 'COMMITTED');

  const key = (profileId, contextKind, tabId) => JSON.stringify([profileId, contextKind, tabId]);
  assert.equal(first.livePartition('normal').records.has(key(profile, 'normal', 10)), true);
  assert.equal(first.livePartition('private').records.has(key(profile, 'private', 10)), true);
  assert.equal(second.livePartition('normal').records.has(key(otherProfile, 'normal', 10)), true);
  assert.equal(first.livePartition('normal').records.size, populated.handoff.records.length);
  assert.equal(first.livePartition('private').records.size, populated.handoff.records.length);
  assert.equal(second.livePartition('normal').records.size, populated.handoff.records.length);

  const beforeMismatch = [...first.livePartition('normal').records];
  const mismatch = first.acquire(otherProfileHandoff);
  assert.equal(mismatch.ok, false);
  assert.equal(mismatch.reason, 'partition_profile_mismatch');
  assert.deepEqual([...first.livePartition('normal').records], beforeMismatch, 'another profile must not mutate this partition');

  first.disposePrivateContext();
  assert.equal(first.livePartition('private').records.size, 0, 'private records must be discarded on disposal');
  assert.equal(first.acquiredSnapshot('private'), undefined, 'private snapshot references must be discarded');
  assert.equal(first.livePartition('normal').records.size, populated.handoff.records.length, 'normal state must survive private disposal');
  assert.equal(second.livePartition('normal').records.size, populated.handoff.records.length, 'other profile state must survive private disposal');
});
