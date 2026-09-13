# Keyword query orchestration: tasks

This is the sole authoritative checklist for RET-KEYWORD. Evidence is mandatory before checking a task.

## RET-KEYWORD-001

**Title:** Generate bounded keyword candidates

- [ ] **RET-KEYWORD-001: Generate bounded keyword candidates**
- Outcome: Generate bounded keyword candidates.
- Prerequisites: API-CANONICAL, RET-FTS, ORG-CONTEXT.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: search service, FTS query adapter, `/api/v1/search`, dashboard search, relevance fixtures.
- Actions:
  1. Parse/bind FTS input safely, apply collection/tag/status/domain/date filters, exclude deleted/stale versions, and cap candidate work.
  2. Preserve optimistic versions, canonical ownership, and recoverable derived work.
  3. Add the required tests in the same delivery slice.
- Migration/compatibility: preserve IDs, personal context, and documented adapter behavior.
- Failure recovery: reject unsafe or stale work without damaging canonical state; expose a stable safe error.
- Verification: Injection-like input, empty query, filter semantics, stale/deleted, and limit tests.
- Complete when: code, tests, documentation impact, and [evidence](execution.md#ret-keyword-001) are reviewed.
- Commit boundary: one to three scoped commits.

## RET-KEYWORD-002

**Title:** Project ranked bookmark results and excerpts

- [ ] **RET-KEYWORD-002: Project ranked bookmark results and excerpts**
- Outcome: Project ranked bookmark results and excerpts.
- Prerequisites: API-CANONICAL, RET-FTS, ORG-CONTEXT.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: search service, FTS query adapter, `/api/v1/search`, dashboard search, relevance fixtures.
- Actions:
  1. Deduplicate by bookmark, return stable requested/actual mode, safe excerpts, context, sort, and opaque cursor behavior.
  2. Preserve optimistic versions, canonical ownership, and recoverable derived work.
  3. Add the required tests in the same delivery slice.
- Migration/compatibility: preserve IDs, personal context, and documented adapter behavior.
- Failure recovery: reject unsafe or stale work without damaging canonical state; expose a stable safe error.
- Verification: Ranking fixtures, excerpt escaping, pagination stability, and legacy-result tests.
- Complete when: code, tests, documentation impact, and [evidence](execution.md#ret-keyword-002) are reviewed.
- Commit boundary: one to three scoped commits.

## RET-KEYWORD-003

**Title:** Integrate API and UI without provider coupling

- [ ] **RET-KEYWORD-003: Integrate API and UI without provider coupling**
- Outcome: Integrate API and UI without provider coupling.
- Prerequisites: API-CANONICAL, RET-FTS, ORG-CONTEXT.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: search service, FTS query adapter, `/api/v1/search`, dashboard search, relevance fixtures.
- Actions:
  1. Make keyword mode fully functional when semantic components are unconfigured or unavailable and report only relevant degradation.
  2. Preserve optimistic versions, canonical ownership, and recoverable derived work.
  3. Add the required tests in the same delivery slice.
- Migration/compatibility: preserve IDs, personal context, and documented adapter behavior.
- Failure recovery: reject unsafe or stale work without damaging canonical state; expose a stable safe error.
- Verification: Provider-absent, Chroma-down, UI flow, API schema, and latency baseline tests.
- Complete when: code, tests, documentation impact, and [evidence](execution.md#ret-keyword-003) are reviewed.
- Commit boundary: one to three scoped commits.
