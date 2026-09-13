# FTS5 indexing and rebuild

**Package:** RET-FTS  
**Status:** not_started  
**Milestone:** [40-keyword-retrieval](../README.md)

## Outcome

Maintain a derived FTS5 index for only active bookmarks and current snapshot versions, with deterministic cleanup and rebuild.

## Navigation

- [Implementation plan](plan.md)
- [Authoritative tasks](tasks.md)
- [Examples](examples.md)
- [Canonical references](references.md)
- [Execution evidence](execution.md)

## Dependencies

CAP-SNAPSHOT, CAP-WORKER, ORG-CONTEXT. See [dependency map](../../00-program/dependency-map.md).

## Owned work

- [RET-FTS-001](tasks.md#ret-fts-001) — Define the current-version FTS projection
- [RET-FTS-002](tasks.md#ret-fts-002) — Process idempotent keyword index jobs
- [RET-FTS-003](tasks.md#ret-fts-003) — Validate and rebuild FTS independently

## Non-goals

No upstream behavior is redefined and no unevidenced runtime capability is marked complete.
