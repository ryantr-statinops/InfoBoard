# References — Data model and migrations

| Type | Reference | Use |
| --- | --- | --- |
| Product | [Domain model](../../../../product/internal-prd/04-domain-model.md), `PR-ORG-*`, `PR-REC-*` | Item, relation, and recovery semantics |
| Architecture | [Data model and ERD](../../../../architecture/02-data-model-and-erd.md), [lifecycle](../../../../architecture/06-lifecycle-and-recovery.md) | Target schema and recovery invariants |
| Quality | [MVP quality gates](../../../../quality/03-mvp-quality-gates.md) | Migration/recovery evidence |
| Operations | [Backup and rebuild](../../../../operations/01-backup-restore-and-rebuild.md) | Safe migration workflow |
| Evidence | [`app/db.py`](../../../../../app/db.py), [`tests/test_core.py`](../../../../../tests/test_core.py) | Current runtime baseline |
| Official | [SQLite ALTER TABLE](https://sqlite.org/lang_altertable.html), [SQLite FTS5](https://sqlite.org/fts5.html) | Migration and search behavior |
