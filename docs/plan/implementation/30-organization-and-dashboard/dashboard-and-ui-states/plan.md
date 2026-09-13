# Dashboard and UI states: implementation plan

## Approach

Implement the English-first capture-organize-retrieve interface and every accepted empty, pending, failure, conflict, legacy, and responsive state. Work in Jinja templates, static assets, UI routes, browser/e2e tests, retaining existing behavior only when target contract tests prove compatibility.

## Delivery flow

1. Add fixtures and safeguards for current data and callers.
2. Introduce the target service/repository or UI boundary.
3. Move reads and writes to the target behavior.
4. Verify concurrency, failure, deletion, and recovery paths.
5. Record exact evidence and update rollups only after acceptance.

## Invariants

Canonical data and user context survive recapture, reindex, and rebuild. Deleted or stale data never reaches public results. UI and adapters call the same application services as the canonical API.

## Rollout

Complete API-CANONICAL, CAP-SNAPSHOT, ORG-CONTEXT, ORG-LEGACY, execute [tasks](tasks.md) in order, and satisfy the milestone acceptance table. Stop for an upstream decision if implementation exposes a contract gap.
