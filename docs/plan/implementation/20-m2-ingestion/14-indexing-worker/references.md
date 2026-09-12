# References — Indexing worker and cache

| Type | Reference | Use |
| --- | --- | --- |
| Product | [Domain model](../../../../product/internal-prd/04-domain-model.md), `PR-CAP-05`, `PR-REC-01`, `PR-REC-02`, `PR-REC-03` | Content lifecycle and recoverability |
| Architecture | [Indexing pipeline](../../../../architecture/04-indexing-pipeline.md), [lifecycle](../../../../architecture/06-lifecycle-and-recovery.md), [interfaces](../../../../architecture/09-interface-contracts.md) | Job states, response projection, and derived stores |
| Design | [UI states](../../../../design/02-ui-states-and-responsive.md) | Technical-to-UI state mapping |
| Quality | [Search evaluation](../../../../quality/02-search-and-performance-evaluation.md) | Index/retrieval evidence |
| Operations | [Backup and rebuild](../../../../operations/01-backup-restore-and-rebuild.md), [health](../../../../operations/02-health-and-troubleshooting.md) | Recovery and degraded mode |
| Evidence | [`app/worker.py`](../../../../../app/worker.py), [`tests/test_search.py`](../../../../../tests/test_search.py) | Current baseline |
| Official | [SQLite FTS5](https://sqlite.org/fts5.html), [RocksDB wiki](https://github.com/facebook/rocksdb/wiki) | Derived-store reference |
