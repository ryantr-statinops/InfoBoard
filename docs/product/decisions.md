# Product decisions

Decision history is append-only. A changed decision receives a new ID and names the record it supersedes.

| ID | Status | Decision | Consequence |
| --- | --- | --- | --- |
| `PD-001` | accepted | Position InfoBoard as a personal, local-first knowledge bookmark manager. | The saved web resource, not a free-form note or AI conversation, is the central object. |
| `PD-002` | accepted | Make public URL capture the only MVP input. | Text/file/browser-profile imports remain in the idea inbox. |
| `PD-003` | accepted | Preserve a bookmark when snapshot capture fails. | Bookmark and snapshot have independent lifecycle states and retry behavior. |
| `PD-004` | accepted | Use collections, tags, and notes together. | Collections and tags are many-to-many; notes remain independent of indexing. |
| `PD-005` | accepted | Require keyword, semantic, and hybrid retrieval for MVP release. | Semantic integration must be verified, while runtime provider failure still falls back to keyword search. |
| `PD-006` | accepted | Use a configurable self-hosted OpenAI-compatible embedding endpoint. | Architecture remains vendor-neutral but locks request, response, model, dimension, timeout, and failure contracts. |
| `PD-007` | accepted | Require installation-level explicit consent before external transfer. | Endpoint/key setup alone does not authorize sending snapshots or queries. |
| `PD-008` | accepted | Keep SQLite authoritative and RocksDB, ChromaDB, and DuckDB derived. | Backup and recovery start with SQLite; every derived store has a rebuild path. |
| `PD-009` | accepted | Keep basic filtered analytics in MVP. | DuckDB accelerates projections, with bounded SQLite fallback. |
| `PD-010` | accepted | Defer AI generation, cloud sync, collaboration, browser portability, and native mobile. | These capabilities cannot enter requirements or implementation without the idea workflow. |
| `PD-011` | accepted | Use English for the MVP interface and canonical documentation. | Localization is not part of MVP acceptance. |
| `PD-012` | accepted | Keep SQLite integer primary keys as canonical identifiers. | Existing IDs remain stable; the API does not introduce UUID aliases. |
| `PD-013` | accepted | Preserve existing text and file items as read-only legacy sources. | Legacy items remain readable, searchable, organizable, and soft-deletable, but cannot be newly created or recaptured. |
| `PD-014` | accepted | Keep old `/api/*` endpoints as adapters for exactly one compatibility release. | `/api/v1/*` is canonical; adapters emit deprecation metadata and are then removed by a planned task. |
| `PD-015` | accepted | Retain soft-deleted canonical data indefinitely in MVP. | Derived data is cleaned up immediately; MVP has no restore, purge, or permanent-delete endpoint. |
| `PD-016` | accepted | Read the semantic API key only from `INFOBOARD_SEMANTIC_API_KEY`. | The settings API reports key presence but never accepts, returns, or persists the secret. |
| `PD-017` | accepted | Run one sequential in-process durable worker. | Installation remains simple; SQLite leases and bounded retry recover interrupted work. |
| `PD-018` | accepted | Bound analytics to all-time totals/top lists and time series of at most 366 days. | Time series defaults to 30 days; top lists return at most 20 rows through DuckDB or SQLite fallback. |
| `PD-019` | accepted | Use URL normalization policy version 1 for identity. | A versioned tracker-removal and canonicalization policy governs deduplication and future migrations. |
| `PD-020` | accepted | Separate capture attempts from immutable successful snapshots. | Pending and failed work never becomes a snapshot version or replaces current readable content. |
| `PD-021` | accepted | Treat FTS5 as a derived index inside the SQLite database. | Canonical SQLite rows are authoritative; restore and repair may rebuild FTS completely. |

## Record format

Every future record includes ID, date, status, context, decision, alternatives, impact, affected requirements, and any superseded decision. Decisions that alter externally visible behavior, schema, privacy, or dependencies must be accepted before architecture or implementation work begins. Records `PD-001` through `PD-010` originated in the product-definition pass; `PD-011` through `PD-021` were accepted during architecture closure.
