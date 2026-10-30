import assert from 'node:assert/strict';
import test from 'node:test';
import { readFile } from 'node:fs/promises';

import {
  FIELD_CLASSES,
  FIELD_POLICY,
  assertClassifiedField,
  assertNoForbiddenFields,
  classifyField,
  isForbiddenField,
} from '../../../extension/dist/src/privacy/field-classes.js';

const fixture = JSON.parse(await readFile('fixtures/privacy/phase-17/field-classes.json', 'utf8'));

test('field-class matrix matches the shared contract fixture', () => {
  assert.deepEqual(FIELD_CLASSES, fixture.classes);
  assert.deepEqual(FIELD_POLICY, fixture.fields);
  assert.equal(new Set(FIELD_POLICY.map((policy) => policy.field)).size, FIELD_POLICY.length);
});

test('every field is classified and forbidden fields are rejected', () => {
  for (const policy of fixture.fields) {
    assert.equal(classifyField(policy.field)?.field_class, policy.field_class);
    if (policy.field_class === 'forbidden') {
      assert.equal(isForbiddenField(policy.field), true);
      assert.throws(() => assertNoForbiddenFields([policy.field]), /FORBIDDEN_FIELD/);
    } else {
      assert.equal(assertClassifiedField(policy.field).field, policy.field);
    }
  }
  assert.equal(classifyField('unclassified_sentinel'), undefined);
  assert.throws(() => assertClassifiedField('unclassified_sentinel'), /UNCLASSIFIED_FIELD/);
});
