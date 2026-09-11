# 92 — Requirement traceability

**Status:** `active`

## Mapping convention

`PR-*` in the Internal PRD is canonical product intent. `RQ-*` is implementation evidence mapped to one or more `PR-*`; it is not a second product-requirement system. `T*` task IDs connect each RQ to implementation and execution evidence. A `partial` or `missing` result does not satisfy the release gate.

## MVP matrix

| ID | Product source | Requirement | Epic/task evidence | Test/evidence | Status |
| --- | --- | --- | --- | --- | --- |
| RQ-001 | `PR-CAP-01`, `PR-UX-01` | Manual text and dashboard list | 10: [T10-001](../10-m1-local-dashboard/10-application-foundation/tasks.md#t10-001), 12: [T12-001](../10-m1-local-dashboard/12-dashboard/tasks.md#t12-001), [T12-002](../10-m1-local-dashboard/12-dashboard/tasks.md#t12-002) | HTTP/UI create flow | partial |
| RQ-002 | `PR-ORG-01…02` | Many-to-many collections | 11: [T11-002](../10-m1-local-dashboard/11-data-and-migrations/tasks.md#t11-002), 12: [T12-003](../10-m1-local-dashboard/12-dashboard/tasks.md#t12-003) | Collection attach/filter test | partial |
| RQ-003 | `PR-ORG-03` | Notes CRUD | 12: [T12-004](../10-m1-local-dashboard/12-dashboard/tasks.md#t12-004) | Note integration test | partial |
| RQ-004 | `PR-CAP-01…02` | TXT/Markdown/PDF/URL import | 13: [T13-001](../20-m2-ingestion/13-ingestion-sources/tasks.md#t13-001)…[T13-005](../20-m2-ingestion/13-ingestion-sources/tasks.md#t13-005) | Extractor + API tests | partial |
| RQ-005 | `PR-CAP-03` | Hash/URL deduplication | 11: [T11-002](../10-m1-local-dashboard/11-data-and-migrations/tasks.md#t11-002), 13: [T13-004](../20-m2-ingestion/13-ingestion-sources/tasks.md#t13-004) | Duplicate fixture | partial |
| RQ-006 | `PR-CAP-05`, `PR-REC-01…02` | Durable indexing/retry/restart | 14: [T14-001](../20-m2-ingestion/14-indexing-worker/tasks.md#t14-001)…[T14-005](../20-m2-ingestion/14-indexing-worker/tasks.md#t14-005) | Crash/requeue test | partial |
| RQ-007 | `PR-RET-02`, `PR-RET-04` | FTS5 keyword search | 15: [T15-001](../30-m3-search-and-insights/15-search-and-discovery/tasks.md#t15-001) | Multilingual FTS test | partial |
| RQ-008 | `PR-RET-03`, `PR-RET-05` | Chroma semantic search | 15: [T15-002](../30-m3-search-and-insights/15-search-and-discovery/tasks.md#t15-002) | Full-mode smoke/eval | missing |
| RQ-009 | `PR-RET-03`, `PR-RET-05` | Hybrid RRF/related/clusters | 15: [T15-003](../30-m3-search-and-insights/15-search-and-discovery/tasks.md#t15-003)…[T15-005](../30-m3-search-and-insights/15-search-and-discovery/tasks.md#t15-005) | Retrieval evaluation | missing |
| RQ-010 | `PR-RET-06` | Dashboard KPI/analytics | 12: [T12-005](../10-m1-local-dashboard/12-dashboard/tasks.md#t12-005), 16: [T16-001](../30-m3-search-and-insights/16-analytics/tasks.md#t16-001)…[T16-004](../30-m3-search-and-insights/16-analytics/tasks.md#t16-004) | DuckDB/SQLite parity | partial |
| RQ-011 | `PR-SEC-01…02` | Limits/SSRF/escaping | 13: [T13-002](../20-m2-ingestion/13-ingestion-sources/tasks.md#t13-002), [T13-003](../20-m2-ingestion/13-ingestion-sources/tasks.md#t13-003), [T13-005](../20-m2-ingestion/13-ingestion-sources/tasks.md#t13-005); 17: [T17-001](../40-m4-release/17-security/tasks.md#t17-001)…[T17-004](../40-m4-release/17-security/tasks.md#t17-004) | Security suite | partial |
| RQ-012 | `PR-REC-05` | Backup/restore/rebuild | 18: [T18-001](../40-m4-release/18-reliability/tasks.md#t18-001)…[T18-005](../40-m4-release/18-reliability/tasks.md#t18-005) | Recovery transcript | missing |
| RQ-013 | `PR-REC-04` | Health/diagnostics | 10: [T10-003](../10-m1-local-dashboard/10-application-foundation/tasks.md#t10-003), [T10-004](../10-m1-local-dashboard/10-application-foundation/tasks.md#t10-004); 19: [T19-001](../40-m4-release/19-observability/tasks.md#t19-001)…[T19-004](../40-m4-release/19-observability/tasks.md#t19-004) | Component failure test | partial |
| RQ-014 | `PR-OPS-01…02` | CI, benchmark, and release docs | 20: [T20-001](../40-m4-release/20-testing-and-ci/tasks.md#t20-001)…[T20-005](../40-m4-release/20-testing-and-ci/tasks.md#t20-005); 21: [T21-001](../40-m4-release/21-packaging/tasks.md#t21-001)…[T21-004](../40-m4-release/21-packaging/tasks.md#t21-004) | CI run + benchmark report | missing |

## Traceability rules

- PR descriptions list affected requirement IDs and task IDs.
- Test names/docstrings or reports contain the RQ/task ID or link back to this matrix.
- Requirement changes update product/architecture docs, epic plan, task checklist, decision register, and this matrix in one review cycle.
- M4 release requires every `RQ-001…014` to be `verified` or covered by an explicit waiver.

## Post-MVP discovery

Discovery packages `30–35` use `D*` IDs and do not receive `RQ-*` until their implementation-ready gates approve an outcome, data boundary, threat model, and acceptance.
