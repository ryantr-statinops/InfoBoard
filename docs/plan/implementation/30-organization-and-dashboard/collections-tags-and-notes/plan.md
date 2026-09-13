# Collections, tags, notes, and bookmark state: implementation plan

## Approach

Deliver versioned organization resources and memberships that survive recapture, reindex, and rebuild. Work in organization repositories/services, API routers and schemas, membership tests, retaining existing behavior only when target contract tests prove compatibility.

## Delivery flow

1. Add fixtures and safeguards for current data and callers.
2. Introduce the target service/repository or UI boundary.
3. Move reads and writes to the target behavior.
4. Verify concurrency, failure, deletion, and recovery paths.
5. Record exact evidence and update rollups only after acceptance.

## Invariants

Canonical data and user context survive recapture, reindex, and rebuild. Deleted or stale data never reaches public results. UI and adapters call the same application services as the canonical API.

## Rollout

Complete DB-MIGRATION, API-CANONICAL, execute [tasks](tasks.md) in order, and satisfy the milestone acceptance table. Stop for an upstream decision if implementation exposes a contract gap.
