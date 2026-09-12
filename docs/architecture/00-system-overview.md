# Architecture — System overview

## Current state

FastAPI/Uvicorn, SQLite/FTS5, the server-rendered dashboard, and basic services exist at partial coverage. Minimal keyword search and ingestion work; the worker, semantic index, analytics, and recovery do not yet meet the target contract.

## Target contract

```mermaid
flowchart LR
    User[Local user] --> UI[Jinja2 + HTMX dashboard]
    UI --> API[FastAPI routes]
    API --> Services[Domain services]
    Services --> Ingest[Ingestion pipeline]
    Services --> Retrieval[Retrieval pipeline]
    Services --> Analytics[Analytics service]
    Ingest --> SQLite[(SQLite + snapshots<br/>system of record)]
    Ingest --> Worker[Index worker]
    Worker --> SQLite
    Worker -. full mode .-> Rocks[(RocksDB cache)]
    Worker -. full mode .-> Embed[Embedding provider]
    Embed -. full mode .-> Chroma[(ChromaDB vectors)]
    Retrieval --> SQLite
    Retrieval -. full mode .-> Chroma
    Analytics --> SQLite
    Analytics -. full mode .-> Duck[(DuckDB read-only)]
    Duck --> SQLite
```

## Boundaries

- Routes parse, validate, and coordinate; business behavior lives in services.
- Storage adapters are the only boundary to the database.
- SQLite and stored snapshots are the authoritative data sources.
- ChromaDB, RocksDB, and DuckDB are derived stores with fallback or rebuild paths.
- Core mode does not depend on native semantic/analytics dependencies.

## Implementation gap

- Route, service, storage, and background-work boundaries exist only at partial coverage.
- Optional adapters and component-aware health do not yet satisfy the target boundaries.

## Owning work

Epics 10–11 establish application/data boundaries; epics 14–16 add worker, retrieval, and analytics adapters; epic 19 owns component health.

## Evidence required

Module-boundary tests, isolated application startup, core-only integration tests, and enabled-optional-component failure tests.
