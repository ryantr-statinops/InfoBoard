# 92 — Requirement traceability

**Status:** `active`

## Mapping convention

`PR-*` trong Internal PRD là product intent canonical. `RQ-*` dưới đây là requirement/evidence triển khai, map về một hoặc nhiều `PR-*`; nó không phải hệ product requirement thứ hai. Mỗi RQ có ID ổn định, product source, epic, implementation evidence, test và acceptance status. `partial` không được tính là đạt release gate.

## MVP matrix

| ID | Product source | Requirement | Epic | Test/evidence | Status |
| --- | --- | --- | --- | --- | --- |
| RQ-001 | `PR-CAP-01`, `PR-UX-01` | Text nhập tay và dashboard list | 10, 12 | HTTP/UI create flow | partial |
| RQ-002 | `PR-ORG-01…02` | Collections nhiều-nhiều | 11, 12 | collection attach/filter test | partial |
| RQ-003 | `PR-ORG-03` | Notes CRUD | 12 | note integration test | partial |
| RQ-004 | `PR-CAP-01…02` | TXT/Markdown/PDF/URL import | 13 | extractor + API tests | partial |
| RQ-005 | `PR-CAP-03` | Hash/URL dedup | 11, 13 | duplicate fixture | partial |
| RQ-006 | `PR-CAP-05`, `PR-REC-01…02` | Durable indexing/retry/restart | 14 | crash/requeue test | partial |
| RQ-007 | `PR-RET-02`, `PR-RET-04` | FTS5 keyword search | 15 | multilingual FTS test | partial |
| RQ-008 | `PR-RET-03`, `PR-RET-05` | Chroma semantic search | 15 | full-mode smoke/eval | missing |
| RQ-009 | `PR-RET-03`, `PR-RET-05` | Hybrid RRF/related/clusters | 15 | retrieval evaluation | missing |
| RQ-010 | `PR-RET-06` | Dashboard KPI/analytics | 12, 16 | DuckDB/SQLite parity | partial |
| RQ-011 | `PR-SEC-01…02` | Limits/SSRF/escaping | 13, 17 | security suite | partial |
| RQ-012 | `PR-REC-05` | Backup/restore/rebuild | 18 | recovery transcript | missing |
| RQ-013 | `PR-REC-04` | Health/diagnostics | 10, 19 | component failure test | partial |
| RQ-014 | `PR-OPS-01…02` | CI, benchmark, release docs | 20, 21 | CI run + benchmark report | missing |

## Traceability rules

- PR description phải liệt kê requirement IDs.
- Test name/docstring chứa ID hoặc liên kết tới matrix.
- Khi requirement đổi, cập nhật product/architecture doc, epic, decision register và matrix trong cùng review cycle.
- Release M4 chỉ đạt khi mọi `RQ-001…014` là `verified` hoặc có waiver được ghi rõ.

## Hậu MVP IDs

`RQ-030` extension, `RQ-031` cloud sync, `RQ-032` collaboration và `RQ-033` AI được tạo khi discovery gate chốt outcome; trước đó chỉ theo dõi ở roadmap.
