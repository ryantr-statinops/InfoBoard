import assert from 'node:assert/strict';
import test from 'node:test';

import {
  guardActivation,
  isActivationGuardSuccess,
} from '../../../extension/dist/src/browser/activation-guard.js';
import {
  parseProfileID,
  parseProjectionEpoch,
  parseRevision,
  tabIdentityKey,
} from '../../../extension/dist/domain/index.js';

const PROFILE = parseProfileID('123e4567-e89b-42d3-a456-426614174000');
const OTHER_PROFILE = parseProfileID('123e4567-e89b-42d3-a456-426614174001');
const EPOCH = parseProjectionEpoch('123e4567-e89b-42d3-a456-426614174010');
const OTHER_EPOCH = parseProjectionEpoch('123e4567-e89b-42d3-a456-426614174011');
const REVISION = parseRevision(7);

function identity(overrides = {}) {
  return { profile_id: PROFILE, context_kind: 'normal', tab_id: 7, ...overrides };
}

function reference(overrides = {}) {
  return {
    profile_id: PROFILE,
    context_kind: 'normal',
    query_sequence: 4,
    result_id: 'result-7',
    projection_epoch: EPOCH,
    projection_revision: REVISION,
    tab_id: 7,
    window_id: 2,
    tab_identity: identity(),
    ...overrides,
  };
}

function surface(overrides = {}) {
  return {
    profile_id: PROFILE,
    context_kind: 'normal',
    query_sequence: 4,
    selected_result_id: 'result-7',
    ...overrides,
  };
}

function record(overrides = {}) {
  return {
    tab_identity: identity(),
    window_id: 2,
    eligible: true,
    projection_epoch: EPOCH,
    projection_revision: REVISION,
    result_id: 'result-7',
    ...overrides,
  };
}

function projection(overrides = {}) {
  return {
    profile_id: PROFILE,
    context_kind: 'normal',
    epoch: EPOCH,
    revision: REVISION,
    records: new Map([[tabIdentityKey(identity()), record()]]),
    ...overrides,
  };
}

function input(overrides = {}) {
  return {
    reference: reference(),
    surface: surface(),
    projection: projection(),
    ...overrides,
  };
}

test('accepts an exact profile/context/fence/tab/window reference', () => {
  const result = guardActivation(input());
  assert.deepEqual(result, { outcome: 'valid' });
  assert.equal(isActivationGuardSuccess(result), true);
});

test('rejects another profile before looking at tab identity', () => {
  const result = guardActivation(input({
    reference: reference({ profile_id: OTHER_PROFILE }),
    projection: projection({ records: new Map() }),
  }));
  assert.deepEqual(result, { outcome: 'profile_mismatch' });
  assert.equal(isActivationGuardSuccess(result), false);
});

test('rejects another context partition', () => {
  const result = guardActivation(input({
    reference: reference({ context_kind: 'private' }),
  }));
  assert.deepEqual(result, { outcome: 'context_mismatch' });
});

test('rejects a projection that is rebuilding', () => {
  const result = guardActivation(input({
    projection: projection({ epoch: null }),
  }));
  assert.deepEqual(result, { outcome: 'projection_rebuilding' });
});

test('rejects a superseded query sequence before checking the tab', () => {
  const result = guardActivation(input({
    reference: reference({ query_sequence: 3 }),
    projection: projection({ records: new Map() }),
  }));
  assert.deepEqual(result, { outcome: 'stale_selection' });
});

test('rejects a result that is no longer the selected result', () => {
  const result = guardActivation(input({
    reference: reference({ result_id: 'result-old' }),
  }));
  assert.deepEqual(result, { outcome: 'stale_selection' });
});

test('rejects a changed projection epoch or revision', () => {
  const epochResult = guardActivation(input({
    reference: reference({ projection_epoch: OTHER_EPOCH }),
  }));
  assert.deepEqual(epochResult, { outcome: 'stale_selection' });

  const revisionResult = guardActivation(input({
    reference: reference({ projection_revision: parseRevision(6) }),
  }));
  assert.deepEqual(revisionResult, { outcome: 'stale_selection' });
});

test('rejects a missing tab without returning a replacement', () => {
  const result = guardActivation(input({
    projection: projection({ records: new Map() }),
  }));
  assert.deepEqual(result, { outcome: 'tab_missing' });
});

test('rejects a reused tab id when the current result or identity differs', () => {
  const changedResult = guardActivation(input({
    projection: projection({
      records: new Map([[tabIdentityKey(identity()), record({ result_id: 'result-new' })]]),
    }),
  }));
  assert.deepEqual(changedResult, { outcome: 'tab_reused' });

  const changedIdentity = guardActivation(input({
    projection: projection({
      records: new Map([[tabIdentityKey(identity()), record({ tab_identity: identity({ tab_id: 8 }) })]]),
    }),
  }));
  assert.deepEqual(changedIdentity, { outcome: 'tab_reused' });
});

test('rejects a moved tab when the exact window no longer matches', () => {
  const result = guardActivation(input({
    reference: reference({ window_id: 3 }),
  }));
  assert.deepEqual(result, { outcome: 'window_missing' });
});

test('rejects an ineligible record', () => {
  const result = guardActivation(input({
    projection: projection({
      records: new Map([[tabIdentityKey(identity()), record({ eligible: false })]]),
    }),
  }));
  assert.deepEqual(result, { outcome: 'ineligible' });
});

test('rejects a record whose projection fence no longer matches', () => {
  const result = guardActivation(input({
    projection: projection({
      records: new Map([[tabIdentityKey(identity()), record({ projection_revision: parseRevision(6) })]]),
    }),
  }));
  assert.deepEqual(result, { outcome: 'stale_selection' });
});

test('rejects malformed references without throwing or exposing raw data', () => {
  const result = guardActivation(input({
    reference: reference({ tab_identity: null }),
  }));
  assert.deepEqual(result, { outcome: 'stale_selection' });
  assert.deepEqual(Object.keys(result), ['outcome']);
});

test('every rejected outcome is non-activatable and carries only a bounded outcome', () => {
  const rejected = [
    guardActivation(input({ reference: reference({ profile_id: OTHER_PROFILE }) })),
    guardActivation(input({ reference: reference({ context_kind: 'private' }) })),
    guardActivation(input({ projection: projection({ epoch: null }) })),
    guardActivation(input({ reference: reference({ query_sequence: 3 }) })),
    guardActivation(input({ reference: reference({ result_id: 'result-old' }) })),
    guardActivation(input({ reference: reference({ projection_epoch: OTHER_EPOCH }) })),
    guardActivation(input({ projection: projection({ records: new Map() }) })),
    guardActivation(input({ reference: reference({ window_id: 3 }) })),
    guardActivation(input({
      projection: projection({
        records: new Map([[tabIdentityKey(identity()), record({ eligible: false })]]),
      }),
    })),
  ];

  for (const result of rejected) {
    assert.notEqual(result.outcome, 'valid');
    assert.equal(isActivationGuardSuccess(result), false);
    assert.deepEqual(Object.keys(result), ['outcome']);
  }
});
