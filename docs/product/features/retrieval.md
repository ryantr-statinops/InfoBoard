# Feature specification: retrieval

## Outcome

The user can find a saved bookmark using remembered words, meaning, source, organization, or time.

## Modes

- **Keyword:** local lexical retrieval over current bookmark metadata and snapshot text.
- **Semantic:** vector retrieval using query embeddings from the configured compatible endpoint.
- **Hybrid:** deterministic fusion of keyword and semantic candidates; this is the default when semantic service is ready.

Semantic capability is required for MVP release, but provider availability is not required for every request. Keyword fallback remains available during provider, cache, or ChromaDB failure.

## Filters and results

- Shared filters: collection, tag, organization status, source domain, capture date range, and saved date range.
- Results represent bookmarks, not chunks or snapshot versions.
- Each result includes bookmark ID, title, canonical URL/domain, excerpt, matched mode, current status, and relevant collection/tag context.
- Deleted bookmarks, stale versions, and the current bookmark in a related-items query are excluded.

## Provider boundary

- Semantic controls remain unconfigured until an endpoint, model, API key, successful connection check, and explicit consent are recorded.
- Snapshot text is sent only for indexing current versions after consent.
- Search queries are sent only when semantic or hybrid mode is invoked after consent.
- Revocation stops new external calls immediately; local keyword search remains available.

## Acceptance

- Keyword search succeeds with provider, RocksDB, or ChromaDB unavailable.
- Configured semantic and hybrid searches return current-version results through an OpenAI-compatible endpoint.
- Hybrid fusion is deterministic for an identical fixture and configuration.
- Filter semantics match list and analytics semantics.
- Degraded responses identify the unavailable capability without exposing secrets.
