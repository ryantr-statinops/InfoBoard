import assert from 'node:assert/strict';
import test from 'node:test';
import { execFileSync } from 'node:child_process';
import { readFile } from 'node:fs/promises';

import { parseProfileID, parseProjectionEpoch, parseRevision } from '../../../extension/dist/domain/index.js';
import { ScopeGuard, parseScope } from '../../../extension/dist/src/privacy/scope.js';
import { PrivateLifetime } from '../../../extension/dist/src/privacy/private-context.js';
import { auditManifest } from '../../../extension/dist/src/privacy/permissions.js';
import { redactDiagnosticRecord } from '../../../extension/dist/src/privacy/redaction.js';
import { validateFrameBytes, validateQueryInput, validateSnapshotCount } from '../../../extension/dist/src/privacy/limits.js';

const python = process.platform === 'win32' ? 'python' : 'python3';
const root = process.cwd();

function runPython(args) {
  return execFileSync(python, args, { cwd: root, encoding: 'utf8', timeout: 120_000 });
}

const crossRead = JSON.parse(await readFile('fixtures/privacy/phase-17/profile-cross-read.json', 'utf8'));
const crossActivate = JSON.parse(await readFile('fixtures/privacy/phase-17/profile-cross-activate.json', 'utf8'));
const separation = JSON.parse(await readFile('fixtures/privacy/phase-17/private-separation.json', 'utf8'));
const teardown = JSON.parse(await readFile('fixtures/privacy/phase-17/private-teardown.json', 'utf8'));
const redaction = JSON.parse(await readFile('fixtures/privacy/phase-17/redaction-sentinels.json', 'utf8'));
const deniedManifest = JSON.parse(await readFile('fixtures/privacy/phase-17/manifest-denied.json', 'utf8'));

const A = parseProfileID(crossRead.scope_a.profile_id);
const EPOCH = parseProjectionEpoch(crossRead.epoch_a);
const REVISION = parseRevision(crossRead.revision);

function record(contextKind, tabId) {
  return {
    profile_id: A,
    context_kind: contextKind,
    tab_identity: { profile_id: A, context_kind: contextKind, tab_id: tabId },
    window_id: 2,
    group_id: null,
    title_display: 'Example',
    title_search: 'example',
    url_search: 'https://example.test/path',
    url_display: 'https://example.test',
    domain_display: 'example.test',
    domain_search: 'example.test',
    pinned: false,
    active: false,
    eligible: true,
    observed_at: 1,
    projection_epoch: EPOCH,
    projection_revision: REVISION,
  };
}

test('cross-profile confused deputy is rejected before state or activation', () => {
  const guard = new ScopeGuard(parseScope(crossRead.scope_a));
  assert.deepEqual(
    guard.acceptHostResponse(crossRead.attempts[0].response, REVISION),
    { ok: false, code: 'PROFILE_MISMATCH' },
  );
  assert.deepEqual(
    guard.acceptRecord(record('normal', 99), EPOCH, REVISION),
    { ok: true },
  );
  const foreign = { ...record('normal', 99), profile_id: parseProfileID(crossRead.scope_b.profile_id) };
  foreign.tab_identity = { ...foreign.tab_identity, profile_id: foreign.profile_id };
  assert.deepEqual(guard.acceptRecord(foreign, EPOCH, REVISION), { ok: false, code: 'PROFILE_MISMATCH' });
  for (const testCase of crossActivate.cases) {
    const reference = {
      ...testCase.reference,
      profile_id: parseProfileID(testCase.reference.profile_id),
      projection_epoch: parseProjectionEpoch(testCase.reference.projection_epoch),
      projection_revision: parseRevision(testCase.reference.projection_revision),
      tab_identity: {
        ...testCase.reference.tab_identity,
        profile_id: parseProfileID(testCase.reference.tab_identity.profile_id),
      },
    };
    const outcome = guard.acceptActivation({
      reference,
      surface: { profile_id: A, context_kind: 'normal', query_sequence: 4, selected_result_id: 'result-7' },
      projection: {
        profile_id: A,
        context_kind: 'normal',
        epoch: EPOCH,
        revision: REVISION,
        records: new Map(),
      },
    }).outcome;
    assert.equal(outcome, testCase.expected);
  }
});

test('private-to-normal leakage and stale private response are impossible', () => {
  const lifetime = new PrivateLifetime(parseScope(separation.normal_scope), parseScope(separation.private_scope));
  lifetime.begin(separation.private_scope.private_nonce);
  lifetime.admitNormalSnapshot(EPOCH, REVISION, [record('normal', separation.normal_tab_id)]);
  lifetime.admitPrivateSnapshot(EPOCH, REVISION, [record('private', separation.private_tab_id)]);
  assert.equal(lifetime.normalRecords().length, 1);
  assert.equal(lifetime.normalRecords()[0].context_kind, 'normal');
  assert.equal(lifetime.privateRecords().length, 1);
  lifetime.end();
  assert.equal(lifetime.privateRecords().length, teardown.expected.private_count_after_teardown);
  assert.deepEqual(
    lifetime.acceptLatePrivateResponse(
      teardown.private_scope.private_nonce,
      teardown.late_response,
      REVISION,
    ),
    { ok: false, code: 'PRIVATE_CONTEXT_ENDED' },
  );
  assert.equal(lifetime.acceptPrivateActivation({}).outcome, 'private_context_ended');
});

test('permission creep, diagnostic exfiltration, and resource exhaustion fail closed', () => {
  assert.equal(auditManifest(deniedManifest).ok, false);
  const redacted = redactDiagnosticRecord({
    ...redaction.context,
    event_id: redaction.sensitive_values[0],
    error_class: redaction.sensitive_values[8],
  });
  const serialized = JSON.stringify(redacted);
  for (const sentinel of redaction.sensitive_values) assert.equal(serialized.includes(sentinel), false);
  assert.equal(validateFrameBytes(1_048_577).ok, false);
  assert.equal(validateSnapshotCount(1001).ok, false);
  assert.equal(validateQueryInput({ query: 'a'.repeat(513), currentWindowId: 1 }).ok, false);
  assert.equal(validateQueryInput({ query: 'go', resultLimit: 51, currentWindowId: 1 }).ok, false);
});

test('retention bypass and excluded-surface audits produce no pass for blocked environments', () => {
  const retention = JSON.parse(runPython([
    'tests/privacy/phase-17/audit_retention.py',
    '--db', 'fixtures/privacy/phase-17/retention.sqlite',
    '--max-age-days', '30',
    '--max-diagnostics-bytes', '10000000',
    '--json',
  ]));
  assert.equal(retention.ok, true);
  assert.equal(retention.activation_counts.profile_a, 500);
  assert.equal(retention.activation_counts.profile_b, 10);
  assert.equal(retention.diagnostics_bytes <= 10_000_000, true);
  assert.equal(retention.diagnostics_scope, 'global');

  const permissions = JSON.parse(runPython([
    'tests/privacy/phase-17/validate_permissions.py',
    '--manifest', 'extension/manifest.json',
    '--fixtures', 'fixtures/privacy/phase-17',
    '--json',
  ]));
  assert.equal(permissions.ok, true);
  assert.deepEqual(permissions.permissions, ['nativeMessaging', 'storage', 'tabGroups', 'tabs']);
  assert.equal(permissions.api_call_set.includes('history'), false);
  assert.equal(permissions.api_call_set.includes('cookies'), false);

  runPython([
    'tests/privacy/phase-17/validate_redaction.py',
    '--fixtures', 'fixtures/privacy/phase-17',
    '--diagnostics', 'fixtures/privacy/phase-17/output',
  ]);
  const matrix = JSON.parse(runPython([
    'tests/privacy/phase-17/run_browser_matrix.py',
    '--browsers', 'chrome,edge',
    '--contexts', 'normal,private',
    '--fixtures', 'fixtures/privacy/phase-17',
    '--json',
  ]));
  assert.equal(matrix.summary.pass, 0);
  assert.equal(matrix.summary.blocked, matrix.summary.total);
  assert.equal(matrix.summary.all_pass, false);
  for (const entry of matrix.entries) {
    assert.equal(entry.status, 'blocked');
    assert.equal(entry.pass, false);
    assert.ok(entry.reason.length > 0);
  }
});

test('the catalog registers exactly the eleven IP-17 fixtures without adding seams', async () => {
  const catalog = JSON.parse(await readFile('fixtures/catalog.json', 'utf8'));
  assert.equal(catalog.fixtures.length, 93);
  assert.equal(catalog.phase_seams.length, 19);
  const ids = new Set(catalog.fixtures.map((fixture) => fixture.fixture_id));
  for (const fixtureId of [
    'FX-PRIVACY-PERMISSION-MATRIX',
    'FX-PRIVACY-PROFILE-CROSS-READ',
    'FX-PRIVACY-PROFILE-CROSS-ACTIVATE',
    'FX-PRIVACY-PRIVATE-SEPARATION',
    'FX-PRIVACY-PRIVATE-TEARDOWN',
    'FX-PRIVACY-PERMISSION-DENIED',
    'FX-PRIVACY-REDACTION',
    'FX-PRIVACY-SECRET-REJECTION',
    'FX-PRIVACY-INPUT-LIMITS',
    'FX-PRIVACY-RETENTION-AUDIT',
    'FX-PRIVACY-EXCLUDED-SURFACE',
  ]) {
    assert.equal(ids.has(fixtureId), true, fixtureId);
  }
});
