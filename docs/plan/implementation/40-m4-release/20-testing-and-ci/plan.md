# 20 — Testing, performance, and CI

**Plan status:** `ready`  
**Delivery status:** `not_started`  
**Baseline coverage:** `partial`  
**Milestone:** M4  
**Dependencies:** 10–19  
**Quality commands:** `uv run pytest -q`, `uv run ruff check .`

## Test layers

Unit tests cover normalization, chunking, hashing, extractors, FTS sanitization, RRF, and filters. Integration tests cover migrations, FTS, fake derived adapters, and worker recovery. HTTP tests use `httpx`/ASGI lifespan with isolated temporary databases. UI tests cover status, fragments, escaping, and empty/loading/error states. Security tests cover SSRF, limits, Origin/Host, XSS, and redaction. Full smoke tests use real optional dependencies only when available.

## Determinism and benchmark

Fixtures accept `tmp_path`, patch settings before opening connections, and never seed the real `data/` directory. Fake embeddings are deterministic by hash; network/PDF parsing uses fixtures or a local server.

Benchmark data contains 1,000 items/10,000 chunks. Warm the model before measurement and report p50/p95/p99, memory, CPU, model/version, and hardware. Target search p95 is under one second after warm-up; model startup is excluded.

## CI and acceptance

CI runs format/lint/type checks, core unit/HTTP tests, migration/recovery/security suites, and optional full-mode smoke. Benchmarks run manually/nightly and remain comparable across hardware.

- No test uses the default database.
- Full suite is repeatable.
- CI fails on migration, lint, or core-test failure.
- Traceability and benchmark evidence are reviewable before release.
