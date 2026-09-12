# Architecture — Interface contracts

## Current state

The runtime exposes basic FastAPI endpoints, SQLite-backed services, and server-rendered pages. Response envelopes, pagination, durable job responses, content-version projection, component health, and HTMX fragment behavior do not yet consistently meet the target contract below.

The runtime snapshot remains the evidence for implemented behavior. This document is the canonical target interface contract and is not proof of delivery.

## Target contract

### API conventions

- API prefix: `/api`; fragment routes stay outside `/api` and return HTML.
- JSON responses use `Content-Type: application/json`; uploads use multipart.
- IDs are SQLite-generated integers; clients do not create IDs.
- Timestamps are UTC ISO-8601 values at the HTTP boundary.
- List responses use `{items, total, limit, offset}` with default limit 20 and maximum 100.
- Default list sort is `created_at DESC, id DESC`; search sorts by score and then time.

### Error envelope

```json
{
  "error": {
    "code": "duplicate_item",
    "message": "Item with the same content already exists",
    "details": {}
  }
}
```

HTTP mapping: `400` invalid input, `404` missing resource, `409` duplicate/conflict, `413` limit exceeded, `422` schema validation, and `503 unavailable` when SQLite or FTS5 cannot support core mode. Failure of an enabled optional component returns a successful core response with explicit degraded metadata when a safe fallback exists.

### Item, job, and UI states

- Item organization states: `inbox`, `active`, `archived`; `deleted_at` represents internal soft deletion.
- Durable job states: `queued`, `extracting`, `chunking`, `embedding`, `indexed`, `failed`.
- UI projection: `queued → queued`, `extracting | chunking | embedding → processing`, `indexed → indexed`, `failed → failed`.
- `content_version` increases when editable text changes; deleted items and stale versions are excluded from public reads and retrieval.

### Endpoint boundaries

| Endpoint | Target behavior |
| --- | --- |
| `GET /api/health` | Report core and optional components as `ok`, `disabled`, `degraded`, or `unavailable`; return `503` for SQLite/FTS5 failure and `200 degraded` for failure of an enabled optional component. |
| `POST /api/items` | Create text, deduplicate, and return `202` with `item_id`, `job_id`, and state. |
| `POST /api/items/upload` | Accept a bounded supported file with optional title/collections and return the same job contract. |
| `POST /api/items/url` | Validate and canonicalize a public URL before using the same job contract. |
| `GET /api/items` | Filter by collection, status, source, date, and query with the pagination wrapper. |
| `GET /api/items/{id}` | Return metadata, current content, collections, notes, chunks, and job projection. |
| `PATCH /api/items/{id}` | Update title/status/collections or editable text; text changes create a content version. |
| `DELETE /api/items/{id}` | Soft-delete immediately, queue derived cleanup, and return `202`. |
| `POST /api/items/{id}/reindex` | Create or requeue a job for the current content version. |
| `POST /api/reindex` | Rebuild or requeue items/indexes selected by the maintenance contract. |
| Collection/note endpoints | Provide CRUD plus idempotent item attach/detach; deleting a collection never deletes its items. |
| `POST /api/search` | Return item-level keyword results in core mode and optional semantic/hybrid metadata only when full mode is enabled. |
| `GET /api/items/{id}/related` | Return up to five valid related items when full mode is enabled, excluding the current item and stale/deleted data. |
| `GET /api/analytics` | Return KPI/activity/distribution/cluster data with shared filters and the SQLite fallback when optional DuckDB acceleration is unavailable. |

### Service interfaces

```text
Extractor.extract(input, limits) -> ExtractedDocument
ExtractedDocument = title + text + source_type + source_url + original_filename + metadata
VectorIndex.upsert(chunks, vectors, metadata)
VectorIndex.query(vector, filters, limit)
EmbeddingProvider.embed(texts, model_revision) -> vectors
EmbeddingProvider.model_id + revision + dimension + max_tokens
EmbeddingProvider.tokenize(text) -> tokens
Analytics.query(filters) -> KPI result + mode metadata
```

Every derived result contains stable item/content-version provenance and is hydrated from SQLite before it reaches a public response.

### Core, full, and HTMX behavior

- Core mode requires SQLite/FTS5 and extraction dependencies; keyword retrieval and SQLite analytics fallback remain available without semantic dependencies.
- Full mode is explicitly configured and may add sentence-transformers, ChromaDB, RocksDB, and DuckDB.
- Full mode not configured is a normal core-mode state, not degraded.
- An enabled full-mode component that becomes unavailable reports degraded status while preserving the safe core response.
- HTMX fragment errors preserve submitted input and return renderable, escaped HTML; successful mutations may use `HX-Trigger` for list refresh and notifications.

## Implementation gap

- Current create/list/search/analytics responses do not consistently use the target envelopes and shared filters.
- Health is static and does not inspect SQLite, FTS5, or enabled optional components.
- Content-version, durable job, soft-delete cleanup, and HTMX fragment contracts are incomplete.
- Optional-component configuration and degraded metadata are not implemented consistently.

## Owning work

- Epics 10–12 own application, data, and dashboard boundaries.
- Epics 13–16 own extraction, job, retrieval, and analytics interfaces.
- Epics 18–19 own cleanup, recovery, health, and diagnostics behavior.
- Epics 20–21 own contract verification and release documentation.

## Evidence required

- HTTP integration tests for success, pagination, validation, duplicate, deletion, and unavailable/degraded responses.
- Migration and worker tests proving content-version and durable job projection.
- Core-only and configured-full-mode health/fallback scenarios.
- HTMX browser/fragment tests for escaped errors, preserved input, and refresh events.
