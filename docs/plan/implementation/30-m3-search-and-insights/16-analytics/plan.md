# 16 — Analytics và insights

**Status:** `partial`
**Milestone:** M3–M4
**Dependencies:** 11, 12
**Adapters:** DuckDB optional, SQLite fallback

## Outcome

Dashboard hiển thị KPI và aggregate nhất quán với bộ filter; analytics không ghi transactional data và không làm chậm hoặc khóa SQLite lâu.

## Query contract

- `items_total`, `chunks_total`, `collections_total`, `active_jobs`.
- Activity theo ngày tạo/import trong range giới hạn.
- Phân bố source/status/collection; ghi chú item nhiều collection có thể đếm lặp.
- Cluster summary và trạng thái insufficient-data.
- Response có `engine: duckdb|sqlite-fallback` và `degraded` reason nếu cần.

## Design

DuckDB mở SQLite read-only qua adapter/extension; mọi query có date/row bound và parameter binding. Nếu extension unavailable hoặc query lỗi, dùng aggregate SQLite tương đương và báo engine fallback.

## Commit slices

1. `feat: define analytics result schema and shared filters`
2. `feat: add duckdb read-only aggregate queries`
3. `feat: add sqlite analytics fallback and degraded status`
4. `feat: render activity distributions and cluster cards`
5. `test: cover analytics parity and filter consistency`

## Failure/security

Không cho user gửi SQL; không copy database sang DuckDB persistent không cần thiết; không ghi raw query/content vào log. Timeout hoặc range quá lớn trả lỗi bounded, không block request vô hạn.

## Acceptance

KPI khớp SQLite fixture; filter list/search/analytics cùng kết quả phạm vi; DuckDB tắt vẫn dashboard usable; chart empty/loading/error states render được.

## Review gate

So sánh output DuckDB và SQLite trên fixture, kiểm tra read-only và đo query với 10.000 chunks.

## Execution log

`app/analytics.py` hiện có adapter tối thiểu và fallback; chưa có toàn bộ aggregate/UI.
