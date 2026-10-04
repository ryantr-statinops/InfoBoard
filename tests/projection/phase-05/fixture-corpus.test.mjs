// The extension-side consumer of the shared IP-05-T01 projection boundary
// corpus. Every case is driven through the real producer boundary in
// extension/src/projection/reconciler.ts: a handoff that cannot state profile_id,
// context_kind, projection_epoch, and projection_revision is refused with a
// bounded reason, and an admitted submission forwards the IP-04 allocated fence
// and record values unchanged. Snapshot authority, canonical ordering, revision
// sequencing, delta reduction, commit, and recovery are later IP-05 tasks and are
// asserted nowhere here: whether an admitted record may enter the projection, and
// which revision lineage it belongs to, are IP-02's and IP-04's decisions. The
// declared lineage and identity-count evidence of the artifact is IP-02 reducer
// output, so the Go host consumer and the fixture runner check it.
import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile, stat } from 'node:fs/promises';

import { parseProfileID } from '../../../extension/dist/domain/index.js';
import {
  admitHandoff,
  ProjectionReconciler,
  PROJECTION_REJECT_REASONS,
} from '../../../extension/dist/src/projection/reconciler.js';

const ARTIFACT = 'fixtures/projection/phase-05/phase-05.json';
const CONTRACT_SOURCE = [
  'extension/domain/index.ts',
  'extension/src/projection/reconciler.ts',
  'host/internal/domain/domain.go',
  'host/internal/projection/reconciler.go',
];
const REQUIRED_IDS = [
  'FX-PROJECTION-BOUNDARY-HANDOFF-IDENTITY',
  'FX-PROJECTION-BOUNDARY-REQUIRED-FIELDS',
  'FX-PROJECTION-BOUNDARY-SYNTHETIC-IDENTITY',
];
const STAGE_BOUNDARY = 'boundary';
const STAGE_PROJECTION = 'projection';
const APPLY_RECORD = 'record';
const APPLY_NONE = 'none';
const CONTEXTS = ['normal', 'private'];

// The event sequence IP-04 allocates on a handoff. A T01 record payload states no
// sequence of its own, so this consumer declares the first sequence of the epoch;
// the four mandatory identity fields still come from the fixture alone.
const ALLOCATED_SEQUENCE = 1;

// The bounded reason each declared missing field produces. IP-05-T01 refuses for
// no other reason, so an artifact cause maps onto exactly one of them: the fence
// fields name themselves, and a record whose tab identity cannot be stated is
// refused as a record the fence cannot admit.
const MISSING_FIELD_REASONS = new Map([
  ['missing_profile_id', 'missing_profile_id'],
  ['missing_context_kind', 'missing_context_kind'],
  ['missing_projection_epoch', 'missing_projection_epoch'],
  ['missing_projection_revision', 'missing_projection_revision'],
  ['missing_tab_identity_profile_id', 'invalid_record'],
  ['missing_tab_identity_context_kind', 'invalid_record'],
  ['missing_tab_identity_tab_id', 'invalid_record'],
]);

const corpus = JSON.parse(await readFile(ARTIFACT, 'utf8'));
const boundary = corpus.boundary;
const fixtures = new Map(corpus.fixtures.map((fixture) => [fixture.fixture_id, fixture]));

function includes(values, value) {
  return values.includes(value);
}

// handoffFor presents one fixture case as the IP-04 observer handoff the producer
// receives. A record payload case becomes a snapshot handoff whose fence is the
// record's own mandatory fields; an IP-04 handoff case keeps the fields it states.
// previous_revision and event_sequence are optional handoff fields the artifact
// lists and the producer requires, so this consumer states them from the case
// instead of letting every case fail on a field it never claims to declare.
function handoffFor(item) {
  if (item.handoff) {
    return {
      ...item.handoff,
      previous_revision: item.handoff.previous_revision ?? 0,
      event_sequence: item.handoff.event_sequence ?? ALLOCATED_SEQUENCE,
    };
  }
  const payload = item.payload;
  return {
    kind: 'snapshot',
    profile_id: payload.profile_id,
    context_kind: payload.context_kind,
    projection_epoch: payload.projection_epoch,
    previous_revision: payload.previous_revision ?? 0,
    projection_revision: payload.projection_revision,
    event_sequence: ALLOCATED_SEQUENCE,
    records: [payload],
  };
}

// assertForwarded proves an admitted submission was forwarded, not repaired: every
// IP-04 allocated fence value and every record field is carried through exactly as
// declared, so no identity, epoch, revision, or sequence was invented.
function assertForwarded(handoff, admission) {
  const { submission } = admission;
  const { fence } = submission;
  assert.equal(submission.kind, handoff.kind, 'submission kind');
  assert.equal(fence.profile_id, handoff.profile_id, 'profile_id was rewritten');
  assert.equal(fence.context_kind, handoff.context_kind, 'context_kind was rewritten');
  assert.equal(fence.projection_epoch, handoff.projection_epoch, 'projection_epoch was rewritten');
  assert.equal(fence.previous_revision, handoff.previous_revision, 'previous_revision was rewritten');
  assert.equal(fence.projection_revision, handoff.projection_revision, 'projection_revision was rewritten');
  assert.equal(fence.event_sequence, handoff.event_sequence, 'event_sequence was rewritten');
  assert.deepEqual(submission.records, handoff.records ?? [], 'snapshot records were rewritten');
  assert.deepEqual(submission.record, handoff.record, 'the record was rewritten');
  assert.equal(submission.resync_required, handoff.resync_required === true);
  assert.equal(submission.resync_reason, handoff.resync_reason);
}

function assertRefused(handoff, admission) {
  assert.equal(admission.ok, false, `${handoff.kind} was admitted without a complete identity`);
  assert.ok(PROJECTION_REJECT_REASONS.includes(admission.reason), `reason ${admission.reason} is outside the closed vocabulary`);
}

// drive runs every case of one fixture through the real boundary once and returns
// each case with the handoff it presented and the admission it received.
function drive(fixture) {
  const profile = parseProfileID(fixture.input.profile_id);
  assert.ok(CONTEXTS.includes(fixture.input.context_kind), `${fixture.fixture_id} declares an unknown context`);
  return {
    profile,
    admissions: fixture.input.cases.map((item) => {
      const handoff = handoffFor(item);
      return { item, handoff, admission: admitHandoff(profile, handoff) };
    }),
  };
}

// syntheticCase blanks every mandatory partition and fence value of a case and
// substitutes an invalid numeric tab id, so the artifact's declared
// synthetic-fallback probe reaches the real boundary instead of a copy of its rules.
function syntheticCase(item) {
  const partition = new Set(boundary.mandatory_partition_fields);
  const fence = new Set(boundary.mandatory_fence_fields);
  const blank = (value) => {
    if (Array.isArray(value)) return value.map(blank);
    if (typeof value !== 'object' || value === null) return value;
    return Object.fromEntries(Object.entries(value).map(([key, entry]) => {
      if (key === 'tab_id') return [key, -1];
      if (partition.has(key) || fence.has(key)) return [key, typeof entry === 'string' ? '' : 0];
      return [key, blank(entry)];
    }));
  };
  return {
    ...item,
    handoff: item.handoff ? blank(item.handoff) : undefined,
    payload: item.payload ? blank(item.payload) : undefined,
  };
}

test('IP-05-T01 boundary corpus names every mandatory field and both consumers', async () => {
  assert.equal(corpus.schema_version, 1);
  assert.equal(corpus.phase, 'IP-05');
  assert.equal(corpus.artifact, 'projection-boundary');
  assert.equal(corpus.owner_task, 'IP-05-T01');
  assert.deepEqual(corpus.shared_consumers, ['go', 'node']);
  for (const source of CONTRACT_SOURCE) {
    assert.ok((await stat(source)).isFile(), `contract source ${source} is absent`);
  }
  assert.deepEqual([...fixtures.keys()].sort(), REQUIRED_IDS);
  for (const id of corpus.reserved_fixture_ids) {
    assert.equal(fixtures.has(id), false, `${id} is reserved for a later IP-05 task`);
  }
  assert.equal(boundary.synthetic_identity_fallback_allowed, false);
  const mandatory = [...boundary.mandatory_partition_fields, ...boundary.mandatory_fence_fields];
  assert.equal(new Set(mandatory).size, mandatory.length);
  for (const field of mandatory) {
    assert.ok(includes(boundary.snapshot_record_required_fields, field), `${field} is absent from the record requirement set`);
    assert.ok(boundary.identity_sources[field], `${field} declares no identity source`);
  }
  assert.equal(Object.keys(boundary.identity_sources).length, mandatory.length);
  for (const field of boundary.mandatory_partition_fields) {
    assert.ok(includes(boundary.tab_identity_required_fields, field), `tab identity does not require ${field}`);
  }
  const handoff = boundary.ip04_handoff_boundary;
  assert.equal(handoff.owner, 'IP-04');
  assert.equal(handoff.consumer, 'IP-05');
  for (const kind of ['snapshot', 'delta']) {
    for (const field of mandatory) {
      assert.ok(includes(handoff.required_handoff_fields[kind], field), `${kind} handoff does not require ${field}`);
    }
  }
  assert.deepEqual(handoff.required_handoff_fields.status, ['kind']);
  for (const kind of handoff.handoff_kinds) {
    const mapped = handoff.handoff_kind_map[kind];
    assert.ok(includes(handoff.message_kinds, mapped), `handoff kind ${kind} maps to an undeclared message kind`);
  }
  for (const kind of handoff.message_kinds) {
    assert.ok(handoff.required_handoff_fields[kind]?.length > 0, `message kind ${kind} declares no required handoff field`);
  }
  for (const fixture of corpus.fixtures) {
    assert.equal(fixture.owner_task, corpus.owner_task);
    assert.ok(fixture.requirement_ids.length > 0);
    assert.ok(fixture.title.length > 0);
    assert.ok(fixture.expected.privacy_assertions.length > 0);
    assert.ok(fixture.input.cases.length > 0);
  }
});

test('IP-05-T01 refuses every record that omits a mandatory identity or fence field', () => {
  const fixture = fixtures.get('FX-PROJECTION-BOUNDARY-REQUIRED-FIELDS');
  const { admissions } = drive(fixture);
  const refused = admissions.filter(({ item }) => item.stage === STAGE_BOUNDARY);
  assert.equal(refused.length, fixture.input.cases.length - 1, 'every case but the complete record must be refused');
  const covered = new Set();
  for (const { item, handoff, admission } of refused) {
    assertRefused(handoff, admission);
    const reason = MISSING_FIELD_REASONS.get(item.cause);
    assert.ok(reason, `${item.case_id} cause ${item.cause} is not a missing mandatory field`);
    assert.equal(admission.reason, reason, `${item.case_id} reason`);
    assert.equal(item.expected_outcome, boundary.fail_closed_outcome);
    assert.equal(item.apply, APPLY_NONE, `${item.case_id} must not reach the projection stage`);
    assert.equal(item.synthetic_fallback_probe, true, `${item.case_id} must probe the synthetic fallback`);
    covered.add(item.cause);
  }
  // Every mandatory partition, fence, and tab identity field has a case that proves
  // the producer refuses a record which cannot state it.
  for (const field of boundary.mandatory_partition_fields) {
    assert.ok(covered.has(`missing_${field}`), `the fixture proves no refusal for ${field}`);
  }
  for (const field of boundary.mandatory_fence_fields) {
    assert.ok(covered.has(`missing_${field}`), `the fixture proves no refusal for ${field}`);
  }
  for (const field of boundary.tab_identity_required_fields) {
    assert.ok(covered.has(`missing_tab_identity_${field}`), `the fixture proves no refusal for tab identity ${field}`);
  }
});

test('IP-05-T01 forwards the IP-04 allocated fence and record values unchanged', () => {
  for (const id of REQUIRED_IDS) {
    const fixture = fixtures.get(id);
    const { profile, admissions } = drive(fixture);
    for (const { item, handoff, admission } of admissions) {
      assert.equal(item.apply, item.stage === STAGE_PROJECTION ? APPLY_RECORD : APPLY_NONE, `${item.case_id} apply`);
      if (item.stage === STAGE_BOUNDARY) {
        assertRefused(handoff, admission);
        assert.equal(admission.reason, MISSING_FIELD_REASONS.get(item.cause), `${item.case_id} reason`);
        continue;
      }
      assert.equal(item.cause, null, `${item.case_id} must not declare a boundary cause`);
      if (admission.ok) {
        // An admitted fence always names the bound profile and a real context, so
        // no submission can be partitioned into an identity the peer never sent.
        assert.equal(admission.submission.fence.profile_id, profile);
        assert.ok(CONTEXTS.includes(admission.submission.fence.context_kind));
        assertForwarded(handoff, admission);
        continue;
      }
      assertRefused(handoff, admission);
      assert.ok(
        !includes(boundary.accepted_outcomes, item.expected_outcome),
        `${fixture.fixture_id}/${item.case_id} is declared usable but was refused with ${admission.reason}`,
      );
    }
  }
});

test('IP-05-T01 refuses a placeholder, blank, or zero identity instead of substituting one', () => {
  const fixture = fixtures.get('FX-PROJECTION-BOUNDARY-SYNTHETIC-IDENTITY');
  const { profile, admissions } = drive(fixture);
  const reasons = new Set();
  for (const { item, handoff, admission } of admissions) {
    if (admission.ok) {
      // Whatever the handoff declared is forwarded unchanged, including a revision
      // of zero: a declared value is never replaced by a default, and whether a
      // revision-zero lineage may commit is IP-05-T04's decision, so this boundary
      // neither repairs the value nor refuses it here.
      assertForwarded(handoff, admission);
      continue;
    }
    assertRefused(handoff, admission);
    reasons.add(admission.reason);
  }
  // Every identity defect class this fixture declares reaches the producer as a
  // bounded reason naming that field. A blank tab id is the required-fields
  // fixture's case; the probe below blanks every mandatory value at once.
  for (const reason of ['invalid_profile_id', 'invalid_context_kind', 'invalid_projection_epoch', 'invalid_record', 'missing_profile_id']) {
    assert.ok(reasons.has(reason), `the fixture proves no refusal with reason ${reason}`);
  }
  // The declared probe must be refused as well, so a zero-valued identity never
  // completes a boundary failure.
  for (const item of fixture.input.cases.filter((entry) => entry.synthetic_fallback_probe)) {
    const probe = handoffFor(syntheticCase(item));
    assertRefused(probe, admitHandoff(profile, probe));
  }
  // The record the artifact declares usable must clear the boundary and carry the
  // allocated identity verbatim; otherwise it would be the fallback the rest refuse.
  const usable = admissions.filter(({ item }) => includes(boundary.accepted_outcomes, item.expected_outcome));
  assert.equal(usable.length, 1, 'the fixture declares exactly one usable record');
  assert.ok(usable[0].admission.ok, `the usable record was refused with ${usable[0].admission.reason}`);
  const payload = usable[0].item.payload;
  assert.equal(payload.profile_id, fixture.input.profile_id);
  assert.equal(payload.context_kind, fixture.input.context_kind);
  assert.equal(payload.projection_epoch, fixture.input.projection_epoch);
});

test('IP-05-T01 accepts an IP-04 handoff only when it states its allocated identity', () => {
  const fixture = fixtures.get('FX-PROJECTION-BOUNDARY-HANDOFF-IDENTITY');
  const { admissions } = drive(fixture);
  const mapped = boundary.ip04_handoff_boundary.handoff_kind_map;
  for (const { item, handoff, admission } of admissions) {
    assert.equal(item.message_kind, mapped[handoff.kind], `${item.case_id} message kind`);
    if (item.stage === STAGE_PROJECTION) {
      assertForwarded(handoff, admission);
      for (const field of boundary.mandatory_partition_fields) {
        assert.equal(handoff[field], fixture.input[field], `${item.case_id} handoff ${field}`);
      }
      for (const field of boundary.mandatory_fence_fields) {
        assert.ok(Object.hasOwn(handoff, field), `${item.case_id} must state ${field}`);
      }
      assert.ok(Array.isArray(handoff.records), `${item.case_id} must carry its record payload`);
      continue;
    }
    assertRefused(handoff, admission);
    assert.equal(admission.reason, MISSING_FIELD_REASONS.get(item.cause), `${item.case_id} reason`);
    assert.equal(item.expected_outcome, boundary.fail_closed_outcome, `${item.case_id} outcome`);
  }
});

test('IP-05-T01 producer keeps one observed fence per bound partition and publishes no lineage', () => {
  for (const id of REQUIRED_IDS) {
    const fixture = fixtures.get(id);
    const profile = parseProfileID(fixture.input.profile_id);
    const reconciler = new ProjectionReconciler(profile);
    assert.deepEqual(reconciler.view().partitions.map((entry) => entry.context_kind), CONTEXTS);
    let admitted = 0;
    let refused = 0;
    let observedFence = null;
    for (const item of fixture.input.cases) {
      const handoff = handoffFor(item);
      const admission = reconciler.admit(handoff);
      if (admission.ok) {
        admitted += 1;
        observedFence = handoff;
        assertForwarded(handoff, admission);
        continue;
      }
      refused += 1;
      assert.ok(PROJECTION_REJECT_REASONS.includes(admission.reason), `${item.case_id} reason ${admission.reason}`);
    }
    const view = reconciler.view();
    assert.equal(view.profile_id, profile);
    assert.equal(view.diagnostics.handoffs_admitted, admitted, `${fixture.fixture_id} admitted`);
    assert.equal(view.diagnostics.handoffs_rejected, refused, `${fixture.fixture_id} refused`);
    assert.equal(
      Object.values(view.diagnostics.rejections_by_reason).reduce((total, value) => total + value, 0),
      refused,
      `${fixture.fixture_id} rejection counters`,
    );
    // The observed fence of a partition is the last handoff that stated it: never a
    // substituted value and never a second counter of the producer's own.
    for (const partition of view.partitions) {
      const observed = observedFence?.context_kind === partition.context_kind ? observedFence : null;
      assert.equal(partition.last_observed_epoch, observed?.projection_epoch ?? null, `${fixture.fixture_id} ${partition.context_kind} epoch`);
      assert.equal(partition.last_observed_revision, observed?.projection_revision ?? null, `${fixture.fixture_id} ${partition.context_kind} revision`);
      assert.equal(partition.last_observed_sequence, observed?.event_sequence ?? null, `${fixture.fixture_id} ${partition.context_kind} sequence`);
      // Admitting a handoff is not snapshot authority, so no record is acquired and
      // no lineage is published here.
      assert.equal(partition.record_count, 0, `${fixture.fixture_id} ${partition.context_kind} records`);
      const live = reconciler.livePartition(partition.context_kind);
      assert.equal(live.records.size, 0, `${fixture.fixture_id} ${partition.context_kind} live records`);
      assert.equal(live.epoch, null, `${fixture.fixture_id} ${partition.context_kind} live epoch`);
      assert.equal(Number(live.revision), 0, `${fixture.fixture_id} ${partition.context_kind} live revision`);
    }
  }
});