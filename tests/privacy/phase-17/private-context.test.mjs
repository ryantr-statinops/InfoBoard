import assert from 'node:assert/strict';
import test from 'node:test';
import { readFile } from 'node:fs/promises';

import { parseProfileID, parseProjectionEpoch, parseRevision } from '../../../extension/dist/domain/index.js';
import { parseScope } from '../../../extension/dist/src/privacy/scope.js';
import {
  PrivateLifetime,
  privateActivationOutcome,
  probePrivateContext,
  shouldPersistActivation,
} from '../../../extension/dist/src/privacy/private-context.js';

const separation = JSON.parse(await readFile('fixtures/privacy/phase-17/private-separation.json', 'utf8'));
const teardown = JSON.parse(await readFile('fixtures/privacy/phase-17/private-teardown.json', 'utf8'));
const denied = JSON.parse(await readFile('fixtures/privacy/phase-17/permission-denied.json', 'utf8'));

const A = parseProfileID(separation.normal_scope.profile_id);
const EPOCH = parseProjectionEpoch(separation.epoch);
const REVISION = parseRevision(separation.revision);

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

function lifetime() {
  return new PrivateLifetime(parseScope(separation.normal_scope), parseScope(separation.private_scope));
}

test('private and normal records stay in separate arenas', () => {
  const privateLifetime = lifetime();
  privateLifetime.begin(separation.private_scope.private_nonce);
  assert.deepEqual(
    privateLifetime.admitNormalSnapshot(EPOCH, REVISION, [record('normal', separation.normal_tab_id)]),
    { ok: true, admitted: 1 },
  );
  assert.deepEqual(
    privateLifetime.admitPrivateSnapshot(EPOCH, REVISION, [record('private', separation.private_tab_id)]),
    { ok: true, admitted: 1 },
  );
  assert.equal(privateLifetime.normalRecords().length, separation.expected.normal_count);
  assert.equal(privateLifetime.privateRecords().length, separation.expected.private_count);
  assert.equal(privateLifetime.normalRecords()[0].context_kind, 'normal');
  assert.equal(privateLifetime.privateRecords()[0].context_kind, 'private');
  assert.deepEqual(privateLifetime.counts(), { normal: 1, private: 1 });
});

test('private teardown clears memory and rejects a late response', () => {
  const privateLifetime = lifetime();
  privateLifetime.begin(teardown.private_scope.private_nonce);
  privateLifetime.admitNormalSnapshot(EPOCH, REVISION, [record('normal', separation.normal_tab_id)]);
  privateLifetime.admitPrivateSnapshot(EPOCH, REVISION, [record('private', separation.private_tab_id)]);
  privateLifetime.end();
  assert.equal(privateLifetime.privateRecords().length, teardown.expected.private_count_after_teardown);
  assert.equal(privateLifetime.normalRecords().length, teardown.expected.normal_count_after_teardown);
  assert.deepEqual(
    privateLifetime.acceptLatePrivateResponse(
      teardown.private_scope.private_nonce,
      teardown.late_response,
      REVISION,
    ),
    { ok: false, code: teardown.expected.late_response },
  );
  assert.deepEqual(privateLifetime.acceptPrivateActivation({}).outcome, 'private_context_ended');
});

test('private capability denial is deterministic and never escalates', async () => {
  for (const testCase of denied.cases) {
    const capability = await probePrivateContext({
      privateContextAccess: async () => testCase.browser_result,
    });
    assert.equal(capability.state, testCase.expected_state);
    assert.equal(capability.collectPrivate, testCase.expected_collect_private);
    assert.equal(capability.escalate, testCase.expected_escalation);
  }
  assert.equal(shouldPersistActivation('normal'), true);
  assert.equal(shouldPersistActivation('private'), false);
  assert.deepEqual(privateActivationOutcome('private', true), { persisted: false, outcome: 'not_persisted' });
  assert.deepEqual(privateActivationOutcome('private', false), { persisted: false, outcome: 'ignored' });
});
