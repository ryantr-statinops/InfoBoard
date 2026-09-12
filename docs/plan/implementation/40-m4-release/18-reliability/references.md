# References — Reliability, backup, and recovery

| Type | Reference | Use |
| --- | --- | --- |
| Product | [Success criteria](../../../../product/internal-prd/06-success-criteria.md), `PR-REC-01`, `PR-REC-02`, `PR-REC-03`, `PR-REC-04`, `PR-REC-05` | Recoverability promise |
| Architecture | [Lifecycle and recovery](../../../../architecture/06-lifecycle-and-recovery.md), [data model](../../../../architecture/02-data-model-and-erd.md) | Canonical/derived boundary |
| Quality | [MVP quality gates](../../../../quality/03-mvp-quality-gates.md) | Recovery evidence |
| Operations | [Backup and rebuild](../../../../operations/01-backup-restore-and-rebuild.md) | Operator workflow |
| Evidence | [`app/db.py`](../../../../../app/db.py), [`tests/test_core.py`](../../../../../tests/test_core.py) | Current baseline |
| Official | [SQLite backup API](https://sqlite.org/backup.html), [SQLite integrity check](https://sqlite.org/pragma.html#pragma_integrity_check) | Recovery reference |
