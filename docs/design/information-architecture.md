# Information architecture

## Primary navigation

| Destination | Purpose |
| --- | --- |
| Library | Browse and filter saved bookmarks; default to non-archived items. |
| Inbox | Review newly captured bookmarks and failed enrichment. |
| Collections | Create collections and browse membership. |
| Tags | Browse and maintain normalized labels. |
| Analytics | Inspect filtered library summaries and processing health. |
| Settings | Configure semantic provider, consent, maintenance, backup, and diagnostics. |

## Library workspace

The library uses one stable workspace with:

- Search input and explicit keyword/semantic/hybrid mode.
- Shared filters for collection, tag, status, domain, saved date, and capture date.
- Bookmark results showing title, domain, excerpt, organization context, and processing state.
- A detail surface that preserves query, filters, sort, page/cursor, and scroll context when opened or closed.
- A capture action available from every primary destination.

## Bookmark detail

Detail presents canonical data before enrichment:

1. URL, title, description, domain, saved time, and organization status.
2. Collections, tags, and notes.
3. Current snapshot content and provenance when available.
4. Capture and indexing status with retry actions.
5. Edit, recapture, archive, and soft-delete actions.

For a migrated legacy text/file item, detail shows a `Legacy source` label and omits capture/recapture actions. Read, organization, search, archive, and soft-delete remain available.

Storage names such as SQLite, RocksDB, ChromaDB, and DuckDB appear only in settings or diagnostics, never in the core workflow.

## URL state

Browse and search state is URL-addressable where practical. Opening a bookmark must not discard the prior query, filters, sort, or pagination context. Semantic provider secrets and consent actions must never be encoded in a URL.
