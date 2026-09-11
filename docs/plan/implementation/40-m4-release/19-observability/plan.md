# 19 — Observability and diagnostics

**Plan status:** `ready`  
**Delivery status:** `not_started`  
**Baseline coverage:** `partial`  
**Milestone:** M4  
**Dependencies:** 10, 14, 15, 16, 17

## Outcome

Operators can diagnose startup, ingestion, search, and derived dependencies without exposing user content or secrets.

## Event and health contracts

Structured events contain `timestamp`, `level`, `component`, `event`, `request_id`/`job_id`, `duration_ms`, `state`, `error_code`, `retry_count`, `cache_hit`, and `degraded`. Content/title/query values are represented only by length/hash when required.

`/api/health` reports SQLite, FTS, worker, Chroma, RocksDB, DuckDB, and model components. SQLite or FTS5 unavailable returns HTTP `503 unavailable`; derived dependency failure returns HTTP `200 degraded`. List/read may continue during FTS failure, but core mode is not advertised as healthy and the UI points to repair/rebuild.

The UI maps `queued` to queued, `extracting | chunking | embedding` to processing, `indexed` to indexed, and `failed` to failed.

## Diagnostics and acceptance

The target `diagnose` command produces versions, schema, health, recent error codes, and timings without database content or secrets. Worker retry events include job ID and duration. A semantic failure is visibly degraded while keyword search remains usable.

## Review gate

Capture happy/error logs, assert forbidden fields are absent, and simulate every component failure in health tests.
