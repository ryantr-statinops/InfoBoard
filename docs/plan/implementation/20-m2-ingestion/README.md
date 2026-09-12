# M2 — Reliable ingestion

**Outcome:** Supported sources become durable items through one bounded extraction and indexing lifecycle.
**Gate:** Source import, chunk/index, retry, and restart scenarios pass with evidence.
**Dependencies:** M1

## Epic packages

- [13 — Ingestion sources](13-ingestion-sources/README.md)
- [14 — Indexing worker and cache](14-indexing-worker/README.md)

## Delivery order

`13 → 14`; the extractor document contract must be stable before the worker consumes jobs.

## Acceptance gate

Text, TXT, Markdown, PDF, and public URL imports create the expected snapshot/chunks; limits and SSRF are enforced; retry/restart does not lose or duplicate jobs.
