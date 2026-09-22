# Refactor workspace

## Status

`discovery`

This workspace records the restart of the product direction. The previous InfoBoard idea is retired for this refactor. Its documentation remains in the repository as historical material only.

## Current direction

The first product direction under evaluation is a Chromium-first tab search extension:

```text
Keyboard shortcut -> instant search UI -> fuzzy/ranked tab results -> switch to tab
```

The target runtime is a warm Go process or a stable Native Messaging host. Fast lexical search is the hot path. ChromaDB is optional and must not block normal tab switching.

## Rules

1. Start from a user problem, not from a database.
2. Chromium desktop is the first support boundary.
3. Keep the hot path local, bounded, and measurable.
4. Treat SQLite as persistence only if the product needs persistence.
5. Add ChromaDB, RocksDB, or DuckDB only when a concrete behavior requires them.
6. Record provisional decisions here before implementation work begins.
7. Do not copy old InfoBoard requirements, APIs, schemas, or migration plans into the new product.

## Documents

1. [Browser landscape](browser-landscape.md)
2. [Tab search idea brief](idea-brief.md)
3. [Runtime architecture](runtime-architecture.md)
4. [Decision log](decisions.md)
5. [Refactor roadmap](roadmap.md)
