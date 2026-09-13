# MVP acceptance

The MVP is releasable only when every applicable item has recorded evidence or an explicit reviewed waiver.

## Product flow

- Capture a valid public URL, preserve the bookmark immediately, complete metadata/snapshot, and retain it after restart.
- Reject invalid/private/unsafe URLs without unintended canonical data.
- Preserve a bookmark and offer idempotent retry after snapshot failure.
- Organize with multiple collections/tags, notes, and status without index coupling.
- Soft delete excludes the bookmark from detail, browse, all search modes, and analytics.
- Browse/search context survives detail navigation at desktop and mobile viewports.

## Retrieval and analytics

- Keyword search operates with provider, RocksDB, and ChromaDB unavailable.
- A configured, consented compatible endpoint completes document indexing and semantic queries.
- Semantic/hybrid provider failure reports degradation and falls back to keyword.
- Current-version, deleted, filter, deduplication, excerpt, and deterministic fusion rules pass.
- DuckDB metrics match canonical fixtures; stale/unavailable DuckDB uses equivalent bounded SQLite fallback.
- Published keyword, semantic, hybrid relevance and performance targets pass.

## Data, privacy, and recovery

- SQLite is proven authoritative; each derived record has version/revision identity.
- No snapshot/query is sent before explicit consent; revocation stops new calls before returning success.
- API keys and raw content are absent from API output, logs, diagnostics, and backup manifests.
- Migration, crash/retry, lease recovery, cache/vector incompatibility, and cleanup scenarios preserve canonical data.
- Verified backup restores SQLite and snapshots, then rebuilds FTS, RocksDB, ChromaDB, and DuckDB in order.
- Clean installation, health, upgrade, rollback, and data-path safeguards pass from a clean checkout.

## Documentation gate

- Product requirements map to feature behavior, architecture contract, tests, and implementation tasks.
- Public API/state/schema/storage documentation matches the released behavior.
- Setup and recovery commands have clean-environment transcripts.
- Out-of-scope ideas do not appear as implemented or committed features.
