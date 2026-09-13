# Testing and CI: tasks

This is the sole authoritative checklist for RELEASE-TEST.

## RELEASE-TEST-001

**Title:** Complete deterministic unit and integration layers

- [ ] **RELEASE-TEST-001: Complete deterministic unit and integration layers**
- Outcome: Complete deterministic unit and integration layers.
- Prerequisites: All feature packages; reliability test utilities.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: `tests/`, fixtures/fakes, CI configuration, evidence artifacts.
- Actions:
  1. Cover migrations, services, capture, jobs, stores, provider, API, security, and recovery with isolated paths and controlled fakes.
  2. Preserve reproducibility, redaction, canonical authority, and compatibility boundaries.
  3. Record acceptance evidence and every approved deviation.
- Migration/compatibility: supported paths must preserve user data and match released guidance.
- Failure recovery: stop release on missing evidence, unsafe migration, contract drift, or unapproved waiver.
- Verification: Repeated clean runs and proof that no user data is touched.
- Complete when: artifacts and [evidence](execution.md#release-test-001) are reviewed.
- Commit boundary: one to three scoped commits.

## RELEASE-TEST-002

**Title:** Add contract, browser, relevance, and performance gates

- [ ] **RELEASE-TEST-002: Add contract, browser, relevance, and performance gates**
- Outcome: Add contract, browser, relevance, and performance gates.
- Prerequisites: All feature packages; reliability test utilities.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: `tests/`, fixtures/fakes, CI configuration, evidence artifacts.
- Actions:
  1. Validate API schemas/errors, 1440/390 UI flows, retrieval golden sets, provider smoke, and performance bounds.
  2. Preserve reproducibility, redaction, canonical authority, and compatibility boundaries.
  3. Record acceptance evidence and every approved deviation.
- Migration/compatibility: supported paths must preserve user data and match released guidance.
- Failure recovery: stop release on missing evidence, unsafe migration, contract drift, or unapproved waiver.
- Verification: Contract reports, screenshots, relevance and resource reports.
- Complete when: artifacts and [evidence](execution.md#release-test-002) are reviewed.
- Commit boundary: one to three scoped commits.

## RELEASE-TEST-003

**Title:** Enforce CI and evidence publication

- [ ] **RELEASE-TEST-003: Enforce CI and evidence publication**
- Outcome: Enforce CI and evidence publication.
- Prerequisites: All feature packages; reliability test utilities.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: `tests/`, fixtures/fakes, CI configuration, evidence artifacts.
- Actions:
  1. Run Ruff and all suites from a clean checkout; retain redacted artifacts; fail on contract, migration, link, or traceability drift.
  2. Preserve reproducibility, redaction, canonical authority, and compatibility boundaries.
  3. Record acceptance evidence and every approved deviation.
- Migration/compatibility: supported paths must preserve user data and match released guidance.
- Failure recovery: stop release on missing evidence, unsafe migration, contract drift, or unapproved waiver.
- Verification: Fresh CI pass, retained artifacts, and intentional-failure proof.
- Complete when: artifacts and [evidence](execution.md#release-test-003) are reviewed.
- Commit boundary: one to three scoped commits.
