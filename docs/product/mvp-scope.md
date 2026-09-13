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

## MVP release boundary

Semantic search is required release functionality. The release must prove a configured-provider indexing and query flow. A temporary provider outage may degrade semantic retrieval, but keyword search and canonical bookmark workflows must continue.

## Out of scope

- Manual text, PDF, Markdown, TXT, image, audio, or video ingestion.
- Browser extension, automatic browser bookmark import, profile migration, cookies, sessions, localStorage, or IndexedDB portability.
- Cloud synchronization, accounts, collaboration, sharing, or multi-tenant operation.
- AI summaries, chat, recommendations, generated tags, or autonomous research.
- Native mobile applications or network access from another device.
- Authenticated/private-page capture, credential storage, or login automation.
- OCR, citation management, and annotation inside page content.

Out-of-scope capabilities may enter [the idea inbox](ideas.md), but they must not appear in MVP requirements or implementation tasks until accepted.
