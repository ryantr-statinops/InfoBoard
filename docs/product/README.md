# InfoBoard — Product documentation

Product docs được chia theo thời điểm ra quyết định:

- [Internal PRD](internal-prd/README.md) là nguồn sự thật cho sản phẩm đang build và nghiệm thu trong MVP hiện tại.
- [Next Plan](next-plan/README.md) chứa các hướng hậu MVP còn ở discovery, chưa phải implementation requirement.
- [`docs/plan/implementation/`](../plan/implementation/) là nguồn sự thật cho cách triển khai, milestone, test và execution evidence.

## Thứ tự đọc

1. Đọc [Internal PRD](internal-prd/README.md) để hiểu goal, user, scope, requirements, workflows và acceptance.
2. Tra [Domain model](internal-prd/domain-model.md) khi cần thống nhất thuật ngữ và entity semantics.
3. Đọc [implementation current state](../plan/implementation/00-program/current-state.md) và [roadmap](../plan/implementation/00-program/roadmap.md) để biết trạng thái build.
4. Chỉ đọc [Next Plan](next-plan/README.md) khi nghiên cứu capability hậu MVP.

## Boundary

```text
Internal PRD   = What and why we build now
Implementation = How and in which order we build it
Next Plan      = What we may build after discovery
```

Một capability trong Next Plan không được coi là MVP commitment nếu chưa đi qua discovery gate và được đưa chính thức vào Internal PRD cùng implementation roadmap.
