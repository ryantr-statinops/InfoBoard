import assert from 'node:assert/strict';
import test from 'node:test';

import {
  ActivationController,
  ChromeActivationBrowser,
} from '../../../extension/dist/src/browser/activation-controller.js';
import {
  parseProfileID,
  parseProjectionEpoch,
  parseRevision,
  tabIdentityKey,
} from '../../../extension/dist/domain/index.js';

const PROFILE = parseProfileID('123e4567-e89b-42d3-a456-426614174000');
const OTHER_PROFILE = parseProfileID('123e4567-e89b-42d3-a456-426614174001');
const EPOCH = parseProjectionEpoch('123e4567-e89b-42d3-a456-426614174010');
const REVISION = parseRevision(7);

function identity(overrides = {}) {
  return { profile_id: PROFILE, context_kind: 'normal', tab_id: 7, ...overrides };
}

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
    tab_identity: identity(),
    ...overrides,
  };
}

function surface(overrides = {}) {
  return {
    profile_id: PROFILE,
    context_kind: 'normal',
    query_sequence: 4,
    selected_result_id: 'result-7',
    ...overrides,
  };
}

function record(overrides = {}) {
  return {
    tab_identity: identity(),
    window_id: 2,
    eligible: true,
    projection_epoch: EPOCH,
    projection_revision: REVISION,
    result_id: 'result-7',
    ...overrides,
  };
}

function projection(overrides = {}) {
  return {
    profile_id: PROFILE,
    context_kind: 'normal',
    epoch: EPOCH,
    revision: REVISION,
    records: new Map([[tabIdentityKey(identity()), record()]]),
    ...overrides,
  };
}

function input(overrides = {}) {
  return {
    reference: reference(),
    surface: surface(),
    projection: projection(),
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

class FakeActivationBrowser {
  calls = [];
  tab = { id: 7, windowId: 2, active: false };
  getTabImpl = async () => ({ ok: true, value: { ...this.tab } });
  focusWindowImpl = async () => ({ ok: true, value: undefined });
  activateTabImpl = async () => {
    this.tab.active = true;
    return { ok: true, value: { ...this.tab } };
  };

  async getTab(tabId) {
    this.calls.push(['tabs.get', tabId]);
    return this.getTabImpl(tabId);
  }

  async focusWindow(windowId) {
    this.calls.push(['windows.update', windowId, { focused: true }]);
    return this.focusWindowImpl(windowId);
  }

  async activateTab(tabId) {
    this.calls.push(['tabs.update', tabId, { active: true }]);
    return this.activateTabImpl(tabId);
  }
}

async function waitFor(predicate) {
  for (let attempt = 0; attempt < 20; attempt += 1) {
    if (predicate()) return;
    await new Promise(resolve => setTimeout(resolve, 0));
  }
  throw new Error('condition was not reached');
}

test('performs the exact get/focus/activate sequence and confirms the returned tab', async () => {
  const browser = new FakeActivationBrowser();
  let confirmed = 0;
  const controller = new ActivationController({
    browser,
    now: () => 100,
    onConfirmed: () => { confirmed += 1; },
  });

  const result = await controller.activate(input());

  assert.deepEqual(result, {
    requestId: 'activation-1',
    ok: true,
    confirmed: true,
    browserState: 'tab_active',
    errorCode: undefined,
    durationMs: 0,
  });
  assert.deepEqual(browser.calls, [
    ['tabs.get', 7],
    ['windows.update', 2, { focused: true }],
    ['tabs.update', 7, { active: true }],
  ]);
  assert.equal(confirmed, 1);
});

test('a rejected guard performs zero browser calls and never confirms activation', async () => {
  const browser = new FakeActivationBrowser();
  let confirmed = 0;
  const controller = new ActivationController({
    browser,
    onConfirmed: () => { confirmed += 1; },
  });

  const result = await controller.activate(input({
    reference: reference({ profile_id: OTHER_PROFILE }),
  }));

  assert.equal(result.ok, false);
  assert.equal(result.confirmed, true);
  assert.equal(result.browserState, 'unchanged');
  assert.equal(result.errorCode, 'profile_mismatch');
  assert.deepEqual(browser.calls, []);
  assert.equal(confirmed, 0);
});

test('duplicate Enter for the same reference reuses one in-flight operation', async () => {
  const browser = new FakeActivationBrowser();
  const gate = deferred();
  browser.getTabImpl = () => gate.promise;
  const controller = new ActivationController({ browser });

  const first = controller.activate(input());
  const second = controller.activate(input());
  assert.equal(first, second);
  assert.deepEqual(browser.calls, [['tabs.get', 7]]);

  gate.resolve({ ok: true, value: { id: 7, windowId: 2, active: false } });
  const [firstResult, secondResult] = await Promise.all([first, second]);
  assert.equal(firstResult.requestId, 'activation-1');
  assert.equal(secondResult.requestId, 'activation-1');
  assert.equal(firstResult.ok, true);
  assert.equal(browser.calls.length, 3);
});

test('a different request while activation is pending is rejected without another browser call', async () => {
  const browser = new FakeActivationBrowser();
  const gate = deferred();
  browser.getTabImpl = () => gate.promise;
  const controller = new ActivationController({ browser });

  const first = controller.activate(input());
  const second = await controller.activate(input({
    reference: reference({ result_id: 'result-8', tab_id: 8 }),
    surface: surface({ selected_result_id: 'result-8' }),
  }));

  assert.equal(second.ok, false);
  assert.equal(second.browserState, 'unchanged');
  assert.equal(second.errorCode, 'activation_in_progress');
  assert.equal(second.requestId, 'activation-2');
  assert.equal(browser.calls.length, 1);

  gate.resolve({ ok: true, value: { id: 7, windowId: 2, active: false } });
  await first;
});

test('rejects a tab whose current browser window differs from the reference', async () => {
  const browser = new FakeActivationBrowser();
  browser.getTabImpl = async () => ({ ok: true, value: { id: 7, windowId: 9, active: false } });
  const controller = new ActivationController({ browser });

  const result = await controller.activate(input());

  assert.equal(result.ok, false);
  assert.equal(result.browserState, 'unchanged');
  assert.equal(result.errorCode, 'window_missing');
  assert.deepEqual(browser.calls, [['tabs.get', 7]]);
});

test('rejects a tab whose current browser id differs from the reference', async () => {
  const browser = new FakeActivationBrowser();
  browser.getTabImpl = async () => ({ ok: true, value: { id: 8, windowId: 2, active: false } });
  const controller = new ActivationController({ browser });

  const result = await controller.activate(input());

  assert.equal(result.ok, false);
  assert.equal(result.browserState, 'unchanged');
  assert.equal(result.errorCode, 'tab_reused');
  assert.deepEqual(browser.calls, [['tabs.get', 7]]);
});

test('a window-focus failure leaves browser state unchanged and does not activate', async () => {
  const browser = new FakeActivationBrowser();
  browser.focusWindowImpl = async () => ({
    ok: false,
    error: { kind: 'permission_denied', retryable: false },
  });
  const controller = new ActivationController({ browser });

  const result = await controller.activate(input());

  assert.equal(result.ok, false);
  assert.equal(result.browserState, 'unchanged');
  assert.equal(result.errorCode, 'browser_permission_denied');
  assert.deepEqual(browser.calls, [
    ['tabs.get', 7],
    ['windows.update', 2, { focused: true }],
  ]);
});

test('a tab-activation failure reports partial window focus without claiming success', async () => {
  const browser = new FakeActivationBrowser();
  browser.activateTabImpl = async () => ({
    ok: false,
    error: { kind: 'not_found', retryable: false },
  });
  let confirmed = 0;
  const controller = new ActivationController({
    browser,
    onConfirmed: () => { confirmed += 1; },
  });

  const result = await controller.activate(input());

  assert.equal(result.ok, false);
  assert.equal(result.confirmed, true);
  assert.equal(result.browserState, 'window_focused');
  assert.equal(result.errorCode, 'tab_missing');
  assert.equal(confirmed, 0);
  assert.equal(browser.calls.length, 3);
});

test('verification failure after tab activation remains an unconfirmed partial result', async () => {
  const browser = new FakeActivationBrowser();
  browser.activateTabImpl = async () => ({
    ok: true,
    value: { id: 7, windowId: 2, active: false },
  });
  const controller = new ActivationController({ browser });

  const result = await controller.activate(input());

  assert.equal(result.ok, false);
  assert.equal(result.browserState, 'window_focused');
  assert.equal(result.errorCode, 'verification_failed');
  assert.equal(browser.calls.length, 3);
});

test('cancel before window focus stops after the read and reports unchanged state', async () => {
  const browser = new FakeActivationBrowser();
  const gate = deferred();
  browser.getTabImpl = () => gate.promise;
  const controller = new ActivationController({ browser });

  const pending = controller.activate(input());
  controller.cancel();
  gate.resolve({ ok: true, value: { id: 7, windowId: 2, active: false } });

  const result = await pending;
  assert.equal(result.ok, false);
  assert.equal(result.browserState, 'unchanged');
  assert.equal(result.errorCode, 'activation_cancelled');
  assert.deepEqual(browser.calls, [['tabs.get', 7]]);
});

test('cancel after window focus stops before tab activation and reports partial state', async () => {
  const browser = new FakeActivationBrowser();
  const gate = deferred();
  browser.focusWindowImpl = () => gate.promise;
  const controller = new ActivationController({ browser });

  const pending = controller.activate(input());
  await waitFor(() => browser.calls.some(call => call[0] === 'windows.update'));
  controller.cancel();
  gate.resolve({ ok: true, value: undefined });

  const result = await pending;
  assert.equal(result.ok, false);
  assert.equal(result.browserState, 'window_focused');
  assert.equal(result.errorCode, 'activation_cancelled');
  assert.deepEqual(browser.calls, [
    ['tabs.get', 7],
    ['windows.update', 2, { focused: true }],
  ]);
});

test('sequential requests receive monotonic request identities', async () => {
  const browser = new FakeActivationBrowser();
  const controller = new ActivationController({ browser });

  const first = await controller.activate(input());
  const second = await controller.activate(input({
    reference: reference({ result_id: 'result-8' }),
    surface: surface({ selected_result_id: 'result-8' }),
    projection: projection({
      records: new Map([[tabIdentityKey(identity()), record({ result_id: 'result-8' })]]),
    }),
  }));

  assert.equal(first.requestId, 'activation-1');
  assert.equal(second.requestId, 'activation-2');
});

test('ChromeActivationBrowser maps API values and bounded failures', async () => {
  const api = {
    tabs: {
      get: async tabId => ({ id: tabId, windowId: 2, active: false }),
      update: async (tabId, update) => ({ id: tabId, windowId: 2, active: update.active }),
    },
    windows: {
      update: async (windowId, update) => ({ id: windowId, focused: update.focused }),
    },
  };
  const browser = new ChromeActivationBrowser(api);

  assert.deepEqual(await browser.getTab(7), {
    ok: true,
    value: { id: 7, windowId: 2, active: false },
  });
  assert.deepEqual(await browser.focusWindow(2), { ok: true, value: undefined });
  assert.deepEqual(await browser.activateTab(7), {
    ok: true,
    value: { id: 7, windowId: 2, active: true },
  });

  const denied = new ChromeActivationBrowser({
    tabs: {
      get: async () => { throw new Error('permission denied'); },
      update: async () => { throw new Error('permission denied'); },
    },
    windows: {
      update: async () => { throw new Error('permission denied'); },
    },
  });
  assert.deepEqual(await denied.getTab(7), {
    ok: false,
    error: { kind: 'permission_denied', retryable: false },
  });
});
