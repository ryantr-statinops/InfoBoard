# Tasks — Testing, performance, and CI

- [ ] <a id="t20-001"></a>**T20-001 — Add isolated SQLite and HTTP fixtures**
  - Product: `PR-OPS-02` · Evidence: `RQ-014`
  - Depends on: `T10-001`, `T11-001` · Code: pytest fixtures and ASGI lifespan setup
  - Verify: all existing tests use temporary state
  - Accept when: tests never touch the default database and are repeatable.
  - Commit: `test: add isolated sqlite and http fixtures`
  - Evidence: [execution entry](execution.md#t20-001)

- [ ] <a id="t20-002"></a>**T20-002 — Cover item, ingestion, and recovery integration**
  - Product: `PR-CAP-*`, `PR-REC-*` · Evidence: `RQ-001…006`, `RQ-012`
  - Depends on: M1, M2, 18 · Code: integration suites
  - Verify: complete capture/import/retry/restore API flow
  - Accept when: canonical content and recovery behavior are tested end-to-end.
  - Commit: `test: cover complete item ingestion and recovery api`
  - Evidence: [execution entry](execution.md#t20-002)

- [ ] <a id="t20-003"></a>**T20-003 — Add multilingual, retrieval, and security suites**
  - Product: `PR-RET-*`, `PR-SEC-*` · Evidence: `RQ-007…011`
  - Depends on: 15, 17 · Code: query/security fixtures
  - Verify: Vietnamese/English retrieval, fallback, SSRF, limits, and redaction tests
  - Accept when: core and security gates are deterministic.
  - Commit: `test: add multilingual and security suites`
  - Evidence: [execution entry](execution.md#t20-003)

- [ ] <a id="t20-004"></a>**T20-004 — Add benchmark dataset and p95 report**
  - Product: `PR-RET-*`, `PR-RET-06` · Evidence: `RQ-009`, `RQ-010`, `RQ-014`
  - Depends on: 15, 16 · Code: benchmark dataset/harness
  - Verify: 1,000-item/10,000-chunk warm/cold benchmark
  - Accept when: configuration and p50/p95/p99 results are reproducible.
  - Commit: `test: add benchmark dataset and p95 report`
  - Evidence: [execution entry](execution.md#t20-004)

- [ ] <a id="t20-005"></a>**T20-005 — Run quality gates in CI**
  - Product: `PR-OPS-02` · Evidence: `RQ-014`
  - Depends on: `T20-001…004` · Code: CI workflow and quality gate documentation
  - Verify: CI run with intentional migration/lint/core-test failure
  - Accept when: required failures block the workflow and optional full smoke is clearly separated.
  - Commit: `ci: run quality gates on pull requests`
  - Evidence: [execution entry](execution.md#t20-005)
