# Legacy item preservation: tasks

This is the sole authoritative checklist for ORG-LEGACY. Evidence is mandatory before checking a task.

## ORG-LEGACY-001

**Title:** Project legacy sources through bookmark reads

- [ ] **ORG-LEGACY-001: Project legacy sources through bookmark reads**
- Outcome: Project legacy sources through bookmark reads.
- Prerequisites: DB-MIGRATION, CAP-SNAPSHOT, ORG-CONTEXT.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: migration projections, bookmark services/API, legacy fixtures, UI state tests.
- Actions:
  1. Return source_kind, available metadata/content, organization, notes, and current snapshot while preserving integer IDs.
  2. Preserve optimistic versions, canonical ownership, and recoverable derived work.
  3. Add the required tests in the same delivery slice.
- Migration/compatibility: preserve IDs, personal context, and documented adapter behavior.
- Failure recovery: reject unsafe or stale work without damaging canonical state; expose a stable safe error.
- Verification: Text/file/no-content projections and list/detail/search parity tests.
- Complete when: code, tests, documentation impact, and [evidence](execution.md#org-legacy-001) are reviewed.
- Commit boundary: one to three scoped commits.

## ORG-LEGACY-002

**Title:** Disable unsupported legacy creation and capture

- [ ] **ORG-LEGACY-002: Disable unsupported legacy creation and capture**
- Outcome: Disable unsupported legacy creation and capture.
- Prerequisites: DB-MIGRATION, CAP-SNAPSHOT, ORG-CONTEXT.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: migration projections, bookmark services/API, legacy fixtures, UI state tests.
- Actions:
  1. Remove maintained UI creation paths; return stable 410 for old writes and 409 for capture/recapture without hiding existing items.
  2. Preserve optimistic versions, canonical ownership, and recoverable derived work.
  3. Add the required tests in the same delivery slice.
- Migration/compatibility: preserve IDs, personal context, and documented adapter behavior.
- Failure recovery: reject unsafe or stale work without damaging canonical state; expose a stable safe error.
- Verification: Route, error-envelope, UI-action absence, and adapter tests.
- Complete when: code, tests, documentation impact, and [evidence](execution.md#org-legacy-002) are reviewed.
- Commit boundary: one to three scoped commits.

## ORG-LEGACY-003

**Title:** Preserve organization and soft-delete behavior

- [ ] **ORG-LEGACY-003: Preserve organization and soft-delete behavior**
- Outcome: Preserve organization and soft-delete behavior.
- Prerequisites: DB-MIGRATION, CAP-SNAPSHOT, ORG-CONTEXT.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: migration projections, bookmark services/API, legacy fixtures, UI state tests.
- Actions:
  1. Allow supported metadata, membership, note, status, and soft-delete operations with optimistic concurrency.
  2. Preserve optimistic versions, canonical ownership, and recoverable derived work.
  3. Add the required tests in the same delivery slice.
- Migration/compatibility: preserve IDs, personal context, and documented adapter behavior.
- Failure recovery: reject unsafe or stale work without damaging canonical state; expose a stable safe error.
- Verification: Mutation, deletion exclusion, and no-permanent-delete tests.
- Complete when: code, tests, documentation impact, and [evidence](execution.md#org-legacy-003) are reviewed.
- Commit boundary: one to three scoped commits.
