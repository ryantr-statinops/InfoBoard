# M3 — Search and insights

**Outcome:** Users can retrieve saved content with keyword search, optional semantic search, and useful analytics.
**Gate:** Keyword search is always available; semantic, related-content, and analytics failures are safe and visible.
**Dependencies:** M2 chunks/index jobs

## Epic packages

- [15 — Search and discovery](15-search-and-discovery/README.md)
- [16 — Analytics and insights](16-analytics/README.md)

## Delivery order

Harden FTS first, then semantic/related/clusters; analytics uses the shared filter contract and may proceed in parallel.

## Acceptance gate

Keyword fallback works when derived stores fail; full mode provides Chroma/RRF; analytics matches SQLite; Vietnamese/English queries meet the retrieval target.
