# Analytics pipeline

## MVP projection

DuckDB accelerates these accepted metrics:

- Active bookmark count.
- New bookmarks grouped by day/week/month.
- Counts by organization status and source domain.
- Top collections and tags.
- Snapshot and indexing state counts.

The projection contains only the canonical identifiers, dimensions, timestamps, and states needed for these metrics. Raw snapshot text, note bodies, API keys, and provider responses are excluded.

## Shared filter contract

Analytics, browse, and search share normalized collection IDs, tag IDs, bookmark status, source domain, saved date range, and capture date range. Empty filter arrays mean no restriction. Date ranges are UTC half-open intervals `[from, to)`. Soft-deleted bookmarks are always excluded and cannot be reintroduced by filters.

## Refresh flow

1. Read the last successful SQLite source watermark from `analytics_checkpoints`.
2. Select changed canonical rows up to a fixed transaction snapshot/watermark.
3. Apply idempotent upserts/deletes to a staging DuckDB projection.
4. Validate row counts, deleted/current-version exclusions, and a projection checksum.
5. Atomically expose the refreshed projection and commit its checkpoint metadata.

A full rebuild creates a new projection beside the active one, verifies it, then switches the active path. Failed staging output is disposable and cannot change SQLite.

## Query boundary

- Application analytics connections are read-only.
- Queries use predefined statements, bound parameters, result-row limits, and bounded date ranges.
- DuckDB never attaches arbitrary filesystem paths supplied by a request.
- A projection is current only when its source watermark and schema revision are compatible with SQLite metadata.

## Fallback

When DuckDB is missing, stale, corrupt, or incompatible, the service runs equivalent bounded SQLite aggregates for the supported MVP range. The response reports `actual_backend: sqlite` and any safe degradation code. If fallback exceeds its bound or SQLite is unavailable, analytics alone returns unavailable; it never serves known-stale metrics as current.

## Consistency evidence

A versioned fixture must produce equivalent populations and metric values through DuckDB and SQLite fallback. Refresh interruption, soft delete, status change, membership change, and restore/rebuild scenarios are part of the required test matrix.
