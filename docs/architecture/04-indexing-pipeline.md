# Architecture — Indexing pipeline

## Current state

The current worker lifecycle is a synchronous simulation; background polling, durable leases, real embeddings, and persistent Chroma integration are not implemented.

## Target contract

```mermaid
stateDiagram-v2
    [*] --> queued
    queued --> extracting
    extracting --> chunking
    chunking --> indexed: core mode
    chunking --> embedding: full mode enabled
    embedding --> indexed
    extracting --> failed
    chunking --> failed
    embedding --> failed
    failed --> queued: retry <= 3
    extracting --> queued: restart/requeue
    chunking --> queued: restart/requeue
    embedding --> queued: restart/requeue
```

```mermaid
flowchart LR
    Job[SQLite index job] --> Claim[Claim + lease]
    Claim --> Chunks[Build current-version chunks]
    Chunks --> Cache{Embedding cache hit?}
    Cache -->|yes| Upsert[Chroma upsert]
    Cache -->|no| Embed[Embedding provider]
    Embed --> Rocks[(RocksDB cache)]
    Embed --> Upsert
    Upsert --> Verify[Verify current version]
    Verify --> Done[Mark indexed in SQLite]
```

## Invariants

- The worker indexes only the current content version of a non-deleted item.
- Core mode completes after transactional chunk/FTS indexing; the `embedding` state is entered only when full mode is enabled.
- The cache key includes content hash, provider, model revision, and dimension.
- Retry/upsert is idempotent; restart requeues jobs that were in progress.
- Exceeding the retry limit moves the job to `failed` with a safe error message.
- Deletion or a version change before completion causes the worker to skip or clean up stale output.

## UI projection

`extracting`, `chunking`, and `embedding` project to the UI state `processing`. `queued`, `indexed`, and `failed` remain unchanged; UI mapping does not change the durable SQLite job state.

## Implementation gap

- The current worker is synchronous and lacks polling, leases, durable checkpoints, restart recovery, and persistent vector/cache adapters.
- Retry counting and stale-version cleanup do not yet use the target schema.

## Owning work

Epic 14 owns the durable worker lifecycle; epic 15 owns optional semantic persistence; epic 18 owns restart/rebuild recovery; epic 19 owns job diagnostics.

## Evidence required

Crash/requeue tests, retry-limit tests, idempotent upsert fixtures, stale-version/deletion races, and configured-full-mode adapter smoke tests.
