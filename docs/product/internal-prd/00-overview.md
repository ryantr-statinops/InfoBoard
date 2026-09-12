# Internal PRD — Overview

## Product goal

InfoBoard là một **Advanced Bookmark Manager** chạy trên máy: lưu URL và snapshot nội dung, thêm notes và collections, rồi tìm lại thông tin bằng keyword search. Full mode có thể bổ sung semantic/hybrid retrieval khi người dùng chủ động cấu hình. Sản phẩm có thể mở rộng sang browser portability sau MVP.

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

## Core MVP scope

- Một người dùng trên máy cá nhân; giao diện tiếng Việt và tìm kiếm nội dung Việt/Anh.
- Nhập text, bài web công khai, PDF có text, Markdown và TXT.
- Lưu metadata nguồn và snapshot text để đọc offline.
- Tổ chức bằng collections, notes và trạng thái `inbox`, `active`, `archived`.
- List, filter, detail, edit, soft-delete và search.
- Keyword search hoạt động độc lập với full mode và các derived dependencies.
- Trạng thái xử lý, failure/retry, backup, restore và rebuild rõ ràng.
- Dashboard responsive trên desktop và mobile viewport, nhưng chỉ truy cập local.

## Optional full mode

Full mode không phải điều kiện để phát hành core MVP. Chỉ khi người dùng chủ động
cấu hình đầy đủ dependency và model, hệ thống mới bật các capability sau:

- Semantic hoặc hybrid retrieval bổ sung cho keyword search.
- Related content và topic clusters dựa trên derived indexes.
- Analytics acceleration bằng derived engine; SQLite fallback vẫn là đường chạy core.
- Khi full mode chưa được cấu hình, core UI không hiển thị degraded state vì thiếu
  full-mode dependency.
- Khi full mode đã được cấu hình nhưng dependency lỗi, UI hiển thị degraded state
  và giữ nguyên keyword/read flow.

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
- Dashboard/API experience: [Design](../../design/README.md); technical interface contracts sẽ được khóa trong Architecture.
- Milestone và execution evidence được xác định sau khi Product, Design, Architecture, Quality và Operations hoàn tất.
