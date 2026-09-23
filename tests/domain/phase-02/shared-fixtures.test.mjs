import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { parseProfileID, parseProjectionEpoch, emptyProjection, acceptSnapshot, applyEvent, applyBatch, validateActivationReference, tabIdentityKey, retainActivations } from '../../../extension/dist/domain/index.js';

const epoch = parseProjectionEpoch('123e4567-e89b-42d3-a456-426614174001');
function record(profile_id, tab_id, projection_epoch = epoch) {
  const tab_identity = { profile_id, context_kind: 'normal', tab_id };
  return { profile_id, context_kind: 'normal', tab_identity, window_id: 1, group_id: null, title_display: 'duplicate', title_search: 'duplicate', url_search: 'https://example.test/', url_display: 'example.test', domain_display: 'example.test', domain_search: 'example.test', pinned: false, active: false, eligible: true, observed_at: 1, projection_epoch, projection_revision: 1 };
}

test('TypeScript domain consumes the shared Phase-02 fixture catalog', async () => {
  const catalog = JSON.parse(await readFile('fixtures/domain/phase-02/phase-02.json', 'utf8'));
  const ids = catalog.fixtures.map(fixture => fixture.fixture_id).sort();
  assert.deepEqual(ids, ['FX-ACTIVATION-RETENTION', 'FX-IDENTITY-TIEBREAK', 'FX-OWNERSHIP-BOUNDARY', 'FX-PROFILE-ISOLATION', 'FX-PROJECTION-LIFECYCLE', 'FX-REVISION-FENCE']);
  const isolation = catalog.fixtures.find(fixture => fixture.fixture_id === 'FX-PROFILE-ISOLATION');
  const profiles = isolation.input.profile_ids.map(parseProfileID);
  assert.notEqual(tabIdentityKey({ profile_id: profiles[0], context_kind: 'normal', tab_id: isolation.input.same_tab_id }), tabIdentityKey({ profile_id: profiles[1], context_kind: 'normal', tab_id: isolation.input.same_tab_id }));
  const tiebreak = catalog.fixtures.find(fixture => fixture.fixture_id === 'FX-IDENTITY-TIEBREAK');
  assert.notEqual(tabIdentityKey({ profile_id: profiles[0], context_kind: 'normal', tab_id: tiebreak.input.tab_ids[0] }), tabIdentityKey({ profile_id: profiles[0], context_kind: 'normal', tab_id: tiebreak.input.tab_ids[1] }));
});

test('snapshot, ordered upsert, duplicate, and remove converge atomically', () => {
  const profile = parseProfileID('123e4567-e89b-42d3-a456-426614174000');
  const state = emptyProjection(profile, 'normal');
  const first = record(profile, 7), second = record(profile, 8);
  second.projection_revision = 2;
  assert.equal(acceptSnapshot(state, epoch, [first]), 'APPLIED');
  const add = { profile_id: profile, context_kind: 'normal', epoch, sequence: 2, previous_revision: 1, operation: 'upsert', tab_identity: second.tab_identity, record: second };
  assert.equal(applyEvent(state, add), 'APPLIED');
  assert.equal(applyEvent(state, add), 'DUPLICATE');
  const remove = { profile_id: profile, context_kind: 'normal', epoch, sequence: 3, previous_revision: 2, operation: 'remove', tab_identity: first.tab_identity };
  assert.equal(applyEvent(state, remove), 'APPLIED');
  assert.deepEqual([...state.records.values()].map(row => row.tab_identity.tab_id), [8]);
  assert.equal(Number(state.revision), 3);
});

test('activation retention enforces scope, valid source, and exact age boundary', () => {
  const profile = parseProfileID('123e4567-e89b-42d3-a456-426614174000');
  const other = parseProfileID('123e4567-e89b-42d3-a456-426614174002');
  const now = 40 * 86400000, boundary = now - 30 * 86400000;
  const base = { profile_id: profile, context_kind: 'normal', tab_identity: { profile_id: profile, context_kind: 'normal', tab_id: 4 }, domain: 'example.test', activated_at: boundary, source: 'keyboard_enter', storage_sequence: 1 };
  const rows = retainActivations([base, { ...base, activated_at: boundary - 1 }, { ...base, activated_at: now + 1 }, { ...base, context_kind: 'private' }, { ...base, profile_id: other }, { ...base, source: 'unknown' }], now);
  assert.deepEqual(rows, [base]);
});
