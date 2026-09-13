# Canonical schema and migrations

**Package:** DB-MIGRATION  
**Status:** not_started  
**Milestone:** [10-foundation](../README.md)

## Outcome

Migrate the existing integer-ID database to bookmarks, attempts, immutable snapshots, organization, revisions, and checkpoints without losing user data.

## Navigation

- [Implementation plan](plan.md)
- [Authoritative tasks](tasks.md)
- [Examples](examples.md)
- [Canonical references](references.md)
- [Execution evidence](execution.md)

## Dependencies

APP-LIFECYCLE. See the program [dependency map](../../00-program/dependency-map.md).

## Owned work

- [DB-MIGRATION-001](tasks.md#db-migration-001) — Create a versioned migration runner and target canonical schema
- [DB-MIGRATION-002](tasks.md#db-migration-002) — Forward-migrate existing items and relationships
- [DB-MIGRATION-003](tasks.md#db-migration-003) — Validate activation and rebuild derived state

## Non-goals

This package does not redefine upstream contracts, claim unevidenced runtime completion, or take ownership from another package.
