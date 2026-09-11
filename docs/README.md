# InfoBoard — Documentation

Tài liệu InfoBoard được tổ chức theo chuỗi quyết định từ sản phẩm đến thực thi:

```text
Product → Architecture → Design → Quality/Operations → Implementation
```

## Đọc theo vai trò

- Product/design: bắt đầu tại [Internal PRD](product/internal-prd/README.md), sau đó đọc [Design](design/README.md).
- Engineering: đọc [Internal PRD](product/internal-prd/README.md), [Architecture](architecture/README.md) và [Implementation](plan/implementation/README.md).
- QA/reviewer: đọc [Quality](quality/README.md), [traceability](product/internal-prd/08-requirement-traceability.md) và milestone acceptance.
- Maintainer/operator: đọc [Operations](operations/README.md) và implementation execution logs.
- Future discovery: đọc [Next Plan](product/next-plan/README.md); nội dung này không phải MVP commitment.

## Nguồn sự thật

| Layer | Câu hỏi | Nội dung canonical |
| --- | --- | --- |
| Product | Xây gì, cho ai, thành công là gì? | `docs/product/` |
| Architecture | Hệ thống vận hành và giữ dữ liệu thế nào? | `docs/architecture/` |
| Design | Người dùng tương tác thế nào? | `docs/design/` |
| Quality | Điều kiện nào chứng minh đủ tốt? | `docs/quality/` |
| Operations | Cài đặt, phục hồi và nâng cấp thế nào? | `docs/operations/` |
| Implementation | Chia việc, trạng thái và evidence ở đâu? | `docs/plan/implementation/` |

Các file `docs/plan/00–04` chỉ là compatibility entry points và không còn là nguồn nội dung canonical.
