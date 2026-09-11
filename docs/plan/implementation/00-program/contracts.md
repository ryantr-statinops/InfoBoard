# 04 — Cross-cutting contracts

**Status:** `ready`
**Applies to:** core and full modes

## API conventions

- Prefix: `/api`.
- JSON uses `Content-Type: application/json`; uploads use multipart.
- IDs are current SQLite integers; clients do not create IDs.
- Responses use UTC ISO-8601 timestamps; the database may store UTC text.
- Lists use `{items, total, limit, offset}` with default limit 20 and maximum 100.
- Default sort is `created_at DESC, id DESC`; search sorts by score then time.

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

HTTP mapping: `400` invalid input, `404` missing resource, `409` duplicate/conflict, `413` limit exceeded, `422` schema validation, and `503 unavailable` when SQLite or FTS5 cannot support core mode. Derived dependency failure returns degraded state without failing the core request.

## Item and job states

- Item organization: `inbox`, `active`, `archived`, `deleted` (soft delete internally).
- Technical job states: `queued`, `extracting`, `chunking`, `embedding`, `indexed`, `failed`.
- UI mapping: `queued → queued`, `extracting | chunking | embedding → processing`, `indexed → indexed`, `failed → failed`.
- `deleted_at` excludes an item from public list/search/detail.
- `content_version` increases when text changes; old chunks/vectors are not returned.

## Endpoint contract

| Endpoint | Contract |
| --- | --- |
| `POST /api/items` | Create text, deduplicate, and return `202` with `item_id`, `job_id`, and state. |
| `POST /api/items/upload` | Multipart file with optional title/collections and the same job contract. |
| `POST /api/items/url` | Validate/canonicalize a public URL and use the same job contract. |
| `GET /api/items` | Filter by collection/status/source/date/search with pagination wrapper. |
| `GET /api/items/{id}` | Return metadata, current content, collections, notes, chunks, and job metadata. |
| `PATCH /api/items/{id}` | Update title/status/collections or text for text sources; text changes create a version. |
| `DELETE /api/items/{id}` | Soft-delete immediately and enqueue cleanup; return `202`. |
| `POST /api/items/{id}/reindex` | Create/requeue a job for the current content version. |
| `POST /api/reindex` | Reindex items/indexes requiring maintenance. |
| Collection/note endpoints | CRUD and idempotent attach/detach without deleting an item when a collection is removed. |
| `POST /api/search` | Query plus shared filters; return excerpt, score, retrieval mode, and source. |
| `GET /api/items/{id}/related` | Return up to five related items, excluding the current item. |
| `GET /api/analytics` | Return KPI/activity/distribution/cluster data using shared filters. |
| `GET /api/health` | Component status; `503` for SQLite/FTS5 unavailability and `200 degraded` for derived-only failure. |

## Provider interfaces

```text
EmbeddingProvider.embed(texts) -> vectors
EmbeddingProvider.model_id / revision / dimension / max_tokens
EmbeddingProvider.tokenize(text) -> tokens
Extractor.extract(input) -> ExtractedDocument(title, text, source_type, source_url, original_filename, metadata)
VectorIndex.upsert(chunks, vectors, metadata)
VectorIndex.query(vector, filters, limit)
```

## Modes and HTMX

Core mode requires SQLite/FTS5 and may use no semantic provider. Full mode adds optional ChromaDB, sentence-transformers, RocksDB, and DuckDB. Missing derived dependencies produce degraded state and preserve keyword fallback.

Fragment routes stay outside `/api`, return HTML partials, and may use `HX-Trigger` for toast/list refresh. Errors return renderable fragments while preserving unsuccessful form input.
