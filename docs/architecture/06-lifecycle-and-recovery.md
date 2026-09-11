# Architecture — Lifecycle and recovery

## Current state

Soft delete và reindex endpoint tồn tại một phần; derived cleanup, versioned migrations và verified backup/restore/rebuild chưa hoàn chỉnh.

## Target delete pipeline

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

- Backup tối thiểu gồm SQLite, snapshots và format/version metadata.
- Derived stores không cần nằm trong authoritative backup.
- Restore không ghi đè dữ liệu hiện có nếu chưa có explicit path/backup.
- Schema rollback dùng restore backup; không tự động down-migrate.
- Rebuild phải resumable/idempotent và không sửa notes/collections/status.
