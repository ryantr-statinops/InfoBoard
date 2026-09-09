# Kiến trúc dữ liệu và trách nhiệm database

## Nguyên tắc nền tảng

Mỗi hệ thống lưu trữ sở hữu một workload riêng. SQLite là nguồn dữ liệu sự thật; các database khác là lớp dẫn xuất, vận hành hoặc phân tích và phải có thể xây dựng lại từ SQLite cùng nội dung đã lưu.

## SQLite: nguồn dữ liệu sự thật của ứng dụng

SQLite lưu dữ liệu sản phẩm bền vững:

- items và metadata nguồn;
- snapshot text đã trích xuất;
- chunks và định danh ổn định của chúng;
- collections và quan hệ nhiều-nhiều giữa item–collection;
- trạng thái đọc và note cá nhân theo item;
- lịch sử search và metadata cấu hình.

SQLite sở hữu các quan hệ để render trang reader, library và collection. Một item không bao giờ được xem là hợp lệ chỉ vì index còn tồn tại.

## ChromaDB: semantic retrieval index

ChromaDB lưu embedding theo `chunk_id` của SQLite. Nó trả về các `chunk_id` cùng score tương đồng; InfoBoard sau đó lấy metadata item và đoạn text có thẩm quyền từ SQLite.

ChromaDB không phải metadata store chính. Collection của nó có thể bị xóa và xây dựng lại bằng cách index lại chunks từ SQLite.

## RocksDB: trạng thái vận hành và cache

RocksDB xử lý dữ liệu key-value có tần suất cao và có thể tái tạo:

- trạng thái job index (`queued`, `extracting`, `embedding`, `completed`, `failed`);
- embedding cache theo `content_hash + embedding_provider + model_version`;
- tiến độ ngắn hạn hoặc retry marker khi cần.

Embedding cache tránh tính lại embedding nếu nội dung giống nhau được import lần nữa. Trạng thái item bền vững, có ảnh hưởng đến UI hoặc khả năng phục hồi của người dùng, cũng phải được phản chiếu trong SQLite.

## DuckDB: lớp analytics chỉ đọc

DuckDB gắn SQLite ở chế độ chỉ đọc để trả lời các truy vấn dashboard mà không tạo bản sao application database. Nó phục vụ các phép tổng hợp như item theo collection/nguồn, phân bố trạng thái đọc và hoạt động theo thời gian.

DuckDB không được dùng cho transactional write. Các truy vấn analytics phải suy giảm an toàn nếu extension local không khả dụng.

## Luồng dữ liệu

```text
URL/file/text
  -> extractor và normalizer
  -> SQLite: item + snapshot + chunks
  -> RocksDB: cache/trạng thái job
  -> ChromaDB: embedding của chunks

Search query
  -> SQLite: keyword retrieval
  -> ChromaDB: semantic retrieval
  -> SQLite: hydrate kết quả hiển thị

SQLite (read-only attach)
  -> DuckDB aggregate queries
  -> dashboard
```

