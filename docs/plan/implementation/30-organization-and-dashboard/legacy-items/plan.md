# Legacy item preservation: implementation plan

## Approach

Expose migrated text and file items as readable, searchable, organizable, soft-deletable legacy bookmarks without new writes or recapture. Work in migration projections, bookmark services/API, legacy fixtures, UI state tests, retaining existing behavior only when target contract tests prove compatibility.

## Delivery flow

1. Add fixtures and safeguards for current data and callers.
2. Introduce the target service/repository or UI boundary.
3. Move reads and writes to the target behavior.
4. Verify concurrency, failure, deletion, and recovery paths.
5. Record exact evidence and update rollups only after acceptance.

## Invariants

Canonical data and user context survive recapture, reindex, and rebuild. Deleted or stale data never reaches public results. UI and adapters call the same application services as the canonical API.

## Rollout

Complete DB-MIGRATION, CAP-SNAPSHOT, ORG-CONTEXT, execute [tasks](tasks.md) in order, and satisfy the milestone acceptance table. Stop for an upstream decision if implementation exposes a contract gap.
