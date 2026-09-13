# Canonical API foundation

**Package:** API-CANONICAL  
**Status:** not_started  
**Milestone:** [10-foundation](../README.md)

## Outcome

Provide `/api/v1/*` envelopes, validation, pagination, stable errors, correlation IDs, and optimistic concurrency while isolating compatibility adapters.

## Navigation

- [Implementation plan](plan.md)
- [Authoritative tasks](tasks.md)
- [Examples](examples.md)
- [Canonical references](references.md)
- [Execution evidence](execution.md)

## Dependencies

APP-LIFECYCLE, DB-MIGRATION. See the program [dependency map](../../00-program/dependency-map.md).

## Owned work

- [API-CANONICAL-001](tasks.md#api-canonical-001) — Build shared API schemas, validation, cursors, and error handling
- [API-CANONICAL-002](tasks.md#api-canonical-002) — Implement optimistic mutation conventions
- [API-CANONICAL-003](tasks.md#api-canonical-003) — Separate canonical routers from compatibility adapters

## Non-goals

This package does not redefine upstream contracts, claim unevidenced runtime completion, or take ownership from another package.
