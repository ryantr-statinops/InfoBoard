# Analytics API and UI

**Package:** ANA-API  
**Status:** not_started  
**Milestone:** [80-analytics](../README.md)

## Outcome

Expose all-time totals/top lists and bounded UTC time series through shared filters with clear backend and degradation metadata.

## Navigation

- [Plan](plan.md)
- [Authoritative tasks](tasks.md)
- [Examples](examples.md)
- [References](references.md)
- [Execution evidence](execution.md)

## Dependencies

API-CANONICAL, ANA-PROJECTION, ORG-UI. See [dependency map](../../00-program/dependency-map.md).

## Owned work

- [ANA-API-001](tasks.md#ana-api-001) — Implement analytics request and response contracts
- [ANA-API-002](tasks.md#ana-api-002) — Apply shared filters and backend metadata
- [ANA-API-003](tasks.md#ana-api-003) — Build bounded dashboard analytics states

## Non-goals

This package does not move authority out of SQLite, hide known staleness, or claim success without reproducible evidence.
