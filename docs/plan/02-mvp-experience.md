# MVP Experience and Interfaces

## Supported input

The first release accepts:

- a public article URL;
- PDF;
- Markdown;
- plain-text file;
- manually entered text.

Saving a URL creates a text snapshot rather than only a bookmark. The importer accepts only HTTP(S) URLs that resolve to public addresses; pages requiring login, social-media applications, video pages, and image capture are not MVP requirements.

## Main screens

### Library

A filterable list of items showing title, source, collections, reading state, saved time, and indexing outcome. The user can filter by collection, source type, state, or search query.

### Collections

Collections represent projects or areas such as `MLOps`, `Career`, or `Research`. Items may belong to multiple collections. Collection membership is the primary organization mechanism; tags are not required in the first release.

### Item reader

The reader shows the saved text snapshot, original URL or file metadata, collection memberships, a lightweight state (`unread`, `reading`, `completed`, or `archived`), and one item-level personal note.

### Search

Search combines SQLite keyword retrieval with Chroma semantic retrieval. Results show the source item, a relevant excerpt, collection memberships, and the reason/score needed to judge relevance. Filters narrow both result paths consistently.

### Dashboard

The dashboard answers practical questions: What has been saved recently? Which collections are growing? What has been completed versus unread? Which sources and topics dominate the library? It avoids recommendations and automated summaries in the MVP.

## HTTP interface

- `POST /items`: save a URL, upload a supported file, or submit text; returns a new item in `indexing` state.
- `GET /items` and `GET /items/{id}`: list and read stored items.
- `PATCH /items/{id}`: update reading state, note, title, or collection memberships.
- `GET`, `POST`, and `PATCH /collections`: manage collections.
- `POST /search`: run hybrid retrieval with optional source, collection, and state filters.
- `GET /analytics`: return dashboard aggregates.

## Explicitly deferred

Bookmark-HTML import and a browser extension are future input adapters. They must feed the same ingestion pipeline but are not part of the first release. User accounts, cloud sync, collaboration, web-image snapshots, highlights tied to text ranges, and recommendation feeds are also deferred.

