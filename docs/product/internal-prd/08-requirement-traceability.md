# Internal PRD — Requirement traceability

**Status:** `active`

Bảng này nối product requirement với tài liệu thiết kế hệ thống và epic triển khai. `PR-*` là nguồn product intent canonical; `RQ-*` là requirement/evidence triển khai và không tạo một hệ product requirement thứ hai. Trạng thái thực tế vẫn được quản lý trong [implementation current state](../../plan/implementation/00-program/current-state.md) và [implementation traceability](../../plan/implementation/90-governance/requirement-traceability.md).

| Product requirements | Product outcome | Architecture/design reference | Epic | Acceptance |
| --- | --- | --- | --- | --- |
| `PR-CAP-01…05` | Capture và persistence an toàn | Architecture ingestion, data model | 10, 11, 13, 14 | M1, M2 |
| `PR-ORG-01…04` | Collections, notes, status, delete | Data model, core flows | 11, 12 | M1 |
| `PR-RET-01…06` | Filter, retrieval và analytics có fallback | Search/analytics architecture | 15, 16 | M3 |
| `PR-REC-01…05` | Feedback, retry, fallback và recovery | Indexing, lifecycle/recovery, operations | 14, 18, 19 | M2, M4 |
| `PR-UX-01…04` | Core flow dễ dùng và responsive | Design IA, flows, UI states | 12 | M1 |
| `PR-SEC-01…03` | Input/network/rendering/log privacy boundary | Security architecture, quality strategy | 13, 17, 20 | M2, M4 |
| `PR-SEC-04` | Cloud provider consent boundary | [Next Plan](../next-plan/README.md) | — | Future discovery; not an M1–M4 gate |
| `PR-OPS-01…02` | Repeatable release, upgrade và evidence | Quality gates, operations | 20, 21 | M4 |

## Epic coverage

| Epic | Requirement source | Canonical supporting docs |
| --- | --- | --- |
| 10 Foundation | `PR-CAP-05`, `PR-REC-03…04` | System overview, tech stack |
| 11 Data/migrations | `PR-CAP-02…05`, `PR-ORG-01…04` | Data model/ERD, lifecycle/recovery |
| 12 Dashboard | `PR-ORG-*`, `PR-UX-*` | Design IA, core flows, UI states |
| 13 Ingestion | `PR-CAP-01…04`, `PR-SEC-01…02` | Ingestion pipeline, security boundaries |
| 14 Index worker | `PR-CAP-05`, `PR-REC-01…03` | Indexing pipeline |
| 15 Search | `PR-RET-01…05` | Search and analytics architecture |
| 16 Analytics | `PR-RET-01`, `PR-RET-06`, `PR-REC-03…04` | Search and analytics architecture |
| 17 Security | `PR-SEC-01…03` | Privacy/trust, security boundaries |
| 18 Reliability | `PR-REC-02…05` | Lifecycle/recovery, operations |
| 19 Observability | `PR-REC-01…04` | Quality attributes, diagnostics operations |
| 20 Testing/CI | `PR-OPS-02` và acceptance của mọi `PR-*` | Test strategy, MVP quality gates |
| 21 Packaging | `PR-CAP-05`, `PR-REC-05`, `PR-OPS-01` | Setup, upgrade/release/rollback |

## Rules

- Không thêm epic behavior nếu chưa có product requirement hoặc decision/waiver tương ứng.
- `Current state` là bằng chứng hiện trạng; `Target state` chỉ là thiết kế đích.
- Khi requirement đổi, cập nhật PRD, traceability, architecture/design và epic reference trong cùng review cycle.
- `PR-SEC-04` and other Next Plan capabilities stay outside M1–M4 acceptance until the discovery gate passes and the Internal PRD is updated.
