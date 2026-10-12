import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';

import { parseProfileID } from '../../../extension/dist/domain/index.js';
import { acquireSnapshot } from '../../../extension/dist/src/projection/reconciler.js';

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
