# 11 — Data model and migrations

**Plan status:** `ready`  
**Delivery status:** `not_started`  
**Baseline coverage:** `partial`  
**Milestone:** M1  
**Dependencies:** 10  
**System of record:** SQLite + FTS5

## Outcome

The schema is versioned, repeatable to upgrade, and preserves existing items, notes, collections, and jobs. Every derived index can be rebuilt from SQLite.

## Target schema and invariants

- `items` stores source metadata, `content_hash`, `content_version`, status default `inbox`, timestamps, and `deleted_at`.
- `item_contents` is identified by `(item_id, content_version)`.
- `chunks` stores version, `position`, text, hash, and stable index data.
- `collections`, `item_collections`, `notes`, `search_history`, `index_jobs`, and `schema_version` preserve the domain relationships.
- `index_jobs.retry_count` is the target field; the runtime `attempts` field requires migration.
- Foreign keys are enabled for every connection; item/collection relations are unique; content versions increase monotonically.
- Soft-deleted items are excluded from public queries while remaining available for cleanup and recovery.

## Migration strategy

1. Add a baseline migration for the running schema.
2. Number SQL migrations; use idempotency only for bootstrap.
3. Lock the database, run each migration transactionally, verify expected columns/indexes, and update the version.
4. Back up before a production migration. Roll back data through restore, not automatic down-migrations.
5. Change the default for newly created items from `active` to `inbox` without rewriting existing organization status.

## Failure and acceptance

Migration failure rolls back the transaction and reports version/file context. FTS is rebuilt from canonical tables when damaged and is never treated as the source of truth.

- Fresh DB has the expected schema, pragmas, and indexes.
- The current database snapshot upgrades without row loss.
- Re-running migrations creates no duplicates.
- Concurrent connections never observe a half-applied schema.
- Delete/restore preserves notes, collections, and content versions.
