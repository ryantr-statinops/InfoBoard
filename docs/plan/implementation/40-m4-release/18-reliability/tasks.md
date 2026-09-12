# Tasks — Reliability, backup, and recovery

- [ ] <a id="t18-001"></a>**T18-001 — Create a consistent SQLite backup manifest**
  - Product: `PR-REC-01`, `PR-REC-02` · Evidence: `RQ-012`
  - Depends on: `T11-001`, `T14-001` · Code: target backup/manifest command
  - Verify: checkpoint/pause, checksum, permissions, and secret-exclusion tests
  - Accept when: a backup is self-describing and can be verified before restore.
  - Commit: `feat: add sqlite backup manifest and verify command`
  - Evidence: [execution entry](execution.md#t18-001)

- [ ] <a id="t18-002"></a>**T18-002 — Add safe restore and migration flow**
  - Product: `PR-REC-01…02` · Evidence: `RQ-012`
  - Depends on: `T18-001`, `T11-004` · Code: restore confirmation and migration path
  - Verify: clean-machine restore and interrupted-restore fixture
  - Accept when: active data is protected by safety copy and transactional migration.
  - Commit: `feat: add safe restore and migration flow`
  - Evidence: [execution entry](execution.md#t18-002)

- [ ] <a id="t18-003"></a>**T18-003 — Add derived-index cleanup and rebuild**
  - Product: `PR-REC-03` · Evidence: `RQ-012`
  - Depends on: `T14-004`, `T18-002` · Code: FTS/semantic/cache maintenance commands
  - Verify: delete-derived-stores and rebuild comparison
  - Accept when: SQLite content reconstructs all supported derived stores.
  - Commit: `feat: add derived index cleanup and rebuild`
  - Evidence: [execution entry](execution.md#t18-003)

- [ ] <a id="t18-004"></a>**T18-004 — Add crash and integrity checks**
  - Product: `PR-REC-04…05` · Evidence: `RQ-012`, `RQ-013`
  - Depends on: `T18-002`, `T18-003` · Code: integrity/health checks
  - Verify: interrupted restore, deleted-item, and restart fixtures
  - Accept when: recovery detects damage before resume and keeps public data safe.
  - Commit: `feat: add crash recovery integrity checks`
  - Evidence: [execution entry](execution.md#t18-004)

- [ ] <a id="t18-005"></a>**T18-005 — Verify backup, restore, delete, and rebuild**
  - Product: `PR-REC-01…05` · Evidence: `RQ-012`
  - Depends on: `T18-004` · Code: recovery test suite and transcript
  - Verify: `uv run pytest -q` plus documented recovery guide
  - Accept when: recovery evidence is repeatable on a database copy.
  - Commit: `test: cover backup restore delete and rebuild`
  - Evidence: [execution entry](execution.md#t18-005)
