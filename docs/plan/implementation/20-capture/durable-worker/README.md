# Durable sequential worker

**Package:** CAP-WORKER  
**Status:** not_started  
**Milestone:** [20-capture](../README.md)

## Outcome

Run one in-process worker that safely leases, retries, checkpoints, and recovers capture and derived jobs.

## Navigation

- [Implementation plan](plan.md)
- [Authoritative tasks](tasks.md)
- [Examples](examples.md)
- [Canonical references](references.md)
- [Execution evidence](execution.md)

## Dependencies

APP-LIFECYCLE, DB-MIGRATION, CAP-SNAPSHOT. See the program [dependency map](../../00-program/dependency-map.md).

## Owned work

- [CAP-WORKER-001](tasks.md#cap-worker-001) — Implement sequential durable claim and lease semantics
- [CAP-WORKER-002](tasks.md#cap-worker-002) — Add bounded retry and safe checkpoint recovery
- [CAP-WORKER-003](tasks.md#cap-worker-003) — Integrate worker lifecycle and graceful shutdown

## Non-goals

This package does not redefine upstream contracts, claim unevidenced runtime completion, or take ownership from another package.
