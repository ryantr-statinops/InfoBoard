# Configuration

## Local defaults

- Bind address: `127.0.0.1`; no authentication is implied outside loopback.
- Each storage/snapshot path resolves beneath one explicit InfoBoard data directory unless individually overridden.
- Startup displays safe resolved paths without secrets and refuses ambiguous/unwritable canonical paths.
- Process environment is the MVP configuration source for secrets. `.env` auto-loading is not part of the contract.

## Component configuration

| Component | Required configuration | Startup behavior |
| --- | --- | --- |
| SQLite/FTS5 | Canonical database and snapshot paths | Fail startup/health if unsafe, incompatible, or unreadable. |
| Capture worker | Limits, user agent, concurrency, retry policy | Core capture unavailable if worker cannot initialize. |
| Semantic provider | Endpoint, model, timeout, explicit consent, and `INFOBOARD_SEMANTIC_API_KEY` environment presence | `unconfigured` until all setup and consent conditions pass. |
| RocksDB | Derived cache path and active revision | Recreate/bypass on safe cache failure. |
| ChromaDB | Derived vector path and active revision | Semantic degraded when configured but unavailable. |
| DuckDB | Derived analytics path and projection revision | Use bounded SQLite fallback when unavailable. |

## Provider setup

1. Resolve and validate an HTTPS endpoint or an HTTP endpoint resolving exclusively to loopback.
2. Set `INFOBOARD_SEMANTIC_API_KEY` in the application process environment and restart; the UI never accepts the key.
3. Record model ID, timeout, and request limits.
4. Verify compatible embeddings response and vector dimension with non-sensitive test input.
5. Show the versioned disclosure and obtain explicit installation-level consent.
6. Create an index revision and start semantic indexing.

Verification does not grant consent. Removing/revoking consent stops external calls before acknowledging completion and starts local semantic cleanup. Removing the environment key requires restart and produces `unconfigured`. Health and UI distinguish `unconfigured` from failure after complete configuration.

## Health interpretation

- SQLite or FTS5 unavailable: HTTP 503 and no healthy-core claim.
- Capture unavailable: bookmark read/organization may continue; new capture reports unavailable.
- Configured provider/Chroma unavailable: HTTP 200 degraded when keyword/canonical workflows are safe.
- RocksDB unavailable: degraded cache with provider calls only when consent remains valid.
- DuckDB unavailable: SQLite analytics fallback or analytics-only degradation.
- Never-configured provider: unconfigured setup state, not an outage; release evidence still requires a configured semantic flow.
