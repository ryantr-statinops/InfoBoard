# Dashboard and UI states: tasks

This is the sole authoritative checklist for ORG-UI. Evidence is mandatory before checking a task.

## ORG-UI-001 — Build library and detail information architecture

- [ ] **ORG-UI-001: Build library and detail information architecture**
- Outcome: Build library and detail information architecture.
- Prerequisites: API-CANONICAL, CAP-SNAPSHOT, ORG-CONTEXT, ORG-LEGACY.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: Jinja templates, static assets, UI routes, browser/e2e tests.
- Actions:
  1. Implement navigation, filters, summaries, detail context, organization controls, notes, and capture state without storage jargon.
  2. Preserve optimistic versions, canonical ownership, and recoverable derived work.
  3. Add the required tests in the same delivery slice.
- Migration/compatibility: preserve IDs, personal context, and documented adapter behavior.
- Failure recovery: reject unsafe or stale work without damaging canonical state; expose a stable safe error.
- Verification: Keyboard, focus, empty/data, navigation, and 1440/390 layout tests.
- Complete when: code, tests, documentation impact, and [evidence](execution.md#org-ui-001) are reviewed.
- Commit boundary: one to three scoped commits.

## ORG-UI-002 — Project asynchronous and degraded states safely

- [ ] **ORG-UI-002: Project asynchronous and degraded states safely**
- Outcome: Project asynchronous and degraded states safely.
- Prerequisites: API-CANONICAL, CAP-SNAPSHOT, ORG-CONTEXT, ORG-LEGACY.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: Jinja templates, static assets, UI routes, browser/e2e tests.
- Actions:
  1. Show pending/processing/failed snapshot, retry, stale-write conflict, provider unconfigured/no-consent, degraded component, and next action.
  2. Preserve optimistic versions, canonical ownership, and recoverable derived work.
  3. Add the required tests in the same delivery slice.
- Migration/compatibility: preserve IDs, personal context, and documented adapter behavior.
- Failure recovery: reject unsafe or stale work without damaging canonical state; expose a stable safe error.
- Verification: State fixture screenshots, action availability, error copy, and retry flow tests.
- Complete when: code, tests, documentation impact, and [evidence](execution.md#org-ui-002) are reviewed.
- Commit boundary: one to three scoped commits.

## ORG-UI-003 — Represent legacy and destructive behavior

- [ ] **ORG-UI-003: Represent legacy and destructive behavior**
- Outcome: Represent legacy and destructive behavior.
- Prerequisites: API-CANONICAL, CAP-SNAPSHOT, ORG-CONTEXT, ORG-LEGACY.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: Jinja templates, static assets, UI routes, browser/e2e tests.
- Actions:
  1. Label legacy source constraints, hide unsupported capture, and confirm soft-delete without exposing purge/restore.
  2. Preserve optimistic versions, canonical ownership, and recoverable derived work.
  3. Add the required tests in the same delivery slice.
- Migration/compatibility: preserve IDs, personal context, and documented adapter behavior.
- Failure recovery: reject unsafe or stale work without damaging canonical state; expose a stable safe error.
- Verification: Legacy detail, confirmation, post-delete navigation, and English-copy tests.
- Complete when: code, tests, documentation impact, and [evidence](execution.md#org-ui-003) are reviewed.
- Commit boundary: one to three scoped commits.
