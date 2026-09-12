# Quality — Test strategy

## Layers

- Unit: normalization, hashing, chunking, ranking, validation, and domain rules.
- Storage integration: migrations, foreign keys, FTS sync, and version/delete semantics.
- Service integration: ingestion, worker retry/restart, fake derived adapters, and rebuild.
- HTTP integration: API contracts, errors, pagination, partial rendering, and escaping.
- Security: URL/redirect SSRF, limits, Host/Origin, unsafe markup, and log redaction.
- Recovery: backup copy, restore, migrate, rebuild, and integrity verification.
- UI smoke: add → detail → note/collection/status → search on desktop/mobile.
- Full-mode smoke: real dependency/model checks separate from the deterministic core suite.

## Fixture policy

- Tests do not use the default database or user data.
- Real databases are created in a temporary directory for migration/recovery tests.
- Unit tests use fake embedding/providers; the full model is limited to smoke/evaluation suites.
- Fixtures cover Vietnamese/English, accented/unaccented text, duplicates, empty PDFs, redirects, and stale versions.

## CI stages

1. Dependency/install validation.
2. Ruff and type checks.
3. Core unit/integration/HTTP tests.
4. Security and migration/recovery tests.
5. Optional full-mode smoke.
6. Benchmark/evaluation according to the release gate, not hidden inside the core unit suite.
