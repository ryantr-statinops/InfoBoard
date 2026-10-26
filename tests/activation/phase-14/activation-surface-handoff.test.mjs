import assert from 'node:assert/strict';
import test from 'node:test';

import {
  buildActivationGuardInput,
  createActivationHandoff,
} from '../../../extension/dist/src/search/activation-handoff.js';
import { guardActivation } from '../../../extension/dist/src/browser/activation-guard.js';
import {
  initialSurfaceState,
  permittedActions,
  reduceSurfaceState,
  statusMessage,
} from '../../../extension/dist/src/search/surface-state.js';
import {
  parseProfileID,
  parseProjectionEpoch,
} from '../../../extension/dist/domain/index.js';

const PROFILE = parseProfileID('123e4567-e89b-42d3-a456-426614174000');
const OTHER_PROFILE = parseProfileID('123e4567-e89b-42d3-a456-426614174001');
const EPOCH = parseProjectionEpoch('123e4567-e89b-42d3-a456-426614174020');
const OTHER_EPOCH = parseProjectionEpoch('123e4567-e89b-42d3-a456-426614174021');
const CONTEXT = { browserFamily: 'chrome', profileId: PROFILE, contextKind: 'normal' };

function row(overrides = {}) {
  return {
    resultId: 'result-7',
    tabId: 7,
    windowId: 2,
    title: 'Example title',
    domain: 'Example.Test',
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

test('builds a guard reference from the current revision-bound query view', () => {
  const built = buildActivationGuardInput(request(), model(), CONTEXT);
  assert.equal(built.ok, true);
  assert.equal(built.input.reference.profile_id, PROFILE);
  assert.equal(built.input.reference.context_kind, 'normal');
  assert.equal(built.input.reference.query_sequence, 3);
  assert.equal(built.input.reference.result_id, 'result-7');
  assert.equal(built.input.reference.projection_epoch, EPOCH);
  assert.equal(built.input.reference.projection_revision, 4);
  assert.equal(built.input.reference.tab_id, 7);
  assert.equal(built.input.reference.window_id, 2);
  assert.deepEqual(built.input.reference.tab_identity, {
    profile_id: PROFILE,
    context_kind: 'normal',
    tab_id: 7,
  });
  assert.equal(built.input.projection.records.size, 1);
  assert.equal(guardActivation(built.input).outcome, 'valid');
});

test('rejects stale, moved, missing, and foreign references without browser calls', () => {
  const cases = [
    [{ sequence: 4 }, 'stale_selection'],
    [{ projectionRevision: 5 }, 'stale_selection'],
    [{ projectionEpoch: OTHER_EPOCH }, 'stale_selection'],
    [{ rows: [] }, 'tab_missing'],
    [{ rows: [row({ tabId: 8 })] }, 'stale_selection'],
    [{ rows: [row({ windowId: 3 })] }, 'stale_selection'],
    [{ profileId: OTHER_PROFILE }, 'profile_mismatch'],
  ];
  for (const [overrides, expected] of cases) {
    const built = buildActivationGuardInput(request(), model(overrides), CONTEXT);
    assert.equal(built.ok, false);
    assert.equal(built.outcome, expected);
  }
});

test('confirmed exact activation reports before closing the surface', async () => {
  const events = [];
  const browser = new FakeBrowser();
  const transport = new FakeTransport();
  const originalGet = browser.getTab.bind(browser);
  browser.getTab = async tabId => {
    events.push('getTab');
    return originalGet(tabId);
  };
  const originalFocus = browser.focusWindow.bind(browser);
  browser.focusWindow = async windowId => {
    events.push('focusWindow');
    return originalFocus(windowId);
  };
  const originalActivate = browser.activateTab.bind(browser);
  browser.activateTab = async tabId => {
    events.push('activateTab');
    return originalActivate(tabId);
  };
  transport.impl = async submission => {
    events.push('report');
    return { ok: true, value: { accepted: true } };
  };
  const handoff = createActivationHandoff({
    context: CONTEXT,
    querySource: { view: () => model() },
    closeSurface: () => events.push('close'),
    browser,
    transport,
  });

  const outcome = await handoff.activate(request());

  assert.deepEqual(events, ['getTab', 'focusWindow', 'activateTab', 'report', 'close']);
  assert.deepEqual(outcome, {
    ok: true,
    confirmed: true,
    browserState: 'tab_active',
    nextAction: 'dismiss',
    errorCode: undefined,
    reportState: 'pending',
  });
  assert.equal(transport.calls.length, 1);
  assert.equal(transport.calls[0].outcome, 'activated');
  assert.equal(transport.calls[0].result_reference.result_id, 'result-7');
});

test('partial browser failure keeps the surface available and reports a bounded next action', async () => {
  const browser = new FakeBrowser();
  browser.activateResult = {
    ok: false,
    error: { kind: 'operation_failed', retryable: true },
  };
  const transport = new FakeTransport();
  let closed = 0;
  const handoff = createActivationHandoff({
    context: CONTEXT,
    querySource: { view: () => model() },
    closeSurface: () => { closed += 1; },
    browser,
    transport,
  });

  const outcome = await handoff.activate(request());

  assert.equal(outcome.ok, false);
  assert.equal(outcome.confirmed, true);
  assert.equal(outcome.browserState, 'window_focused');
  assert.equal(outcome.errorCode, 'browser_operation_failed');
  assert.equal(outcome.nextAction, 'retry');
  assert.equal(closed, 0);
  assert.deepEqual(browser.calls, [['getTab', 7], ['focusWindow', 2], ['activateTab', 7]]);
  assert.equal(transport.calls[0].outcome, 'failed');
  assert.equal(transport.calls[0].error_code, 'INTERNAL_FAILURE');
});

test('stale selection performs zero browser calls and offers refresh', async () => {
  const browser = new FakeBrowser();
  const handoff = createActivationHandoff({
    context: CONTEXT,
    querySource: { view: () => model({ sequence: 4 }) },
    closeSurface: () => { throw new Error('must not close'); },
    browser,
  });

  const outcome = await handoff.activate(request());

  assert.equal(outcome.ok, false);
  assert.equal(outcome.confirmed, true);
  assert.equal(outcome.browserState, 'unchanged');
  assert.equal(outcome.errorCode, 'stale_selection');
  assert.equal(outcome.nextAction, 'refresh');
  assert.equal(browser.calls.length, 0);
});

test('revalidation stops a changed revision after the initial tab read', async () => {
  const browser = new FakeBrowser();
  let current = model();
  browser.getTab = async tabId => {
    browser.calls.push(['getTab', tabId]);
    current = model({ sequence: 4 });
    return { ok: true, value: { id: 7, windowId: 2, active: false } };
  };
  const handoff = createActivationHandoff({
    context: CONTEXT,
    querySource: { view: () => current },
    closeSurface: () => { throw new Error('must not close'); },
    browser,
  });

  const outcome = await handoff.activate(request());

  assert.equal(outcome.ok, false);
  assert.equal(outcome.browserState, 'unchanged');
  assert.equal(outcome.errorCode, 'stale_selection');
  assert.equal(outcome.nextAction, 'refresh');
  assert.deepEqual(browser.calls, [['getTab', 7]]);
});

test('verification failure never closes or claims success', async () => {
  const browser = new FakeBrowser();
  browser.activateResult = {
    ok: true,
    value: { id: 7, windowId: 2, active: false },
  };
  let closed = 0;
  const handoff = createActivationHandoff({
    context: CONTEXT,
    querySource: { view: () => model() },
    closeSurface: () => { closed += 1; },
    browser,
  });

  const outcome = await handoff.activate(request());

  assert.equal(outcome.ok, false);
  assert.equal(outcome.browserState, 'window_focused');
  assert.equal(outcome.errorCode, 'verification_failed');
  assert.equal(outcome.nextAction, 'retry');
  assert.equal(closed, 0);
});

test('host unavailability after confirmation is diagnostics only', async () => {
  const browser = new FakeBrowser();
  const transport = new FakeTransport();
  transport.impl = async () => ({
    ok: false,
    source: 'client',
    error: { code: 'HOST_UNAVAILABLE', retryable: true },
  });
  const { ActivationReporter } = await import('../../../extension/dist/src/runtime/activation-reporter.js');
  const reporter = new ActivationReporter({ transport });
  let closed = 0;
  const handoff = createActivationHandoff({
    context: CONTEXT,
    querySource: { view: () => model() },
    closeSurface: () => { closed += 1; },
    browser,
    reporter,
  });

  const outcome = await handoff.activate(request());
  await new Promise(resolve => setTimeout(resolve, 0));

  assert.equal(outcome.ok, true);
  assert.equal(outcome.browserState, 'tab_active');
  assert.equal(closed, 1);
  assert.equal(reporter.view().state, 'unavailable');
});

test('an unexpected browser exception becomes a bounded result, not a raw exception', async () => {
  const browser = new FakeBrowser();
  browser.getTab = async () => { throw new Error('browser disconnected'); };
  let closed = 0;
  const handoff = createActivationHandoff({
    context: CONTEXT,
    querySource: { view: () => model() },
    closeSurface: () => { closed += 1; },
    browser,
  });

  const outcome = await handoff.activate(request());

  assert.equal(outcome.ok, false);
  assert.equal(outcome.confirmed, true);
  assert.equal(outcome.browserState, 'unknown');
  assert.equal(outcome.errorCode, 'browser_operation_failed');
  assert.equal(outcome.nextAction, 'retry');
  assert.equal(closed, 0);
});

test('surface state carries explicit browser state, next action, and safe messages', () => {
  let state = reduceSurfaceState(initialSurfaceState(), { type: 'ready' });
  state = reduceSurfaceState(state, { type: 'query', query: 'docs', sequence: 3 });
  state = reduceSurfaceState(state, { type: 'model', model: model() });
  assert.equal(state.projectionEpoch, EPOCH);

  const pending = reduceSurfaceState(state, { type: 'activation_started' });
  assert.equal(pending.activation.pending, true);
  assert.equal(pending.activation.browserState, 'unchanged');
  assert.equal(pending.activation.nextAction, 'none');

  const stale = reduceSurfaceState(pending, {
    type: 'activation_result',
    ok: false,
    confirmed: true,
    browserState: 'unchanged',
    nextAction: 'refresh',
    errorCode: 'stale_selection',
    reportState: 'unavailable',
  });
  assert.equal(stale.activation.browserState, 'unchanged');
  assert.equal(stale.activation.nextAction, 'refresh');
  assert.match(statusMessage(stale), /Results changed/);
  assert.ok(permittedActions(stale).includes('retry'));

  const success = reduceSurfaceState(pending, {
    type: 'activation_result',
    ok: true,
    confirmed: true,
    browserState: 'tab_active',
    nextAction: 'dismiss',
    reportState: 'pending',
  });
  assert.equal(success.activation.browserState, 'tab_active');
  assert.equal(success.activation.nextAction, 'dismiss');
  assert.equal(statusMessage(success), 'Opened selected tab');
});
