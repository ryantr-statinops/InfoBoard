# Canonical API foundation: tasks

This file is the sole authoritative checklist for API-CANONICAL. Do not check an item until its evidence anchor contains real results.

## API-CANONICAL-001

**Title:** Build shared API schemas, validation, cursors, and error handling

- [ ] **API-CANONICAL-001: Build shared API schemas, validation, cursors, and error handling**
- Outcome: Build shared API schemas, validation, cursors, and error handling.
- Prerequisites: APP-LIFECYCLE, DB-MIGRATION.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: `app/main.py`, API router/schema/error modules, service and HTTP contract tests.
- Actions:
  1. Implement versioned opaque cursors, common envelopes, limits, filter semantics, correlation IDs, and safe error registry mapping.
  2. Keep canonical writes transactional and derived work recoverable.
  3. Add tests with the implementation rather than deferring verification.
- Migration/compatibility: preserve canonical IDs and user data; do not broaden legacy or adapter behavior.
- Failure recovery: fail safely, retain canonical state, and record a stable error/checkpoint where applicable.
- Verification: Schema, boundary, unknown-field, cursor-tamper, and redaction contract tests.
- Complete when: implementation, tests, documentation impact, and [evidence](execution.md#api-canonical-001) are reviewed.
- Commit boundary: one to three scoped commits.

## API-CANONICAL-002

**Title:** Implement optimistic mutation conventions

- [ ] **API-CANONICAL-002: Implement optimistic mutation conventions**
- Outcome: Implement optimistic mutation conventions.
- Prerequisites: APP-LIFECYCLE, DB-MIGRATION.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: `app/main.py`, API router/schema/error modules, service and HTTP contract tests.
- Actions:
  1. Require integer `If-Match`, increment resource versions for owned mutations, and return stable stale-write conflicts.
  2. Keep canonical writes transactional and derived work recoverable.
  3. Add tests with the implementation rather than deferring verification.
- Migration/compatibility: preserve canonical IDs and user data; do not broaden legacy or adapter behavior.
- Failure recovery: fail safely, retain canonical state, and record a stable error/checkpoint where applicable.
- Verification: Concurrent update, membership mutation, missing precondition, and conflict tests.
- Complete when: implementation, tests, documentation impact, and [evidence](execution.md#api-canonical-002) are reviewed.
- Commit boundary: one to three scoped commits.

## API-CANONICAL-003

**Title:** Separate canonical routers from compatibility adapters

- [ ] **API-CANONICAL-003: Separate canonical routers from compatibility adapters**
- Outcome: Separate canonical routers from compatibility adapters.
- Prerequisites: APP-LIFECYCLE, DB-MIGRATION.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: `app/main.py`, API router/schema/error modules, service and HTTP contract tests.
- Actions:
  1. Mount `/api/v1/*` as canonical services and ensure `/api/*` adapters cannot own business logic or unique behavior.
  2. Keep canonical writes transactional and derived work recoverable.
  3. Add tests with the implementation rather than deferring verification.
- Migration/compatibility: preserve canonical IDs and user data; do not broaden legacy or adapter behavior.
- Failure recovery: fail safely, retain canonical state, and record a stable error/checkpoint where applicable.
- Verification: Router ownership and canonical/adapter parity tests.
- Complete when: implementation, tests, documentation impact, and [evidence](execution.md#api-canonical-003) are reviewed.
- Commit boundary: one to three scoped commits.
