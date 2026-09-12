# Internal PRD — Users and use cases

## Primary user

Người dùng phổ thông muốn lưu bài viết, ghi chú hoặc file và tìm lại sau này mà không cần hiểu database, indexing hay embedding. Flow mặc định phải ít bước, dùng ngôn ngữ rõ ràng và có lỗi phục hồi được.

### Nhu cầu chính

- Lưu nhanh một thông tin và biết nó đã được lưu thành công.
- Đọc lại nội dung khi nguồn không còn hoặc không có mạng.
- Gom thông tin theo project/lĩnh vực mà không bị giới hạn một nhóm duy nhất.
- Tìm bằng keyword trong core mode; tìm theo ý nghĩa gần đúng là tùy chọn của full mode.
- Không mất dữ liệu khi ứng dụng hoặc thành phần nâng cao gặp lỗi.

## Advanced user path

Power user và privacy-conscious user cần kiểm soát local storage, provider, backup/rebuild và diagnostics. Advanced options được tách khỏi flow mặc định.

## Core use cases

| ID | Use case | Outcome |
| --- | --- | --- |
| UC-01 | Thêm text | Item được lưu vào inbox và có thể tìm lại sau restart. |
| UC-02 | Nhập URL/file | Snapshot được tạo cùng provenance và trạng thái xử lý rõ ràng. |
| UC-03 | Tổ chức item | Item được gán nhiều collection, note và trạng thái. |
| UC-04 | Duyệt dashboard | Người dùng lọc, mở detail và quay lại đúng context. |
| UC-05 | Tìm kiếm | Keyword luôn dùng được; semantic chỉ bổ sung khi full mode được cấu hình. |
| UC-06 | Phục hồi lỗi | Người dùng retry job hoặc tiếp tục core flow khi derived dependency lỗi. |
| UC-07 | Bảo toàn dữ liệu | Backup/restore giữ dữ liệu chính và cho phép rebuild index. |

## Experience modes

- **Simple mode:** default an toàn, ít cấu hình, feedback bằng ngôn ngữ phổ thông.
- **Advanced mode:** provider, maintenance, diagnostics và technical metadata; không thay đổi semantics của dữ liệu chính.
