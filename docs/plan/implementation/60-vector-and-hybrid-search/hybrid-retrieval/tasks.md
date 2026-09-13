# Semantic and hybrid retrieval: tasks

This is the sole authoritative checklist for VEC-HYBRID.

## VEC-HYBRID-001 — Implement semantic query execution

- [ ] **VEC-HYBRID-001: Implement semantic query execution**
- Outcome: Implement semantic query execution.
- Prerequisites: RET-KEYWORD, SEM-CONFIG, VEC-CHROMA.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: search orchestrator, fusion/ranking module, `/api/v1/search`, UI mode controls, relevance tests.
- Actions:
  1. Gate configuration/consent, embed bounded query text, retrieve revision-compatible candidates, and apply canonical/shared filters.
  2. Keep consent, revision, canonical-validation, and redaction checks at the final use boundary.
  3. Add success, incompatibility, outage, and recovery tests with the implementation.
- Migration/compatibility: do not reuse local-model vectors or incompatible revisions; preserve keyword behavior.
- Failure recovery: derived failure degrades or retries safely and never rolls back canonical content.
- Verification: Unconfigured/no-consent, filter, stale/deleted, timeout, and query-limit tests.
- Complete when: implementation and tests pass and [evidence](execution.md#vec-hybrid-001) is reviewed.
- Commit boundary: one to three scoped commits.

## VEC-HYBRID-002 — Implement deterministic hybrid rank fusion

- [ ] **VEC-HYBRID-002: Implement deterministic hybrid rank fusion**
- Outcome: Implement deterministic hybrid rank fusion.
- Prerequisites: RET-KEYWORD, SEM-CONFIG, VEC-CHROMA.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: search orchestrator, fusion/ranking module, `/api/v1/search`, UI mode controls, relevance tests.
- Actions:
  1. Fuse bounded keyword/vector candidates, deduplicate by bookmark, define ties, preserve excerpts/context, and paginate stable output.
  2. Keep consent, revision, canonical-validation, and redaction checks at the final use boundary.
  3. Add success, incompatibility, outage, and recovery tests with the implementation.
- Migration/compatibility: do not reuse local-model vectors or incompatible revisions; preserve keyword behavior.
- Failure recovery: derived failure degrades or retries safely and never rolls back canonical content.
- Verification: Golden ranking, ties, duplicate chunks, repeated query, cursor, and filter tests.
- Complete when: implementation and tests pass and [evidence](execution.md#vec-hybrid-002) is reviewed.
- Commit boundary: one to three scoped commits.

## VEC-HYBRID-003 — Project explicit degradation and fallback

- [ ] **VEC-HYBRID-003: Project explicit degradation and fallback**
- Outcome: Project explicit degradation and fallback.
- Prerequisites: RET-KEYWORD, SEM-CONFIG, VEC-CHROMA.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: search orchestrator, fusion/ranking module, `/api/v1/search`, UI mode controls, relevance tests.
- Actions:
  1. Return requested/actual mode and component degradation; fall back to keyword on provider/vector failure without blocking list/detail/organization.
  2. Keep consent, revision, canonical-validation, and redaction checks at the final use boundary.
  3. Add success, incompatibility, outage, and recovery tests with the implementation.
- Migration/compatibility: do not reuse local-model vectors or incompatible revisions; preserve keyword behavior.
- Failure recovery: derived failure degrades or retries safely and never rolls back canonical content.
- Verification: Provider and Chroma outage, partial candidate, API/UI state, and keyword parity tests.
- Complete when: implementation and tests pass and [evidence](execution.md#vec-hybrid-003) is reviewed.
- Commit boundary: one to three scoped commits.
