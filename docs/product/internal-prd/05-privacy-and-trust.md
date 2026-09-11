# Internal PRD — Privacy and trust

## MVP data boundary

MVP xử lý text người dùng cung cấp, URL công khai và file local được chọn rõ ràng. Bookmark, browser profile, credentials, session state và browser-specific storage nằm ngoài phạm vi.

## Product promises

- Dữ liệu chính được lưu local theo mặc định.
- Nội dung chỉ gửi tới cloud provider khi người dùng chủ động cấu hình.
- Raw content, credential và secret không xuất hiện trong log.
- Xóa item ẩn dữ liệu khỏi user-facing flows ngay và có cleanup path.
- Backup/restore không phụ thuộc derived index.
- Health state giải thích capability degraded mà không tiết lộ dữ liệu nhạy cảm.

## Input and network trust

- File và request body có size/type limits.
- Extracted content là untrusted và phải escape khi render.
- URL ingestion chỉ truy cập nguồn public sau validation.
- Redirect, DNS resolution và địa chỉ kết nối được kiểm tra để chặn private-network access.
- Parser failure không được làm mất dữ liệu đã commit hoặc để lại trạng thái mơ hồ.

## Consent

- Import/upload xuất phát từ thao tác rõ ràng của người dùng.
- Provider/network capability cần cấu hình hoặc consent tương ứng.
- Simple mode dùng default an toàn; advanced mode không bỏ qua safety invariant.

## Future sensitive data

Cookies, passwords, session tokens, localStorage và IndexedDB chỉ được xem xét trong [Browser portability](../next-plan/00-browser-portability.md), sau threat model, compatibility matrix, consent, backup và rollback.
