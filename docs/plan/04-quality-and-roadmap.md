# Quality Plan and Roadmap

## MVP acceptance checks

- Import fixture content from one public HTML article, one PDF, one Markdown file, one text file, and manual text.
- Confirm that each import produces a readable snapshot and deterministic SQLite chunks.
- Place one item in multiple collections; update state and note; restart the app and confirm durability.
- Import identical content twice and verify the second index run reuses the RocksDB embedding cache.
- Verify keyword and semantic search return the correct source item and excerpt; collection/state filters apply to both paths.
- Simulate extraction and embedding failures, expose their states, and verify retry behavior.
- Compare DuckDB dashboard figures with SQLite source rows.
- Verify private/internal URLs and unsupported file types are rejected before content processing.

## Observability

Store structured import outcomes, indexing duration, embedding-cache hit/miss counts, search latency, and result counts. These local metrics are sufficient to diagnose MVP behavior and feed the dashboard without external telemetry.

## Delivery order

1. Establish project configuration, SQLite schema, local storage locations, and basic library UI.
2. Implement manual text and file ingestion, snapshot reader, collections, state, and note.
3. Add safe public-article extraction and background indexing state.
4. Add local embedding, ChromaDB semantic search, and hybrid search results.
5. Add RocksDB cache/retry behavior and DuckDB dashboard aggregates.
6. Add configurable cloud embedding providers after the local path is verified.
7. Add bookmark-HTML import, then a browser extension, as adapters over the stable ingestion API.

## Non-goals for the MVP

The first release does not promise multi-user access, remote sync, real-time collaboration, automated recommendations, OCR, video transcripts, article image archiving, or range-based highlights. Each can be added later without changing SQLite ownership of the primary data.

