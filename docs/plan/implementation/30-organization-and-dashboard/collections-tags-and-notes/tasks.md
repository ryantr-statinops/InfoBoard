# Collections, tags, notes, and bookmark state: tasks

This is the sole authoritative checklist for ORG-CONTEXT. Evidence is mandatory before checking a task.

## ORG-CONTEXT-001

**Title:** Implement collection and tag resources

- [ ] **ORG-CONTEXT-001: Implement collection and tag resources**
- Outcome: Implement collection and tag resources.
- Prerequisites: DB-MIGRATION, API-CANONICAL.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: organization repositories/services, API routers and schemas, membership tests.
- Actions:
  1. Add normalized uniqueness, list/create/update/delete behavior, integer versions, and safe membership cleanup.
  2. Preserve optimistic versions, canonical ownership, and recoverable derived work.
  3. Add the required tests in the same delivery slice.
- Migration/compatibility: preserve IDs, personal context, and documented adapter behavior.
- Failure recovery: reject unsafe or stale work without damaging canonical state; expose a stable safe error.
- Verification: CRUD, normalization, duplicate, If-Match, and deletion tests.
- Complete when: code, tests, documentation impact, and [evidence](execution.md#org-context-001) are reviewed.
- Commit boundary: one to three scoped commits.

## ORG-CONTEXT-002

**Title:** Implement versioned bookmark memberships and status

- [ ] **ORG-CONTEXT-002: Implement versioned bookmark memberships and status**
- Outcome: Implement versioned bookmark memberships and status.
- Prerequisites: DB-MIGRATION, API-CANONICAL.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: organization repositories/services, API routers and schemas, membership tests.
- Actions:
  1. Apply idempotent collection/tag mutations through bookmark version guards and support inbox, active, and archived.
  2. Preserve optimistic versions, canonical ownership, and recoverable derived work.
  3. Add the required tests in the same delivery slice.
- Migration/compatibility: preserve IDs, personal context, and documented adapter behavior.
- Failure recovery: reject unsafe or stale work without damaging canonical state; expose a stable safe error.
- Verification: Idempotency, concurrent membership, status transition, and deleted-resource tests.
- Complete when: code, tests, documentation impact, and [evidence](execution.md#org-context-002) are reviewed.
- Commit boundary: one to three scoped commits.

## ORG-CONTEXT-003

**Title:** Implement independent personal notes

- [ ] **ORG-CONTEXT-003: Implement independent personal notes**
- Outcome: Implement independent personal notes.
- Prerequisites: DB-MIGRATION, API-CANONICAL.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: organization repositories/services, API routers and schemas, membership tests.
- Actions:
  1. Create, paginate, update, and soft-delete bookmark-owned notes without coupling their content to capture or indexing.
  2. Preserve optimistic versions, canonical ownership, and recoverable derived work.
  3. Add the required tests in the same delivery slice.
- Migration/compatibility: preserve IDs, personal context, and documented adapter behavior.
- Failure recovery: reject unsafe or stale work without damaging canonical state; expose a stable safe error.
- Verification: Ownership, body-limit, concurrency, recapture-preservation, and deletion tests.
- Complete when: code, tests, documentation impact, and [evidence](execution.md#org-context-003) are reviewed.
- Commit boundary: one to three scoped commits.
