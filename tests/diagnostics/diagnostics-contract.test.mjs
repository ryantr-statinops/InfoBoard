import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const root = new URL('../../', import.meta.url);

async function json(path) {
  return JSON.parse(await readFile(new URL(path, root), 'utf8'));
}

test('phase-16 diagnostics fixtures are registered without adding seams', async () => {
  const catalog = await json('fixtures/catalog.json');
  assert.equal(catalog.fixtures.length, 82);
  assert.equal(catalog.phase_seams.length, 19);
  const ids = new Set(catalog.fixtures.map(fixture => fixture.fixture_id));
  for (const fixture of [
    'FX-HEALTH-TRANSITIONS',
    'FX-ERROR-STATUS-MATRIX',
    'FX-REDACTION-SENSITIVE-FIELDS',
    'FX-REPAIR-BOUNDS',
    'FX-RETENTION-BOUNDARIES',
  ]) {
    assert.ok(ids.has(fixture), `${fixture} is registered`);
  }
  const seam = catalog.phase_seams.find(item => item.phase === 'IP-16');
  assert.deepEqual(seam.fixture_ids, [
    'FX-HEALTH-TRANSITIONS',
    'FX-ERROR-STATUS-MATRIX',
    'FX-REDACTION-SENSITIVE-FIELDS',
    'FX-REPAIR-BOUNDS',
    'FX-RETENTION-BOUNDARIES',
  ]);
  assert.equal(seam.fixture_signal, 'final_health');
});

test('health transition fixture is bounded and ends stopped', async () => {
  const text = await readFile(new URL('fixtures/diagnostics/health-transitions.jsonl', root), 'utf8');
  const lines = text.trim().split(/\r?\n/).map(line => JSON.parse(line));
  const transitions = lines.filter(line => line.step);
  const final = lines.find(line => line.final)?.final;
  assert.equal(transitions.length, 8);
  assert.equal(final.final_health, 'stopping');
  assert.equal(final.event_count, 8);
  assert.equal(final.lexical_available, false);
  for (const transition of transitions) {
    assert.ok(transition.from_state);
    assert.ok(transition.to_state);
    assert.ok(transition.reason_code);
    assert.ok(Number.isInteger(transition.expected_event_sequence));
  }
});

test('error status matrix covers protocol and local classes', async () => {
  const fixture = await json('fixtures/diagnostics/error-status-matrix.json');
  assert.equal(fixture.cases.length, 15);
  const classes = new Set(fixture.cases.map(item => item.error_class));
  for (const expected of [
    'INVALID_FRAME',
    'PROTOCOL_MISMATCH',
    'PROFILE_MISMATCH',
    'PAYLOAD_LIMIT',
    'SNAPSHOT_REQUIRED',
    'REVISION_MISMATCH',
    'INDEX_REBUILDING',
    'QUERY_TIMEOUT',
    'PERSISTENCE_DEGRADED',
    'HOST_SHUTDOWN',
    'INTERNAL_FAILURE',
    'HOST_MISSING',
    'PERMISSION_LIMITED',
    'STORAGE_QUARANTINED',
    'SCHEMA_UPGRADE_REQUIRED',
  ]) {
    assert.ok(classes.has(expected), `${expected} is mapped`);
  }
  for (const item of fixture.cases) {
    assert.equal(typeof item.retryable, 'boolean');
    assert.ok(item.status);
    assert.ok(item.action);
  }
});

test('redaction fixture names every sensitive context slot', async () => {
  const fixture = await json('fixtures/diagnostics/redaction-sensitive-fields.json');
  for (const key of ['title', 'url', 'query', 'token', 'content', 'browser_id', 'tab_id', 'private_context_data']) {
    assert.ok(Object.hasOwn(fixture.context, key), `${key} is injected`);
  }
  for (const value of fixture.expected_absent) {
    assert.ok(value.length > 0);
  }
  assert.ok(fixture.expected_absent.length >= 8);
});

test('repair and retention fixtures keep their bounded budgets', async () => {
  const repair = await json('fixtures/diagnostics/repair-bounds.json');
  assert.equal(repair.cases.length, 6);
  assert.ok(repair.cases.some(item => item.name === 'rebuild_cancellation'));
  assert.ok(repair.cases.some(item => item.name === 'persistence_reopen_timeout'));
  assert.ok(repair.cases.some(item => item.name === 'repeated_user_repair'));

  const retention = await json('fixtures/diagnostics/retention-boundaries.json');
  assert.equal(retention.retention.max_age_seconds, 604800);
  assert.equal(retention.retention.max_bytes, 10000000);
  assert.equal(retention.retention.max_record_bytes, 65536);
  assert.equal(retention.cases.length, 6);
  for (const item of retention.cases) {
    if (item.expected_total_lte) {
      assert.ok(item.expected_total_lte <= 10000000);
    }
  }
});
