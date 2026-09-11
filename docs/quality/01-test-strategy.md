# Quality — Test strategy

## Layers

- Unit: normalization, hashing, chunking, ranking, validation và domain rules.
- Storage integration: migrations, foreign keys, FTS sync, version/delete semantics.
- Service integration: ingestion, worker retry/restart, fake derived adapters và rebuild.
- HTTP integration: API contracts, errors, pagination, partial rendering và escaping.
- Security: URL/redirect SSRF, limits, Host/Origin, unsafe markup và log redaction.
- Recovery: backup copy, restore, migrate, rebuild và integrity verification.
- UI smoke: add → detail → note/collection/status → search trên desktop/mobile.
- Full-mode smoke: real dependency/model checks tách khỏi deterministic core suite.

## Fixture policy

- Test không dùng database mặc định hoặc dữ liệu người dùng.
- Database thật được tạo trong temporary directory cho migration/recovery tests.
- Unit tests dùng fake embedding/provider; full model chỉ ở smoke/evaluation suite.
- Fixtures bao phủ Việt/Anh, có/không dấu, duplicates, empty PDF, redirects và stale versions.

## CI stages

1. Dependency/install validation.
2. Ruff và type checks.
3. Core unit/integration/HTTP tests.
4. Security và migration/recovery tests.
5. Optional full-mode smoke.
6. Benchmark/evaluation theo release gate, không chạy ngầm trong core unit suite.
