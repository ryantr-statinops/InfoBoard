# InfoBoard — Implementation workspace

**Canonical context:** [Product](../../product/internal-prd/README.md) · [Architecture](../../architecture/README.md) · [Design](../../design/README.md) · [Quality](../../quality/README.md) · [Operations](../../operations/README.md)

Đây là cấu trúc triển khai theo milestone và epic package. Mỗi milestone có một README/acceptance gate; mỗi epic có `plan.md` (ý định, contract, commit slices) và `execution.md` (log triển khai, test, PR, evidence).

## Cách đọc và trạng thái

Đọc theo thứ tự: [program](00-program/roadmap.md) → milestone đang active → epic `plan.md` → `execution.md`. Trạng thái hợp lệ: `draft → ready → in_progress → review → merged → verified`; blocker dùng `blocked` và phải ghi điều kiện gỡ.

## Milestones

- [M1 — Local dashboard](10-m1-local-dashboard/README.md)
- [M2 — Reliable ingestion](20-m2-ingestion/README.md)
- [M3 — Search and insights](30-m3-search-and-insights/README.md)
- [M4 — Hardening and release](40-m4-release/README.md)
- [M5+ — Post-MVP discovery](50-post-mvp/README.md)

## Program và governance

- [Charter](00-program/charter.md) · [Current state](00-program/current-state.md) · [Roadmap](00-program/roadmap.md)
- [Workflow](00-program/delivery-workflow.md) · [Contracts](00-program/contracts.md) · [Risks](00-program/risks.md)
- [Operational runbooks](90-governance/runbooks.md) · [Decision register](90-governance/decision-register.md) · [Traceability](90-governance/requirement-traceability.md)

## Quy tắc package

`plan.md` là implementation intent và được sửa khi quyết định thay đổi. `execution.md` chỉ ghi bằng chứng thực tế, không thay đổi acceptance hồi tố. Feature có nhiều commit nhưng chỉ một review gate; sau khi user merge PR, cập nhật execution rồi mới chuyển package kế tiếp.
