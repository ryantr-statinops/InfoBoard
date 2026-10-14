import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';

import { parseProfileID } from '../../../extension/dist/domain/index.js';
import { ProjectionReconciler, acquireSnapshot, admitHandoff } from '../../../extension/dist/src/projection/reconciler.js';

const corpus = JSON.parse(await readFile('fixtures/projection/phase-05/reconciliation.json', 'utf8'));
const authority = JSON.parse(await readFile(corpus.base_artifact, 'utf8'));
const base = authority.cases.find((item) => item.case_id === corpus.base_case_id).handoff;
const fixture = (id) => corpus.fixtures.find((item) => item.fixture_id === id);
const key = (profileId, contextKind, tabId) => JSON.stringify([profileId, contextKind, tabId]);

function eventHandoff(event, baseHandoff = base) {
  const record = structuredClone(baseHandoff.records.find((candidate) => candidate.tab_identity.tab_id === event.tab_id));
  assert.ok(record, `fixture base must contain tab ${event.tab_id}`);
  if (event.title_display !== undefined) record.title_display = event.title_display;
  if (event.title_search !== undefined) record.title_search = event.title_search;
  record.projection_revision = event.revision;
  const handoff = {
    ...baseHandoff,
    kind: event.kind,
    previous_revision: event.previous_revision,
    projection_revision: event.revision,
    event_sequence: event.sequence,
    effective_change: event.effective_change,
    tab_identity: record.tab_identity,
  };
  if (event.kind !== 'remove') handoff.record = record;
  else delete handoff.record;
  return handoff;
}

function partitionHandoff(profileId, contextKind, tabIds) {
  const handoff = structuredClone(base);
  handoff.records = handoff.records.filter((record) => tabIds.includes(record.tab_identity.tab_id));
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
function snapshotHandoff(epoch, tabIds, sequence) {
  const handoff = structuredClone(base);
  handoff.projection_epoch = epoch;
  handoff.previous_revision = 0;
  handoff.projection_revision = 1;
  handoff.event_sequence = sequence;
  handoff.records = handoff.records.filter((record) => tabIds.includes(record.tab_identity.tab_id));
  for (const record of handoff.records) {
    record.projection_epoch = epoch;
    record.projection_revision = 1;
  }
  return handoff;
}

function admit(profileId, handoff) {
  const result = admitHandoff(parseProfileID(profileId), handoff);
  assert.equal(result.ok, true, 'shared fixture event must be admitted');
  return result.submission;
}

function committedBase() {
  const reconciler = new ProjectionReconciler(parseProfileID(corpus.profile_id));
  assert.equal(reconciler.acquire(structuredClone(base)).ok, true);
  assert.equal(reconciler.commitSnapshot('normal').outcome, 'COMMITTED');
  return reconciler;
}

test('IP-05-T11 shared ordered-delta fixture verifies exact canonical records', () => {
  const item = fixture('FX-PROJECTION-ORDERED-DELTAS');
  const reconciler = committedBase();
  let outcomes = [];
  for (const event of item.events) {
    const result = reconciler.reduceDeltas('normal', [admit(corpus.profile_id, eventHandoff(event))]);
    outcomes.push(result.outcome);
  }
  assert.deepEqual(outcomes, item.expected.outcomes);
  const state = reconciler.livePartition('normal');
  assert.equal(Number(state.revision), item.expected.revision);
  assert.equal(state.records.size, item.expected.identity_ids.length);
  assert.deepEqual([...state.records.values()].map((record) => record.tab_identity.tab_id), item.expected.identity_ids);
  assert.equal(state.records.get(key(corpus.profile_id, 'normal', 11)).title_display, item.expected.record_title);
});

test('IP-05-T11 duplicate fixture is idempotent; stale revision and dropped event fixtures require resync', () => {
  const duplicateFixture = fixture('FX-PROJECTION-DUPLICATE-DELTA');
  const duplicate = committedBase();
  const submission = admit(corpus.profile_id, eventHandoff(duplicateFixture.events[0]));
  const results = [duplicate.reduceDeltas('normal', [submission]).outcome];
  if (duplicateFixture.replay_exactly) results.push(duplicate.reduceDeltas('normal', [submission]).outcome);
  assert.deepEqual(results, duplicateFixture.expected.outcomes);
  assert.equal(Number(duplicate.livePartition('normal').revision), duplicateFixture.expected.revision);
  assert.deepEqual([...duplicate.livePartition('normal').records.values()].map((record) => record.tab_identity.tab_id), duplicateFixture.expected.identity_ids);
  assert.equal(duplicate.livePartition('normal').records.get(key(corpus.profile_id, 'normal', 11)).title_display, duplicateFixture.expected.record_title);

  for (const id of ['FX-PROJECTION-STALE-REVISION', 'FX-PROJECTION-MISSED-EVENT']) {
    const item = fixture(id);
    const reconciler = committedBase();
    const result = reconciler.reduceDeltas('normal', [admit(corpus.profile_id, eventHandoff(item.events[0]))]);
    assert.deepEqual([result.outcome], item.expected.outcomes, id);
    assert.equal(result.resync_required, item.expected.resync_required, id);
    assert.equal(Number(reconciler.livePartition('normal').revision), item.expected.revision, id);
    assert.deepEqual([...reconciler.livePartition('normal').records.values()].map((record) => record.tab_identity.tab_id), item.expected.identity_ids, id);
  }
});

test('IP-05-T11 atomic replacement, restart, and profile/context fixtures agree on identity sets', () => {
  const replaceFixture = fixture('FX-PROJECTION-ATOMIC-REPLACE');
  const replaced = committedBase();
  const replacement = structuredClone(base);
  replacement.projection_epoch = replaceFixture.fresh_snapshot.epoch;
  replacement.projection_revision = replaceFixture.expected.revision;
  replacement.previous_revision = 0;
  replacement.event_sequence = replaceFixture.fresh_snapshot.sequence;
  replacement.records = replacement.records.filter((record) => replaceFixture.fresh_snapshot.record_ids.includes(record.tab_identity.tab_id));
  for (const record of replacement.records) {
    record.projection_epoch = replacement.projection_epoch;
    record.projection_revision = 1;
  }
  assert.equal(replaced.acquire(replacement).ok, true);
  assert.equal(replaced.commitSnapshot('normal').outcome, replaceFixture.expected.outcome);
  assert.equal(replaced.livePartition('normal').epoch, replaceFixture.expected.epoch);
  assert.equal(Number(replaced.livePartition('normal').revision), replaceFixture.expected.revision);
  assert.deepEqual([...replaced.livePartition('normal').records.values()].map((record) => record.tab_identity.tab_id), replaceFixture.expected.identity_ids);

  const restartFixture = fixture('FX-PROJECTION-RESTART');
  const restarted = committedBase();
  restarted.requireSnapshot('normal');
  const stale = restarted.reduceDeltas('normal', [admit(corpus.profile_id, eventHandoff(restartFixture.events[0]))]);
  const fresh = structuredClone(base);
  fresh.projection_epoch = restartFixture.fresh_snapshot.epoch;
  fresh.previous_revision = 0;
  fresh.projection_revision = 1;
  fresh.event_sequence = restartFixture.fresh_snapshot.sequence;
  fresh.records = fresh.records.filter((record) => restartFixture.fresh_snapshot.record_ids.includes(record.tab_identity.tab_id));
  for (const record of fresh.records) {
    record.projection_epoch = fresh.projection_epoch;
    record.projection_revision = 1;
  }
  assert.equal(restarted.acquire(fresh).ok, true);
  assert.equal(restarted.commitSnapshot('normal').outcome, 'COMMITTED');
  assert.deepEqual([stale.outcome, 'COMMITTED'], restartFixture.expected.outcomes);
  assert.equal(restarted.livePartition('normal').epoch, restartFixture.expected.epoch);
  assert.equal(Number(restarted.livePartition('normal').revision), restartFixture.expected.revision);
  assert.deepEqual([...restarted.livePartition('normal').records.values()].map((record) => record.tab_identity.tab_id), restartFixture.expected.identity_ids);

  const isolation = fixture('FX-PROJECTION-PROFILE-BOUNDARY');
  const partitions = [];
  for (const profileId of isolation.profiles) {
    const partition = new ProjectionReconciler(parseProfileID(profileId));
    for (const contextKind of isolation.contexts) {
      const handoff = partitionHandoff(profileId, contextKind, isolation.same_tab_ids);
      assert.equal(partition.acquire(handoff).ok, true);
      assert.equal(partition.commitSnapshot(contextKind).outcome, 'COMMITTED');
      const ids = [...partition.livePartition(contextKind).records.values()].map((record) => record.tab_identity.tab_id);
      assert.deepEqual(ids, isolation.same_tab_ids);
      partitions.push({ profileId, contextKind, partition, handoff });
    }
  }
  assert.equal(partitions.length, isolation.expected.partition_count);
  const target = partitions[0];
  const before = [...target.partition.livePartition(target.contextKind).records];
  const mismatch = target.partition.acquire(partitionHandoff(isolation.profiles[1], target.contextKind, isolation.same_tab_ids));
  assert.equal(mismatch.ok, false);
  assert.deepEqual([...target.partition.livePartition(target.contextKind).records], before);
  target.partition.disposePrivateContext();
  assert.equal(target.partition.livePartition('private').records.size, 0);
  assert.equal(target.partition.livePartition('normal').records.size, isolation.expected.records_per_partition);
});

test('IP-05-T11 canonical and safe-handoff fixtures preserve deterministic, content-free views', () => {
  const canonical = fixture('FX-PROJECTION-CANONICAL-OUTPUT');
  const reordered = structuredClone(base);
  const rows = new Map(reordered.records.map((record) => [record.tab_identity.tab_id, record]));
  reordered.records = canonical.input_record_order.map((id) => rows.get(id));
  const result = acquireSnapshot(parseProfileID(corpus.profile_id), reordered);
  assert.equal(result.ok, true);
  assert.deepEqual(result.snapshot.records.map((record) => record.tab_identity.tab_id), canonical.expected.identity_ids);
  const optional = result.snapshot.records.find((record) => record.tab_identity.tab_id === canonical.expected.optional_absence_tab_id);
  assert.equal(Object.hasOwn(optional, 'window_label_display'), false);

  const metadata = fixture('FX-PROJECTION-SAFE-HANDOFF-METADATA');
  const privateValues = structuredClone(base);
  privateValues.records[0].title_display = metadata.raw_values[0];
  privateValues.records[0].title_search = metadata.raw_values[1];
  privateValues.records[0].url_display = 'https://' + metadata.raw_values[2] + '.invalid';
  privateValues.records[0].url_search = privateValues.records[0].url_display;
  const reconciler = new ProjectionReconciler(parseProfileID(corpus.profile_id));
  assert.equal(reconciler.acquire(privateValues).ok, true);
  assert.equal(reconciler.commitSnapshot('normal').outcome, 'COMMITTED');
  const view = reconciler.view();
  const serialized = JSON.stringify(view);
  for (const raw of metadata.raw_values) assert.equal(serialized.includes(raw), false);
  assert.equal(view.profile_id, corpus.profile_id);
});
test('IP-05-T12 repeats recovery across a fresh session and epoch', () => {
  const item = fixture('FX-PROJECTION-RECOVERY-SEQUENCE');
  const reconciler = committedBase();
  let currentSnapshot = structuredClone(base);

  for (const [index, cycle] of item.cycles.entries()) {
    if (cycle.require_snapshot) reconciler.requireSnapshot('normal');
    if (cycle.start_snapshot !== undefined) {
      currentSnapshot = snapshotHandoff(cycle.start_snapshot.epoch, cycle.start_snapshot.record_ids, cycle.start_snapshot.sequence);
      assert.equal(reconciler.acquire(currentSnapshot).ok, true);
      assert.equal(reconciler.commitSnapshot('normal').outcome, 'COMMITTED');
    }

    const accepted = reconciler.reduceDeltas('normal', [admit(corpus.profile_id, eventHandoff(cycle.accepted_event, currentSnapshot))]);
    assert.equal(accepted.outcome, cycle.expected.accepted_outcome, 'cycle ' + (index + 1) + ' accepted delta');
    const dropped = reconciler.reduceDeltas('normal', [admit(corpus.profile_id, eventHandoff(cycle.dropped_event, currentSnapshot))]);
    assert.equal(dropped.outcome, cycle.expected.dropped_outcome, 'cycle ' + (index + 1) + ' dropped event');
    assert.equal(dropped.resync_required, cycle.expected.resync_required, 'cycle ' + (index + 1) + ' requires a full snapshot');
    assert.deepEqual([...reconciler.livePartition('normal').records.values()].map((record) => record.tab_identity.tab_id), cycle.expected.resync_identity_ids);

    currentSnapshot = snapshotHandoff(cycle.recovery_snapshot.epoch, cycle.recovery_snapshot.record_ids, cycle.recovery_snapshot.sequence);
    assert.equal(reconciler.acquire(currentSnapshot).ok, true);
    assert.equal(reconciler.commitSnapshot('normal').outcome, 'COMMITTED');
    const recovered = reconciler.livePartition('normal');
    assert.equal(recovered.epoch, cycle.recovery_snapshot.epoch);
    assert.equal(Number(recovered.revision), 1);
    assert.deepEqual([...recovered.records.values()].map((record) => record.tab_identity.tab_id), cycle.expected.identity_ids);
  }

  const final = reconciler.livePartition('normal');
  const identities = [...final.records.values()].map((record) => record.tab_identity.tab_id);
  assert.deepEqual(identities, item.eligible_browser_ids, 'final committed identities must equal the eligible browser oracle');
  assert.equal(new Set(identities).size, item.expected.unique_identity_count, 'the committed map must not duplicate identities');
  assert.equal(final.epoch, item.expected.final_epoch);
  assert.equal(Number(final.revision), item.expected.final_revision);
  assert.equal(Number(reconciler.view().snapshots.find((entry) => entry.context_kind === 'normal').projection_revision), item.expected.final_revision);
});
