# Architecture — Tech stack

## Current state

The baseline uses Python, FastAPI, Jinja2, and SQLite/FTS5. Some full-mode adapters exist minimally; dependency smoke tests, lockfile verification, and degraded behavior are incomplete. `pypdf` and the extraction/full-mode dependencies are not declared in `pyproject.toml`; this is an implementation gap, not an installed capability.

## Target state

| Layer | Technology | Responsibility | Mode |
| --- | --- | --- | --- |
| Runtime | Python 3.12 | Application runtime | Core |
| HTTP | FastAPI + Uvicorn | API, routes, lifespan | Core |
| UI | Jinja2 + HTMX + static CSS | Server-rendered dashboard | Core |
| Primary data | SQLite + FTS5 | Transactions, metadata, snapshots, keyword retrieval | Core |
| Vector index | ChromaDB | Semantic retrieval by chunk ID | Full |
| Cache | RocksDB/rocksdict | Rebuildable embedding/cache state | Full |
| Analytics | DuckDB | Read-only aggregate queries | Full, SQLite fallback |
| Embedding | sentence-transformers | Local semantic vectors | Full |
| Extraction | Trafilatura, `pypdf`, Markdown/text parsers | Public URL and file extraction | Core |
| Test | pytest + httpx | Unit, integration and HTTP tests | Development |
| Quality | Ruff + type check + CI | Static and release gates | Development |

## Dependency policy

- Core mode must install and run independently.
- Full mode is provided through optional dependencies and reports a clear degraded state when they are missing.
- Models are downloaded only through an explicit preparation action; record model ID, revision, and dimension.
- Do not use a runtime CDN; static assets must work offline.
