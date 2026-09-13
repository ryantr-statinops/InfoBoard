# DuckDB analytics projection

**Package:** ANA-PROJECTION  
**Status:** not_started  
**Milestone:** [80-analytics](../README.md)

## Outcome

Build a persistent, read-only DuckDB projection with refresh checkpoints, stale detection, rebuild, and schema-equivalent bounded SQLite fallback.

## Navigation

- [Plan](plan.md)
- [Authoritative tasks](tasks.md)
- [Examples](examples.md)
- [References](references.md)
- [Execution evidence](execution.md)

## Dependencies

DB-MIGRATION, ORG-CONTEXT, RET-FTS, CAP-WORKER. See [dependency map](../../00-program/dependency-map.md).

## Owned work

- [ANA-PROJECTION-001](tasks.md#ana-projection-001) — Define the analytics projection and checkpoint
- [ANA-PROJECTION-002](tasks.md#ana-projection-002) — Implement atomic refresh and stale detection
- [ANA-PROJECTION-003](tasks.md#ana-projection-003) — Implement bounded SQLite fallback and rebuild

## Non-goals

This package does not move authority out of SQLite, hide known staleness, or claim success without reproducible evidence.
