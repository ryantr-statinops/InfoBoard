# InfoBoard implementation plans

This directory is the implementation control plane for the current InfoBoard product contract.

InfoBoard is a Chrome/Edge desktop extension for Linux, macOS, and Windows. The extension owns browser APIs, tab observation, the focused search surface, and tab activation. A local Go Native Messaging host owns the framed protocol, current-profile projection reconciliation, in-memory lexical search, deterministic ranking, bounded SQLite persistence, health, and diagnostics.

## Source of truth

The binding product contract is in [`docs/plan/refactor/README.md`](../refactor/README.md), [`architecture.md`](../refactor/architecture.md), [`requirements.md`](../refactor/requirements.md), [`runtime-protocol.md`](../refactor/runtime-protocol.md), and [`verification-and-acceptance.md`](../refactor/verification-and-acceptance.md). If an implementation plan conflicts with those documents, update the plan before implementation begins.

## Workspace contract

- [`index.md`](index.md) is the plan inventory, dependency DAG, requirement ownership map, logical target layout, and normative branch/integration/release protocol.
- `phase-01` through `phase-20` are standalone scope and acceptance plans. Each has stable task IDs and a commit-to-task map in Section 7; `> Status:` is static kickoff metadata, not live progress.
- [`README.md`](README.md) is the sole implementation execution tracker. One coordinator updates the dashboard; phase owners report evidence and do not edit shared tracker rows.
- Implementation targets are logical `to-create` paths under `extension/`, `host/`, `packaging/`, `fixtures/`, and `tests/` until source scaffolding exists.
- Plan-file authoring has its own branch protocol. Future product implementation uses phase work branches from latest `origin/dev`, integrates to `dev` only after dependency and acceptance gates, and never commits directly to `main`.
- The coordinator opens a `dev` → `main` release PR only after release gates pass; leave it open for the user to merge.
- Do not add unrelated product areas, browser data sources, remote services, or unbounded persistence.

## Execution order

Follow the waves and dependencies in [`index.md`](index.md). Plan authors own only their assigned phase plan file. During product implementation, the coordinator assigns phase owners and updates the [`README.md`](README.md) tracker after evidence reports; agents never update shared dashboard rows. Validate the corpus after plan merges and run integration checks on `dev`.

## Product implementation execution

This dashboard is the **single source of truth** for implementation status. Phase files define scope, stable task IDs, commit slices and acceptance contracts; they are not progress trackers. Update this table only through the coordinator below.

### Release integration

- **dev → main release PR:** Not started · PR: — · latest commit: — · release checks/evidence: — · note: Leave the release PR open; only the user merges it. Agents never commit directly to `main` or merge this PR.

### Execution protocol

1. A phase owner starts from the latest `origin/dev` on a dedicated phase work branch. Implement only assigned task IDs; preserve logical `to-create` paths until source scaffolding exists.
2. Each planned commit uses the exact conventional message and task mapping in that phase's Section 7. Checks are prerequisites for each slice; never claim a future command passed while its source/fixture prerequisite is absent.
3. Phase owners send the coordinator a structured progress report. Phase owners do **not** edit this shared dashboard, including when phases run concurrently. One coordinator owns all README tracker updates to prevent write conflicts.
4. Coordinator records exact latest SHA, branch, PR reference, last executed check and evidence location, blocker/next action, and UTC update time. Notes must be short and actionable; never include raw title, URL, query, token, secret, or private-record content.
5. After phase acceptance checks pass on its work branch, status becomes `Ready for dev integration` with exact commands/results/evidence. Integrate only after every index.md dependency has merged into `dev`; run integration checks on `dev`. After those pass mark `Integrated in dev`, then `Complete` only when every phase acceptance criterion is met.
6. A real blocker moves the row to `Blocked` with blocker, owner, and next action. On resumption, record new evidence and use `In progress`. Plan-authoring commits do not count as implementation progress.
7. Product implementation branches derive from `dev`, never `main`; no phase work is committed directly to `main`. When IP-20 release gates pass, the coordinator opens a PR from `dev` to `main` and leaves it open for the user to merge manually.

**Allowed phase statuses:** `Not started`, `In progress`, `Blocked`, `Ready for dev integration`, `Integrated in dev`, `Complete`. `Complete` means phase acceptance passed and changes are integrated in `dev`; release PR state is tracked separately above.

**Agent → coordinator update:**

```text
phase: IP-XX
status: <allowed status>
owner: <agent/person>
work_branch: <branch>
pr: <URL/number or —>
latest_commit: <full SHA or —>
last_check: <exact command and result, or —>
evidence: <path/link or —>
blocker: <what/owner/next action or —>
updated_at: <UTC timestamp>
```

### Phase progress board

| Phase | Plan | Status | Owner / agent | Work branch | PR | Latest commit SHA | Last check / evidence reference | Blocker / next action | Updated (UTC) |
|---|---|---|---|---|---|---|---|---|---|
| IP-01 | [Phase 01](phase-01-contract-and-fixtures.md) | Not started | — | — | — | — | — | — | — |
| IP-02 | [Phase 02](phase-02-domain-identifiers-and-ownership.md) | Not started | — | — | — | — | — | — | — |
| IP-03 | [Phase 03](phase-03-extension-scaffold-and-command.md) | Not started | — | — | — | — | — | — | — |
| IP-04 | [Phase 04](phase-04-tab-observation-and-eligibility.md) | Not started | — | — | — | — | — | — | — |
| IP-05 | [Phase 05](phase-05-profile-projection-and-reconciliation.md) | Not started | — | — | — | — | — | — | — |
| IP-06 | [Phase 06](phase-06-host-bootstrap-and-lifecycle.md) | Not started | — | — | — | — | — | — | — |
| IP-07 | [Phase 07](phase-07-native-messaging-protocol.md) | Not started | — | — | — | — | — | — | — |
| IP-08 | [Phase 08](phase-08-sqlite-storage-and-migrations.md) | Not started | — | — | — | — | — | — | — |
| IP-09 | [Phase 09](phase-09-projection-sync-and-recovery.md) | Not started | — | — | — | — | — | — | — |
| IP-10 | [Phase 10](phase-10-lexical-normalization-and-index.md) | Not started | — | — | — | — | — | — | — |
| IP-11 | [Phase 11](phase-11-deterministic-ranking-and-explanations.md) | Not started | — | — | — | — | — | — | — |
| IP-12 | [Phase 12](phase-12-query-api-and-degraded-results.md) | Not started | — | — | — | — | — | — | — |
| IP-13 | [Phase 13](phase-13-search-surface-and-accessibility.md) | Not started | — | — | — | — | — | — | — |
| IP-14 | [Phase 14](phase-14-activation-and-stale-result-guard.md) | Not started | — | — | — | — | — | — | — |
| IP-15 | [Phase 15](phase-15-configuration-reset-and-uninstall.md) | Not started | — | — | — | — | — | — | — |
| IP-16 | [Phase 16](phase-16-diagnostics-health-and-observability.md) | Not started | — | — | — | — | — | — | — |
| IP-17 | [Phase 17](phase-17-privacy-permissions-and-profile-isolation.md) | Not started | — | — | — | — | — | — | — |
| IP-18 | [Phase 18](phase-18-packaging-and-host-registration.md) | Not started | — | — | — | — | — | — | — |
| IP-19 | [Phase 19](phase-19-cross-platform-browser-verification.md) | Not started | — | — | — | — | — | — | — |
| IP-20 | [Phase 20](phase-20-release-acceptance-and-handoff.md) | Not started | — | — | — | — | — | — | — |
