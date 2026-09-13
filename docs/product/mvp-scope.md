# MVP scope

## In scope

- Capture a public HTTP or HTTPS URL.
- Normalize and deduplicate URLs deterministically.
- Store bookmark metadata before asynchronous enrichment.
- Fetch metadata and extract a readable content snapshot.
- Keep the bookmark and expose retry when snapshot capture fails.
- Organize bookmarks with collections, tags, notes, and `inbox`, `active`, or `archived` state.
- Browse, filter, edit, soft-delete, and inspect processing status.
- Local keyword search over current bookmark and snapshot content.
- Semantic and hybrid search using an explicitly configured self-hosted OpenAI-compatible embedding endpoint.
- Installation-level consent before any snapshot text or query is sent to the provider.
- Basic dashboard analytics using filters shared with browse and search.
- Backup, restore, migration, health, and rebuild paths for canonical and derived stores.
- Responsive core workflows for 1440 px and 390 px viewports.
- Preserve pre-MVP text and file items as read-only legacy sources that remain readable, searchable, organizable, and soft-deletable.

## MVP release boundary

Semantic search is required release functionality. The release must prove a configured-provider indexing and query flow. A temporary provider outage may degrade semantic retrieval, but keyword search and canonical bookmark workflows must continue.

## Out of scope

- Creating or recapturing manual text, PDF, Markdown, TXT, image, audio, or video items. Existing text/file items are retained under the legacy compatibility contract.
- Browser extension, automatic browser bookmark import, profile migration, cookies, sessions, localStorage, or IndexedDB portability.
- Cloud synchronization, accounts, collaboration, sharing, or multi-tenant operation.
- AI summaries, chat, recommendations, generated tags, or autonomous research.
- Native mobile applications or network access from another device.
- Authenticated/private-page capture, credential storage, or login automation.
- OCR, citation management, and annotation inside page content.

Out-of-scope capabilities may enter [the idea inbox](ideas.md), but they must not appear in MVP requirements or implementation tasks until accepted.

## Compatibility boundary

- `/api/v1/*` is the canonical MVP API.
- Existing `/api/*` routes remain behavior-preserving adapters for one release only and expose deprecation metadata.
- Existing integer identifiers remain stable through migration.
- Compatibility preserves user data and transition time; it does not keep legacy creation behavior in the new product surface.
