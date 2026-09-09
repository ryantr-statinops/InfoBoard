# Kế hoạch triển khai

## Hình dạng ứng dụng

Xây dựng web app local bằng Python, FastAPI, Jinja templates và HTMX. Browser là UI shell; chưa cần đóng gói desktop. Giữ ứng dụng theo các module ingestion, library data, retrieval, analytics và rendering.

## Ingestion pipeline

1. Validate input và tạo SQLite item với trạng thái index.
2. Extract và normalize text thành snapshot bền vững.
3. Chia snapshot thành chunks xác định được và lưu trong SQLite.
4. Tính content hash cho từng chunk.
5. Kiểm tra RocksDB xem embedding theo hash, provider và model version đã tồn tại chưa.
6. Tạo embedding còn thiếu qua provider đã cấu hình và upsert vào ChromaDB theo chunk ID.
7. Đánh dấu job là hoàn tất hoặc lỗi, đồng thời có đường retry.

Implementation cần định nghĩa interface `EmbeddingProvider`, với trách nhiệm duy nhất là tạo embedding theo batch text. Implementation mặc định chạy sentence-transformers local. Cloud provider là implementation tùy chọn, chỉ bật khi có cấu hình rõ ràng.

## Retrieval

Triển khai keyword retrieval và semantic retrieval thành hai service riêng, sau đó merge và deduplicate chunk results trước khi hydrate kết quả từ SQLite. Nếu Chroma index đang thiếu hoặc được xây dựng lại, keyword search vẫn dùng được. Lưu latency và result count vào SQLite search history.

## Dashboard

Chỉ dùng DuckDB phía sau analytics service. Gắn SQLite read-only, chạy aggregate queries và trả view-model cho templates thay vì trả SQL. Dashboard không được write qua DuckDB.

## Độ tin cậy và phục hồi

Service phải an toàn khi restart: dữ liệu SQLite vẫn dùng được nếu ChromaDB hoặc RocksDB bị reset, và app có re-index để tạo lại derived state. Import thất bại vẫn giữ metadata nguồn và lý do lỗi để retry hoặc xóa.

## Bảo mật cơ bản

Trước khi fetch URL, validate scheme, resolve host và từ chối loopback, private, link-local và reserved network addresses. Giới hạn kích thước tải xuống và thời gian extract. Luôn coi text import là dữ liệu, không phải chỉ dẫn để thực thi.

