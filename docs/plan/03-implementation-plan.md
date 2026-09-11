# Implementation plan

Tài liệu triển khai chi tiết đã được tổ chức lại tại [`docs/plan/implementation/`](implementation/).

## Cách bắt đầu

1. Đọc [implementation README](implementation/README.md) để biết mục lục và trạng thái.
2. Xem [current state và gap](implementation/01-current-state-and-gap.md).
3. Theo [master roadmap](implementation/02-master-roadmap.md) để chọn epic đúng dependency.
4. Tuân thủ [delivery workflow](implementation/03-delivery-workflow.md) khi code trên nhánh `dev`.

## Phạm vi

Bộ tài liệu mới bao phủ application foundation, data/migrations, dashboard, ingestion, indexing, search, analytics, security, reliability, observability, testing, packaging và các hướng hậu MVP. Các file `00–04` còn lại vẫn giữ vai trò product/architecture reference cấp cao.

## Trạng thái

Epic MVP `10–21` được mô tả implementation-ready hoặc partial theo hiện trạng code. Epic hậu MVP `30–33` mới ở mức roadmap + discovery gate; không triển khai chúng trước khi các quyết định sản phẩm được khóa.
