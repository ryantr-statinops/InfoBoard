# Embedding pipeline and revisions

**Package:** SEM-EMBED  
**Status:** not_started  
**Milestone:** [50-semantic-provider](../README.md)

## Outcome

Produce validated, bounded embeddings through durable revision-aware jobs and reject incompatible provider results.

## Navigation

- [Plan](plan.md)
- [Authoritative tasks](tasks.md)
- [Examples](examples.md)
- [References](references.md)
- [Execution evidence](execution.md)

## Dependencies

SEM-CONFIG, CAP-WORKER, CAP-SNAPSHOT; see [dependency map](../../00-program/dependency-map.md).

## Owned work

- [SEM-EMBED-001](tasks.md#sem-embed-001) — Create immutable index revision identity
- [SEM-EMBED-002](tasks.md#sem-embed-002) — Chunk and submit bounded snapshot content
- [SEM-EMBED-003](tasks.md#sem-embed-003) — Validate provider responses and durable outcomes

## Non-goals

This package does not make a derived store authoritative, send content without consent, or redefine provider/search contracts.
