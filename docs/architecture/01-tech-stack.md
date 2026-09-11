# Architecture — Tech stack

## Current state

Baseline đang dùng Python, FastAPI, Jinja2 và SQLite/FTS5. Một số full-mode adapter tồn tại ở mức tối thiểu; dependency smoke test, lockfile và degraded behavior chưa hoàn tất. `pypdf` cùng các extraction/full-mode dependencies chưa được khai báo trong `pyproject.toml`; đây là implementation gap, không phải capability đã cài đặt.

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

- Core mode phải cài và chạy độc lập.
- Full mode được cung cấp qua optional dependencies và báo degraded rõ ràng khi thiếu.
- Model chỉ được tải bằng prepare action rõ ràng; lưu model ID, revision và dimension.
- Không dùng runtime CDN; static assets phải chạy offline.
