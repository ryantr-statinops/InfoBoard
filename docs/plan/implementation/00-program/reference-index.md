# Implementation reference index

Use this index to find the source of a decision before editing an epic package.

## Canonical repository layers

| Question | Canonical source |
| --- | --- |
| What are we building and why? | [Internal PRD](../../../product/internal-prd/README.md) |
| What is the target system and data contract? | [Architecture](../../../architecture/README.md) |
| What should the user see and do? | [Design](../../../design/README.md) |
| How do we prove quality? | [Quality](../../../quality/README.md) |
| How do we run, repair, and release it? | [Operations](../../../operations/README.md) |
| What is future discovery only? | [Next Plan](../../../product/next-plan/README.md) |

## Runtime and test areas

| Area | Current code | Current tests |
| --- | --- | --- |
| Application and routes | [`app/main.py`](../../../../app/main.py), [`app/services.py`](../../../../app/services.py) | [`tests/test_api_flow.py`](../../../../tests/test_api_flow.py) |
| SQLite and migrations | [`app/db.py`](../../../../app/db.py) | [`tests/test_core.py`](../../../../tests/test_core.py) |
| Worker and indexing | [`app/worker.py`](../../../../app/worker.py), [`app/semantic.py`](../../../../app/semantic.py) | [`tests/test_search.py`](../../../../tests/test_search.py) |
| Analytics | [`app/analytics.py`](../../../../app/analytics.py) | [`tests/test_core.py`](../../../../tests/test_core.py) |
| UI templates | [`app/templates/dashboard.html`](../../../../app/templates/dashboard.html) | [`tests/test_api_flow.py`](../../../../tests/test_api_flow.py) |

These paths describe the current baseline. A planned or missing path must be labelled `target` in an epic package.

## Official references

Epic reference files should link the specific official documentation used by the task, record the relevant version, and explain why it matters. Prefer primary documentation for:

- [FastAPI](https://fastapi.tiangolo.com/)
- [SQLite](https://sqlite.org/docs.html) and [FTS5](https://sqlite.org/fts5.html)
- [pypdf](https://pypdf.readthedocs.io/)
- [Chroma](https://docs.trychroma.com/)
- [RocksDB](https://github.com/facebook/rocksdb/wiki)
- [DuckDB](https://duckdb.org/docs/)

Do not copy external documentation into this repository. The reference index is navigation and provenance, not a second technical source of truth.
