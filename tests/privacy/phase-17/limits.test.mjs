import assert from 'node:assert/strict';
import test from 'node:test';
import { readFile } from 'node:fs/promises';

import {
  OPERATING_TAB_LIMIT,
  detectSecret,
  rejectSecrets,
  validateDeltaCount,
  validateFrameBytes,
  validateHTTPURL,
  validatePayloadBytes,
  validateQueryInput,
  validateSnapshotCount,
} from '../../../extension/dist/src/privacy/limits.js';

const limitsFixture = JSON.parse(await readFile('fixtures/privacy/phase-17/input-limits.json', 'utf8'));
const secretFixture = JSON.parse(await readFile('fixtures/privacy/phase-17/secret-rejection.json', 'utf8'));

test('input limits match the shared fixture and fail closed', () => {
  assert.equal(limitsFixture.operating_tab_limit, OPERATING_TAB_LIMIT);
  for (const testCase of limitsFixture.cases) {
    let result;
    switch (testCase.kind) {
      case 'frame_bytes':
        result = validateFrameBytes(testCase.observed);
        break;
      case 'payload_bytes':
        result = validatePayloadBytes(testCase.observed);
        break;
      case 'query_scalars':
        result = validateQueryInput({ query: 'a'.repeat(testCase.observed), currentWindowId: 1 });
        break;
      case 'result_limit':
        result = validateQueryInput({ query: 'go', resultLimit: testCase.observed, currentWindowId: 1 });
        break;
      case 'snapshot_tabs':
        result = validateSnapshotCount(testCase.observed);
        break;
      case 'delta_operations':
        result = validateDeltaCount(testCase.observed);
        break;
      case 'negative_window':
        result = validateQueryInput({ query: 'go', currentWindowId: testCase.observed });
        break;
      case 'malformed_url':
        result = validateHTTPURL(testCase.observed);
        break;
      case 'valid_query':
        result = validateQueryInput({ query: 'a'.repeat(testCase.observed), currentWindowId: 1 });
        break;
      default:
        assert.fail(`unknown case ${testCase.kind}`);
    }
    if (testCase.expected === 'ok') {
      assert.equal(result.ok, true, testCase.case_id);
    } else {
      assert.equal(result.ok, false, testCase.case_id);
      assert.equal(result.code, testCase.expected, testCase.case_id);
      if (testCase.bound !== undefined) assert.equal(result.bound, testCase.bound, testCase.case_id);
    }
  }
  assert.equal(validateHTTPURL('https://example.test/path').ok, true);
  assert.equal(validateHTTPURL('not a url').ok, false);
});

test('secret-shaped fields and values are rejected without echoing the value', () => {
  for (const testCase of secretFixture.cases) {
    const findings = detectSecret(testCase.field, testCase.value);
    assert.ok(findings.some((finding) => finding.kind === testCase.expected_kind), testCase.case_id);
    assert.equal(JSON.stringify(findings).includes(testCase.value), false, testCase.case_id);
  }
  const fields = Object.fromEntries(secretFixture.cases.map((testCase) => [testCase.field, testCase.value]));
  const findings = rejectSecrets(fields);
  assert.ok(findings.length >= secretFixture.cases.length);
  assert.equal(JSON.stringify(findings).includes('SENTINEL'), false);
});
