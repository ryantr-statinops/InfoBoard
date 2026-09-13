# Bookmark capture and snapshots: tasks

This file is the sole authoritative checklist for CAP-SNAPSHOT. Do not check an item until its evidence anchor contains real results.

## CAP-SNAPSHOT-001 — Create or resolve the canonical bookmark and durable attempt

- [ ] **CAP-SNAPSHOT-001: Create or resolve the canonical bookmark and durable attempt**
- Outcome: Create or resolve the canonical bookmark and durable attempt.
- Prerequisites: DB-MIGRATION, API-CANONICAL, CAP-URL.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: capture application service, bookmark/attempt/snapshot repositories, managed content storage, API tests.
- Actions:
  1. Commit normalized identity, organization request, attempt number, and pending state before network work; make duplicate submission deterministic.
  2. Keep canonical writes transactional and derived work recoverable.
  3. Add tests with the implementation rather than deferring verification.
- Migration/compatibility: preserve canonical IDs and user data; do not broaden legacy or adapter behavior.
- Failure recovery: fail safely, retain canonical state, and record a stable error/checkpoint where applicable.
- Verification: New, duplicate, deleted-URL resave, and transaction-failure tests.
- Complete when: implementation, tests, documentation impact, and [evidence](execution.md#cap-snapshot-001) are reviewed.
- Commit boundary: one to three scoped commits.

## CAP-SNAPSHOT-002 — Persist successful content as an immutable snapshot

- [ ] **CAP-SNAPSHOT-002: Persist successful content as an immutable snapshot**
- Outcome: Persist successful content as an immutable snapshot.
- Prerequisites: DB-MIGRATION, API-CANONICAL, CAP-URL.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: capture application service, bookmark/attempt/snapshot repositories, managed content storage, API tests.
- Actions:
  1. Write managed content atomically, verify checksum/provenance, insert the next content version, and switch current_snapshot_id only after success.
  2. Keep canonical writes transactional and derived work recoverable.
  3. Add tests with the implementation rather than deferring verification.
- Migration/compatibility: preserve canonical IDs and user data; do not broaden legacy or adapter behavior.
- Failure recovery: fail safely, retain canonical state, and record a stable error/checkpoint where applicable.
- Verification: Atomic-file, checksum, recapture, and interrupted-commit tests.
- Complete when: implementation, tests, documentation impact, and [evidence](execution.md#cap-snapshot-002) are reviewed.
- Commit boundary: one to three scoped commits.

## CAP-SNAPSHOT-003 — Expose safe failure and idempotent retry

- [ ] **CAP-SNAPSHOT-003: Expose safe failure and idempotent retry**
- Outcome: Expose safe failure and idempotent retry.
- Prerequisites: DB-MIGRATION, API-CANONICAL, CAP-URL.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: capture application service, bookmark/attempt/snapshot repositories, managed content storage, API tests.
- Actions:
  1. Keep failed attempts separate from snapshots, retain the current successful snapshot, enforce latest-attempt retry rules, and use safe codes.
  2. Keep canonical writes transactional and derived work recoverable.
  3. Add tests with the implementation rather than deferring verification.
- Migration/compatibility: preserve canonical IDs and user data; do not broaden legacy or adapter behavior.
- Failure recovery: fail safely, retain canonical state, and record a stable error/checkpoint where applicable.
- Verification: Fetch/extraction failure, retry replay, active-attempt conflict, and legacy capture rejection tests.
- Complete when: implementation, tests, documentation impact, and [evidence](execution.md#cap-snapshot-003) are reviewed.
- Commit boundary: one to three scoped commits.
