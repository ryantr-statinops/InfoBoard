# Architecture — Ingestion pipeline

## Current state

Text/hash/chunk processing and TXT/Markdown/PDF/URL extractors exist at a basic level. URL/PDF limits, redirect validation, parser hardening, and durable handoff to the worker remain partial.

## Target contract

```mermaid
flowchart LR
    Input[Text / URL / File] --> Validate[Validate type, size, URL]
    Validate --> Extract[Extractor]
    Extract --> Normalize[Normalize text + metadata]
    Normalize --> Hash[Content hash + dedup]
    Hash --> Commit[SQLite transaction<br/>item + snapshot + job]
    Commit --> Queue[Queued index job]
    Queue --> UI[Processing status]
```

## Source rules

- Text is stored directly and is the only source that edits content in the MVP.
- TXT/Markdown use bounded decoding/parsing; a textless PDF returns a clear unsupported error.
- URLs are limited to public HTTP(S), with DNS/connectivity checks on every redirect.
- Duplicate input keeps the existing item; it may attach a new collection but must not overwrite notes/status.
- Temporary files must be cleaned up on both success and failure paths.

## Transaction boundary

The item, snapshot/content version, and index job are committed in SQLite before worker processing. A derived index must not be used to confirm that an item was saved.

## Implementation gap

- Source-specific limits, PDF failure behavior, redirect validation, parser hardening, and temporary-file cleanup remain incomplete.
- Durable handoff from the SQLite transaction to worker processing is not implemented end to end.

## Owning work

Epic 13 owns source validation/extraction/deduplication; epic 14 owns durable worker handoff; epic 17 owns cross-cutting network and rendering controls.

## Evidence required

Bounded extractor tests, SSRF/redirect fixtures, duplicate scenarios, transaction-failure tests, restart tests, and temporary-file cleanup checks.
