# Tasks — Search and discovery

- [ ] <a id="t15-001"></a>**T15-001 — Harden FTS query, filters, and excerpts**
  - Product: `PR-RET-01`, `PR-RET-02`, `PR-RET-04` · Evidence: `RQ-007`
  - Depends on: `T11-003`, `T12-005` · Code: FTS query adapter and shared filters
  - Verify: multilingual query, deleted-item, sanitize, and highlight tests
  - Accept when: core-mode keyword search is safe and deterministic.
  - Commit: `feat: harden fts search and shared filters`
  - Evidence: [execution entry](execution.md#t15-001)

- [ ] <a id="t15-002"></a>**T15-002 — Add semantic retrieval boundary**
  - Product: `PR-RET-03` · Evidence: `RQ-008`
  - Depends on: `T14-004` · Code: optional Chroma/embedding adapter
  - Verify: full-mode smoke and provider-unavailable test
  - Accept when: semantic retrieval is optional and reports source/provenance.
  - Commit: `feat: add optional semantic retrieval adapter`
  - Evidence: [execution entry](execution.md#t15-002)

- [ ] <a id="t15-003"></a>**T15-003 — Implement hybrid RRF ranking**
  - Product: `PR-RET-03`, `PR-RET-05` · Evidence: `RQ-009`
  - Depends on: `T15-001`, `T15-002` · Code: rank fusion and SQLite hydration
  - Verify: hybrid ranking fixtures and semantic-failure fallback
  - Accept when: semantic failure never removes keyword results.
  - Commit: `feat: implement hybrid retrieval and rrf ranking`
  - Evidence: [execution entry](execution.md#t15-003)

- [ ] <a id="t15-004"></a>**T15-004 — Add related content and topic clusters**
  - Product: `PR-RET-05` · Evidence: `RQ-009`
  - Depends on: `T15-003` · Code: related/cluster service and UI state
  - Verify: provenance, empty, degraded, and deleted-source tests
  - Accept when: every result can be traced to valid source items.
  - Commit: `feat: add related content and topic clusters`
  - Evidence: [execution entry](execution.md#t15-004)

- [ ] <a id="t15-005"></a>**T15-005 — Evaluate retrieval quality and performance**
  - Product: `PR-RET-01`, `PR-RET-02`, `PR-RET-03`, `PR-RET-04`, `PR-RET-05` · Evidence: `RQ-007`, `RQ-008`, `RQ-009`
  - Depends on: `T15-004` · Code: query set and benchmark harness
  - Verify: Vietnamese/English top-five, warm/cold latency, and fallback report
  - Accept when: agreed retrieval and performance targets have reproducible evidence.
  - Commit: `test: add search evaluation and benchmark evidence`
  - Evidence: [execution entry](execution.md#t15-005)
