# Keyword query orchestration

**Package:** RET-KEYWORD  
**Status:** not_started  
**Milestone:** [40-keyword-retrieval](../README.md)

## Outcome

Provide local keyword search with shared filters, safe query handling, stable pagination, excerpts, and no semantic dependency.

## Navigation

- [Implementation plan](plan.md)
- [Authoritative tasks](tasks.md)
- [Examples](examples.md)
- [Canonical references](references.md)
- [Execution evidence](execution.md)

## Dependencies

API-CANONICAL, RET-FTS, ORG-CONTEXT. See [dependency map](../../00-program/dependency-map.md).

## Owned work

- [RET-KEYWORD-001](tasks.md#ret-keyword-001) — Generate bounded keyword candidates
- [RET-KEYWORD-002](tasks.md#ret-keyword-002) — Project ranked bookmark results and excerpts
- [RET-KEYWORD-003](tasks.md#ret-keyword-003) — Integrate API and UI without provider coupling

## Non-goals

No upstream behavior is redefined and no unevidenced runtime capability is marked complete.
