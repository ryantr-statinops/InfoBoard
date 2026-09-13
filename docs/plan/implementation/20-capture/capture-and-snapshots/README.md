# Bookmark capture and snapshots

**Package:** CAP-SNAPSHOT  
**Status:** not_started  
**Milestone:** [20-capture](../README.md)

## Outcome

Persist bookmarks and attempts before enrichment, then atomically activate immutable successful snapshots without disturbing personal context.

## Navigation

- [Implementation plan](plan.md)
- [Authoritative tasks](tasks.md)
- [Examples](examples.md)
- [Canonical references](references.md)
- [Execution evidence](execution.md)

## Dependencies

DB-MIGRATION, API-CANONICAL, CAP-URL. See the program [dependency map](../../00-program/dependency-map.md).

## Owned work

- [CAP-SNAPSHOT-001](tasks.md#cap-snapshot-001) — Create or resolve the canonical bookmark and durable attempt
- [CAP-SNAPSHOT-002](tasks.md#cap-snapshot-002) — Persist successful content as an immutable snapshot
- [CAP-SNAPSHOT-003](tasks.md#cap-snapshot-003) — Expose safe failure and idempotent retry

## Non-goals

This package does not redefine upstream contracts, claim unevidenced runtime completion, or take ownership from another package.
