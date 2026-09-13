# Packaging and MVP release: tasks

This is the sole authoritative checklist for RELEASE-PACKAGE.

## RELEASE-PACKAGE-001 — Lock dependencies and clean-install workflow

- [ ] **RELEASE-PACKAGE-001: Lock dependencies and clean-install workflow**
- Outcome: Lock dependencies and clean-install workflow.
- Prerequisites: RELEASE-TEST, REL-RECOVERY, REL-SECURITY.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: `pyproject.toml`, lockfile, environment example, startup/package scripts, release artifacts.
- Actions:
  1. Declare required stores, HTTP, and test dependencies with reproducible setup and safe defaults.
  2. Preserve reproducibility, redaction, canonical authority, and compatibility boundaries.
  3. Record acceptance evidence and every approved deviation.
- Migration/compatibility: supported paths must preserve user data and match released guidance.
- Failure recovery: stop release on missing evidence, unsafe migration, contract drift, or unapproved waiver.
- Verification: Clean install, import/startup, unconfigured provider, and path tests.
- Complete when: artifacts and [evidence](execution.md#release-package-001) are reviewed.
- Commit boundary: one to three scoped commits.

## RELEASE-PACKAGE-002 — Verify upgrade, backup, rollback, and rebuild

- [ ] **RELEASE-PACKAGE-002: Verify upgrade, backup, rollback, and rebuild**
- Outcome: Verify upgrade, backup, rollback, and rebuild.
- Prerequisites: RELEASE-TEST, REL-RECOVERY, REL-SECURITY.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: `pyproject.toml`, lockfile, environment example, startup/package scripts, release artifacts.
- Actions:
  1. Run legacy-to-target upgrade with backup, verify rollback, restore canonical data, and rebuild derived stores.
  2. Preserve reproducibility, redaction, canonical authority, and compatibility boundaries.
  3. Record acceptance evidence and every approved deviation.
- Migration/compatibility: supported paths must preserve user data and match released guidance.
- Failure recovery: stop release on missing evidence, unsafe migration, contract drift, or unapproved waiver.
- Verification: Representative dataset transcript, checksums, counts, and failure rollback.
- Complete when: artifacts and [evidence](execution.md#release-package-002) are reviewed.
- Commit boundary: one to three scoped commits.

## RELEASE-PACKAGE-003 — Assemble and approve MVP release evidence

- [ ] **RELEASE-PACKAGE-003: Assemble and approve MVP release evidence**
- Outcome: Assemble and approve MVP release evidence.
- Prerequisites: RELEASE-TEST, REL-RECOVERY, REL-SECURITY.
- Canonical references: [package references](references.md#canonical-contracts).
- Source areas: `pyproject.toml`, lockfile, environment example, startup/package scripts, release artifacts.
- Actions:
  1. Collect acceptance, quality, recovery, UI, performance, provider, known-limit, and compatibility evidence.
  2. Preserve reproducibility, redaction, canonical authority, and compatibility boundaries.
  3. Record acceptance evidence and every approved deviation.
- Migration/compatibility: supported paths must preserve user data and match released guidance.
- Failure recovery: stop release on missing evidence, unsafe migration, contract drift, or unapproved waiver.
- Verification: No missing requirement or unapproved waiver; redaction and reviewer approval.
- Complete when: artifacts and [evidence](execution.md#release-package-003) are reviewed.
- Commit boundary: one to three scoped commits.
