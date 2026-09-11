# 12 — Dashboard và item workspace

**Status:** `in_progress`  
**Milestone:** M1  
**Dependencies:** 10, 11, 13, 15, 16  
**Reference:** `docs/plan/02-mvp-experience.md`

## Outcome

Dashboard one-page responsive là nơi người dùng thêm, lọc, tìm và mở thông tin. Desktop dùng sidebar + detail drawer; mobile dùng một cột và panel toàn màn hình.

## UI contract

- Jinja2 render page/partial; HTMX tải list, KPI, detail và toast mà không cần SPA build.
- Static assets local, không phụ thuộc CDN; favicon được phục vụ tại `/favicon.ico`.
- Filter được phản ánh trong URL: `q`, collection, source, status, date range.
- Form lỗi giữ nguyên title/content; thành công trigger reload list/KPI.
- Tất cả text user-generated được escape; không dùng `|safe` cho content.

## Feature slices

1. Shell: sidebar, header/search, KPI cards, recent list, responsive breakpoints.
2. Create: text/URL/file selector, collection picker, submit/loading/success/error.
3. Filter/search: shared query params, empty/loading/indexing/degraded states.
4. Detail: content, source, collections, notes, status, related items, close-to-previous-filter.
5. Mutations: edit title/status/text (text source only), add/remove collection, note CRUD, confirm delete.

## Non-goals

Không có rich text editor, drag-and-drop kanban, realtime multi-user hay client-side router.

## HTMX fragments

`GET /partials/dashboard`, `/partials/items`, `/partials/items/{id}`, `/partials/analytics`, `/partials/toast`; fragment trả HTML cùng context contract với API. API vẫn là public integration surface.

## Commit slices

1. `refactor: split dashboard template into partials and static assets`
2. `feat: build responsive dashboard shell and favicon`
3. `feat: add htmx item creation and filter refresh`
4. `feat: add item detail drawer and mutation controls`
5. `feat: add dashboard loading empty and degraded states`
6. `test: cover dashboard fragments and item workspace`

## Accessibility và responsive acceptance

Keyboard có thể mở/đóng panel, focus visible, label cho form và `aria-live` cho toast. Không tràn ngang tại 390 px; layout mục tiêu 1440 px khớp mockup; content dài wrap và có giới hạn chiều cao hợp lý.

## Tests

Template render 200, fragment context, form error preservation, escaped content, filter URL, detail open/close, note/collection mutation và soft-delete visibility. Manual smoke test browser trên desktop/mobile sau mỗi PR.

## Review gate

Reviewer chạy app, thêm một item, lọc, mở detail, thêm note/collection, sửa status, xóa và reload để xác nhận state bền vững.

## Execution log

Dashboard hiện mới có page/form/list cơ bản; các slices trên chưa triển khai đầy đủ.
