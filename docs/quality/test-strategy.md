# Test strategy

## Layers

| Layer | Required coverage |
| --- | --- |
| Domain unit | URL normalization, deduplication, state transitions, filters, chunk identity, ranking, consent, and retry rules. |
| SQLite integration | Integer-ID migration, legacy mapping, foreign keys, organization relations, attempt/snapshot invariants, versions/leases, FTS rebuild, and soft delete. |
| Capture integration | Public fetch fixtures, redirects, DNS/address checks, limits, extraction, atomic files, restart, and retry. |
| Derived storage integration | RocksDB cache compatibility/corruption, Chroma upsert/query/revision isolation, and DuckDB projection/fallback. |
| Provider contract | OpenAI-compatible request/response validation using a controlled fake plus a configured real endpoint smoke test. |
| HTTP contract | All `/api/v1` schemas, limits, pagination, filters, `If-Match`, stable errors, API-key absence/presence, component states, and one-release adapter parity. |
| Security | SSRF/rebinding/redirect, Host/Origin/CSRF, XSS, resource limits, secret/content redaction, consent, revocation, and archive safety. |
| Recovery | Backup, checksum, restore, migration, rebuild order, interrupted rebuild, and canonical preservation. |
| UI | Capture-organize-retrieve, failed snapshot retry, semantic setup/degradation, keyboard/focus, and 1440/390 px layouts. |
| Evaluation | Versioned keyword/semantic/hybrid relevance set and reproducible performance benchmark. |

## Fixtures and isolation

- Tests never use the default/user data directory.
- Storage tests use explicit temporary paths for SQLite, RocksDB, ChromaDB, DuckDB, and snapshots.
- Provider unit/integration tests use a deterministic local fake and assert that no request occurs before consent.
- Real compatible-provider smoke tests use dedicated credentials and non-sensitive fixture text.
- Fixtures include normalization-v1 tracker/query cases, unsafe destinations, redirects, migrated legacy text/files, changed/deleted snapshots, Unicode/English text, stale versions, collection/tag combinations, provider dimension mismatch, and stale analytics checkpoints.

## Retrieval evaluation

- Maintain a versioned English/Unicode query set with expected bookmark IDs and filters.
- Score keyword, semantic, and hybrid independently; record provider/model/revision, fixture checksum, and fusion constants.
- MVP target: expected bookmark appears in top five for at least 80% of the published query set in each claimed mode.
- Verify deterministic hybrid ordering for identical candidates/configuration and exclusion of deleted/stale versions.

## Performance baseline

Use 1,000 bookmarks and approximately 10,000 chunks. Record hardware, OS, dependency versions, provider/network conditions, warm-up, samples, and p50/p95 for capture queueing, list/filter, keyword search, semantic/hybrid search, analytics, and rebuild. The MVP target is p95 below one second for warmed keyword and semantic/hybrid search on the published reference environment; external provider latency is reported separately.

## CI and release evidence

Run dependency validation, lint/type checks, deterministic unit/integration/HTTP/security suites, recovery tests, UI smoke, and benchmarks. A real provider smoke and semantic evaluation are mandatory for MVP release even though deterministic CI uses a fake. No test result may imply that a target-only component is implemented.
