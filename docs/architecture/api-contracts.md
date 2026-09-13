# API contracts

All JSON endpoints use `/api/v1`, UTF-8, RFC 3339 UTC timestamps, UUID identifiers, and `application/json`. Unknown fields are rejected for mutation requests. HTML/HTMX routes may wrap the same application services but cannot define different behavior.

## Common envelopes

Successful collection responses use:

```json
{
  "data": [],
  "page": {"limit": 50, "next_cursor": null},
  "meta": {"correlation_id": "uuid"}
}
```

Errors use:

```json
{
  "error": {
    "code": "SNAPSHOT_FETCH_TIMEOUT",
    "message": "The bookmark was saved, but its snapshot could not be captured.",
    "retryable": true,
    "field_errors": {}
  },
  "meta": {"correlation_id": "uuid"}
}
```

Error codes are stable. Messages are safe and may evolve. Responses never expose local paths, raw captured content, API keys, upstream bodies, stack traces, or SQL/provider details.

## Shared types

```text
BookmarkStatus = inbox | active | archived
SnapshotState = pending | processing | ready | failed
JobState = queued | processing | indexed | failed
SearchMode = keyword | semantic | hybrid
ComponentState = unconfigured | ready | degraded | unavailable
ConsentState = not_granted | granted | revoked
```

`BookmarkSummary` contains `id`, `canonical_url`, `title`, `description`, `source_domain`, `status`, current snapshot summary, collections, tags, and timestamps. `BookmarkDetail` adds notes, current snapshot provenance/content projection, and recent job summaries. Neither exposes managed-file references.

## Bookmark and snapshot endpoints

| Method and path | Request/behavior | Response |
| --- | --- | --- |
| `POST /api/v1/bookmarks` | `{url, collection_ids?, tags?}`; validate, normalize, create/find, and queue capture | `202` with bookmark, `created`, and capture job; duplicate may return `200` |
| `GET /api/v1/bookmarks` | Cursor pagination plus shared filters and sort | `200` bookmark summaries |
| `GET /api/v1/bookmarks/{id}` | Current non-deleted detail | `200` detail or `404` |
| `PATCH /api/v1/bookmarks/{id}` | Partial `{title?, description?, status?, collection_ids?, tags?}` | `200` updated detail |
| `DELETE /api/v1/bookmarks/{id}` | Soft delete; optional `If-Match` version | `202` with cleanup job |
| `POST /api/v1/bookmarks/{id}/snapshots` | Explicit recapture; no active duplicate job | `202` snapshot/job state |
| `POST /api/v1/bookmarks/{id}/snapshots/retry` | Retry latest retryable failed attempt | `202` same logical operation requeued |
| `GET /api/v1/bookmarks/{id}/snapshots/current` | Current ready snapshot projection or latest failed/pending state | `200` state/provenance/content projection |

Mutation conflicts return `409`; validation/unsafe URLs return `422`; size limits return `413`; rate/busy limits return `429` with safe retry guidance.

## Organization endpoints

- `GET|POST /api/v1/collections`, `PATCH|DELETE /api/v1/collections/{id}`.
- `PUT|DELETE /api/v1/bookmarks/{id}/collections/{collection_id}` for idempotent membership.
- `GET|POST /api/v1/tags`, `PATCH|DELETE /api/v1/tags/{id}`.
- `PUT|DELETE /api/v1/bookmarks/{id}/tags/{tag_id}` for idempotent membership.
- `GET|POST /api/v1/bookmarks/{id}/notes`, `PATCH|DELETE /api/v1/bookmarks/{id}/notes/{note_id}`.

Collection/tag deletion returns `204` after removing memberships and never deletes bookmarks. Names are normalized server-side; normalized-name conflict returns `409` and identifies the existing resource safely.

## Browse, search, and analytics filters

Shared query/body filters are:

```json
{
  "collection_ids": [],
  "tag_ids": [],
  "statuses": ["inbox", "active"],
  "domains": [],
  "saved_from": null,
  "saved_to": null,
  "captured_from": null,
  "captured_to": null
}
```

Arrays use OR within one dimension and AND across dimensions. Date ranges are UTC half-open intervals. Deleted bookmarks are excluded unconditionally.

`POST /api/v1/search` accepts `{query, mode, filters, limit, cursor?}`. It returns bookmark results plus `requested_mode`, `actual_mode`, `components`, and safe `degradations`. Empty query is allowed only for keyword browse/filter semantics; semantic/hybrid require non-empty bounded text.

`POST /api/v1/analytics` accepts `{filters, interval?}` and returns accepted KPI/time-series values plus `actual_backend`, source watermark, and degradation metadata.

## Semantic-provider endpoints

| Method and path | Contract |
| --- | --- |
| `GET /api/v1/settings/semantic` | Return endpoint, model, timeout, consent state, key presence, active revision, and health; never key value. |
| `PUT /api/v1/settings/semantic` | Validate and store endpoint/model/timeout and optional write-only `api_key`; does not grant consent. |
| `POST /api/v1/settings/semantic/verify` | Perform bounded compatibility/model check; does not persist consent or index content. |
| `POST /api/v1/settings/semantic/consent` | Require `{granted: true, disclosure_version}`; record explicit consent and allow indexing. |
| `DELETE /api/v1/settings/semantic/consent` | Revoke consent, stop external calls, pause jobs, and queue semantic derived cleanup. |

Configuration changes that alter compatibility create a pending index revision. The previous active revision remains queryable until the new revision is complete, unless consent is revoked or it becomes unsafe/incompatible.

## Health and maintenance

- `GET /api/v1/health` returns overall status and component status for SQLite, FTS5, capture worker, semantic provider, RocksDB, ChromaDB, DuckDB, and analytics projection.
- `POST /api/v1/maintenance/rebuild` requires an explicit target from `fts | rocksdb | chroma | duckdb | all-derived`, returns `202`, and never targets SQLite canonical tables.
- `GET /api/v1/maintenance/jobs/{id}` returns safe progress/checkpoint/error metadata.

Overall HTTP health is `503` when SQLite or mandatory local keyword retrieval is unavailable. A configured semantic/derived failure returns `200` with overall `degraded` when canonical and keyword workflows are safe. An unconfigured semantic provider is `unconfigured`, not `degraded`, although MVP release acceptance still requires configured semantic evidence.
