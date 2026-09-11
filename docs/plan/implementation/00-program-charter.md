# 00 — Program charter

**Status:** `ready`  
**Milestone:** toàn chương trình  
**Owner:** project maintainer  
**Dependencies:** `docs/plan/00-product-vision.md`, `01-data-architecture.md`

## Mục tiêu

InfoBoard là information dashboard local-first cho một người dùng: thu thập text, URL công khai và file; tổ chức bằng collections; đọc, ghi note và tìm lại bằng keyword hoặc semantic search. SQLite là system of record. Các index/cache/analytics khác đều có thể rebuild từ dữ liệu chính.

## Kết quả cần đạt

- Người dùng có thể nhập, lọc, mở, sửa, ghi note, gán collection và xóa item.
- Dashboard có trải nghiệm responsive, trạng thái index rõ ràng và không làm mất dữ liệu khi restart.
- Keyword search hoạt động độc lập; semantic/hybrid search là capability nâng cao có fallback an toàn.
- Backup, restore, migration, test và runbook đủ để chạy local dài hạn.
- Hậu MVP có hướng phát triển rõ nhưng không làm phình phạm vi MVP.

## Phạm vi theo giai đoạn

### MVP local-first

Python 3.12, FastAPI/Uvicorn, Jinja2 + HTMX, SQLite/FTS5, ingestion text/Markdown/TXT/PDF/public URL, worker tuần tự, optional ChromaDB/sentence-transformers/RocksDB/DuckDB, backup và quality gate.

### Hậu MVP

Browser extension, multi-device/cloud sync, identity/collaboration và AI/provider ecosystem. Các phần này chỉ triển khai sau discovery gate trong epic `30–33`.

## Không thuộc MVP

Không có authentication, cloud sync, chat, OCR, video transcript, recommendation feed bắt buộc, multi-tenant hay mobile native app.

## Nguyên tắc bất biến

1. SQLite là nguồn sự thật duy nhất cho item, content, quan hệ, note và job state có ảnh hưởng tới recovery.
2. Derived store hỏng không được làm mất khả năng đọc hoặc keyword search.
3. Không tải model hoặc gọi mạng ngầm trong một truy vấn dashboard.
4. Mọi input phải có giới hạn, được normalize và escape khi render.
5. Mỗi feature đi qua test, commit nhỏ, push và user review trước feature kế tiếp.

## Definition of done cấp chương trình

Tất cả tiêu chí trong `92-requirement-traceability.md` đạt; test/lint/benchmark pass; README và runbook khớp cách chạy thực tế; release có backup/restore verification; branch `main` sạch và tag được phiên bản MVP.

## Rủi ro chấp nhận

Model local và các native wheels có thể không tương thích mọi Linux. Core mode phải luôn chạy với SQLite/FTS5; full mode báo degraded rõ ràng nếu dependency dẫn xuất không cài được.
