# InfoBoard — Internal Product Requirements Document

**Status:** `active`
**Audience:** product owner, designer và engineering team
**Implementation reference:** [`docs/plan/implementation/`](../../plan/implementation/)

## 1. Product goal

InfoBoard là knowledge hub local-first giúp một người dùng thu thập, lưu giữ, tổ chức và tìm lại thông tin cá nhân từ text, URL và file. Sản phẩm ưu tiên trải nghiệm đơn giản, dữ liệu bền vững và khả năng hoạt động khi các thành phần nâng cao không sẵn sàng.

Outcome của MVP là người dùng có thể hoàn thành vòng lặp:

```text
Capture → Understand source → Organize → Retrieve
```

Export, browser import và full browser-data portability là hướng phát triển tiếp theo, không phải acceptance của MVP hiện tại.

## 2. Users

### Primary user

Người dùng phổ thông muốn lưu bài viết, ghi chú hoặc file và tìm lại sau này mà không cần hiểu database, indexing hay embedding. Flow mặc định phải ít bước, dùng ngôn ngữ rõ ràng và có trạng thái lỗi phục hồi được.

### Advanced user path

Power user và privacy-conscious user cần kiểm soát local storage, provider, backup/rebuild và chẩn đoán. Advanced options được tách khỏi flow mặc định và không làm simple mode phức tạp hơn.

## 3. Problems

- Thông tin cá nhân phân tán giữa web, browser, file và ghi chú.
- Link có thể mất hoặc nội dung nguồn thay đổi; người dùng cần snapshot để đọc lại offline.
- Người dùng thường nhớ chủ đề hoặc ý nghĩa thay vì vị trí chính xác của thông tin.
- Hệ thống lưu trữ hoặc index phụ thuộc dịch vụ có thể khiến dữ liệu khó phục hồi.
- Chuyển browser hoặc thiết bị có thể làm mất context; vấn đề này được nghiên cứu trong [Next Plan](../next-plan/README.md).

## 4. MVP scope

MVP dành cho một người dùng trên máy cá nhân, giao diện tiếng Việt và hỗ trợ tìm kiếm nội dung tiếng Việt/Anh.

Trong phạm vi:

- Nhập text trực tiếp, bài web công khai, PDF có text, Markdown và TXT.
- Lưu metadata nguồn và snapshot text để xem offline.
- Tổ chức item bằng nhiều collection, note và trạng thái `inbox`, `active`, `archived`.
- List, filter, mở detail, sửa metadata/note/status và soft-delete item.
- Keyword search luôn hoạt động; semantic/hybrid search là capability nâng cao có fallback.
- Hiển thị trạng thái xử lý, lỗi và degraded mode rõ ràng.
- Backup, restore và rebuild derived indexes từ dữ liệu chính.
- Dashboard responsive trên desktop và mobile viewport, vẫn chỉ truy cập local.

## 5. Non-goals của MVP

- Import bookmark hoặc browser extension.
- Di chuyển cookies, login session, password, localStorage hoặc IndexedDB.
- Cloud sync, tài khoản, collaboration hoặc multi-tenant.
- Chat với dữ liệu, AI summary, recommendation feed, OCR hoặc video transcript.
- Mobile native app hoặc truy cập từ thiết bị khác.
- Tự tải model hoặc gửi nội dung lên cloud khi người dùng chưa cấu hình rõ ràng.

## 6. Functional requirements

### Capture và persistence

- Người dùng tạo item từ text, URL công khai hoặc file được hỗ trợ.
- Hệ thống lưu source, title, snapshot hiện tại, content version và timestamps.
- Input trùng không tạo item ngoài ý muốn; thay đổi text tạo content version mới.
- Dữ liệu chính còn nguyên sau restart.

### Organization

- Một item thuộc nhiều collection.
- Người dùng tạo/đổi tên/xóa collection mà không làm mất item.
- Người dùng thêm, sửa, xóa note và đổi trạng thái item.
- Item bị soft-delete không xuất hiện trong list, detail công khai hoặc search.

### Retrieval

- Người dùng lọc theo collection, source, status và khoảng thời gian.
- Keyword search hoạt động độc lập với semantic dependencies.
- Khi full mode khả dụng, semantic/hybrid search và related items bổ sung discovery.
- Kết quả trả về theo item, có title, source và excerpt đủ để nhận biết.

### Feedback và recovery

- UI phân biệt queued/processing/indexed/failed và cung cấp retry khi phù hợp.
- Lỗi derived store không ngăn đọc dữ liệu chính hoặc keyword search.
- Health/degraded state chỉ rõ capability bị ảnh hưởng.
- Backup/restore bảo toàn item, snapshot, collection và note; indexes có thể rebuild.

## 7. Core workflows

### Add and organize

1. Người dùng chọn Text, URL hoặc File.
2. Hệ thống validate input, tạo item và hiển thị trạng thái xử lý.
3. Người dùng gán collection hoặc giữ item trong inbox.
4. Item xuất hiện trong dashboard và tồn tại sau restart.

### Review and update

1. Người dùng lọc hoặc chọn item gần đây.
2. Detail panel hiển thị content, source, collections, note và trạng thái.
3. Người dùng cập nhật title, note, collection hoặc status.
4. Đóng panel trả về đúng filter context trước đó.

### Search and recover

1. Người dùng nhập query và tùy chọn filter.
2. Core mode trả keyword results; full mode có thể kết hợp semantic results.
3. Người dùng mở item từ excerpt hoặc related result.
4. Nếu semantic/analytics lỗi, UI báo degraded và giữ core flow hoạt động.

## 8. Product constraints

- Local-first; MVP bind local host và không có authentication.
- SQLite là nguồn sự thật cho item, content, relationships, notes và recovery-relevant state.
- Derived stores không được là nơi duy nhất chứa dữ liệu người dùng.
- Input phải có giới hạn, normalize và escape khi render.
- Không có network call hoặc model download ngầm trong dashboard request.
- Simple mode là mặc định; cấu hình provider, rebuild và diagnostics thuộc advanced path.

## 9. Privacy and trust boundary

- Chỉ gửi nội dung tới cloud provider khi người dùng chủ động cấu hình.
- Không log raw content, credential hoặc secret.
- Xóa item phải ẩn ngay và có cleanup/rebuild behavior rõ ràng.
- URL ingestion chỉ truy cập nguồn public sau validation và phải chặn private-network access.
- Dữ liệu browser nhạy cảm nằm ngoài MVP và tuân theo discovery gate riêng trong Next Plan.

## 10. Success criteria

- Người dùng hoàn thành add → organize → open → note → search mà không cần advanced setup.
- Năm nguồn MVP tạo snapshot/chunk đúng và dữ liệu tồn tại sau restart.
- Keyword search hoạt động khi semantic, cache hoặc analytics dependency unavailable.
- Search evaluation Việt/Anh có item đích trong top 5 cho ít nhất 16/20 query chuẩn.
- Backup/restore khôi phục dữ liệu chính và rebuild index thành công.
- Desktop 1440 px và mobile 390 px không tràn ngang trong core flow.
- Trên tập 1.000 item/10.000 chunks, search p95 dưới một giây khi model đã tải, với cấu hình máy được ghi lại.

## 11. Product acceptance

Internal PRD được xem là đạt cho MVP khi:

- Mọi capability trong MVP scope có implementation evidence và test tương ứng.
- Không requirement hậu MVP nào trở thành release blocker của M1–M4.
- Data integrity, fallback, security limits và recovery gates đều pass.
- README, runbook và behavior thực tế thống nhất.
- Requirement traceability không còn trạng thái `missing` hoặc `partial` nếu không có waiver được ghi nhận.

## 12. Dependencies

- Product concepts dùng định nghĩa trong [glossary](glossary.md).
- Kiến trúc dữ liệu hiện hành nằm tại [`docs/plan/01-data-architecture.md`](../../plan/01-data-architecture.md).
- Dashboard/API experience hiện hành nằm tại [`docs/plan/02-mvp-experience.md`](../../plan/02-mvp-experience.md).
- Milestone, quality gate và execution evidence nằm trong [implementation plan](../../plan/implementation/README.md).

## 13. Open decisions

Các quyết định dưới đây không chặn M1, nhưng phải được khóa trước milestone liên quan:

- Model embedding local và revision mặc định sau dependency smoke test.
- Full-mode dependency matrix trên các Linux distribution được hỗ trợ.
- Export format đầu tiên cho portable content.
- Tiêu chí chuyển bookmark import từ Next Plan vào implementation roadmap.
- UX boundary chính xác giữa simple mode và advanced settings.
