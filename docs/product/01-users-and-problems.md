# InfoBoard — Users and problems

## Primary user: người dùng phổ thông

Họ muốn giữ lại bài viết, link, ghi chú, file và bookmark mà không phải nhớ dữ liệu đang nằm ở browser hay thư mục nào. Họ cần một flow đơn giản, preview rõ ràng và mặc định an toàn.

## Advanced path: power user và privacy-conscious user

Họ cần local-first, export đầy đủ, lựa chọn profile/source, kiểm soát format, conflict handling, audit/log và khả năng chạy workflow bằng CLI. Advanced path bổ sung quyền kiểm soát nhưng không làm trải nghiệm mặc định phức tạp.

## Problems

- Thông tin bị phân tán giữa browser, bookmark, file và web.
- Chuyển Chrome sang Firefox hoặc giữa các môi trường dễ làm mất context và dữ liệu.
- Bookmark/URL có thể mang theo được, nhưng cookies, session, localStorage và IndexedDB phụ thuộc browser và khó portable.
- Người dùng thường nhớ chủ đề hoặc ý nghĩa, không nhớ chính xác nơi đã lưu.
- Các công cụ migration thường tách rời giữa import, lưu trữ lâu dài và tìm kiếm lại.

## Product opportunity

InfoBoard nối capture, knowledge retrieval và portability trong một local workspace: dữ liệu được hiểu theo nguồn, lưu thành snapshot có provenance và có thể export theo từng lớp dữ liệu.
