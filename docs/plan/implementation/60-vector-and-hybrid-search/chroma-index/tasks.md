# ChromaDB semantic index: tasks

This is the sole authoritative checklist for VEC-CHROMA.

## VEC-CHROMA-001

**Title:** Implement revision-scoped collections and idempotent upserts

- [ ] **VEC-CHROMA-001: Implement revision-scoped collections and idempotent upserts**
- Outcome: Implement revision-scoped collections and idempotent upserts.
- Prerequisites: SEM-EMBED, CAP-WORKER.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: Chroma adapter, vector index job handler, revision activation, cleanup/rebuild tests.
- Actions:
  1. Store bookmark, snapshot version, chunk, and revision identity; never create canonical records from vector metadata.
  2. Keep consent, revision, canonical-validation, and redaction checks at the final use boundary.
  3. Add success, incompatibility, outage, and recovery tests with the implementation.
- Migration/compatibility: do not reuse local-model vectors or incompatible revisions; preserve keyword behavior.
- Failure recovery: derived failure degrades or retries safely and never rolls back canonical content.
- Verification: Replay, revision isolation, duplicate chunk, and partial-write tests.
- Complete when: implementation and tests pass and [evidence](execution.md#vec-chroma-001) is reviewed.
- Commit boundary: one to three scoped commits.

## VEC-CHROMA-002

**Title:** Validate semantic candidates against SQLite

- [ ] **VEC-CHROMA-002: Validate semantic candidates against SQLite**
- Outcome: Validate semantic candidates against SQLite.
- Prerequisites: SEM-EMBED, CAP-WORKER.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: Chroma adapter, vector index job handler, revision activation, cleanup/rebuild tests.
- Actions:
  1. Reject deleted bookmarks, non-current snapshots, inactive revisions, and missing canonical rows before result projection.
  2. Keep consent, revision, canonical-validation, and redaction checks at the final use boundary.
  3. Add success, incompatibility, outage, and recovery tests with the implementation.
- Migration/compatibility: do not reuse local-model vectors or incompatible revisions; preserve keyword behavior.
- Failure recovery: derived failure degrades or retries safely and never rolls back canonical content.
- Verification: Deleted/stale/orphan/revision-race and canonical-unavailable tests.
- Complete when: implementation and tests pass and [evidence](execution.md#vec-chroma-002) is reviewed.
- Commit boundary: one to three scoped commits.

## VEC-CHROMA-003

**Title:** Implement cleanup, health, and rebuild

- [ ] **VEC-CHROMA-003: Implement cleanup, health, and rebuild**
- Outcome: Implement cleanup, health, and rebuild.
- Prerequisites: SEM-EMBED, CAP-WORKER.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: Chroma adapter, vector index job handler, revision activation, cleanup/rebuild tests.
- Actions:
  1. Remove deleted/revoked/superseded data, detect unavailable/corrupt indexes, and rebuild from current snapshots through embedding jobs.
  2. Keep consent, revision, canonical-validation, and redaction checks at the final use boundary.
  3. Add success, incompatibility, outage, and recovery tests with the implementation.
- Migration/compatibility: do not reuse local-model vectors or incompatible revisions; preserve keyword behavior.
- Failure recovery: derived failure degrades or retries safely and never rolls back canonical content.
- Verification: Cleanup interruption, outage, corruption, rebuild, and count-consistency tests.
- Complete when: implementation and tests pass and [evidence](execution.md#vec-chroma-003) is reviewed.
- Commit boundary: one to three scoped commits.
