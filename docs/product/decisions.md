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

## Record format

Every future record includes ID, date, status, context, decision, alternatives, impact, affected requirements, and any superseded decision. Decisions that alter externally visible behavior, schema, privacy, or dependencies must be accepted before architecture or implementation work begins.
