# Tasks — Testing, performance, and CI

- [ ] <a id="t20-001"></a>**T20-001 — Add isolated SQLite and HTTP fixtures**
  - Product: `PR-OPS-02` · Evidence: `RQ-014`
  - Depends on: `T10-001`, `T11-001` · Code: pytest fixtures and ASGI lifespan setup
  - Verify: all existing tests use temporary state
  - Accept when: tests never touch the default database and are repeatable.
  - Commit: `test: add isolated sqlite and http fixtures`
  - Evidence: [execution entry](execution.md#t20-001)

- [ ] <a id="t20-002"></a>**T20-002 — Cover item, ingestion, and recovery integration**
  - Product: `PR-CAP-01`, `PR-CAP-02`, `PR-CAP-03`, `PR-CAP-04`, `PR-CAP-05`, `PR-ORG-03`, `PR-REC-01`, `PR-REC-02`, `PR-REC-03`, `PR-REC-04`, `PR-REC-05` · Evidence: `RQ-001`, `RQ-002`, `RQ-004`, `RQ-005`, `RQ-006`, `RQ-012`
  - Depends on: `T12-005`, `T14-005`, `T18-005` · Code: integration suites
  - Verify: complete capture/import/retry/restore API flow
  - Accept when: canonical content and recovery behavior are tested end-to-end.
  - Commit: `test: cover complete item ingestion and recovery api`
  - Evidence: [execution entry](execution.md#t20-002)

- [ ] <a id="t20-003"></a>**T20-003 — Add multilingual, retrieval, and security suites**
  - Product: `PR-RET-01`, `PR-RET-02`, `PR-RET-03`, `PR-RET-04`, `PR-RET-05`, `PR-RET-06`, `PR-SEC-01`, `PR-SEC-02`, `PR-SEC-03` · Evidence: `RQ-007`, `RQ-008`, `RQ-009`, `RQ-010`, `RQ-011`
  - Depends on: `T15-005`, `T17-004` · Code: query/security fixtures
  - Verify: Vietnamese/English retrieval, fallback, SSRF, limits, and redaction tests
  - Accept when: core and security gates are deterministic.
  - Commit: `test: add multilingual and security suites`
  - Evidence: [execution entry](execution.md#t20-003)

- [ ] <a id="t20-004"></a>**T20-004 — Add benchmark dataset and p95 report**
  - Product: `PR-RET-01`, `PR-RET-02`, `PR-RET-03`, `PR-RET-06`, `PR-OPS-02` · Evidence: `RQ-009`, `RQ-010`, `RQ-014`
  - Depends on: `T15-005`, `T16-004` · Code: benchmark dataset/harness
  - Verify: 1,000-item/10,000-chunk warm/cold benchmark
  - Accept when: configuration and p50/p95/p99 results are reproducible.
  - Commit: `test: add benchmark dataset and p95 report`
  - Evidence: [execution entry](execution.md#t20-004)

- [ ] <a id="t20-005"></a>**T20-005 — Run quality gates in CI**
  - Product: `PR-OPS-02` · Evidence: `RQ-014`
  - Depends on: `T20-001`, `T20-002`, `T20-003`, `T20-004` · Code: CI workflow and quality gate documentation
  - Verify: CI run with intentional migration/lint/core-test failure
  - Accept when: required failures block the workflow and optional full smoke is clearly separated.
  - Commit: `ci: run quality gates on pull requests`
  - Evidence: [execution entry](execution.md#t20-005)
