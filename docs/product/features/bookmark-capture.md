# Feature specification: bookmark capture

## Outcome

The user can submit a public web URL and immediately retain a manageable bookmark while InfoBoard enriches it with metadata and a durable content snapshot.

## Primary flow

1. The user submits an HTTP or HTTPS URL.
2. InfoBoard validates and deterministically normalizes it.
3. A new canonical bookmark is created in `inbox`, or the existing bookmark is returned.
4. A background capture job fetches public content within configured limits.
5. Successful extraction creates a new immutable snapshot version and queues indexing.
6. The UI moves from pending/processing to ready without requiring a page reload.

## Duplicate behavior

- URL-normalization policy revision 1 is the URL identity contract for MVP; remote canonical hints never replace it.
- Submitting an existing active bookmark returns it and may attach explicitly requested collections or tags.
- Duplicate capture never overwrites notes, status, existing collections, tags, or a valid current snapshot.
- A user may explicitly request recapture; successful recapture creates the next content version.

## Failure behavior

- Invalid or unsafe input creates no bookmark.
- Fetch, redirect, content-type, size, timeout, or extraction failure preserves a successfully created bookmark and marks snapshot capture `failed`.
- A failed bookmark remains editable, organizable, and discoverable by URL/title metadata.
- Retry reuses the bookmark, creates no duplicate job for the same active attempt, and only advances the current version after snapshot persistence succeeds.

## Acceptance

- Valid public URL capture survives application restart.
- Canonically equivalent URLs do not create unintended duplicates.
- Failed extraction leaves a visible bookmark and actionable retry.
- Recapture never removes personal notes or organization.
- A successful snapshot records provenance, capture time, checksum, and content version.
