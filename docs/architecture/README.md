# InfoBoard — Architecture

Architecture docs mô tả cấu trúc hệ thống và các invariants kỹ thuật. Mỗi tài liệu phân biệt:

- **Current state:** capability đã có theo snapshot hiện hành.
- **Target state:** thiết kế đích; không phải bằng chứng hoàn thành.

## Đọc theo thứ tự

1. [System overview](00-system-overview.md)
2. [Tech stack](01-tech-stack.md)
3. [Data model and ERD](02-data-model-and-erd.md)
4. [Ingestion pipeline](03-ingestion-pipeline.md)
5. [Indexing pipeline](04-indexing-pipeline.md)
6. [Search and analytics](05-search-and-analytics.md)
7. [Lifecycle and recovery](06-lifecycle-and-recovery.md)
8. [Security boundaries](07-security-boundaries.md)
9. [Architecture decisions](08-architecture-decisions.md)

Implementation status và evidence nằm tại [`docs/plan/implementation/`](../plan/implementation/), không được suy ra từ target diagrams.
