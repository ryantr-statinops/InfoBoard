# Data Architecture and Database Responsibilities

## Guiding rule

Each storage system owns one workload. SQLite is the source of truth; the other databases are derived, operational, or analytical layers and must be rebuildable from SQLite plus the saved content.

## SQLite: application source of truth

SQLite stores the durable product data:

- items and their source metadata;
- extracted text snapshots;
- chunks and their stable identifiers;
- collections and many-to-many item memberships;
- reading state and item-level personal notes;
- search history and configuration metadata.

SQLite owns the relationships that render the reader, library, and collection views. An item is never considered valid solely because an index entry exists.

## ChromaDB: semantic retrieval index

ChromaDB stores embeddings keyed by SQLite chunk IDs. It returns the most semantically similar chunk IDs and scores; InfoBoard then fetches the authoritative item metadata and text snippets from SQLite.

ChromaDB is not the authoritative metadata store. Its collection can be deleted and rebuilt by re-indexing SQLite chunks.

## RocksDB: operational state and cache

RocksDB handles high-frequency, disposable key-value data:

- indexing job state (`queued`, `extracting`, `embedding`, `completed`, `failed`);
- embedding cache keyed by `content_hash + embedding_provider + model_version`;
- optional short-lived progress or retry markers.

The embedding cache prevents duplicate work when identical text is imported again. Durable item-level status is also mirrored in SQLite when it affects the UI or user recovery.

## DuckDB: read-only analytics layer

DuckDB attaches SQLite in read-only mode to answer dashboard queries without duplicating the application database. It powers counts and grouped analysis such as items by collection/source, reading-state distribution, and activity over time.

DuckDB must not be used for transactional writes. Analytics queries should degrade gracefully if its local extension is unavailable.

## Data flow

```text
URL/file/text
  -> extractor and normalizer
  -> SQLite item + snapshot + chunks
  -> RocksDB cache/job state
  -> ChromaDB chunk embeddings

Search query
  -> SQLite full-text/keyword retrieval
  -> ChromaDB semantic retrieval
  -> SQLite hydration of visible results

SQLite (read-only attach)
  -> DuckDB aggregate queries
  -> dashboard
```

