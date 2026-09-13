# API error registry

Every error uses the common envelope. `retryable` describes whether repeating the same logical operation may succeed without changing user input; clients still respect explicit retry guidance and limits.

| HTTP | Code | Retryable | Meaning |
| --- | --- | --- | --- |
| 400 | `INVALID_REQUEST` | no | Malformed JSON or request shape. |
| 400 | `INVALID_CURSOR` | no | Cursor is malformed or does not match endpoint/sort/filters. |
| 400 | `INVALID_DATE_RANGE` | no | Range/order/interval is invalid or exceeds 366 days. |
| 413 | `CAPTURE_CONTENT_TOO_LARGE` | no | Remote compressed/decompressed content exceeded a configured bound. |
| 422 | `VALIDATION_FAILED` | no | One or more fields violate documented limits/values. |
| 422 | `URL_NOT_ALLOWED` | no | URL scheme, credentials, host, port, or syntax is not accepted. |
| 422 | `URL_DESTINATION_UNSAFE` | no | DNS, connected address, or redirect targets a prohibited destination. |
| 404 | `BOOKMARK_NOT_FOUND` | no | Bookmark is absent or soft-deleted. |
| 404 | `COLLECTION_NOT_FOUND` | no | Collection does not exist. |
| 404 | `TAG_NOT_FOUND` | no | Tag does not exist. |
| 404 | `NOTE_NOT_FOUND` | no | Note does not exist under the bookmark. |
| 404 | `CAPTURE_ATTEMPT_NOT_FOUND` | no | Capture attempt does not exist under the bookmark. |
| 404 | `SNAPSHOT_NOT_FOUND` | no | Bookmark has no successful current snapshot. |
| 409 | `RESOURCE_VERSION_CONFLICT` | no | `If-Match` version is stale. |
| 409 | `NORMALIZED_NAME_CONFLICT` | no | Collection/tag normalized name already exists. |
| 409 | `CAPTURE_ALREADY_ACTIVE` | yes | Bookmark already has queued/processing capture. |
| 409 | `CAPTURE_NOT_RETRYABLE` | no | Attempt is not latest failed or has no manual retry transition. |
| 409 | `LEGACY_CAPTURE_UNSUPPORTED` | no | Legacy text/file item cannot capture or recapture. |
| 428 | `PRECONDITION_REQUIRED` | no | Required `If-Match` header is absent. |
| 429 | `WORK_QUEUE_BUSY` | yes | Bounded local queue limit is reached. |
| 502 | `CAPTURE_UPSTREAM_FAILED` | yes | Public source returned an unusable upstream response. |
| 504 | `SNAPSHOT_FETCH_TIMEOUT` | yes | Fetch/extraction exceeded time bound. |
| 503 | `SQLITE_UNAVAILABLE` | yes | Canonical database cannot be used safely. |
| 503 | `FTS_UNAVAILABLE` | yes | Keyword index is unavailable and requires repair/rebuild. |
| 503 | `WORKER_UNAVAILABLE` | yes | In-process worker failed initialization or stopped unexpectedly. |
| 409 | `PROVIDER_NOT_CONFIGURED` | no | Endpoint/model/key presence/verification is incomplete. |
| 403 | `PROVIDER_CONSENT_REQUIRED` | no | External content/query transfer has not been granted. |
| 422 | `PROVIDER_ENDPOINT_UNSAFE` | no | Provider URL violates HTTPS/loopback-HTTP policy. |
| 502 | `PROVIDER_INCOMPATIBLE` | no | Response shape/model/dimension violates the embedding contract. |
| 503 | `PROVIDER_UNAVAILABLE` | yes | Configured provider is temporarily unreachable/failing. |
| 503 | `ROCKSDB_UNAVAILABLE` | yes | Cache is unusable; owning response may continue via bypass. |
| 503 | `CHROMA_UNAVAILABLE` | yes | Semantic index is unusable; search may continue via keyword. |
| 503 | `DUCKDB_UNAVAILABLE` | yes | Analytics projection is unusable; request may continue through SQLite. |
| 409 | `INDEX_REVISION_MISMATCH` | no | Derived data is incompatible with active model/chunk/schema revision. |
| 409 | `ANALYTICS_PROJECTION_STALE` | yes | Projection watermark is behind canonical data. |
| 409 | `RETRY_EXHAUSTED` | no | Automatic three-attempt budget is exhausted; manual retry is required. |
| 410 | `LEGACY_WRITE_REMOVED` | no | Deprecated text/file creation or upload no longer has canonical behavior. |
| 409 | `MIGRATION_REQUIRED` | no | Database schema must be migrated before serving requests. |
| 500 | `MIGRATION_FAILED` | no | Migration aborted and active canonical data was not switched. |
| 500 | `BACKUP_FAILED` | yes | Backup did not complete verification. |
| 422 | `BACKUP_INVALID` | no | Manifest/checksum/path validation failed. |
| 500 | `RESTORE_FAILED` | no | Restore failed; active installation remains unchanged. |
| 409 | `REBUILD_CONFIRMATION_REQUIRED` | no | Explicit target confirmation is absent. |
| 500 | `REBUILD_FAILED` | yes | Derived staging rebuild failed; canonical/last safe revision remains. |

Degraded component codes may appear in successful response metadata when fallback completed safely. The top-level HTTP status then reflects the fulfilled request, not the unavailable optional path. Logs may attach internal exception categories to a correlation ID but never change or expose the public registry contract.
