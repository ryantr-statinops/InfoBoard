# Tasks — Observability and diagnostics

- [ ] <a id="t19-001"></a>**T19-001 — Add structured application and job events**
  - Product: `PR-REC-04` · Evidence: `RQ-013`
  - Depends on: `T10-003`, `T14-003` · Code: event schema and lifecycle logging
  - Verify: happy/error event fixtures and forbidden-field scan
  - Accept when: events include correlation/timing fields and exclude content/secrets.
  - Commit: `feat: add structured application and job events`
  - Evidence: [execution entry](execution.md#t19-001)

- [ ] <a id="t19-002"></a>**T19-002 — Expand component health checks**
  - Product: `PR-REC-04` · Evidence: `RQ-013`
  - Depends on: `T10-003`, `T14-004` · Code: SQLite/FTS/derived health contract
  - Verify: component failure matrix and HTTP status tests
  - Accept when: SQLite/FTS failure is `503`, derived failure is `200 degraded`.
  - Commit: `feat: expand component health checks`
  - Evidence: [execution entry](execution.md#t19-002)

- [ ] <a id="t19-003"></a>**T19-003 — Add a local diagnostics bundle**
  - Product: `PR-REC-04` · Evidence: `RQ-013`
  - Depends on: `T19-001`, `T19-002` · Code: target `diagnose` command and UI actions
  - Verify: bundle content/secret scan
  - Accept when: diagnostics are useful without including DB content or secrets.
  - Commit: `feat: add local diagnostics bundle`
  - Evidence: [execution entry](execution.md#t19-003)

- [ ] <a id="t19-004"></a>**T19-004 — Verify redaction, health, and degraded states**
  - Product: `PR-REC-04` · Evidence: `RQ-013`
  - Depends on: `T19-003` · Code: observability test suite
  - Verify: `uv run pytest -q`
  - Accept when: every component state and UI mapping has evidence.
  - Commit: `test: verify redaction health and degraded states`
  - Evidence: [execution entry](execution.md#t19-004)
