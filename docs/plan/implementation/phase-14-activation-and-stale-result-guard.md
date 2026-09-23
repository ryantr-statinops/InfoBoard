# Phase 14 — Activation and stale-result guard

> Plan ID: IP-14
> Status: See README.md execution tracker
> Execution owner: extension browser-activation owner with Go status/recency integration owner
> Dependencies: IP-05, IP-07, IP-08, IP-12, IP-13
> Parallel boundary: IP-15 and IP-16 may work on non-overlapping paths after their own dependencies; IP-14 consumes the IP-13 selected-row action seam and must not edit its surface implementation.
> Requirement IDs: FR-007, FR-011, NFR-003 (primary); FR-012, FR-015 (supporting evidence)
> Owned paths: `extension/src/browser/activation-controller.ts` (to-create), `extension/src/browser/activation-guard.ts` (to-create), `extension/src/runtime/activation-reporter.ts` (to-create), `fixtures/activation/phase-14/` (to-create), `tests/activation/phase-14/` (to-create)

## 1. Mục tiêu

- Make `Enter` a browser-owned action: the extension, not the Go Native Messaging host, validates and activates the selected open tab and its exact window.
- Define a fail-closed activation guard over the result reference, current profile, current tab projection revision, tab identity, window identity, eligibility, and context partition. A stale or removed result MUST never cause a different tab to be selected implicitly.
- Define the observable success, stale, missing-tab, browser-error, host-unavailable, persistence-degraded, and reset/uninstall outcomes that IP-13 renders. Browser state is claimed changed only after the browser API confirms the requested exact tab/window operation.
- Keep the hot path bounded and measurable: selection-to-activation p95 MUST be `<= 100 ms`, excluding browser scheduling delays, and reporting recency MUST NOT block a confirmed browser activation.
- Supply the activation status and bounded recency handoff consumed by IP-07/IP-08 without moving browser authority, live projection authority, or SQLite ownership into this phase.

## 2. Phạm vi

- Bao gồm:
  - An extension-side activation request carrying an opaque result reference with `profile_id`, `tab_id`, `window_id`, `projection_revision`, `projection_epoch`/tab identity, `context_kind`, and the query/result sequence needed to reject superseded UI selections. The title, URL, or display text MUST NOT be used as identity.
  - A single-flight Enter path from the IP-13 selected-row action seam into `activation-controller.ts`. Repeated Enter presses while one request is pending are deduplicated by request identity; they do not launch parallel browser operations.
  - Guard order before any mutating browser call: verify the focused surface session belongs to the current profile/context; verify the result came from the current accepted query and current projection revision; verify the exact tab identity is still present and eligible; verify the exact current `window_id`; and reject missing, removed, reused, or mismatched identities without a fallback query.
  - Browser-owned activation sequence for Chrome and Edge: read the exact tab, focus the exact window, activate the exact tab, and verify returned IDs/active state. The controller MUST NOT select an active tab by title, URL, index, position, or “next result” after a guard failure.
  - Race handling between guard and browser calls. A remove, move, profile/context change, revision advance, tab-ID reuse, window close, browser shutdown, permission failure, or API rejection maps to a bounded status and keeps the surface available when activation is not confirmed. A successful window focus followed by failed tab activation is reported as partial browser-state change, not as success.
  - Activation status reporting through the IP-07 messages `activation_observed`, `activation_failed`, and `activation_ack`. The extension reports the confirmed browser outcome; the host records only bounded diagnostics/recency and never performs a browser API call.
  - Bounded recency metadata: on confirmed activation, report only profile/context identity, opaque tab identity, normalized domain when allowed, activation timestamp, source (`keyboard`), and protocol/projection metadata. Do not include query text, title, URL query/fragment, page data, or browser secrets. IP-08 enforces the per-profile count/age limits and the persistence-degraded session-only fallback.
  - User-visible feedback contracts for stale selection, missing tab, unavailable/recovering host, persistence degradation, and activation failure: explain what happened, whether browser state changed, and the bounded next action. A confirmed activation may close the search surface; an unconfirmed activation MUST keep it available for retry or a refreshed selection.
  - Reset/uninstall teardown for activation state: cancel/discard pending requests and in-memory reporter state, remove owned recency through IP-15/IP-08, and never close tabs or alter unrelated browser state.
  - Fixed browser-adapter fakes, Native Messaging fixtures, race schedules, profile/context fixtures, and an end-to-end latency harness covering the exact observable outcomes.
- Ngoài phạm vi:
  - Query parsing, result ordering, result freshness policy, or stale-response suppression; IP-10/IP-11/IP-12 own those contracts, and IP-13 owns keyboard selection/rendering. This phase consumes their revision-bound selected-result seam.
  - Tab/window/group event normalization, eligibility policy, projection epoch/revision allocation, or full-snapshot reconciliation; IP-04/IP-05 own those boundaries. The controller reads their committed current projection and never invents a revision.
  - Native Messaging framing, handshake, message schema/version negotiation, typed error literals, or transport reconnect; IP-06/IP-07 own them. This phase supplies activation payload semantics and consumes their acknowledgement.
  - SQLite schema, migrations, retention implementation, configuration, reset ownership manifest, or uninstall sequencing; IP-08/IP-15 own those boundaries. This phase MUST remain functional with session-only recency when persistence is degraded.
  - New browser permissions, private-data expansion, page evaluation, network access, cloud services, or any data source outside the current profile's eligible open-tab metadata.
  - An automatic retry against a different result, silent downgrade to an older projection, or any compatibility path that can activate an unrelated tab.

## 3. Điều kiện tiên quyết

- IP-02/IP-05 provide the canonical `profile_id`, `ContextKind`, `TabIdentity`, projection epoch/revision, eligible-tab record, and atomic committed projection. A tab ID alone is not a durable identity across removal/reuse.
- IP-07 provides the exact framed activation messages, envelope fields (`protocol`, `type`, `request_id`, `profile_id`, `projection_revision`, `payload`), bounded errors, request correlation, and `activation_ack` semantics. This phase does not rename wire literals.
- IP-08 provides the activation metadata schema, profile-scoped retention (500 records or 30 days), persistence health, reset behavior, and session-only fallback when SQLite is unavailable.
- IP-12 provides result references bound to the accepted profile, query sequence, projection revision, and ranking output. IP-13 provides the selected-row Enter action and the stale/activation-failure presentation seam without requiring this phase to own UI layout.
- Read the binding [`user-experience.md`](../refactor/user-experience.md), [`architecture.md`](../refactor/architecture.md), [`runtime-protocol.md`](../refactor/runtime-protocol.md), [`domain-and-privacy.md`](../refactor/domain-and-privacy.md), [`persistence-and-lifecycle.md`](../refactor/persistence-and-lifecycle.md), [`requirements.md`](../refactor/requirements.md), and [`verification-and-acceptance.md`](../refactor/verification-and-acceptance.md) before implementation.
- No implementation source tree is observed for these boundaries. Every path in this plan is a logical `to-create` target; if implementation exists when this phase starts, reread it and replace only the affected target with the exact observed path.
- The fixture harness MUST inject browser API calls, projection events, clock values, Native Messaging responses, tab/window races, private-context boundaries, host loss, and reset/uninstall transitions without using a real user's browser profile.

## 4. Đầu ra cần bàn giao

- [ ] `extension/src/browser/activation-guard.ts` (to-create):
  - Strict result-reference validation for profile, context, query sequence, projection epoch/revision, exact `tab_id`, exact `window_id`, and current eligible record.
  - A typed guard outcome distinguishing `valid`, `stale_selection`, `profile_mismatch`, `tab_missing`, `window_missing`, `tab_reused`, `ineligible`, and `projection_rebuilding`; no outcome contains raw URL/title/error text.
  - A no-fallback invariant: a failed guard performs zero activation calls and cannot substitute another result.
- [ ] `extension/src/browser/activation-controller.ts` (to-create):
  - Single-flight request lifecycle from selected row to browser confirmation, with monotonic request identity and cancellation on surface teardown, projection change, profile change, or reset/uninstall.
  - Exact browser sequence: `tabs.get(tab_id)`/equivalent validation, `windows.update(window_id, {focused: true})`, `tabs.update(tab_id, {active: true})`, and response identity/active-state verification. API errors are converted to bounded stages and browser-state outcomes.
  - Race-safe handling for tab removal, tab-ID reuse, window closure, move/revision advance, permission denial, worker restart, and browser shutdown between each step. No call may target a replacement tab.
  - Success closes the search surface only after exact tab/window activation is confirmed; failures leave it available and expose a retry/refresh action through the IP-13 seam.
  - Monotonic timing instrumentation from accepted Enter dispatch through confirmed activation, with browser scheduling delay separated from controller/validation/reporting time.
- [ ] `extension/src/runtime/activation-reporter.ts` (to-create):
  - Bounded serializers for `activation_observed` and `activation_failed`, profile/revision binding, request correlation, safe stage/error code, and `browser_state` (`unchanged`, `window_focused`, `tab_active`, `unknown`).
  - Non-blocking acknowledgement handling: a host or SQLite failure after confirmed browser activation changes diagnostics/recency status only and MUST NOT claim that activation failed or re-run the browser operation.
  - Teardown that cancels pending report work and emits no data after reset/uninstall ownership cleanup.
- [ ] `fixtures/activation/phase-14/` (to-create) with machine-readable inputs and exact expected observables:
  - `ACT-001-enter-exact-tab-window.json`: selected result with matching profile, context, epoch/revision, tab ID, and window ID focuses that window, activates that exact tab, reports observed success, and closes the surface.
  - `ACT-002-profile-or-context-mismatch.json`: a result from another profile/private partition performs zero browser activation calls and returns a bounded mismatch/refresh state.
  - `ACT-003-revision-mismatch.json`: current revision differs from the result revision; no browser call occurs, the surface shows stale selection, and a refresh/requery action is available.
  - `ACT-004-tab-removed-before-enter.json`: the selected tab is absent before validation; no fallback result is activated and the user sees a bounded missing/stale failure.
  - `ACT-005-tab-removed-between-focus-and-activate.json`: window focus may succeed, tab activation fails because the exact tab disappeared; outcome is partial browser-state change with no success claim and no substitute tab.
  - `ACT-006-tab-id-reused.json`: the browser reuses a numeric tab ID under a new tab identity/revision; the old result is rejected and the new tab is never activated by the old request.
  - `ACT-007-multi-window-focus.json`: two eligible tabs have similar display labels; Enter focuses and activates the selected result's exact window/tab pair, not the currently focused window.
  - `ACT-008-duplicate-enter-and-reconnect.json`: repeated Enter and host disconnect produce one browser operation; confirmed activation remains success while reporter status is unavailable/recovering.
  - `ACT-009-browser-failure-and-retry.json`: permission/API/shutdown failures expose safe stage and next action, preserve the surface, and retry only the same explicitly selected reference after a fresh validation.
  - `ACT-010-recency-retention-and-degraded-storage.json`: confirmed activations carry only bounded recency fields, prune at IP-08 limits, and continue with session-only recency when SQLite is degraded; failed activations create no recency record.
  - `ACT-011-reset-uninstall-cleanup.json`: teardown removes owned pending/recency state without closing tabs or changing unrelated browser/profile data.
  - `ACT-012-selection-to-activation-100ms.json`: fixed projection/result references and fake immediate browser calls measure 20 warm-up and 200 iterations, report p50/p95/p99, and fail when controller-path p95 exceeds 100 ms.
- [ ] `tests/activation/phase-14/` (to-create) covering guard precedence, exact ID equality, profile/context isolation, revision fencing, tab/window races, single-flight behavior, status acknowledgement, bounded recency, persistence degradation, teardown, and browser-state claims.
- [ ] A benchmark command under `tests/activation/phase-14/cmd/selection-to-activation-benchmark/` (to-create) that consumes `ACT-012`, records environment and browser-scheduling exclusions, reports p50/p95/p99 plus activation-call count and status counts, and exits non-zero when p95 exceeds 100 ms or any stale fixture performs an unintended call.

## 5. Skill và tài liệu áp dụng

- Skill tags:
  - [`api-design`](../../../.agent/skills/personal/engineering/backend/api-design/SKILL.md) — define activation request/result references, correlated observed/failed/acknowledgement messages, idempotency, bounded status fields, and fail-closed compatibility.
  - [`go`](../../../.agent/skills/personal/engineering/backend/go/SKILL.md) — implement the host-side status/recency handoff, deadlines, bounded serialization, cancellation, and persistence-degraded behavior without blocking browser activation.
  - [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md) — construct deterministic browser fakes and race fixtures that assert observable browser calls, status, profile isolation, and latency boundaries.
  - [`debugging`](../../../.agent/skills/common/engineering/debugging/SKILL.md) — model remove/move/reuse/window-close races, distinguish partial browser mutation from confirmed activation, and preserve diagnosable recovery paths.
  - [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md) — record the standalone ownership contract, exact fixture outcomes, bounded commands, and acceptance evidence.
  - [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md) — order guard, controller, reporter, fixture, race, and performance slices with explicit dependencies and no shared-path drift.
  - [`git-workflow`](../../../.agent/skills/common/engineering/git-workflow/SKILL.md) — keep the future implementation commits narrow, reviewable, and independently verifiable.
- Tài liệu trong `docs/`:
  - [`docs/plan/refactor/README.md`](../refactor/README.md) — binding Chromium desktop boundary, local-only behavior, and source-of-truth rule.
  - [`docs/plan/refactor/requirements.md`](../refactor/requirements.md) — FR-007, FR-011, FR-012, FR-015, NFR-003, and operational acceptance signals.
  - [`docs/plan/refactor/user-experience.md`](../refactor/user-experience.md) — Enter journey, stale selection, activation failure, focus, and trust rules.
  - [`docs/plan/refactor/architecture.md`](../refactor/architecture.md) — extension ownership of browser authority, host ownership of local computation, and failure isolation.
  - [`docs/plan/refactor/runtime-protocol.md`](../refactor/runtime-protocol.md) — revision-bound result use and `activation_observed`/`activation_failed` acknowledgement messages.
  - [`docs/plan/refactor/domain-and-privacy.md`](../refactor/domain-and-privacy.md) — profile/tab identity, activation-record fields, retention, and privacy invariants.
  - [`docs/plan/refactor/persistence-and-lifecycle.md`](../refactor/persistence-and-lifecycle.md) — bounded recency, degraded persistence, reset, and lifecycle behavior.
  - [`docs/plan/refactor/verification-and-acceptance.md`](../refactor/verification-and-acceptance.md) — activation journeys, race/recovery coverage, redaction, and p95 evidence.
- Quy ước code, ADR, context ngoài `docs/`: [`CONTEXT.md`](../../../CONTEXT.md) defines extension ownership of activation, bounded local data, deterministic/recoverable behavior, and the `to-create` source-tree assumption.

## 6. Công việc triển khai

- [ ] `IP-14-T01` **Reference and guard contract:** define the result-reference schema, identity comparison rules, projection-state gate, context/profile fence, typed outcomes, and zero-call invariant for every rejected request.
- [ ] `IP-14-T02` **Browser controller:** implement single-flight Enter handling, exact `tabs.get`/window-focus/tab-activation sequence, response verification, cancellation, and stage-specific browser-state accounting. Close the surface only after confirmed exact activation.
- [ ] `IP-14-T03` **Race and recovery behavior:** inject revision advance, remove, move, window close, numeric ID reuse, worker restart, permission denial, and browser shutdown at every boundary. Reject stale requests and never choose a replacement tab or older revision.
- [ ] `IP-14-T04` **Reporter and acknowledgement:** serialize the IP-07 activation status messages with profile/revision/request identity, safe error/status codes, bounded partial-state metadata, and non-blocking host/persistence degradation handling.
- [ ] `IP-14-T05` **Recency handoff:** send only the IP-08 activation metadata fields on confirmed activation, enforce source/timestamp/domain bounds, omit private durable data where required, and prove failed activation does not append recency.
- [ ] `IP-14-T06` **Surface handoff:** provide IP-13 a typed success/stale/missing/failure/degraded result with explicit `browser_state` and next action; never expose raw browser exceptions or imply an unconfirmed activation.
- [ ] `IP-14-T07` **Reset/uninstall teardown:** cancel in-flight work, discard pending reports, invoke the IP-08/IP-15 owned cleanup seam, and prove no tab/window mutation occurs during cleanup.
- [ ] `IP-14-T08` **Fixtures and tests:** implement `ACT-001` through `ACT-011` as fixture-driven tests over fake browser APIs and fake Native Messaging/storage boundaries; assert exact call logs and user-visible status models.
- [ ] `IP-14-T09` **Performance proof:** implement `ACT-012` with a monotonic clock, 20 warm-up and 200 measured iterations, separate controller and browser-scheduling measurements, and a non-zero exit on p95 failure.

## 7. Kế hoạch commit

1. `feat(activation): implement ip-14-t01`
   - Task IDs: `IP-14-T01`.
   - Owned target paths: `extension/src/browser/activation-controller.ts` (to-create), `extension/src/browser/activation-guard.ts` (to-create), `extension/src/runtime/activation-reporter.ts` (to-create), `fixtures/activation/phase-14/` (to-create), `tests/activation/phase-14/` (to-create).
   - Behavior: **Reference and guard contract:** define the result-reference schema, identity comparison rules, projection-state gate, context/profile fence, typed outcomes, and zero-call invariant for every rejected request.
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./tests/activation/phase-14/... -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Reference and guard contract:** define the result-reference schema, identity comparison rules, projection-state gate, context/profile fence, typed outcomes, and zero-call invariant for every rejected request.
   - Dependency gate: all index.md dependencies for IP-14 have merged to dev; phase work branch starts from latest origin/dev.

2. `feat(activation): implement ip-14-t02`
   - Task IDs: `IP-14-T02`.
   - Owned target paths: `extension/src/browser/activation-controller.ts` (to-create), `extension/src/browser/activation-guard.ts` (to-create), `extension/src/runtime/activation-reporter.ts` (to-create), `fixtures/activation/phase-14/` (to-create), `tests/activation/phase-14/` (to-create).
   - Behavior: **Browser controller:** implement single-flight Enter handling, exact `tabs.get`/window-focus/tab-activation sequence, response verification, cancellation, and stage-specific browser-state accounting. Close the surface only after confirmed exact activation.
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./tests/activation/phase-14/... -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Browser controller:** implement single-flight Enter handling, exact `tabs.get`/window-focus/tab-activation sequence, response verification, cancellation, and stage-specific browser-state accounting. Close the surface only after confirmed exact activation.
   - Dependency gate: all index.md dependencies for IP-14 have merged to dev; phase work branch starts from latest origin/dev.

3. `feat(activation): implement ip-14-t03`
   - Task IDs: `IP-14-T03`.
   - Owned target paths: `extension/src/browser/activation-controller.ts` (to-create), `extension/src/browser/activation-guard.ts` (to-create), `extension/src/runtime/activation-reporter.ts` (to-create), `fixtures/activation/phase-14/` (to-create), `tests/activation/phase-14/` (to-create).
   - Behavior: **Race and recovery behavior:** inject revision advance, remove, move, window close, numeric ID reuse, worker restart, permission denial, and browser shutdown at every boundary. Reject stale requests and never choose a replacement tab or older revision.
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./tests/activation/phase-14/... -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Race and recovery behavior:** inject revision advance, remove, move, window close, numeric ID reuse, worker restart, permission denial, and browser shutdown at every boundary. Reject stale requests and never choose a replacement tab or older revision.
   - Dependency gate: all index.md dependencies for IP-14 have merged to dev; phase work branch starts from latest origin/dev.

4. `feat(activation): implement ip-14-t04`
   - Task IDs: `IP-14-T04`.
   - Owned target paths: `extension/src/browser/activation-controller.ts` (to-create), `extension/src/browser/activation-guard.ts` (to-create), `extension/src/runtime/activation-reporter.ts` (to-create), `fixtures/activation/phase-14/` (to-create), `tests/activation/phase-14/` (to-create).
   - Behavior: **Reporter and acknowledgement:** serialize the IP-07 activation status messages with profile/revision/request identity, safe error/status codes, bounded partial-state metadata, and non-blocking host/persistence degradation handling.
   - Fixture and command: IP-07; run `go test ./tests/activation/phase-14/... -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Reporter and acknowledgement:** serialize the IP-07 activation status messages with profile/revision/request identity, safe error/status codes, bounded partial-state metadata, and non-blocking host/persistence degradation handling.
   - Dependency gate: all index.md dependencies for IP-14 have merged to dev; phase work branch starts from latest origin/dev.

5. `feat(activation): implement ip-14-t05`
   - Task IDs: `IP-14-T05`.
   - Owned target paths: `extension/src/browser/activation-controller.ts` (to-create), `extension/src/browser/activation-guard.ts` (to-create), `extension/src/runtime/activation-reporter.ts` (to-create), `fixtures/activation/phase-14/` (to-create), `tests/activation/phase-14/` (to-create).
   - Behavior: **Recency handoff:** send only the IP-08 activation metadata fields on confirmed activation, enforce source/timestamp/domain bounds, omit private durable data where required, and prove failed activation does not append recency.
   - Fixture and command: IP-08; run `go test ./tests/activation/phase-14/... -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Recency handoff:** send only the IP-08 activation metadata fields on confirmed activation, enforce source/timestamp/domain bounds, omit private durable data where required, and prove failed activation does not append recency.
   - Dependency gate: all index.md dependencies for IP-14 have merged to dev; phase work branch starts from latest origin/dev.

6. `feat(activation): implement ip-14-t06`
   - Task IDs: `IP-14-T06`.
   - Owned target paths: `extension/src/browser/activation-controller.ts` (to-create), `extension/src/browser/activation-guard.ts` (to-create), `extension/src/runtime/activation-reporter.ts` (to-create), `fixtures/activation/phase-14/` (to-create), `tests/activation/phase-14/` (to-create).
   - Behavior: **Surface handoff:** provide IP-13 a typed success/stale/missing/failure/degraded result with explicit `browser_state` and next action; never expose raw browser exceptions or imply an unconfirmed activation.
   - Fixture and command: IP-13; run `go test ./tests/activation/phase-14/... -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Surface handoff:** provide IP-13 a typed success/stale/missing/failure/degraded result with explicit `browser_state` and next action; never expose raw browser exceptions or imply an unconfirmed activation.
   - Dependency gate: all index.md dependencies for IP-14 have merged to dev; phase work branch starts from latest origin/dev.

7. `feat(activation): implement ip-14-t07`
   - Task IDs: `IP-14-T07`.
   - Owned target paths: `extension/src/browser/activation-controller.ts` (to-create), `extension/src/browser/activation-guard.ts` (to-create), `extension/src/runtime/activation-reporter.ts` (to-create), `fixtures/activation/phase-14/` (to-create), `tests/activation/phase-14/` (to-create).
   - Behavior: **Reset/uninstall teardown:** cancel in-flight work, discard pending reports, invoke the IP-08/IP-15 owned cleanup seam, and prove no tab/window mutation occurs during cleanup.
   - Fixture and command: IP-08, IP-15; run `go test ./tests/activation/phase-14/... -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Reset/uninstall teardown:** cancel in-flight work, discard pending reports, invoke the IP-08/IP-15 owned cleanup seam, and prove no tab/window mutation occurs during cleanup.
   - Dependency gate: all index.md dependencies for IP-14 have merged to dev; phase work branch starts from latest origin/dev.

8. `test(activation): implement ip-14-t08`
   - Task IDs: `IP-14-T08`.
   - Owned target paths: `extension/src/browser/activation-controller.ts` (to-create), `extension/src/browser/activation-guard.ts` (to-create), `extension/src/runtime/activation-reporter.ts` (to-create), `fixtures/activation/phase-14/` (to-create), `tests/activation/phase-14/` (to-create).
   - Behavior: **Fixtures and tests:** implement `ACT-001` through `ACT-011` as fixture-driven tests over fake browser APIs and fake Native Messaging/storage boundaries; assert exact call logs and user-visible status models.
   - Fixture and command: ACT-001, ACT-011; run `go test ./tests/activation/phase-14/... -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Fixtures and tests:** implement `ACT-001` through `ACT-011` as fixture-driven tests over fake browser APIs and fake Native Messaging/storage boundaries; assert exact call logs and user-visible status models.
   - Dependency gate: all index.md dependencies for IP-14 have merged to dev; phase work branch starts from latest origin/dev.

9. `test(activation): implement ip-14-t09`
   - Task IDs: `IP-14-T09`.
   - Owned target paths: `extension/src/browser/activation-controller.ts` (to-create), `extension/src/browser/activation-guard.ts` (to-create), `extension/src/runtime/activation-reporter.ts` (to-create), `fixtures/activation/phase-14/` (to-create), `tests/activation/phase-14/` (to-create).
   - Behavior: **Performance proof:** implement `ACT-012` with a monotonic clock, 20 warm-up and 200 measured iterations, separate controller and browser-scheduling measurements, and a non-zero exit on p95 failure.
   - Fixture and command: ACT-012; run `go test ./tests/activation/phase-14/... -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Performance proof:** implement `ACT-012` with a monotonic clock, 20 warm-up and 200 measured iterations, separate controller and browser-scheduling measurements, and a non-zero exit on p95 failure.
   - Dependency gate: all index.md dependencies for IP-14 have merged to dev; phase work branch starts from latest origin/dev.

## 8. Kiểm chứng và nghiệm thu

- [ ] From repository root, run `go test ./tests/activation/phase-14/... -count=1` and the extension-targeted `npm test -- --runInBand tests/activation/phase-14/activation-controller.test.ts`; every `ACT-001` through `ACT-011` outcome MUST be asserted through observable browser calls, status models, and bounded payloads.
- [ ] `ACT-001` and `ACT-007` prove FR-007: keyboard Enter focuses the expected window and activates the exact selected tab, including a multi-window case; arrow-selection input is consumed from the IP-13 action seam.
- [ ] `ACT-003` through `ACT-006` prove FR-011: stale, removed, moved, reused, mismatched, or unavailable results never activate a different tab silently, never fall back to an older revision, and expose refresh/retry guidance.
- [ ] `ACT-008` and reporter assertions support FR-012: host absence/recovery and persistence degradation are visible in status/diagnostics without converting confirmed browser activation into failure or blocking the hot path.
- [ ] `ACT-011` supports FR-015: reset/uninstall removes only owned pending/recency state and does not close tabs, focus an unrelated window, or alter unrelated browser/profile data.
- [ ] Run `GOMAXPROCS=1 go run ./tests/activation/phase-14/cmd/selection-to-activation-benchmark --fixture fixtures/activation/phase-14/ACT-012-selection-to-activation-100ms.json --warmup 20 --iterations 200 --p95-budget-ms 100`; record p50/p95/p99, controller timing, browser-scheduling exclusion, result count, activation-call count, environment, and exit status. NFR-003 passes only with p95 `<= 100 ms`.
- [ ] Every failure message states the failed stage, whether browser state changed (`unchanged`, `window_focused`, `tab_active`, or `unknown`), and the next safe action; no raw browser exception, URL, title, query, or secret is emitted by the activation path.
- [ ] Every accepted activation uses exact profile/context, projection revision, tab identity, and window/tab IDs; no fixture permits activation from another profile, another revision, another window, or a replacement tab.
- [ ] The search surface closes only after confirmed exact activation; missing-tab, stale, race, host-unavailable, persistence-degraded, and browser-error outcomes remain actionable and do not claim success.
- [ ] Acceptance is complete only when no known in-scope activation, stale-reference, race, acknowledgement, recency, reset/uninstall, or p95 defect remains unobservable.

## 9. Rủi ro và quyết định còn mở

- **Rủi ro:** A browser tab can disappear, move, or be replaced after validation and before the activation call. **Phương án xử lý đã chọn:** bind the request to profile/context, projection epoch/revision, tab identity, exact tab ID, and exact window ID; revalidate before each mutating call, stop on any mismatch/error, and never select a substitute.
- **Rủi ro:** Focusing a window may succeed while activating the tab fails. **Phương án xử lý đã chọn:** record `browser_state=window_focused`, keep the surface open, report a bounded partial failure, and require a fresh explicit retry rather than implying activation success.
- **Rủi ro:** A host or SQLite failure may occur after the browser confirms activation. **Phương án xử lý đã chọn:** browser confirmation is authoritative for user outcome; reporter acknowledgement and durable recency are non-blocking, with a visible degraded diagnostic but no compensating browser action.
- **Rủi ro:** Numeric browser tab IDs may be reused. **Phương án xử lý đã chọn:** require the current projection epoch/tab identity and revision in addition to strict numeric ID equality; a reused ID is stale until a new result is selected from the current revision.
- **Rủi ro:** Duplicate Enter events may cause duplicate activation/reporting. **Phương án xử lý đã chọn:** single-flight request ledger and request correlation; duplicate events are acknowledged/ignored without a second browser operation.
- **Câu hỏi còn mở chỉ khi câu trả lời có thể thay đổi contract:** None. IP-05 supplies revision/identity semantics, IP-07 supplies wire acknowledgement fields, IP-08 supplies retention/degraded persistence, and IP-13 supplies the selected-row action seam.

## 10. References ngoài `docs/`

- Skill: [`api-design`](../../../.agent/skills/personal/engineering/backend/api-design/SKILL.md)
- Skill: [`go`](../../../.agent/skills/personal/engineering/backend/go/SKILL.md)
- Skill: [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md)
- Skill: [`debugging`](../../../.agent/skills/common/engineering/debugging/SKILL.md)
- Skill: [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md)
- Skill: [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md)
- Skill: [`git-workflow`](../../../.agent/skills/common/engineering/git-workflow/SKILL.md)
- Project context: [`CONTEXT.md`](../../../CONTEXT.md)
- Source/config/test path ngoài `docs/`: `extension/src/browser/activation-controller.ts` (to-create), `extension/src/browser/activation-guard.ts` (to-create), `extension/src/runtime/activation-reporter.ts` (to-create), `tests/activation/phase-14/` (to-create)
- Fixture/tool/artifact ngoài `docs/`: `fixtures/activation/phase-14/ACT-001` through `ACT-012` (to-create); benchmark command `tests/activation/phase-14/cmd/selection-to-activation-benchmark/` (to-create)
