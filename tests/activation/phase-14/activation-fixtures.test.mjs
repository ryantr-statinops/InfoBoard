import assert from 'node:assert/strict';
import { readdir, readFile } from 'node:fs/promises';
import test from 'node:test';

import { createActivationHandoff } from '../../../extension/dist/src/search/activation-handoff.js';
import {
  initialSurfaceState,
  reduceSurfaceState,
  statusMessage,
} from '../../../extension/dist/src/search/surface-state.js';
import {
  ActivationRecencyHandoff,
  ActivationReporter,
} from '../../../extension/dist/src/runtime/activation-reporter.js';
import {
  parseProfileID,
  parseProjectionEpoch,
} from '../../../extension/dist/domain/index.js';

const FIXED_NOW = 1_700_000_000_000;

function deferred() {
  let resolve;
  let reject;
  const promise = new Promise((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });
  return { promise, resolve, reject };
}

async function until(predicate) {
  for (let attempt = 0; attempt < 20; attempt += 1) {
    if (predicate()) return;
    await new Promise(resolve => setImmediate(resolve));
  }
  assert.fail('condition was not reached');
}

async function settle() {
  for (let attempt = 0; attempt < 8; attempt += 1) {
    await new Promise(resolve => setImmediate(resolve));
  }
}

function toRow(row) {
  return {
    resultId: row.result_id,
    tabId: row.tab_id,
    windowId: row.window_id,
    title: row.title,
    domain: row.domain,
    pinned: false,
    explanations: [],
  };
}

function failure(kind) {
  return { ok: false, error: { kind, retryable: kind === 'operation_failed' } };
}

function ok(value) {
  return { ok: true, value };
}

class FakeBrowser {
  calls = [];

  constructor(fixture, model) {
    this.fixture = fixture;
    this.model = model;
    this.gate = deferred();
  }

  async getTab(tabId) {
    this.calls.push(['getTab', tabId]);
    if (this.fixture.input.browser.defer_get) await this.gate.promise;

    const configured = this.fixture.input.browser.get_tab;
    if (configured === 'missing') return failure('not_found');

    const race = this.fixture.input.race;
    if (race === 'after_get_revision_advance') {
      this.model.projectionRevision += 1;
    }
    if (race === 'after_get_tab_removed') {
      this.model.rows = [];
    }
    if (race === 'after_get_tab_moved') {
      this.model.rows = this.model.rows.map(row => row.tabId === tabId
        ? { ...row, windowId: 3 }
        : row);
    }
    if (race === 'after_get_tab_reused') {
      this.model.sequence += 1;
      this.model.projectionRevision += 1;
      this.model.rows = [{
        ...toRow(this.fixture.input.selected),
        resultId: 'replacement-7',
        title: 'Replacement tab',
      }];
    }
    return ok(configured);
  }

  async focusWindow(windowId) {
    this.calls.push(['focusWindow', windowId]);
    const configured = this.fixture.input.browser.focus_window;
    if (configured === 'missing') return failure('not_found');
    if (configured === 'permission_denied') return failure('permission_denied');

    if (this.fixture.input.race === 'after_focus_tab_removed') {
      this.model.rows = [];
    }
    if (this.fixture.input.race === 'after_focus_revision_advance') {
      this.model.projectionRevision += 1;
    }
    return ok(undefined);
  }

  async activateTab(tabId) {
    this.calls.push(['activateTab', tabId]);
    const configured = this.fixture.input.browser.activate_tab;
    if (configured === 'missing') return failure('not_found');
    if (configured === 'permission_denied') return failure('permission_denied');
    if (configured === 'wrong_tab') return ok({ id: 8, windowId: 2, active: true });
    if (configured === 'inactive') return ok({ id: 7, windowId: 2, active: false });
    return ok(configured);
  }
}

class FakeTransport {
  calls = [];

  constructor(mode) {
    this.mode = mode;
  }

  async reportActivation(submission) {
    this.calls.push(submission);
    switch (this.mode) {
      case 'host_unavailable':
        return {
          ok: false,
          source: 'client',
          error: { code: 'HOST_UNAVAILABLE', retryable: true },
        };
      case 'persistence_degraded':
        return {
          ok: false,
          source: 'host',
          error: {
            code: 'PERSISTENCE_DEGRADED',
            retryable: true,
            message_key: 'PERSISTENCE_DEGRADED',
          },
        };
      case 'rejected':
        return { ok: true, value: { accepted: false } };
      default:
        return { ok: true, value: { accepted: true } };
    }
  }
}

class FakeRecencySink {
  calls = [];

  constructor(mode) {
    this.mode = mode;
  }

  append(record) {
    if (this.mode === 'degraded') throw new Error('storage unavailable');
    this.calls.push(record);
  }
}

function observations(actual) {
  return new Map(Object.entries(actual));
}

function assertExpected(fixture, actual) {
  const observed = observations(actual);
  for (const expected of fixture.expected.observations) {
    assert.deepEqual(
      observed.get(expected.signal),
      expected.value,
      `${fixture.fixture_id}: ${expected.signal}`,
    );
  }
}

async function runFixture(fixture) {
  const input = fixture.input;
  const profileId = parseProfileID(input.profile_id);
  const contextProfileId = parseProfileID(input.context_profile_id ?? input.profile_id);
  const projectionEpoch = parseProjectionEpoch(input.projection_epoch);
  const selected = toRow(input.selected);
  const initialModel = {
    profileId,
    sequence: input.query_sequence,
    status: 'results',
    rows: input.rows.map(toRow),
    projectionRevision: input.projection_revision,
    projectionEpoch,
  };
  const model = { ...initialModel, rows: [...initialModel.rows] };
  const browser = new FakeBrowser(fixture, model);
  const transport = new FakeTransport(input.transport);
  const sink = new FakeRecencySink(input.storage);
  const recency = new ActivationRecencyHandoff({
    sink,
    now: () => FIXED_NOW,
  });
  const reporter = new ActivationReporter({ transport, recency });
  const cleanups = [];
  let closeCount = 0;

  const handoff = createActivationHandoff({
    context: {
      browserFamily: 'chrome',
      profileId: contextProfileId,
      contextKind: input.context_kind,
    },
    querySource: { view: () => model },
    closeSurface: () => { closeCount += 1; },
    browser,
    reporter,
    cleanupOwnedState: context => { cleanups.push(context); },
  });

  const request = {
    resultId: selected.resultId,
    tabId: selected.tabId,
    windowId: selected.windowId,
    projectionRevision: input.request_projection_revision ?? initialModel.projectionRevision,
    projectionEpoch: input.request_projection_epoch ?? initialModel.projectionEpoch,
    querySequence: input.request_query_sequence ?? initialModel.sequence,
  };

  const activation = handoff.activate(request);
  const duplicate = input.duplicate_enter ? handoff.activate(request) : undefined;

  let teardownResult;
  if (input.teardown !== null) {
    await until(() => browser.calls.length > 0);
    teardownResult = await handoff.teardown(input.teardown);
    browser.gate.resolve(input.browser.get_tab);
  } else if (input.browser.defer_get) {
    await until(() => browser.calls.length > 0);
    browser.gate.resolve(input.browser.get_tab);
  }

  const outcome = await activation;
  if (duplicate !== undefined) await duplicate;
  await settle();

  let surface = reduceSurfaceState(initialSurfaceState(), { type: 'ready' });
  surface = reduceSurfaceState(surface, {
    type: 'query',
    query: 'example',
    sequence: initialModel.sequence,
  });
  surface = reduceSurfaceState(surface, { type: 'model', model: initialModel });
  surface = reduceSurfaceState(surface, { type: 'activation_started' });
  surface = reduceSurfaceState(surface, {
    type: 'activation_result',
    ok: outcome.ok,
    confirmed: outcome.confirmed,
    browserState: outcome.browserState,
    nextAction: outcome.nextAction,
    errorCode: outcome.errorCode,
    reportState: outcome.reportState,
  });

  const recencyView = recency.view();
  const record = recencyView.record;
  return {
    browser_calls: browser.calls,
    activation_ok: outcome.ok,
    browser_state: outcome.browserState,
    next_action: outcome.nextAction,
    error_code: outcome.errorCode ?? null,
    surface_closed: closeCount > 0,
    surface_status: surface.status,
    status_message: statusMessage(surface),
    report_state: reporter.view().state,
    report_outcome: transport.calls[0]?.outcome ?? null,
    recency_state: recencyView.state,
    recency_count: sink.calls.length,
    recency_record_keys: record === undefined ? [] : Object.keys(record).sort(),
    mutation_calls: browser.calls.filter(([method]) => method !== 'getTab').length,
    replacement_tab_activated: browser.calls.some(([method, id]) => (
      method === 'activateTab' && id !== selected.tabId
    )),
    duplicate_enter_single_flight: browser.calls.filter(([method]) => method === 'getTab').length === 1,
    cleanup_state: teardownResult?.cleanup ?? null,
    cleanup_count: cleanups.length,
    reports_discarded: teardownResult?.reportsDiscarded ?? null,
    teardown_reason: teardownResult?.reason ?? null,
  };
}

const fixtureFiles = (await readdir('fixtures/activation/phase-14'))
  .filter(name => name.startsWith('ACT-') && name.endsWith('.json'))
  .sort();

for (const file of fixtureFiles) {
  test(`fixture ${file}`, async () => {
    const fixture = JSON.parse(await readFile(`fixtures/activation/phase-14/${file}`, 'utf8'));
    const actual = await runFixture(fixture);
    assertExpected(fixture, actual);
  });
}
