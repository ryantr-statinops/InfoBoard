# Architecture — Lifecycle and recovery

## Current state

Soft delete and the reindex endpoint exist partially; derived cleanup, versioned migrations, and verified backup/restore/rebuild are incomplete.

## Target state — delete and recovery pipelines

```mermaid
flowchart LR
    Request[Delete confirmed] --> Soft[Set deleted_at in SQLite]
    Soft --> Hidden[Hide from list/detail/search]
    Soft --> Cleanup[Queue cleanup job]
    Cleanup --> Vector[Delete vectors]
    Cleanup --> Cache[Invalidate cache/job markers]
    Vector --> Verify[Verify no derived references]
    Cache --> Verify
```

## Target recovery pipeline

```mermaid
flowchart LR
    Backup[Backup SQLite + snapshots + metadata] --> Restore[Restore into validated location]
    Restore --> Integrity[SQLite integrity check]
    Integrity --> Migrate[Run versioned migrations]
    Migrate --> Rebuild[Rebuild FTS/vector/cache]
    Rebuild --> Health[Health + smoke verification]
```

## Recovery rules

- A minimum backup includes SQLite, snapshots, and format/version metadata.
- Derived stores do not need to be part of the authoritative backup.
- Restore must not overwrite existing data without an explicit path/safety backup.
- Schema rollback uses backup restore; automatic down-migration is not supported.
- Rebuild must be resumable/idempotent and must not modify notes/collections/status.
