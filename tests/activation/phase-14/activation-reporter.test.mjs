import assert from 'node:assert/strict';
import test from 'node:test';

import {
  ActivationRecencyHandoff,
  ActivationReporter,
  buildActivationMetadata,
  buildActivationSubmission,
  isConfirmedActivation,
  mapActivationErrorToWire,
  normalizeActivationDomain,
} from '../../../extension/dist/src/runtime/activation-reporter.js';
import {
  parseProfileID,
  parseProjectionEpoch,
  parseRevision,
  retainActivations,
} from '../../../extension/dist/domain/index.js';

const PROFILE = parseProfileID('123e4567-e89b-42d3-a456-426614174000');
const EPOCH = parseProjectionEpoch('123e4567-e89b-42d3-a456-426614174010');
const REVISION = parseRevision(7);

function reference(overrides = {}) {
  return {
    profile_id: PROFILE,
    context_kind: 'normal',
    query_sequence: 4,
    result_id: 'result-7',
    projection_epoch: EPOCH,
    projection_revision: REVISION,
    tab_id: 7,
    window_id: 2,
    tab_identity: { profile_id: PROFILE, context_kind: 'normal', tab_id: 7 },
    ...overrides,
  };
}

function outcome(overrides = {}) {
  return {
    requestId: 'activation-1',
    ok: true,
    confirmed: true,
    browserState: 'tab_active',
    errorCode: undefined,
    durationMs: 4,
    ...overrides,
  };
}

function deferred() {
  let resolve;
  let reject;
  const promise = new Promise((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });
  return { promise, resolve, reject };
}

class FakeTransport {
  calls = [];
  impl = async () => ({ ok: true, value: { accepted: true } });

  async reportActivation(submission) {
    this.calls.push(submission);
    return this.impl(submission);
  }
}

test('serializes only the opaque result reference and observed literal', () => {
  const submission = buildActivationSubmission(reference(), outcome());

  assert.deepEqual(submission, {
    result_reference: {
      tab_id: 7,
      projection_epoch: EPOCH,
      projection_revision: 7,
      result_id: 'result-7',
    },
    tab_id: 7,
    projection_revision: 7,
    outcome: 'activated',
    error_code: undefined,
  });
  assert.equal(isConfirmedActivation(outcome()), true);
  assert.equal(JSON.stringify(submission).includes('title'), false);
  assert.equal(JSON.stringify(submission).includes('url'), false);
});

test('maps bounded activation failures onto the host closed error enum', () => {
  assert.equal(mapActivationErrorToWire('stale_selection'), 'REVISION_MISMATCH');
  assert.equal(mapActivationErrorToWire('profile_mismatch'), 'PROFILE_MISMATCH');
  assert.equal(mapActivationErrorToWire('context_mismatch'), 'PROFILE_MISMATCH');
  assert.equal(mapActivationErrorToWire('projection_rebuilding'), 'INDEX_REBUILDING');
  assert.equal(mapActivationErrorToWire('browser_permission_denied'), 'INTERNAL_FAILURE');
  assert.equal(mapActivationErrorToWire('activation_cancelled'), 'HOST_SHUTDOWN');

  const failed = buildActivationSubmission(reference(), outcome({
    ok: false,
    confirmed: true,
    browserState: 'window_focused',
    errorCode: 'stale_selection',
  }));
  assert.equal(failed.outcome, 'failed');
  assert.equal(failed.error_code, 'REVISION_MISMATCH');
});

test('a non-active browser state is never serialized as observed success', () => {
  const failed = buildActivationSubmission(reference(), outcome({
    ok: false,
    browserState: 'window_focused',
    errorCode: 'verification_failed',
  }));
  assert.equal(failed.outcome, 'failed');
  assert.equal(failed.error_code, 'INTERNAL_FAILURE');
  assert.equal(isConfirmedActivation(outcome({ browserState: 'window_focused' })), false);
  assert.equal(isConfirmedActivation(outcome({ errorCode: 'verification_failed' })), false);
});

test('acknowledged reporting leaves confirmed browser activation authoritative', async () => {
  const transport = new FakeTransport();
  const states = [];
  const reporter = new ActivationReporter({
    transport,
    onState: status => states.push(status.state),
  });

  const status = await reporter.report({ reference: reference(), outcome: outcome() });

  assert.deepEqual(status, {
    state: 'acknowledged',
    accepted: true,
    activationOk: true,
    browserConfirmed: true,
    browserState: 'tab_active',
    errorCode: undefined,
    hostErrorCode: undefined,
  });
  assert.equal(transport.calls.length, 1);
  assert.equal(transport.calls[0].outcome, 'activated');
  assert.deepEqual(states, ['reporting', 'acknowledged']);
  assert.equal(reporter.isPending(), false);
});

test('host persistence degradation is diagnostic only after browser confirmation', async () => {
  const transport = new FakeTransport();
  transport.impl = async () => ({
    ok: false,
    source: 'host',
    error: { code: 'PERSISTENCE_DEGRADED', retryable: true, message_key: 'persistence_degraded' },
  });
  const reporter = new ActivationReporter({ transport });

  const status = await reporter.report({ reference: reference(), outcome: outcome() });

  assert.equal(status.state, 'degraded');
  assert.equal(status.accepted, false);
  assert.equal(status.activationOk, true);
  assert.equal(status.browserConfirmed, true);
  assert.equal(status.browserState, 'tab_active');
  assert.equal(status.hostErrorCode, 'PERSISTENCE_DEGRADED');
});

test('host absence is unavailable and does not rewrite the browser outcome', async () => {
  const transport = new FakeTransport();
  transport.impl = async () => ({
    ok: false,
    source: 'client',
    error: { code: 'HOST_UNAVAILABLE', retryable: true },
  });
  const reporter = new ActivationReporter({ transport });

  const status = await reporter.report({ reference: reference(), outcome: outcome() });

  assert.equal(status.state, 'unavailable');
  assert.equal(status.activationOk, true);
  assert.equal(status.browserConfirmed, true);
  assert.equal(status.accepted, false);
  assert.equal(status.hostErrorCode, undefined);
});

test('a rejected acknowledgement and a transport exception are bounded degradation', async () => {
  const rejectedTransport = new FakeTransport();
  rejectedTransport.impl = async () => ({ ok: true, value: { accepted: false } });
  const rejected = await new ActivationReporter({ transport: rejectedTransport })
    .report({ reference: reference(), outcome: outcome() });
  assert.equal(rejected.state, 'degraded');
  assert.equal(rejected.activationOk, true);

  const throwingTransport = new FakeTransport();
  throwingTransport.impl = async () => { throw new Error('socket closed'); };
  const thrown = await new ActivationReporter({ transport: throwingTransport })
    .report({ reference: reference(), outcome: outcome() });
  assert.equal(thrown.state, 'unavailable');
  assert.equal(thrown.activationOk, true);
  assert.equal(thrown.browserConfirmed, true);
});

test('report starts immediately but does not block on transport resolution', async () => {
  const transport = new FakeTransport();
  const gate = deferred();
  transport.impl = () => gate.promise;
  const reporter = new ActivationReporter({ transport });

  const pending = reporter.report({ reference: reference(), outcome: outcome() });
  assert.equal(transport.calls.length, 1);
  assert.equal(reporter.view().state, 'reporting');
  assert.equal(reporter.isPending(), true);

  gate.resolve({ ok: true, value: { accepted: true } });
  const status = await pending;
  assert.equal(status.state, 'acknowledged');
  assert.equal(reporter.isPending(), false);
});

test('cancel and dispose stop pending acknowledgement without changing browser state', async () => {
  const transport = new FakeTransport();
  const gate = deferred();
  transport.impl = () => gate.promise;
  const reporter = new ActivationReporter({ transport });

  const pending = reporter.report({ reference: reference(), outcome: outcome() });
  reporter.cancel();
  gate.resolve({ ok: true, value: { accepted: true } });
  const cancelled = await pending;
  assert.equal(cancelled.state, 'cancelled');
  assert.equal(cancelled.activationOk, true);
  assert.equal(cancelled.browserConfirmed, true);

  reporter.dispose();
  const afterDispose = await reporter.report({ reference: reference(), outcome: outcome() });
  assert.equal(afterDispose.state, 'cancelled');
  assert.equal(transport.calls.length, 1);
});

const NOW = 1_700_000_000_000;

function recencyInput(overrides = {}) {
  return {
    reference: reference(),
    outcome: outcome(),
    domain: 'Example.Test',
    source: 'keyboard_enter',
    ...overrides,
  };
}

class FakeRecencySink {
  calls = [];
  impl = async record => { this.calls.push(record); };

  async append(record) {
    return this.impl(record);
  }
}

test('builds only bounded normal-context activation metadata', () => {
  const build = buildActivationMetadata(recencyInput(), NOW);

  assert.equal(build.state, 'valid');
  assert.deepEqual(build.record, {
    profile_id: PROFILE,
    context_kind: 'normal',
    tab_identity: { profile_id: PROFILE, context_kind: 'normal', tab_id: 7 },
    domain: 'example.test',
    activated_at: NOW,
    source: 'keyboard_enter',
    storage_sequence: 0,
  });
  assert.equal(normalizeActivationDomain(' Example.Test. '), 'example.test');
  assert.equal(normalizeActivationDomain('https://example.test/path?q=1'), null);
  assert.equal(normalizeActivationDomain('example.test:443'), null);
  assert.equal(normalizeActivationDomain('example test'), null);
  assert.deepEqual(retainActivations([build.record], NOW), [build.record]);
  assert.equal(JSON.stringify(build.record).includes('title'), false);
  assert.equal(JSON.stringify(build.record).includes('url'), false);
});

test('failed, private, and malformed recency inputs never produce a record', () => {
  assert.equal(buildActivationMetadata(recencyInput({
    outcome: outcome({ ok: false, browserState: 'window_focused', errorCode: 'stale_selection' }),
  }), NOW).state, 'not_confirmed');

  assert.equal(buildActivationMetadata(recencyInput({
    reference: reference({
      context_kind: 'private',
      tab_identity: { profile_id: PROFILE, context_kind: 'private', tab_id: 7 },
    }),
  }), NOW).state, 'private_context');

  assert.equal(buildActivationMetadata(recencyInput({
    reference: reference({
      tab_identity: { profile_id: PROFILE, context_kind: 'normal', tab_id: 8 },
    }),
  }), NOW).state, 'invalid_identity');

  assert.equal(buildActivationMetadata(recencyInput({
    reference: reference({
      tab_identity: { profile_id: parseProfileID('123e4567-e89b-42d3-a456-426614174001'), context_kind: 'normal', tab_id: 7 },
    }),
  }), NOW).state, 'profile_mismatch');

  assert.equal(buildActivationMetadata(recencyInput({ domain: 'https://example.test/?q=secret' }), NOW).state, 'invalid_domain');
  assert.equal(buildActivationMetadata(recencyInput({ activatedAt: NOW + 120_000 }), NOW).state, 'invalid_timestamp');
  assert.equal(buildActivationMetadata(recencyInput({ activatedAt: NOW - 31 * 86_400_000 }), NOW).state, 'invalid_timestamp');
  assert.equal(buildActivationMetadata(recencyInput({ source: 'mouse' }), NOW).state, 'invalid_source');
  assert.equal(buildActivationMetadata(recencyInput({ storageSequence: -1 }), NOW).state, 'invalid_sequence');
});

test('recency handoff appends exactly once only after confirmed activation', async () => {
  const sink = new FakeRecencySink();
  const handoff = new ActivationRecencyHandoff({ sink, now: () => NOW });

  const result = await handoff.append(recencyInput());
  assert.equal(result.state, 'appended');
  assert.deepEqual(sink.calls, [result.record]);
  assert.deepEqual(handoff.view(), result);

  const failed = await handoff.append(recencyInput({
    outcome: outcome({ ok: false, browserState: 'unchanged', errorCode: 'tab_missing' }),
  }));
  assert.equal(failed.state, 'not_confirmed');
  assert.equal(sink.calls.length, 1);
});

test('private context and an unavailable sink stay bounded and non-throwing', async () => {
  const sink = new FakeRecencySink();
  const handoff = new ActivationRecencyHandoff({ sink, now: () => NOW });
  const privateResult = await handoff.append(recencyInput({
    reference: reference({
      context_kind: 'private',
      tab_identity: { profile_id: PROFILE, context_kind: 'private', tab_id: 7 },
    }),
  }));
  assert.equal(privateResult.state, 'private_context');
  assert.equal(sink.calls.length, 0);

  const disabled = await new ActivationRecencyHandoff({ now: () => NOW }).append(recencyInput());
  assert.equal(disabled.state, 'disabled');
  assert.ok(disabled.record);

  const failingSink = new FakeRecencySink();
  failingSink.impl = async () => { throw new Error('storage unavailable'); };
  const failed = await new ActivationRecencyHandoff({ sink: failingSink, now: () => NOW }).append(recencyInput());
  assert.equal(failed.state, 'sink_failed');
});

test('cancelled recency handoff does not report an append after teardown', async () => {
  const sink = new FakeRecencySink();
  const gate = deferred();
  sink.impl = () => gate.promise;
  const handoff = new ActivationRecencyHandoff({ sink, now: () => NOW });

  const pending = handoff.append(recencyInput());
  handoff.cancel();
  gate.resolve();
  const result = await pending;
  assert.equal(result.state, 'cancelled');
});

test('reporter runs recency and status without blocking on either boundary', async () => {
  const sink = new FakeRecencySink();
  const recency = new ActivationRecencyHandoff({ sink, now: () => NOW });
  const transport = new FakeTransport();
  const reporter = new ActivationReporter({ transport, recency });

  const status = await reporter.report(recencyInput());
  assert.equal(status.state, 'acknowledged');
  assert.equal(recency.view().state, 'appended');
  assert.equal(sink.calls.length, 1);
});

test('reporter never appends recency for failed or private activation', async () => {
  const sink = new FakeRecencySink();
  const recency = new ActivationRecencyHandoff({ sink, now: () => NOW });
  const transport = new FakeTransport();
  const reporter = new ActivationReporter({ transport, recency });

  const failedStatus = await reporter.report(recencyInput({
    outcome: outcome({ ok: false, browserState: 'unchanged', errorCode: 'tab_missing' }),
  }));
  assert.equal(failedStatus.state, 'acknowledged');
  assert.equal(recency.view().state, 'not_confirmed');
  assert.equal(sink.calls.length, 0);

  const privateStatus = await reporter.report(recencyInput({
    reference: reference({
      context_kind: 'private',
      tab_identity: { profile_id: PROFILE, context_kind: 'private', tab_id: 7 },
    }),
  }));
  assert.equal(privateStatus.state, 'acknowledged');
  assert.equal(recency.view().state, 'private_context');
  assert.equal(sink.calls.length, 0);
});
