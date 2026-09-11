# 04 — Cross-cutting contracts

**Status:** `ready`
**Applies to:** core mode và full mode

## API conventions

- Prefix: `/api`.
- JSON dùng `Content-Type: application/json`; upload dùng multipart.
- ID là integer SQLite hiện tại; client không tự tạo ID.
- Timestamp là UTC ISO-8601 trong response; database có thể lưu UTC text.
- List có `{items, total, limit, offset}`; `limit` mặc định 20, tối đa 100.
- Sort mặc định `created_at DESC, id DESC`; search sort theo score rồi thời gian.

## Error envelope

```json
{
  "error": {
    "code": "duplicate_item",
    "message": "Item with the same content already exists",
    "details": {}
  }
}
```

HTTP mapping: `400` input invalid, `404` resource missing, `409` duplicate/conflict, `413` vượt giới hạn, `422` schema validation, `503` SQLite unavailable hoặc dependency bắt buộc degraded.

## Item và job states

- Item organization: `inbox`, `active`, `archived`, `deleted` (soft delete nội bộ).
- Index job: `queued`, `extracting`, `chunking`, `embedding`, `indexed`, `failed`.
- `deleted_at` luôn loại item khỏi list/search/detail public.
- `content_version` tăng khi text item được sửa; chunks/vectors cũ không được trả về.

## Endpoint contract

| Endpoint | Contract chính |
| --- | --- |
| `POST /api/items` | Tạo text, dedup; trả `202` với `item_id`, `job_id`, `state`. |
| `POST /api/items/upload` | Multipart file, title/collections tùy chọn; cùng job contract. |
| `POST /api/items/url` | URL public, canonicalize/validate; cùng job contract. |
| `GET /api/items` | Filter collection/status/source/date/search, pagination wrapper. |
| `GET /api/items/{id}` | Metadata, current content, collections, notes, chunks/job metadata. |
| `PATCH /api/items/{id}` | title/status/collections hoặc text nếu source là `text`; tạo version mới khi đổi text. |
| `DELETE /api/items/{id}` | Soft delete ngay, enqueue cleanup; trả `202`. |
| `POST /api/items/{id}/reindex` | Tạo/requeue job cho content version hiện tại. |
| `POST /api/reindex` | Reindex các item/index cần rebuild; chỉ local maintenance. |
| `GET/POST/PATCH/DELETE /api/collections` | CRUD collection, không xóa item khi xóa collection. |
| `POST/DELETE /api/items/{id}/collections/{collection_id}` | Attach/detach quan hệ idempotent. |
| `GET/POST/PATCH/DELETE /api/items/{id}/notes` | Note CRUD, kiểm tra item tồn tại. |
| `POST /api/search` | Query + shared filters; trả excerpt, score, retrieval mode/source. |
| `GET /api/items/{id}/related` | Tối đa 5 item, loại chính item hiện tại. |
| `GET /api/analytics` | KPI, activity, distributions, clusters theo shared filters. |
| `GET /api/health` | Component status; `503` chỉ khi SQLite không dùng được. |

## Provider interfaces

```text
EmbeddingProvider.embed(texts) -> vectors
EmbeddingProvider.model_id
EmbeddingProvider.revision
EmbeddingProvider.dimension
EmbeddingProvider.max_tokens
EmbeddingProvider.tokenize(text) -> tokens

Extractor.extract(input) -> ExtractedDocument(title, text, source_type, source_url, original_filename, metadata)
VectorIndex.upsert(chunks, vectors, metadata)
VectorIndex.query(vector, filters, limit)
```

## Core/full mode

Core mode bắt buộc chạy với SQLite/FTS5 và provider giả hoặc không semantic. Full mode bật ChromaDB, sentence-transformers, RocksDB và DuckDB qua optional extra; thiếu một derived dependency phải trả degraded status và giữ keyword fallback.

## HTMX contract

Các route fragment không nằm dưới `/api`, trả HTML partial và dùng `HX-Trigger` cho toast/list refresh. Request lỗi phải trả fragment lỗi có thể render, đồng thời giữ form input chưa gửi thành công.
