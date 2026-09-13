# Health and observability: tasks

This is the sole authoritative checklist for REL-HEALTH.

## REL-HEALTH-001 — Implement component and overall health semantics

- [ ] **REL-HEALTH-001: Implement component and overall health semantics**
- Outcome: Implement component and overall health semantics.
- Prerequisites: All component-owning packages through ANA-PROJECTION.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: health service/router, logging middleware, maintenance job projection, diagnostics tests.
- Actions:
  1. Distinguish unconfigured, ready, degraded, and unavailable; make canonical/FTS unavailability 503 while optional failure remains safe degradation.
  2. Keep SQLite canonical and make projection, maintenance, and diagnostics behavior bounded.
  3. Add failure injection and parity/security tests in the same slice.
- Migration/compatibility: preserve IDs and data; validate before activation; keep API envelopes stable.
- Failure recovery: prefer safe fallback or explicit degradation; never repair canonical rows from a derived store.
- Verification: Full failure matrix, startup transition, stale projection, and unconfigured-provider tests.
- Complete when: implementation, tests, operational evidence, and [task evidence](execution.md#rel-health-001) are reviewed.
- Commit boundary: one to three scoped commits.

## REL-HEALTH-002 — Add correlation and redacted structured diagnostics

- [ ] **REL-HEALTH-002: Add correlation and redacted structured diagnostics**
- Outcome: Add correlation and redacted structured diagnostics.
- Prerequisites: All component-owning packages through ANA-PROJECTION.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: health service/router, logging middleware, maintenance job projection, diagnostics tests.
- Actions:
  1. Propagate correlation IDs through requests/jobs, record stable codes and bounded metadata, and redact secrets/content/provider responses.
  2. Keep SQLite canonical and make projection, maintenance, and diagnostics behavior bounded.
  3. Add failure injection and parity/security tests in the same slice.
- Migration/compatibility: preserve IDs and data; validate before activation; keep API envelopes stable.
- Failure recovery: prefer safe fallback or explicit degradation; never repair canonical rows from a derived store.
- Verification: Redaction snapshots, correlation propagation, exception, malformed response, and log-size tests.
- Complete when: implementation, tests, operational evidence, and [task evidence](execution.md#rel-health-002) are reviewed.
- Commit boundary: one to three scoped commits.

## REL-HEALTH-003 — Expose maintenance job progress safely

- [ ] **REL-HEALTH-003: Expose maintenance job progress safely**
- Outcome: Expose maintenance job progress safely.
- Prerequisites: All component-owning packages through ANA-PROJECTION.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: health service/router, logging middleware, maintenance job projection, diagnostics tests.
- Actions:
  1. Project queued/processing/indexed/failed state, retryability, checkpoints, and safe errors without leaking internal paths or content.
  2. Keep SQLite canonical and make projection, maintenance, and diagnostics behavior bounded.
  3. Add failure injection and parity/security tests in the same slice.
- Migration/compatibility: preserve IDs and data; validate before activation; keep API envelopes stable.
- Failure recovery: prefer safe fallback or explicit degradation; never repair canonical rows from a derived store.
- Verification: State transition, unknown job, exhausted retry, and response-schema tests.
- Complete when: implementation, tests, operational evidence, and [task evidence](execution.md#rel-health-003) are reviewed.
- Commit boundary: one to three scoped commits.
