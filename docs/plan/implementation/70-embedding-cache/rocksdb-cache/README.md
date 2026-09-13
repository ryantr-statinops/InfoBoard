# RocksDB embedding cache

**Package:** CACHE-ROCKS  
**Status:** not_started  
**Milestone:** [70-embedding-cache](../README.md)

## Outcome

Reuse compatible document embeddings through a revision-scoped cache that can be bypassed, cleaned, or rebuilt without affecting canonical state.

## Navigation

- [Plan](plan.md)
- [Authoritative tasks](tasks.md)
- [Examples](examples.md)
- [References](references.md)
- [Execution evidence](execution.md)

## Dependencies

SEM-EMBED; see [dependency map](../../00-program/dependency-map.md).

## Owned work

- [CACHE-ROCKS-001](tasks.md#cache-rocks-001) — Define versioned cache keys and validated values
- [CACHE-ROCKS-002](tasks.md#cache-rocks-002) — Integrate safe hit, miss, and bypass behavior
- [CACHE-ROCKS-003](tasks.md#cache-rocks-003) — Implement revocation cleanup and rebuild semantics

## Non-goals

This package does not make a derived store authoritative, send content without consent, or redefine provider/search contracts.
