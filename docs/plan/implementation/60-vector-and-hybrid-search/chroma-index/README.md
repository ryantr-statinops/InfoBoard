# ChromaDB semantic index

**Package:** VEC-CHROMA  
**Status:** not_started  
**Milestone:** [60-vector-and-hybrid-search](../README.md)

## Outcome

Maintain a persistent revision-scoped vector index whose results are always revalidated against current canonical SQLite state.

## Navigation

- [Plan](plan.md)
- [Authoritative tasks](tasks.md)
- [Examples](examples.md)
- [References](references.md)
- [Execution evidence](execution.md)

## Dependencies

SEM-EMBED, CAP-WORKER; see [dependency map](../../00-program/dependency-map.md).

## Owned work

- [VEC-CHROMA-001](tasks.md#vec-chroma-001) — Implement revision-scoped collections and idempotent upserts
- [VEC-CHROMA-002](tasks.md#vec-chroma-002) — Validate semantic candidates against SQLite
- [VEC-CHROMA-003](tasks.md#vec-chroma-003) — Implement cleanup, health, and rebuild

## Non-goals

This package does not make a derived store authoritative, send content without consent, or redefine provider/search contracts.
