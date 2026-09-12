# Tasks — Analytics and insights

- [ ] <a id="t16-001"></a>**T16-001 — Define shared analytics filters and KPI query**
  - Product: `PR-RET-06` · Evidence: `RQ-010`
  - Depends on: `T12-003`, `T15-001` · Code: analytics query contract
  - Verify: filter and KPI fixture tests
  - Accept when: analytics filters mean the same thing as list/search filters.
  - Commit: `feat: define shared analytics filters and kpis`
  - Evidence: [execution entry](execution.md#t16-001)

- [ ] <a id="t16-002"></a>**T16-002 — Implement SQLite analytics fallback**
  - Product: `PR-RET-06` · Evidence: `RQ-010`
  - Depends on: `T16-001` · Code: SQLite aggregation path
  - Verify: KPI correctness and empty-data tests
  - Accept when: core mode returns correct KPI values without optional stores.
  - Commit: `feat: implement sqlite analytics fallback`
  - Evidence: [execution entry](execution.md#t16-002)

- [ ] <a id="t16-003"></a>**T16-003 — Add optional DuckDB acceleration and degraded response**
  - Product: `PR-RET-06` · Evidence: `RQ-010`
  - Depends on: `T16-002` · Code: DuckDB adapter and mode metadata
  - Verify: DuckDB/SQLite parity and dependency-unavailable tests
  - Accept when: optional-store failure is truthful and falls back safely.
  - Commit: `feat: add optional duckdb analytics mode`
  - Evidence: [execution entry](execution.md#t16-003)

- [ ] <a id="t16-004"></a>**T16-004 — Benchmark analytics and dashboard states**
  - Product: `PR-RET-06` · Evidence: `RQ-010`
  - Depends on: `T16-003` · Code: benchmark fixture and UI state checks
  - Verify: cold/warm latency and component-degraded browser smoke
  - Accept when: KPI accuracy, latency, and degraded messaging are evidenced.
  - Commit: `test: benchmark analytics and degraded dashboard states`
  - Evidence: [execution entry](execution.md#t16-004)
