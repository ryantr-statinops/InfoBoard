# Analytics API and UI: implementation plan

## Approach

Expose all-time totals/top lists and bounded UTC time series through shared filters with clear backend and degradation metadata. Implement in analytics service/router/schemas, dashboard analytics views, API/browser tests with bounded inputs, explicit checkpoints, stable public state, and failure injection from the first delivery slice.

## Flow

1. Establish fixtures and measurable consistency/security invariants.
2. Implement the bounded service or projection boundary.
3. Add atomic activation, fallback, degradation, or rollback behavior.
4. Verify interruption, corruption, deletion, stale data, and restart.
5. Record reproducible, redacted evidence and update rollups.

## Operational boundary

Canonical SQLite and referenced snapshot files are the only required recovery inputs. Optional and derived components never write back to canonical tables. Known-stale output is never presented as current.

## Rollout

Complete API-CANONICAL, ANA-PROJECTION, ORG-UI; deliver [tasks](tasks.md) in order; satisfy milestone acceptance before marking done.
