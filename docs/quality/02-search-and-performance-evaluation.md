# Quality — Search and performance evaluation

## Retrieval evaluation

- Use a versioned query set of at least 20 Vietnamese/English queries, including synonyms and accented/unaccented forms.
- Each query declares its target item and applied filters.
- Evaluate core and full modes separately.
- Core release target: keyword search places the target item in the top five for at least 16/20 queries.
- Full-mode target: when full mode is in release scope, semantic/hybrid retrieval separately places the target item in the top five for at least 16/20 queries; this evidence is required for the full-mode claim, not for a core-only release.
- Record retrieval mode, model ID/revision, index version, and fixture checksum.

## Performance benchmark

- Standard dataset: 1,000 items and approximately 10,000 chunks.
- Measure search p50/p95, dashboard aggregate latency, ingestion throughput, and rebuild duration.
- Core keyword-search target is p95 below one second without requiring a model.
- When full mode is in release scope, semantic/hybrid search has a separate p95-below-one-second target after model warm-up; exclude download/startup.
- Record CPU, RAM, OS, Python/dependency versions, model, and warm-up/sample counts.
- Do not treat the target as a guarantee for all hardware; explain regressions against the same-environment baseline.

## Failure evaluation

- Chroma unavailable: the keyword response still succeeds and reports its mode.
- DuckDB not configured: use the bounded SQLite fallback as normal core behavior. If configured DuckDB becomes unavailable, preserve the fallback and show truthful degraded metadata.
- Model/cache mismatch: do not use vectors/cache with the wrong revision/dimension.
- Deleted/stale chunks: do not appear in results.

The Chroma failure scenario starts from an explicitly enabled full-mode configuration. A core-only configuration with no Chroma/model is healthy and does not emit degraded status.
