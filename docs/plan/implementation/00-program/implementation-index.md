# Implementation index

This is the program-level index for delivery status. Product intent remains in the [Internal PRD](../../../product/internal-prd/README.md); this page tracks execution units and evidence.

## Current delivery snapshot

This snapshot describes the current working baseline, not completed delivery:

| Signal | Current value |
| --- | --- |
| Runtime baseline | `partial` |
| Current phase | M1 preparation |
| Verified requirements | none |
| Execution evidence | not started |
| Known gaps | schema migration, interface-contract adoption, dependencies, health checks |

`Delivery` measures implementation progress, `Coverage` measures the runtime baseline,
and the RQ matrix measures requirement evidence. The `execution.md` files remain
append-only evidence logs and are not the status dashboard.

## MVP delivery map

| Milestone | Epic | Package | Product / evidence | Plan | Tasks | References | Execution | Delivery | Coverage |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| M1 | 10 | [Application foundation](../10-m1-local-dashboard/10-application-foundation/README.md) | PR-UX-01, PR-REC-04, PR-OPS-01 / RQ-001, RQ-013, RQ-014 | [plan](../10-m1-local-dashboard/10-application-foundation/plan.md) | [tasks](../10-m1-local-dashboard/10-application-foundation/tasks.md) | [refs](../10-m1-local-dashboard/10-application-foundation/references.md) | [execution](../10-m1-local-dashboard/10-application-foundation/execution.md) | not_started | partial |
| M1 | 11 | [Data and migrations](../10-m1-local-dashboard/11-data-and-migrations/README.md) | PR-CAP-02, PR-CAP-04, PR-ORG-01, PR-ORG-02, PR-ORG-04, PR-REC-01, PR-REC-02, PR-REC-03 / RQ-002, RQ-005, RQ-006 | [plan](../10-m1-local-dashboard/11-data-and-migrations/plan.md) | [tasks](../10-m1-local-dashboard/11-data-and-migrations/tasks.md) | [refs](../10-m1-local-dashboard/11-data-and-migrations/references.md) | [execution](../10-m1-local-dashboard/11-data-and-migrations/execution.md) | not_started | partial |
| M1 | 12 | [Dashboard](../10-m1-local-dashboard/12-dashboard/README.md) | PR-CAP-01, PR-CAP-04, PR-ORG-01, PR-ORG-02, PR-ORG-03, PR-ORG-04, PR-UX-01, PR-UX-02, PR-UX-03, PR-UX-04, PR-REC-04 / RQ-001, RQ-002, RQ-003, RQ-013 | [plan](../10-m1-local-dashboard/12-dashboard/plan.md) | [tasks](../10-m1-local-dashboard/12-dashboard/tasks.md) | [refs](../10-m1-local-dashboard/12-dashboard/references.md) | [execution](../10-m1-local-dashboard/12-dashboard/execution.md) | not_started | partial |
| M2 | 13 | [Ingestion sources](../20-m2-ingestion/13-ingestion-sources/README.md) | PR-CAP-01, PR-CAP-02, PR-CAP-03, PR-SEC-01, PR-SEC-02 / RQ-004, RQ-005, RQ-011 | [plan](../20-m2-ingestion/13-ingestion-sources/plan.md) | [tasks](../20-m2-ingestion/13-ingestion-sources/tasks.md) | [refs](../20-m2-ingestion/13-ingestion-sources/references.md) | [execution](../20-m2-ingestion/13-ingestion-sources/execution.md) | not_started | partial |
| M2 | 14 | [Indexing worker](../20-m2-ingestion/14-indexing-worker/README.md) | PR-CAP-05, PR-REC-01, PR-REC-02, PR-REC-03 / RQ-006 | [plan](../20-m2-ingestion/14-indexing-worker/plan.md) | [tasks](../20-m2-ingestion/14-indexing-worker/tasks.md) | [refs](../20-m2-ingestion/14-indexing-worker/references.md) | [execution](../20-m2-ingestion/14-indexing-worker/execution.md) | not_started | partial |
| M3 | 15 | [Search and discovery](../30-m3-search-and-insights/15-search-and-discovery/README.md) | PR-RET-01, PR-RET-02, PR-RET-03, PR-RET-04, PR-RET-05 / RQ-007, conditional RQ-008, conditional RQ-009 | [plan](../30-m3-search-and-insights/15-search-and-discovery/plan.md) | [tasks](../30-m3-search-and-insights/15-search-and-discovery/tasks.md) | [refs](../30-m3-search-and-insights/15-search-and-discovery/references.md) | [execution](../30-m3-search-and-insights/15-search-and-discovery/execution.md) | not_started | partial |
| M3 | 16 | [Analytics](../30-m3-search-and-insights/16-analytics/README.md) | PR-RET-01, PR-RET-06, PR-REC-03, PR-REC-04 / RQ-010 | [plan](../30-m3-search-and-insights/16-analytics/plan.md) | [tasks](../30-m3-search-and-insights/16-analytics/tasks.md) | [refs](../30-m3-search-and-insights/16-analytics/references.md) | [execution](../30-m3-search-and-insights/16-analytics/execution.md) | not_started | partial |
| M4 | 17 | [Security](../40-m4-release/17-security/README.md) | PR-SEC-01, PR-SEC-02, PR-SEC-03 / RQ-011 | [plan](../40-m4-release/17-security/plan.md) | [tasks](../40-m4-release/17-security/tasks.md) | [refs](../40-m4-release/17-security/references.md) | [execution](../40-m4-release/17-security/execution.md) | not_started | partial |
| M4 | 18 | [Reliability](../40-m4-release/18-reliability/README.md) | PR-REC-01, PR-REC-02, PR-REC-03, PR-REC-04, PR-REC-05 / RQ-012, RQ-013 | [plan](../40-m4-release/18-reliability/plan.md) | [tasks](../40-m4-release/18-reliability/tasks.md) | [refs](../40-m4-release/18-reliability/references.md) | [execution](../40-m4-release/18-reliability/execution.md) | not_started | missing |
| M4 | 19 | [Observability](../40-m4-release/19-observability/README.md) | PR-REC-04 / RQ-013 | [plan](../40-m4-release/19-observability/plan.md) | [tasks](../40-m4-release/19-observability/tasks.md) | [refs](../40-m4-release/19-observability/references.md) | [execution](../40-m4-release/19-observability/execution.md) | not_started | partial |
| M4 | 20 | [Testing and CI](../40-m4-release/20-testing-and-ci/README.md) | PR-OPS-02 / RQ-001, RQ-002, RQ-004, RQ-005, RQ-006, RQ-007, conditional RQ-008, conditional RQ-009, RQ-010, RQ-011, RQ-012, RQ-014 | [plan](../40-m4-release/20-testing-and-ci/plan.md) | [tasks](../40-m4-release/20-testing-and-ci/tasks.md) | [refs](../40-m4-release/20-testing-and-ci/references.md) | [execution](../40-m4-release/20-testing-and-ci/execution.md) | not_started | missing |
| M4 | 21 | [Packaging](../40-m4-release/21-packaging/README.md) | PR-OPS-01, PR-OPS-02 / RQ-014 | [plan](../40-m4-release/21-packaging/plan.md) | [tasks](../40-m4-release/21-packaging/tasks.md) | [refs](../40-m4-release/21-packaging/references.md) | [execution](../40-m4-release/21-packaging/execution.md) | not_started | missing |

The package links become active as each milestone commit is applied. Status values above are the baseline to audit, not evidence of implementation completion.

## Discovery map

| ID | Package | Product source | Status |
| --- | --- | --- | --- |
| 30 | Browser extension | [Next Plan](../../../product/next-plan/01-browser-extension.md) | discovery |
| 31 | Cloud sync | [Next Plan](../../../product/next-plan/03-cloud-sync.md) | discovery |
| 32 | Collaboration | [Next Plan](../../../product/next-plan/03-cloud-sync.md) | discovery, dependent on 31 |
| 33 | AI and providers | [Next Plan](../../../product/next-plan/04-ai-and-providers.md) | discovery |
| 34 | Browser portability | [Next Plan](../../../product/next-plan/00-browser-portability.md) | discovery |
| 35 | Migration assistant | [Next Plan](../../../product/next-plan/02-migration-assistant.md) | discovery |
