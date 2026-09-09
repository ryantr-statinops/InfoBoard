# Kế hoạch chất lượng và roadmap

## Tiêu chí nghiệm thu MVP

- Import fixture từ một bài HTML công khai, một PDF, một file Markdown, một file text và text nhập tay.
- Xác nhận mỗi lần import tạo snapshot có thể đọc và SQLite chunks xác định được.
- Đặt một item trong nhiều collections; cập nhật state và note; restart app và xác nhận dữ liệu còn nguyên.
- Import lại nội dung giống nhau và xác nhận lần index thứ hai tái sử dụng RocksDB embedding cache.
- Xác nhận keyword và semantic search trả đúng item nguồn cùng đoạn trích; collection/state filters hoạt động cho cả hai nhánh.
- Mô phỏng lỗi extract và embedding, hiển thị trạng thái lỗi và kiểm tra retry.
- So sánh số liệu dashboard DuckDB với các rows nguồn trong SQLite.
- Xác nhận URL private/internal và định dạng file không hỗ trợ bị từ chối trước khi xử lý nội dung.

## Observability

Lưu outcome có cấu trúc của import, thời gian index, số lần cache hit/miss của embedding, latency search và result count. Các metrics local này đủ để chẩn đoán MVP và nuôi dashboard mà không cần telemetry bên ngoài.

## Thứ tự triển khai

1. Thiết lập cấu hình project, SQLite schema, vị trí local storage và library UI cơ bản.
2. Triển khai manual text + file ingestion, snapshot reader, collections, state và note.
3. Thêm safe public-article extraction và background indexing state.
4. Thêm local embedding, ChromaDB semantic search và hybrid search results.
5. Thêm RocksDB cache/retry và DuckDB dashboard aggregates.
6. Thêm cloud embedding provider có cấu hình sau khi local path đã được kiểm chứng.
7. Thêm bookmark-HTML import, rồi browser extension, như adapter trên ingestion API ổn định.

## Non-goals của MVP

Phiên bản đầu không cam kết multi-user access, remote sync, cộng tác real-time, recommendation tự động, OCR, video transcript, lưu ảnh bài viết hoặc highlight theo vùng text. Tất cả có thể được thêm sau mà không thay đổi vai trò sở hữu dữ liệu chính của SQLite.

