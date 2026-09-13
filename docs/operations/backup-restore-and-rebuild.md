# Backup, restore, and rebuild

## Backup procedure contract

1. Resolve canonical database, snapshot source, and destination; reject overlap or unsafe broad paths.
2. Create a transactionally consistent SQLite backup without copying a live database file unsafely.
3. Copy only snapshot files referenced by the backup transaction.
4. Write a manifest with app/schema version, UTC time, relative paths, sizes, and checksums.
5. Open the backup read-only; run SQLite integrity/foreign-key checks and verify every referenced snapshot/checksum.
6. Mark backup complete only after verification. Derived stores are not required backup content.

## Restore procedure contract

1. Refuse to overwrite populated canonical data without a verified safety backup.
2. Validate archive paths, manifest, checksums, versions, and sizes before extraction/use.
3. Restore into a temporary or explicitly selected destination.
4. Verify SQLite and snapshot references, then run versioned migrations.
5. Keep provider calls disabled until endpoint, key reference, and restored consent are reviewed.
6. Switch canonical data only after read/list/keyword smoke checks pass.
7. Rebuild derived components and run full health/consistency checks.

## Rebuild targets and order

```text
fts
-> rocksdb
-> chroma
-> duckdb
-> all component health and consistency checks
```

- `fts` rebuilds current, non-deleted canonical content locally.
- `rocksdb` creates a revision-scoped embedding cache; external calls require active consent.
- `chroma` builds a new revision namespace and activates only after completeness/dimension/query checks.
- `duckdb` builds a staging projection and activates after watermark/checksum/parity checks.
- `all-derived` follows this order and may pause before external work when provider/consent is unavailable.

Every operation requires an explicit target, resolved path, confirmation, durable progress, idempotent retry, and safe cancellation boundary. It never changes canonical notes, collections, tags, bookmark state, URL, or snapshots.

## Failure and rollback

- Backup failure leaves no artifact marked complete.
- Restore failure leaves the active installation unchanged and preserves the safety backup.
- Migration failure returns to the safety backup; automatic destructive down-migration is unsupported.
- Derived rebuild failure preserves the last compatible active revision and the staging failure evidence.
- Canonical corruption is investigated on a copy; deleting the database is never a troubleshooting step.

## Required evidence

Record command/action, exact safe paths, commit/version, environment, source and result checksums, row/file counts, component revisions, duration, health result, and redacted failure output. A documented but unexecuted procedure is not release evidence.
