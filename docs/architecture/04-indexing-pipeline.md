# Architecture — Indexing pipeline

## Current state

Worker lifecycle hiện là xử lý đồng bộ mô phỏng; chưa có background polling, durable lease, embedding thật hoặc persistent Chroma integration.

## Target state

```mermaid
stateDiagram-v2
    [*] --> queued
    queued --> extracting
    extracting --> chunking
    chunking --> embedding
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

- Worker chỉ index current content version của non-deleted item.
- Cache key gồm content hash, provider, model revision và dimension.
- Retry/upsert idempotent; restart requeue job đang xử lý.
- Quá retry limit chuyển `failed` và giữ error message an toàn.
- Xóa hoặc đổi version trước khi hoàn tất khiến worker skip/cleanup stale output.
