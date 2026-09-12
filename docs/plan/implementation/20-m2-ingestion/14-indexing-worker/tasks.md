# Tasks — Indexing worker and cache

- [ ] <a id="t14-001"></a>**T14-001 — Persist the indexing job lifecycle**
  - Product: `PR-CAP-05`, `PR-REC-01` · Evidence: `RQ-006`
  - Depends on: `T11-002`, `T13-001` · Code: job persistence and worker state transitions
  - Verify: lifecycle transition tests
  - Accept when: queued/extracting/chunking/embedding/indexed/failed states are durable.
  - Commit: `feat: persist indexing job lifecycle`
  - Evidence: [execution entry](execution.md#t14-001)

- [ ] <a id="t14-002"></a>**T14-002 — Implement deterministic chunk and version rules**
  - Product: `PR-CAP-05`, `PR-REC-01` · Evidence: `RQ-006`
  - Depends on: `T13-001`, `T11-002` · Code: chunker and content-version handling
  - Verify: repeated-input hash/position fixtures
  - Accept when: identical content produces stable chunks and no duplicates.
  - Commit: `feat: add deterministic chunk and content version rules`
  - Evidence: [execution entry](execution.md#t14-002)

- [ ] <a id="t14-003"></a>**T14-003 — Add retry, stale-job recovery, and restart handling**
  - Product: `PR-REC-01…02` · Evidence: `RQ-006`
  - Depends on: `T14-001` · Code: retry counter, lease/reclaim, restart path
  - Verify: crash/requeue/restart tests
  - Accept when: restart does not lose items or create duplicate chunks.
  - Commit: `feat: add worker retry and restart recovery`
  - Evidence: [execution entry](execution.md#t14-003)

- [ ] <a id="t14-004"></a>**T14-004 — Isolate derived indexes and cache**
  - Product: `PR-RET-02…05`, `PR-REC-01` · Evidence: `RQ-006`, `RQ-007…009`
  - Depends on: `T14-002` · Code: FTS, semantic adapter, and RocksDB cache boundary
  - Verify: rebuild and dependency-unavailable tests
  - Accept when: derived-store failure leaves SQLite content recoverable.
  - Commit: `feat: isolate derived indexes and cache recovery`
  - Evidence: [execution entry](execution.md#t14-004)

- [ ] <a id="t14-005"></a>**T14-005 — Verify indexing recovery and UI mapping**
  - Product: `PR-CAP-05`, `PR-REC-01…02` · Evidence: `RQ-006`
  - Depends on: `T14-003`, `T14-004` · Code: worker tests and state mapper
  - Verify: `uv run pytest tests/test_search.py -q` plus restart fixture
  - Accept when: technical states map to `queued | processing | indexed | failed` in the UI.
  - Commit: `test: cover indexing retry restart and degraded modes`
  - Evidence: [execution entry](execution.md#t14-005)
