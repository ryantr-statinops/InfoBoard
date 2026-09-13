# FTS5 indexing and rebuild: implementation plan

## Approach

Maintain a derived FTS5 index for only active bookmarks and current snapshot versions, with deterministic cleanup and rebuild. Work in `app/db.py`, FTS repository/index jobs, rebuild maintenance, integration tests, retaining existing behavior only when target contract tests prove compatibility.

## Delivery flow

1. Add fixtures and safeguards for current data and callers.
2. Introduce the target service/repository or UI boundary.
3. Move reads and writes to the target behavior.
4. Verify concurrency, failure, deletion, and recovery paths.
5. Record exact evidence and update rollups only after acceptance.

## Invariants

Canonical data and user context survive recapture, reindex, and rebuild. Deleted or stale data never reaches public results. UI and adapters call the same application services as the canonical API.

## Rollout

Complete CAP-SNAPSHOT, CAP-WORKER, ORG-CONTEXT, execute [tasks](tasks.md) in order, and satisfy the milestone acceptance table. Stop for an upstream decision if implementation exposes a contract gap.
