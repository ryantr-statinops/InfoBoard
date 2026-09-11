# 19 — Observability và diagnostics

**Status:** `draft`
**Canonical references:** [Quality attributes](../../../../quality/00-quality-attributes.md) · [Health/troubleshooting](../../../../operations/02-health-and-troubleshooting.md) · [Security boundaries](../../../../architecture/07-security-boundaries.md)
**Milestone:** M4
**Dependencies:** 10, 14, 15, 16, 17

## Outcome

Có đủ thông tin để chẩn đoán startup, ingestion, search và derived dependency mà không ghi nội dung người dùng hoặc secret.

## Event schema

Structured event gồm `timestamp`, `level`, `component`, `event`, `request_id`/`job_id`, `duration_ms`, `state`, `error_code`, `retry_count`, `cache_hit` và `degraded`. Content/title/query chỉ được log dạng length/hash nếu cần.

## Health model

`/api/health` trả status tổng và component: sqlite, fts, worker, chroma, rocksdb, duckdb, model. `ok`, `degraded`, `unavailable`; SQLite hoặc FTS5 unavailable trả HTTP 503, còn derived dependency unavailable trả HTTP 200 với trạng thái degraded.

## Diagnostics

CLI `diagnose` tạo bundle metadata (versions, schema, health, recent error codes, timings) không bao gồm DB content/secret. UI hiển thị reason và action retry/rebuild phù hợp.

## Commit slices

1. `feat: add structured application and job events`
2. `feat: expand component health checks`
3. `feat: add local diagnostics bundle`
4. `test: verify redaction health and degraded states`

## Acceptance

Worker retry có job ID và duration; semantic unavailable hiện degraded nhưng keyword usable; health phản ánh đúng component; diagnostics không chứa content/secret khi scan.

## Review gate

Capture log trong happy path/error path, assert forbidden fields không xuất hiện và verify health status bằng dependency giả lập lỗi.

## Execution log

Hiện health chỉ trả `{"status":"ok"}`; cần mở rộng theo contract.
