# Configuration

## Local defaults

- Bind address: `127.0.0.1`; no authentication is implied outside loopback.
- Each storage/snapshot path resolves beneath one explicit InfoBoard data directory unless individually overridden.
- Startup displays safe resolved paths without secrets and refuses ambiguous/unwritable canonical paths.
- Configuration precedence and `.env` loading behavior must be implemented and tested before setup documentation names exact commands.

## Component configuration

| Component | Required configuration | Startup behavior |
| --- | --- | --- |
| SQLite/FTS5 | Canonical database and snapshot paths | Fail startup/health if unsafe, incompatible, or unreadable. |
| Capture worker | Limits, user agent, concurrency, retry policy | Core capture unavailable if worker cannot initialize. |
| Semantic provider | Endpoint, model, write-only API key, timeout, explicit consent | `unconfigured` until all setup and consent conditions pass. |
| RocksDB | Derived cache path and active revision | Recreate/bypass on safe cache failure. |
| ChromaDB | Derived vector path and active revision | Semantic degraded when configured but unavailable. |
| DuckDB | Derived analytics path and projection revision | Use bounded SQLite fallback when unavailable. |

## Provider setup

1. Resolve and validate the HTTPS/explicitly permitted endpoint.
2. Store the API key through the secret boundary and persist only its opaque reference.
3. Record model ID, timeout, and request limits.
4. Verify compatible embeddings response and vector dimension with non-sensitive test input.
5. Show the versioned disclosure and obtain explicit installation-level consent.
6. Create an index revision and start semantic indexing.

Verification does not grant consent. Removing/revoking consent stops external calls before acknowledging completion and starts local semantic cleanup. Health and UI distinguish `unconfigured` from failure after configuration.

## Health interpretation

- SQLite or FTS5 unavailable: HTTP 503 and no healthy-core claim.
- Capture unavailable: bookmark read/organization may continue; new capture reports unavailable.
- Configured provider/Chroma unavailable: HTTP 200 degraded when keyword/canonical workflows are safe.
- RocksDB unavailable: degraded cache with provider calls only when consent remains valid.
- DuckDB unavailable: SQLite analytics fallback or analytics-only degradation.
- Never-configured provider: unconfigured setup state, not an outage; release evidence still requires a configured semantic flow.
