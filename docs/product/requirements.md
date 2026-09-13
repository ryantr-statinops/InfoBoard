# MVP requirements

This is the only product-requirement register. IDs remain stable after publication; a superseding decision changes status or acceptance rather than silently changing meaning.

| ID | Requirement | Acceptance source |
| --- | --- | --- |
| `CAP-01` | A valid public URL creates or resolves one canonical bookmark before enrichment begins. | [Capture](features/bookmark-capture.md) |
| `CAP-02` | Capture stores source metadata and a versioned content snapshot when extraction succeeds. | [Capture](features/bookmark-capture.md) |
| `CAP-03` | Snapshot failure preserves the bookmark, exposes a safe reason, and permits idempotent retry. | [Capture](features/bookmark-capture.md) |
| `CAP-04` | URL normalization and deduplication are deterministic and do not overwrite personal context. | [Capture](features/bookmark-capture.md) |
| `CAP-05` | Existing text/file items survive migration as read-only legacy sources and remain readable, searchable, organizable, and soft-deletable. | [MVP scope](mvp-scope.md) |
| `ORG-01` | A bookmark can belong to multiple collections and have multiple tags. | [Organization](features/organization.md) |
| `ORG-02` | A bookmark supports personal notes and `inbox`, `active`, or `archived` state. | [Organization](features/organization.md) |
| `ORG-03` | Recapture, reindex, and derived-store rebuild preserve organization and notes. | [Organization](features/organization.md) |
| `ORG-04` | Soft-deleted bookmarks are excluded from every user-facing read path. | [Organization](features/organization.md) |
| `RET-01` | Browse and search share collection, tag, status, domain, and date filters. | [Retrieval](features/retrieval.md) |
| `RET-02` | Local keyword search works without the external semantic provider or derived vector store. | [Retrieval](features/retrieval.md) |
| `RET-03` | A configured MVP installation provides semantic and hybrid search through the compatible provider contract. | [Retrieval](features/retrieval.md) |
| `RET-04` | Results exclude deleted and stale snapshot versions and contain enough context to identify a bookmark. | [Retrieval](features/retrieval.md) |
| `RET-05` | Provider or vector-store failure degrades to keyword retrieval without blocking canonical workflows. | [Retrieval](features/retrieval.md) |
| `ANA-01` | Dashboard analytics use the same filters and canonical semantics as browse and search. | [Analytics](features/analytics.md) |
| `ANA-02` | DuckDB failure uses a bounded SQLite fallback or reports analytics-only degradation. | [Analytics](features/analytics.md) |
| `PRI-01` | No snapshot text or search query leaves the computer before endpoint setup and explicit installation consent. | [Security](../architecture/security-and-privacy.md) |
| `PRI-02` | API keys, raw content, and sensitive provider responses never appear in public responses, diagnostics, or logs. | [Security](../architecture/security-and-privacy.md) |
| `REL-01` | SQLite is authoritative; RocksDB, ChromaDB, and DuckDB are rebuildable derived stores. | [Storage architecture](../architecture/storage-architecture.md) |
| `REL-02` | Restart, retry, backup, restore, and rebuild preserve canonical bookmarks and personal context. | [Recovery](../architecture/lifecycle-and-recovery.md) |
| `UX-01` | The capture-organize-retrieve loop works without exposing storage or indexing terminology. | [Core flows](../design/core-flows.md) |
| `UX-02` | Empty, processing, failed, unconfigured, and degraded states explain the next safe action. | [UI states](../design/ui-states.md) |
| `UX-03` | Core workflows do not overflow horizontally at 1440 px or 390 px. | [UI states](../design/ui-states.md) |
| `UX-04` | The MVP interface and its canonical user-facing messages are English-first. | [Users and jobs](users-and-jobs.md) |

All requirements are mandatory for the MVP unless an accepted decision explicitly marks one deferred and updates scope, acceptance, and roadmap together.
