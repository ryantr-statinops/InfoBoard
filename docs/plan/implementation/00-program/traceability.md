# Requirement traceability

This matrix assigns delivery ownership. Detailed feature, architecture, quality, task, and evidence links live in each package reference file.

| Requirement | Primary packages |
| --- | --- |
| CAP-01 | [CAP-URL](../20-capture/url-validation-and-normalization/README.md), [CAP-SNAPSHOT](../20-capture/capture-and-snapshots/README.md) |
| CAP-02 | [CAP-SNAPSHOT](../20-capture/capture-and-snapshots/README.md), [CAP-WORKER](../20-capture/durable-worker/README.md) |
| CAP-03 | [CAP-SNAPSHOT](../20-capture/capture-and-snapshots/README.md), [CAP-WORKER](../20-capture/durable-worker/README.md) |
| CAP-04 | [CAP-URL](../20-capture/url-validation-and-normalization/README.md) |
| CAP-05 | [DB-MIGRATION](../10-foundation/schema-and-migrations/README.md), [ORG-LEGACY](../30-organization-and-dashboard/legacy-items/README.md) |
| ORG-01 | [ORG-CONTEXT](../30-organization-and-dashboard/collections-tags-and-notes/README.md) |
| ORG-02 | [ORG-CONTEXT](../30-organization-and-dashboard/collections-tags-and-notes/README.md), [ORG-UI](../30-organization-and-dashboard/dashboard-and-ui-states/README.md) |
| ORG-03 | [ORG-CONTEXT](../30-organization-and-dashboard/collections-tags-and-notes/README.md), [REL-RECOVERY](../90-reliability/backup-restore-and-rebuild/README.md) |
| ORG-04 | [API-CANONICAL](../10-foundation/canonical-api/README.md), [RET-FTS](../40-keyword-retrieval/fts-indexing/README.md), [VEC-CHROMA](../60-vector-and-hybrid-search/chroma-index/README.md), [ANA-PROJECTION](../80-analytics/duckdb-projection/README.md) |
| RET-01 | [RET-KEYWORD](../40-keyword-retrieval/keyword-query/README.md), [VEC-HYBRID](../60-vector-and-hybrid-search/hybrid-retrieval/README.md), [ANA-API](../80-analytics/analytics-api-and-ui/README.md) |
| RET-02 | [RET-FTS](../40-keyword-retrieval/fts-indexing/README.md), [RET-KEYWORD](../40-keyword-retrieval/keyword-query/README.md) |
| RET-03 | [SEM-CONFIG](../50-semantic-provider/provider-configuration-and-consent/README.md), [SEM-EMBED](../50-semantic-provider/embedding-pipeline/README.md), [VEC-CHROMA](../60-vector-and-hybrid-search/chroma-index/README.md), [VEC-HYBRID](../60-vector-and-hybrid-search/hybrid-retrieval/README.md) |
| RET-04 | [RET-FTS](../40-keyword-retrieval/fts-indexing/README.md), [RET-KEYWORD](../40-keyword-retrieval/keyword-query/README.md), [VEC-CHROMA](../60-vector-and-hybrid-search/chroma-index/README.md), [VEC-HYBRID](../60-vector-and-hybrid-search/hybrid-retrieval/README.md) |
| RET-05 | [RET-KEYWORD](../40-keyword-retrieval/keyword-query/README.md), [VEC-HYBRID](../60-vector-and-hybrid-search/hybrid-retrieval/README.md), [REL-HEALTH](../90-reliability/health-and-observability/README.md) |
| ANA-01 | [ANA-PROJECTION](../80-analytics/duckdb-projection/README.md), [ANA-API](../80-analytics/analytics-api-and-ui/README.md) |
| ANA-02 | [ANA-PROJECTION](../80-analytics/duckdb-projection/README.md), [ANA-API](../80-analytics/analytics-api-and-ui/README.md) |
| PRI-01 | [SEM-CONFIG](../50-semantic-provider/provider-configuration-and-consent/README.md), [REL-SECURITY](../90-reliability/security-hardening/README.md) |
| PRI-02 | [API-CANONICAL](../10-foundation/canonical-api/README.md), [SEM-CONFIG](../50-semantic-provider/provider-configuration-and-consent/README.md), [REL-HEALTH](../90-reliability/health-and-observability/README.md), [REL-SECURITY](../90-reliability/security-hardening/README.md) |
| REL-01 | [DB-MIGRATION](../10-foundation/schema-and-migrations/README.md), [RET-FTS](../40-keyword-retrieval/fts-indexing/README.md), [VEC-CHROMA](../60-vector-and-hybrid-search/chroma-index/README.md), [CACHE-ROCKS](../70-embedding-cache/rocksdb-cache/README.md), [ANA-PROJECTION](../80-analytics/duckdb-projection/README.md), [REL-RECOVERY](../90-reliability/backup-restore-and-rebuild/README.md) |
| REL-02 | [CAP-WORKER](../20-capture/durable-worker/README.md), [REL-RECOVERY](../90-reliability/backup-restore-and-rebuild/README.md) |
| UX-01 | [ORG-UI](../30-organization-and-dashboard/dashboard-and-ui-states/README.md), [RET-KEYWORD](../40-keyword-retrieval/keyword-query/README.md), [VEC-HYBRID](../60-vector-and-hybrid-search/hybrid-retrieval/README.md) |
| UX-02 | [ORG-UI](../30-organization-and-dashboard/dashboard-and-ui-states/README.md), [REL-HEALTH](../90-reliability/health-and-observability/README.md) |
| UX-03 | [ORG-UI](../30-organization-and-dashboard/dashboard-and-ui-states/README.md), [RELEASE-TEST](../99-release/testing-and-ci/README.md) |
| UX-04 | [ORG-UI](../30-organization-and-dashboard/dashboard-and-ui-states/README.md), [RELEASE-TEST](../99-release/testing-and-ci/README.md) |

## Coverage rule

A requirement is covered only when its package reference file links the requirement, feature behavior, architecture/API/schema contract, quality scenario, task IDs, and execution anchors. Package assignment alone is not completion.
