# Storage architecture

## Ownership matrix

| Store | Purpose | Authority | Writes | Reads | Failure behavior |
| --- | --- | --- | --- | --- | --- |
| SQLite + FTS5 | Canonical bookmarks, snapshots, organization, jobs, settings metadata, and keyword search | Authoritative | Application services and durable workers | All services | Core health unavailable when SQLite cannot be read safely; FTS failure makes retrieval unavailable until rebuilt. |
| RocksDB through `rocksdict` | Reusable document/query embedding cache and provider-result metadata | Derived cache | Semantic index/search services | Semantic index/search services | Treat as cache miss, recreate store, and re-embed as necessary. |
| ChromaDB | Persistent semantic chunk vectors and metadata filters | Derived index | Semantic index worker | Search orchestrator | Semantic degrades; keyword and canonical workflows remain available. |
| DuckDB | Analytics projection and aggregate acceleration | Derived analytical store | Projection refresh worker | Analytics service, read-only | Use bounded SQLite fallback or report analytics-only degradation. |

## Shared derived identity

Every RocksDB cache entry, ChromaDB vector, and DuckDB projection row is namespaced by a deterministic compatibility identity containing:

```text
bookmark_id
content_version
index_revision_id
schema_version
```

Embedding entries additionally include provider endpoint identity, model ID, vector dimension, normalized input checksum, and embedding request revision. Chroma chunk records include chunk ID, chunk position, chunk checksum, and deleted/current-version metadata. DuckDB projection metadata includes source watermark and projection checksum.

## SQLite and FTS5

- SQLite transactions commit canonical entities and durable jobs atomically where the workflow requires both.
- FTS indexes only current, non-deleted bookmark metadata and snapshot text.
- FTS content can be rebuilt from canonical rows and managed snapshot files.
- List/detail may continue during an isolated FTS failure, but core retrieval health is unavailable rather than falsely healthy.
- Migrations are versioned, transactional where SQLite permits, and preceded by a verified backup during upgrade.

## RocksDB cache

- Keys are content-addressed and revision-scoped; values contain vectors plus bounded provider metadata.
- Query embeddings use a separate key namespace from document embeddings.
- Cache corruption, missing keys, or revision mismatch is a miss, never a canonical-data error.
- Revoking consent stops cache population. Existing cache cleanup follows an explicit semantic cleanup job and never touches SQLite snapshots.
- RocksDB is excluded from required backup and may be deleted only by an explicit rebuild/cleanup operation targeting its resolved path.

## ChromaDB index

- One active collection/index namespace corresponds to one compatible index revision.
- Upsert identity is deterministic for bookmark, content version, chunk, and revision.
- Search candidates are validated against SQLite before projection to users.
- A model, dimension, chunking, or schema change creates a new revision; it does not mutate an incompatible active index in place.
- Activation switches only after the new revision passes completeness and query smoke checks.

## DuckDB analytics

- DuckDB reads a controlled projection of canonical fields required by accepted metrics; it does not attach arbitrary user databases.
- Application queries are read-only and bounded by accepted filters/date ranges.
- Refresh records the SQLite source watermark and projection checksum within the projection workflow.
- A stale or failed projection is not shown as current. The service uses bounded SQLite aggregation or returns analytics degradation metadata.
- DuckDB never writes back to SQLite.

## Consistency and backup

- Canonical backup contains SQLite, managed snapshot files, schema/app version, timestamps, and checksums.
- RocksDB, ChromaDB, and DuckDB are excluded from required backup because they are rebuildable.
- Restore validates canonical integrity before any derived component starts.
- Derived cleanup and rebuild are idempotent, resumable, scoped to explicit paths, and recorded as durable jobs.
- No derived store may repair, create, undelete, or change a canonical bookmark.
