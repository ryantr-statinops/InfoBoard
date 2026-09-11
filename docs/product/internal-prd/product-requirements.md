# Internal PRD — Product requirements

## Capture và persistence

- `PR-CAP-01`: Tạo item từ text, URL công khai hoặc file được hỗ trợ.
- `PR-CAP-02`: Lưu source, title, snapshot hiện tại, content version và timestamps.
- `PR-CAP-03`: Input trùng không tạo item ngoài ý muốn.
- `PR-CAP-04`: Chỉnh sửa text tạo content version mới; URL/file giữ snapshot.
- `PR-CAP-05`: Dữ liệu chính còn nguyên sau restart hoặc worker interruption.

## Organization

- `PR-ORG-01`: Một item có thể thuộc nhiều collection.
- `PR-ORG-02`: Tạo, đổi tên hoặc xóa collection không làm mất item.
- `PR-ORG-03`: Người dùng thêm, sửa, xóa note và đổi trạng thái item.
- `PR-ORG-04`: Soft-deleted item không xuất hiện trong list, public detail hoặc search.

## Retrieval

- `PR-RET-01`: Lọc theo collection, source, status và khoảng thời gian.
- `PR-RET-02`: Keyword search hoạt động độc lập với semantic dependencies.
- `PR-RET-03`: Full mode có thể bổ sung semantic/hybrid search và related items.
- `PR-RET-04`: Kết quả theo item có title, source và excerpt đủ để nhận biết.
- `PR-RET-05`: Search không trả content version hoặc item đã bị xóa.

## Feedback và recovery

- `PR-REC-01`: UI phân biệt queued, processing, indexed và failed.
- `PR-REC-02`: Người dùng retry failed job khi lỗi có thể phục hồi.
- `PR-REC-03`: Derived-store failure không ngăn đọc dữ liệu chính hoặc keyword search.
- `PR-REC-04`: Health/degraded state chỉ rõ capability bị ảnh hưởng.
- `PR-REC-05`: Backup/restore bảo toàn item, snapshot, collection và note; index có thể rebuild.

## Experience

- `PR-UX-01`: Core flow dùng được mà không cần advanced setup.
- `PR-UX-02`: Filter context được giữ khi mở và đóng detail.
- `PR-UX-03`: Empty, loading, processing và error state có hướng dẫn tiếp theo.
- `PR-UX-04`: Desktop 1440 px và mobile viewport 390 px không tràn ngang.

## Security và privacy

- `PR-SEC-01`: Input được giới hạn, normalize và escape.
- `PR-SEC-02`: URL ingestion chặn private-network access và redirect không an toàn.
- `PR-SEC-03`: Log không chứa raw content, credential hoặc secret.
- `PR-SEC-04`: Cloud provider chỉ nhận nội dung sau cấu hình chủ động của người dùng.
