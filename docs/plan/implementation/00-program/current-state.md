# Current runtime and gap analysis

**Runtime snapshot:** `f0737245f919697676bbd80b2c5afd68bde635a0` (`feat: add result fusion and analytics adapter`)

Observed runtime evidence is not delivery status. The target remains the canonical product and architecture documentation.

## Baseline and owners

| Area | Runtime evidence | Target gap | Delivery owner |
| --- | --- | --- | --- |
| Application | FastAPI lifespan, Jinja UI, static health route | Central settings, lifecycle boundaries, correlation and errors | APP-LIFECYCLE |
| SQLite | `items`, one-row content, chunks, collections, notes, jobs, contentless FTS | Migrations, target canonical model, revisions and checkpoints | DB-MIGRATION |
| API | Unversioned `/api/*`, raw shapes, no optimistic concurrency | Canonical `/api/v1/*`, envelopes, cursors, `If-Match`, adapters | API-CANONICAL |
| Capture | URL/text/file ingestion runs synchronously | Bookmark-first durable URL capture and immutable successful snapshots | CAP-SNAPSHOT |
| URL safety | Basic scheme/address checks and timeout | Normalization v1, redirect/connection validation and bounded parsing | CAP-URL |
| Worker | Simulated synchronous job transitions | Sequential lifespan worker, leases, checkpoints and bounded retry | CAP-WORKER |
| Organization | Collections and notes exist | Tags, versions, target memberships, status and deletion semantics | ORG-CONTEXT |
| Legacy data | Text/file writes remain enabled | Read-only migrated legacy items with preserved IDs and content | ORG-LEGACY |
| UI | Basic form, list and search | Accepted IA, detail context, explicit states and responsive behavior | ORG-UI |
| Keyword | Basic FTS rebuild and MATCH | Current-version indexing, filters, cursor/sort, safe query and health | RET-FTS, RET-KEYWORD |
| Semantic | Local sentence-transformers adapter | External compatible endpoint, consent, revisions and validation | SEM-CONFIG, SEM-EMBED |
| ChromaDB | Absent | Persistent revisioned vectors and canonical filtering | VEC-CHROMA |
| RocksDB | Absent | Revision-scoped reusable embedding cache | CACHE-ROCKS |
| DuckDB | Minimal in-memory adapter | Persistent projection, checkpoints, parity and stale detection | ANA-PROJECTION |
| Reliability | Basic tests and Ruff | Health, diagnostics, backup/rebuild, security and release evidence | REL-*, RELEASE-* |

## Migration boundary

Preserve integer IDs, canonical user data, content, relationships, notes, timestamps, and deletion markers. Convert usable legacy content to snapshot version 1, rebuild every derived representation, and abort activation when count or mapping checks fail. Existing search history remains only in the verified pre-migration backup.

## Compatibility boundary

Build `/api/v1/*` as canonical. Maintain target-compatible `/api/*` adapters for exactly one release with deprecation metadata. Retired text/file writes return `410 LEGACY_WRITE_REMOVED`. Adapter removal belongs to RELEASE-COMPAT after one released compatibility version.
