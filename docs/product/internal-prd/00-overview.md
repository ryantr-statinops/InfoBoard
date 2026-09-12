# Internal PRD — Overview

## Product goal

InfoBoard là một **Advanced Bookmark Manager** chạy trên máy: lưu URL và snapshot nội dung, thêm notes và collections, rồi tìm kiếm bằng keyword trong core mode. Full mode có thể bổ sung semantic/hybrid retrieval; sản phẩm có thể mở rộng sang browser portability sau MVP.

Định vị này giúp người dùng hiểu sản phẩm từ hành vi bookmark quen thuộc. Đặc tính local-first vẫn được giữ như một nguyên tắc kỹ thuật và quyền sở hữu dữ liệu, không dùng làm tên gọi chính của sản phẩm.

```text
Capture → Understand source → Organize → Retrieve
```

## Problems

- Thông tin cá nhân phân tán giữa web, browser, file và ghi chú.
- Link có thể mất hoặc nội dung nguồn thay đổi; người dùng cần snapshot để đọc offline.
- Người dùng thường nhớ chủ đề hoặc ý nghĩa thay vì vị trí chính xác của thông tin.
- Index hoặc dịch vụ phụ trợ bị lỗi không được làm dữ liệu chính khó phục hồi.
- Chuyển browser hoặc thiết bị có thể làm mất context; đây là hướng nghiên cứu trong [Next Plan](../next-plan/README.md).

## MVP scope

- Một người dùng trên máy cá nhân; giao diện tiếng Việt và tìm kiếm nội dung Việt/Anh.
- Nhập text, bài web công khai, PDF có text, Markdown và TXT.
- Lưu metadata nguồn và snapshot text để đọc offline.
- Tổ chức bằng collections, notes và trạng thái `inbox`, `active`, `archived`.
- List, filter, detail, edit, soft-delete và search.
- Keyword search độc lập; semantic/hybrid search là capability optional của full mode và có fallback an toàn.
- Trạng thái xử lý, degraded mode, backup, restore và rebuild rõ ràng.
- Dashboard responsive trên desktop và mobile viewport, nhưng chỉ truy cập local.

## Non-goals của MVP

- Bookmark import, browser extension hoặc migration assistant.
- Cookies, login sessions, passwords, localStorage hoặc IndexedDB portability.
- Cloud sync, accounts, collaboration hoặc multi-tenant.
- Chat, AI summary, recommendation feed, OCR hoặc video transcript.
- Mobile native app hoặc truy cập từ thiết bị khác.
- Model download hoặc cloud upload khi người dùng chưa chủ động cấu hình.

## Product constraints

- Local-first; MVP bind local host và không có authentication.
- SQLite là nguồn sự thật; derived stores phải có fallback hoặc rebuild path.
- Input có giới hạn, được normalize và escape khi render.
- Dashboard request không tự tải model hoặc tạo network call ngoài hành động người dùng yêu cầu.
- Simple mode là mặc định; provider, rebuild và diagnostics thuộc advanced path.

## Dependencies

- Thuật ngữ và entity semantics: [Domain model](04-domain-model.md).
- Kiến trúc: [Architecture](../../architecture/README.md).
- Dashboard experience: [Design](../../design/README.md); API/interface contracts canonical nằm trong [Architecture interface contracts](../../architecture/09-interface-contracts.md).
- Milestone và evidence: [Implementation plan](../../plan/implementation/README.md).
