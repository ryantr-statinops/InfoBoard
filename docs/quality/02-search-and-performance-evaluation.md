# Quality — Search and performance evaluation

## Retrieval evaluation

- Use a versioned query set of at least 20 Vietnamese/English queries, including synonyms and accented/unaccented forms.
- Each query declares its target item and applied filters.
- Evaluate core and full modes separately.
- Core release target: keyword search places the target item in the top five for at least 16/20 queries.
- Full-mode target: semantic/hybrid retrieval is evaluated only when full mode is in release scope; its evidence is required for the full-mode claim, not for a core-only release.
- Record retrieval mode, model ID/revision, index version, and fixture checksum.

## Performance benchmark

- Standard dataset: 1,000 items and approximately 10,000 chunks.
- Measure search p50/p95, dashboard aggregate latency, ingestion throughput, and rebuild duration.
- Search target is p95 below one second after the model is loaded; exclude cold model download/startup.
- Record CPU, RAM, OS, Python/dependency versions, model, and warm-up/sample counts.
- Do not treat the target as a guarantee for all hardware; explain regressions against the same-environment baseline.

## Failure evaluation

- Chroma unavailable: the keyword response still succeeds and reports its mode.
- DuckDB unavailable: use the bounded SQLite fallback or a clear degraded card.
- Model/cache mismatch: do not use vectors/cache with the wrong revision/dimension.
- Deleted/stale chunks: do not appear in results.
