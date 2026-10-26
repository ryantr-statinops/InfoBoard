import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import test from 'node:test';

const SCRIPT = 'tests/activation/phase-14/cmd/selection-to-activation-benchmark/main.mjs';
const FIXTURE = 'fixtures/activation/phase-14/ACT-012.json';

function runBenchmark(args) {
  const result = spawnSync(
    process.execPath,
    [SCRIPT, '--fixture', FIXTURE, ...args],
    { cwd: process.cwd(), encoding: 'utf8' },
  );
  return {
    status: result.status,
    stdout: result.stdout,
    stderr: result.stderr,
  };
}

test('ACT-012 measures 20 warm-up and 200 measured controller iterations', () => {
  const result = runBenchmark([
    '--warmup', '20',
    '--iterations', '200',
    '--p95-budget-ms', '100',
  ]);
  assert.equal(result.status, 0, result.stderr);
  const report = JSON.parse(result.stdout);

  assert.equal(report.fixture_id, 'FX-ACT-012');
  assert.equal(report.pass, true);
  assert.equal(report.exit_status, 0);
  assert.equal(report.parameters.warmup, 20);
  assert.equal(report.parameters.iterations, 200);
  assert.equal(report.parameters.p95_budget_ms, 100);
  assert.equal(report.environment.clock, 'process.hrtime.bigint (monotonic)');
  assert.equal(report.environment.browser_scheduling_excluded, true);
  assert.equal(report.activation_calls, 600);
  assert.equal(report.successful_activations, 200);
  assert.equal(report.status_counts.acknowledged, 200);
  assert.equal(report.stale_browser_calls, 0);
  assert.ok(report.controller_ms.p95 <= 100);
  assert.ok(report.controller_ms.p50 <= report.controller_ms.p95);
  assert.ok(report.controller_ms.p95 <= report.controller_ms.p99);
  assert.deepEqual(
    report.stale_fixtures.map(item => item.browser_calls),
    [0, 0, 0],
  );
});

test('ACT-012 benchmark exits non-zero when controller p95 exceeds budget', () => {
  const result = runBenchmark([
    '--warmup', '0',
    '--iterations', '5',
    '--p95-budget-ms', '0',
  ]);
  assert.equal(result.status, 1, result.stderr);
  const report = JSON.parse(result.stdout);
  assert.equal(report.pass, false);
  assert.equal(report.exit_status, 1);
  assert.ok(report.controller_ms.p95 > 0);
});
