# Phase 20 — Release acceptance and handoff

> Plan ID: IP-20
> Status: not_started
> Execution owner: Release acceptance and handoff owner
> Dependencies: IP-01, IP-02, IP-03, IP-04, IP-05, IP-06, IP-07, IP-08, IP-09, IP-10, IP-11, IP-12, IP-13, IP-14, IP-15, IP-16, IP-17, IP-18, IP-19
> Parallel boundary: None; runs after IP-01 through IP-19 have produced their contracts and evidence
> Requirement IDs: Operational release acceptance (primary); FR-001 through FR-015 and NFR-001 through NFR-010 (supporting evidence only)
> Owned paths: `fixtures/release/phase-20/` (to-create); `tests/release/phase-20/` (to-create); `artifacts/release/phase-20/` (to-create)

## 1. Mục tiêu

- Turn the completed IP-01 through IP-19 contracts and evidence into one reproducible release decision for the declared Chrome/Edge desktop product on Linux, macOS, and Windows.
- Produce a final traceability matrix that gives every FR-001 through FR-015, every NFR-001 through NFR-010, and each operational rule an observable acceptance signal, primary owner, supporting evidence path, status, and reviewer decision. IP-20 aggregates evidence; it does not take primary ownership from the phase named by the implementation index.
- Freeze the release candidate identity and compatibility tuple: product release, extension package, Go Native Messaging host, Native Messaging manifest, protocol, SQLite schema, ranking model, browser package channel, and installer metadata. A mismatched tuple fails closed and is repairable.
- Leave a reviewer and support engineer with enough redacted evidence to reproduce the release gates from a clean checkout, identify the exact artifacts installed, understand known limitations, execute rollback, and continue support without access to user tab data.

## 2. Phạm vi

### Bao gồm

- Final acceptance of the complete local product boundary: one configurable browser command opens an extension-owned focused search surface; the extension searches the eligible open-tab projection of the current browser profile; the Go Native Messaging host owns framed local protocol, projection reconciliation, in-memory lexical indexing, deterministic ranking, health, diagnostics, and bounded SQLite persistence; the extension owns browser authority and activation.
- Release traceability for shortcut and focus, field search, live query, deterministic ranking, contextual result display, keyboard activation and dismissal, browser-event reconciliation, full-snapshot recovery, stale-result protection, runtime status, degraded persistence, profile/private isolation, reset, uninstall, accessibility, resource bounds, recovery, privacy, and six-cell compatibility.
- Release-candidate metadata and checksums for the Chrome Web Store package, Edge Add-ons package, Linux host installer and binary, macOS host installer and binary, Windows host installer and binary, browser-specific Native Messaging manifests, release configuration, and redacted evidence bundle. Each entry records a stable artifact ID, exact path, byte size, SHA-256, build commit, signing/provenance record, and compatibility tuple.
- Reproducible release gates: clean checkout, pinned commit, deterministic fixture seed, artifact hash verification, protocol/schema/ranking compatibility, six browser/OS matrix results, performance, accessibility, privacy, recovery, install/update/repair/reset/uninstall, documentation consistency, and reviewer sign-off.
- Rollback and handoff records covering extension/host downgrade order, Native Messaging registration repair, migration safety, evidence retention, defect ownership, support diagnostics, and the final release or no-release decision.

### Ngoài phạm vi

- Implementing or changing extension, host, protocol, storage, ranking, browser adapter, UI, packaging, installer, privacy, or diagnostics behavior. A failed gate is routed to its owning phase; IP-20 does not weaken a threshold or add a workaround.
- Supporting browsers or operating systems outside Chrome and Edge desktop on Linux, macOS, and Windows, or treating an exploratory environment as release evidence.
- Collecting page contents, cookies, local storage, network-interception data, whole browsing history, cloud copies, or any new data class. The release bundle contains synthetic IDs, counts, timings, versions, and redacted diagnostics only.
- Adding a network service, loopback listener, permanent background process assumption, broad browser permissions, or cross-profile data flow.
- Claiming primary ownership of any FR/NFR or operational implementation rule owned by IP-03 through IP-19. IP-20 owns release aggregation, gate disposition, compatibility freeze, rollback readiness, and handoff completeness.

## 3. Điều kiện tiên quyết

- IP-01 through IP-19 are merged at known commits and their phase contracts are immutable for this candidate. The implementation index remains the authority for primary requirement ownership and dependency history.
- IP-19 has produced one machine-readable row for each required matrix cell (Chrome and Edge on Linux, macOS, and Windows), with PASS, FAIL, or BLOCKED, exact environment identity, commands, fixture hash, package hashes, evidence paths, and a reason plus owner for every BLOCKED row.
- The release candidate exposes product, extension, host, protocol, SQLite schema, and ranking model versions. The candidate also exposes the exact extension IDs used in browser-specific `allowed_origins`; the host refuses unknown origins, unsupported protocol versions, profile mismatches, and invalid frames.
- IP-18 has supplied reproducible package and installer outputs, Native Messaging manifests, integrity/signature records, install/upgrade/repair/rollback/uninstall behavior, and the ownership manifest used to prove cleanup. A missing artifact or unverifiable provenance is a release blocker.
- IP-16 and IP-17 have supplied redacted diagnostic exports, health transitions, permission/origin inspection, retention findings, and privacy review. IP-08 has supplied schema migration and degraded-persistence evidence; IP-07 has supplied protocol/error/limit evidence; IP-11 has supplied ranking model and deterministic-order evidence.
- The release environment has the required tools (`git`, `python3`, `sha256sum` or an equivalent SHA-256 implementation) and access to the pinned repository commit and artifact store. No user browser profile is used for release evidence.
- Any implementation source path that exists when this phase is executed is reread and substituted for the logical `to-create` paths below. The plan must not claim that a not-yet-created file already exists.

## 4. Đầu ra cần bàn giao

- `fixtures/release/phase-20/release-candidate.json` (to-create): immutable candidate metadata, source commit, release version, build inputs, supported browser/OS matrix, extension IDs, host names, protocol/schema/ranking versions, fixture seed, and release owner.
- `fixtures/release/phase-20/requirement-map.json` (to-create): FR/NFR/operational traceability rows with primary owner, supporting phases, acceptance signal, evidence references, gate status, defect/exception reference, and reviewer decision. It rejects duplicate primary owners and missing requirement IDs.
- `fixtures/release/phase-20/compatibility-policy.json` (to-create): compatibility tuple and fail-closed rules for extension/host/browser/package/protocol/schema/ranking/origin combinations, including allowed upgrade and rollback directions.
- `fixtures/release/phase-20/artifact-manifest.json` (to-create): artifact ID, kind, path, version, build commit, byte count, SHA-256, signature/provenance reference, supported browser/OS, protocol/schema/ranking tuple, and expected ownership. The manifest covers extension packages, host binaries/installers, Native Messaging manifests, and redacted evidence.
- `tests/release/phase-20/validate-release` (to-create): a deterministic validator that checks the candidate, requirement map, compatibility policy, artifact hashes, evidence references, matrix rows, release gates, exclusions, and sign-off records from a clean checkout.
- `tests/release/phase-20/replay-release` (to-create): a clean-checkout replay wrapper that verifies the pinned source commit, downloads or receives the declared artifacts, checks SHA-256 values, runs the release validator, and emits the same machine-readable gate result without reading real browser data.
- `tests/release/phase-20/rollback-drill` (to-create): a disposable-profile and disposable-install rehearsal that verifies downgrade order, registration repair, compatible migration handling, preservation of unrelated browser state, and return to the prior healthy candidate.
- `artifacts/release/phase-20/traceability.json` (to-create): final FR/NFR/operational matrix and gate ledger, with links to IP-19 cell reports and supporting evidence from IP-03 through IP-18.
- `artifacts/release/phase-20/artifact-manifest.json` and `artifacts/release/phase-20/SHA256SUMS` (to-create): copied release manifest and one checksum line per release artifact and evidence index, generated from the exact candidate.
- `artifacts/release/phase-20/release-report.json` (to-create): release decision, gate statuses, artifact IDs/hashes, compatibility result, known limitations, rollback result, unresolved defect count, exceptions, reviewers, timestamps, and handoff references.
- `artifacts/release/phase-20/handoff/` (to-create): redacted replay guide, support diagnostic guide, compatibility/version table, rollback runbook, ownership manifest, reviewer sign-off records, and a list of retained evidence with expiry.

## 5. Skill và tài liệu áp dụng

- Skill tags:
  - `release` — freezes versions and provenance, checks artifacts/checksums, validates lifecycle readiness, records blockers, and prevents publication with an unresolved version conflict or required gate failure.
  - `change-review` — turns gate results into an explicit release/handoff recommendation, separates blocking findings from accepted residual risk, and records responsible owners.
  - `code-review` — reviews the release-facing implementation diff and generated metadata for unintended scope, compatibility, security, regression, and test adequacy before handoff.
  - `testing` — defines observable success, boundary, invalid, failure, and replay coverage for the validator, compatibility tuple, rollback drill, and final evidence ledger.
  - `git-workflow` — keeps the candidate pinned to a clean commit, verifies path scope and provenance, and makes the release decision reproducible without destructive Git operations.
  - `documentation` — makes the release report, replay commands, limitations, rollback instructions, and handoff records understandable to a reviewer and support engineer.
  - `task-planning` — sequences evidence collection, artifact freeze, validation, rollback rehearsal, sign-off, and handoff into independently checkable steps.
- Canonical documents in `docs/`: [product boundary](../refactor/README.md), [requirements](../refactor/requirements.md), [architecture and ownership](../refactor/architecture.md), [runtime protocol](../refactor/runtime-protocol.md), [packaging and operations](../refactor/packaging-and-operations.md), [persistence and lifecycle](../refactor/persistence-and-lifecycle.md), [domain and privacy](../refactor/domain-and-privacy.md), [search and ranking](../refactor/search-and-ranking.md), [user experience](../refactor/user-experience.md), and [verification and acceptance](../refactor/verification-and-acceptance.md). These documents define the contract; this phase records whether the candidate proves it.
- Cross-phase inputs: IP-01 supplies requirement and fixture IDs; IP-03 through IP-05 supply extension/browser/projection evidence; IP-06 through IP-09 supply host/protocol/storage/recovery evidence; IP-10 through IP-12 supply normalization/ranking/query evidence; IP-13 through IP-16 supply UX/activation/configuration/diagnostics evidence; IP-17 and IP-18 supply privacy/package evidence; IP-19 supplies the six-cell matrix and cross-platform evidence.
- Global conventions: synthetic data only; stable IDs; explicit PASS, FAIL, or BLOCKED statuses; SHA-256 checksums; monotonic timestamps for measured behavior; redacted diagnostics; fail-closed compatibility; bounded local data; no implicit acceptance from a missing row or command exit code alone.

## 6. Công việc triển khai

### 6.1 Freeze candidate identity and compatibility

- [ ] Create `release-candidate.json` with `release_version`, `release_commit`, `source_tree_hash`, `extension_version`, `host_version`, `protocol_major`, `protocol_minor`, `schema_version`, `ranking_model_version`, `fixture_seed`, browser package IDs, Native Messaging host name, supported browser/OS matrix, and UTC generation time.
- [ ] Require the candidate identity to include the exact Chrome and Edge extension IDs and the expected `allowed_origins` for each host manifest. A host must fail closed on an extension ID, browser family, profile identity, protocol major, schema version, or ranking model mismatch; a compatible additive protocol revision must be explicitly listed rather than inferred.
- [ ] Create `compatibility-policy.json` with a matrix for every extension package × host installer × browser family × OS cell. Each row declares minimum/maximum supported browser versions, host/platform, protocol major/minor range, schema migration range, ranking model version, origin allowlist, and whether upgrade, repair, or rollback is permitted.
- [ ] Reject a candidate when extension and host versions are not the declared compatible pair, when the host manifest is not generated from the candidate configuration, when a package hash differs from the artifact manifest, or when a required matrix cell has no evidence row. Record the exact mismatch and repair owner.

### 6.2 Build the artifact and checksum manifest

- [ ] Create `artifact-manifest.json` with one row for each candidate artifact: Chrome extension package, Edge extension package, Linux host binary and installer, macOS host binary and installer, Windows host binary and installer, browser-specific Native Messaging manifests, release configuration, evidence index, and release report. Do not mark a generated artifact accepted until its bytes, path, version, source commit, and SHA-256 match the candidate.
- [ ] Generate `SHA256SUMS` from the final bytes and verify it in a clean checkout with `sha256sum --check`. Where a platform lacks `sha256sum`, use an equivalent tool and record its version and command in the report.
- [ ] Record signing or provenance evidence without putting private signing material in the repository. The report stores signer/key identifier, build environment identifier, provenance reference, and verification result; a missing signature is a declared policy decision, not an unrecorded assumption.
- [ ] Check artifact ownership and package contents against IP-17 and IP-18: exact extension permissions, exact Native Messaging origins, restrictive host file permissions, no unexpected symlinks/path traversal, no extra data source, and no network or loopback requirement in the normal path.
- [ ] Include an artifact-to-cell map so each of the six IP-19 rows names the exact extension package, host installer/binary, Native Messaging manifest, source commit, fixture hash, and checksum used. Replaying another artifact under the same cell ID is invalid.

### 6.3 Generate the final FR/NFR/operational traceability matrix

- [ ] Generate one row for each FR-001 through FR-015 and NFR-001 through NFR-010. Each row contains `primary_owner` from the implementation index, `supporting_phases`, `acceptance_signal`, `evidence_ids`, `evidence_paths`, `status`, `blocking_defect_ids`, and reviewer disposition. IP-20 is the primary owner only for release aggregation; it must not overwrite the mapped phase owner.
- [ ] Use the following release-facing acceptance signals and primary owners:

| Requirement | Primary owner | IP-20 supporting evidence and final signal |
|---|---|---|
| FR-001 | IP-03 | Six-cell command/settings and focused-surface evidence; the declared command appears and invokes the surface. |
| FR-002 | IP-13 | Keyboard focus trace; input is focused before text is accepted without pointer use. |
| FR-003 | IP-12 | Field-query fixtures and result reports; current-profile eligible tab fields produce expected matches only. |
| FR-004 | IP-12 | Live-query revision trace; each accepted query revision maps to its rendered result revision. |
| FR-005 | IP-11 | Repeated ranking checksum and score/explanation evidence; same projection, query, timestamp, and model produce identical order and scores. |
| FR-006 | IP-13 | Duplicate-title/context captures; title, domain/URL context, window/group context, and pinned state remain distinguishable. |
| FR-007 | IP-14 | Keyboard activation acknowledgement; the confirmed highlighted result activates the expected tab and window. |
| FR-008 | IP-13 | Dismissal trace; Escape closes the surface without changing the selected browser tab. |
| FR-009 | IP-04 | Event-reconciliation evidence; create/update/move/group/pin/activate/window/remove sequence converges without duplicates. |
| FR-010 | IP-09 | Missed-event/reconnect evidence; a full snapshot converges indexed eligible IDs to browser-visible eligible IDs. |
| FR-011 | IP-14 | Activation-race evidence; a removed or changed result never activates a different tab silently. |
| FR-012 | IP-16 | Health/freshness evidence; host and index states are visible, actionable, and diagnosable. |
| FR-013 | IP-12 | Persistence-degraded evidence; lexical search and valid activation remain available when optional persistence fails. |
| FR-014 | IP-17 | Two-profile/private-context evidence; settings, projections, and activation do not cross profile boundaries. |
| FR-015 | IP-15 | Reset/uninstall ownership evidence; InfoBoard-owned data is removed without changing tabs or unrelated browser data. |
| NFR-001 | IP-13 | Six-cell performance reports; shortcut-to-focused-input p95 is at most 100 ms on declared reference hardware. |
| NFR-002 | IP-12 | Six-cell performance reports; 1,000-tab/64-character query-to-render p95 is at most 50 ms. |
| NFR-003 | IP-14 | Six-cell performance reports; selection-to-activation p95 is at most 100 ms excluding browser scheduling. |
| NFR-004 | IP-11 | Determinism fixture and ordered-ID/score checksums match across repeated runs. |
| NFR-005 | IP-09 | Snapshot convergence reports; indexed IDs equal browser-visible eligible IDs after full synchronization. |
| NFR-006 | IP-16 | Crash/reconnect/rebuild reports; normal host failure reaches healthy state without reinstall. |
| NFR-007 | IP-17 | Permission/runtime inspection; no network, page-content fetch, or non-local inference occurs on the hot path. |
| NFR-008 | IP-13 | Accessibility reports; keyboard, focus, contrast, reduced-motion, semantic labels, and text scaling pass. |
| NFR-009 | IP-17 | Privacy reports; excluded data sources are not requested, collected, persisted, or transmitted. |
| NFR-010 | IP-19 | Six-row compatibility report; Chrome and Edge each pass Linux, macOS, and Windows or have a formally accepted exception. |

- [ ] Add operational rows for bounded protocol timeouts and typed failures (IP-07), transactional/versioned schema changes (IP-08), diagnosable redacted transitions (IP-16), permission/retention enforcement (IP-17), idempotent installer/updater/repair/rollback/uninstaller behavior (IP-18), and release acceptance/handoff (IP-20). Every row has an observable signal and an evidence path.
- [ ] Require `status` to be exactly `PASS`, `FAIL`, or `BLOCKED`. A `BLOCKED` row names the missing capability, owner, attempted command, decision date, and whether the block is release-blocking. A `FAIL` row names a defect and owning phase. No blank, “not run,” or aggregate-only status is accepted.

### 6.4 Apply release gates and defect disposition

- [ ] Build a gate ledger containing: source reproducibility, artifact integrity, version/protocol/schema/ranking compatibility, requirement traceability, six-cell browser compatibility, performance, accessibility, privacy/security, recovery, install/upgrade/repair/reset/uninstall, documentation consistency, rollback rehearsal, and handoff completeness.
- [ ] Mark each gate PASS only when its observable evidence, artifact hashes, and environment identity are present. Mark FAIL for any unmet contract or unresolved defect. Mark BLOCKED for unavailable evidence with a named owner; a blocked required gate prevents release unless a documented exception is explicitly approved by the release reviewer and policy permits exceptions.
- [ ] Treat every unresolved in-scope correctness, security, privacy, compatibility, lifecycle, or data-boundary defect as release-blocking. A defect is resolved only when its owning phase supplies a fix, rerun evidence, and updated hash/traceability references. Cosmetic or exploratory findings are recorded separately and cannot mask a required failure.
- [ ] Reconcile the final report with IP-19: no matrix cell may reference a different artifact, fixture seed, source commit, protocol version, schema version, or ranking model than the candidate manifest. Reconcile release notes and support text with the canonical refactor documents; terminology or behavior drift is a documentation gate failure.

### 6.5 Rehearse rollback and recovery

- [ ] Run `./tests/release/phase-20/rollback-drill --candidate <id> --previous <id> --profile-fixture fixtures/release/phase-20/profile-fixture.json --artifact-manifest fixtures/release/phase-20/artifact-manifest.json --output artifacts/release/phase-20/rollback.json` on a disposable browser profile and disposable host installation for each OS class.
- [ ] Verify rollback order: stop the candidate host session, preserve or transactionally downgrade only a compatible schema, install the previous extension/host pair, restore the previous browser-specific Native Messaging manifest and exact origins, run hello/health/snapshot smoke checks, and return to a healthy projection without closing tabs or changing unrelated browser state.
- [ ] Verify incompatible downgrade fails closed before tab data is accepted, reports a repairable state, leaves the candidate database recoverable, and provides the documented forward-repair path. Never delete the database or broaden permissions to force a downgrade.
- [ ] Record rollback artifact hashes, commands, before/after ownership manifests, health transitions, projection revision, result checksum, process state, and any migration/quarantine action. A successful rollback is evidence; it does not authorize release when another gate fails.

### 6.6 Complete reviewer sign-off and handoff

- [ ] Create a reviewer record with release candidate ID, source commit, artifact-manifest SHA-256, gate ledger SHA-256, requirement-map SHA-256, compatibility result, rollback result, unresolved in-scope defect count, accepted exceptions, reviewer names/roles, decision (`APPROVE`, `REJECT`, or `APPROVE_WITH_EXCEPTION`), timestamp, and next action.
- [ ] Require independent review from release/change-review, implementation/code-review, testing/verification, security/privacy, and product/operations roles. Each reviewer records inspected evidence paths, findings by severity, and explicit disposition; no self-approval by the release builder satisfies the independent review requirement.
- [ ] Produce `handoff/README.txt` with the exact clean-checkout replay command, artifact retrieval identifiers, checksum verification command, compatibility/version table, support diagnostic collection steps, rollback command, evidence retention/expiry, known limitations, and owners. It must contain no raw tab title, URL, query, token, or secret.
- [ ] Produce a known-limitations list distinguishing contract exclusions and non-gating environmental constraints from open in-scope defects. Every limitation states user impact, affected browser/OS cell, evidence, owner, mitigation, and whether it blocks release. No unresolved in-scope defect may appear in an approved release report.
- [ ] Archive the final release report, manifest, checksums, traceability matrix, matrix reports, performance/accessibility/privacy/recovery evidence, rollback record, sign-offs, and source/provenance identifiers as one immutable handoff set. Record retention and deletion date for each artifact.

## 7. Kế hoạch commit

1. `feat(release): add candidate and compatibility manifests`
   - Create the candidate identity, compatibility tuple, browser/OS matrix mapping, version mismatch rules, and source/artifact provenance schema under `fixtures/release/phase-20/`.
   - Check with a deliberately mismatched extension/host/protocol fixture and a valid six-cell metadata fixture; mismatch fails closed and valid metadata passes structural validation.
2. `feat(release): add artifact checksums and traceability ledger`
   - Create the artifact manifest, SHA-256 generation/verification, full FR/NFR/operational requirement map, gate ledger, and evidence-reference validation under `fixtures/release/phase-20/` and `artifacts/release/phase-20/`.
   - Check that every artifact hash, requirement ID, primary owner, evidence path, status, and blocked/failed disposition is present and that a modified byte or duplicate primary owner fails validation.
3. `test(release): add clean replay and rollback drill`
   - Add the clean-checkout replay wrapper and disposable-profile rollback drill with explicit candidate/previous compatibility and ownership checks under `tests/release/phase-20/`.
   - Check valid replay and rollback end in the declared healthy state; corrupt checksum, incompatible downgrade, missing matrix row, and unresolved defect produce a non-approval result without deleting unrelated state.
4. `docs(release): add reviewer sign-off and support handoff records`
   - Generate the redacted release report, sign-off schema, support/rollback guide, known-limitations record, evidence retention map, and final handoff index under `artifacts/release/phase-20/handoff/`.
   - Check a reviewer can reproduce the release gates and identify exact artifact versions and checksums from the handoff without credentials or real browser data.

The documentation change itself is committed separately as `docs(implementation): add phase 20 release acceptance and handoff` and must contain only this phase file.

## 8. Kiểm chứng và nghiệm thu

- [ ] From a fresh directory, clone the repository and pin the candidate commit; verify the checkout is clean before any artifact or evidence operation:

      git clone <repository-url> infoboard-release
      cd infoboard-release
      git fetch --tags --prune origin
      git checkout --detach <release_commit>
      git status --short

- [ ] Verify the candidate and artifact manifests before execution:

      python3 tests/release/phase-20/validate-release --candidate fixtures/release/phase-20/release-candidate.json --requirements fixtures/release/phase-20/requirement-map.json --compatibility fixtures/release/phase-20/compatibility-policy.json --artifacts fixtures/release/phase-20/artifact-manifest.json --evidence artifacts/release/phase-20 --output artifacts/release/phase-20/validation.json
      sha256sum --check artifacts/release/phase-20/SHA256SUMS

  The validator must reject a dirty checkout, source-commit mismatch, checksum mismatch, missing evidence, malformed status, missing primary owner, unsupported tuple, unreviewed exception, unresolved in-scope defect, or unapproved limitation.
- [ ] Replay the release decision without real user data:

      ./tests/release/phase-20/replay-release --candidate fixtures/release/phase-20/release-candidate.json --artifact-manifest fixtures/release/phase-20/artifact-manifest.json --evidence artifacts/release/phase-20 --source-commit <release_commit> --fixture-seed <fixture_seed> --output artifacts/release/phase-20/replay.json

  The output must include the same candidate ID, artifact SHA-256 values, requirement statuses, gate statuses, compatibility result, and reviewer disposition as the archived report.
- [ ] Reconcile every IP-19 C01–C06 row and confirm each row references the frozen extension package, host artifact, Native Messaging manifest, source commit, protocol/schema/ranking tuple, fixture hash, environment identity, command, and evidence directory. A missing row, altered hash, or unowned BLOCKED reason prevents approval.
- [ ] Confirm the complete traceability matrix covers FR-001 through FR-015 exactly once as primary ownership according to the index, NFR-001 through NFR-010 exactly once as primary ownership according to the index, and all operational rules with their phase owners. IP-20's supporting evidence must not change those owners.
- [ ] Confirm the performance gates retain raw samples and p50/p95/p99 evidence for NFR-001 through NFR-003; projection convergence, ranking determinism, resource bounds, runtime recovery, accessibility, privacy, and compatibility each retain their own observable reports. An aggregate green result without underlying evidence is not acceptance.
- [ ] Confirm extension and host versions, protocol major/minor, SQLite schema, ranking model, browser package IDs, Native Messaging `allowed_origins`, and installer checksums agree across candidate metadata, manifests, matrix evidence, release notes, and diagnostics. Unknown or incompatible values fail closed with the documented repair path.
- [ ] Run the rollback drill for Linux, macOS, and Windows host installers with the browser package pair used by each cell. Confirm incompatible downgrade is rejected safely, compatible rollback restores health, and neither path closes tabs or changes unrelated browser state.
- [ ] Confirm install, update, repair, reset, uninstall, and repeated-operation evidence is idempotent on all six browser/OS cells; confirm ownership manifests show only InfoBoard-owned files and registration removed after uninstall.
- [ ] Confirm static permissions, runtime local-only flow, private/profile isolation, redacted diagnostics, retention, and no excluded-data collection pass review. Any unresolved high-severity security/privacy finding blocks release.
- [ ] Confirm accessibility evidence covers keyboard-only invocation, focus, semantics, announcements, contrast, reduced motion, 200% text scale, and narrow-window behavior for every matrix cell; an unavailable tool is an explicit blocked reason or approved exception, never an inferred pass.
- [ ] Confirm the final release report contains zero unresolved in-scope defects, or a `REJECT` decision. `APPROVE_WITH_EXCEPTION` is allowed only for a documented non-contract environmental exception accepted by the responsible reviewer and release policy; it cannot waive correctness, security, privacy, compatibility, lifecycle, or data-boundary failures.
- [ ] Confirm handoff records are complete, redacted, hash-linked, retained according to policy, and replayable by a fresh operator. The release gate is reproducible from a clean checkout and the final reviewer decision is explicit.

## 9. Rủi ro và quyết định còn mở

- Risk: Extension and host are versioned independently and an apparently valid installation combines incompatible protocol, schema, ranking, or origin metadata. Mitigation: freeze one compatibility tuple, hash every artifact, test mismatch fixtures, and fail closed before accepting tab data.
- Risk: Artifact provenance or checksum records drift after matrix testing. Mitigation: generate checksums from final bytes, map every matrix cell to exact hashes, make the manifest immutable, and rerun validation from a clean checkout.
- Risk: A green aggregate hides a missing browser/OS cell or a blocked accessibility capability. Mitigation: require six explicit rows, per-cell PASS/FAIL/BLOCKED status, owner and decision date, and no implicit pass for missing evidence.
- Risk: A rollback damages a schema, profile, tab projection, or unrelated browser state. Mitigation: use disposable installations, preserve recoverable state, permit only declared schema directions, verify ownership manifests, and rehearse each OS class before approval.
- Risk: Release evidence leaks tab metadata or credentials. Mitigation: synthetic fixtures, redacted diagnostics, no real profile access, checksum/provenance references instead of secrets, and a hard validator failure on sensitive fields.
- Risk: A reviewer cannot reproduce a decision because an artifact store, browser build, or machine is unavailable. Mitigation: record retrieval identifiers, exact commands, environment identity, fixture and artifact hashes, retention dates, and a precise BLOCKED or exception record.
- Risk: A later documentation edit describes behavior or versions that differ from the candidate. Mitigation: compare release notes, handoff text, manifests, diagnostics, and canonical refactor documents during the documentation gate.
- Decision: IP-20 does not loosen a product requirement or convert a blocked required gate into a pass. The release reviewer may approve only a documented non-contract environmental exception permitted by release policy; all in-scope defects remain blocking.
- Decision: The candidate's product, extension, host, protocol, schema, and ranking versions are immutable after sign-off. Any change requires a new candidate ID, fresh checksums, affected matrix reruns, and new sign-offs.
- Open decision: exact release version, source commit, artifact storage identifiers, signer identity, reference-machine IDs, and reviewer names are populated only when the implementation candidate exists. Missing values block release acceptance; they do not alter this contract.

## 10. References ngoài `docs/`

- Skill: [release](../../../.agent/skills/common/delivery/release/SKILL.md)
- Skill: [change-review](../../../.agent/skills/common/delivery/change-review/SKILL.md)
- Skill: [code-review](../../../.agent/skills/common/engineering/code-review/SKILL.md)
- Skill: [testing](../../../.agent/skills/common/engineering/testing/SKILL.md)
- Skill: [git-workflow](../../../.agent/skills/common/engineering/git-workflow/SKILL.md)
- Skill: [documentation](../../../.agent/skills/common/engineering/documentation/SKILL.md)
- Skill: [task-planning](../../../.agent/skills/common/foundation/task-planning/SKILL.md)
- Project context: [CONTEXT.md](../../../CONTEXT.md)
- Source/config/test paths outside docs (to-create): `extension/`, `host/`, `packaging/`, `fixtures/release/phase-20/`, `tests/release/phase-20/`, `artifacts/release/phase-20/`.
- Fixture/tool/artifact references outside docs (to-create): `fixtures/release/phase-20/release-candidate.json`, `fixtures/release/phase-20/requirement-map.json`, `fixtures/release/phase-20/compatibility-policy.json`, `fixtures/release/phase-20/artifact-manifest.json`, `tests/release/phase-20/validate-release`, `tests/release/phase-20/replay-release`, `tests/release/phase-20/rollback-drill`, `artifacts/release/phase-20/SHA256SUMS`, `artifacts/release/phase-20/traceability.json`, `artifacts/release/phase-20/release-report.json`, and `artifacts/release/phase-20/handoff/`.
