# Backup, restore, and rebuild

**Package:** REL-RECOVERY  
**Status:** not_started  
**Milestone:** [90-reliability](../README.md)

## Outcome

Create SQLite-consistent canonical backups with managed snapshots and manifests, then restore safely and rebuild all derived stores in dependency order.

## Navigation

- [Plan](plan.md)
- [Authoritative tasks](tasks.md)
- [Examples](examples.md)
- [References](references.md)
- [Execution evidence](execution.md)

## Dependencies

DB-MIGRATION, RET-FTS, VEC-CHROMA, CACHE-ROCKS, ANA-PROJECTION, REL-HEALTH. See [dependency map](../../00-program/dependency-map.md).

## Owned work

- [REL-RECOVERY-001](tasks.md#rel-recovery-001) — Create consistent canonical backup and manifest
- [REL-RECOVERY-002](tasks.md#rel-recovery-002) — Restore to a safe location and validate canonical state
- [REL-RECOVERY-003](tasks.md#rel-recovery-003) — Rebuild derived stores in documented order

## Non-goals

This package does not move authority out of SQLite, hide known staleness, or claim success without reproducible evidence.
