# Examples — Observability and diagnostics

## Target contract: health response

```json
{
  "status": "degraded",
  "http_status": 200,
  "components": {
    "sqlite": "ok",
    "fts": "ok",
    "semantic": "unavailable"
  },
  "action": "continue_keyword_search"
}
```

If SQLite or FTS is unavailable, the response must instead use HTTP `503 unavailable`.
