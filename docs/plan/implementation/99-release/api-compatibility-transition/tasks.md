# API compatibility transition: tasks

This is the sole authoritative checklist for RELEASE-COMPAT.

## RELEASE-COMPAT-001 — Implement behavior-preserving compatibility adapters

- [ ] **RELEASE-COMPAT-001: Implement behavior-preserving compatibility adapters**
- Outcome: Implement behavior-preserving compatibility adapters.
- Prerequisites: API-CANONICAL and all API-owning packages; removal requires one released compatibility version.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: compatibility routers, release manifest, HTTP contract tests, migration notes.
- Actions:
  1. Map supported routes to canonical services; keep no adapter-owned storage or unique behavior.
  2. Preserve reproducibility, redaction, canonical authority, and compatibility boundaries.
  3. Record acceptance evidence and every approved deviation.
- Migration/compatibility: supported paths must preserve user data and match released guidance.
- Failure recovery: stop release on missing evidence, unsafe migration, contract drift, or unapproved waiver.
- Verification: Canonical/adapter parity, IDs, errors, concurrency, and side-effect tests.
- Complete when: artifacts and [evidence](execution.md#release-compat-001) are reviewed.
- Commit boundary: one to three scoped commits.

## RELEASE-COMPAT-002 — Emit deprecation and retirement guidance

- [ ] **RELEASE-COMPAT-002: Emit deprecation and retirement guidance**
- Outcome: Emit deprecation and retirement guidance.
- Prerequisites: API-CANONICAL and all API-owning packages; removal requires one released compatibility version.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: compatibility routers, release manifest, HTTP contract tests, migration notes.
- Actions:
  1. Add Deprecation, manifest-derived Sunset, and migration Link headers; return 410 for retired writes.
  2. Preserve reproducibility, redaction, canonical authority, and compatibility boundaries.
  3. Record acceptance evidence and every approved deviation.
- Migration/compatibility: supported paths must preserve user data and match released guidance.
- Failure recovery: stop release on missing evidence, unsafe migration, contract drift, or unapproved waiver.
- Verification: Header/date/link, retired-write, safe-error, and documentation tests.
- Complete when: artifacts and [evidence](execution.md#release-compat-002) are reviewed.
- Commit boundary: one to three scoped commits.

## RELEASE-COMPAT-003 — Remove adapters only after the compatibility release

- [ ] **RELEASE-COMPAT-003: Remove adapters only after the compatibility release**
- Outcome: Remove adapters only after the compatibility release.
- Prerequisites: API-CANONICAL and all API-owning packages; removal requires one released compatibility version.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: compatibility routers, release manifest, HTTP contract tests, migration notes.
- Actions:
  1. Verify the release boundary, remove old routes, and rerun canonical API regression in the subsequent release.
  2. Preserve reproducibility, redaction, canonical authority, and compatibility boundaries.
  3. Record acceptance evidence and every approved deviation.
- Migration/compatibility: supported paths must preserve user data and match released guidance.
- Failure recovery: stop release on missing evidence, unsafe migration, contract drift, or unapproved waiver.
- Verification: Release-history evidence, old-route absence, and canonical regression.
- Complete when: artifacts and [evidence](execution.md#release-compat-003) are reviewed.
- Commit boundary: one to three scoped commits.
