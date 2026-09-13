# API contracts

`/api/v1/*` is the canonical JSON API. Exact payloads are defined in [API schemas](api-schemas.md); stable failures are defined in [the error registry](api-errors.md). HTML/HTMX routes use the same application services and cannot define different behavior.

## Conventions

- UTF-8 JSON, RFC 3339 UTC timestamps, and SQLite integer identifiers.
- Unknown mutation fields are rejected.
- Content type is `application/json` except HTML routes and future explicit export/download operations.
- Every response includes a correlation ID in the envelope and `X-Correlation-ID` header.
- Mutable resources expose a positive integer `version`.
- Update/delete and bookmark-membership mutations require `If-Match: "<version>"`; absent precondition returns `428`, stale version returns `409 RESOURCE_VERSION_CONFLICT`.
- Soft delete is the only bookmark-delete operation. MVP has no restore or permanent-delete endpoint.

## Pagination and filters

- Cursor pagination is opaque and versioned. The server encodes the active sort key and integer ID; clients must not construct or edit it.
- Default limit is 50 and maximum is 100.
- Supported bookmark/search sorts are `saved_desc`, `updated_desc`, and `title_asc`; default is `saved_desc`.
- A cursor is valid only with the same endpoint, sort, and normalized filters used to create it; mismatch returns `INVALID_CURSOR`.
- Shared filters allow at most 100 IDs/values per dimension. Values are OR within one dimension and dimensions are AND together.
- Date ranges are UTC half-open intervals `[from, to)`. Deleted bookmarks are always excluded.

## Canonical endpoints

### Bookmarks and capture

| Method and path | Preconditions and behavior | Success |
| --- | --- | --- |
| `POST /api/v1/bookmarks` | Validate/normalize URL, create or resolve active bookmark, apply requested organization, queue capture | `202` new bookmark/attempt; `200` existing bookmark |
| `GET /api/v1/bookmarks` | Cursor, limit, sort, shared filters | `200` paginated summaries |
| `GET /api/v1/bookmarks/{id}` | Non-deleted target | `200` detail |
| `PATCH /api/v1/bookmarks/{id}` | `If-Match`; editable metadata/status only | `200` detail with incremented version |
| `DELETE /api/v1/bookmarks/{id}` | `If-Match`; soft delete and derived cleanup | `202` maintenance job summary |
| `POST /api/v1/bookmarks/{id}/capture-attempts` | URL source, `If-Match`, no active attempt | `202` new recapture attempt and bookmark version |
| `POST /api/v1/bookmarks/{id}/capture-attempts/{attempt_id}/retry` | Latest failed attempt, `If-Match` | `202` requeued attempt and bookmark version |
| `GET /api/v1/bookmarks/{id}/capture-attempts` | Non-deleted target; limit/cursor | `200` newest-first attempt page |
| `GET /api/v1/bookmarks/{id}/snapshots/current` | Non-deleted target with current snapshot | `200` successful snapshot projection |

Legacy text/file items use bookmark read/update/organization/search/delete endpoints. Capture/recapture returns `409 LEGACY_CAPTURE_UNSUPPORTED`.

### Organization

| Method and path | Contract |
| --- | --- |
| `GET|POST /api/v1/collections` | Paginate or create a normalized unique collection. |
| `PATCH|DELETE /api/v1/collections/{id}` | Require collection `If-Match`; delete removes memberships only. |
| `PUT|DELETE /api/v1/bookmarks/{id}/collections/{collection_id}` | Require bookmark `If-Match`; idempotent membership mutation returns new bookmark version. |
| `GET|POST /api/v1/tags` | Paginate or create a normalized unique tag. |
| `PATCH|DELETE /api/v1/tags/{id}` | Require tag `If-Match`; delete removes assignments only. |
| `PUT|DELETE /api/v1/bookmarks/{id}/tags/{tag_id}` | Require bookmark `If-Match`; idempotent assignment returns new bookmark version. |
| `GET|POST /api/v1/bookmarks/{id}/notes` | Paginate or create a note. |
| `PATCH|DELETE /api/v1/bookmarks/{id}/notes/{note_id}` | Require note `If-Match`; note must belong to bookmark. |

### Retrieval and analytics

- `POST /api/v1/search` accepts the exact search request, validates filters, and returns bookmark-level results with requested/actual mode and component degradation.
- `POST /api/v1/analytics/timeseries` defaults to 30 days and rejects ranges over 366 days.
- `POST /api/v1/analytics/totals` returns all-time or filtered totals and top lists capped at 20.

### Semantic settings

| Method and path | Contract |
| --- | --- |
| `GET /api/v1/settings/semantic` | Endpoint/model/timeout, key presence, consent, revision, and health; never key value. |
| `PUT /api/v1/settings/semantic` | `If-Match`; accepts endpoint/model/timeout only and revokes verification until rechecked. |
| `POST /api/v1/settings/semantic/verify` | Minimal non-sensitive embedding compatibility check; does not grant consent. |
| `POST /api/v1/settings/semantic/consent` | `If-Match`; require disclosure version and `granted: true`. |
| `DELETE /api/v1/settings/semantic/consent` | `If-Match`; revoke before response, pause calls/jobs, queue local semantic cleanup. |

`api_key_present` is derived exclusively from `INFOBOARD_SEMANTIC_API_KEY`. The API has no key mutation field.

### Health and maintenance

- `GET /api/v1/health` returns overall and component states for canonical SQLite, FTS5, in-process worker, provider, RocksDB, ChromaDB, DuckDB, and analytics projection.
- `POST /api/v1/maintenance/rebuild` accepts `fts`, `rocksdb`, `chroma`, `duckdb`, or `all-derived` and returns `202`.
- `GET /api/v1/maintenance/jobs/{id}` returns safe durable progress.

Canonical SQLite or FTS unavailability returns health HTTP 503. Configured provider/derived failure returns HTTP 200 with overall `degraded` when canonical and keyword workflows remain safe. Incomplete provider setup is `unconfigured`, not an outage.

## One-release compatibility

For the first release containing `/api/v1`, target-compatible `/api/*` routes adapt to canonical services and response semantics. They add `Deprecation: true`, a release-manifest-derived `Sunset` header, and a migration `Link` header. They contain no independent storage/business logic.

- Item list/detail/update/delete, URL capture, collections, notes, search, related retrieval, analytics, health, and reindex routes map to canonical equivalents where behavior exists.
- Old text creation and upload routes have no URL-first canonical equivalent and return `410 LEGACY_WRITE_REMOVED` with migration guidance.
- Adapter removal is a required task for the next release; compatibility is not indefinite.
