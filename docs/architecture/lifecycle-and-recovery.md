# Lifecycle and recovery

## Bookmark and snapshot lifecycle

```mermaid
stateDiagram-v2
    [*] --> Inbox: bookmark persisted
    Inbox --> Active
    Active --> Archived
    Archived --> Active
    Inbox --> Deleted: soft delete
    Active --> Deleted: soft delete
    Archived --> Deleted: soft delete
```

Organization state is independent from snapshot and index state. Soft delete sets `deleted_at`, excludes the bookmark immediately, and queues derived cleanup. Permanent purge is not an MVP user action.

Capture attempts transition `queued -> processing -> succeeded | failed`; a retry returns the same latest failed logical attempt to `queued` while its retry budget remains. Explicit recapture after success creates the next attempt number. Only a succeeded attempt creates one immutable snapshot. A ready snapshot remains current while a later attempt is queued, processing, or failed.

## Durable jobs and restart

- One sequential in-process worker starts and stops with application lifespan and acquires bounded leases in SQLite transactions.
- The worker handles capture attempts and index jobs from one deterministic priority queue; cleanup, capture, keyword, semantic, then analytics is the priority order within creation time.
- A worker renews its lease only while making progress and writes bounded checkpoints at idempotent boundaries. The lease duration is 60 seconds and is renewed before half of it elapses.
- On startup, expired `processing` jobs return to `queued` when retries remain; otherwise they become `failed` with a stable interruption code.
- Completion verifies that the bookmark is non-deleted, the target version is current where required, and the index revision is still valid.
- Automatic retries use delays of 1, 5, and 30 seconds for attempts one through three. After exhaustion, state remains `failed` until an explicit manual retry resets the budget for one new three-attempt cycle.
- Duplicate completion/upsert is harmless. Stale jobs finish as safe superseded outcomes without changing current state.

## Delete and consent cleanup

- Soft delete retains canonical rows indefinitely and queues FTS, RocksDB, ChromaDB, and DuckDB cleanup identified by bookmark/version/revision. MVP exposes no restore or permanent-purge operation.
- Retrieval validates SQLite before returning candidates, so delayed cleanup cannot leak deleted data.
- Consent revocation pauses external semantic work and queues local RocksDB/Chroma semantic cleanup. It does not delete snapshots or keyword data.
- Regranting consent creates or resumes work under a compatible active/new revision; it does not assume old cache/vector completeness.

## Backup contract

Required backup contains:

- A transactionally consistent SQLite backup.
- Managed raw/extracted snapshot files referenced by SQLite.
- Manifest with application version, schema version, backup time, canonical paths relative to the archive, file sizes, and checksums.

RocksDB, ChromaDB, and DuckDB are excluded from required backup. Including them as optional acceleration artifacts never removes the requirement to verify they match the manifest revision before use.

## Restore and rebuild

1. Resolve source and destination; refuse to overwrite populated canonical data without a verified safety backup.
2. Validate manifest and checksums in a temporary/explicit location.
3. Open restored SQLite read-only, run integrity and foreign-key checks, and validate managed snapshot references.
4. Run versioned migrations with rollback to the safety backup on failure.
5. Start with external calls disabled until restored consent/configuration and environment key presence are reviewed.
6. Rebuild in order:

```text
current SQLite snapshots
-> SQLite FTS5 keyword index
-> RocksDB embedding cache
-> ChromaDB semantic vectors
-> DuckDB analytics projection
-> health and consistency verification
```

RocksDB/Chroma rebuild requires valid consent and a verified provider. If unavailable, restore completes for canonical/keyword use and records semantic degraded/unconfigured work still required. DuckDB failure similarly leaves canonical and retrieval workflows intact.

## Rebuild guarantees

- Explicit target and resolved path are displayed before destructive derived-store replacement.
- Build into staging/new revision; activate only after completeness, current-version, deletion, dimension, and smoke checks pass.
- Rebuild is resumable and idempotent and never changes notes, collections, tags, status, URLs, or snapshots.
- Failure preserves canonical data and the last safe active derived revision where compatible.
- Every run records environment, revision, counts, checksum, duration, error code, and final health without raw user content.
