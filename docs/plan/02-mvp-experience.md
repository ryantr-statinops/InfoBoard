# Trải nghiệm và giao diện MVP

## Nguồn nhập được hỗ trợ

Phiên bản đầu nhận:

- URL bài viết công khai;
- PDF;
- Markdown;
- file plain-text;
- text nhập thủ công.

Lưu URL tạo ra snapshot text chứ không chỉ bookmark. Importer chỉ chấp nhận URL HTTP(S) có địa chỉ công khai; trang yêu cầu đăng nhập, ứng dụng mạng xã hội, trang video và lưu ảnh ngoài phạm vi MVP.

## Các màn hình chính

### Library

Danh sách item có filter, hiển thị tiêu đề, nguồn, collections, trạng thái đọc, thời điểm lưu và kết quả index. Người dùng có thể lọc theo collection, loại nguồn, trạng thái hoặc truy vấn search.

### Collections

Collection đại diện cho project hoặc lĩnh vực như `MLOps`, `Career`, hay `Research`. Một item có thể thuộc nhiều collection. Collection là cơ chế tổ chức chính; tags chưa cần ở bản đầu.

### Item reader

Reader hiển thị snapshot text đã lưu, URL gốc hoặc metadata file, các collection liên quan, trạng thái nhẹ (`unread`, `reading`, `completed`, `archived`) và một note cá nhân theo item.

### Search

Search kết hợp keyword retrieval của SQLite và semantic retrieval của Chroma. Kết quả hiển thị item nguồn, đoạn trích liên quan, collections và score/lý do đủ để người dùng đánh giá độ phù hợp. Filters phải thu hẹp nhất quán cả hai nhánh tìm kiếm.

### Dashboard

Dashboard trả lời các câu hỏi thực dụng: Gần đây đã lưu gì? Collection nào đang tăng? Đã hoàn thành bao nhiêu so với chưa đọc? Nguồn và chủ đề nào chiếm nhiều nhất? MVP không bao gồm recommendation hay auto-summary.

## HTTP interface

- `POST /items`: lưu URL, upload file hỗ trợ hoặc gửi text; trả về item mới ở trạng thái `indexing`.
- `GET /items` và `GET /items/{id}`: liệt kê và đọc item đã lưu.
- `PATCH /items/{id}`: cập nhật trạng thái đọc, note, title hoặc collection memberships.
- `GET`, `POST`, `PATCH /collections`: quản lý collections.
- `POST /search`: chạy hybrid retrieval với filter tùy chọn theo source, collection và state.
- `GET /analytics`: trả về aggregate cho dashboard.

## Phần được hoãn rõ ràng

Import bookmark HTML và browser extension là input adapter ở các giai đoạn sau. Chúng phải dùng chung ingestion pipeline nhưng không thuộc MVP. User accounts, cloud sync, collaboration, snapshot ảnh web, highlight theo vị trí text và recommendation feed cũng được hoãn.

