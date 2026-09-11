# 18 — Reliability, backup, and recovery

**Plan status:** `ready` · **Delivery status:** `not_started` · **Baseline coverage:** `missing`

The epic protects SQLite data through backup, safe restore, migration, derived-index rebuild, and crash-recovery verification.

**Requirements:** `PR-REC-01…05`, `RQ-012`  
**Dependencies:** 11, 14, 17, 19

## Reading order

1. [Plan](plan.md)
2. [Tasks](tasks.md)
3. [References](references.md)
4. [Examples](examples.md)
5. [Recovery guide](guides/recovery.md)
6. [Execution evidence](execution.md)

## Package index

| Area | Location |
| --- | --- |
| Canonical data | [`app/db.py`](../../../../../app/db.py) |
| Current tests | [`tests/test_core.py`](../../../../../tests/test_core.py) |
| Operations | [backup and rebuild](../../../../operations/01-backup-restore-and-rebuild.md) |
