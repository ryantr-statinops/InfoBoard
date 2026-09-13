# API schemas

Fields are required unless marked optional/nullable. Examples show the exact field set; mutation requests reject additional fields.

## Envelopes

Single-resource success:

```json
{"data": {}, "meta": {"correlation_id": "01H..."}}
```

Collection success:

```json
{
  "data": [],
  "page": {"limit": 50, "next_cursor": null},
  "meta": {"correlation_id": "01H..."}
}
```

Error:

```json
{
  "error": {
    "code": "RESOURCE_VERSION_CONFLICT",
    "message": "The bookmark changed after it was loaded.",
    "retryable": false,
    "field_errors": {}
  },
  "meta": {"correlation_id": "01H..."}
}
```

`correlation_id` is an opaque diagnostic string, not an entity identifier.

## Shared values and limits

```text
BookmarkStatus = inbox | active | archived
SourceKind = url | legacy_text | legacy_file
CaptureAttemptState = queued | processing | succeeded | failed
JobState = queued | processing | succeeded | failed
SearchMode = keyword | semantic | hybrid
ComponentState = unconfigured | ready | degraded | unavailable
ConsentState = not_granted | granted | revoked
Sort = saved_desc | updated_desc | title_asc
```

| Field | Limit |
| --- | --- |
| URL | 8,192 Unicode characters before parsing |
| Title | 1–500 characters after trim/normalization |
| Description | 0–2,000 characters |
| Collection name | 1–100 characters |
| Tag name | 1–64 characters |
| Note body | 1–100,000 characters |
| Search query | 1–2,000 characters for semantic/hybrid; 0–2,000 for keyword browse |
| Filter dimension | At most 100 IDs/values |
| Page limit | 1–100; default 50 |
| Analytics time series | At most 366 days; default previous 30 days |
| Analytics top list | At most 20 rows |

## Bookmark projections

`BookmarkSummary`:

```json
{
  "id": 42,
  "source_kind": "url",
  "canonical_url": "https://example.com/article",
  "title": "Example article",
  "description": "Saved description",
  "source_domain": "example.com",
  "status": "inbox",
  "version": 3,
  "capture_available": true,
  "current_snapshot": {"id": 81, "content_version": 2, "captured_at": "2026-09-13T08:00:00Z"},
  "latest_capture_state": "succeeded",
  "collections": [{"id": 2, "name": "Research"}],
  "tags": [{"id": 5, "name": "database"}],
  "created_at": "2026-09-12T08:00:00Z",
  "updated_at": "2026-09-13T08:00:00Z"
}
```

Nullable fields are `canonical_url`, `description`, `source_domain`, `current_snapshot`, and `latest_capture_state`. Legacy projections set `source_kind` to `legacy_text`/`legacy_file`, `capture_available` to false, and identify migrated content through `current_snapshot` when present.

`BookmarkDetail` contains every summary field plus:

```json
{
  "original_url": "https://example.com/article?utm_source=old",
  "url_normalization_revision": 1,
  "notes": [{"id": 7, "body": "Use for project X", "version": 1, "created_at": "...", "updated_at": "..."}],
  "snapshot": {"id": 81, "content_version": 2, "final_url": "https://example.com/article", "http_status": 200, "content_type": "text/html", "content_checksum": "sha256:...", "captured_at": "...", "text": "..."},
  "recent_capture_attempts": []
}
```

`original_url`, `url_normalization_revision`, and `snapshot` are nullable. Managed file references never appear.

## Bookmark mutations

Create request:

```json
{"url": "https://example.com/article", "collection_ids": [2], "tags": ["database"]}
```

`collection_ids` and `tags` are optional and default empty. Create response data is:

```json
{"bookmark": {}, "created": true, "capture_attempt": {}}
```

Update request accepts only nullable-present fields from:

```json
{"title": "New title", "description": "Context", "status": "active"}
```

At least one field is required. Empty title is rejected; empty description is stored as an empty string. Organization changes use membership endpoints, not bookmark patch arrays.

## Capture and snapshot

`CaptureAttempt`:

```json
{
  "id": 15,
  "bookmark_id": 42,
  "attempt_number": 3,
  "state": "failed",
  "retry_count": 3,
  "max_retries": 3,
  "resulting_snapshot_id": null,
  "error_code": "SNAPSHOT_FETCH_TIMEOUT",
  "retryable": true,
  "created_at": "...",
  "updated_at": "...",
  "completed_at": "..."
}
```

Lease/checkpoint/internal errors are not public fields. `SuccessfulSnapshot` uses the exact `snapshot` projection defined in bookmark detail.

## Organization resources

- `Collection = {id, name, version, created_at, updated_at}`.
- `Tag = {id, name, version, created_at, updated_at}`.
- `Note = {id, bookmark_id, body, version, created_at, updated_at}`.
- Create requests contain only `name` or `body`.
- Patch requests contain only `name` or `body` and require the resource `If-Match`.
- Bookmark membership success is `{bookmark_id, resource_id, attached, bookmark_version}`.

## Shared filters

```json
{
  "collection_ids": [],
  "tag_ids": [],
  "statuses": ["inbox", "active"],
  "domains": [],
  "source_kinds": ["url", "legacy_text", "legacy_file"],
  "saved_from": null,
  "saved_to": null,
  "captured_from": null,
  "captured_to": null
}
```

## Search

Request:

```json
{"query": "database indexing", "mode": "hybrid", "filters": {}, "sort": "saved_desc", "limit": 50, "cursor": null}
```

Result item:

```json
{
  "bookmark": {},
  "excerpt": "Matched current snapshot text...",
  "matched_modes": ["keyword", "semantic"],
  "score": 0.032522
}
```

Response meta extends the normal envelope with:

```json
{
  "requested_mode": "hybrid",
  "actual_mode": "keyword",
  "components": {"fts": "ready", "provider": "degraded", "rocksdb": "ready", "chroma": "ready"},
  "degradations": [{"code": "PROVIDER_UNAVAILABLE", "message": "Semantic search is temporarily unavailable."}]
}
```

Scores are finite numbers comparable only within one response.

## Analytics

Time-series request:

```json
{"filters": {}, "from": "2026-08-15T00:00:00Z", "to": "2026-09-14T00:00:00Z", "interval": "day"}
```

`interval` is `day | week | month`. Response data is `{series: [{start, end, bookmark_count}]}`.

Totals request is `{filters: {}}`; response data is:

```json
{
  "totals": {"active_bookmarks": 10, "snapshots_ready": 8, "captures_failed": 1, "jobs_failed": 0},
  "by_status": [{"status": "active", "count": 10}],
  "top_domains": [{"domain": "example.com", "count": 4}],
  "top_collections": [{"id": 2, "name": "Research", "count": 4}],
  "top_tags": [{"id": 5, "name": "database", "count": 3}]
}
```

Analytics meta contains `actual_backend: duckdb | sqlite`, `source_watermark`, and safe degradations.

## Semantic settings

`SemanticSettings`:

```json
{
  "endpoint": "https://embeddings.example/v1",
  "model_id": "example-embedding-model",
  "timeout_ms": 10000,
  "api_key_present": true,
  "consent_state": "granted",
  "disclosure_version": "1",
  "version": 2,
  "verified": true,
  "active_index_revision_id": 3,
  "state": "ready"
}
```

Update request accepts exactly `{endpoint, model_id, timeout_ms}` with timeout 1,000–120,000 ms. Verify has an empty `{}` request and returns `{verified, model_id, vector_dimension, state}`. Consent accepts `{granted: true, disclosure_version: "1"}`. No request or response contains `api_key`.

## Health and maintenance

Health data is:

```json
{
  "status": "degraded",
  "components": {
    "sqlite": {"state": "ready", "code": null},
    "fts": {"state": "ready", "code": null},
    "worker": {"state": "ready", "code": null},
    "provider": {"state": "degraded", "code": "PROVIDER_UNAVAILABLE"},
    "rocksdb": {"state": "ready", "code": null},
    "chroma": {"state": "ready", "code": null},
    "duckdb": {"state": "ready", "code": null},
    "analytics_projection": {"state": "ready", "code": null}
  }
}
```

Rebuild request is `{target: "fts|rocksdb|chroma|duckdb|all-derived", confirm: true}`. Maintenance job data is `{id, target, state, progress_current, progress_total, checkpoint_label, error_code, created_at, updated_at, completed_at}`; counts are non-negative integers and nullable when unknown.
