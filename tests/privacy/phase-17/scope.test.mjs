import assert from 'node:assert/strict';
import test from 'node:test';
import { readFile } from 'node:fs/promises';

import {
  parseProfileID,
  parseProjectionEpoch,
  parseRevision,
  tabIdentityKey,
} from '../../../extension/dist/domain/index.js';
import {
  ScopeArena,
  ScopeGuard,
  parseScope,
  sameScope,
  scopeKey,
} from '../../../extension/dist/src/privacy/scope.js';

const crossRead = JSON.parse(await readFile('fixtures/privacy/phase-17/profile-cross-read.json', 'utf8'));
const crossActivate = JSON.parse(await readFile('fixtures/privacy/phase-17/profile-cross-activate.json', 'utf8'));

const A = parseProfileID(crossRead.scope_a.profile_id);
const B = parseProfileID(crossRead.scope_b.profile_id);
const EPOCH = parseProjectionEpoch(crossRead.epoch_a);
const REVISION = parseRevision(crossRead.revision);

function record(profileId = A, contextKind = 'normal', tabId = 7, revision = REVISION) {
  const identity = { profile_id: profileId, context_kind: contextKind, tab_id: tabId };
  return {
    profile_id: profileId,
    context_kind: contextKind,
    tab_identity: identity,
    window_id: 2,
    group_id: 4,
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
    projection_revision: revision,
  };
}

function projection(profileId = A, contextKind = 'normal', records = [record(profileId, contextKind)]) {
  return {
    profile_id: profileId,
    context_kind: contextKind,
    epoch: EPOCH,
    revision: REVISION,
    records: new Map(records.map((item) => [tabIdentityKey(item.tab_identity), item])),
  };
}

function activationInput(reference, projectionView = projection()) {
  return {
    reference,
    surface: {
      profile_id: A,
      context_kind: 'normal',
      query_sequence: 4,
      selected_result_id: 'result-7',
    },
    projection: projectionView,
  };
}

test('scope parsing keeps normal and private scopes separate', () => {
  const normal = parseScope(crossRead.scope_a);
  const privateScope = parseScope({ ...crossRead.scope_a, context_kind: 'private', private_nonce: 'nonce-1' });
  assert.equal(sameScope(normal, privateScope), false);
  assert.notEqual(scopeKey(normal), scopeKey(privateScope));
  assert.throws(() => parseScope({ ...crossRead.scope_a, private_nonce: 'nonce-1' }), /INVALID_SCOPE/);
});

test('cross-profile host response, record, and revision are rejected', () => {
  const guard = new ScopeGuard(parseScope(crossRead.scope_a));
  assert.deepEqual(
    guard.acceptHostResponse(crossRead.attempts[0].response, REVISION),
    { ok: false, code: 'PROFILE_MISMATCH' },
  );
  assert.deepEqual(
    guard.acceptHostResponse({ profile_id: A, context_kind: 'private', projection_revision: 7 }, REVISION),
    { ok: false, code: 'CONTEXT_MISMATCH' },
  );
  assert.deepEqual(
    guard.acceptHostResponse({ profile_id: A, context_kind: 'normal', projection_revision: 8 }, REVISION),
    { ok: false, code: 'REVISION_MISMATCH' },
  );
  assert.deepEqual(
    guard.acceptRecord(record(B), EPOCH, REVISION),
    { ok: false, code: 'PROFILE_MISMATCH' },
  );
  assert.deepEqual(
    guard.acceptRecord(record(A, 'private'), EPOCH, REVISION),
    { ok: false, code: 'CONTEXT_MISMATCH' },
  );
});

test('scope arena never mixes normal and private records', () => {
  const arena = new ScopeArena();
  const normalScope = parseScope(crossRead.scope_a);
  const privateScope = parseScope({ ...crossRead.scope_a, context_kind: 'private', private_nonce: 'nonce-1' });
  arena.beginPrivateContext('nonce-1');
  assert.deepEqual(arena.admitSnapshot(normalScope, EPOCH, REVISION, [record(A)]), { ok: true, admitted: 1 });
  assert.deepEqual(arena.admitSnapshot(privateScope, EPOCH, REVISION, [record(A, 'private', 8)]), { ok: true, admitted: 1 });
  assert.equal(arena.records(normalScope).length, 1);
  assert.equal(arena.records(privateScope).length, 1);
  assert.equal(arena.records(normalScope)[0].context_kind, 'normal');
  assert.deepEqual(arena.counts(), { normal: 1, private: 1 });
  arena.endPrivateContext();
  assert.equal(arena.records(privateScope).length, 0);
  assert.equal(arena.records(normalScope).length, 1);
  assert.deepEqual(arena.counts(), { normal: 1, private: 0 });
});

test('private nonce ending invalidates late records and responses', () => {
  const guard = new ScopeGuard(parseScope({
    ...crossRead.scope_a,
    context_kind: 'private',
    private_nonce: 'nonce-1',
  }));
  assert.deepEqual(guard.acceptRecord(record(A, 'private'), EPOCH, REVISION), { ok: true });
  guard.endPrivateContext();
  assert.deepEqual(guard.acceptRecord(record(A, 'private'), EPOCH, REVISION), { ok: false, code: 'PRIVATE_CONTEXT_ENDED' });
  assert.equal(guard.isPrivateNonceCurrent('nonce-1'), false);
});

test('cross-profile and cross-context activation never reaches the browser', () => {
  const guard = new ScopeGuard(parseScope(crossActivate.scope_a));
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
    const outcome = guard.acceptActivation(activationInput(reference)).outcome;
    assert.equal(outcome, testCase.expected);
    assert.equal(testCase.expected_activation_calls, 0);
  }
});
