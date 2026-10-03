import { test, expect } from '@playwright/test';
import { createProjectionEpoch, parseProfileID, parseRevision } from '../../extension/domain/index.js';
import { evaluateTabEligibility, type EligibilityContext } from '../../extension/src/browser/eligibility.js';

const profileId = parseProfileID('123e4567-e89b-42d3-a456-426614174000');
const windows = [{ id: 1, incognito: false, title: 'Work' }];

function context(overrides: Partial<EligibilityContext> = {}): EligibilityContext {
  return {
    profileId,
    contextKind: 'normal',
    projectionEpoch: createProjectionEpoch(),
    projectionRevision: parseRevision(1),
    observedAt: 100,
    windows,
    groups: [],
    ...overrides,
  };
}

const normalTab = {
  id: 10,
  windowId: 1,
  title: 'Example',
  url: 'https://user:secret@example.test/path?q=search#section',
  pinned: false,
  active: true,
  incognito: false,
};

test('eligible tabs produce a profile/context identity and safe display URL', () => {
  const result = evaluateTabEligibility(normalTab, context());
  expect(result.decision).toBe('eligible');
  if (result.decision !== 'eligible') return;
  expect(result.identity).toEqual({ profile_id: profileId, context_kind: 'normal', tab_id: 10 });
  expect(result.record.domain_display).toBe('example.test');
  expect(result.record.url_display).toBe('https://example.test/path');
  expect(result.record.url_search).not.toContain('user:secret@');
});

test('missing identity, required state, and mismatched context are deferred', () => {
  expect(evaluateTabEligibility({ ...normalTab, id: -1 }, context())).toMatchObject({ decision: 'deferred', reason: 'identity_invalid' });
  expect(evaluateTabEligibility({ ...normalTab, title: undefined }, context())).toMatchObject({ decision: 'deferred', reason: 'required_field_missing' });
  expect(evaluateTabEligibility(normalTab, context({ contextKind: 'private' })).reason).toBe('context_mismatch');
  expect(evaluateTabEligibility({ ...normalTab, windowId: 99 }, context()).reason).toBe('window_unavailable');
});

test('invalid and unsupported URLs are excluded with distinct safe reasons', () => {
  expect(evaluateTabEligibility({ ...normalTab, url: 'not a URL' }, context())).toMatchObject({ decision: 'excluded', reason: 'invalid_url' });
  expect(evaluateTabEligibility({ ...normalTab, url: 'file:///tmp/local.txt' }, context())).toMatchObject({ decision: 'excluded', reason: 'unsupported_scheme' });
});

test('missing optional group metadata preserves the eligible tab without a label', () => {
  const tab = { ...normalTab, groupId: 7 };
  const result = evaluateTabEligibility(tab, context());
  expect(result.decision).toBe('eligible');
  if (result.decision !== 'eligible') return;
  expect(result.record.group_id).toBe(7);
  expect(result.record.group_label_display).toBeUndefined();
});

test('private eligibility is partitioned and marked for private-context teardown', () => {
  const privateWindow = { id: 2, incognito: true };
  const result = evaluateTabEligibility({ ...normalTab, id: 20, windowId: 2, incognito: true }, context({
    contextKind: 'private',
    windows: [privateWindow],
  }));
  expect(result.decision).toBe('eligible');
  if (result.decision !== 'eligible') return;
  expect(result.record.context_kind).toBe('private');
  expect(result.removeOnPrivateContextEnd).toBe(true);
});
