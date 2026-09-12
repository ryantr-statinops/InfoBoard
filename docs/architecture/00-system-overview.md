# Architecture — System overview

## Current state

FastAPI/Uvicorn, SQLite/FTS5, the server-rendered dashboard, and basic services exist at partial coverage. Minimal keyword search and ingestion work; the worker, semantic index, analytics, and recovery do not yet meet the target contract.

## Target state

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
    Worker --> Rocks[(RocksDB cache)]
    Worker --> Embed[Embedding provider]
    Embed --> Chroma[(ChromaDB vectors)]
    Retrieval --> SQLite
    Retrieval --> Chroma
    Analytics --> Duck[(DuckDB read-only)]
    Duck --> SQLite
```

## Boundaries

- Routes parse, validate, and coordinate; business behavior lives in services.
- Storage adapters are the only boundary to the database.
- SQLite and stored snapshots are the authoritative data sources.
- ChromaDB, RocksDB, and DuckDB are derived stores with fallback or rebuild paths.
- Core mode does not depend on native semantic/analytics dependencies.
