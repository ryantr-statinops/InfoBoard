# Provider configuration and consent: tasks

This is the sole authoritative checklist for SEM-CONFIG.

## SEM-CONFIG-001 — Implement environment-only secret and settings projection

- [ ] **SEM-CONFIG-001: Implement environment-only secret and settings projection**
- Outcome: Implement environment-only secret and settings projection.
- Prerequisites: API-CANONICAL, RET-KEYWORD.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: `app/semantic.py`, settings/provider services, `/api/v1/settings/semantic`, security and contract tests.
- Actions:
  1. Read only INFOBOARD_SEMANTIC_API_KEY; persist endpoint/model/timeout/consent metadata; expose key presence without value.
  2. Keep consent, revision, canonical-validation, and redaction checks at the final use boundary.
  3. Add success, incompatibility, outage, and recovery tests with the implementation.
- Migration/compatibility: do not reuse local-model vectors or incompatible revisions; preserve keyword behavior.
- Failure recovery: derived failure degrades or retries safely and never rolls back canonical content.
- Verification: Configuration precedence, missing-key, redaction, unknown-field, and restart tests.
- Complete when: implementation and tests pass and [evidence](execution.md#sem-config-001) is reviewed.
- Commit boundary: one to three scoped commits.

## SEM-CONFIG-002 — Validate endpoint and verification boundaries

- [ ] **SEM-CONFIG-002: Validate endpoint and verification boundaries**
- Outcome: Validate endpoint and verification boundaries.
- Prerequisites: API-CANONICAL, RET-KEYWORD.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: `app/semantic.py`, settings/provider services, `/api/v1/settings/semantic`, security and contract tests.
- Actions:
  1. Require HTTPS except all-loopback HTTP, validate resolution/connection, and verify compatibility with a synthetic non-user payload.
  2. Keep consent, revision, canonical-validation, and redaction checks at the final use boundary.
  3. Add success, incompatibility, outage, and recovery tests with the implementation.
- Migration/compatibility: do not reuse local-model vectors or incompatible revisions; preserve keyword behavior.
- Failure recovery: derived failure degrades or retries safely and never rolls back canonical content.
- Verification: Remote HTTP rejection, loopback allowance, rebinding, timeout, payload, and no-consent verification tests.
- Complete when: implementation and tests pass and [evidence](execution.md#sem-config-002) is reviewed.
- Commit boundary: one to three scoped commits.

## SEM-CONFIG-003 — Implement explicit consent and revocation

- [ ] **SEM-CONFIG-003: Implement explicit consent and revocation**
- Outcome: Implement explicit consent and revocation.
- Prerequisites: API-CANONICAL, RET-KEYWORD.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: `app/semantic.py`, settings/provider services, `/api/v1/settings/semantic`, security and contract tests.
- Actions:
  1. Persist disclosure version and consent separately from setup; gate all snapshot/query transfer; revoke synchronously and queue local cleanup.
  2. Keep consent, revision, canonical-validation, and redaction checks at the final use boundary.
  3. Add success, incompatibility, outage, and recovery tests with the implementation.
- Migration/compatibility: do not reuse local-model vectors or incompatible revisions; preserve keyword behavior.
- Failure recovery: derived failure degrades or retries safely and never rolls back canonical content.
- Verification: No-transfer-before-consent, grant, revoke-during-job, restart, and cleanup tests.
- Complete when: implementation and tests pass and [evidence](execution.md#sem-config-003) is reviewed.
- Commit boundary: one to three scoped commits.
