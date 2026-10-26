import assert from 'node:assert/strict';
import test from 'node:test';

import { ActivationController } from '../../../extension/dist/src/browser/activation-controller.js';
import {
  parseProfileID,
  parseProjectionEpoch,
  parseRevision,
  tabIdentityKey,
} from '../../../extension/dist/domain/index.js';

const PROFILE = parseProfileID('123e4567-e89b-42d3-a456-426614174000');
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

function revisionAdvance() {
  return input({
    projection: projection({
      revision: parseRevision(8),
      records: new Map([[tabIdentityKey(identity()), record({ projection_revision: parseRevision(8) })]]),
    }),
  });
}

function tabRemoved() {
  return input({ projection: projection({ records: new Map() }) });
}

function tabMoved() {
  return input({
    projection: projection({
      records: new Map([[tabIdentityKey(identity()), record({ window_id: 9 })]]),
    }),
  });
}

function tabReused() {
  return input({
    projection: projection({
      records: new Map([[tabIdentityKey(identity()), record({ result_id: 'result-new' })]]),
    }),
  });
}

test('revision advance after tabs.get stops before window focus', async () => {
  const browser = new FakeActivationBrowser();
  let current = input();
  browser.getTabImpl = async () => {
    current = revisionAdvance();
    return { ok: true, value: { id: 7, windowId: 2, active: false } };
  };
  const controller = new ActivationController({ browser, revalidate: () => current });

  const result = await controller.activate(input());

  assert.equal(result.ok, false);
  assert.equal(result.browserState, 'unchanged');
  assert.equal(result.errorCode, 'stale_selection');
  assert.deepEqual(browser.calls, [['tabs.get', 7]]);
});

test('tab removal after tabs.get stops before window focus', async () => {
  const browser = new FakeActivationBrowser();
  let current = input();
  browser.getTabImpl = async () => {
    current = tabRemoved();
    return { ok: true, value: { id: 7, windowId: 2, active: false } };
  };
  const controller = new ActivationController({ browser, revalidate: () => current });

  const result = await controller.activate(input());

  assert.equal(result.browserState, 'unchanged');
  assert.equal(result.errorCode, 'tab_missing');
  assert.deepEqual(browser.calls, [['tabs.get', 7]]);
});

test('tab move after tabs.get stops before window focus', async () => {
  const browser = new FakeActivationBrowser();
  let current = input();
  browser.getTabImpl = async () => {
    current = tabMoved();
    return { ok: true, value: { id: 7, windowId: 2, active: false } };
  };
  const controller = new ActivationController({ browser, revalidate: () => current });

  const result = await controller.activate(input());

  assert.equal(result.browserState, 'unchanged');
  assert.equal(result.errorCode, 'window_missing');
  assert.deepEqual(browser.calls, [['tabs.get', 7]]);
});

test('numeric tab-id reuse after tabs.get stops before window focus', async () => {
  const browser = new FakeActivationBrowser();
  let current = input();
  browser.getTabImpl = async () => {
    current = tabReused();
    return { ok: true, value: { id: 7, windowId: 2, active: false } };
  };
  const controller = new ActivationController({ browser, revalidate: () => current });

  const result = await controller.activate(input());

  assert.equal(result.browserState, 'unchanged');
  assert.equal(result.errorCode, 'tab_reused');
  assert.deepEqual(browser.calls, [['tabs.get', 7]]);
});

test('revision advance after window focus stops before tab activation', async () => {
  const browser = new FakeActivationBrowser();
  let current = input();
  browser.focusWindowImpl = async () => {
    current = revisionAdvance();
    return { ok: true, value: undefined };
  };
  const controller = new ActivationController({ browser, revalidate: () => current });

  const result = await controller.activate(input());

  assert.equal(result.ok, false);
  assert.equal(result.browserState, 'window_focused');
  assert.equal(result.errorCode, 'stale_selection');
  assert.deepEqual(browser.calls, [
    ['tabs.get', 7],
    ['windows.update', 2, { focused: true }],
  ]);
});

test('tab removal after window focus stops before tab activation', async () => {
  const browser = new FakeActivationBrowser();
  let current = input();
  browser.focusWindowImpl = async () => {
    current = tabRemoved();
    return { ok: true, value: undefined };
  };
  const controller = new ActivationController({ browser, revalidate: () => current });

  const result = await controller.activate(input());

  assert.equal(result.browserState, 'window_focused');
  assert.equal(result.errorCode, 'tab_missing');
  assert.deepEqual(browser.calls, [
    ['tabs.get', 7],
    ['windows.update', 2, { focused: true }],
  ]);
});

test('tab move after window focus stops before tab activation', async () => {
  const browser = new FakeActivationBrowser();
  let current = input();
  browser.focusWindowImpl = async () => {
    current = tabMoved();
    return { ok: true, value: undefined };
  };
  const controller = new ActivationController({ browser, revalidate: () => current });

  const result = await controller.activate(input());

  assert.equal(result.browserState, 'window_focused');
  assert.equal(result.errorCode, 'window_missing');
  assert.deepEqual(browser.calls, [
    ['tabs.get', 7],
    ['windows.update', 2, { focused: true }],
  ]);
});

test('a worker/projection restart during revalidation reports rebuilding without mutation', async () => {
  const browser = new FakeActivationBrowser();
  let shouldThrow = false;
  const controller = new ActivationController({
    browser,
    revalidate: () => {
      if (shouldThrow) throw new Error('worker restarted');
      return input();
    },
  });
  browser.focusWindowImpl = async () => {
    shouldThrow = true;
    return { ok: true, value: undefined };
  };

  const result = await controller.activate(input());

  assert.equal(result.browserState, 'window_focused');
  assert.equal(result.errorCode, 'projection_rebuilding');
  assert.deepEqual(browser.calls, [
    ['tabs.get', 7],
    ['windows.update', 2, { focused: true }],
  ]);
});

test('a changed selected reference is rejected instead of retargeting activation', async () => {
  const browser = new FakeActivationBrowser();
  let current = input();
  browser.getTabImpl = async () => {
    current = input({
      reference: reference({ result_id: 'result-8' }),
      surface: surface({ selected_result_id: 'result-8' }),
      projection: projection({
        records: new Map([[tabIdentityKey(identity()), record({ result_id: 'result-8' })]]),
      }),
    });
    return { ok: true, value: { id: 7, windowId: 2, active: false } };
  };
  const controller = new ActivationController({ browser, revalidate: () => current });

  const result = await controller.activate(input());

  assert.equal(result.browserState, 'unchanged');
  assert.equal(result.errorCode, 'stale_selection');
  assert.deepEqual(browser.calls, [['tabs.get', 7]]);
});

test('a missing revalidation result is rejected without browser mutation', async () => {
  const browser = new FakeActivationBrowser();
  let current = input();
  browser.getTabImpl = async () => {
    current = null;
    return { ok: true, value: { id: 7, windowId: 2, active: false } };
  };
  const controller = new ActivationController({ browser, revalidate: () => current });

  const result = await controller.activate(input());

  assert.equal(result.browserState, 'unchanged');
  assert.equal(result.errorCode, 'stale_selection');
  assert.deepEqual(browser.calls, [['tabs.get', 7]]);
});

test('a closed window before focus maps to window_missing without tab activation', async () => {
  const browser = new FakeActivationBrowser();
  browser.focusWindowImpl = async () => ({
    ok: false,
    error: { kind: 'not_found', retryable: false },
  });
  const controller = new ActivationController({ browser });

  const result = await controller.activate(input());

  assert.equal(result.ok, false);
  assert.equal(result.browserState, 'unchanged');
  assert.equal(result.errorCode, 'window_missing');
  assert.deepEqual(browser.calls, [
    ['tabs.get', 7],
    ['windows.update', 2, { focused: true }],
  ]);
});

test('permission denial at each browser boundary maps to a bounded stage', async () => {
  const denied = { ok: false, error: { kind: 'permission_denied', retryable: false } };

  const getBrowser = new FakeActivationBrowser();
  getBrowser.getTabImpl = async () => denied;
  const getResult = await new ActivationController({ browser: getBrowser }).activate(input());
  assert.equal(getResult.browserState, 'unchanged');
  assert.equal(getResult.errorCode, 'browser_permission_denied');
  assert.deepEqual(getBrowser.calls, [['tabs.get', 7]]);

  const focusBrowser = new FakeActivationBrowser();
  focusBrowser.focusWindowImpl = async () => denied;
  const focusResult = await new ActivationController({ browser: focusBrowser }).activate(input());
  assert.equal(focusResult.browserState, 'unchanged');
  assert.equal(focusResult.errorCode, 'browser_permission_denied');
  assert.deepEqual(focusBrowser.calls, [
    ['tabs.get', 7],
    ['windows.update', 2, { focused: true }],
  ]);

  const activateBrowser = new FakeActivationBrowser();
  activateBrowser.activateTabImpl = async () => denied;
  const activateResult = await new ActivationController({ browser: activateBrowser }).activate(input());
  assert.equal(activateResult.browserState, 'window_focused');
  assert.equal(activateResult.errorCode, 'browser_permission_denied');
  assert.deepEqual(activateBrowser.calls, [
    ['tabs.get', 7],
    ['windows.update', 2, { focused: true }],
    ['tabs.update', 7, { active: true }],
  ]);
});

test('browser shutdown at each boundary never claims activation success', async () => {
  const shutdown = { ok: false, error: { kind: 'operation_failed', retryable: true } };

  const getBrowser = new FakeActivationBrowser();
  getBrowser.getTabImpl = async () => shutdown;
  const getResult = await new ActivationController({ browser: getBrowser }).activate(input());
  assert.equal(getResult.ok, false);
  assert.equal(getResult.browserState, 'unchanged');
  assert.equal(getResult.errorCode, 'browser_operation_failed');

  const focusBrowser = new FakeActivationBrowser();
  focusBrowser.focusWindowImpl = async () => shutdown;
  const focusResult = await new ActivationController({ browser: focusBrowser }).activate(input());
  assert.equal(focusResult.ok, false);
  assert.equal(focusResult.browserState, 'unchanged');
  assert.equal(focusResult.errorCode, 'browser_operation_failed');

  const activateBrowser = new FakeActivationBrowser();
  activateBrowser.activateTabImpl = async () => shutdown;
  const activateResult = await new ActivationController({ browser: activateBrowser }).activate(input());
  assert.equal(activateResult.ok, false);
  assert.equal(activateResult.browserState, 'window_focused');
  assert.equal(activateResult.errorCode, 'browser_operation_failed');
});
