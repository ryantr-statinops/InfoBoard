# RocksDB embedding cache: tasks

This is the sole authoritative checklist for CACHE-ROCKS.

## CACHE-ROCKS-001

**Title:** Define versioned cache keys and validated values

- [ ] **CACHE-ROCKS-001: Define versioned cache keys and validated values**
- Outcome: Define versioned cache keys and validated values.
- Prerequisites: SEM-EMBED.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: rocksdict adapter, cache key/value codec, semantic pipeline integration, cache tests.
- Actions:
  1. Key by content checksum, chunk identity, provider/model/dimension/chunking/schema revision; validate bounded vectors and metadata on read.
  2. Keep consent, revision, canonical-validation, and redaction checks at the final use boundary.
  3. Add success, incompatibility, outage, and recovery tests with the implementation.
- Migration/compatibility: do not reuse local-model vectors or incompatible revisions; preserve keyword behavior.
- Failure recovery: derived failure degrades or retries safely and never rolls back canonical content.
- Verification: Compatibility collision, corrupt value, dimension, schema, and serialization tests.
- Complete when: implementation and tests pass and [evidence](execution.md#cache-rocks-001) is reviewed.
- Commit boundary: one to three scoped commits.

## CACHE-ROCKS-002

**Title:** Integrate safe hit, miss, and bypass behavior

- [ ] **CACHE-ROCKS-002: Integrate safe hit, miss, and bypass behavior**
- Outcome: Integrate safe hit, miss, and bypass behavior.
- Prerequisites: SEM-EMBED.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: rocksdict adapter, cache key/value codec, semantic pipeline integration, cache tests.
- Actions:
  1. Use valid hits before provider calls, treat missing/corrupt/unavailable cache as a miss, and never block embedding progress on cache writes.
  2. Keep consent, revision, canonical-validation, and redaction checks at the final use boundary.
  3. Add success, incompatibility, outage, and recovery tests with the implementation.
- Migration/compatibility: do not reuse local-model vectors or incompatible revisions; preserve keyword behavior.
- Failure recovery: derived failure degrades or retries safely and never rolls back canonical content.
- Verification: Hit/miss, write failure, read corruption, provider-call count, and restart tests.
- Complete when: implementation and tests pass and [evidence](execution.md#cache-rocks-002) is reviewed.
- Commit boundary: one to three scoped commits.

## CACHE-ROCKS-003

**Title:** Implement revocation cleanup and rebuild semantics

- [ ] **CACHE-ROCKS-003: Implement revocation cleanup and rebuild semantics**
- Outcome: Implement revocation cleanup and rebuild semantics.
- Prerequisites: SEM-EMBED.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: rocksdict adapter, cache key/value codec, semantic pipeline integration, cache tests.
- Actions:
  1. Remove revision/provider-associated values according to policy, expose health, and allow empty-cache rebuild by normal indexing.
  2. Keep consent, revision, canonical-validation, and redaction checks at the final use boundary.
  3. Add success, incompatibility, outage, and recovery tests with the implementation.
- Migration/compatibility: do not reuse local-model vectors or incompatible revisions; preserve keyword behavior.
- Failure recovery: derived failure degrades or retries safely and never rolls back canonical content.
- Verification: Interrupted cleanup, revocation, model change, delete, and cold rebuild tests.
- Complete when: implementation and tests pass and [evidence](execution.md#cache-rocks-003) is reviewed.
- Commit boundary: one to three scoped commits.
