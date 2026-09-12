# Examples — Analytics and insights

## Target contract: mode-aware KPI response

```json
{
  "mode": "sqlite_fallback",
  "degraded": true,
  "reason": "duckdb_unavailable",
  "kpis": {"total_items": 12, "indexed_items": 10}
}
```

The response distinguishes a correct fallback from a healthy full-mode result and is verified by [T16-003](tasks.md#t16-003).
