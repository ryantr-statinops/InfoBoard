# FTS5 indexing and rebuild: tasks

This is the sole authoritative checklist for RET-FTS. Evidence is mandatory before checking a task.

## RET-FTS-001 — Define the current-version FTS projection

- [ ] **RET-FTS-001: Define the current-version FTS projection**
- Outcome: Define the current-version FTS projection.
- Prerequisites: CAP-SNAPSHOT, CAP-WORKER, ORG-CONTEXT.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: `app/db.py`, FTS repository/index jobs, rebuild maintenance, integration tests.
- Actions:
  1. Index eligible title, description, extracted text, note-derived fields only as allowed, plus bookmark/snapshot/revision identity.
  2. Preserve optimistic versions, canonical ownership, and recoverable derived work.
  3. Add the required tests in the same delivery slice.
- Migration/compatibility: preserve IDs, personal context, and documented adapter behavior.
- Failure recovery: reject unsafe or stale work without damaging canonical state; expose a stable safe error.
- Verification: Ready/failed/current/stale/legacy/deleted projection tests.
- Complete when: code, tests, documentation impact, and [evidence](execution.md#ret-fts-001) are reviewed.
- Commit boundary: one to three scoped commits.

## RET-FTS-002 — Process idempotent keyword index jobs

- [ ] **RET-FTS-002: Process idempotent keyword index jobs**
- Outcome: Process idempotent keyword index jobs.
- Prerequisites: CAP-SNAPSHOT, CAP-WORKER, ORG-CONTEXT.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: `app/db.py`, FTS repository/index jobs, rebuild maintenance, integration tests.
- Actions:
  1. Upsert current content, remove superseded/deleted rows, reject stale jobs, and checkpoint safe failures.
  2. Preserve optimistic versions, canonical ownership, and recoverable derived work.
  3. Add the required tests in the same delivery slice.
- Migration/compatibility: preserve IDs, personal context, and documented adapter behavior.
- Failure recovery: reject unsafe or stale work without damaging canonical state; expose a stable safe error.
- Verification: Replay, out-of-order, recapture, soft-delete, and restart tests.
- Complete when: code, tests, documentation impact, and [evidence](execution.md#ret-fts-002) are reviewed.
- Commit boundary: one to three scoped commits.

## RET-FTS-003 — Validate and rebuild FTS independently

- [ ] **RET-FTS-003: Validate and rebuild FTS independently**
- Outcome: Validate and rebuild FTS independently.
- Prerequisites: CAP-SNAPSHOT, CAP-WORKER, ORG-CONTEXT.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: `app/db.py`, FTS repository/index jobs, rebuild maintenance, integration tests.
- Actions:
  1. Detect schema/content mismatch and rebuild from canonical SQLite plus managed current snapshots without repairing canonical rows.
  2. Preserve optimistic versions, canonical ownership, and recoverable derived work.
  3. Add the required tests in the same delivery slice.
- Migration/compatibility: preserve IDs, personal context, and documented adapter behavior.
- Failure recovery: reject unsafe or stale work without damaging canonical state; expose a stable safe error.
- Verification: Corruption, restore, full rebuild, count, and checksum tests.
- Complete when: code, tests, documentation impact, and [evidence](execution.md#ret-fts-003) are reviewed.
- Commit boundary: one to three scoped commits.
