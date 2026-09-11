# 01 — Current state và gap analysis

**Status:** `ready`  
**Snapshot:** `main` tại merge commit `c8ac771`  
**Branch triển khai:** `dev`

## Hiện trạng đã có

| Khu vực | Mức độ | Bằng chứng |
| --- | --- | --- |
| FastAPI + Uvicorn + lifespan | `partial` | `app/main.py`, `/api/health`, template render. |
| SQLite schema + FTS5 | `partial` | `app/db.py`, tạo bảng và rebuild FTS. Chưa có migration history. |
| Text item, hash dedup, chunk | `done-basic` | `app/services.py`, chunk 200 words/overlap 30. Chưa theo tokenizer provider. |
| Collections và notes API | `partial` | CRUD cơ bản có; UI và contract wrapper chưa hoàn chỉnh. |
| Soft delete/reindex | `partial` | Endpoint có; cleanup derived store chưa có. |
| TXT/Markdown/PDF/URL | `partial` | Có extractor tối thiểu; PDF/URL limit, redirect và parser cần harden. |
| Worker | `partial` | Lifecycle đồng bộ mô phỏng; chưa có background polling/embedding thật. |
| Keyword search | `done-basic` | FTS5 + excerpt; filter/highlight/error contract cần chuẩn hóa. |
| Semantic search | `missing` | Provider interface có, ChromaDB chưa tích hợp. |
| RRF | `done-basic` | Utility và unit test có; chưa nối vào search pipeline. |
| Analytics | `partial` | SQLite fallback và DuckDB adapter tối thiểu. |
| Dashboard | `partial` | Server-rendered form/list; thiếu HTMX, panel, filter nâng cao và KPI. |
| Quality | `partial` | 4 test và Ruff pass; thiếu HTTP integration, benchmark và CI. |

## Khoảng cách ưu tiên

1. Chuẩn hóa foundation/data contract trước khi mở rộng UI.
2. Hoàn thiện dashboard và item workspace để có luồng người dùng end-to-end.
3. Thay extractor/worker tối thiểu bằng pipeline có version, retry và recovery.
4. Nối semantic persistent index nhưng giữ FTS5 fallback.
5. Hoàn thiện analytics, backup, security, observability và release gate.

## Quy tắc chuyển đổi không mất dữ liệu

- Mỗi thay đổi schema bắt buộc migration và backup trước khi chạy.
- Không đổi ý nghĩa `items.id`, `content_hash`, `deleted_at` hoặc note hiện có.
- Index mới đọc từ SQLite; không migrate dữ liệu bằng cách chỉ dựa trên Chroma/RocksDB.
- API response mới có compatibility adapter/redirect trong một milestone nếu UI cũ còn dùng.

## Cách cập nhật file này

Sau mỗi milestone, cập nhật bảng hiện trạng bằng commit hash, test command và evidence link. Không dùng `done` cho capability mới chỉ có adapter hoặc mock.
