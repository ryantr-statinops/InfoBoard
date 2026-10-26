import { expect, test } from '@playwright/test';
import {
  canActivate,
  initialSurfaceState,
  permittedActions,
  reduceSurfaceState,
  selectedRow,
  statusMessage,
  type SurfaceState,
} from '../../../extension/src/search/surface-state.js';
import type { QueryRenderModel, QueryRenderRow } from '../../../extension/src/runtime/query-client.js';

const profileId = '123e4567-e89b-42d3-a456-426614174000';

const row = (id: string, overrides: Partial<QueryRenderRow> = {}): QueryRenderRow => ({
  resultId: id,
  tabId: 1,
  windowId: 1,
  title: `Tab ${id}`,
  domain: 'example.test',
  pinned: false,
  explanations: ['title match'],
  ...overrides,
});

const model = (overrides: Partial<QueryRenderModel> = {}): QueryRenderModel => ({
  profileId,
  sequence: 1,
  status: 'results',
  rows: [row('one')],
  projectionRevision: 4,
  ...overrides,
});

function querying(): SurfaceState {
  return reduceSurfaceState(initialSurfaceState(), { type: 'query', query: 'tab', sequence: 1 });
}

test('opening becomes ready and exposes only safe actions', () => {
  const opening = initialSurfaceState();
  expect(opening.status).toBe('opening');
  expect(opening.browserStateConfirmed).toBe(true);
  expect(permittedActions(opening)).toEqual(['focus-input', 'dismiss']);

  const ready = reduceSurfaceState(opening, { type: 'ready', focusTimestamp: 12, shortcutFocusMs: 42 });
  expect(ready.status).toBe('ready');
  expect(ready.shortcutFocusMs).toBe(42);
  expect(statusMessage(ready)).toBe('Type to search open tabs');
});

test('query clears old rows before a matching response is accepted', () => {
  const results = reduceSurfaceState(querying(), { type: 'model', model: model() });
  expect(results.status).toBe('results');
  expect(results.rows).toHaveLength(1);
  expect(canActivate(results)).toBe(true);
  expect(selectedRow(results)?.resultId).toBe('one');

  const next = reduceSurfaceState(results, { type: 'query', query: 'next', sequence: 2 });
  expect(next.status).toBe('querying');
  expect(next.rows).toEqual([]);
  expect(next.selectedIndex).toBe(-1);
  expect(canActivate(next)).toBe(false);
});

test('late responses from older revisions are ignored', () => {
  const second = reduceSurfaceState(querying(), {
    type: 'model',
    model: model({ sequence: 2, rows: [row('two')] }),
  });
  const lateFirst = reduceSurfaceState(second, {
    type: 'model',
    model: model({ sequence: 1, rows: [row('one')] }),
  });
  expect(lateFirst).toBe(second);
  expect(selectedRow(lateFirst)?.resultId).toBe('two');
});

test('empty, no-result, rebuilding, unavailable and stale states stay distinct', () => {
  const empty = reduceSurfaceState(querying(), {
    type: 'model',
    model: model({ status: 'empty_query', rows: [row('recent')] }),
  });
  expect(empty.status).toBe('empty_query');
  expect(statusMessage(empty)).toBe('Recent and related tabs');

  const none = reduceSurfaceState(querying(), {
    type: 'model',
    model: model({ status: 'no_results', rows: [] }),
  });
  expect(none.status).toBe('no_results');
  expect(statusMessage(none)).toBe('No matching open tabs');

  const rebuilding = reduceSurfaceState(querying(), {
    type: 'model',
    model: model({ status: 'rebuilding', rows: [] }),
  });
  expect(rebuilding.status).toBe('rebuilding');
  expect(permittedActions(rebuilding)).toContain('retry');
  expect(canActivate(rebuilding)).toBe(false);

  const unavailable = reduceSurfaceState(querying(), {
    type: 'model',
    model: model({ status: 'unavailable', rows: [], error: { source: 'client', code: 'HOST_UNAVAILABLE', retryable: true } }),
  });
  expect(unavailable.status).toBe('unavailable');
  expect(unavailable.errorCode).toBe('HOST_UNAVAILABLE');

  const stale = reduceSurfaceState(querying(), {
    type: 'model',
    model: model({ status: 'stale', rows: [] }),
  });
  expect(stale.status).toBe('stale');
  expect(canActivate(stale)).toBe(false);
});

test('persistence degradation keeps lexical rows actionable', () => {
  const degraded = reduceSurfaceState(querying(), {
    type: 'model',
    model: model({ status: 'degraded', persistenceState: 'degraded', rows: [row('one')] }),
  });
  expect(degraded.status).toBe('persistence_degraded');
  expect(degraded.persistenceState).toBe('degraded');
  expect(canActivate(degraded)).toBe(true);
  expect(statusMessage(degraded)).toContain('History is unavailable');
});

test('activation is explicit, confirmed and never guessed', () => {
  const results = reduceSurfaceState(querying(), { type: 'model', model: model() });
  const pending = reduceSurfaceState(results, { type: 'activation_started' });
  expect(pending.activation.pending).toBe(true);
  expect(pending.browserStateConfirmed).toBe(false);
  expect(canActivate(pending)).toBe(false);

  const failed = reduceSurfaceState(pending, {
    type: 'activation_result',
    ok: false,
    confirmed: true,
    errorCode: 'tab_missing',
  });
  expect(failed.status).toBe('activation_failure');
  expect(failed.browserStateConfirmed).toBe(true);
  expect(failed.activation.errorCode).toBe('tab_missing');
  expect(permittedActions(failed)).toContain('retry');
});

test('private and dismissed surfaces expose no query or activation action', () => {
  const privateState = reduceSurfaceState(initialSurfaceState(), { type: 'private_unavailable' });
  expect(privateState.status).toBe('private_unavailable');
  expect(permittedActions(privateState)).toEqual(['dismiss']);

  const dismissed = reduceSurfaceState(querying(), { type: 'dismiss' });
  expect(dismissed.dismissed).toBe(true);
  expect(dismissed.browserStateConfirmed).toBe(true);
  expect(permittedActions(dismissed)).toEqual([]);
});

test('status announcements never echo query text or URLs', () => {
  const sensitive = reduceSurfaceState(initialSurfaceState(), {
    type: 'query',
    query: 'secret-token?auth=abc',
    sequence: 1,
  });
  const states: SurfaceState[] = [
    sensitive,
    reduceSurfaceState(sensitive, { type: 'model', model: model({ status: 'no_results', rows: [] }) }),
    reduceSurfaceState(sensitive, { type: 'model', model: model({ status: 'stale', rows: [] }) }),
    reduceSurfaceState(sensitive, {
      type: 'model',
      model: model({ status: 'unavailable', rows: [], error: { source: 'host', code: 'HOST_SHUTDOWN', retryable: true } }),
    }),
  ];
  for (const state of states) {
    expect(statusMessage(state)).not.toContain('secret-token');
    expect(statusMessage(state)).not.toContain('auth=');
    expect(statusMessage(state)).not.toContain('http');
  }
});
