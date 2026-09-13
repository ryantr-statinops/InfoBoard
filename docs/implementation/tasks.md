# Implementation tasks

All tasks start `not_started`. Links are canonical inputs and acceptance sources; completion evidence must later record commit, command/scenario, environment, and result.

| ID | Status | Delivery unit | Requirements | Canonical contracts | Verification | Depends on |
| --- | --- | --- | --- | --- | --- | --- |
| `T01` | not_started | Central settings, paths, lifespan, versioned migration runner, SQLite repositories, error envelope, base health | `REL-01`, `UX-02` | [System](../architecture/system-overview.md), [data](../architecture/data-model.md), [API](../architecture/api-contracts.md) | Clean install/migrate/restart/health integration | — |
| `T02` | not_started | Migrate existing runtime data into bookmark/snapshot/organization/job schema without loss | `REL-01`, `REL-02` | [Data](../architecture/data-model.md), [recovery](../architecture/lifecycle-and-recovery.md) | Upgrade fixture and rollback-to-backup evidence | `T01` |
| `T03` | not_started | URL parse, normalization, duplicate policy, public-address and redirect validation | `CAP-01`, `CAP-04`, `PRI-01` | [Capture](../architecture/capture-pipeline.md), [security](../architecture/security-and-privacy.md) | Unit plus SSRF/rebinding/redirect suite | `T01` |
| `T04` | not_started | Bookmark-first capture API, durable worker lease, bounded fetch/extraction, atomic managed snapshot | `CAP-01`–`CAP-03`, `REL-02` | [Capture feature](../product/features/bookmark-capture.md), [pipeline](../architecture/capture-pipeline.md) | HTTP/capture/crash/restart fixtures | `T02`, `T03` |
| `T05` | not_started | Recapture, retry, content-version advancement, downstream job creation | `CAP-02`, `CAP-03`, `ORG-03` | [Data](../architecture/data-model.md), [lifecycle](../architecture/lifecycle-and-recovery.md) | Failed/current/new-version idempotency tests | `T04` |
| `T06` | not_started | Collection, tag, note, status and soft-delete services/APIs | `ORG-01`–`ORG-04` | [Organization](../product/features/organization.md), [API](../architecture/api-contracts.md) | Storage and HTTP CRUD/delete-leak tests | `T02` |
| `T07` | not_started | Library/inbox/detail UI, capture flow, shared filters, context preservation, responsive states | `UX-01`–`UX-03` | [Design](../design/README.md), [UI states](../design/ui-states.md) | Keyboard plus 1440/390 px browser smoke | `T04`, `T06` |
| `T08` | not_started | Current-version SQLite FTS indexing, query sanitation, excerpts, filters, rebuild | `RET-01`, `RET-02`, `RET-04` | [Index/search](../architecture/indexing-and-search.md), [retrieval](../product/features/retrieval.md) | Storage/HTTP/rebuild and keyword evaluation | `T05`, `T06` |
| `T09` | not_started | Secret boundary, provider configuration/verification, versioned disclosure, consent/revocation | `PRI-01`, `PRI-02`, `RET-03` | [Security](../architecture/security-and-privacy.md), [provider API](../architecture/api-contracts.md) | No-transfer, redaction, replacement and revocation tests | `T01` |
| `T10` | not_started | Compatible embedding client, validation, batching, index revisions and semantic job lifecycle | `RET-03`, `REL-02` | [Index/search](../architecture/indexing-and-search.md), [data](../architecture/data-model.md) | Fake-provider contract plus real-provider smoke | `T05`, `T09` |
| `T11` | not_started | Revisioned ChromaDB upsert/query, activation, cleanup and canonical candidate validation | `RET-03`–`RET-05`, `ORG-04` | [Storage](../architecture/storage-architecture.md), [index/search](../architecture/indexing-and-search.md) | Dimension/version/deletion/outage/rebuild tests | `T08`, `T10` |
| `T12` | not_started | Hybrid orchestration, deterministic rank fusion, actual-mode/degradation response | `RET-01`–`RET-05` | [Retrieval](../product/features/retrieval.md), [search API](../architecture/api-contracts.md) | Keyword/semantic/hybrid evaluation and fallback | `T08`, `T11` |
| `T13` | not_started | RocksDB document/query embedding cache, compatibility keys, bypass and cleanup | `RET-03`, `RET-05`, `REL-01` | [Storage](../architecture/storage-architecture.md), [index/search](../architecture/indexing-and-search.md) | Hit/miss/corruption/revision/revocation tests | `T10` |
| `T14` | not_started | DuckDB projection, checkpoint/refresh, accepted metrics and bounded SQLite fallback | `ANA-01`, `ANA-02` | [Analytics feature](../product/features/analytics.md), [pipeline](../architecture/analytics-pipeline.md) | DuckDB/SQLite parity and stale/interruption tests | `T06`, `T08` |
| `T15` | not_started | Component health, safe diagnostics and durable maintenance/rebuild API | `UX-02`, `PRI-02`, `REL-01`, `REL-02` | [API](../architecture/api-contracts.md), [operations](../operations/README.md) | Failure matrix, redaction and target-path tests | `T11`, `T13`, `T14` |
| `T16` | not_started | Canonical backup, manifest/checksum, safe restore, migration and ordered derived rebuild | `REL-01`, `REL-02` | [Recovery](../architecture/lifecycle-and-recovery.md), [runbook](../operations/backup-restore-and-rebuild.md) | Full backup/restore/rebuild transcript | `T02`, `T15` |
| `T17` | not_started | Complete CI, security suite, UI smoke, relevance evaluation and performance benchmark | All | [Test strategy](../quality/test-strategy.md), [acceptance](../quality/mvp-acceptance.md) | Clean reproducible evidence bundle | `T03`–`T16` |
| `T18` | not_started | Packaging, clean setup, upgrade/rollback documentation and MVP release review | All | [Configuration](../operations/configuration.md), [acceptance](../quality/mvp-acceptance.md) | Clean checkout/install/upgrade/rollback and final traceability | `T17` |

## Task discipline

- Implement one vertical contract at a time and keep schema/API compatibility explicit.
- Do not mark a task verified from target docs or mock-only behavior.
- Update status only with implementation evidence; do not create empty per-task execution files.
- If a task needs a feature outside [MVP scope](../product/mvp-scope.md), submit it through [the idea inbox](../product/ideas.md) instead of expanding the task.
