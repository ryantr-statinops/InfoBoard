# Feature specification: analytics

## Outcome

The dashboard summarizes the user's saved knowledge without changing canonical data or blocking the core workflow.

## MVP metrics

- Total active bookmarks.
- New bookmarks over time.
- Counts by organization status.
- Counts by source domain.
- Most-used collections and tags.
- Snapshot and indexing state counts.

All metrics use the same collection, tag, status, domain, saved-date, and capture-date filter semantics as browse and search. Soft-deleted bookmarks are excluded.

Time series default to 30 days and accept at most 366 days. All-time views provide totals and top collection/tag/domain lists only; top lists contain at most 20 rows.

## Failure behavior

- DuckDB is an acceleration/projection layer, not a source of truth.
- If its projection is stale or unavailable, InfoBoard uses bounded SQLite aggregation for the supported range.
- If a safe fallback cannot complete, only analytics is marked degraded; capture, organization, read, and search remain available.

## Acceptance

- Metrics match an equivalent canonical SQLite fixture.
- Shared filters return consistent list, search, and analytics populations.
- Stale projection is detected rather than silently shown as current.
- Rebuilding analytics does not write to canonical bookmark tables.
