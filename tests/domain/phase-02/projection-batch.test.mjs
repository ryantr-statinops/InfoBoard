import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { parseProfileID, parseProjectionEpoch, emptyProjection, acceptSnapshot, applyBatch, validateActivationReference } from '../../../extension/dist/domain/index.js';

function row(profile_id, tab_id, projection_epoch) {
  const tab_identity = { profile_id, context_kind: 'normal', tab_id };
  return { profile_id, context_kind: 'normal', tab_identity, window_id: 1, group_id: null, title_display: 'tab', title_search: 'tab', url_search: 'https://example.test', url_display: 'example.test', domain_display: 'example.test', domain_search: 'example.test', pinned: false, active: false, eligible: true, observed_at: 1, projection_epoch, projection_revision: 1 };
}

test('projection batches replay the shared lifecycle fixture and roll back sequence gaps', async () => {
  const catalog = JSON.parse(await readFile('fixtures/domain/phase-02/phase-02.json', 'utf8'));
  const isolation = catalog.fixtures.find(fixture => fixture.fixture_id === 'FX-PROFILE-ISOLATION');
  const lifecycle = catalog.fixtures.find(fixture => fixture.fixture_id === 'FX-PROJECTION-LIFECYCLE').input;
  const profile_id = parseProfileID(isolation.input.profile_ids[0]);
  const epoch = parseProjectionEpoch('123e4567-e89b-42d3-a456-426614174001');
  const state = emptyProjection(profile_id, 'normal');
  const initialRows = lifecycle.initial_tab_ids.map(tab_id => row(profile_id, tab_id, epoch));
  const added = row(profile_id, lifecycle.add_tab_id, epoch);
  added.projection_revision = 2;
  const removed = initialRows.find(value => value.tab_identity.tab_id === lifecycle.remove_tab_id);
  assert.ok(removed);
  assert.equal(acceptSnapshot(state, epoch, initialRows), 'APPLIED');
  const add = { profile_id, context_kind: 'normal', epoch, sequence: 2, previous_revision: 1, operation: 'upsert', tab_identity: added.tab_identity, record: added };
  const bad = { ...add, sequence: 4, previous_revision: 3 };
  assert.equal(applyBatch(state, [add, bad]), 'SNAPSHOT_REQUIRED');
  assert.equal(Number(state.revision), 1);
  assert.deepEqual([...state.records.values()].map(value => value.tab_identity.tab_id), lifecycle.initial_tab_ids);
  const removeEvent = { profile_id, context_kind: 'normal', epoch, sequence: 3, previous_revision: 2, operation: 'remove', tab_identity: removed.tab_identity };
  assert.equal(applyBatch(state, [add, removeEvent]), 'APPLIED');
  assert.deepEqual([...state.records.values()].map(value => value.tab_identity.tab_id), [lifecycle.add_tab_id]);
  const current = { profile_id, context_kind: 'normal', tab_id: lifecycle.add_tab_id, projection_epoch: epoch, projection_revision: state.revision, result_id: 'request-1' };
  assert.equal(validateActivationReference(current, state), true);
  assert.equal(validateActivationReference({ ...current, projection_revision: 2 }, state), false);
  assert.equal(validateActivationReference({ ...current, profile_id: parseProfileID(isolation.input.profile_ids[1]) }, state), false);
});
