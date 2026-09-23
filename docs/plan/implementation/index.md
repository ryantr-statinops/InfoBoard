# Implementation plan index

## Product boundary

InfoBoard is a keyboard-first Chrome/Edge desktop extension for Linux, macOS, and Windows. It searches the eligible open-tab projection of the current browser profile. The extension owns browser authority and activation. A local Go Native Messaging host owns the framed local protocol, projection reconciliation, in-memory lexical index, deterministic ranking, bounded SQLite persistence, health, and diagnostics.

The hot path is local and metadata-only: no page-content indexing, no browser-history access, no cloud service, and no loopback transport. Plans must preserve profile isolation, private-context rules, minimal permissions, stale-result protection, bounded timeouts, reset semantics, and idempotent packaging operations.

Canonical contract: [`README.md`](../refactor/README.md), [`architecture.md`](../refactor/architecture.md), [`requirements.md`](../refactor/requirements.md), [`runtime-protocol.md`](../refactor/runtime-protocol.md), and [`verification-and-acceptance.md`](../refactor/verification-and-acceptance.md).

## Logical target layout

The repository currently has no implementation source tree. Until implementation creates it, the following paths are logical `to-create` targets:

```text
extension/   # to-create: Manifest V3 boundary, browser adapter, focused search surface
host/        # to-create: Go Native Messaging host, projection, index, persistence, diagnostics
packaging/   # to-create: browser package, native manifest, installer and lifecycle artifacts
fixtures/catalog.schema.json # shared fixture envelope schema (IP-01)
fixtures/catalog.json        # shared fixture catalog (IP-01)
fixtures/<phase>/            # to-create: phase-owned fixtures
tests/implementation/validate_corpus.py # present: standard-library corpus/fixture validator
tests/<phase>/                 # to-create: unit, integration, browser, accessibility and performance tests
```

IP-01 owns the exact package/module naming map. The shared [`fixture catalog`](../../../fixtures/catalog.json) and [`catalog schema`](../../../fixtures/catalog.schema.json) define nine reusable observable cases plus future checks, invariants, and one traceable acceptance seam for every later phase. The runtime roots and phase-owned test/fixture roots remain `to-create`; the shared validator is present and phase-specific acceptance suites still extend, rather than inherit as complete, the shared baseline. A phase author must reread the implementation scaffold if it appears and replace a logical path with the exact observed path before committing.

## Phase inventory and dependency DAG

| ID | File | Responsibility | Dependencies | Wave | Primary skill tags |
|---|---|---|---|---|---|
| IP-01 | [`phase-01-contract-and-fixtures.md`](phase-01-contract-and-fixtures.md) | Contract traceability, fixtures, layout, conventions, seams | canonical refactor docs | 0 | requirements-analysis, repository-onboarding, project-scoping, scope-control, task-planning, documentation, testing |
| IP-02 | [`phase-02-domain-identifiers-and-ownership.md`](phase-02-domain-identifiers-and-ownership.md) | Profile, tab identity, projection, eligibility, revisions, ownership | IP-01 | 1A | architecture-review, architecture-tradeoff, databases, secure-development |
| IP-03 | [`phase-03-extension-scaffold-and-command.md`](phase-03-extension-scaffold-and-command.md) | Manifest V3, command, focused surface, adapter seam, minimal permissions | IP-01, IP-02 | 1A | feature-delivery, lightweight-web, secure-development, testing |
| IP-04 | [`phase-04-tab-observation-and-eligibility.md`](phase-04-tab-observation-and-eligibility.md) | Browser events, eligibility, private context, denial, duplicate suppression | IP-02, IP-03 | 2A | feature-delivery, secure-development, testing, debugging |
| IP-05 | [`phase-05-profile-projection-and-reconciliation.md`](phase-05-profile-projection-and-reconciliation.md) | Snapshot, deltas, revisions, missed events, restart, profile isolation | IP-02, IP-04, IP-06, IP-07 | 3A | pipelines, databases, go, testing, debugging |
| IP-06 | [`phase-06-host-bootstrap-and-lifecycle.md`](phase-06-host-bootstrap-and-lifecycle.md) | Go host lifecycle, stdio, session, cancellation, shutdown, reconnect | IP-01, IP-02 | 2A | go, architecture-review, secure-development, testing |
| IP-07 | [`phase-07-native-messaging-protocol.md`](phase-07-native-messaging-protocol.md) | Framed protocol, messages, limits, typed errors, fail-closed behavior | IP-02, IP-06 | 2B | api-design, go, secure-development, testing |
| IP-08 | [`phase-08-sqlite-storage-and-migrations.md`](phase-08-sqlite-storage-and-migrations.md) | SQLite schema, migrations, retention, corruption, reset, degraded persistence | IP-02, IP-06 | 2B | databases, go, secure-development, testing |
| IP-09 | [`phase-09-projection-sync-and-recovery.md`](phase-09-projection-sync-and-recovery.md) | Extension-host sync, atomic snapshots, resync, rebuild, recovery | IP-04, IP-05, IP-07, IP-08 | 3B | pipelines, go, testing, debugging |
| IP-10 | [`phase-10-lexical-normalization-and-index.md`](phase-10-lexical-normalization-and-index.md) | Normalization, token bounds, fields, memory index, rebuild | IP-02, IP-05, IP-09 | 4A | databases, pipelines, go, testing |
| IP-11 | [`phase-11-deterministic-ranking-and-explanations.md`](phase-11-deterministic-ranking-and-explanations.md) | Match classes, score, tie-breaks, provenance, explanations | IP-10 | 4B | architecture-tradeoff, go, testing, documentation |
| IP-12 | [`phase-12-query-api-and-degraded-results.md`](phase-12-query-api-and-degraded-results.md) | Query contracts, limits, revisions, status, degraded results | IP-07, IP-09, IP-10, IP-11 | 5A | api-design, go, testing, secure-development |
| IP-13 | [`phase-13-search-surface-and-accessibility.md`](phase-13-search-surface-and-accessibility.md) | Focused UI, keyboard flow, states, accessibility, narrow windows | IP-03, IP-12 | 5B | lightweight-web, feature-delivery, testing, documentation |
| IP-14 | [`phase-14-activation-and-stale-result-guard.md`](phase-14-activation-and-stale-result-guard.md) | Activation, validation, revision guard, races, recency metadata | IP-05, IP-07, IP-08, IP-12, IP-13 | 6A | api-design, go, testing, debugging |
| IP-15 | [`phase-15-configuration-reset-and-uninstall.md`](phase-15-configuration-reset-and-uninstall.md) | Configuration, profile scope, reset, uninstall, safe defaults | IP-07, IP-08, IP-13, IP-14 | 6B | databases, secure-development, testing, release |
| IP-16 | [`phase-16-diagnostics-health-and-observability.md`](phase-16-diagnostics-health-and-observability.md) | Health state, transitions, redacted diagnostics, repair, retention | IP-06, IP-07, IP-08, IP-09, IP-12 | 6C | debugging, security-review, testing, change-review |
| IP-17 | [`phase-17-privacy-permissions-and-profile-isolation.md`](phase-17-privacy-permissions-and-profile-isolation.md) | Permissions, trust boundaries, private/profile separation, audit | IP-03, IP-04, IP-05, IP-07, IP-08, IP-16 | 7A | secure-development, security-review, architecture-review, testing |
| IP-18 | [`phase-18-packaging-and-host-registration.md`](phase-18-packaging-and-host-registration.md) | Browser packages, host installer, origins, integrity, lifecycle | IP-03, IP-06, IP-15, IP-17 | 7B | release, git-workflow, secure-development, architecture-review |
| IP-19 | [`phase-19-cross-platform-browser-verification.md`](phase-19-cross-platform-browser-verification.md) | Browser/platform matrix, performance, accessibility, recovery evidence | IP-13, IP-14, IP-15, IP-16, IP-17, IP-18 | 8 | testing, change-review, release, lightweight-web |
| IP-20 | [`phase-20-release-acceptance-and-handoff.md`](phase-20-release-acceptance-and-handoff.md) | Final traceability, artifacts, compatibility, rollback, handoff | IP-01..IP-19 | 9 | release, change-review, code-review, testing, git-workflow |

Execution waves: `IP-01`; then `IP-02 + IP-03`; then `IP-04 + IP-06 + IP-07 + IP-08`; then `IP-05 + IP-09 + IP-10`; then `IP-11 + IP-12`; then `IP-13 + IP-14 + IP-15 + IP-16`; then `IP-17 + IP-18`; then `IP-19`; finally `IP-20`.

## Requirement ownership

Each requirement has one primary plan owner. Supporting plans provide evidence but do not claim primary ownership.

| Requirement | Primary phase |
|---|---|
| FR-001 | IP-03 |
| FR-002 | IP-13 |
| FR-003 | IP-12 |
| FR-004 | IP-12 |
| FR-005 | IP-11 |
| FR-006 | IP-13 |
| FR-007 | IP-14 |
| FR-008 | IP-13 |
| FR-009 | IP-04 |
| FR-010 | IP-09 |
| FR-011 | IP-14 |
| FR-012 | IP-16 |
| FR-013 | IP-12 |
| FR-014 | IP-17 |
| FR-015 | IP-15 |
| NFR-001 | IP-13 |
| NFR-002 | IP-12 |
| NFR-003 | IP-14 |
| NFR-004 | IP-11 |
| NFR-005 | IP-09 |
| NFR-006 | IP-16 |
| NFR-007 | IP-17 |
| NFR-008 | IP-13 |
| NFR-009 | IP-17 |
| NFR-010 | IP-19 |

Operational ownership: IP-07 owns protocol versions and bounded timeouts; IP-08 owns transactional schema changes; IP-16 owns diagnosable transitions and redacted logs; IP-17 owns permission and retention enforcement; IP-18 owns idempotent installer, updater, repair, rollback, and uninstaller behavior.

## Phase authoring contract

Every phase file is standalone and must contain the metadata block plus exactly these headings:

1. `Mục tiêu`
2. `Phạm vi`
3. `Điều kiện tiên quyết`
4. `Đầu ra cần bàn giao`
5. `Skill và tài liệu áp dụng`
6. `Công việc triển khai`
7. `Kế hoạch commit`
8. `Kiểm chứng và nghiệm thu`
9. `Rủi ro và quyết định còn mở`
10. `References ngoài docs/`

Section 10 must link a real skill under `.agent/skills/`, `CONTEXT.md`, canonical documents under `docs/plan/refactor/`, and every relevant target outside `docs/`. Use `to-create` for paths not yet present. Each phase must state its sibling boundaries and exact future verification command or fixture contract; never replace verification with a generic test placeholder.

## Phase-plan authoring branch protocol

This protocol applies only to plan-file edits. Plan authors branch from current `dev`, edit only their assigned phase plan, validate it, and integrate the plan change into `dev`. The README execution tracker is coordinator-owned. These authoring branches do not authorize product implementation on `main`.

```bash
git switch dev
git pull --ff-only origin dev
git switch -c plan/implementation-ip-XX-<slug>
# edit only docs/plan/implementation/phase-XX-<slug>.md
git diff --check
git add docs/plan/implementation/phase-XX-<slug>.md
git commit -m "docs(implementation): add phase XX <slug>"
git push -u origin plan/implementation-ip-XX-<slug>
```

After review, integrate the single-file plan branch into `dev`, validate the corpus, and push `dev`. Stop on failed push, non-fast-forward, unexpected path, or unresolved contract conflict; never force-push.

## Product implementation and release protocol

This protocol applies to future implementation code, fixtures, packages, and tests—not authoring plan files.

1. A phase work branch starts from latest `origin/dev`, never `main`. Follow the phase Section 7 task/commit map; change only owned paths and explicitly assigned fixtures/tests.
2. Run phase acceptance checks on the work branch. Only a passing phase may become `Ready for dev integration`; report exact command/result/evidence, SHA, branch, PR, and blocker/next action to the coordinator. Agents do not edit the shared README dashboard.
3. Integrate only after every DAG dependency is integrated into `dev`. Run integration checks on `dev`; record SHA and evidence in README. `Integrated in dev` precedes `Complete`; `Complete` requires acceptance criteria pass and integration.
4. Never commit implementation directly to `main`, merge the release PR, or mark a phase complete before integration acceptance. After IP-20 release gates pass, coordinator opens a `dev` → `main` PR and leaves it open for user review and manual merge.
5. README is the single execution-status source. Never duplicate live progress status in `index.md` or phase files.

See [`README.md`](README.md) for the dashboard, allowed statuses, transition evidence, coordinator ownership, and agent report template.

## Final corpus checks

The validator runs from repository root and verifies that all twenty phases retain exactly sections 1–10; each Section 6 task has one stable phase-local ID and exactly one Section 7 commit owner; each commit has a conventional message, target paths, behavior, Section 8 command/fixture, observable result, and dependency gate; dependency metadata matches this DAG; status metadata points to README; and canonical, skill, and local links resolve. README must contain exactly twenty phase rows, valid status values, resolved phase links, all required progress fields, and a separate release-PR state. Negative probes reject duplicate IDs, unowned tasks, commits without commands or expected outcomes, missing phase rows, and invalid status values.

Repository workflow checks additionally run `git diff --check`, inspect the documentation-only path set, verify the branch/clean state, and confirm local `dev` equals `origin/dev` after push. Implementation source remains absent until product implementation begins.

## Finalized corpus

- The final workspace contains exactly 22 Markdown files directly under `docs/plan/implementation/`: `README.md`, `index.md`, and `phase-01` through `phase-20`.
- Each phase is standalone, uses the ten-section contract, names its dependencies and owned paths, links canonical refactor documents and real skills, and has a future commit/acceptance plan.
- The phase DAG is complete: authoring waves are finished through IP-20; implementation work must start from the latest `dev` tip and preserve the single-owner requirement map above.
- Final validation must run from a clean checkout, prove local links and metadata, reject excluded historical scope, verify only this directory changed for this rebuild, and compare local `dev` with `origin/dev`.
