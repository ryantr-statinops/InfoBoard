import assert from 'node:assert/strict';
import test from 'node:test';

import { createSettingsLifecycleClient } from '../../../extension/dist/src/settings/lifecycle-client.js';
import {
  DEFAULT_CONFIG,
  defaultProfileConfig,
  validateProfileConfig,
} from '../../../extension/dist/src/settings/contract.js';
import { PROTOCOL_MAJOR } from '../../../extension/dist/src/runtime/native-messaging-client.js';
import { parseProfileID } from '../../../extension/dist/domain/index.js';

const PROFILE_ID = parseProfileID('123e4567-e89b-42d3-a456-426614174000');

class PortEvent {
  listeners = new Set();
  addListener(listener) { this.listeners.add(listener); }
  removeListener(listener) { this.listeners.delete(listener); }
  emit(value) { for (const listener of this.listeners) listener(value); }
}

class FakeNativePort {
  messages = [];
  onMessage = new PortEvent();
  onDisconnect = new PortEvent();
  disconnectCount = 0;
  postMessage(message) { this.messages.push(message); }
  disconnect() { this.disconnectCount += 1; }
}

function harness() {
  const port = new FakeNativePort();
  const timers = new Map();
  let nextTimer = 1;
  const client = createSettingsLifecycleClient({
    profileId: PROFILE_ID,
    hostName: 'com.example.infoboard',
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

function response(request, type, payload) {
  return {
    protocol: PROTOCOL_MAJOR,
    type,
    request_id: request.request_id,
    profile_id: PROFILE_ID,
    projection_revision: request.projection_revision,
    payload,
  };
}

async function connect(harness) {
  const pending = harness.client.start();
  const hello = harness.port.messages.at(-1);
  harness.port.onMessage.emit(response(hello, 'hello_ack', {
    accepted_protocol: PROTOCOL_MAJOR,
    host_version: '0.1.0',
  }));
  assert.equal((await pending).ok, true);
}

function configuration(overrides = {}) {
  return {
    ...defaultProfileConfig(PROFILE_ID, PROTOCOL_MAJOR, 'rank-1'),
    ...overrides,
  };
}

function lifecyclePayload(operation, overrides = {}) {
  return {
    operation,
    profile_id: PROFILE_ID,
    status: 'complete',
    retryable: false,
    persistence: 'ready',
    durable: true,
    error_class: '',
    completed_steps: [],
    owned_state_verified: true,
    projection_revision: 0,
    configuration: configuration(),
    resync_required: false,
    ...overrides,
  };
}

test('settings contract publishes the IP-15 defaults and rejects profile crossing', () => {
  assert.deepEqual(DEFAULT_CONFIG, {
    result_limit: 10,
    density: 'comfortable',
    theme: 'system',
    context_labels: true,
    recency_enabled: true,
  });
  assert.equal(validateProfileConfig(configuration(), PROFILE_ID).result_limit, 10);
  assert.throws(
    () => validateProfileConfig(configuration({ profile_id: '123e4567-e89b-42d3-a456-426614174999' }), PROFILE_ID),
    error => error.name === 'ConfigValidationError' && error.field === 'profile_id',
  );
});

test('settings lifecycle client loads, saves, resets, and uninstalls through the native envelope', async () => {
  const state = harness();
  await connect(state);

  const loadPending = state.client.load();
  const loadEnvelope = state.port.messages.at(-1);
  assert.equal(loadEnvelope.type, 'lifecycle');
  assert.equal(loadEnvelope.payload.operation, 'load');
  assert.equal(Object.hasOwn(loadEnvelope.payload, 'configuration'), false);
  state.port.onMessage.emit(response(loadEnvelope, 'lifecycle_result', lifecyclePayload('load')));
  const loaded = await loadPending;
  assert.equal(loaded.ok, true);
  assert.equal(loaded.value.configuration.result_limit, 10);
  assert.equal(loaded.value.configuration.recency_enabled, true);

  const next = configuration({ result_limit: 25, density: 'compact', theme: 'dark', context_labels: false, recency_enabled: false });
  const updatePending = state.client.update(next);
  const updateEnvelope = state.port.messages.at(-1);
  assert.equal(updateEnvelope.payload.operation, 'update');
  assert.equal(updateEnvelope.payload.configuration.result_limit, 25);
  state.port.onMessage.emit(response(updateEnvelope, 'lifecycle_result', lifecyclePayload('update', {
    configuration: next,
  })));
  const updated = await updatePending;
  assert.equal(updated.ok, true);
  assert.equal(updated.value.configuration.theme, 'dark');

  const resetPending = state.client.reset();
  const resetEnvelope = state.port.messages.at(-1);
  assert.equal(resetEnvelope.payload.operation, 'reset');
  state.port.onMessage.emit(response(resetEnvelope, 'lifecycle_result', lifecyclePayload('reset', {
    completed_steps: ['storage_reset', 'index_cleared', 'snapshot_requested'],
    resync_required: true,
  })));
  const reset = await resetPending;
  assert.equal(reset.ok, true);
  assert.equal(reset.value.resync_required, true);

  const uninstallPending = state.client.uninstall();
  const uninstallEnvelope = state.port.messages.at(-1);
  assert.equal(uninstallEnvelope.payload.operation, 'uninstall');
  state.port.onMessage.emit(response(uninstallEnvelope, 'lifecycle_result', lifecyclePayload('uninstall', {
    completed_steps: ['storage_uninstall', 'paths_removed'],
  })));
  const uninstalled = await uninstallPending;
  assert.equal(uninstalled.ok, true);
  assert.deepEqual(uninstalled.value.completed_steps, ['storage_uninstall', 'paths_removed']);
});

test('settings lifecycle client rejects invalid update payloads before sending', async () => {
  const state = harness();
  await connect(state);
  const before = state.port.messages.length;
  const rejected = await state.client.update(configuration({ result_limit: 0 }));
  assert.deepEqual(rejected, {
    ok: false,
    source: 'client',
    error: { code: 'INVALID_SUBMISSION', retryable: false },
  });
  assert.equal(state.port.messages.length, before);
});

test('settings lifecycle client fails closed on a profile-crossing lifecycle result', async () => {
  const state = harness();
  await connect(state);
  const pending = state.client.load();
  const envelope = state.port.messages.at(-1);
  state.port.onMessage.emit(response(envelope, 'lifecycle_result', lifecyclePayload('load', {
    profile_id: '123e4567-e89b-42d3-a456-426614174999',
  })));
  const result = await pending;
  assert.equal(result.ok, false);
  assert.equal(result.source, 'host');
  assert.equal(result.error.code, 'INVALID_FRAME');
});
