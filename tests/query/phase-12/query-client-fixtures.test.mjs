import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

import { QueryClient } from '../../../extension/dist/src/runtime/query-client.js';

const fixtures = JSON.parse(await readFile('fixtures/query/phase-12/QUERY-002-live-input-revisions.json', 'utf8'));
const recovery = JSON.parse(await readFile('fixtures/query/phase-12/QUERY-008-host-down-recovery.json', 'utf8'));

class DeferredTransport {
  revision = 42;
  submissions = [];
  resolvers = [];

  view() {
    return { acknowledged_revision: this.revision };
  }

  query(submission) {
    this.submissions.push(submission);
    return new Promise(resolve => this.resolvers.push(resolve));
  }

  resolve(index, result) {
    this.resolvers[index](result);
  }
}

function success(sequence, revision = 42) {
  return {
    ok: true,
    value: {
      results: [{
        result_id: `result-${sequence}`,
        tab_id: 10,
        window_id: 1,
        display: { title_display: 'Example', domain_display: 'example.test' },
        explanations: ['title_match'],
      }],
      ranking_model_version: 'rank-v1',
      projection_revision: revision,
      projection_epoch: 'epoch-1',
      status: 'results',
      accepted_limit: 5,
      query_sequence: sequence,
      index_state: 'ready',
      persistence_state: 'healthy',
      freshness_age_ms: 0,
      timing: { total_micros: 1 },
    },
  };
}

test('FX-QUERY-LIVE-REVISIONS keeps the latest accepted query visible', async () => {
  const transport = new DeferredTransport();
  const client = new QueryClient({ profileId: fixtures.input.profile_id, transport });
  const first = client.search({ query: fixtures.input.cases[0].query });
  const second = client.search({ query: fixtures.input.cases[1].query });

  assert.equal(transport.submissions[0].query_sequence, fixtures.input.cases[0].query_sequence);
  assert.equal(transport.submissions[1].query_sequence, fixtures.input.cases[1].query_sequence);

  transport.resolve(1, success(2));
  const latest = await second;
  assert.equal(latest.status, 'results');
  assert.equal(latest.sequence, 2);

  transport.resolve(0, success(1));
  const superseded = await first;
  assert.equal(superseded.status, 'superseded');
  assert.deepEqual(superseded.rows, []);
  assert.equal(client.view().sequence, 2);
  assert.equal(client.view().rows[0].resultId, 'result-2');
});

test('FX-QUERY-HOST-DOWN-RECOVERY never claims rows before recovery', async () => {
  const transport = new DeferredTransport();
  const client = new QueryClient({ profileId: recovery.input.profile_id, transport });
  const down = client.search({ query: recovery.input.query });
  transport.resolve(0, {
    ok: false,
    source: 'client',
    error: { code: 'HOST_UNAVAILABLE', retryable: true },
  });
  const unavailable = await down;
  assert.equal(unavailable.status, 'unavailable');
  assert.equal(unavailable.error.retryable, true);
  assert.deepEqual(unavailable.rows, []);

  transport.revision = recovery.input.recovery_revision;
  const recovered = client.search({ query: recovery.input.query });
  transport.resolve(1, success(2, recovery.input.recovery_revision));
  const model = await recovered;
  assert.equal(model.status, 'results');
  assert.equal(model.projectionRevision, recovery.input.recovery_revision);
  assert.equal(model.rows.length, 1);
});
