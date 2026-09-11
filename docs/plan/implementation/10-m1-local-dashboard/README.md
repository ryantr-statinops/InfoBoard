# M1 — Local dashboard

**Gate:** dashboard local dùng được end-to-end
**Dependencies:** program contracts, current SQLite schema

## Epic packages

- [10 — Application foundation](10-application-foundation/plan.md)
- [11 — Data model và migrations](11-data-and-migrations/plan.md)
- [12 — Dashboard và item workspace](12-dashboard/plan.md)

## Thứ tự

`10 → 11 → 12`; có thể chạy UI spike của `12` song song nhưng không merge behavior trước contract của `10–11`.

## Acceptance gate

Fresh install, thêm text, list/filter, mở detail, thêm note/collection, sửa status và soft-delete; restart không mất dữ liệu; desktop/mobile không tràn layout.
