# Health and observability

**Package:** REL-HEALTH  
**Status:** not_started  
**Milestone:** [90-reliability](../README.md)

## Outcome

Expose safe component health, durable progress, correlation, and actionable degradation across SQLite, FTS, worker, provider, RocksDB, ChromaDB, and DuckDB.

## Navigation

- [Plan](plan.md)
- [Authoritative tasks](tasks.md)
- [Examples](examples.md)
- [References](references.md)
- [Execution evidence](execution.md)

## Dependencies

All component-owning packages through ANA-PROJECTION. See [dependency map](../../00-program/dependency-map.md).

## Owned work

- [REL-HEALTH-001](tasks.md#rel-health-001) — Implement component and overall health semantics
- [REL-HEALTH-002](tasks.md#rel-health-002) — Add correlation and redacted structured diagnostics
- [REL-HEALTH-003](tasks.md#rel-health-003) — Expose maintenance job progress safely

## Non-goals

This package does not move authority out of SQLite, hide known staleness, or claim success without reproducible evidence.
