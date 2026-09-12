# Internal PRD — Core workflows

## Add and organize

1. Người dùng chọn Text, URL hoặc File.
2. Hệ thống validate input trước khi tạo dữ liệu ngoài ý muốn.
3. Hệ thống tạo item/job và hiển thị trạng thái xử lý.
4. Người dùng gán collection hoặc giữ item trong inbox.
5. Item xuất hiện trong dashboard và tồn tại sau restart.

## Review and update

1. Người dùng lọc hoặc chọn item gần đây.
2. Detail panel hiển thị content, source, collections, note và trạng thái.
3. Người dùng cập nhật title, note, collection hoặc status.
4. Text source có thể được sửa và tạo version mới.
5. Đóng panel trả về đúng filter context trước đó.

## Search and retrieve

1. Người dùng nhập query và tùy chọn filter.
2. Core mode trả keyword results mà không cần full-mode dependency.
3. Người dùng nhận biết item từ title, source và excerpt rồi mở detail.
4. Related result không gồm chính item đang xem hoặc dữ liệu đã xóa/cũ version.

Khi người dùng đã chủ động cấu hình full mode, semantic/hybrid results có thể
được bổ sung vào keyword results. Full-mode failure chỉ tạo degraded state sau
khi full mode đã được cấu hình; keyword/read flow vẫn tiếp tục hoạt động.

## Failure and retry

1. UI hiển thị bước xử lý và lý do lỗi an toàn.
2. Lỗi có thể phục hồi cung cấp retry; retry không tạo duplicate.
3. Restart đưa job dang dở về trạng thái có thể tiếp tục.
4. Semantic hoặc analytics lỗi làm UI degraded nhưng giữ đọc và keyword search.

## Delete and recoverability

1. Người dùng xác nhận xóa item.
2. Item bị ẩn ngay bằng soft delete.
3. Derived data được cleanup theo job idempotent.
4. Restore dữ liệu chính chạy trước rebuild derived indexes.
