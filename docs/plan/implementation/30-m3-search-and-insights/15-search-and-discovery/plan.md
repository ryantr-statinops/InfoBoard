# 15 — Search and discovery

**Plan status:** `ready`  
**Delivery status:** `not_started`  
**Baseline coverage:** `partial`  
**Milestone:** M3  
**Dependencies:** 11, 12, 14

## Outcome

Users can find saved items by keyword in core mode and by semantic similarity in full mode, with stable source provenance and safe fallbacks.

## Retrieval pipeline

```mermaid
flowchart LR
    Query --> Filters[Shared filters]
    Filters --> FTS[SQLite FTS5]
    Filters --> Semantic[Optional semantic search]
    FTS --> RRF[RRF fusion]
    Semantic --> RRF
    RRF --> Hydrate[SQLite hydration]
    Hydrate --> Results[Source, score, excerpt]
```

- FTS query input is sanitized and always available in core mode.
- Semantic search is optional and never becomes a hard dependency for keyword results.
- Hybrid retrieval uses RRF and hydrates final records from SQLite.
- Related items and clusters include provenance and explicit empty/degraded states.
- Shared filters have identical semantics for search and analytics.

## Acceptance and evaluation

- Vietnamese/English query fixtures meet the agreed top-five retrieval target.
- FTS returns safe excerpts/highlights without leaking deleted content.
- Semantic/provider failure preserves keyword results and reports degraded mode.
- Related/cluster output can be traced to source items.
- Warm and cold latency, recall, and fallback results are recorded in the evaluation report.

## Review gate

Review query sanitization, ranking behavior, deleted-item filtering, provider absence, and benchmark reproducibility before enabling full mode.
