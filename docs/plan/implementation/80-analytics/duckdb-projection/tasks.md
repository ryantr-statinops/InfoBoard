# DuckDB analytics projection: tasks

This is the sole authoritative checklist for ANA-PROJECTION.

## ANA-PROJECTION-001

**Title:** Define the analytics projection and checkpoint

- [ ] **ANA-PROJECTION-001: Define the analytics projection and checkpoint**
- Outcome: Define the analytics projection and checkpoint.
- Prerequisites: DB-MIGRATION, ORG-CONTEXT, RET-FTS, CAP-WORKER.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: `app/analytics.py`, projection job handler, analytics checkpoints, parity fixtures.
- Actions:
  1. Project eligible bookmark dimensions/events with schema revision, canonical watermark, refresh time, and consistency metadata.
  2. Keep SQLite canonical and make projection, maintenance, and diagnostics behavior bounded.
  3. Add failure injection and parity/security tests in the same slice.
- Migration/compatibility: preserve IDs and data; validate before activation; keep API envelopes stable.
- Failure recovery: prefer safe fallback or explicit degradation; never repair canonical rows from a derived store.
- Verification: Fresh/incremental projection, deleted/legacy/status/filter dimension, and schema tests.
- Complete when: implementation, tests, operational evidence, and [task evidence](execution.md#ana-projection-001) are reviewed.
- Commit boundary: one to three scoped commits.

## ANA-PROJECTION-002

**Title:** Implement atomic refresh and stale detection

- [ ] **ANA-PROJECTION-002: Implement atomic refresh and stale detection**
- Outcome: Implement atomic refresh and stale detection.
- Prerequisites: DB-MIGRATION, ORG-CONTEXT, RET-FTS, CAP-WORKER.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: `app/analytics.py`, projection job handler, analytics checkpoints, parity fixtures.
- Actions:
  1. Build or update outside canonical writes, activate only verified projections, detect lag/checksum/schema mismatch, and never serve known-stale data as current.
  2. Keep SQLite canonical and make projection, maintenance, and diagnostics behavior bounded.
  3. Add failure injection and parity/security tests in the same slice.
- Migration/compatibility: preserve IDs and data; validate before activation; keep API envelopes stable.
- Failure recovery: prefer safe fallback or explicit degradation; never repair canonical rows from a derived store.
- Verification: Interrupted refresh, concurrent canonical write, stale watermark, checksum, and rollback tests.
- Complete when: implementation, tests, operational evidence, and [task evidence](execution.md#ana-projection-002) are reviewed.
- Commit boundary: one to three scoped commits.

## ANA-PROJECTION-003

**Title:** Implement bounded SQLite fallback and rebuild

- [ ] **ANA-PROJECTION-003: Implement bounded SQLite fallback and rebuild**
- Outcome: Implement bounded SQLite fallback and rebuild.
- Prerequisites: DB-MIGRATION, ORG-CONTEXT, RET-FTS, CAP-WORKER.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: `app/analytics.py`, projection job handler, analytics checkpoints, parity fixtures.
- Actions:
  1. Return equivalent schemas/filter semantics for supported bounds, expose actual_backend, and recreate DuckDB from canonical SQLite.
  2. Keep SQLite canonical and make projection, maintenance, and diagnostics behavior bounded.
  3. Add failure injection and parity/security tests in the same slice.
- Migration/compatibility: preserve IDs and data; validate before activation; keep API envelopes stable.
- Failure recovery: prefer safe fallback or explicit degradation; never repair canonical rows from a derived store.
- Verification: DuckDB outage/corruption, 30/366-day, top-20, parity, and full-rebuild tests.
- Complete when: implementation, tests, operational evidence, and [task evidence](execution.md#ana-projection-003) are reviewed.
- Commit boundary: one to three scoped commits.
