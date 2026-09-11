# Examples — Search and discovery

## Target contract: hybrid retrieval

```text
filters → FTS5 + optional semantic → RRF → SQLite hydration → source/score/excerpt
```

The final record is hydrated from SQLite even when a derived store returns the ranking signal.

## Target contract: degraded semantic search

```json
{
  "mode": "keyword",
  "degraded": true,
  "reason": "semantic_store_unavailable"
}
```

This preserves core-mode search and must be covered by [T15-003](tasks.md#t15-003).
