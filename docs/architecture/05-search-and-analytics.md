# Architecture — Search and analytics

## Current state

FTS5 keyword search và RRF utility đã có mức cơ bản nhưng chưa nối đầy đủ; Chroma semantic search chưa có. Analytics có SQLite fallback và DuckDB adapter tối thiểu.

## Target search pipeline

```mermaid
flowchart LR
    Query[Query + shared filters] --> FTS[SQLite FTS5]
    Query --> Mode{Full mode ready?}
    Mode -->|yes| Embed[Query embedding]
    Embed --> Chroma[Chroma candidates]
    FTS --> RRF[RRF merge]
    Chroma --> RRF
    Mode -->|no| Keyword[Keyword ranking]
    RRF --> Hydrate[Hydrate current SQLite rows]
    Keyword --> Hydrate
    Hydrate --> Results[Item results + excerpt + mode]
```

## Target analytics pipeline

```mermaid
flowchart LR
    Filters[Shared filters] --> Duck[DuckDB read-only aggregates]
    Duck --> Cards[KPI / activity / distributions]
    Duck -. unavailable .-> SQLite[SQLite bounded fallback]
    SQLite --> Cards
    SQLiteSource[(SQLite system of record)] --> Duck
```

## Rules

- Keyword search luôn usable trong core mode.
- Chroma chỉ trả chunk IDs/scores; content và metadata được hydrate từ SQLite.
- Deleted/stale-version candidates bị loại trước response.
- Analytics dùng cùng filter semantics với item list.
- DuckDB/Chroma failure trả degraded mode, không làm toàn dashboard/search lỗi.
