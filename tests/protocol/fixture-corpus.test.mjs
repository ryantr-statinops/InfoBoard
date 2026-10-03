// The extension-side consumer of the shared IP-07 fixture corpus. This file
// validates the deterministic corpus itself; the client seam contract lives in
// native-messaging-client.test.mjs and consumes the same artifact.
import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';

const ARTIFACT = 'fixtures/protocol/phase-07.json';
const REQUIRED_IDS = Array.from({ length: 13 }, (_, index) => `NM-PROTO-${String(index + 1).padStart(3, '0')}`);
const WIRE_ERROR_CODES = [
  'INVALID_FRAME', 'PROTOCOL_MISMATCH', 'PROFILE_MISMATCH', 'PAYLOAD_LIMIT',
  'SNAPSHOT_REQUIRED', 'REVISION_MISMATCH', 'INDEX_REBUILDING', 'QUERY_TIMEOUT',
  'PERSISTENCE_DEGRADED', 'HOST_SHUTDOWN', 'INTERNAL_FAILURE',
];
const MESSAGE_CONTRACT = {
  hello: 'extension_to_host',
  hello_ack: 'host_to_extension',
  snapshot: 'extension_to_host',
  delta: 'extension_to_host',
  sync_ack: 'host_to_extension',
  resync_required: 'host_to_extension',
  query: 'extension_to_host',
  query_result: 'host_to_extension',
  activation_observed: 'extension_to_host',
  activation_failed: 'extension_to_host',
  activation_ack: 'host_to_extension',
  health: 'extension_to_host',
  health_result: 'host_to_extension',
  error: 'host_to_extension',
};
const DELTA_OPERATIONS = ['create', 'update', 'move', 'group', 'pin', 'activate', 'remove'];
const UUID_V4 = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
const SNAKE_CASE = /^[a-z][a-z0-9_]*$/;

const corpus = JSON.parse(await readFile(ARTIFACT, 'utf8'));

function encodeFrame(payload) {
  const header = Buffer.alloc(4);
  header.writeUInt32LE(payload.length, 0);
  return Buffer.concat([header, payload]);
}

// The same declared rules the Go consumer applies: exact payload padding,
// generated record and operation counts, and exact query scalar counts.
function materialize(spec) {
  if (typeof spec.raw_payload_text === 'string') {
    return { bytes: encodeFrame(Buffer.from(spec.raw_payload_text, 'utf8')), payloadBytes: Buffer.byteLength(spec.raw_payload_text, 'utf8') };
  }
  const object = { ...(spec.payload ?? {}) };
  if (spec.payload_padding) {
    // The filler must be one ASCII byte so the encoded length stays monotonic,
    // which lets a binary search hit the declared byte count exactly.
    const filler = spec.payload_padding.filler || 'x';
    if (filler.length !== 1 || filler.charCodeAt(0) > 0x7f) {
      throw new Error('fixture padding filler must be one ASCII byte');
    }
    const target = spec.payload_padding.total_json_payload_bytes;
    const sizeAt = (size) => Buffer.byteLength(JSON.stringify({ ...object, [spec.payload_padding.field]: filler.repeat(size) }), 'utf8');
    if (sizeAt(0) > target) {
      throw new Error('unpadded fixture payload already exceeds the declared padding target');
    }
    let low = 0;
    let high = target + 1;
    while (low < high) {
      const middle = Math.floor((low + high) / 2);
      if (sizeAt(middle) < target) {
        low = middle + 1;
      } else {
        high = middle;
      }
    }
    object[spec.payload_padding.field] = filler.repeat(low);
  }
  if (typeof spec.query_scalars === 'number') {
    object.query = 'a'.repeat(spec.query_scalars);
  }
  if (typeof spec.repeat_record_count === 'number' && spec.repeat_record_count > 0) {
    object.tabs = Array.from({ length: spec.repeat_record_count }, (_, index) => ({
      tab_id: index, window_id: 1, title_display: `T${index}`,
      url_display: `t${index}.example`, domain_display: `t${index}.example`,
      pinned: false, active: false, group_id: null, context_kind: 'normal',
    }));
  }
  if (typeof spec.repeat_operation_count === 'number' && spec.repeat_operation_count > 0) {
    object.operations = Array.from({ length: spec.repeat_operation_count }, (_, index) => ({ operation: 'remove', tab_id: index }));
  }
  const payloadBytes = Buffer.byteLength(JSON.stringify(object), 'utf8');
  const envelope = {
    protocol: spec.protocol ?? corpus.envelope.protocol_major,
    type: spec.type,
    request_id: spec.request_id ?? '',
    profile_id: spec.profile_id,
    projection_revision: spec.projection_revision ?? corpus.envelope.projection_revision_min,
    payload: object,
  };
  const encoded = Buffer.from(JSON.stringify(envelope), 'utf8');
  return { bytes: encodeFrame(encoded), payloadBytes };
}

function replay(fixture) {
  return fixture.input_frames.flatMap((spec) => {
    const wire = materialize(spec);
    const repeats = Math.max(1, spec.repeat_count ?? 1);
    return Array.from({ length: repeats }, () => ({ spec, ...wire }));
  });
}

test('IP-07 corpus declares NM-PROTO-001..013 exactly once', () => {
  assert.equal(corpus.schema_version, 1);
  assert.equal(corpus.phase, 'IP-07');
  assert.equal(corpus.artifact, 'native-messaging-protocol');
  assert.equal(corpus.contract_source, 'host/internal/protocol');
  assert.deepEqual(corpus.shared_consumers, ['go', 'node']);
  assert.deepEqual(corpus.fixtures.map((fixture) => fixture.fixture_id), REQUIRED_IDS);
  for (const fixture of corpus.fixtures) {
    assert.ok(fixture.title.length > 0, `${fixture.fixture_id} has no title`);
    assert.ok(fixture.requirement_ids.length > 0, `${fixture.fixture_id} binds no requirement`);
  }
});

test('frame ownership stays with IP-06 and both boundaries are declared', () => {
  assert.equal(corpus.frame_encoding.owner, 'IP-06');
  assert.equal(corpus.frame_encoding.length_prefix_bytes, 4);
  assert.equal(corpus.frame_encoding.byte_order, 'little_endian');
  assert.equal(corpus.transport_boundary.max_frame_bytes, 1048576);
  assert.equal(corpus.transport_boundary.max_json_payload_bytes, 262144);
  assert.ok(corpus.transport_boundary.max_json_payload_bytes < corpus.transport_boundary.max_frame_bytes);
  assert.equal(corpus.transport_boundary.payload_limit_enforced_before_decode, true);
  assert.equal(corpus.transport_boundary.over_frame_boundary_owner, 'IP-06');
});

test('the envelope contract is the strict six-field shape', () => {
  assert.deepEqual(corpus.envelope.required_fields, ['protocol', 'type', 'request_id', 'profile_id', 'projection_revision', 'payload']);
  assert.equal(corpus.envelope.unknown_fields_rejected, true);
  assert.equal(corpus.envelope.duplicate_keys_rejected, true);
  assert.equal(corpus.envelope.trailing_bytes_rejected, true);
  assert.equal(corpus.envelope.protocol_major, 1);
  assert.equal(corpus.envelope.request_id_max_bytes, 128);
  assert.equal(corpus.envelope.profile_id_max_bytes, 128);
  assert.equal(corpus.envelope.projection_revision_min, 0);
  assert.equal(corpus.envelope.profile_identity_source, 'envelope_profile_id');
});

test('limits and deadlines are bounded and resolve by name', () => {
  const limits = corpus.limits;
  for (const [name, value] of Object.entries(limits)) {
    assert.ok(value > 0, `limit ${name} must be positive`);
  }
  assert.equal(limits.query_max_scalars, 512);
  assert.equal(limits.snapshot_max_records, 10000);
  assert.equal(limits.delta_max_operations, 1000);
  assert.equal(limits.query_max_results, 50);
  assert.equal(limits.title_max_scalars, 512);
  assert.equal(limits.domain_max_bytes, 255);
  assert.equal(limits.url_max_bytes, 2048);
  assert.equal(limits.capabilities_max_count, 32);
  assert.equal(limits.message_type_max_bytes, 64);

  const deadlines = corpus.deadlines_ms;
  assert.deepEqual(Object.keys(deadlines).sort(), [
    'activation_status', 'delta_apply', 'frame_read', 'health', 'hello',
    'query_host_budget', 'query_request_budget', 'shutdown_drain', 'snapshot_apply',
  ]);
  for (const [name, value] of Object.entries(deadlines)) {
    assert.ok(value > 0 && value <= 300000, `deadline ${name} must be bounded`);
  }
  assert.ok(deadlines.query_host_budget < deadlines.query_request_budget);
  assert.ok(deadlines.snapshot_apply < deadlines.hello);
});

test('the message contract covers every type and direction', () => {
  const declared = new Map(corpus.message_types.map((entry) => [entry.type, entry]));
  assert.equal(declared.size, Object.keys(MESSAGE_CONTRACT).length);
  for (const [type, direction] of Object.entries(MESSAGE_CONTRACT)) {
    assert.ok(declared.has(type), `${type} is absent from the contract`);
    assert.equal(declared.get(type).direction, direction);
    for (const field of declared.get(type).payload_fields) {
      assert.match(field, SNAKE_CASE);
    }
  }
  for (const field of ['code', 'retryable', 'message_key']) {
    assert.ok(declared.get('error').payload_fields.includes(field), `error must require ${field}`);
  }
  assert.deepEqual(corpus.error_optional_fields.sort(), ['expected_revision', 'expected_sequence', 'retry_after_ms']);
  assert.deepEqual(corpus.delta_operations, DELTA_OPERATIONS);
});

test('error, resync, and state vocabularies match the binding contract', () => {
  assert.deepEqual([...corpus.error_codes].sort(), [...WIRE_ERROR_CODES].sort());
  for (const reason of corpus.resync_reasons) {
    assert.match(reason, SNAKE_CASE);
  }
  for (const state of ['unbound', 'bound', 'ready', 'degraded', 'incompatible', 'disconnected', 'shutdown']) {
    assert.ok(corpus.session_states.includes(state), `session state ${state} is absent`);
  }
  for (const state of ['Ready', 'Degraded', 'Reconnecting', 'Incompatible', 'Shutdown']) {
    assert.ok(corpus.client_states.includes(state), `client state ${state} is absent`);
  }
});

test('fixture identities stay opaque and distinct', () => {
  const identity = corpus.identity;
  assert.match(identity.profile_id, UUID_V4);
  assert.match(identity.other_profile_id, UUID_V4);
  assert.notEqual(identity.profile_id, identity.other_profile_id);
  assert.match(identity.projection_epoch, UUID_V4);
  assert.notEqual(identity.projection_epoch, identity.second_projection_epoch);
  assert.ok(identity.host_version.length > 0 && identity.extension_version.length > 0);
  assert.ok(identity.ranking_model_version.length > 0);
  assert.equal(identity.context_kind, 'normal');
  assert.ok(identity.capabilities.length <= corpus.limits.capabilities_max_count);
  assert.deepEqual(identity.supported_protocol_versions, [corpus.envelope.protocol_major]);
  assert.ok(identity.replacement_tab_ids.length > identity.tab_ids.length);
});

test('every fixture declares inputs, pre-state, and bounded observables', () => {
  const known = new Set(WIRE_ERROR_CODES);
  const declaredTypes = new Set(corpus.message_types.map((entry) => entry.type));
  for (const fixture of corpus.fixtures) {
    assert.ok(fixture.input_frames.length > 0, `${fixture.fixture_id} declares no input frames`);
    assert.ok(corpus.session_states.includes(fixture.pre_state.state), `${fixture.fixture_id} pre-state is outside the vocabulary`);
    assert.ok(corpus.session_states.includes(fixture.expected.state), `${fixture.fixture_id} expected state is outside the vocabulary`);
    if (fixture.expected.client_state) {
      assert.ok(corpus.client_states.includes(fixture.expected.client_state), `${fixture.fixture_id} client state is outside the vocabulary`);
    }
    assert.equal(fixture.expected.no_side_effect, true, `${fixture.fixture_id} must declare its no-side-effect assertion`);
    assert.equal(fixture.expected.bounded_completion, true, `${fixture.fixture_id} must declare bounded completion`);
    assert.equal(fixture.expected.redaction.carries_raw_tab_fields, false);
    assert.equal(fixture.expected.redaction.echoes_request_id_when_parseable, true);
    assert.equal(fixture.expected.redaction.message_key_is_closed_vocabulary, true);
    if (fixture.expected.error_code) {
      assert.ok(known.has(fixture.expected.error_code), `${fixture.fixture_id} code is outside the vocabulary`);
    }
    assert.ok(fixture.expected.emitted_frames.length > 0);
    for (const frame of fixture.expected.emitted_frames) {
      assert.ok(declaredTypes.has(frame.type), `${fixture.fixture_id} emits undeclared type ${frame.type}`);
      if (frame.code) {
        assert.ok(known.has(frame.code), `${fixture.fixture_id} emits unknown code ${frame.code}`);
      }
      if ((frame.emit_count ?? 0) > 1) {
        assert.equal(frame.identical_bytes, true, `${fixture.fixture_id} repeated emit must declare identical bytes`);
      }
    }
    for (const spec of fixture.input_frames) {
      const declaresRaw = typeof spec.raw_payload_text === 'string';
      const declaresSemantic = typeof spec.type === 'string';
      assert.ok(declaresRaw !== declaresSemantic, `${fixture.fixture_id} has an ambiguous input frame`);
      if (spec.expected_error_code) {
        assert.ok(known.has(spec.expected_error_code), `${fixture.fixture_id} case expects an unknown code`);
      }
      if (typeof spec.type === 'string' && spec.type.length > corpus.limits.message_type_max_bytes) {
        assert.fail(`${fixture.fixture_id} message type exceeds the declared maximum`);
      }
      if (typeof spec.profile_id === 'string' && spec.profile_id.length > corpus.envelope.profile_id_max_bytes) {
        assert.fail(`${fixture.fixture_id} profile id exceeds the declared maximum`);
      }
    }
  }
});

test('replayed fixture bytes are deterministic across runs', () => {
  for (const fixture of corpus.fixtures) {
    const first = replay(fixture);
    const second = replay(fixture);
    assert.equal(first.length, second.length, `${fixture.fixture_id} replay length differs`);
    for (let index = 0; index < first.length; index += 1) {
      assert.ok(first[index].bytes.equals(second[index].bytes), `${fixture.fixture_id} frame ${index} bytes are not deterministic`);
      const declared = first[index].bytes.readUInt32LE(0);
      assert.equal(first[index].bytes.length - 4, declared, `${fixture.fixture_id} frame ${index} length prefix disagrees`);
      if (first[index].spec.expected_error_code) {
        continue;
      }
      assert.ok(declared <= corpus.transport_boundary.max_frame_bytes, `${fixture.fixture_id} accepted frame exceeds the transport boundary`);
      assert.ok(first[index].payloadBytes <= corpus.transport_boundary.max_json_payload_bytes, `${fixture.fixture_id} accepted payload exceeds the JSON cap`);
      const shape = JSON.parse(first[index].bytes.subarray(4).toString('utf8'));
      assert.equal(shape.protocol, first[index].spec.protocol ?? corpus.envelope.protocol_major);
      assert.equal(shape.type, first[index].spec.type);
      assert.equal(typeof shape.payload, 'object');
    }
  }
});

test('the payload boundary is exact to one byte', () => {
  const base = { type: 'query', profile_id: corpus.identity.profile_id, projection_revision: 1, payload: { query: 'alpha', result_limit: 10, current_window_id: 1 } };
  const atLimit = materialize({ ...base, payload_padding: { field: 'note', total_json_payload_bytes: corpus.transport_boundary.max_json_payload_bytes, filler: 'x' } });
  assert.equal(atLimit.payloadBytes, corpus.transport_boundary.max_json_payload_bytes);
  assert.ok(atLimit.bytes.length - 4 < corpus.transport_boundary.max_frame_bytes);
  const overLimit = materialize({ ...base, payload_padding: { field: 'note', total_json_payload_bytes: corpus.transport_boundary.max_json_payload_bytes + 1, filler: 'x' } });
  assert.equal(overLimit.payloadBytes - corpus.transport_boundary.max_json_payload_bytes, 1);
  assert.ok(overLimit.bytes.length - 4 < corpus.transport_boundary.max_frame_bytes);
});

test('query scalars land exactly on the declared boundary', () => {
  const base = { type: 'query', profile_id: corpus.identity.profile_id, projection_revision: 1, payload: { query: 'a', result_limit: 10, current_window_id: 1 } };
  for (const scalars of [corpus.limits.query_max_scalars, corpus.limits.query_max_scalars + 1]) {
    const wire = materialize({ ...base, query_scalars: scalars });
    const shape = JSON.parse(wire.bytes.subarray(4).toString('utf8'));
    assert.equal([...shape.payload.query].length, scalars);
  }
});

test('limit precedence states that the payload cap binds before the record cap', () => {
  assert.ok(corpus.limit_precedence.length > 0);
  const cases = new Map(corpus.limit_precedence.map((entry) => [entry.case, entry]));
  assert.equal(cases.get('frame_bytes_over_1048576').owner, 'IP-06');
  assert.equal(cases.get('json_payload_over_262144').owner, 'IP-07');
  const recordCase = cases.get('snapshot_records_over_10000');
  assert.equal(recordCase.reachable_on_the_wire, false);
  assert.ok(recordCase.note.length > 0);
  const wire = materialize({ type: 'snapshot', profile_id: corpus.identity.profile_id, payload: { sequence: 1, tabs: [] }, repeat_record_count: corpus.limits.snapshot_max_records });
  assert.ok(wire.payloadBytes > corpus.transport_boundary.max_json_payload_bytes, 'the record cap must be unreachable behind the payload cap');
});

test('expected outputs never restate a sensitive sentinel', () => {
  const sentinels = Object.values(corpus.privacy_sentinels);
  for (const sentinel of sentinels) {
    assert.ok(sentinel.trim().length > 0, 'privacy sentinel set is incomplete');
  }
  for (const fixture of corpus.fixtures) {
    const encoded = JSON.stringify(fixture.expected).toLowerCase();
    for (const sentinel of sentinels) {
      assert.ok(!encoded.includes(sentinel.toLowerCase()), `${fixture.fixture_id} restates ${sentinel}`);
    }
  }
});

test('the shared catalog still routes IP-07 through the reconnect seam', async () => {
  const catalog = JSON.parse(await readFile('fixtures/catalog.json', 'utf8'));
  assert.equal(catalog.fixtures.length, 9);
  const seams = catalog.phase_seams.filter((seam) => seam.phase === 'IP-07');
  assert.equal(seams.length, 1);
  assert.equal(seams[0].ownership_relation, 'supporting');
  assert.equal(seams[0].fixture_signal, 'snapshot_requested_after_reconnect');
  assert.ok(seams[0].fixture_ids.includes('FX-RECONNECT'));
  // NM-PROTO-* are phase-local fixtures; the shared catalog's reconnect fixture
  // stays owned by IP-09 with IP-07 as a supporting seam.
  const owned = catalog.fixtures.filter((fixture) => fixture.owner_phase === 'IP-07');
  assert.deepEqual(owned, []);
  assert.equal(catalog.fixtures.find((fixture) => fixture.fixture_id === 'FX-RECONNECT').owner_phase, 'IP-09');
});
