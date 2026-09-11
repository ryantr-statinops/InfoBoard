# 20 — Testing, performance và CI

**Status:** `partial`  
**Milestone:** M4  
**Dependencies:** 10–19  
**Quality commands:** `uv run pytest -q`, `uv run ruff check .`

## Test layers

- Unit: normalize, chunk, hash, extractors, FTS sanitize, RRF, filters.
- Integration: SQLite migrations, FTS, fake Chroma/RocksDB/DuckDB adapters, worker recovery.
- HTTP: `httpx`/ASGI transport với lifespan đúng; mỗi test có DB temp riêng.
- UI/template: status 200, partial context, escaping, empty/loading/error states.
- Security: SSRF, size limits, Origin/Host, XSS/log redaction.
- Full smoke: optional dependencies/model thật, không chạy mặc định trong unit suite.

## Fixtures và determinism

Fixture factory nhận `tmp_path`, monkeypatch settings trước import connection; seed dataset không dùng `data/` thật. Embedding fake deterministic theo hash; network/PDF parser dùng fixture/local server.

## Benchmark

Generate 1.000 items/10.000 chunks, warm model trước đo; ghi p50/p95/p99, memory, CPU, model/version và hardware. Mục tiêu search p95 < 1s sau warm-up; không tính model startup.

## CI stages

1. Format/lint/type checks.
2. Core unit + HTTP integration.
3. Migration/recovery/security.
4. Optional full extra smoke khi runner đủ native wheels/model cache.
5. Benchmark manual/nightly, không block PR nếu hardware khác nhưng phải lưu kết quả.

## Commit slices

1. `test: add isolated sqlite and http fixtures`
2. `test: cover complete item ingestion and recovery api`
3. `test: add multilingual and security suites`
4. `test: add benchmark dataset and p95 report`
5. `ci: run quality gates on pull requests`

## Acceptance

Không test nào dùng DB mặc định; full suite repeatable; tất cả quality commands pass; benchmark report có cấu hình; CI fail khi migration, lint hoặc core test fail.

## Review gate

Review test coverage map trong `92-requirement-traceability.md`, inspect flaky test policy và chạy local command trước PR.

## Execution log

Hiện có 4 test unit/service và Ruff pass; chưa có HTTP integration/CI/benchmark.
