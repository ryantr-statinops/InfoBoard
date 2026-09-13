# Backup, restore, and rebuild: tasks

This is the sole authoritative checklist for REL-RECOVERY.

## REL-RECOVERY-001

**Title:** Create consistent canonical backup and manifest

- [ ] **REL-RECOVERY-001: Create consistent canonical backup and manifest**
- Outcome: Create consistent canonical backup and manifest.
- Prerequisites: DB-MIGRATION, RET-FTS, VEC-CHROMA, CACHE-ROCKS, ANA-PROJECTION, REL-HEALTH.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: backup/restore services or CLI, maintenance jobs, archive/manifest validation, disaster-recovery tests.
- Actions:
  1. Snapshot SQLite safely, include referenced managed content, record schema/version/checksums, exclude secrets, and verify archive boundaries.
  2. Keep SQLite canonical and make projection, maintenance, and diagnostics behavior bounded.
  3. Add failure injection and parity/security tests in the same slice.
- Migration/compatibility: preserve IDs and data; validate before activation; keep API envelopes stable.
- Failure recovery: prefer safe fallback or explicit degradation; never repair canonical rows from a derived store.
- Verification: Concurrent-write, missing content, checksum, traversal, secret exclusion, and interrupted-backup tests.
- Complete when: implementation, tests, operational evidence, and [task evidence](execution.md#rel-recovery-001) are reviewed.
- Commit boundary: one to three scoped commits.

## REL-RECOVERY-002

**Title:** Restore to a safe location and validate canonical state

- [ ] **REL-RECOVERY-002: Restore to a safe location and validate canonical state**
- Outcome: Restore to a safe location and validate canonical state.
- Prerequisites: DB-MIGRATION, RET-FTS, VEC-CHROMA, CACHE-ROCKS, ANA-PROJECTION, REL-HEALTH.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: backup/restore services or CLI, maintenance jobs, archive/manifest validation, disaster-recovery tests.
- Actions:
  1. Extract without traversal/overwrite hazards, verify manifest/checksums/schema, migrate if supported, and avoid activating invalid data.
  2. Keep SQLite canonical and make projection, maintenance, and diagnostics behavior bounded.
  3. Add failure injection and parity/security tests in the same slice.
- Migration/compatibility: preserve IDs and data; validate before activation; keep API envelopes stable.
- Failure recovery: prefer safe fallback or explicit degradation; never repair canonical rows from a derived store.
- Verification: Tampered archive, wrong version, unsafe path, partial restore, count, and rollback tests.
- Complete when: implementation, tests, operational evidence, and [task evidence](execution.md#rel-recovery-002) are reviewed.
- Commit boundary: one to three scoped commits.

## REL-RECOVERY-003

**Title:** Rebuild derived stores in documented order

- [ ] **REL-RECOVERY-003: Rebuild derived stores in documented order**
- Outcome: Rebuild derived stores in documented order.
- Prerequisites: DB-MIGRATION, RET-FTS, VEC-CHROMA, CACHE-ROCKS, ANA-PROJECTION, REL-HEALTH.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: backup/restore services or CLI, maintenance jobs, archive/manifest validation, disaster-recovery tests.
- Actions:
  1. Validate SQLite/current snapshots, rebuild FTS then RocksDB/Chroma/DuckDB dependencies, checkpoint interruption, and run consistency/health verification.
  2. Keep SQLite canonical and make projection, maintenance, and diagnostics behavior bounded.
  3. Add failure injection and parity/security tests in the same slice.
- Migration/compatibility: preserve IDs and data; validate before activation; keep API envelopes stable.
- Failure recovery: prefer safe fallback or explicit degradation; never repair canonical rows from a derived store.
- Verification: Empty-derived restore, interruption at each stage, retry, mismatch, and end-to-end transcript.
- Complete when: implementation, tests, operational evidence, and [task evidence](execution.md#rel-recovery-003) are reviewed.
- Commit boundary: one to three scoped commits.
