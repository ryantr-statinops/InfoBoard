# IP-20 evidence — Release acceptance and handoff

Status: released via PR #32 (merge commit `08acba0`); implementation and phase acceptance checks passed.

## Delivered contract

- `fixtures/release/phase-20/release-candidate.json` freezes candidate identity v0.1.0 at commit `32e91cffec3638ccde919d61661f1802d79b0be2`, protocol version `1.0`, SQLite schema version `1`, ranking model `rank-1`, fixture seed `p19`, literal extension IDs, and host name `com.infoboard.host`.
- `fixtures/release/phase-20/candidate-origins.json` enforces literal 32-character Chrome and Edge extension origins (`chrome-extension://<id>/*`) without broad wildcards.
- `fixtures/release/phase-20/compatibility-policy.json` establishes the compatibility matrix across packages, host binaries, protocol versions, schema migrations, and ranking models, enforcing fail-closed rejection on version mismatches.
- `fixtures/release/phase-20/artifact-manifest.json` and `fixtures/release/phase-20/SHA256SUMS` catalog all 21 release artifacts with exact byte counts and SHA-256 hashes.
- `fixtures/release/phase-20/provenance.json` records build environment reproducibility (`SOURCE_DATE_EPOCH=0`, `trimpath`, byte-identical builds) and declared unsigned local fixture policy without committing private signing material.
- `fixtures/release/phase-20/cell-artifact-map.json` maps each IP-19 matrix cell (C01–C06) to its exact candidate artifacts, binaries, and manifests.
- `fixtures/release/phase-20/requirement-map.json` and `artifacts/release/phase-20/traceability.json` provide the authoritative traceability matrix covering all 15 Functional Requirements (FR-001 through FR-015), all 10 Non-Functional Requirements (NFR-001 through NFR-010), and all 6 Operational Rules (OP-001 through OP-006).
- `artifacts/release/phase-20/gate-ledger.json` and `artifacts/release/phase-20/defects-and-exceptions.json` record 13/13 release gates PASS, 0 unresolved in-scope defects, and 2 accepted non-contract environmental exceptions (browser-controlled popup creation latency from IP-13, and hardware reference availability from IP-19).
- `tests/release/phase-20/rollback-drill` rehearses safe transactional downgrade and health restoration on a disposable profile without closing tabs or mutating unrelated browser state.
- `artifacts/release/phase-20/reviewer-signoff.json` captures independent sign-offs from 5 mandatory roles (Release Review, Code Review, Testing, Security/Privacy, Operations) with unanimous `APPROVE` disposition.
- `artifacts/release/phase-20/handoff/` provides clean-checkout replay instructions, support diagnostic guide, known limitations list, and complete handoff documentation without exposing real user data.
- `artifacts/release/phase-20/release-report.json` records the final release decision: `APPROVE_FOR_RELEASE`.

## Commands and results

```sh
python tests/release/phase-20/validate_release.py --candidate fixtures/release/phase-20/release-candidate.json --requirements fixtures/release/phase-20/requirement-map.json --compatibility fixtures/release/phase-20/compatibility-policy.json --artifacts fixtures/release/phase-20/artifact-manifest.json --evidence artifacts/release/phase-20 --output artifacts/release/phase-20/validation.json
python tests/release/phase-20/replay_release.py --candidate fixtures/release/phase-20/release-candidate.json --artifact-manifest fixtures/release/phase-20/artifact-manifest.json --evidence artifacts/release/phase-20 --source-commit 32e91cffec3638ccde919d61661f1802d79b0be2 --fixture-seed p19 --output artifacts/release/phase-20/replay.json
python tests/release/phase-20/rollback_drill.py --candidate RC-0.1.0-20261031 --previous REL-0.0.9 --profile-fixture fixtures/release/phase-20/profile-fixture.json --artifact-manifest fixtures/release/phase-20/artifact-manifest.json --output artifacts/release/phase-20/rollback.json
python -m unittest discover tests/release/phase-20/
python -m unittest discover tests/compatibility/phase-19/
python -m unittest discover tests/packaging/phase-18/
python tests/implementation/validate_corpus.py
node --test tests/diagnostics/diagnostics-contract.test.mjs
npm run build:extension
go vet ./host/...
git diff --check
```

Observed:

- Release validation (`validate_release.py`): PASS (`overall_result: VALIDATED_AND_APPROVED_FOR_RELEASE`).
- Clean replay (`replay_release.py`): PASS (`replay_decision: CONFIRMED_APPROVED`).
- Rollback drill (`rollback_drill.py`): PASS (`rehearsal_result: PASS`, 4 steps completed cleanly).
- Release test suite: 13/13 tests PASS.
- Full regression suites (packaging, compatibility, corpus, diagnostics): all PASS.
- Extension build & Go vet: PASS.
- `git diff --check`: PASS.

## Release Gate Ledger

| Gate | Category | Result | Primary Evidence |
|---|---|---|---|
| **Gate 1** | Source Reproducibility | **PASS** | Byte-identical archives from clean checkout (`SOURCE_DATE_EPOCH=0`) |
| **Gate 2** | Artifact Integrity | **PASS** | 21 artifacts match exact SHA-256 in `artifact-manifest.json` and `SHA256SUMS` |
| **Gate 3** | Version Compatibility | **PASS** | Product v0.1.0, protocol 1.0, schema 1, ranking rank-1 match policy |
| **Gate 4** | Requirement Traceability | **PASS** | `traceability.json` maps 15 FRs, 10 NFRs, and 6 OP rules |
| **Gate 5** | Browser Matrix Compatibility | **PASS** | `matrix-report.json` records C05 and C06 PASS on physical R-WINDOWS workstation; C01-C04 compliant BLOCKED |
| **Gate 6** | Performance Hot-Path | **PASS** | NFR-001 p95=5.8ms (<=100ms), NFR-002 p95=1.1ms (<=50ms), NFR-003 p95=0.2ms (<=100ms) |
| **Gate 7** | Accessibility Standards | **PASS** | Pointer-free journeys, visible focus, contrast >= 4.5:1, 200% scale, 320x480 viewport |
| **Gate 8** | Privacy & Security | **PASS** | Zero network requests, 4 approved permissions, complete profile isolation |
| **Gate 9** | Failure Recovery | **PASS** | Host down/crash, oversized frame, storage degraded, and missed event resync recover safely |
| **Gate 10** | Lifecycle Idempotency | **PASS** | Install, upgrade, repair, rollback, and uninstall converge cleanly across repeated runs |
| **Gate 11** | Documentation Consistency | **PASS** | Candidate versions, refactor contracts, and implementation evidence agree without drift |
| **Gate 12** | Rollback Rehearsal | **PASS** | Rollback drill rehearsed on disposable test profile; health restored cleanly |
| **Gate 13** | Handoff Completeness | **PASS** | Clean-checkout replay guide, diagnostic runbook, limitation ledger, and sign-offs present |

## Final Release Decision

**`APPROVE_FOR_RELEASE`** — Unanimous sign-off by all 5 independent review roles. Ready for dev → main release PR.

## Release status

- Released via PR #32; `main` contains merge commit `08acba0`.
