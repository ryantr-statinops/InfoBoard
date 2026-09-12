# Operations — Backup, restore and rebuild

## Current state

The backup/restore/rebuild target is designed, but no verified release command/transcript exists yet.

## Backup contract

The authoritative backup includes the SQLite database, stored snapshots, and a manifest with app/schema version, timestamp, and checksums. ChromaDB/RocksDB/DuckDB artifacts are not required because they can be rebuilt.

## Target backup flow

1. Resolve and display the exact source/destination.
2. Ensure the SQLite backup is consistent; do not copy a live database unsafely.
3. Copy snapshots and create the manifest/checksums.
4. Verify that the backup opens and the SQLite integrity check passes.

## Target restore flow

1. Do not overwrite a populated destination without a safety backup.
2. Verify the manifest/checksums and restore to a temporary or explicitly selected location.
3. Run the SQLite integrity check.
4. Run versioned migrations.
5. Rebuild required FTS from SQLite/snapshots; rebuild vectors/cache only when full mode is configured for the restored installation.
6. Run health and the core smoke flow before switching to restored data.

## Rebuild rules

- Rebuild is idempotent/resumable and does not modify notes, collections, or organization status.
- A model/provider metadata mismatch creates a new index rather than reusing a cache with the wrong revision/dimension.
- Failure preserves canonical data and permits retry.
- A core-only restore is complete after canonical data, migrations, FTS, health, and core smoke verification; optional full-mode indexes do not block it.
