# Semantic and hybrid retrieval

**Package:** VEC-HYBRID  
**Status:** not_started  
**Milestone:** [60-vector-and-hybrid-search](../README.md)

## Outcome

Orchestrate semantic query embeddings, vector candidates, shared filters, deterministic fusion, deduplication, and keyword fallback.

## Navigation

- [Plan](plan.md)
- [Authoritative tasks](tasks.md)
- [Examples](examples.md)
- [References](references.md)
- [Execution evidence](execution.md)

## Dependencies

RET-KEYWORD, SEM-CONFIG, VEC-CHROMA; see [dependency map](../../00-program/dependency-map.md).

## Owned work

- [VEC-HYBRID-001](tasks.md#vec-hybrid-001) — Implement semantic query execution
- [VEC-HYBRID-002](tasks.md#vec-hybrid-002) — Implement deterministic hybrid rank fusion
- [VEC-HYBRID-003](tasks.md#vec-hybrid-003) — Project explicit degradation and fallback

## Non-goals

This package does not make a derived store authoritative, send content without consent, or redefine provider/search contracts.
