import assert from 'node:assert/strict';
import test from 'node:test';

import { QueryClient } from '../../../extension/dist/src/runtime/query-client.js';

const PROFILE_ID = '123e4567-e89b-42d3-a456-426614174000';

class DeferredTransport {
  revision = 42;
  submissions = [];
  resolvers = [];

  view() {
    return { acknowledged_revision: this.revision };
  }

  query(submission) {
    this.submissions.push(submission);
    return new Promise(resolve => {
      this.resolvers.push(resolve);
    });
  }

  resolve(index, result) {
    this.resolvers[index](result);
  }
}

function success(sequence, value = {}) {
  return {
    ok: true,
    value: {
      results: [{
        result_id: 'result-1',
        tab_id: 9,
        window_id: 2,
        display: {
          group_id: 4,
          title_display: 'Example',
          domain_display: 'example.test',
          window_label_display: 'Work',
          pinned: true,
          score: 900,
          best_class: 'exact_token',
        },
        explanations: ['title_match'],
      }],
      ranking_model_version: 'rank-v1',
      projection_revision: 42,
      projection_epoch: 'epoch-1',
      status: 'results',
      accepted_limit: 5,
      query_sequence: sequence,
      index_state: 'ready',
      persistence_state: 'healthy',
      freshness_age_ms: 0,
      timing: { total_micros: 2 },
      ...value,
    },
  };
}

function hostFailure(code, extra = {}) {
  return {
    ok: false,
    source: 'host',
    error: { code, retryable: true, message_key: code, ...extra },
  };
}

function clientFailure(code, retryable = true) {
  return { ok: false, source: 'client', error: { code, retryable } };
}

function makeClient() {
  const transport = new DeferredTransport();
  const client = new QueryClient({ profileId: PROFILE_ID, transport });
  return { client, transport };
}

test('search sends the latest context and renders bounded rows', async () => {
  const { client, transport } = makeClient();
  const seen = [];
  client.subscribe(model => seen.push(model.status));
  const pending = client.search({ query: 'alpha', resultLimit: 5, currentWindowId: 2, currentGroupId: 7 });

  assert.equal(transport.submissions.length, 1);
  assert.deepEqual(transport.submissions[0], {
    query: 'alpha',
    result_limit: 5,
    current_window_id: 2,
    current_group_id: 7,
    query_sequence: 1,
  });
  transport.resolve(0, success(1));
  const model = await pending;

  assert.equal(model.profileId, PROFILE_ID);
  assert.equal(model.sequence, 1);
  assert.equal(model.status, 'results');
  assert.equal(model.projectionRevision, 42);
  assert.equal(model.rankingModelVersion, 'rank-v1');
  assert.equal(model.acceptedLimit, 5);
  assert.equal(model.indexState, 'ready');
  assert.equal(model.persistenceState, 'healthy');
  assert.equal(model.rows.length, 1);
  assert.deepEqual(model.rows[0], {
    resultId: 'result-1',
    tabId: 9,
    windowId: 2,
    groupId: 4,
    title: 'Example',
    domain: 'example.test',
    windowLabel: 'Work',
    groupLabel: undefined,
    pinned: true,
    score: 900,
    bestClass: 'exact_token',
    explanations: ['title_match'],
  });
  assert.deepEqual(seen, ['loading', 'results']);
});

test('empty and no-result statuses remain distinct from unavailable', async () => {
  const { client, transport } = makeClient();
  const emptyPending = client.search({ query: '' });
  transport.resolve(0, success(1, { status: 'empty_query', results: [] }));
  const empty = await emptyPending;
  assert.equal(empty.status, 'empty_query');
  assert.deepEqual(empty.rows, []);

  const missingPending = client.search({ query: 'missing' });
  transport.resolve(1, success(2, { status: 'no_results', results: [] }));
  const missing = await missingPending;
  assert.equal(missing.status, 'no_results');
  assert.deepEqual(missing.rows, []);

  const unavailablePending = client.search({ query: 'alpha' });
  transport.resolve(2, clientFailure('HOST_UNAVAILABLE'));
  const unavailable = await unavailablePending;
  assert.equal(unavailable.status, 'unavailable');
  assert.deepEqual(unavailable.rows, []);
});

test('a superseded response cannot replace the latest visible result', async () => {
  const { client, transport } = makeClient();
  const first = client.search({ query: 'alpha' });
  const second = client.search({ query: 'beta' });

  transport.resolve(1, success(2, {
    results: [{
      result_id: 'result-2',
      tab_id: 10,
      window_id: 2,
      display: { title_display: 'Beta', domain_display: 'beta.test' },
      explanations: [],
    }],
  }));
  const latest = await second;
  assert.equal(latest.status, 'results');
  assert.equal(latest.rows[0].title, 'Beta');

  transport.resolve(0, success(1));
  const superseded = await first;
  assert.equal(superseded.status, 'superseded');
  assert.deepEqual(superseded.rows, []);
  assert.equal(client.view().rows[0].title, 'Beta');
});

test('a response with a stale sequence or revision is not rendered', async () => {
  const { client, transport } = makeClient();
  const wrongSequence = client.search({ query: 'alpha' });
  transport.resolve(0, success(2));
  const staleSequence = await wrongSequence;
  assert.equal(staleSequence.status, 'stale');
  assert.deepEqual(staleSequence.rows, []);

  const revisionChanged = client.search({ query: 'alpha' });
  transport.revision = 43;
  transport.resolve(1, success(2));
  const staleRevision = await revisionChanged;
  assert.equal(staleRevision.status, 'stale');
  assert.deepEqual(staleRevision.rows, []);
});

test('host failures map to bounded actionable statuses', async () => {
  const cases = [
    ['REVISION_MISMATCH', 'revision_mismatch', { expected_revision: 43 }],
    ['INDEX_REBUILDING', 'rebuilding', {}],
    ['QUERY_TIMEOUT', 'timeout', {}],
    ['PAYLOAD_LIMIT', 'invalid', {}],
    ['PERSISTENCE_DEGRADED', 'degraded', {}],
  ];
  for (const [code, status, extra] of cases) {
    const { client, transport } = makeClient();
    const pending = client.search({ query: 'alpha' });
    transport.resolve(0, hostFailure(code, extra));
    const model = await pending;
    assert.equal(model.status, status);
    assert.deepEqual(model.rows, []);
    if (extra.expected_revision !== undefined) {
      assert.equal(model.error.expectedRevision, extra.expected_revision);
    }
  }
});

test('cancel supersedes an in-flight request and ignores its late result', async () => {
  const { client, transport } = makeClient();
  const pending = client.search({ query: 'alpha' });
  const cancelled = client.cancel();
  assert.equal(cancelled.status, 'cancelled');
  assert.equal(cancelled.sequence, 2);

  transport.resolve(0, success(1));
  const late = await pending;
  assert.equal(late.status, 'superseded');
  assert.equal(client.view().status, 'cancelled');
});

test('invalid input is rejected locally without sending a query', async () => {
  const { client, transport } = makeClient();
  const model = await client.search({ query: 7 });
  assert.equal(model.status, 'invalid');
  assert.equal(model.error.source, 'client');
  assert.equal(transport.submissions.length, 0);
  assert.equal(client.view().sequence, 0);
});
