# M3 — Search and insights

**Outcome:** Users can retrieve saved content with keyword search, optional semantic search, and useful analytics.
**Gate:** Core mode releases with keyword search, filters, analytics, and safe fallback; semantic, related-content, and cluster behavior is an optional full-mode gate.
**Dependencies:** M2 chunks/index jobs

## Epic packages

- [15 — Search and discovery](15-search-and-discovery/README.md)
- [16 — Analytics and insights](16-analytics/README.md)

## Delivery order

Harden FTS first, then semantic/related/clusters; analytics uses the shared filter contract and may proceed in parallel.

## Acceptance gate

Core mode provides keyword fallback when derived stores fail; analytics matches SQLite; and Vietnamese/English keyword queries meet the retrieval target. If full mode is included in release scope, it must also provide the Chroma/RRF contract.
