# Application lifecycle and configuration: tasks

This file is the sole authoritative checklist for APP-LIFECYCLE. Do not check an item until its evidence anchor contains real results.

## APP-LIFECYCLE-001

**Title:** Introduce typed settings and validated local data paths

- [ ] **APP-LIFECYCLE-001: Introduce typed settings and validated local data paths**
- Outcome: Introduce typed settings and validated local data paths.
- Prerequisites: None.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: `app/main.py`, new settings/lifecycle modules, `.env.example`, application tests.
- Actions:
  1. Define precedence and defaults; validate paths without creating unsafe locations; expose redacted diagnostics.
  2. Keep canonical writes transactional and derived work recoverable.
  3. Add tests with the implementation rather than deferring verification.
- Migration/compatibility: preserve canonical IDs and user data; do not broaden legacy or adapter behavior.
- Failure recovery: fail safely, retain canonical state, and record a stable error/checkpoint where applicable.
- Verification: Unit tests for defaults, overrides, invalid paths, and redaction.
- Complete when: implementation, tests, documentation impact, and [evidence](execution.md#app-lifecycle-001) are reviewed.
- Commit boundary: one to three scoped commits.

## APP-LIFECYCLE-002

**Title:** Own startup and shutdown ordering in the FastAPI lifespan

- [ ] **APP-LIFECYCLE-002: Own startup and shutdown ordering in the FastAPI lifespan**
- Outcome: Own startup and shutdown ordering in the FastAPI lifespan.
- Prerequisites: None.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: `app/main.py`, new settings/lifecycle modules, `.env.example`, application tests.
- Actions:
  1. Initialize canonical SQLite before derived components; start one worker after migrations; stop intake and drain safely on shutdown.
  2. Keep canonical writes transactional and derived work recoverable.
  3. Add tests with the implementation rather than deferring verification.
- Migration/compatibility: preserve canonical IDs and user data; do not broaden legacy or adapter behavior.
- Failure recovery: fail safely, retain canonical state, and record a stable error/checkpoint where applicable.
- Verification: Lifespan integration tests for clean start, partial component failure, and shutdown.
- Complete when: implementation, tests, documentation impact, and [evidence](execution.md#app-lifecycle-002) are reviewed.
- Commit boundary: one to three scoped commits.
