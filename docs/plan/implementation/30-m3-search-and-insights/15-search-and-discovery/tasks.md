# Tasks — Search and discovery

- [ ] <a id="t15-001"></a>**T15-001 — Harden FTS query, filters, and excerpts**
  - Product: `PR-RET-01`, `PR-RET-02`, `PR-RET-04` · Evidence: `RQ-007`
  - Depends on: `T11-003`, `T12-005` · Code: FTS query adapter and shared filters
  - Verify: multilingual query, deleted-item, sanitize, and highlight tests
  - Accept when: core-mode keyword search is safe and deterministic.
  - Commit: `feat: harden fts search and shared filters`
  - Evidence: [execution entry](execution.md#t15-001)

- [ ] <a id="t15-002"></a>**T15-002 — Add optional full-mode semantic retrieval boundary**
  - Product: `PR-RET-03` · Evidence: conditional `RQ-008`
  - Depends on: `T14-004` · Code: optional Chroma/embedding adapter
  - Verify: full-mode smoke and provider-unavailable test
  - Accept when: semantic retrieval is isolated behind an optional full-mode boundary and reports source/provenance without affecting core keyword search.
  - Commit: `feat: add optional semantic retrieval adapter`
  - Evidence: [execution entry](execution.md#t15-002)

- [ ] <a id="t15-003"></a>**T15-003 — Implement optional full-mode hybrid RRF ranking**
  - Product: `PR-RET-03`, `PR-RET-05` · Evidence: conditional `RQ-009`
  - Depends on: `T15-001`, `T15-002` · Code: rank fusion and SQLite hydration
  - Verify: hybrid ranking fixtures and semantic-failure fallback
  - Accept when: full-mode hybrid ranking is isolated and semantic failure never removes core keyword results.
  - Commit: `feat: implement hybrid retrieval and rrf ranking`
  - Evidence: [execution entry](execution.md#t15-003)

- [ ] <a id="t15-004"></a>**T15-004 — Add optional full-mode related content and topic clusters**
  - Product: `PR-RET-05` · Evidence: conditional `RQ-009`
  - Depends on: `T15-003` · Code: related/cluster service and UI state
  - Verify: provenance, empty, degraded, and deleted-source tests
  - Accept when: optional full-mode output can be traced to valid source items and its absence leaves core retrieval usable.
  - Commit: `feat: add related content and topic clusters`
  - Evidence: [execution entry](execution.md#t15-004)

- [ ] <a id="t15-005"></a>**T15-005 — Evaluate retrieval quality and performance**
  - Product: `PR-RET-01`, `PR-RET-02`, `PR-RET-03`, `PR-RET-04`, `PR-RET-05` · Evidence: `RQ-007`, conditional `RQ-008`, conditional `RQ-009`
  - Depends on: `T15-004` · Code: query set and benchmark harness
  - Verify: core Vietnamese/English keyword top-five, warm/cold latency, fallback report, and optional full-mode evaluation when in scope
  - Accept when: core retrieval and performance targets have reproducible evidence; full-mode targets are reported separately when full mode is in release scope.
  - Commit: `test: add search evaluation and benchmark evidence`
  - Evidence: [execution entry](execution.md#t15-005)
