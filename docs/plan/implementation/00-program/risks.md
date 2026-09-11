# 05 — Dependencies and risks

**Status:** `ready`
**Environment:** Linux, Python 3.12, `uv`

## Dependency matrix

| Component | Mode | Role | Fallback |
| --- | --- | --- | --- |
| FastAPI/Uvicorn/Jinja2/HTMX assets | core | Web/API/UI | None |
| SQLite FTS5 | core | System of record + keyword search | Block startup if unavailable |
| `pypdf` | core | PDF text extraction | Report unsupported parser |
| sentence-transformers | full | Local embeddings | Keyword search |
| ChromaDB | full | Persistent vector index | Keyword search |
| `rocksdict` | full | Embedding/cache state | SQLite/no cache |
| DuckDB | full | Read-only analytics | SQLite aggregation |

`pyproject.toml` must provide the full extra before it is advertised. The lockfile is updated only after dependency smoke tests pass. Models are prepared explicitly and record model/revision/dimension metadata.

## Risk register

| Risk | Impact | Mitigation | Gate |
| --- | --- | --- | --- |
| Native wheel unavailable on Linux/Python 3.12 | Full mode cannot install | Keep core mode; pin verified versions; document failure | 05/21 |
| Large/slow model download | Slow startup/UX | Lazy singleton, prepare command, warm-up benchmark | 15/20 |
| Chroma data loss | Semantic unavailable | Rebuild from SQLite chunks | 18 |
| DNS rebinding/redirect SSRF | Local security issue | Check every redirect and connected address | 13/17 |
| Schema drift | Data loss | Versioned migration, backup, upgrade test | 11/18 |
| Worker crash mid-step | Stuck jobs | Durable checkpoints, lease/requeue, idempotent upsert | 14 |
| Heavy analytics query | Slow dashboard | Read-only connection, bounded range, fallback | 16/20 |
| API contract drift | Broken UI | Canonical contract, integration tests, traceability | 04/20 |

## Dependency acceptance

Every new dependency needs rationale, version range, license check, import smoke test, failure message, fallback, and removal path. Do not add runtime CDN assets; static assets must work offline.

## Security assumption

MVP binds only to `127.0.0.1`. Opening the network post-MVP requires a threat model and authentication gate.
