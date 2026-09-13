# Embedding pipeline and revisions: tasks

This is the sole authoritative checklist for SEM-EMBED.

## SEM-EMBED-001 — Create immutable index revision identity

- [ ] **SEM-EMBED-001: Create immutable index revision identity**
- Outcome: Create immutable index revision identity.
- Prerequisites: SEM-CONFIG, CAP-WORKER, CAP-SNAPSHOT.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: provider client, chunker, index revision repository, semantic job handler, fake-provider tests.
- Actions:
  1. Bind endpoint identity, provider/model, vector dimension, chunking revision, and schema version; activate only verified compatible revisions.
  2. Keep consent, revision, canonical-validation, and redaction checks at the final use boundary.
  3. Add success, incompatibility, outage, and recovery tests with the implementation.
- Migration/compatibility: do not reuse local-model vectors or incompatible revisions; preserve keyword behavior.
- Failure recovery: derived failure degrades or retries safely and never rolls back canonical content.
- Verification: Model/dimension/chunking change, activation, rollback, and stale-revision tests.
- Complete when: implementation and tests pass and [evidence](execution.md#sem-embed-001) is reviewed.
- Commit boundary: one to three scoped commits.

## SEM-EMBED-002 — Chunk and submit bounded snapshot content

- [ ] **SEM-EMBED-002: Chunk and submit bounded snapshot content**
- Outcome: Chunk and submit bounded snapshot content.
- Prerequisites: SEM-CONFIG, CAP-WORKER, CAP-SNAPSHOT.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: provider client, chunker, index revision repository, semantic job handler, fake-provider tests.
- Actions:
  1. Read only current eligible snapshots, create deterministic chunk IDs, bound batches/payloads, gate consent at send time, and redact diagnostics.
  2. Keep consent, revision, canonical-validation, and redaction checks at the final use boundary.
  3. Add success, incompatibility, outage, and recovery tests with the implementation.
- Migration/compatibility: do not reuse local-model vectors or incompatible revisions; preserve keyword behavior.
- Failure recovery: derived failure degrades or retries safely and never rolls back canonical content.
- Verification: Chunk determinism, size boundaries, revocation race, deleted/stale snapshot, and retry tests.
- Complete when: implementation and tests pass and [evidence](execution.md#sem-embed-002) is reviewed.
- Commit boundary: one to three scoped commits.

## SEM-EMBED-003 — Validate provider responses and durable outcomes

- [ ] **SEM-EMBED-003: Validate provider responses and durable outcomes**
- Outcome: Validate provider responses and durable outcomes.
- Prerequisites: SEM-CONFIG, CAP-WORKER, CAP-SNAPSHOT.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: provider client, chunker, index revision repository, semantic job handler, fake-provider tests.
- Actions:
  1. Validate count, numeric vectors, dimensions, model metadata, and payload bounds before handing results to derived stores.
  2. Keep consent, revision, canonical-validation, and redaction checks at the final use boundary.
  3. Add success, incompatibility, outage, and recovery tests with the implementation.
- Migration/compatibility: do not reuse local-model vectors or incompatible revisions; preserve keyword behavior.
- Failure recovery: derived failure degrades or retries safely and never rolls back canonical content.
- Verification: Malformed JSON, wrong count/dimension/model, NaN/Inf, timeout, partial response, and exhaustion tests.
- Complete when: implementation and tests pass and [evidence](execution.md#sem-embed-003) is reviewed.
- Commit boundary: one to three scoped commits.
