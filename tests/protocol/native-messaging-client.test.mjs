import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

import {
  NativeMessagingClient,
  PROTOCOL_DEADLINES,
  PROTOCOL_MAJOR,
} from '../../extension/dist/src/runtime/native-messaging-client.js';
import {
  parseProfileID,
  parseProjectionEpoch,
} from '../../extension/dist/domain/index.js';

const PROFILE_ID = parseProfileID('123e4567-e89b-42d3-a456-426614174000');
const PROJECTION_EPOCH = parseProjectionEpoch('123e4567-e89b-42d3-a456-426614174001');
const FIXTURES = JSON.parse(await readFile('fixtures/protocol/phase-07.json', 'utf8')).fixtures;

class PortEvent {
  listeners = new Set();

  addListener(listener) {
    this.listeners.add(listener);
  }

  removeListener(listener) {
    this.listeners.delete(listener);
  }

  emit(value) {
    for (const listener of this.listeners) listener(value);
  }
}

class FakeNativePort {
  messages = [];
  onMessage = new PortEvent();
  onDisconnect = new PortEvent();
  disconnectCount = 0;

  postMessage(message) {
    this.messages.push(message);
  }

  disconnect() {
    this.disconnectCount += 1;
  }
}

function makeHarness() {
  const port = new FakeNativePort();
  const timers = new Map();
  let nextTimer = 1;
  const client = new NativeMessagingClient({
    profileId: PROFILE_ID,
    contextKind: 'normal',
    hostName: 'com.example.infoboard',
    extensionVersion: '1.0.0',
    browserFamily: 'chrome',
    browserVersion: '126.0',
    capabilities: ['tab_projection'],
    supportedProtocols: [PROTOCOL_MAJOR],
    createPort: hostName => {
      assert.equal(hostName, 'com.example.infoboard');
      return port;
    },
    setTimer: (handler, timeoutMs) => {
      const handle = nextTimer++;
      timers.set(handle, { handler, timeoutMs });
      return handle;
    },
    clearTimer: handle => timers.delete(handle),
  });
  return { client, port, timers };
}

function hostResponse(request, type, payload, options = {}) {
  return {
    protocol: PROTOCOL_MAJOR,
    type,
    request_id: options.requestId ?? request.request_id,
    profile_id: options.profileId ?? PROFILE_ID,
    projection_revision: options.projectionRevision ?? request.projection_revision,
    payload,
  };
}

async function connect(harness) {
  const pending = harness.client.start();
  const hello = harness.port.messages.at(-1);
  harness.port.onMessage.emit(hostResponse(hello, 'hello_ack', {
    accepted_protocol: PROTOCOL_MAJOR,
    host_version: '0.1.0',
  }));
  assert.deepEqual(await pending, {
    ok: true,
    value: { accepted_protocol: PROTOCOL_MAJOR, host_version: '0.1.0' },
  });
  return hello;
}

async function acceptSnapshot(harness, projectionRevision = 42, sequence = 7) {
  const pending = harness.client.sendSnapshot({
    epoch: PROJECTION_EPOCH,
    projection_revision: projectionRevision,
    records: [],
    sequence,
  });
  const snapshot = harness.port.messages.at(-1);
  harness.port.onMessage.emit(hostResponse(snapshot, 'sync_ack', {
    accepted_revision: projectionRevision,
    accepted_sequence: sequence,
  }, { projectionRevision }));
  assert.deepEqual(await pending, {
    ok: true,
    value: { accepted_revision: projectionRevision, accepted_sequence: sequence },
  });
  return snapshot;
}

function queryResultPayload() {
  return {
    results: [{ result_id: 'result-1', tab_id: 9, window_id: 2, title_display: 'Example' }],
    ranking_model_version: 'rank-1',
    timing: { elapsed_ms: 3 },
  };
}

test('hello binds profile in the envelope and ignores an uncorrelated acknowledgement', async () => {
  const harness = makeHarness();
  const pending = harness.client.start();
  const hello = harness.port.messages.at(-1);

  assert.deepEqual(Object.keys(hello).sort(), [
    'payload', 'profile_id', 'projection_revision', 'protocol', 'request_id', 'type',
  ]);
  assert.equal(hello.type, 'hello');
  assert.equal(hello.protocol, PROTOCOL_MAJOR);
  assert.equal(hello.profile_id, PROFILE_ID);
  assert.equal(hello.payload.context_kind, 'normal');
  assert.equal(Object.hasOwn(hello.payload, 'profile_id'), false);
  assert.equal(ArrayBuffer.isView(hello), false);

  harness.port.onMessage.emit(hostResponse(hello, 'hello_ack', {
    accepted_protocol: PROTOCOL_MAJOR,
    host_version: '0.1.0',
  }, { requestId: 'not-the-pending-request' }));
  assert.equal(harness.client.view().host_acknowledged, false);
  assert.equal(harness.client.view().diagnostics.uncorrelated_responses, 1);

  harness.port.onMessage.emit(hostResponse(hello, 'hello_ack', {
    accepted_protocol: PROTOCOL_MAJOR,
    host_version: '0.1.0',
  }));
  assert.deepEqual(await pending, {
    ok: true,
    value: { accepted_protocol: PROTOCOL_MAJOR, host_version: '0.1.0' },
  });
});

test('NM-PROTO-005: client refuses queries before an authoritative snapshot', async () => {
  const fixture = FIXTURES.find(entry => entry.fixture_id === 'NM-PROTO-005');
  assert.ok(fixture, 'shared protocol fixture NM-PROTO-005 must exist');
  const expectedQuery = fixture.input_frames.find(frame => frame.type === 'query');
  assert.equal(expectedQuery.expected_error_code, 'SNAPSHOT_REQUIRED');

  const harness = makeHarness();
  await connect(harness);
  const sentBeforeQuery = harness.port.messages.length;
  const result = await harness.client.query({
    query: expectedQuery.payload.query,
    result_limit: expectedQuery.payload.result_limit,
    current_window_id: expectedQuery.payload.current_window_id,
  });

  assert.equal(result.ok, false);
  assert.equal(result.source, 'client');
  assert.equal(result.error.code, expectedQuery.expected_error_code);
  assert.equal(harness.port.messages.length, sentBeforeQuery);
});

test('snapshot carries its authoritative revision rather than deriving one from record count', async () => {
  const harness = makeHarness();
  await connect(harness);

  const sentBeforeInvalidSubmission = harness.port.messages.length;
  const invalid = await harness.client.sendSnapshot({
    epoch: PROJECTION_EPOCH,
    projection_revision: -1,
    records: [],
    sequence: 7,
  });
  assert.equal(invalid.ok, false);
  assert.equal(invalid.source, 'client');
  assert.equal(invalid.error.code, 'INVALID_SUBMISSION');
  assert.equal(harness.port.messages.length, sentBeforeInvalidSubmission);

  const snapshot = await acceptSnapshot(harness, 42, 7);
  assert.deepEqual(Object.keys(snapshot).sort(), [
    'payload', 'profile_id', 'projection_revision', 'protocol', 'request_id', 'type',
  ]);
  assert.equal(snapshot.type, 'snapshot');
  assert.equal(snapshot.projection_revision, 42);
  assert.deepEqual(snapshot.payload, { tabs: [], sequence: 7 });
  assert.equal(harness.client.view().state, 'Ready');
  assert.equal(harness.client.view().acknowledged_revision, 42);
});

test('query results resolve only for the correlated request and acknowledged revision', async () => {
  const harness = makeHarness();
  await connect(harness);
  await acceptSnapshot(harness, 42, 7);

  const pending = harness.client.query({ query: 'alpha', result_limit: 10, current_window_id: 2 });
  const query = harness.port.messages.at(-1);
  const payload = queryResultPayload();
  harness.port.onMessage.emit(hostResponse(query, 'query_result', payload, {
    requestId: 'another-request',
    projectionRevision: 42,
  }));
  assert.equal(harness.client.view().diagnostics.uncorrelated_responses, 1);

  harness.port.onMessage.emit(hostResponse(query, 'query_result', payload, { projectionRevision: 42 }));
  assert.deepEqual(await pending, {
    ok: true,
    value: {
      results: [{ result_id: 'result-1', tab_id: 9, window_id: 2, display: { title_display: 'Example' } }],
      ranking_model_version: 'rank-1',
      projection_revision: 42,
      timing: { elapsed_ms: 3 },
    },
  });
});

test('query results from a stale revision are rejected rather than returned', async () => {
  const harness = makeHarness();
  await connect(harness);
  await acceptSnapshot(harness, 42, 7);

  const pending = harness.client.query({ query: 'alpha', result_limit: 10, current_window_id: 2 });
  const query = harness.port.messages.at(-1);
  harness.port.onMessage.emit(hostResponse(query, 'query_result', queryResultPayload(), {
    projectionRevision: 41,
  }));

  const result = await pending;
  assert.equal(result.ok, false);
  assert.equal(result.source, 'host');
  assert.equal(result.error.code, 'REVISION_MISMATCH');
  assert.equal(harness.client.view().diagnostics.dropped_results, 1);
});

test('handshake protocol mismatch moves client to terminal incompatibility and stops sending', async () => {
  const harness = makeHarness();
  const pending = harness.client.start();
  const hello = harness.port.messages.at(-1);
  harness.port.onMessage.emit(hostResponse(hello, 'hello_ack', {
    accepted_protocol: PROTOCOL_MAJOR + 1,
    host_version: '0.1.0',
  }));

  const result = await pending;
  assert.equal(result.ok, false);
  assert.equal(result.source, 'host');
  assert.equal(result.error.code, 'PROTOCOL_MISMATCH');
  assert.equal(harness.client.view().state, 'Incompatible');

  const sentAfterHandshake = harness.port.messages.length;
  const rejectedQuery = await harness.client.query({ query: 'alpha', result_limit: 10, current_window_id: 2 });
  assert.equal(rejectedQuery.ok, false);
  assert.equal(rejectedQuery.error.code, 'HOST_UNAVAILABLE');
  assert.equal(harness.port.messages.length, sentAfterHandshake);
});

test('hello deadline settles the pending request and exposes a retryable local failure', async () => {
  const harness = makeHarness();
  const pending = harness.client.start();
  const hello = harness.port.messages.at(-1);
  const timer = [...harness.timers.values()].find(entry => entry.timeoutMs === PROTOCOL_DEADLINES.handshakeMs);
  assert.ok(timer, 'hello must use the declared handshake deadline');

  timer.handler();
  const result = await pending;
  assert.equal(result.ok, false);
  assert.equal(result.source, 'client');
  assert.equal(result.error.code, 'DEADLINE_EXCEEDED');
  assert.equal(harness.client.view().diagnostics.requests_timed_out, 1);
  assert.equal(harness.port.messages.length, 1);
  assert.equal(hello.type, 'hello');
});
