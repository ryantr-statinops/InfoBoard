# Guide — Recovery verification

This guide is for a copied test database only. Do not point destructive commands at a developer's active `data/` directory.

## Prerequisites

- A disposable database copy with known item, note, collection, and job counts.
- A writable temporary backup/output directory.
- The app stopped or worker paused.

## Steps

1. Create a backup and save its manifest/checksum output.
2. Validate the backup before changing the test copy.
3. Restore into a new path and run migrations.
4. Delete or isolate derived stores, then rebuild them from SQLite.
5. Run integrity and health checks, then compare counts/content/relations with the baseline.

## Expected result and evidence

The restored copy has matching canonical data; derived indexes are rebuilt; deleted items remain excluded. Save command output, manifest, comparison, and date in [execution.md](../execution.md).

## Failure and rollback

Stop on checksum, migration, or integrity failure. Keep the original copy untouched and restore from the last verified backup.
