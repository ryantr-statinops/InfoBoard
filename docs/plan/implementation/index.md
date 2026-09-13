# Implementation index

This is the review dashboard for the implementation program. Package checklists are authoritative; this file is a rollup and must be updated in the same commit as a package status transition.

## Status legend

`not_started` → `in_progress` → `done`. Use `blocked` when progress cannot continue and record the recovery context in the package execution log.

## Delivery index

| Milestone | Packages | Status | Prerequisites | Exit gate |
| --- | ---: | --- | --- | --- |
| M10 Foundation | 3 | not_started | None | Migration and clean-install evidence |
| M20 Capture | 3 | not_started | M10 | CAP requirements verified |
| M30 Organization and dashboard | 3 | not_started | M10, M20 snapshot projection | ORG and core UX verified |
| M40 Keyword retrieval | 2 | not_started | M20, M30 | Local retrieval verified |
| M50 Semantic provider | 2 | not_started | M20, M40 query contract | Consent and provider smoke verified |
| M60 Vector and hybrid search | 2 | not_started | M40, M50 | Semantic, hybrid, and fallback verified |
| M70 Embedding cache | 1 | not_started | M50 | Cache isolation and recovery verified |
| M80 Analytics | 2 | not_started | M30, M40 | DuckDB/SQLite parity verified |
| M90 Reliability | 3 | not_started | M20–M80 | Security and recovery suites verified |
| M99 Release | 3 | not_started | M90 | All MVP acceptance evidence complete |

## Package catalog

| Order | Package | Milestone | Status |
| ---: | --- | --- | --- |
| 1 | [APP-LIFECYCLE](10-foundation/application-lifecycle/README.md) | M10 | not_started |
| 2 | [DB-MIGRATION](10-foundation/schema-and-migrations/README.md) | M10 | not_started |
| 3 | [API-CANONICAL](10-foundation/canonical-api/README.md) | M10 | not_started |
| 4 | [CAP-URL](20-capture/url-validation-and-normalization/README.md) | M20 | not_started |
| 5 | [CAP-SNAPSHOT](20-capture/capture-and-snapshots/README.md) | M20 | not_started |
| 6 | [CAP-WORKER](20-capture/durable-worker/README.md) | M20 | not_started |
| 7 | [ORG-CONTEXT](30-organization-and-dashboard/collections-tags-and-notes/README.md) | M30 | not_started |
| 8 | [ORG-LEGACY](30-organization-and-dashboard/legacy-items/README.md) | M30 | not_started |
| 9 | [ORG-UI](30-organization-and-dashboard/dashboard-and-ui-states/README.md) | M30 | not_started |
| 10 | [RET-FTS](40-keyword-retrieval/fts-indexing/README.md) | M40 | not_started |
| 11 | [RET-KEYWORD](40-keyword-retrieval/keyword-query/README.md) | M40 | not_started |
| 12 | [SEM-CONFIG](50-semantic-provider/provider-configuration-and-consent/README.md) | M50 | not_started |
| 13 | [SEM-EMBED](50-semantic-provider/embedding-pipeline/README.md) | M50 | not_started |
| 14 | [VEC-CHROMA](60-vector-and-hybrid-search/chroma-index/README.md) | M60 | not_started |
| 15 | [VEC-HYBRID](60-vector-and-hybrid-search/hybrid-retrieval/README.md) | M60 | not_started |
| 16 | [CACHE-ROCKS](70-embedding-cache/rocksdb-cache/README.md) | M70 | not_started |
| 17 | [ANA-PROJECTION](80-analytics/duckdb-projection/README.md) | M80 | not_started |
| 18 | [ANA-API](80-analytics/analytics-api-and-ui/README.md) | M80 | not_started |
| 19 | [REL-HEALTH](90-reliability/health-and-observability/README.md) | M90 | not_started |
| 20 | [REL-RECOVERY](90-reliability/backup-restore-and-rebuild/README.md) | M90 | not_started |
| 21 | [REL-SECURITY](90-reliability/security-hardening/README.md) | M90 | not_started |
| 22 | [RELEASE-TEST](99-release/testing-and-ci/README.md) | M99 | not_started |
| 23 | [RELEASE-PACKAGE](99-release/packaging-and-release/README.md) | M99 | not_started |
| 24 | [RELEASE-COMPAT](99-release/api-compatibility-transition/README.md) | M99 | not_started |

## Review views

- Delivery sequence and gates: [roadmap](00-program/roadmap.md)
- Package dependencies: [dependency map](00-program/dependency-map.md)
- Requirement coverage: [traceability matrix](00-program/traceability.md)
- Program rules: [conventions](00-program/conventions.md)
- Final audit: [review checklist](00-program/review-checklist.md)

## Release gates

| Gate | Status |
| --- | --- |
| All required package tasks are complete and evidenced | pending |
| All milestone acceptance files are satisfied | pending |
| Product requirement coverage has no gaps | pending |
| Canonical and compatibility API behavior passes | pending |
| Backup, restore, rebuild, privacy, and security evidence passes | pending |
| Clean install, upgrade, rollback, UI, performance, and provider smoke evidence passes | pending |
