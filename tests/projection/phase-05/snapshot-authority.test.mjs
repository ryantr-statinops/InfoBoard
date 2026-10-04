// The extension-side consumer of the shared IP-05-T02 snapshot-authority corpus.
// Every declared case is driven through the real producer acquisition in
// extension/src/projection/reconciler.ts: a completed read of the known current
// partition is authority with exactly the records IP-04 declared, an intentional
// empty read is authority and reports itself explicitly empty, and a denied,
// failed, partial, unavailable, or resync-required handoff is not authority at all:
// it publishes no record, always requires a fresh authoritative read, and leaves the
// previously acquired snapshot untouched. The consumer asserts only what the real
// acquisition reports; it never restates an authority rule. Acquisition is not
// commit, so IP-05-T07 still owns publication and no case here may leave a live
// projection map or a committed lineage behind.
import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile, stat } from 'node:fs/promises';

import { parseProfileID } from '../../../extension/dist/domain/index.js';
import {
  ProjectionReconciler,
  PROJECTION_MAX_SNAPSHOT_RECORDS,
  PROJECTION_REJECT_REASONS,
  PROJECTION_SNAPSHOT_OUTCOMES,
  acquireSnapshot,
  readProjectionRecord,
} from '../../../extension/dist/src/projection/reconciler.js';

const ARTIFACT = 'fixtures/projection/phase-05/snapshot-authority.json';
const BOUNDARY_ARTIFACT = 'fixtures/projection/phase-05/phase-05.json';
const FIXTURE_ID = 'FX-PROJECTION-SNAPSHOT-AUTHORITY';
const OWNER_TASK = 'IP-05-T02';
const AUTHORITATIVE = 'AUTHORITATIVE';
const CONTEXTS = ['normal', 'private'];
const SIGNAL_CONDITIONS = ['completed', 'denied', 'failed', 'unsupported', 'incomplete', 'disconnect', 'undeclared'];
const OWNERS_REQUIRED = ['IP-05-T03', 'IP-05-T04', 'IP-05-T05', 'IP-05-T07', 'IP-05-T09', 'IP-05-T12'];
// The acquisition clauses every corpus must prove, so the evidence cannot shrink
// without a case disappearing from the artifact.
const REQUIRED_REASONS = [
  'missing_records',
  'missing_context_kind',
  'missing_projection_epoch',
  'snapshot_resync_required',
  'missing_resync_marker',
  'invalid_resync_marker',
  'non_snapshot_handoff',
  'unpartitioned_handoff',
  'invalid_record',
  'partition_profile_mismatch',
];
// The bounded outcome vocabulary each corpus observation counts.
const OUTCOME_SIGNALS = {
  AUTHORITATIVE: 'authoritative_cases',
  SNAPSHOT_REQUIRED: 'snapshot_required_cases',
  PROFILE_MISMATCH: 'profile_mismatch_cases',
};

const corpus = JSON.parse(await readFile(ARTIFACT, 'utf8'));
const boundary = JSON.parse(await readFile(BOUNDARY_ARTIFACT, 'utf8'));
const authority = corpus.authority;
const cases = corpus.cases;
const profile = parseProfileID(corpus.input.profile_id);

function includes(values, value) {
  return values.includes(value);
}

// recordsOf returns the declared record list of a case without repairing it: a case
// that declares no list stays distinguishable from a case that declares an empty one.
function recordsOf(item) {
  return Array.isArray(item.handoff.records) ? item.handoff.records : null;
}

function isPlainObject(value) {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

// setAtPath writes one value into the dotted field path an expansion names. Every
// object on the path is copied on the way down, so each materialized record owns its
// own nested objects and the shared template is never mutated or aliased.
function setAtPath(template, path, value) {
  const keys = String(path).split('.');
  const root = { ...template };
  let target = root;
  for (const key of keys.slice(0, -1)) {
    const child = target[key];
    assert.ok(isPlainObject(child), `the expansion path ${path} is not a record path`);
    target[key] = { ...child };
    target = target[key];
  }
  target[keys.at(-1)] = value;
  return root;
}

// expandedRecords materializes the bounded record list a case declares through
// record_expansion. The artifact stores a template, a count, and the one identity
// field the expansion substitutes instead of writing a ten-thousand-record body as
// fixture text, so the real acquisition still receives the declared number of
// records while the artifact stays reviewable. The list is an arithmetic sequence of
// identities over one template, which keeps it deterministic across both consumers.
function expandedRecords(item) {
  const expansion = item.record_expansion;
  if (expansion === undefined) return null;
  assert.ok(isPlainObject(expansion.template), `${item.case_id} declares an expansion with no template`);
  assert.ok(Number.isSafeInteger(expansion.count) && expansion.count > 0, `${item.case_id} declares no expansion count`);
  assert.ok(Number.isSafeInteger(expansion.identity_start), `${item.case_id} declares no expansion identity start`);
  assert.ok(Number.isSafeInteger(expansion.identity_step) && expansion.identity_step > 0, `${item.case_id} declares no expansion step`);
  assert.ok(typeof expansion.identity_path === 'string' && expansion.identity_path.length > 0, `${item.case_id} declares no expansion identity path`);
  assert.ok(expansion.expansion_note.length > 0, `${item.case_id} states no expansion note`);
  const records = [];
  for (let index = 0; index < expansion.count; index += 1) {
    records.push(setAtPath(expansion.template, expansion.identity_path, expansion.identity_start + index * expansion.identity_step));
  }
  return records;
}

// materializedRecords is the record list a case really declares: its literal list, or
// the one its expansion describes. A case that declares neither stays null, which is
// what keeps a missing list distinguishable from an empty one.
function materializedRecords(item) {
  return expandedRecords(item) ?? recordsOf(item);
}

// handoffOf returns the handoff the real acquisition is driven with. An expanding case
// declares no literal list, so the list its expansion describes is handed over here
// rather than being written into the artifact.
function handoffOf(item) {
  const records = expandedRecords(item);
  return records === null ? item.handoff : { ...item.handoff, records };
}

function observationOf(signal) {
  const declared = corpus.expected.observations.find((entry) => entry.signal === signal);
  assert.ok(declared, `the corpus declares no ${signal} observation`);
  return declared.value;
}

// declaredRecordValues collects the text a case declares, so a bounded failure can be
// proven to carry no title, URL, or domain value. The values are collected once each:
// an expanded list repeats one template, so the unique set is the whole vocabulary a
// refusal could leak without making the leak scan unbounded.
function declaredRecordValues(item) {
  const values = new Set();
  for (const record of materializedRecords(item) ?? []) {
    for (const field of ['title_display', 'title_search', 'url_display', 'url_search', 'domain_display', 'domain_search']) {
      if (typeof record[field] === 'string' && record[field] !== '') values.add(record[field]);
    }
  }
  return [...values];
}

// snapshotView returns the acquired-snapshot view of one context.
function snapshotView(reconciler, contextKind) {
  return reconciler.view().snapshots.find((entry) => entry.context_kind === contextKind);
}

// assertAuthority proves an authoritative acquisition forwarded the fence and the
// records exactly as IP-04 declared them, and that the boundary reported no resync
// requirement of its own.
function assertAuthority(item, handoff, acquisition) {
  const declared = item.expected.producer;
  assert.equal(acquisition.ok, true, `${item.case_id} was not authority: ${acquisition.reason}`);
  assert.equal(acquisition.authoritative, true, `${item.case_id} authoritative flag`);
  assert.equal(acquisition.outcome, declared.outcome, `${item.case_id} outcome`);
  assert.equal(acquisition.snapshot.record_count, materializedRecords(item).length, `${item.case_id} record count`);
  assert.equal(acquisition.snapshot.explicitly_empty, declared.explicitly_empty, `${item.case_id} explicitly empty`);
  // The authority is the browser read IP-04 published, not a projection this module
  // assembled: the fence and every record are forwarded verbatim.
  assert.equal(acquisition.snapshot.fence.profile_id, handoff.profile_id, `${item.case_id} profile`);
  assert.equal(acquisition.snapshot.fence.context_kind, handoff.context_kind, `${item.case_id} context`);
  assert.equal(acquisition.snapshot.fence.projection_epoch, handoff.projection_epoch, `${item.case_id} epoch`);
  assert.equal(acquisition.snapshot.fence.projection_revision, handoff.projection_revision, `${item.case_id} revision`);
  assert.equal(acquisition.snapshot.fence.previous_revision, handoff.previous_revision, `${item.case_id} previous revision`);
  assert.equal(acquisition.snapshot.fence.event_sequence, handoff.event_sequence, `${item.case_id} sequence`);
  assert.deepEqual([...acquisition.snapshot.records], materializedRecords(item), `${item.case_id} records were rewritten`);
}

// assertNoAuthority proves a non-authoritative acquisition published nothing that
// could be read as a converged empty projection: no record, an explicit fresh-read
// requirement, and the IP-04 marker of the failed read carried through bounded.
function assertNoAuthority(item, handoff, acquisition) {
  const declared = item.expected.producer;
  assert.equal(acquisition.ok, false, `${item.case_id} was treated as authority`);
  assert.equal(acquisition.authoritative, false, `${item.case_id} authoritative flag`);
  assert.equal(acquisition.outcome, declared.outcome, `${item.case_id} outcome`);
  assert.equal(acquisition.reason, declared.reason, `${item.case_id} reason`);
  assert.ok(PROJECTION_REJECT_REASONS.includes(acquisition.reason), `${item.case_id} reason ${acquisition.reason} is outside the closed vocabulary`);
  assert.equal(acquisition.resync_required, true, `${item.case_id} must require a fresh authoritative read`);
  assert.equal(acquisition.snapshot, undefined, `${item.case_id} must publish no snapshot`);
  assert.equal(declared.record_count, 0, `${item.case_id} declares records in a non-authoritative acquisition`);
  if (handoff.resync_reason !== undefined) {
    assert.equal(acquisition.resync_reason, handoff.resync_reason, `${item.case_id} resync marker was rewritten`);
  }
  // The failure is the only value the runtime may report, so it must carry bounded
  // vocabulary only and no tab-derived text.
  const rendered = JSON.stringify(acquisition);
  for (const value of declaredRecordValues(item)) {
    assert.ok(!rendered.includes(value), `${item.case_id} the reported failure leaks a declared record value`);
  }
}

// driveAll runs every case in declared order through one real reconciler and returns
// the acquisition each case received. Acquisition is not commit, so the live
// projection map must stay empty for the whole corpus.
function driveAll() {
  const reconciler = new ProjectionReconciler(profile);
  const acquired = [];
  let retained = { normal: undefined, private: undefined };
  let firstAuthority;
  let firstRefusal;
  let refusals = 0;
  for (const item of cases) {
    const handoff = handoffOf(item);
    const before = snapshotView(reconciler, item.partition_context);
    const previous = retained[item.partition_context];
    // The pure seam and the stateful seam must agree for every case.
    const pure = acquireSnapshot(profile, handoff);
    const acquisition = reconciler.acquire(handoff);
    assert.deepEqual(
      JSON.parse(JSON.stringify(acquisition)),
      JSON.parse(JSON.stringify(pure)),
      `${item.case_id}: the reconciler and the pure acquisition disagree`,
    );
    if (item.acquisition === AUTHORITATIVE) {
      assertAuthority(item, handoff, acquisition);
      const view = snapshotView(reconciler, item.partition_context);
      assert.equal(view.retained_authoritative, true, `${item.case_id} retained flag`);
      assert.equal(view.record_count, item.expected.producer.record_count, `${item.case_id} retained record count`);
      assert.equal(view.last_outcome, AUTHORITATIVE, `${item.case_id} retained outcome`);
      assert.equal(view.epoch, handoff.projection_epoch, `${item.case_id} retained epoch`);
      assert.equal(view.projection_revision, handoff.projection_revision, `${item.case_id} retained revision`);
      assert.equal(view.event_sequence, handoff.event_sequence, `${item.case_id} retained sequence`);
      retained[item.partition_context] = reconciler.acquiredSnapshot(item.partition_context);
      if (firstAuthority === undefined) {
        firstAuthority = { case_id: item.case_id, context: item.partition_context, snapshot: retained[item.partition_context] };
        assert.equal(
          firstAuthority.snapshot.record_count,
          corpus.expected.invariants.producer_retained_records.authoritative_case,
          `${item.case_id}: the corpus declares a different authoritative record count`,
        );
      }
    } else {
      assertNoAuthority(item, handoff, acquisition);
      refusals += 1;
      // A denied, failed, partial, or unavailable read must not erase what a
      // completed read acquired, and it must not create authority where none existed.
      assert.deepEqual(
        JSON.parse(JSON.stringify(reconciler.acquiredSnapshot(item.partition_context) ?? null)),
        JSON.parse(JSON.stringify(previous ?? null)),
        `${item.case_id}: a non-authoritative read changed the retained snapshot`,
      );
      if (acquisition.fence === undefined) {
        // Nothing about this handoff can be attributed to a partition, so no
        // partition view may move at all.
        assert.deepEqual(snapshotView(reconciler, item.partition_context), before, `${item.case_id}: an unattributable handoff moved its partition view`);
      } else {
        const view = snapshotView(reconciler, item.partition_context);
        assert.equal(view.last_outcome, acquisition.outcome, `${item.case_id} reported outcome`);
        assert.equal(view.retained_authoritative, before.retained_authoritative, `${item.case_id} retained flag`);
      }
      if (firstRefusal === undefined) {
        firstRefusal = { case_id: item.case_id, context: item.partition_context, view: snapshotView(reconciler, item.partition_context) };
        assert.equal(
          firstRefusal.view.record_count,
          corpus.expected.invariants.producer_retained_records.after_first_refusal,
          `${item.case_id}: the corpus declares a different retained record count after a refusal`,
        );
      }
    }
    acquired.push(acquisition);
    // Acquisition never writes the live projection map: IP-05-T07 owns commit.
    const live = reconciler.livePartition(item.partition_context);
    assert.equal(live.records.size, 0, `${item.case_id} wrote the live projection map`);
    assert.equal(live.epoch, null, `${item.case_id} published a live epoch`);
    assert.equal(Number(live.revision), 0, `${item.case_id} published a live revision`);
  }
  return { reconciler, acquired, refusals, retained, firstAuthority, firstRefusal };
}

test('IP-05-T02 snapshot-authority corpus claims the reserved fixture and defers the rest of the phase', async () => {
  assert.equal(corpus.schema_version, 1);
  assert.equal(corpus.phase, 'IP-05');
  assert.equal(corpus.artifact, 'projection-snapshot-authority');
  assert.equal(corpus.owner_task, OWNER_TASK);
  assert.equal(corpus.fixture_id, FIXTURE_ID);
  assert.deepEqual(corpus.shared_consumers, ['go', 'node']);
  assert.ok(corpus.requirement_ids.length > 0);
  assert.ok(corpus.note.length > 0 && corpus.outcome_note.length > 0);
  for (const source of corpus.contract_source) {
    assert.ok((await stat(source)).isFile(), `contract source ${source} is absent`);
  }
  assert.ok(corpus.expected.privacy_assertions.length > 0, 'the corpus declares no privacy assertion');
  assert.ok(corpus.expected.invariants.note.length > 0, 'the corpus states no invariant note');
  // The reserved ID may be claimed only once and only while the boundary artifact
  // still reserves it without implementing it.
  assert.deepEqual(corpus.claims.reserved_fixture_ids, [FIXTURE_ID]);
  assert.ok(corpus.claims.claim_note.length > 0, 'the corpus states no claim note');
  assert.equal(corpus.claims.boundary_artifact, BOUNDARY_ARTIFACT);
  assert.ok(includes(boundary.reserved_fixture_ids, FIXTURE_ID), `${FIXTURE_ID} is not reserved by the boundary artifact`);
  assert.ok(
    !boundary.fixtures.some((fixture) => fixture.fixture_id === FIXTURE_ID),
    `the boundary artifact already implements ${FIXTURE_ID}`,
  );
  // The declared vocabulary is the real one, and the boundary owns one record bound.
  assert.deepEqual([...authority.outcome_vocabulary].sort(), [...PROJECTION_SNAPSHOT_OUTCOMES].sort());
  assert.equal(authority.authoritative_outcome, AUTHORITATIVE);
  assert.ok(PROJECTION_MAX_SNAPSHOT_RECORDS > 0, 'the declared snapshot record bound is absent from the producer');
  for (const reason of authority.reject_reason_vocabulary) {
    assert.ok(PROJECTION_REJECT_REASONS.includes(reason), `declared reason ${reason} is outside the real vocabulary`);
  }
  for (const reason of REQUIRED_REASONS) {
    assert.ok(cases.some((item) => item.expected.producer.reason === reason), `the corpus proves no refusal with reason ${reason}`);
  }
  for (const condition of authority.required_conditions) {
    assert.ok(condition.id && condition.rule.length > 0, `condition ${condition.id} declares no rule`);
  }
  for (const precondition of authority.record_preconditions) {
    assert.ok(precondition.field && precondition.owner && precondition.rule.length > 0, `record precondition ${precondition.field} is incomplete`);
  }
  for (const [signal, effect] of Object.entries(authority.resync_signals)) {
    assert.ok(signal.length > 0 && effect.length > 0, `resync signal ${signal} declares no effect`);
  }
  assert.ok(authority.producer_signal_note.length > 0 && authority.host_field_note.length > 0, 'the corpus states no producer signal or host field note');
  for (const owner of OWNERS_REQUIRED) {
    assert.ok(
      authority.deferred_clauses.some((clause) => clause.owner_task === owner && clause.reason.length > 0),
      `the corpus defers no clause to ${owner}`,
    );
  }
  for (const clause of authority.deferred_clauses) {
    assert.ok(clause.owner_task.startsWith('IP-05-T') && clause.owner_task !== OWNER_TASK, `${clause.clause} names owner ${clause.owner_task}`);
  }
});

test('IP-05-T02 corpus cases agree with the producer acquisition they declare', () => {
  const signals = new Set();
  const tallies = { AUTHORITATIVE: 0, SNAPSHOT_REQUIRED: 0, PROFILE_MISMATCH: 0 };
  const seen = new Set();
  for (const item of cases) {
    assert.ok(item.case_id && item.note.length > 0 && item.handoff, `${item.case_id} declares no id, handoff, or note`);
    assert.ok(!seen.has(item.case_id), `${item.case_id} appears twice`);
    seen.add(item.case_id);
    assert.ok(includes(CONTEXTS, item.partition_context), `${item.case_id} partition_context`);
    if (item.acquisition === AUTHORITATIVE) {
      assert.equal(item.partition_context, corpus.input.context_kind, `${item.case_id} authority for an unbound context`);
    }
    assert.ok(includes(authority.outcome_vocabulary, item.acquisition), `${item.case_id} acquisition ${item.acquisition}`);
    signals.add(item.acquisition_signal);
    const declared = item.expected.producer;
    assert.equal(declared.outcome, item.acquisition, `${item.case_id} declared outcome`);
    assert.equal(declared.authoritative, item.acquisition === AUTHORITATIVE, `${item.case_id} declared authority`);
    if (item.acquisition === AUTHORITATIVE) {
      assert.equal(declared.reason, null, `${item.case_id} declares a reason`);
      assert.equal(declared.resync_required, false, `${item.case_id} declares a resync requirement`);
      assert.ok(materializedRecords(item) !== null, `${item.case_id} must declare its record list explicitly`);
      assert.equal(declared.record_count, materializedRecords(item).length, `${item.case_id} declared record count`);
      assert.equal(declared.explicitly_empty, declared.record_count === 0, `${item.case_id} declared empty flag`);
      assert.equal(declared.retained, true, `${item.case_id} declared retention`);
      assert.equal(item.expected.host.call, 'snapshot', `${item.case_id} authority must bind the snapshot call`);
      tallies.AUTHORITATIVE += 1;
      continue;
    }
    // Every non-authoritative acquisition publishes no record and requires a fresh
    // read, so a failed read can never look like a converged empty projection.
    assert.ok(includes(authority.reject_reason_vocabulary, declared.reason), `${item.case_id} reason outside the declared vocabulary`);
    assert.equal(declared.resync_required, true, `${item.case_id} must require a fresh authoritative read`);
    assert.equal(declared.record_count, 0, `${item.case_id} declares records in a non-authoritative acquisition`);
    assert.equal(declared.explicitly_empty, false, `${item.case_id} declares an empty read it never made`);
    assert.equal(declared.retained, false, `${item.case_id} declares retention for a non-authoritative read`);
    const host = item.expected.host;
    assert.ok(includes(['snapshot', 'handoff'], host.call), `${item.case_id} host call`);
    if (host.call === 'handoff') assert.notEqual(item.handoff.kind, 'snapshot', `${item.case_id} snapshot handoff bound as a handoff`);
    if (host.validation === 'REFUSED') {
      assert.ok(host.cause && host.code && host.index !== null, `${item.case_id} refusal declares no cause, code, or position`);
      assert.equal(host.records, 0, `${item.case_id} refused call declares records`);
    } else {
      assert.equal(host.cause, null, `${item.case_id} admitted call declares a cause`);
      assert.equal(host.code, null, `${item.case_id} admitted call declares a code`);
      assert.equal(host.index, null, `${item.case_id} admitted call declares a position`);
      assert.equal(host.records, materializedRecords(item)?.length ?? 0, `${item.case_id} admitted call record count`);
    }
    tallies[item.acquisition] += 1;
  }
  for (const signal of SIGNAL_CONDITIONS) {
    assert.ok(signals.has(signal), `the corpus declares no ${signal} acquisition signal`);
  }
  for (const [outcome, signal] of Object.entries(OUTCOME_SIGNALS)) {
    assert.ok(tallies[outcome] > 0, `the corpus declares no ${outcome} case`);
    assert.equal(observationOf(signal), tallies[outcome], `${signal} disagrees with the declared cases`);
  }
});

test('IP-05-T02 acquires a complete populated snapshot and an explicit empty read as authority', () => {
  const populated = cases.find((item) => item.case_id === 'complete_populated_snapshot');
  const empty = cases.find((item) => item.case_id === 'successful_explicit_empty_snapshot');
  assert.ok(populated && empty, 'the corpus declares no populated or empty authority case');
  assert.equal(populated.acquisition, AUTHORITATIVE);
  assert.equal(observationOf('authoritative_record_count'), populated.expected.producer.record_count);
  assert.deepEqual(recordsOf(empty), [], 'the empty read must declare an explicit empty list');
  assert.equal(empty.expected.producer.explicitly_empty, true, 'the empty read must declare itself explicitly empty');
  const { reconciler, firstAuthority, retained } = driveAll();
  assert.equal(firstAuthority.case_id, populated.case_id, 'the first authoritative case');
  assert.equal(firstAuthority.snapshot.record_count, observationOf('authoritative_record_count'), 'retained authoritative records');
  assert.equal(firstAuthority.snapshot.explicitly_empty, false, 'the populated read is not an empty read');
  assert.equal(firstAuthority.snapshot.fence.projection_epoch, populated.handoff.projection_epoch);
  // An intentional empty read is still authority: it converges the retained set on
  // no eligible tab instead of leaving the previous read in place.
  assert.equal(retained.normal.record_count, 0, 'the empty read converges the retained set');
  assert.equal(retained.normal.explicitly_empty, true, 'the retained snapshot is explicitly empty');
  assert.equal(retained.private, undefined, 'no case acquires authority for the private partition');
  const emptyReconciler = new ProjectionReconciler(profile);
  const emptyAcquisition = emptyReconciler.acquire(empty.handoff);
  assert.equal(emptyAcquisition.ok, true, 'an explicit empty read must be authority');
  assert.equal(emptyAcquisition.snapshot.explicitly_empty, true);
  assert.equal(emptyAcquisition.snapshot.record_count, 0);
  assert.equal(snapshotView(emptyReconciler, 'normal').record_count, 0, 'an empty read retains no record');
  assert.equal(snapshotView(emptyReconciler, 'normal').retained_authoritative, true, 'an empty read is still retained authority');
  assert.equal(observationOf('committed_queryable_partitions'), 0, 'IP-05-T07 owns commit, so no partition may publish here');
  for (const partition of reconciler.view().partitions) {
    assert.equal(partition.record_count, 0, `${partition.context_kind} live records`);
  }
});

test('IP-05-T02 withholds authority from denied, failed, partial, unavailable, and resync-required handoffs', () => {
  const withheld = cases.filter((item) => item.acquisition !== AUTHORITATIVE);
  assert.ok(withheld.length > 0, 'the corpus withholds no authority case');
  const { reconciler, refusals, firstRefusal } = driveAll();
  assert.equal(refusals, withheld.length, 'every non-authoritative case must be refused by the real acquisition');
  // The declared retention invariants hold across the whole corpus: the completed
  // read keeps its records and no failed read adds or removes one.
  const declared = corpus.expected.invariants.producer_retained_records;
  assert.equal(declared.authoritative_case, observationOf('authoritative_record_count'));
  assert.ok(firstRefusal, 'the corpus drives no non-authoritative case');
  assert.equal(withheld.some((item) => item.case_id === firstRefusal.case_id), true, 'the first refusal must come from a withheld case');
  assert.equal(firstRefusal.view.record_count, declared.after_first_refusal, 'a refusal must not erase acquired records');
  assert.equal(firstRefusal.view.retained_authoritative, true, 'a refusal must not drop acquired authority');
  assert.equal(firstRefusal.view.last_outcome, 'SNAPSHOT_REQUIRED', 'a refusal reports a fresh-read outcome');
  assert.equal(observationOf('non_authoritative_record_count'), 0, 'a non-authoritative acquisition publishes no record');
  assert.equal(observationOf('partial_outputs'), 0, 'no acquisition publishes a partial record set');
  const diagnostics = reconciler.view().diagnostics;
  assert.equal(diagnostics.snapshots_authoritative, cases.length - refusals, 'authoritative acquisitions');
  assert.equal(diagnostics.snapshots_unauthoritative, refusals);
  assert.equal(
    Object.values(diagnostics.snapshot_refusals_by_reason).reduce((total, value) => total + value, 0),
    refusals,
    'snapshot refusal counters',
  );
});

test('IP-05-T02 refuses a read whose resync marker is absent or non-boolean', () => {
  const undeclared = cases.find((item) => item.case_id === 'undeclared_read_outcome_empty_snapshot');
  const nonBoolean = cases.find((item) => item.case_id === 'non_boolean_read_outcome_empty_snapshot');
  assert.ok(undeclared && nonBoolean, 'the corpus declares no undeclared-read-outcome case');
  // Both cases look like a completed empty read: a complete fence and an explicit
  // empty record list. Only the marker differs, and only a marker that IP-04 could
  // have stated as a boolean decides whether the browser call finished at all.
  for (const item of [undeclared, nonBoolean]) {
    assert.deepEqual(recordsOf(item), [], `${item.case_id} must declare an explicit empty record list`);
    assert.ok(
      typeof item.handoff.resync_required !== 'boolean',
      `${item.case_id} declares a boolean marker`,
    );
    const acquisition = acquireSnapshot(profile, item.handoff);
    assert.equal(acquisition.ok, false, `${item.case_id} was treated as a completed empty read`);
    assert.equal(acquisition.authoritative, false, `${item.case_id} authoritative flag`);
    assert.equal(acquisition.reason, item.expected.producer.reason, `${item.case_id} reason`);
    assert.equal(acquisition.resync_required, true, `${item.case_id} must require a fresh authoritative read`);
    assert.equal(acquisition.snapshot, undefined, `${item.case_id} published a snapshot`);
    // A refused undeclared read leaves an acquired snapshot exactly as it was.
    const reconciler = new ProjectionReconciler(profile);
    reconciler.acquire(cases.find((entry) => entry.case_id === 'complete_populated_snapshot').handoff);
    const before = reconciler.acquiredSnapshot('normal');
    assert.equal(before.record_count, observationOf('authoritative_record_count'), 'the completed read retained its records');
    reconciler.acquire(item.handoff);
    assert.deepEqual(
      JSON.parse(JSON.stringify(reconciler.acquiredSnapshot('normal'))),
      JSON.parse(JSON.stringify(before)),
      `${item.case_id} cleared the acquired snapshot`,
    );
    const view = snapshotView(reconciler, 'normal');
    assert.equal(view.last_outcome, 'SNAPSHOT_REQUIRED', `${item.case_id} reported outcome`);
    assert.equal(view.retained_authoritative, true, `${item.case_id} retained authority`);
    assert.equal(view.explicitly_empty, false, `${item.case_id} reported an empty read it never made`);
  }
  // The declared marker stays decisive in the other direction: an explicit false
  // with an empty record list is still the one authoritative empty read.
  const empty = cases.find((item) => item.case_id === 'successful_explicit_empty_snapshot');
  assert.equal(empty.handoff.resync_required, false, 'the empty read declares its marker');
  assert.equal(acquireSnapshot(profile, empty.handoff).ok, true, 'an explicit empty read must stay authority');
});

test('IP-05-T02 fails closed before it materializes a record payload', () => {
  const populated = cases.find((item) => item.case_id === 'complete_populated_snapshot');
  assert.ok(populated, 'the corpus declares no completed read');
  const [valid] = recordsOf(populated);
  assert.ok(valid, 'the completed read declares no record');
  // Every probe below carries a record the IP-02 record contract would refuse, so a
  // boundary that validated records before its authority gates would report
  // invalid_record instead of the reason declared here. Observing the gate reason is
  // therefore the evidence that no record was read, copied, or staged first.
  const unusable = { ...valid, projection_epoch: corpus.input.foreign_epoch };
  const oversized = Array.from({ length: PROJECTION_MAX_SNAPSHOT_RECORDS + 1 }, () => unusable);
  const probes = [
    { label: 'kind gate', handoff: { ...populated.handoff, kind: 'status', records: [unusable] }, reason: 'non_snapshot_handoff' },
    { label: 'marker gate', handoff: { ...populated.handoff, resync_required: true, records: [unusable] }, reason: 'snapshot_resync_required' },
    { label: 'undeclared marker gate', handoff: { ...populated.handoff, records: [unusable], resync_required: undefined }, reason: 'missing_resync_marker' },
    { label: 'record-list gate', handoff: { ...populated.handoff, records: { 0: unusable } }, reason: 'missing_records' },
    { label: 'bound gate', handoff: { ...populated.handoff, records: oversized }, reason: 'snapshot_bounds_exceeded' },
  ];
  for (const probe of probes) {
    const handoff = { ...probe.handoff };
    if (handoff.resync_required === undefined) delete handoff.resync_required;
    const reconciler = new ProjectionReconciler(profile);
    reconciler.acquire(populated.handoff);
    const before = reconciler.acquiredSnapshot('normal');
    const acquisition = reconciler.acquire(handoff);
    assert.equal(acquisition.ok, false, `${probe.label} was treated as authority`);
    assert.equal(acquisition.reason, probe.reason, `${probe.label} reason`);
    assert.equal(acquisition.snapshot, undefined, `${probe.label} published a snapshot`);
    assert.deepEqual(
      JSON.parse(JSON.stringify(reconciler.acquiredSnapshot('normal'))),
      JSON.parse(JSON.stringify(before)),
      `${probe.label} cleared the acquired snapshot`,
    );
    assert.equal(reconciler.livePartition('normal').records.size, 0, `${probe.label} wrote the live map`);
  }
  // The same complete handoff without the gate defeats is still authority, so the
  // probes above prove ordering rather than a permanently closed boundary.
  assert.equal(acquireSnapshot(profile, { ...populated.handoff, records: [valid] }).ok, true);
  // The bound itself is inclusive and is enforced by length alone: exactly the
  // declared maximum clears the gate, and one more record is refused before any of
  // them is validated.
  const atBound = Array.from({ length: PROJECTION_MAX_SNAPSHOT_RECORDS }, () => valid);
  const admitted = acquireSnapshot(profile, { ...populated.handoff, records: atBound });
  assert.equal(admitted.ok, true, `${PROJECTION_MAX_SNAPSHOT_RECORDS} records must clear the bound`);
  assert.equal(admitted.snapshot.record_count, PROJECTION_MAX_SNAPSHOT_RECORDS);
  assert.equal(
    acquireSnapshot(profile, { ...populated.handoff, records: [...atBound, valid] }).reason,
    'snapshot_bounds_exceeded',
    'one record past the bound must be refused',
  );
});

test('IP-05-T02 refuses one invalid record without publishing the records before it', () => {
  const invalid = cases.filter((item) => item.case_id.startsWith('invalid_record_'));
  assert.ok(invalid.length > 0, 'the corpus proves no invalid-record refusal');
  for (const item of invalid) {
    const records = recordsOf(item);
    assert.ok(records.length >= 2, `${item.case_id} must declare a valid record before the invalid one`);
    const leading = records[0];
    assert.equal(leading.eligible, true, `${item.case_id} leading record eligibility`);
    assert.equal(leading.projection_epoch, item.handoff.projection_epoch, `${item.case_id} leading record epoch`);
    assert.equal(typeof leading.tab_identity.tab_id, 'number', `${item.case_id} leading record tab id`);
    const acquisition = acquireSnapshot(profile, item.handoff);
    assert.equal(acquisition.ok, false, `${item.case_id} was accepted`);
    assert.equal(acquisition.reason, 'invalid_record', `${item.case_id} reason`);
    assert.equal(acquisition.snapshot, undefined, `${item.case_id} published a snapshot`);
    // The leading valid record proves the refusal came from the later record, and
    // nothing before it may survive as a partial authority.
    const reconciler = new ProjectionReconciler(profile);
    reconciler.acquire(item.handoff);
    assert.equal(reconciler.acquiredSnapshot('normal'), undefined, `${item.case_id} retained a partial snapshot`);
    assert.equal(snapshotView(reconciler, 'normal').retained_authoritative, false, `${item.case_id} retained authority`);
    assert.equal(reconciler.livePartition('normal').records.size, 0, `${item.case_id} wrote the live map`);
  }
  // A snapshot of another profile contradicts the bound partition instead of merely
  // lacking a field, so it maps onto the non-retryable protocol code.
  const foreign = cases.find((item) => item.case_id === 'snapshot_for_unbound_profile');
  assert.ok(foreign, 'the corpus declares no unbound-profile case');
  assert.equal(foreign.acquisition, 'PROFILE_MISMATCH');
  assert.equal(acquireSnapshot(profile, foreign.handoff).outcome, 'PROFILE_MISMATCH');
  assert.ok(
    cases.filter((item) => item.acquisition === 'SNAPSHOT_REQUIRED').every((item) => item.expected.producer.outcome !== 'PROFILE_MISMATCH'),
    'an incomplete read must stay retryable',
  );
});

test('IP-05-T02 refuses a valid read one record past the IP-02/IP-07 bound without replacing or clearing the acquired snapshot', () => {
  // The bound comes from the corpus, not from a number restated here: the artifact
  // declares the single limit IP-02 acceptSnapshot and the IP-07 snapshot_records
  // limit share, and the real producer bound has to be that same value.
  const bound = authority.max_snapshot_records;
  assert.equal(bound, 10000, 'the declared IP-02/IP-07 snapshot record bound');
  assert.equal(PROJECTION_MAX_SNAPSHOT_RECORDS, bound, 'the producer enforces a different bound than the corpus declares');
  assert.ok(
    includes(authority.reject_reason_vocabulary, 'snapshot_bounds_exceeded'),
    'the declared vocabulary drops the size-gate refusal reason',
  );
  const bounded = authority.required_conditions.find((condition) => condition.id === 'bounded_record_count');
  assert.ok(bounded && bounded.rule.length > 0, 'the corpus states no bounded record count clause');

  const overLimit = cases.find((item) => item.case_id === 'snapshot_over_record_limit');
  const populated = cases.find((item) => item.case_id === 'complete_populated_snapshot');
  assert.ok(overLimit && populated, 'the corpus declares no over-limit or completed read');
  // The payload is a bounded expansion rather than a serialized body, so the artifact
  // stays reviewable while the real acquisition still receives the declared count.
  assert.equal(recordsOf(overLimit), null, 'the over-limit case serializes its record list');
  const records = materializedRecords(overLimit);
  assert.equal(records.length, overLimit.record_expansion.count, 'the materialized record count');
  assert.equal(records.length, bound + 1, 'the over-limit read must sit exactly one record past the bound');
  // The expansion is an arithmetic identity sequence, so the list is deterministic and
  // every record is its own tab rather than a repetition of one identity.
  assert.equal(records[0].tab_identity.tab_id, overLimit.record_expansion.identity_start, 'the first identity');
  assert.equal(records[bound].tab_identity.tab_id, overLimit.record_expansion.identity_start + bound * overLimit.record_expansion.identity_step, 'the identity one record past the bound');
  assert.equal(
    new Set(records.map((record) => record.tab_identity.tab_id)).size,
    records.length,
    'the expanded list repeats an identity',
  );

  // The payload is a valid snapshot: every record clears the real IP-02 record
  // contract against the real fence, so nothing but its length can refuse it. The
  // same records truncated to the bound are authority, which is what makes the length
  // the single difference between the two reads.
  const atBound = { ...overLimit.handoff, records: records.slice(0, bound) };
  const justOver = { ...overLimit.handoff, records };
  const admitted = acquireSnapshot(profile, atBound);
  assert.equal(admitted.ok, true, `a valid read of ${bound} records must be authority`);
  assert.equal(admitted.snapshot.record_count, bound, 'the at-bound record count');
  assert.equal(admitted.snapshot.explicitly_empty, false, 'the at-bound read is not empty');
  const fence = admitted.snapshot.fence;
  assert.equal(
    records.filter((record) => readProjectionRecord(record, fence).ok).length,
    records.length,
    'the over-limit payload must be valid record by record, so only its length refuses it',
  );

  const reconciler = new ProjectionReconciler(profile);
  assert.equal(reconciler.acquire(populated.handoff).ok, true, 'the completed read is authority');
  const retained = reconciler.acquiredSnapshot(overLimit.partition_context);
  assert.equal(retained.record_count, observationOf('authoritative_record_count'), 'the completed read retained its records');
  const viewBefore = snapshotView(reconciler, overLimit.partition_context);
  const privateBefore = snapshotView(reconciler, 'private');
  assert.equal(viewBefore.retained_authoritative, true, 'the completed read is retained authority');

  const refusal = reconciler.acquire(justOver);
  assert.equal(refusal.ok, false, 'a valid read one record past the bound was treated as authority');
  assert.equal(refusal.authoritative, false, 'the over-limit read reported authority');
  assert.equal(refusal.reason, overLimit.expected.producer.reason, 'the over-limit refusal reason');
  assert.equal(refusal.reason, 'snapshot_bounds_exceeded', 'only the size gate may refuse an all-valid payload');
  assert.equal(refusal.outcome, overLimit.expected.producer.outcome, 'the over-limit refusal outcome');
  assert.equal(refusal.resync_required, true, 'the refusal must require a fresh authoritative read');
  assert.equal(refusal.snapshot, undefined, 'the refusal published a snapshot');
  assert.equal(refusal.fence.context_kind, overLimit.partition_context, 'the refusal names its partition');
  assert.equal(refusal.fence.projection_epoch, overLimit.handoff.projection_epoch, 'the refusal forwards the declared fence');

  // Neither replaced nor cleared: the retained read is still the earlier one, its count
  // is the acquired count rather than the oversized one or zero, and it is never
  // reported as a converged empty projection.
  assert.deepEqual(
    JSON.parse(JSON.stringify(reconciler.acquiredSnapshot(overLimit.partition_context))),
    JSON.parse(JSON.stringify(retained)),
    'the over-limit read replaced or cleared the acquired snapshot',
  );
  const viewAfter = snapshotView(reconciler, overLimit.partition_context);
  assert.equal(viewAfter.retained_authoritative, true, 'the over-limit read dropped acquired authority');
  assert.equal(viewAfter.record_count, observationOf('authoritative_record_count'), 'the retained record count moved');
  assert.equal(viewAfter.explicitly_empty, false, 'the refusal reported an empty projection it never read');
  assert.equal(viewAfter.epoch, viewBefore.epoch, 'the retained epoch moved');
  assert.equal(viewAfter.projection_revision, viewBefore.projection_revision, 'the retained revision moved');
  assert.equal(viewAfter.event_sequence, viewBefore.event_sequence, 'the retained sequence moved');
  // The newest attempt stays observable next to the data it did not replace.
  assert.equal(viewAfter.last_outcome, 'SNAPSHOT_REQUIRED', 'the over-limit read must still be observable');
  assert.deepEqual(snapshotView(reconciler, 'private'), privateBefore, 'a refused normal read moved the private partition');
  assert.equal(reconciler.acquiredSnapshot('private'), undefined, 'the refusal created authority for another partition');

  const diagnostics = reconciler.view().diagnostics;
  assert.equal(diagnostics.snapshots_authoritative, 1, 'exactly one of the two reads is authority');
  assert.equal(diagnostics.snapshots_unauthoritative, 1, 'the over-limit read is the only refusal');
  assert.equal(diagnostics.snapshot_refusals_by_reason.snapshot_bounds_exceeded, 1, 'the size-gate refusal counter');
  assert.equal(observationOf('non_authoritative_record_count'), 0, 'a non-authoritative acquisition publishes no record');
  assert.equal(observationOf('partial_outputs'), 0, 'a refused read publishes no partial set');
  assert.equal(reconciler.livePartition(overLimit.partition_context).records.size, 0, 'the over-limit read wrote the live map');
  assert.equal(reconciler.livePartition(overLimit.partition_context).epoch, null, 'the over-limit read published a live epoch');

  // The refusal is the only value the runtime may report, so none of the oversized
  // payload's own text may appear in it.
  const rendered = JSON.stringify(refusal);
  for (const value of declaredRecordValues(overLimit)) {
    assert.ok(!rendered.includes(value), 'the over-limit refusal leaks a declared record value');
  }

  // Ordering: the size gate runs before the record walk. A payload that is over the
  // bound and unusable at its very last record still reports the size gate, because
  // the walk that would have refused that record never runs. The truncated payload is
  // authority and the poisoned record is genuinely unusable, which is what makes the
  // walk's verdict the only alternative a reordering could have produced.
  const unusable = { ...records.at(-1), projection_epoch: corpus.input.foreign_epoch };
  assert.equal(readProjectionRecord(unusable, fence).ok, false, 'the poisoned record must be one the record walk refuses');
  const poisoned = [...records.slice(0, bound), unusable];
  assert.equal(poisoned.length, bound + 1, 'the poisoned payload must stay one record past the bound');
  assert.equal(acquireSnapshot(profile, { ...overLimit.handoff, records: poisoned.slice(0, bound) }).ok, true, 'the truncated poisoned payload is valid');
  const walked = acquireSnapshot(profile, { ...overLimit.handoff, records: poisoned });
  assert.equal(walked.ok, false, 'the poisoned over-bound read was treated as authority');
  assert.equal(walked.reason, 'snapshot_bounds_exceeded', 'the record walk ran before the size gate');
  assert.equal(walked.snapshot, undefined, 'the poisoned refusal published a snapshot');
});
