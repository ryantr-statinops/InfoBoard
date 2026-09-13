# Analytics API and UI: tasks

This is the sole authoritative checklist for ANA-API.

## ANA-API-001 — Implement analytics request and response contracts

- [ ] **ANA-API-001: Implement analytics request and response contracts**
- Outcome: Implement analytics request and response contracts.
- Prerequisites: API-CANONICAL, ANA-PROJECTION, ORG-UI.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: analytics service/router/schemas, dashboard analytics views, API/browser tests.
- Actions:
  1. Default time series to previous 30 days, enforce half-open UTC ranges up to 366 days, cap top lists at 20, and reject unsupported shapes.
  2. Keep SQLite canonical and make projection, maintenance, and diagnostics behavior bounded.
  3. Add failure injection and parity/security tests in the same slice.
- Migration/compatibility: preserve IDs and data; validate before activation; keep API envelopes stable.
- Failure recovery: prefer safe fallback or explicit degradation; never repair canonical rows from a derived store.
- Verification: Boundary dates, timezone, empty range, day 366/367, top limit, and schema tests.
- Complete when: implementation, tests, operational evidence, and [task evidence](execution.md#ana-api-001) are reviewed.
- Commit boundary: one to three scoped commits.

## ANA-API-002 — Apply shared filters and backend metadata

- [ ] **ANA-API-002: Apply shared filters and backend metadata**
- Outcome: Apply shared filters and backend metadata.
- Prerequisites: API-CANONICAL, ANA-PROJECTION, ORG-UI.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: analytics service/router/schemas, dashboard analytics views, API/browser tests.
- Actions:
  1. Use OR within dimensions and AND across dimensions; align deletion/legacy/status/domain/date semantics with browse/search; return actual_backend and freshness.
  2. Keep SQLite canonical and make projection, maintenance, and diagnostics behavior bounded.
  3. Add failure injection and parity/security tests in the same slice.
- Migration/compatibility: preserve IDs and data; validate before activation; keep API envelopes stable.
- Failure recovery: prefer safe fallback or explicit degradation; never repair canonical rows from a derived store.
- Verification: Cross-surface filter parity, DuckDB/SQLite response equivalence, and metadata tests.
- Complete when: implementation, tests, operational evidence, and [task evidence](execution.md#ana-api-002) are reviewed.
- Commit boundary: one to three scoped commits.

## ANA-API-003 — Build bounded dashboard analytics states

- [ ] **ANA-API-003: Build bounded dashboard analytics states**
- Outcome: Build bounded dashboard analytics states.
- Prerequisites: API-CANONICAL, ANA-PROJECTION, ORG-UI.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: analytics service/router/schemas, dashboard analytics views, API/browser tests.
- Actions:
  1. Render totals/trends/top collections/tags/domains, empty/loading/degraded states, and responsive layouts without exposing storage jargon.
  2. Keep SQLite canonical and make projection, maintenance, and diagnostics behavior bounded.
  3. Add failure injection and parity/security tests in the same slice.
- Migration/compatibility: preserve IDs and data; validate before activation; keep API envelopes stable.
- Failure recovery: prefer safe fallback or explicit degradation; never repair canonical rows from a derived store.
- Verification: Empty/data/fallback/degraded browser tests at 1440 and 390 px.
- Complete when: implementation, tests, operational evidence, and [task evidence](execution.md#ana-api-003) are reviewed.
- Commit boundary: one to three scoped commits.
