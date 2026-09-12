# Internal PRD — Requirement traceability

**Status:** `active`

Bảng này nối product requirement với tài liệu thiết kế hệ thống và epic triển khai. `PR-*` là nguồn product intent canonical; `RQ-*` là requirement/evidence triển khai và không tạo một hệ product requirement thứ hai. Trạng thái thực tế vẫn được quản lý trong [implementation current state](../../plan/implementation/00-program/current-state.md) và [implementation traceability](../../plan/implementation/90-governance/requirement-traceability.md).

| Product requirements | Product outcome | Architecture/design reference | Epic | Acceptance |
| --- | --- | --- | --- | --- |
| `PR-CAP-01…05` | Capture và persistence an toàn | [Ingestion](../../architecture/03-ingestion-pipeline.md), [data model](../../architecture/02-data-model-and-erd.md), [interfaces](../../architecture/09-interface-contracts.md) | 10, 11, 13, 14 | M1, M2 |
| `PR-ORG-01…04` | Collections, notes, status, delete | [Data model](../../architecture/02-data-model-and-erd.md), [core flows](../../design/01-core-flows-and-screens.md) | 11, 12 | M1 |
| `PR-RET-01…06` | Core keyword/filter/analytics; `PR-RET-03` conditional full mode | [Search/analytics](../../architecture/05-search-and-analytics.md), [interfaces](../../architecture/09-interface-contracts.md), [design flows](../../design/01-core-flows-and-screens.md) | 15, 16 | M3 core + optional full-mode gate |
| `PR-REC-01…05` | Feedback, retry, fallback và recovery | [Indexing](../../architecture/04-indexing-pipeline.md), [lifecycle/recovery](../../architecture/06-lifecycle-and-recovery.md), [operations](../../operations/README.md) | 14, 18, 19 | M2, M4 |
| `PR-UX-01…04` | Core flow dễ dùng và responsive | [Design requirement map](../../design/README.md), [flows](../../design/01-core-flows-and-screens.md), [UI states](../../design/02-ui-states-and-responsive.md) | 12 | M1–M4 by surface |
| `PR-SEC-01…03` | Input/network/rendering/log privacy boundary | [Security architecture](../../architecture/07-security-boundaries.md), [quality strategy](../../quality/01-test-strategy.md) | 13, 17, 20 | M2, M4 |
| `PR-SEC-04` | Cloud provider consent boundary | [Next Plan](../next-plan/README.md) | — | Future discovery; not an M1–M4 gate |
| `PR-OPS-01…02` | Repeatable release, upgrade và evidence | [Quality gates](../../quality/03-mvp-quality-gates.md), [operations](../../operations/README.md) | 20, 21 | M4 |

## Epic coverage

| Epic | Requirement source | Canonical supporting docs |
| --- | --- | --- |
| 10 Foundation | `PR-UX-01`, `PR-REC-04`, `PR-OPS-01` | System overview, tech stack |
| 11 Data/migrations | `PR-CAP-02`, `PR-CAP-04`, `PR-ORG-01`, `PR-ORG-02`, `PR-ORG-04`, `PR-REC-01`, `PR-REC-02`, `PR-REC-03` | Data model/ERD, lifecycle/recovery |
| 12 Dashboard | `PR-CAP-01`, `PR-CAP-04`, `PR-ORG-01`, `PR-ORG-02`, `PR-ORG-03`, `PR-ORG-04`, `PR-UX-01`, `PR-UX-02`, `PR-UX-03`, `PR-UX-04`, `PR-REC-04` | Design IA, core flows, UI states |
| 13 Ingestion | `PR-CAP-01`, `PR-CAP-02`, `PR-CAP-03`, `PR-SEC-01`, `PR-SEC-02` | Ingestion pipeline, security boundaries |
| 14 Index worker | `PR-CAP-05`, `PR-REC-01`, `PR-REC-02`, `PR-REC-03` | Indexing pipeline |
| 15 Search | `PR-RET-01`, `PR-RET-02`, conditional full-mode `PR-RET-03`, `PR-RET-04`, `PR-RET-05` | Search and analytics architecture |
| 16 Analytics | `PR-RET-01`, `PR-RET-06`, `PR-REC-03`, `PR-REC-04` | Search and analytics architecture |
| 17 Security | `PR-SEC-01`, `PR-SEC-02`, `PR-SEC-03` | Privacy/trust, security boundaries |
| 18 Reliability | `PR-REC-01`, `PR-REC-02`, `PR-REC-03`, `PR-REC-04`, `PR-REC-05` | Lifecycle/recovery, operations |
| 19 Observability | `PR-REC-04` | Quality attributes, diagnostics operations |
| 20 Testing/CI | `PR-OPS-02` | Test strategy, MVP quality gates |
| 21 Packaging | `PR-OPS-01`, `PR-OPS-02` | Setup, upgrade/release/rollback |

## Rules

- Không thêm epic behavior nếu chưa có product requirement hoặc decision/waiver tương ứng.
- `Current state` là bằng chứng hiện trạng; `Target contract` chỉ là thiết kế đích.
- `PR-RET-03` chỉ trở thành core-release blocker nếu full mode được đưa vào release scope.
- Khi requirement đổi, cập nhật PRD, traceability, architecture/design và epic reference trong cùng review cycle.
- `PR-SEC-04` and other Next Plan capabilities stay outside M1–M4 acceptance until the discovery gate passes and the Internal PRD is updated.
