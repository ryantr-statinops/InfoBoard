# Internal PRD — Domain model

Tài liệu này mô tả semantics cấp sản phẩm; schema chi tiết nằm trong tài liệu kiến trúc.

## Terminology

| Thuật ngữ | Định nghĩa |
| --- | --- |
| Item | Đơn vị thông tin chính gồm content, source, collections, notes, status và timestamps. |
| Source | Nơi hoặc hình thức tạo item: text, URL, PDF, Markdown hoặc TXT. |
| Snapshot | Bản nội dung được lưu tại một thời điểm để đọc, tìm kiếm và phục hồi độc lập với nguồn. |
| Portable content | Nội dung/metadata có thể import-export tương đối độc lập với browser. |
| Browser-specific state | Cookies, sessions, localStorage, IndexedDB và dữ liệu phụ thuộc browser profile. |
| Simple mode | Trải nghiệm mặc định ít cấu hình và dùng default an toàn. |
| Advanced mode | Lựa chọn provider, maintenance, diagnostics và technical metadata. |
| Provenance | Nguồn gốc của item/snapshot, gồm source, URL/filename, thời điểm và version. |

## Core entities

| Entity | Vai trò | Quan hệ chính |
| --- | --- | --- |
| Item | Đơn vị thông tin người dùng quản lý | Có source, snapshot/version, collections, notes và jobs |
| Snapshot | Nội dung đã lưu tại một version | Thuộc item; là nguồn để đọc/rebuild |
| Chunk | Phần nội dung dùng cho retrieval | Thuộc item và content version hiện hành |
| Collection | Nhóm theo project hoặc lĩnh vực | Quan hệ nhiều-nhiều với item |
| Note | Ghi chú cá nhân | Thuộc item và không bị ghi đè khi reindex |
| Index job | Trạng thái xử lý/retry | Gắn với item và content version |
| Source | Provenance của dữ liệu | Text, URL, PDF, Markdown hoặc TXT trong MVP |

## Item lifecycle

```text
Create → Inbox → Active → Archived
                  ↘ Soft deleted → Derived cleanup
```

Organization status độc lập với index-job state. Item chỉ hợp lệ khi dữ liệu chính tồn tại; vector hoặc cache không tự tạo ra item.

## Invariants

- Một item có nhiều collection; xóa collection không xóa item.
- Notes tồn tại độc lập với reindex và derived-store rebuild.
- Soft-deleted item bị loại khỏi mọi user-facing read path.
- Chunk/vector cũ không được trả sau khi content version mới có hiệu lực.
- Derived stores có thể rebuild từ dữ liệu chính.
- Job retry và cleanup phải idempotent.
