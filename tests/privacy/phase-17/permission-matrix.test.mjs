import assert from 'node:assert/strict';
import test from 'node:test';
import { readFile } from 'node:fs/promises';

import {
  APPROVED_COMMANDS,
  APPROVED_MANIFEST_PERMISSIONS,
  PERMISSION_MATRIX,
  auditManifest,
  privateAccessDecision,
} from '../../../extension/dist/src/privacy/permissions.js';

const fixture = JSON.parse(await readFile('fixtures/privacy/phase-17/permission-matrix.json', 'utf8'));
const manifest = JSON.parse(await readFile('extension/manifest.json', 'utf8'));
const deniedManifest = JSON.parse(await readFile('fixtures/privacy/phase-17/manifest-denied.json', 'utf8'));

test('permission matrix matches the contract fixture', () => {
  assert.deepEqual(PERMISSION_MATRIX, fixture.matrix);
  assert.deepEqual(APPROVED_MANIFEST_PERMISSIONS, fixture.approved_permissions);
  assert.deepEqual(APPROVED_COMMANDS, fixture.approved_commands);
  assert.equal(new Set(PERMISSION_MATRIX.map((row) => row.capability)).size, PERMISSION_MATRIX.length);
});

test('approved manifest passes and permission creep fails closed', () => {
  assert.equal(auditManifest(manifest).ok, true);
  const audit = auditManifest(deniedManifest);
  assert.equal(audit.ok, false);
  for (const finding of [
    'permissions_not_approved_set',
    'denied_permission:history',
    'denied_permission:cookies',
    'denied_permission:scripting',
    'host_permissions_present',
    'optional_permissions_present',
    'broad_host_pattern',
    'content_scripts_present',
    'commands_not_approved_set',
  ]) {
    assert.ok(audit.findings.includes(finding), `missing ${finding}`);
  }
});

test('private denial never escalates or falls back', () => {
  assert.deepEqual(privateAccessDecision({ ok: true, value: true }), {
    state: 'available',
    collectPrivate: true,
    escalate: false,
  });
  for (const result of [
    { ok: false, error: { kind: 'permission_denied', retryable: false } },
    { ok: false, error: { kind: 'unsupported', retryable: false } },
    { ok: true, value: false },
  ]) {
    const decision = privateAccessDecision(result);
    assert.equal(decision.collectPrivate, false);
    assert.equal(decision.escalate, false);
  }
});
