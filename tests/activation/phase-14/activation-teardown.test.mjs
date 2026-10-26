import assert from 'node:assert/strict';
import test from 'node:test';

import { createActivationHandoff } from '../../../extension/dist/src/search/activation-handoff.js';
import { ActivationReporter } from '../../../extension/dist/src/runtime/activation-reporter.js';
import {
  parseProfileID,
  parseProjectionEpoch,
} from '../../../extension/dist/domain/index.js';

const PROFILE = parseProfileID('123e4567-e89b-42d3-a456-426614174000');
const EPOCH = parseProjectionEpoch('123e4567-e89b-42d3-a456-426614174030');
const CONTEXT = { browserFamily: 'chrome', profileId: PROFILE, contextKind: 'normal' };

function row(overrides = {}) {
  return {
    resultId: 'result-7',
    tabId: 7,
    windowId: 2,
    title: 'Example title',
    domain: 'example.test',
    pinned: false,
    explanations: ['title match'],
    ...overrides,
  };
}

function model(overrides = {}) {
  return {
    profileId: PROFILE,
    sequence: 3,
    status: 'results',
    rows: [row()],
    projectionRevision: 4,
    projectionEpoch: EPOCH,
    ...overrides,
  };
}

function request(overrides = {}) {
  return {
    resultId: 'result-7',
    tabId: 7,
    windowId: 2,
    projectionRevision: 4,
    projectionEpoch: EPOCH,
    querySequence: 3,
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

class FakeBrowser {
  calls = [];
  getResult = { ok: true, value: { id: 7, windowId: 2, active: false } };
  focusResult = { ok: true, value: undefined };
  activateResult = { ok: true, value: { id: 7, windowId: 2, active: true } };

  async getTab(tabId) {
    this.calls.push(['getTab', tabId]);
    return this.getResult;
  }

  async focusWindow(windowId) {
    this.calls.push(['focusWindow', windowId]);
    return this.focusResult;
  }

  async activateTab(tabId) {
    this.calls.push(['activateTab', tabId]);
    return this.activateResult;
  }
}

class FakeTransport {
  calls = [];
  impl = async () => ({ ok: true, value: { accepted: true } });

  async reportActivation(submission) {
    this.calls.push(submission);
    return this.impl(submission);
  }
}

async function until(predicate) {
  for (let attempt = 0; attempt < 20; attempt += 1) {
    if (predicate()) return;
    await new Promise(resolve => setImmediate(resolve));
  }
  assert.fail('condition was not reached');
}

test('reset teardown cancels in-flight activation before any mutating browser call', async () => {
  const gate = deferred();
  const browser = new FakeBrowser();
  browser.getTab = async tabId => {
    browser.calls.push(['getTab', tabId]);
    return gate.promise;
  };
  const transport = new FakeTransport();
  const reporter = new ActivationReporter({ transport });
  const cleanups = [];
  let closed = 0;
  const handoff = createActivationHandoff({
    context: CONTEXT,
    querySource: { view: () => model() },
    closeSurface: () => { closed += 1; },
    browser,
    reporter,
    cleanupOwnedState: context => {
      assert.equal(reporter.isPending(), false);
      cleanups.push(context);
    },
  });

  const pending = handoff.activate(request());
  await until(() => browser.calls.length === 1);
  const teardown = await handoff.teardown('reset');

  assert.deepEqual(teardown, {
    reason: 'reset',
    cancelled: true,
    reportsDiscarded: true,
    cleanup: 'complete',
  });
  assert.deepEqual(cleanups, [{
    reason: 'reset',
    profileId: PROFILE,
    contextKind: 'normal',
  }]);
  assert.equal(closed, 0);
  assert.equal(transport.calls.length, 0);

  gate.resolve({ ok: true, value: { id: 7, windowId: 2, active: false } });
  const outcome = await pending;
  assert.equal(outcome.ok, false);
  assert.equal(outcome.confirmed, true);
  assert.equal(outcome.browserState, 'unchanged');
  assert.equal(outcome.errorCode, 'activation_cancelled');
  assert.equal(outcome.nextAction, 'none');
  assert.equal(outcome.reportState, 'cancelled');
  assert.deepEqual(browser.calls, [['getTab', 7]]);

  const afterTeardown = await handoff.activate(request());
  assert.equal(afterTeardown.errorCode, 'activation_cancelled');
  assert.deepEqual(browser.calls, [['getTab', 7]]);
  assert.equal(closed, 0);

  const repeated = await handoff.teardown('uninstall');
  assert.equal(repeated, teardown);
  assert.equal(cleanups.length, 1);
});

test('uninstall teardown discards a pending report and emits no later status', async () => {
  const browser = new FakeBrowser();
  const transport = new FakeTransport();
  const gate = deferred();
  transport.impl = () => gate.promise;
  const states = [];
  const reporter = new ActivationReporter({
    transport,
    onState: status => states.push(status.state),
  });
  const cleanups = [];
  let closed = 0;
  const handoff = createActivationHandoff({
    context: CONTEXT,
    querySource: { view: () => model() },
    closeSurface: () => { closed += 1; },
    browser,
    reporter,
    cleanupOwnedState: context => {
      assert.equal(reporter.isPending(), false);
      cleanups.push(context);
    },
  });

  const activation = await handoff.activate(request());
  assert.equal(activation.ok, true);
  assert.equal(closed, 1);
  assert.equal(reporter.isPending(), true);
  assert.equal(transport.calls.length, 1);

  const teardown = await handoff.teardown('uninstall');

  assert.equal(teardown.cleanup, 'complete');
  assert.equal(teardown.reportsDiscarded, true);
  assert.equal(reporter.isPending(), false);
  assert.equal(reporter.view().state, 'cancelled');
  assert.deepEqual(cleanups, [{
    reason: 'uninstall',
    profileId: PROFILE,
    contextKind: 'normal',
  }]);
  assert.deepEqual(browser.calls, [['getTab', 7], ['focusWindow', 2], ['activateTab', 7]]);
  assert.equal(closed, 1);

  gate.resolve({ ok: true, value: { accepted: true } });
  await new Promise(resolve => setImmediate(resolve));
  assert.equal(reporter.view().state, 'cancelled');
  assert.deepEqual(states, ['reporting']);
  assert.equal(transport.calls.length, 1);
});

test('cleanup seam absence and failure stay bounded without browser mutation', async () => {
  const browser = new FakeBrowser();
  let closed = 0;
  const withoutSeam = createActivationHandoff({
    context: CONTEXT,
    querySource: { view: () => model() },
    closeSurface: () => { closed += 1; },
    browser,
  });

  const notConfigured = await withoutSeam.teardown('reset');
  assert.equal(notConfigured.cleanup, 'not_configured');
  assert.equal(closed, 0);
  assert.deepEqual(browser.calls, []);

  const failing = createActivationHandoff({
    context: CONTEXT,
    querySource: { view: () => model() },
    closeSurface: () => { closed += 1; },
    browser,
    cleanupOwnedState: () => { throw new Error('owned cleanup unavailable'); },
  });

  const degraded = await failing.teardown('uninstall');
  assert.equal(degraded.cleanup, 'degraded');
  assert.equal(degraded.cancelled, true);
  assert.equal(degraded.reportsDiscarded, true);
  assert.equal(closed, 0);
  assert.deepEqual(browser.calls, []);
});
