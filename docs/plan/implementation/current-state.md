# Current runtime and gap analysis

**Runtime snapshot:** `f0737245f919697676bbd80b2c5afd68bde635a0` (`feat: add result fusion and analytics adapter`)

This document records observed runtime evidence and gaps against target contracts. It does not downgrade the accepted target or claim delivery from documentation.

## Baseline

| Area | Current runtime evidence | Target gap |
| --- | --- | --- |
| Application | FastAPI/Uvicorn lifespan, Jinja dashboard, static `/api/health` | No centralized settings, `/api/v1` envelopes, component health, correlation/error contract, or exact lifecycle boundary |
| SQLite | `items`, one-row `item_contents`, `chunks`, collections, notes, search history, jobs, contentless FTS | No migrations, bookmarks/capture attempts/immutable snapshots/tags/revisions/versions/checkpoints |
| Identifiers | Existing SQLite integer IDs | Already aligned with target ID decision; migration must preserve them |
| Capture | Synchronous URL fetch inside request; text/file/upload creation also enabled | Target persists bookmark/attempt first, uses bounded sequential worker, URL-only new capture, and read-only legacy compatibility |
| URL safety | Basic scheme, DNS private/loopback/link-local/reserved check, size and timeout | No normalization-v1 tracker policy, redirect/connected-address/rebinding validation, decompression/parser/port bounds |
| Worker | Synchronous simulated job transitions through extracting/chunking/embedding | No in-process background loop, leases, checkpoints, priority, bounded backoff, capture attempts, or real derived writes |
| Keyword | FTS5 rebuild and simple `MATCH`/snippet | No versioned current-snapshot index, shared filters, cursor/sort contract, safe query behavior, or health/rebuild evidence |
| Semantic | Local `sentence-transformers` adapter returning empty fallback when dependency is absent | Target requires external compatible endpoint, environment key, consent, response validation, index revisions, and real release smoke |
| RocksDB | Absent | No `rocksdict` cache, revision keys, bypass, corruption handling, cleanup, or rebuild |
| ChromaDB | Absent | No persistent vectors, namespace activation, canonical validation, fallback, cleanup, or rebuild |
| DuckDB | Optional in-memory adapter using `sqlite_scan`, otherwise SQLite count fallback | No persistent projection, checkpoint/watermark/checksum, accepted bounded metrics, parity, stale detection, or rebuild |
| API | Unversioned `/api/*`, raw lists/dicts, integer IDs, no optimistic concurrency | No `/api/v1`, exact schemas/errors/cursors, `If-Match`, deprecation adapters, provider/maintenance interfaces |
| UI | Basic server-rendered form/list/search | No accepted IA, detail/context, collections/tags/notes workspace, capture states, semantic setup/degradation, legacy state, or responsive evidence |
| Quality | Four tests pass and Ruff passes at the runtime snapshot | No migration, HTTP, storage, provider, security, recovery, UI, relevance, performance, or release evidence |

## Forward migration boundary

- Preserve every existing item integer ID, item content, collection/membership, note, timestamp, and soft-delete marker.
- Convert non-URL source types to `legacy_text` or `legacy_file`; expose read/search/organization/delete but no new creation or recapture.
- Convert usable `item_contents` into immutable snapshot version 1 and set the current snapshot pointer.
- Rebuild chunks and FTS; do not treat existing derived rows as authority.
- Do not reinterpret simulated `index_jobs`; record counts and enqueue target jobs.
- Keep search history only in the verified pre-migration backup.
- Abort without switching active data when canonical count/mapping checks fail.

## API transition

- Build `/api/v1/*` as canonical services and update the maintained UI to use them.
- Adapt target-compatible `/api/*` routes for one released version with deprecation/sunset/migration headers.
- Return `410 LEGACY_WRITE_REMOVED` from old text creation/upload routes because they have no canonical URL-first equivalent.
- Remove all adapters only in the subsequent release through explicit task `T20`.

## Planning implications

The forthcoming implementation workspace must begin with migration/settings boundaries, then deliver vertical slices without treating existing adapters as completed target capabilities. Baseline code is reusable where tests prove compatibility, but the target model, pipeline, provider, stores, and APIs require deliberate replacement rather than status relabeling.
