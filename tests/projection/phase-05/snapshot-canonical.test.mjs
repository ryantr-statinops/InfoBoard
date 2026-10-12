import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';

import { acceptSnapshot, emptyProjection, parseProfileID, parseProjectionEpoch } from '../../../extension/dist/domain/index.js';
import { acquireSnapshot, admitHandoff, reduceProjectionBatch } from '../../../extension/dist/src/projection/reconciler.js';

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
  const result = reduceProjectionBatch(state, 1, [noOp, update, remove].map(submissionOf));
  assert.equal(result.outcome, 'APPLIED');
  assert.equal(Number(state.revision), 3, 'the no-op sequence must not advance projection revision');
  assert.equal(result.event_sequence, 4, 'every accepted handoff must advance event sequence');
  assert.equal(state.records.size, 2, 'the reducer should not drop unrelated records');
  assert.equal(state.records.has(JSON.stringify([profile, 'normal', 10])), false);
  assert.equal(state.records.get(JSON.stringify([profile, 'normal', 11])).title_display, 'Beta updated');
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
