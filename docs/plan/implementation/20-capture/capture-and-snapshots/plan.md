# Bookmark capture and snapshots: implementation plan

## Approach

Persist bookmarks and attempts before enrichment, then atomically activate immutable successful snapshots without disturbing personal context. Implement this as a bounded package over capture application service, bookmark/attempt/snapshot repositories, managed content storage, API tests. Reuse current runtime behavior only where contract tests prove it compatible.

## Delivery flow

1. Establish tests and migration/compatibility safeguards.
2. Introduce the target boundary behind application services.
3. Move callers to the target behavior.
4. Verify failure recovery and canonical-data preservation.
5. Record commits, commands, results, and deviations in [execution evidence](execution.md).

## Data and failure rules

Canonical SQLite writes commit before dependent derived work. Partial external or derived failure must remain retryable and must not invent successful state. Logs and responses contain safe codes and correlation data, never secrets or raw captured content.

## Rollout

Complete prerequisites (DB-MIGRATION, API-CANONICAL, CAP-URL), deliver [tasks](tasks.md) in order, satisfy milestone acceptance, then update rollup status in the same commit. Contract changes require an accepted upstream documentation change before implementation continues.
