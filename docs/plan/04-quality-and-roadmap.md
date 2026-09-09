# Kiểm thử và lộ trình

## Thứ tự thực hiện

1. Thử kỹ thuật trên Python 3.12/Linux: cài binding RocksDB, Chroma, DuckDB SQLite extension, embedding Việt/Anh; khóa phiên bản và model revision sau khi chạy được.
2. Nền tảng: uv/lockfile, schema/migration SQLite, dashboard và text nhập tay, collections, note, trạng thái.
3. Import URL/PDF/MD/TXT và worker có checkpoint/retry.
4. FTS5, embedding local, Chroma, cache RocksDB, hybrid search, related và nhóm nội dung.
5. DuckDB analytics, giao diện responsive, backup/restore và kiểm thử tích hợp.
6. Adapter cloud embedding; sau MVP mới thêm bookmark HTML rồi browser extension.

## Nghiệm thu

- Năm nguồn nhập tạo nội dung đúng; text/metadata/collections còn nguyên sau restart.
- Trùng input không sinh item ngoài ý muốn; embedding cache tái sử dụng với cùng cấu hình, không dùng nhầm model.
- Sửa text, retry sau crash và đổi model không trả chunks cũ hay mất note.
- Xóa item dọn vector; xóa collection giữ item; backup/restore phục hồi dữ liệu chính và index lại được.
- Search Việt/Anh, có/không dấu, filters và fallback keyword đúng; bộ 20 query có item đích trong top 5 ở ít nhất 16 query.
- Analytics khớp SQLite; nhóm nội dung có empty state; lỗi từng database dẫn xuất không làm mất dữ liệu chính.
- Test URL redirect vào mạng nội bộ, file quá lớn, PDF không text, escape nội dung và yêu cầu ghi từ origin khác.
- Kiểm tra UI desktop 1440 px/mobile 390 px: thêm item → mở panel → sửa note → tìm lại; không tràn ngang.
- Đo trên 1.000 item/10.000 chunks: mục tiêu search p95 dưới 1 giây khi model đã tải, loại thời gian khởi động; ghi cấu hình máy và kết quả thực, không coi là bảo đảm phần cứng.

## Công cụ và quan sát

- pytest + httpx cho unit/integration; bài kiểm tra recovery dùng database thật trong thư mục tạm.
- Ruff, type check và test chạy trong CI; kiểm tra embedding thật ở smoke test riêng, unit test dùng provider giả.
- Log job_id, bước xử lý, thời gian, mã lỗi, cache hit/miss; không log text hay secret.
- Dependency/model/extension cần mạng được tải trong bước setup; khi thiếu phải báo hướng dẫn, không tự tải trong một truy vấn dashboard.
