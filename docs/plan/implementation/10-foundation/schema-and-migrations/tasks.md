# Canonical schema and migrations: tasks

This file is the sole authoritative checklist for DB-MIGRATION. Do not check an item until its evidence anchor contains real results.

## DB-MIGRATION-001

**Title:** Create a versioned migration runner and target canonical schema

- [ ] **DB-MIGRATION-001: Create a versioned migration runner and target canonical schema**
- Outcome: Create a versioned migration runner and target canonical schema.
- Prerequisites: APP-LIFECYCLE.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: `app/db.py`, migration modules, repository modules, migration fixtures.
- Actions:
  1. Add migration metadata, target tables, partial unique indexes, foreign keys, active-job uniqueness, and transaction boundaries.
  2. Keep canonical writes transactional and derived work recoverable.
  3. Add tests with the implementation rather than deferring verification.
- Migration/compatibility: preserve canonical IDs and user data; do not broaden legacy or adapter behavior.
- Failure recovery: fail safely, retain canonical state, and record a stable error/checkpoint where applicable.
- Verification: Fresh-database schema tests and migration rollback tests.
- Complete when: implementation, tests, documentation impact, and [evidence](execution.md#db-migration-001) are reviewed.
- Commit boundary: one to three scoped commits.

## DB-MIGRATION-002

**Title:** Forward-migrate existing items and relationships

- [ ] **DB-MIGRATION-002: Forward-migrate existing items and relationships**
- Outcome: Forward-migrate existing items and relationships.
- Prerequisites: APP-LIFECYCLE.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: `app/db.py`, migration modules, repository modules, migration fixtures.
- Actions:
  1. Preserve integer IDs and personal context; map source kinds; create snapshot version 1 when content exists; record pre/post counts.
  2. Keep canonical writes transactional and derived work recoverable.
  3. Add tests with the implementation rather than deferring verification.
- Migration/compatibility: preserve canonical IDs and user data; do not broaden legacy or adapter behavior.
- Failure recovery: fail safely, retain canonical state, and record a stable error/checkpoint where applicable.
- Verification: Fixture migrations for URL, text, file, deleted, incomplete-content, and relationship cases.
- Complete when: implementation, tests, documentation impact, and [evidence](execution.md#db-migration-002) are reviewed.
- Commit boundary: one to three scoped commits.

## DB-MIGRATION-003

**Title:** Validate activation and rebuild derived state

- [ ] **DB-MIGRATION-003: Validate activation and rebuild derived state**
- Outcome: Validate activation and rebuild derived state.
- Prerequisites: APP-LIFECYCLE.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: `app/db.py`, migration modules, repository modules, migration fixtures.
- Actions:
  1. Abort active switch on mapping/checksum failure; treat chunks, FTS, jobs, and analytics as rebuild inputs rather than authority.
  2. Keep canonical writes transactional and derived work recoverable.
  3. Add tests with the implementation rather than deferring verification.
- Migration/compatibility: preserve canonical IDs and user data; do not broaden legacy or adapter behavior.
- Failure recovery: fail safely, retain canonical state, and record a stable error/checkpoint where applicable.
- Verification: Failure-injection and canonical-count verification tests.
- Complete when: implementation, tests, documentation impact, and [evidence](execution.md#db-migration-003) are reviewed.
- Commit boundary: one to three scoped commits.
