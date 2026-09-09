# Implementation Plan

## Application shape

Build a local web app using Python, FastAPI, Jinja templates, and HTMX. The browser is the UI shell; no desktop packaging is required initially. Keep the application modular around ingestion, library data, retrieval, analytics, and rendering.

## Ingestion pipeline

1. Validate the input and create an SQLite item with an indexing state.
2. Extract and normalize text into a durable snapshot.
3. Split the snapshot into deterministic chunks and save them in SQLite.
4. Compute a content hash for every chunk.
5. Check RocksDB for an embedding matching the hash, provider, and model version.
6. Generate missing embeddings through the configured provider and upsert them into ChromaDB by chunk ID.
7. Mark the job complete or failed, exposing a retry path.

The implementation must define an `EmbeddingProvider` interface whose only responsibility is embedding batches of text. The default implementation runs a local sentence-transformers model. Cloud providers are optional implementations enabled only through explicit configuration.

## Retrieval

Implement keyword and semantic retrieval as separate services, then merge and deduplicate their chunk results before hydrating results from SQLite. A missing or rebuilding Chroma index must leave keyword search available. Record latency and result count in SQLite search history.

## Dashboard

Use DuckDB only behind an analytics service. Attach SQLite read-only, execute aggregate queries, and return view-model data rather than SQL to templates. The dashboard must not write through DuckDB.

## Reliability and recovery

The service must be restart-safe: SQLite data remains usable if ChromaDB or RocksDB is reset, and the app provides re-indexing to recreate derived state. Failed imports preserve their source metadata and error reason for retry or removal.

## Security baseline

Before fetching a URL, validate scheme, resolve its host, and reject loopback, private, link-local, and reserved network addresses. Limit download size and extraction time. Treat imported text as data, never as executable instructions.

