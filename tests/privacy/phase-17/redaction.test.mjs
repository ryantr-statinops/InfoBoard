import assert from 'node:assert/strict';
import test from 'node:test';
import { readFile } from 'node:fs/promises';

import {
  assertNoSentinels,
  displayURL,
  localDigest,
  redactDiagnosticRecord,
  safeError,
} from '../../../extension/dist/src/privacy/redaction.js';

const fixture = JSON.parse(await readFile('fixtures/privacy/phase-17/redaction-sentinels.json', 'utf8'));
const diagnostics = JSON.parse(await readFile('fixtures/privacy/phase-17/output/diagnostics.json', 'utf8'));
const exported = JSON.parse(await readFile('fixtures/privacy/phase-17/output/export.json', 'utf8'));

test('diagnostic redaction drops unknown fields and every sentinel value', () => {
  const output = redactDiagnosticRecord({
    ...fixture.context,
    event_id: fixture.sensitive_values[0],
    session_id: fixture.sensitive_values[5],
    error_class: fixture.sensitive_values[8],
    component: fixture.sensitive_values[3],
    counts: { records: 1, dropped: 0, retries: 0, failures: 0, leaked: fixture.sensitive_values[0] },
  });
  assertNoSentinels(output, fixture.sensitive_values);
  assert.deepEqual(output.counts, { records: 1, dropped: 0, retries: 0, failures: 0 });
  assert.equal('title_display' in output, false);
  assert.equal('nested' in output, false);
});

test('display URLs remove credentials, query, and fragment', () => {
  const display = displayURL(fixture.context.url_display);
  assert.equal(display, 'https://example.test/path');
  assertNoSentinels(display, fixture.sensitive_values);
});

test('safe errors and local digests contain no raw metadata', async () => {
  const safe = safeError(fixture.sensitive_values[8], true, fixture.sensitive_values[3]);
  assertNoSentinels(safe, fixture.sensitive_values);
  const digest = await localDigest(fixture.context.private_id);
  assert.match(digest, /^local-sha256:[0-9a-f]{16}$/);
  assert.equal(digest.includes(fixture.context.private_id), false);
});

test('committed outputs contain no sentinel and the guard fails closed', () => {
  assertNoSentinels(diagnostics, fixture.sensitive_values);
  assertNoSentinels(exported, fixture.sensitive_values);
  assert.throws(
    () => assertNoSentinels({ leaked: fixture.sensitive_values[0] }, fixture.sensitive_values),
    /SENSITIVE_OUTPUT/,
  );
});
