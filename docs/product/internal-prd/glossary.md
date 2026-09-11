# InfoBoard — Product glossary

| Thuật ngữ | Định nghĩa |
| --- | --- |
| `item` | Đơn vị thông tin chính gồm title, content/snapshot, source, collections, notes, status và timestamps. |
| `source` | Nơi hoặc hình thức tạo ra item, ví dụ text nhập tay, URL, PDF, Markdown hoặc TXT. |
| `snapshot` | Bản nội dung đã trích xuất và lưu tại một thời điểm để đọc, tìm kiếm và phục hồi độc lập với nguồn. |
| `collection` | Nhóm do người dùng quản lý; một item có thể thuộc nhiều collection. |
| `content version` | Phiên bản nội dung hiện hành của item; tăng khi nội dung text có thể sửa được thay đổi. |
| `derived store` | Index, cache hoặc analytics store có thể rebuild từ dữ liệu chính. |
| `portable content` | Nội dung và metadata có thể import/export tương đối độc lập với browser, như bookmark, URL, title và page snapshot. |
| `browser-specific state` | Dữ liệu phụ thuộc profile hoặc runtime browser, như cookies, session, localStorage, IndexedDB và Service Worker storage. |
| `simple mode` | Trải nghiệm mặc định ít cấu hình, dùng default an toàn và thông báo thân thiện. |
| `advanced mode` | Lựa chọn nâng cao cho provider, source/profile, export format, conflict handling, diagnostics hoặc CLI. |
| `core mode` | Chế độ bắt buộc với SQLite/FTS5, không phụ thuộc semantic hoặc analytics store. |
| `full mode` | Chế độ có thêm embedding, vector index, cache và analytics dependencies. |
| `provenance` | Thông tin nguồn gốc của item/snapshot, gồm source, URL hoặc filename, thời điểm và version liên quan. |
| `migration report` | Kết quả theo từng record hoặc loại dữ liệu: imported, skipped, conflicted hoặc failed. |
