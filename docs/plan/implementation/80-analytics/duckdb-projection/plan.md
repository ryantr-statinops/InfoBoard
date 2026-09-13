# DuckDB analytics projection: implementation plan

## Approach

Build a persistent, read-only DuckDB projection with refresh checkpoints, stale detection, rebuild, and schema-equivalent bounded SQLite fallback. Implement in `app/analytics.py`, projection job handler, analytics checkpoints, parity fixtures with bounded inputs, explicit checkpoints, stable public state, and failure injection from the first delivery slice.

## Flow

1. Establish fixtures and measurable consistency/security invariants.
2. Implement the bounded service or projection boundary.
3. Add atomic activation, fallback, degradation, or rollback behavior.
4. Verify interruption, corruption, deletion, stale data, and restart.
5. Record reproducible, redacted evidence and update rollups.

## Operational boundary

Canonical SQLite and referenced snapshot files are the only required recovery inputs. Optional and derived components never write back to canonical tables. Known-stale output is never presented as current.

## Rollout

Complete DB-MIGRATION, ORG-CONTEXT, RET-FTS, CAP-WORKER; deliver [tasks](tasks.md) in order; satisfy milestone acceptance before marking done.
