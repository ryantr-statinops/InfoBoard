# InfoBoard — Implementation plan

Tài liệu trong thư mục này chuyển tầm nhìn sản phẩm và kiến trúc cấp cao thành các epic có thể triển khai, kiểm thử và review. Các file `docs/plan/00–04` vẫn là nguồn mô tả cấp cao; khi có khác biệt, quyết định mới nhất trong `04-cross-cutting-contracts.md` và decision register được ưu tiên cho implementation.

## Cách đọc

1. Đọc `00-program-charter.md` và `01-current-state-and-gap.md` để biết mục tiêu và hiện trạng.
2. Dùng `02-master-roadmap.md` để chọn epic đủ dependency và trạng thái `ready`.
3. Đọc contract chung trong `04-cross-cutting-contracts.md` trước khi sửa API, schema hoặc UI.
4. Theo `03-delivery-workflow.md` để thực hiện từng feature, commit và PR.
5. Ghi kết quả vào execution log của epic, decision register và traceability matrix.

## Quy ước trạng thái

```text
draft → ready → in_progress → review → merged → verified
                     └──────→ blocked
```

- `draft`: còn thiếu quyết định hoặc phụ thuộc.
- `ready`: decision-complete, có thể bắt đầu.
- `in_progress`: đang code trên `dev`.
- `review`: đã push, chờ user kiểm tra/merge PR.
- `merged`: PR đã merge vào `main`.
- `verified`: test và acceptance gate đã được xác nhận sau merge.
- `blocked`: có blocker cụ thể; phải ghi bằng chứng và điều kiện gỡ blocker.

## Mục lục

### Điều phối

- [00 — Program charter](00-program-charter.md)
- [01 — Current state và gap](01-current-state-and-gap.md)
- [02 — Master roadmap](02-master-roadmap.md)
- [03 — Delivery workflow](03-delivery-workflow.md)
- [04 — Cross-cutting contracts](04-cross-cutting-contracts.md)
- [05 — Dependencies và risks](05-dependencies-and-risks.md)

### MVP epics

- [10 — Application foundation](10-application-foundation.md)
- [11 — Data model và migrations](11-data-model-and-migrations.md)
- [12 — Dashboard và item workspace](12-dashboard-and-item-workspace.md)
- [13 — Ingestion sources](13-ingestion-sources.md)
- [14 — Indexing worker và cache](14-indexing-worker-and-cache.md)
- [15 — Search và discovery](15-search-and-discovery.md)
- [16 — Analytics và insights](16-analytics-and-insights.md)
- [17 — Security và privacy](17-security-and-privacy.md)
- [18 — Reliability, backup và recovery](18-reliability-backup-and-recovery.md)
- [19 — Observability và diagnostics](19-observability-and-diagnostics.md)
- [20 — Testing, performance và CI](20-testing-performance-and-ci.md)
- [21 — Packaging, release và upgrades](21-packaging-release-and-upgrades.md)

### Hậu MVP (roadmap + discovery gate)

- [30 — Browser extension và capture](30-browser-extension-and-capture.md)
- [31 — Cloud sync và multi-device](31-cloud-sync-and-multi-device.md)
- [32 — Identity, sharing và collaboration](32-identity-sharing-and-collaboration.md)
- [33 — AI features và providers](33-ai-features-and-providers.md)

### Vận hành và quản trị quyết định

- [90 — Operational runbooks](90-operational-runbooks.md)
- [91 — Decision register](91-decision-register.md)
- [92 — Requirement traceability](92-requirement-traceability.md)

## Trạng thái hiện tại

| Epic | Trạng thái | Ghi chú |
| --- | --- | --- |
| 10–11 | `partial` | Skeleton FastAPI/SQLite có, cần tách lớp và migration versioned. |
| 12 | `in_progress` | Dashboard cơ bản có; detail workspace và HTMX chưa hoàn thiện. |
| 13–14 | `partial` | Text/file/URL và worker mô phỏng có; cần extractor/worker production. |
| 15–16 | `partial` | FTS5, RRF utility và analytics fallback có; Chroma/DuckDB đầy đủ chưa có. |
| 17–21 | `draft` | Một phần giới hạn/bảo mật/test có, cần hoàn thiện theo epic. |
| 30–33 | `discovery` | Chỉ roadmap, chưa implementation-ready. |

## Quy tắc cập nhật

Mỗi thay đổi quyết định phải cập nhật epic liên quan và `91-decision-register.md`. Mỗi yêu cầu phải có dòng mapping trong `92-requirement-traceability.md`. Không đánh dấu `verified` nếu chưa có test/evidence hoặc chưa hoàn tất review gate.
