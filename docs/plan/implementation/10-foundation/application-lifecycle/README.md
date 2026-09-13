# Application lifecycle and configuration

**Package:** APP-LIFECYCLE  
**Status:** not_started  
**Milestone:** [10-foundation](../README.md)

## Outcome

Establish typed settings, deterministic paths, startup/shutdown ordering, and a safe application lifespan.

## Navigation

- [Implementation plan](plan.md)
- [Authoritative tasks](tasks.md)
- [Examples](examples.md)
- [Canonical references](references.md)
- [Execution evidence](execution.md)

## Dependencies

None. See the program [dependency map](../../00-program/dependency-map.md).

## Owned work

- [APP-LIFECYCLE-001](tasks.md#app-lifecycle-001) — Introduce typed settings and validated local data paths
- [APP-LIFECYCLE-002](tasks.md#app-lifecycle-002) — Own startup and shutdown ordering in the FastAPI lifespan

## Non-goals

This package does not redefine upstream contracts, claim unevidenced runtime completion, or take ownership from another package.
