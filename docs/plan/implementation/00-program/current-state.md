# 01 — Current state and gap analysis

**Status:** `ready`
**Runtime snapshot:** `f073724` — latest commit changing `app/`, tests, or runtime dependency files
**Documentation baseline:** `b5751f5` — last synchronized docs baseline before this implementation-playbook rebuild
**Working-tree docs:** uncommitted reconciliation changes are present; this file is not evidence of a committed runtime state
**Delivery branch:** `dev`

## Existing baseline

| Area | Coverage | Evidence |
| --- | --- | --- |
| FastAPI + Uvicorn + lifespan | `partial` | `app/main.py`, `/api/health`, template render |
| SQLite schema + FTS5 | `partial` | `app/db.py`, table creation, FTS rebuild; no migration history |
| Text item, hash dedup, chunk | `done-basic` | `app/services.py`; 200-word chunks with 30-word overlap |
| Collections and notes API | `partial` | Basic CRUD; UI/wrapper contract incomplete |
| Soft delete/reindex | `partial` | Endpoint exists; derived cleanup is missing |
| TXT/Markdown/PDF/URL | `partial` | Minimal extractor; limits, redirects, and parser need hardening |
| Worker | `partial` | Synchronous simulation; no background polling/real embedding |
| Keyword search | `done-basic` | FTS5 + excerpt; filter/highlight/error contract needs normalization |
| Semantic search | `missing` | Provider interface exists; ChromaDB is not integrated |
| RRF | `done-basic` | Utility/unit test exists; not connected to search pipeline |
| Analytics | `partial` | Minimal SQLite fallback and DuckDB adapter |
| Dashboard | `partial` | Server-rendered form/list; HTMX, panels, advanced filters, KPI missing |
| Quality | `partial` | Four tests and Ruff pass; HTTP integration, benchmark, and CI missing |

## Known contract gaps

- Runtime creates new items with default `active`; target M1 is `inbox`.
- `pypdf` and full-mode dependencies are not declared in `pyproject.toml`.
- Runtime schema lacks complete content-version/migration history and uses `index_jobs.attempts` instead of target `retry_count`.
- `/api/health` currently returns `{"status":"ok"}` and does not inspect SQLite/FTS5/derived components.

## Current API versus target contract

The cross-cutting [contracts](contracts.md) page is intentionally forward-looking.
The following endpoint differences are known implementation gaps, not evidence of
completed behavior:

| Endpoint/area | Current runtime | Target owner |
| --- | --- | --- |
| `GET /api/health` | Static `ok` response; no SQLite/FTS5/derived checks | 10, 19 |
| `POST /api/items` | Returns the synchronous service result | 12, 14 |
| `GET /api/items` | Basic filters and a raw list response | 12, 15 |
| `GET /api/items/{id}` | Returns current metadata/content without full version/job projection | 11, 12, 14 |
| `PATCH /api/items/{id}` | Updates a limited metadata set; text version behavior is not complete | 11, 12 |
| `DELETE /api/items/{id}` | Returns an immediate delete response; derived cleanup is not queued | 18 |
| `POST /api/search` | Minimal query response without the target filter/score contract | 15 |
| `GET /api/analytics` | Minimal metrics without shared filters or full degraded metadata | 16 |

Implementation work must update this table only when runtime evidence changes. The
target contract remains non-evidence until the linked task is verified.

## Priority gaps

1. Stabilize foundation/data contracts before expanding UI.
2. Complete dashboard and item workspace end to end.
3. Replace minimal extractor/worker with versioned retryable pipeline.
4. Connect persistent semantic indexing while retaining FTS fallback.
5. Complete analytics, backup, security, observability, and release gates.

## Data-safe change rules

- Every schema change requires a migration and backup.
- Do not change the meaning of `items.id`, `content_hash`, `deleted_at`, or existing notes.
- New indexes read from SQLite; never migrate data using only Chroma/RocksDB.
- New API responses require a compatibility adapter while old UI paths remain in use.

Update the runtime snapshot only after a commit genuinely changes runtime files. Record docs-only baselines separately. Never use `done` for a capability represented only by an adapter or mock.
