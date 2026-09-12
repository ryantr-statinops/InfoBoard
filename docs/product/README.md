# InfoBoard — Product documentation

Product docs được chia theo thời điểm ra quyết định:

- [Internal PRD](internal-prd/README.md) là nguồn sự thật cho product definition và MVP scope hiện tại.
- [Next Plan](next-plan/README.md) chứa các hướng hậu MVP còn ở discovery, chưa phải implementation requirement.
- `docs/plan/` sẽ mô tả cách triển khai và milestone sau khi các documentation layer được khóa.

## Thứ tự đọc

1. Đọc [Internal PRD](internal-prd/README.md) để hiểu goal, user, scope, requirements, workflows và acceptance.
2. Tra [Domain model](internal-prd/04-domain-model.md) khi cần thống nhất thuật ngữ và entity semantics.
3. Đọc [Design](../design/README.md), [Architecture](../architecture/README.md), [Quality](../quality/README.md) và [Operations](../operations/README.md) để xem các layer hỗ trợ.
4. Chỉ đọc [Next Plan](next-plan/README.md) khi nghiên cứu capability hậu MVP.

## Boundary

```text
Internal PRD   = What and why we build now
Supporting docs = How the product should behave and be supported
Plan           = How and in which order we build it later
Next Plan      = What we may build after discovery
```

Một capability trong Next Plan không được coi là MVP commitment nếu chưa đi qua discovery gate và được đưa chính thức vào Internal PRD cùng implementation roadmap.
