import { readFile } from 'node:fs/promises';
import { dirname, join, resolve } from 'node:path';
import os from 'node:os';

import { createActivationHandoff } from '../../../../../extension/dist/src/search/activation-handoff.js';
import {
  ActivationRecencyHandoff,
  ActivationReporter,
} from '../../../../../extension/dist/src/runtime/activation-reporter.js';
import {
  parseProfileID,
  parseProjectionEpoch,
} from '../../../../../extension/dist/domain/index.js';

function nowMs() {
  return Number(process.hrtime.bigint()) / 1_000_000;
}

function elapsedMs(startedAt) {
  return Number(process.hrtime.bigint() - startedAt) / 1_000_000;
}

function percentile(values, percent) {
  if (values.length === 0) return 0;
  const sorted = [...values].sort((left, right) => left - right);
  const index = Math.min(
    sorted.length - 1,
    Math.max(0, Math.ceil((percent / 100) * sorted.length) - 1),
  );
  return sorted[index];
}

function summarize(values) {
  return {
    p50: percentile(values, 50),
    p95: percentile(values, 95),
    p99: percentile(values, 99),
  };
}

function ok(value) {
  return { ok: true, value };
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

function modelFor(input) {
  return {
    profileId: parseProfileID(input.profile_id),
    sequence: input.query_sequence,
    status: 'results',
    rows: input.rows.map(toRow),
    projectionRevision: input.projection_revision,
    projectionEpoch: parseProjectionEpoch(input.projection_epoch),
  };
}

function requestFor(input, model) {
  return {
    resultId: input.selected.result_id,
    tabId: input.selected.tab_id,
    windowId: input.selected.window_id,
    projectionRevision: input.request_projection_revision ?? model.projectionRevision,
    projectionEpoch: input.request_projection_epoch ?? model.projectionEpoch,
    querySequence: input.request_query_sequence ?? model.sequence,
  };
}

class FakeBrowser {
  calls = [];
  browserMs = 0;

  constructor(input, model) {
    this.input = input;
    this.model = model;
  }

  async record(method, id, action) {
    this.calls.push([method, id]);
    const startedAt = process.hrtime.bigint();
    const delay = this.input.browser_scheduling_delay_ms ?? 0;
    if (delay > 0) {
      await new Promise(resolvePromise => setTimeout(resolvePromise, delay));
    }
    const result = action();
    this.browserMs += elapsedMs(startedAt);
    return result;
  }

  getTab(tabId) {
    return this.record('getTab', tabId, () => ok(this.input.browser.get_tab));
  }

  focusWindow(windowId) {
    return this.record('focusWindow', windowId, () => ok(undefined));
  }

  activateTab(tabId) {
    return this.record('activateTab', tabId, () => ok(this.input.browser.activate_tab));
  }
}

class FakeTransport {
  calls = 0;

  async reportActivation() {
    this.calls += 1;
    return { ok: true, value: { accepted: true } };
  }
}

class FakeRecencySink {
  calls = 0;

  append() {
    this.calls += 1;
  }
}

class CountingBrowser {
  calls = 0;

  async getTab() {
    this.calls += 1;
    return ok({ id: 7, windowId: 2, active: false });
  }

  async focusWindow() {
    this.calls += 1;
    return ok(undefined);
  }

  async activateTab() {
    this.calls += 1;
    return ok({ id: 7, windowId: 2, active: true });
  }
}

async function settle() {
  await new Promise(resolvePromise => setImmediate(resolvePromise));
}

async function runStaleFixture(fixturePath) {
  const fixture = JSON.parse(await readFile(fixturePath, 'utf8'));
  const input = fixture.input;
  const model = modelFor(input);
  const browser = new CountingBrowser();
  const handoff = createActivationHandoff({
    context: {
      browserFamily: 'chrome',
      profileId: parseProfileID(input.context_profile_id ?? input.profile_id),
      contextKind: input.context_kind,
    },
    querySource: { view: () => model },
    closeSurface: () => {},
    browser,
  });
  const outcome = await handoff.activate(requestFor(input, model));
  return {
    fixture_id: fixture.fixture_id,
    browser_calls: browser.calls,
    error_code: outcome.errorCode ?? null,
  };
}

function readNumber(args, name, fallback) {
  const index = args.indexOf(name);
  if (index === -1) return fallback;
  const value = Number(args[index + 1]);
  if (!Number.isInteger(value) || value < 0) {
    throw new Error(`${name} must be a non-negative integer`);
  }
  return value;
}

async function main() {
  const args = process.argv.slice(2);
  if (args.includes('--help')) {
    console.log(
      'Usage: node tests/activation/phase-14/cmd/selection-to-activation-benchmark/main.mjs '
      + '--fixture <path> --warmup 20 --iterations 200 --p95-budget-ms 100',
    );
    return;
  }

  const fixtureArg = args.indexOf('--fixture');
  const fixturePath = resolve(
    fixtureArg === -1
      ? 'fixtures/activation/phase-14/ACT-012.json'
      : args[fixtureArg + 1],
  );
  const fixture = JSON.parse(await readFile(fixturePath, 'utf8'));
  const input = fixture.input;
  const warmup = readNumber(args, '--warmup', input.warmup ?? 20);
  const iterations = readNumber(args, '--iterations', input.iterations ?? 200);
  const budget = readNumber(args, '--p95-budget-ms', input.p95_budget_ms ?? 100);
  if (iterations === 0) throw new Error('--iterations must be greater than zero');

  const model = modelFor(input);
  const request = requestFor(input, model);
  const browser = new FakeBrowser(input, model);
  const transport = new FakeTransport();
  const sink = new FakeRecencySink();
  const recency = new ActivationRecencyHandoff({ sink, now: () => 1_700_000_000_000 });
  const reporter = new ActivationReporter({ transport, recency });
  const handoff = createActivationHandoff({
    context: {
      browserFamily: 'chrome',
      profileId: parseProfileID(input.context_profile_id ?? input.profile_id),
      contextKind: input.context_kind,
    },
    querySource: { view: () => model },
    closeSurface: () => {},
    browser,
    reporter,
  });

  for (let index = 0; index < warmup; index += 1) {
    await handoff.activate(request);
    await settle();
  }

  const controllerSamples = [];
  const browserSamples = [];
  const totalSamples = [];
  const statusCounts = {};
  let activationCalls = 0;
  let successfulActivations = 0;

  for (let index = 0; index < iterations; index += 1) {
    browser.calls = [];
    browser.browserMs = 0;
    const startedAt = process.hrtime.bigint();
    const outcome = await handoff.activate(request);
    const totalMs = elapsedMs(startedAt);
    const browserMs = browser.browserMs;
    controllerSamples.push(Math.max(0, totalMs - browserMs));
    browserSamples.push(browserMs);
    totalSamples.push(totalMs);
    activationCalls += browser.calls.length;
    if (outcome.ok && outcome.browserState === 'tab_active') successfulActivations += 1;
    await settle();
    const state = reporter.view().state;
    statusCounts[state] = (statusCounts[state] ?? 0) + 1;
  }

  const staleFixtures = [];
  for (const relative of input.stale_fixtures ?? []) {
    staleFixtures.push(await runStaleFixture(join(dirname(fixturePath), relative)));
  }
  const staleBrowserCalls = staleFixtures.reduce(
    (total, result) => total + result.browser_calls,
    0,
  );

  const controller = summarize(controllerSamples);
  const pass = controller.p95 <= budget
    && staleBrowserCalls === 0
    && successfulActivations === iterations;
  const report = {
    fixture_id: fixture.fixture_id,
    pass,
    exit_status: pass ? 0 : 1,
    environment: {
      node: process.version,
      platform: process.platform,
      arch: process.arch,
      cpu: os.cpus()[0]?.model ?? 'unknown',
      cpus: os.cpus().length,
      clock: 'process.hrtime.bigint (monotonic)',
      browser_scheduling_excluded: true,
      browser_scheduling_delay_ms: input.browser_scheduling_delay_ms ?? 0,
      fake_browser: 'immediate',
    },
    parameters: {
      warmup,
      iterations,
      p95_budget_ms: budget,
    },
    controller_ms: controller,
    browser_scheduling_ms: summarize(browserSamples),
    total_ms: summarize(totalSamples),
    activation_calls: activationCalls,
    successful_activations: successfulActivations,
    status_counts: statusCounts,
    stale_fixtures: staleFixtures,
    stale_browser_calls: staleBrowserCalls,
  };
  console.log(JSON.stringify(report, null, 2));
  if (!pass) process.exitCode = 1;
}

await main();
