# Backup, restore, and rebuild: implementation plan

## Approach

Create SQLite-consistent canonical backups with managed snapshots and manifests, then restore safely and rebuild all derived stores in dependency order. Implement in backup/restore services or CLI, maintenance jobs, archive/manifest validation, disaster-recovery tests with bounded inputs, explicit checkpoints, stable public state, and failure injection from the first delivery slice.

## Flow

1. Establish fixtures and measurable consistency/security invariants.
2. Implement the bounded service or projection boundary.
3. Add atomic activation, fallback, degradation, or rollback behavior.
4. Verify interruption, corruption, deletion, stale data, and restart.
5. Record reproducible, redacted evidence and update rollups.

## Operational boundary

Canonical SQLite and referenced snapshot files are the only required recovery inputs. Optional and derived components never write back to canonical tables. Known-stale output is never presented as current.

## Rollout

Complete DB-MIGRATION, RET-FTS, VEC-CHROMA, CACHE-ROCKS, ANA-PROJECTION, REL-HEALTH; deliver [tasks](tasks.md) in order; satisfy milestone acceptance before marking done.
