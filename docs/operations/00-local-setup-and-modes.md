# Operations — Local setup and modes

## Current state

The project runs on the Python/FastAPI baseline; clean install, optional full extras, and the prepare-model workflow are not release-verified.

`pypdf` and the full-mode extras are target requirements, not proof that the current
manifest installs them. Do not advertise a clean core/full installation until the
packaging tasks produce a reproducible install transcript.

The current runtime reads `INFOBOARD_DB` from the process environment only. The
`INFOBOARD_CHROMA_DIR` and `INFOBOARD_MAX_FILE_MB` entries in `.env.example` are
configuration placeholders until their settings boundary is implemented. Copying
`.env.example` to `.env` does not auto-load values in the current runtime.

## Target setup flow

1. Install Python 3.12 and `uv`.
2. Clone the repository and install core dependencies from the lockfile.
3. Export supported environment variables such as `INFOBOARD_DB`; do not commit secrets.
4. Run versioned migrations against the explicitly selected database.
5. Check `/api/health`, then start `app.main:app` on `127.0.0.1`.
6. Install full extras or prepare a model only when the user selects semantic/analytics capability.

## Modes

| Mode | Required | Failure behavior |
| --- | --- | --- |
| Core | FastAPI, Jinja2/HTMX assets, SQLite/FTS5, extraction dependencies | SQLite/FTS unavailable must fail startup/health |
| Full | Embedding model, ChromaDB, RocksDB, DuckDB | Missing derived dependencies report degraded state; core mode remains available |

## Configuration principles

- Data directory, bind address, and provider configuration have safe local defaults.
- Model download is an explicit preparation action and never occurs during a dashboard request.
- Health reports component readiness and remediation guidance without exposing secrets or sensitive paths.
