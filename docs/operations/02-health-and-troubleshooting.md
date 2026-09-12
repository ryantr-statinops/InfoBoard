# Operations — Health and troubleshooting

## Current state

The basic health endpoint exists; component-level status, the diagnostics bundle, and verified runbooks remain draft/partial.

## Target health model

`/api/health` reports `ok`, `degraded`, or `unavailable` for SQLite, FTS, the worker, ChromaDB, RocksDB, DuckDB, and the model. SQLite or FTS5 unavailability makes core health return HTTP 503; derived dependency unavailability returns HTTP 200 with degraded status.

When FTS5 fails, list/detail may still read from SQLite, but the application must not be advertised as healthy core mode because keyword retrieval is mandatory. The UI must state that keyword search is unavailable and guide the user to check/rebuild FTS.

## Troubleshooting routes

### Startup failure

- Check Python/dependencies, config/data path, permissions, SQLite integrity, and migration version.
- Do not delete the database to retry; work on a copy when investigating corruption.

### Stuck or failed indexing

- Identify the job/item/content version, state, retry count, and error code.
- Requeue only the current version of a non-deleted item.
- After the retry limit, keep the failed state and use the rebuild/diagnostic workflow.

### Semantic or analytics degraded

- Confirm that the core keyword/read flow still works.
- Check dependency/model metadata, derived-store paths, and health details.
- Rebuild the derived store from SQLite after source-data integrity is confirmed.

### Search/data mismatch

- Compare the item current version/deleted state with FTS/vector candidates.
- Rebuild the index; do not edit SQLite based on derived-store content.

## Diagnostics boundary

The bundle/log contains component versions, timing, job/error codes, and hashed identifiers; it does not contain raw content, sensitive queries, tokens, secrets, or unnecessary filesystem details.
