# Phase 04 — Tab observation and eligibility

> Plan ID: IP-04
> Status: See README.md execution tracker
> Execution owner: extension browser-events implementer
> Dependencies: IP-02, IP-03
> Parallel boundary: IP-06, IP-07, IP-08 after the IP-02/IP-03 contracts are stable; no shared implementation files
> Requirement IDs: FR-009 (primary); FR-003, NFR-005, NFR-007, NFR-009 (supporting)
> Owned paths: `extension/src/browser/tab-observer.ts` (to-create), `extension/src/browser/eligibility.ts` (to-create), `extension/src/browser/event-normalizer.ts` (to-create), `fixtures/browser/phase-04/` (to-create), `tests/browser/phase-04/` (to-create)

## 1. Mục tiêu

- Implement the extension-owned observation boundary for the current browser profile. Convert Chrome and Edge tab, tab-group, and window notifications into a typed, profile-scoped event stream without allowing browser API details to leak into the host or UI.
- Reconcile create, update, move, group, pin, activate, window, and remove events into the eligible open-tab projection contract from IP-02. A full snapshot remains authoritative; event application is an ordered optimization between snapshots, not an independent source of truth.
- Define one explicit eligibility decision for every observed tab: include, exclude, or defer because required browser state is unavailable. Private-context handling, unsupported/denied API access, closed windows, and malformed or incomplete event payloads must be observable and fail closed.
- Suppress duplicate and ineffective notifications while preserving meaningful changes such as URL/title updates, window or group changes, active/pinned transitions, and eligibility transitions. The same fixture event sequence must converge to exactly one record per eligible tab identity.
- Hand off deterministic deltas and resync signals to the IP-05 projection reconciler. This phase does not own host transport, indexing, ranking, persistence, or activation execution.

## 2. Phạm vi

- Bao gồm:
  - A Chrome/Edge browser-adapter listener seam for `tabs.onCreated`, `tabs.onUpdated`, `tabs.onMoved`, `tabs.onAttached`, `tabs.onDetached`, `tabs.onActivated`, `tabs.onRemoved`, `tabGroups.onUpdated`, `tabGroups.onMoved`, `tabGroups.onRemoved` where available, and `windows.onCreated`, `windows.onRemoved`, `windows.onFocusChanged`.
  - A normalized event vocabulary for `create`, `update`, `move`, `group`, `pin`, `activate`, `window_changed`, `window_removed`, and `remove`, with event source, profile/context identity, browser IDs, changed fields, and an ordering/recovery marker.
  - Eligibility evaluation against the current profile and allowed open-tab metadata: tab ID, window ID, group ID when present, title, URL, domain, window/group labels when available, pinned state, active state, context kind, and current projection epoch.
  - A full eligible-tab snapshot path used at startup, after worker restart, after permission/API denial, after event loss, and whenever event ordering cannot be trusted.
  - Per-tab duplicate suppression and coalescing rules that are idempotent for repeated notifications but do not drop an effective state change. A remove event is terminal for its identity in the current epoch; a later browser-ID reuse cannot resurrect the prior identity.
  - Normal and private-context policy: normal tabs may enter the live in-memory projection; private tabs are isolated by context and are removed when the private context ends. Private records must never be written to durable storage or leak into a normal-context snapshot.
  - Explicit permission-denial, unsupported-API, missing-field, and browser shutdown behavior. These states emit bounded diagnostic metadata and request a safe resync or degraded status rather than silently treating an incomplete snapshot as empty.
  - Browser event fixtures, adapter fakes, reducer seams, and convergence tests for Chrome/Edge parity.
- Ngoài phạm vi:
  - Native Messaging framing, hello/handshake, host lifecycle, reconnect transport, or protocol error literals; IP-06 and IP-07 own those boundaries.
  - Projection revision ownership, host-side reconciliation, atomic host snapshot replacement, index rebuild, and query availability; IP-05, IP-09, and IP-10 own those behaviors. This phase emits the event contract they consume.
  - Ranking, normalization, score weights, result rendering, activation calls, stale-result activation guards, configuration, reset, uninstall, or SQLite writes.
  - Any page evaluation, page-content collection, remote fetch, network access, unapproved browser data source, or browser-wide data store. The observer is limited to permitted current open-tab metadata.
  - Treating event order, event count, or a transient missing event as authoritative over a full browser snapshot.

## 3. Điều kiện tiên quyết

- IP-01 is merged and defines the shared fixture schema, logical roots, bounded values, `FX-TAB-EVENTS`, `FX-PERMISSION-DENIED`, requirement ownership, and global commands. The phase must extend those fixture IDs rather than introduce an incompatible catalog.
- IP-02 is merged and defines `profile_id`, `ContextKind`, `TabIdentity`, `ProjectionEpoch`, `ProjectionRevision`, `EligibleTabRecord`, missing-field behavior, lifecycle states, and extension ownership of browser authority. The observer must use these types and must not invent title/URL-based identity.
- IP-03 is merged and provides the Manifest V3 service-worker and browser-adapter seam, exact minimal permissions, Chrome/Edge namespace selection, and lifecycle hooks. Listener registration must be top-level and safe after worker suspension/resume; it must not add permissions or call browser globals outside the adapter.
- Read the binding [`target architecture`](../refactor/architecture.md), [`domain and privacy contract`](../refactor/domain-and-privacy.md), [`browser landscape`](../refactor/browser-landscape.md), [`requirements`](../refactor/requirements.md), [`runtime protocol`](../refactor/runtime-protocol.md), and [`verification and acceptance`](../refactor/verification-and-acceptance.md).
- Source roots are currently absent. Every path named in this phase is a logical `to-create` target and must be replaced with the exact implementation path only if the source tree exists when implementation begins.
- The fixture harness must be able to emit events with controlled ordering, repeat an event, omit optional fields, deny an API call, close a private window, reuse a browser tab ID in a new epoch, and return an independent full snapshot for oracle comparison.

## 4. Đầu ra cần bàn giao

- `extension/src/browser/tab-observer.ts` (to-create):
  - Listener registration and teardown for all supported tab/window/group event sources.
  - A startup/full-snapshot method that reads only the current profile's open tabs and returns a bounded set of raw adapter records for eligibility evaluation.
  - Event callbacks that capture the smallest safe event payload and enqueue normalized events without performing host calls or browser activation.
  - Lifecycle behavior for worker restart, browser shutdown, listener registration failure, and event queue overflow: mark the stream uncertain and request a full snapshot.
- `extension/src/browser/event-normalizer.ts` (to-create):
  - Conversion of Chrome and Edge event signatures into the common event vocabulary, including change masks for title, URL, status, window, group, pinned, active, and other eligible fields.
  - Validation of profile/context, tab/window/group IDs, event epoch, and bounded strings before a reducer sees the event.
  - Explicit handling for `onUpdated` events that arrive in several partial notifications and for equivalent browser notifications emitted by both tab and window APIs.
- `extension/src/browser/eligibility.ts` (to-create):
  - A pure eligibility function with a decision (`eligible`, `excluded`, or `deferred`), reason code, normalized allowed fields, and context disposal action when relevant.
  - Rules for normal/private contexts, closed or missing windows, unsupported URL schemes, inaccessible or missing metadata, and denied/unsupported browser capabilities. Reasons must be safe for diagnostics and must not contain raw title or URL values.
  - A per-profile/context identity key based on IP-02 identity fields; never use title, URL, domain, array position, or event sequence as identity.
- `fixtures/browser/phase-04/` (to-create): stable JSON fixtures, each with input event/snapshot stream, expected normalized decisions, expected projection IDs/fields, diagnostic reason where applicable, and owning requirements:
  - `FX-TAB-EVENTS` — create a tab, update title/URL, move it, group it, pin it, activate it, update its window, and remove it. Replaying any notification once produces the same final set as applying it once; final eligible identities are unique.
  - `FX-TAB-EVENT-COALESCE` — deliver repeated and reordered partial updates for one tab. Effective field changes are retained, no-op duplicates do not advance the effective state, and an uncertain order requests resync rather than guessing.
  - `FX-TAB-WINDOW-LIFECYCLE` — create/focus/close windows, attach/detach/move tabs, and remove a window. Tabs in a removed window are removed or deferred consistently; no orphaned eligible record remains.
  - `FX-TAB-ELIGIBILITY` — mix eligible normal tabs with unsupported, incomplete, closed-window, and excluded tabs. Only allowed records enter the projection, every excluded record has a stable reason code, and unavailable optional labels do not exclude an otherwise valid tab.
  - `FX-PRIVATE-CONTEXT` — observe an isolated private window and its tabs, then close that context. Private records are isolated from normal records, never durable, and disappear at context end.
  - `FX-PERMISSION-DENIED` — deny one required browser capability and separately deny an optional metadata capability. The first case yields a diagnosable unavailable/deferred state and resync/retry action; the second retains safe fields while reporting the missing optional field.
  - `FX-TAB-ID-REUSE` — remove a tab and reuse its numeric browser ID in a new projection epoch. The new tab has a distinct `TabIdentity`; no stale record or old event mutates it.
  - `FX-TAB-SNAPSHOT-CONVERGENCE` — apply a long event stream with dropped/repeated notifications, then compare to the independently obtained full eligible snapshot. The result is exactly the snapshot ID set and field state, with no duplicate IDs.
- `tests/browser/phase-04/` (to-create): adapter parity tests, pure eligibility/reason-code tests, reducer/event-normalizer tests, private-context disposal tests, denial diagnostics tests, and the convergence/property fixture runner. Tests must assert observable projection and status outcomes rather than listener implementation details.
- A handoff contract for IP-05 stating which normalized event fields are authoritative, when to advance an effective projection revision, when to set `resync_required`, and which diagnostic counters/error classes are emitted. The handoff must state that the browser full snapshot is the authority after uncertainty.

## 5. Skill và tài liệu áp dụng

- Skill tags:
  - [`common/workflow/feature-delivery`](../../../.agent/skills/common/workflow/feature-delivery/SKILL.md) — scope the browser-event slice end to end, including adapter seam, fixtures, test evidence, and IP-05 handoff without leaking into query or host work.
  - [`common/security/secure-development`](../../../.agent/skills/common/security/secure-development/SKILL.md) — enforce least privilege, fail-closed eligibility, profile/private isolation, bounded event fields, and safe handling of browser-supplied values.
  - [`common/engineering/testing`](../../../.agent/skills/common/engineering/testing/SKILL.md) — define deterministic event fixtures, browser API fakes, convergence oracles, failure cases, and observable acceptance signals.
  - [`common/engineering/debugging`](../../../.agent/skills/common/engineering/debugging/SKILL.md) — make event ordering gaps, permission/API denial, worker restarts, duplicates, and projection divergence diagnosable without raw tab content.
  - [`common/engineering/documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md) — preserve a standalone, auditable phase contract and exact external references.
  - [`common/foundation/task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md) — break the boundary into listener, normalization, eligibility, fixture, and acceptance slices with explicit dependencies.
  - [`common/engineering/git-workflow`](../../../.agent/skills/common/engineering/git-workflow/SKILL.md) — keep this one-file phase branch and the future implementation commits reviewable and isolated.
- Tài liệu trong `docs/`:
  - [`refactor/README.md`](../refactor/README.md) — binding product boundary and source-of-truth rule.
  - [`refactor/architecture.md`](../refactor/architecture.md) — extension ownership, full snapshot/delta flow, and recovery boundary.
  - [`refactor/domain-and-privacy.md`](../refactor/domain-and-privacy.md) — eligible record fields, trust boundaries, private-context and privacy invariants.
  - [`refactor/browser-landscape.md`](../refactor/browser-landscape.md) — Chrome/Edge API and profile/context assumptions.
  - [`refactor/requirements.md`](../refactor/requirements.md) — FR-003, FR-009, NFR-005, NFR-007, and NFR-009 acceptance contracts.
  - [`refactor/runtime-protocol.md`](../refactor/runtime-protocol.md) — resync/revision handoff constraints for the later host boundary.
  - [`refactor/verification-and-acceptance.md`](../refactor/verification-and-acceptance.md) — recovery, privacy, browser, and convergence evidence requirements.
- Quy ước code, ADR, context ngoài `docs/`:
  - [`CONTEXT.md`](../../../CONTEXT.md) — current vocabulary, ownership, privacy, and test-seam rules.
  - Use the IP-02 domain records and IP-03 adapter interface as contracts; do not add a second browser abstraction or a second profile identity mechanism.

## 6. Công việc triển khai

- [ ] `IP-04-T01` Map every supported Chrome/Edge event source to the common event vocabulary. Record which event fields are required, optional, browser-specific, or a resync trigger; keep registration idempotent across service-worker restarts.
- [ ] `IP-04-T02` Implement the pure eligibility decision. Validate profile/context and bounded IDs, keep only allowed open-tab metadata, assign stable reason codes, and distinguish `excluded` from `deferred` when a retry or full snapshot can recover state.
- [ ] `IP-04-T03` Implement full snapshot acquisition and an independent snapshot oracle in the fake browser adapter. Treat an empty snapshot as authoritative only when the browser call completed successfully and the profile/context is known.
- [ ] `IP-04-T04` Normalize partial `onUpdated` changes and tab/window/group notifications into field-level deltas. Merge equivalent updates by `TabIdentity`; preserve meaningful title/URL/window/group/pin/active/eligibility changes.
- [ ] `IP-04-T05` Define duplicate suppression: repeated identical effective state is a no-op; conflicting or out-of-order revisions mark uncertainty; remove is idempotent; browser-ID reuse requires a new epoch/identity; no notification may produce two records for one identity.
- [ ] `IP-04-T06` Define window and group cascades. A removed or inaccessible window must not leave eligible orphan tabs; a group label update must affect only tabs in that group; missing optional group/window labels must preserve the tab with an explicit absent value.
- [ ] `IP-04-T07` Define private-context behavior and teardown. Keep private records in the current in-memory projection only, partition them from normal records, and remove them on private-context end, browser shutdown, reset, or disconnect according to IP-02 lifecycle rules.
- [ ] `IP-04-T08` Define permission/API-denial behavior. Required capability denial produces a bounded unavailable/deferred state, retry/resnapshot signal, and structured diagnostic counter; optional capability denial preserves safe fields and marks only the missing field.
- [ ] `IP-04-T09` Emit the IP-05 handoff with event sequence metadata, effective-change marker, resync reason, and safe counters. Never log or transmit raw titles, full URLs, query strings, fragments, or page data as diagnostics.
- [ ] `IP-04-T10` Add fixtures and tests for `FX-TAB-EVENTS`, `FX-TAB-EVENT-COALESCE`, `FX-TAB-WINDOW-LIFECYCLE`, `FX-TAB-ELIGIBILITY`, `FX-PRIVATE-CONTEXT`, `FX-PERMISSION-DENIED`, `FX-TAB-ID-REUSE`, and `FX-TAB-SNAPSHOT-CONVERGENCE`.
- [ ] `IP-04-T11` Run a Chrome fake-adapter and Edge fake-adapter through the same fixture corpus. Differences in API namespace/signature must not alter normalized decisions or final projection output.

## 7. Kế hoạch commit

1. `feat(browser): implement ip-04-t01`
   - Task IDs: `IP-04-T01`.
   - Owned target paths: `extension/src/browser/tab-observer.ts` (to-create), `extension/src/browser/eligibility.ts` (to-create), `extension/src/browser/event-normalizer.ts` (to-create), `fixtures/browser/phase-04/` (to-create), `tests/browser/phase-04/` (to-create).
   - Behavior: Map every supported Chrome/Edge event source to the common event vocabulary. Record which event fields are required, optional, browser-specific, or a resync trigger; keep registration idempotent across service-worker restarts.
   - Fixture and command: the observable fixture/outcome stated by this task; run `npm test -- --runInBand tests/browser/phase-04`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Map every supported Chrome/Edge event source to the common event vocabulary. Record which event fields are required, optional, browser-specific, or a resync trigger; keep registration idempotent across service-worker restarts.
   - Dependency gate: all index.md dependencies for IP-04 have merged to dev; phase work branch starts from latest origin/dev.

2. `feat(browser): implement ip-04-t02`
   - Task IDs: `IP-04-T02`.
   - Owned target paths: `extension/src/browser/tab-observer.ts` (to-create), `extension/src/browser/eligibility.ts` (to-create), `extension/src/browser/event-normalizer.ts` (to-create), `fixtures/browser/phase-04/` (to-create), `tests/browser/phase-04/` (to-create).
   - Behavior: Implement the pure eligibility decision. Validate profile/context and bounded IDs, keep only allowed open-tab metadata, assign stable reason codes, and distinguish `excluded` from `deferred` when a retry or full snapshot can recover state.
   - Fixture and command: the observable fixture/outcome stated by this task; run `npm test -- --runInBand tests/browser/phase-04`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Implement the pure eligibility decision. Validate profile/context and bounded IDs, keep only allowed open-tab metadata, assign stable reason codes, and distinguish `excluded` from `deferred` when a retry or full snapshot can recover state.
   - Dependency gate: all index.md dependencies for IP-04 have merged to dev; phase work branch starts from latest origin/dev.

3. `feat(browser): implement ip-04-t03`
   - Task IDs: `IP-04-T03`.
   - Owned target paths: `extension/src/browser/tab-observer.ts` (to-create), `extension/src/browser/eligibility.ts` (to-create), `extension/src/browser/event-normalizer.ts` (to-create), `fixtures/browser/phase-04/` (to-create), `tests/browser/phase-04/` (to-create).
   - Behavior: Implement full snapshot acquisition and an independent snapshot oracle in the fake browser adapter. Treat an empty snapshot as authoritative only when the browser call completed successfully and the profile/context is known.
   - Fixture and command: the observable fixture/outcome stated by this task; run `npm test -- --runInBand tests/browser/phase-04`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Implement full snapshot acquisition and an independent snapshot oracle in the fake browser adapter. Treat an empty snapshot as authoritative only when the browser call completed successfully and the profile/context is known.
   - Dependency gate: all index.md dependencies for IP-04 have merged to dev; phase work branch starts from latest origin/dev.

4. `feat(browser): implement ip-04-t04`
   - Task IDs: `IP-04-T04`.
   - Owned target paths: `extension/src/browser/tab-observer.ts` (to-create), `extension/src/browser/eligibility.ts` (to-create), `extension/src/browser/event-normalizer.ts` (to-create), `fixtures/browser/phase-04/` (to-create), `tests/browser/phase-04/` (to-create).
   - Behavior: Normalize partial `onUpdated` changes and tab/window/group notifications into field-level deltas. Merge equivalent updates by `TabIdentity`; preserve meaningful title/URL/window/group/pin/active/eligibility changes.
   - Fixture and command: the observable fixture/outcome stated by this task; run `npm test -- --runInBand tests/browser/phase-04`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Normalize partial `onUpdated` changes and tab/window/group notifications into field-level deltas. Merge equivalent updates by `TabIdentity`; preserve meaningful title/URL/window/group/pin/active/eligibility changes.
   - Dependency gate: all index.md dependencies for IP-04 have merged to dev; phase work branch starts from latest origin/dev.

5. `feat(browser): implement ip-04-t05`
   - Task IDs: `IP-04-T05`.
   - Owned target paths: `extension/src/browser/tab-observer.ts` (to-create), `extension/src/browser/eligibility.ts` (to-create), `extension/src/browser/event-normalizer.ts` (to-create), `fixtures/browser/phase-04/` (to-create), `tests/browser/phase-04/` (to-create).
   - Behavior: Define duplicate suppression: repeated identical effective state is a no-op; conflicting or out-of-order revisions mark uncertainty; remove is idempotent; browser-ID reuse requires a new epoch/identity; no notification may produce two records for one identity.
   - Fixture and command: the observable fixture/outcome stated by this task; run `npm test -- --runInBand tests/browser/phase-04`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Define duplicate suppression: repeated identical effective state is a no-op; conflicting or out-of-order revisions mark uncertainty; remove is idempotent; browser-ID reuse requires a new epoch/identity; no notification may produce two records for one identity.
   - Dependency gate: all index.md dependencies for IP-04 have merged to dev; phase work branch starts from latest origin/dev.

6. `feat(browser): implement ip-04-t06`
   - Task IDs: `IP-04-T06`.
   - Owned target paths: `extension/src/browser/tab-observer.ts` (to-create), `extension/src/browser/eligibility.ts` (to-create), `extension/src/browser/event-normalizer.ts` (to-create), `fixtures/browser/phase-04/` (to-create), `tests/browser/phase-04/` (to-create).
   - Behavior: Define window and group cascades. A removed or inaccessible window must not leave eligible orphan tabs; a group label update must affect only tabs in that group; missing optional group/window labels must preserve the tab with an explicit absent value.
   - Fixture and command: the observable fixture/outcome stated by this task; run `npm test -- --runInBand tests/browser/phase-04`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Define window and group cascades. A removed or inaccessible window must not leave eligible orphan tabs; a group label update must affect only tabs in that group; missing optional group/window labels must preserve the tab with an explicit absent value.
   - Dependency gate: all index.md dependencies for IP-04 have merged to dev; phase work branch starts from latest origin/dev.

7. `feat(browser): implement ip-04-t07`
   - Task IDs: `IP-04-T07`.
   - Owned target paths: `extension/src/browser/tab-observer.ts` (to-create), `extension/src/browser/eligibility.ts` (to-create), `extension/src/browser/event-normalizer.ts` (to-create), `fixtures/browser/phase-04/` (to-create), `tests/browser/phase-04/` (to-create).
   - Behavior: Define private-context behavior and teardown. Keep private records in the current in-memory projection only, partition them from normal records, and remove them on private-context end, browser shutdown, reset, or disconnect according to IP-02 lifecycle rules.
   - Fixture and command: IP-02; run `npm test -- --runInBand tests/browser/phase-04`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Define private-context behavior and teardown. Keep private records in the current in-memory projection only, partition them from normal records, and remove them on private-context end, browser shutdown, reset, or disconnect according to IP-02 lifecycle rules.
   - Dependency gate: all index.md dependencies for IP-04 have merged to dev; phase work branch starts from latest origin/dev.

8. `feat(browser): implement ip-04-t08`
   - Task IDs: `IP-04-T08`.
   - Owned target paths: `extension/src/browser/tab-observer.ts` (to-create), `extension/src/browser/eligibility.ts` (to-create), `extension/src/browser/event-normalizer.ts` (to-create), `fixtures/browser/phase-04/` (to-create), `tests/browser/phase-04/` (to-create).
   - Behavior: Define permission/API-denial behavior. Required capability denial produces a bounded unavailable/deferred state, retry/resnapshot signal, and structured diagnostic counter; optional capability denial preserves safe fields and marks only the missing field.
   - Fixture and command: the observable fixture/outcome stated by this task; run `npm test -- --runInBand tests/browser/phase-04`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Define permission/API-denial behavior. Required capability denial produces a bounded unavailable/deferred state, retry/resnapshot signal, and structured diagnostic counter; optional capability denial preserves safe fields and marks only the missing field.
   - Dependency gate: all index.md dependencies for IP-04 have merged to dev; phase work branch starts from latest origin/dev.

9. `feat(browser): implement ip-04-t09`
   - Task IDs: `IP-04-T09`.
   - Owned target paths: `extension/src/browser/tab-observer.ts` (to-create), `extension/src/browser/eligibility.ts` (to-create), `extension/src/browser/event-normalizer.ts` (to-create), `fixtures/browser/phase-04/` (to-create), `tests/browser/phase-04/` (to-create).
   - Behavior: Emit the IP-05 handoff with event sequence metadata, effective-change marker, resync reason, and safe counters. Never log or transmit raw titles, full URLs, query strings, fragments, or page data as diagnostics.
   - Fixture and command: IP-05; run `npm test -- --runInBand tests/browser/phase-04`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Emit the IP-05 handoff with event sequence metadata, effective-change marker, resync reason, and safe counters. Never log or transmit raw titles, full URLs, query strings, fragments, or page data as diagnostics.
   - Dependency gate: all index.md dependencies for IP-04 have merged to dev; phase work branch starts from latest origin/dev.

10. `test(browser): implement ip-04-t10`
   - Task IDs: `IP-04-T10`.
   - Owned target paths: `extension/src/browser/tab-observer.ts` (to-create), `extension/src/browser/eligibility.ts` (to-create), `extension/src/browser/event-normalizer.ts` (to-create), `fixtures/browser/phase-04/` (to-create), `tests/browser/phase-04/` (to-create).
   - Behavior: Add fixtures and tests for `FX-TAB-EVENTS`, `FX-TAB-EVENT-COALESCE`, `FX-TAB-WINDOW-LIFECYCLE`, `FX-TAB-ELIGIBILITY`, `FX-PRIVATE-CONTEXT`, `FX-PERMISSION-DENIED`, `FX-TAB-ID-REUSE`, and `FX-TAB-SNAPSHOT-CONVERGENCE`.
   - Fixture and command: FX-TAB-EVENTS, FX-TAB-EVENT-COALESCE, FX-TAB-WINDOW-LIFECYCLE, FX-TAB-ELIGIBILITY, FX-PRIVATE-CONTEXT, FX-PERMISSION-DENIED, FX-TAB-ID-REUSE, FX-TAB-SNAPSHOT-CONVERGENCE; run `npm test -- --runInBand tests/browser/phase-04`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Add fixtures and tests for `FX-TAB-EVENTS`, `FX-TAB-EVENT-COALESCE`, `FX-TAB-WINDOW-LIFECYCLE`, `FX-TAB-ELIGIBILITY`, `FX-PRIVATE-CONTEXT`, `FX-PERMISSION-DENIED`, `FX-TAB-ID-REUSE`, and `FX-TAB-SNAPSHOT-CONVERGENCE`.
   - Dependency gate: all index.md dependencies for IP-04 have merged to dev; phase work branch starts from latest origin/dev.

11. `test(browser): implement ip-04-t11`
   - Task IDs: `IP-04-T11`.
   - Owned target paths: `extension/src/browser/tab-observer.ts` (to-create), `extension/src/browser/eligibility.ts` (to-create), `extension/src/browser/event-normalizer.ts` (to-create), `fixtures/browser/phase-04/` (to-create), `tests/browser/phase-04/` (to-create).
   - Behavior: Run a Chrome fake-adapter and Edge fake-adapter through the same fixture corpus. Differences in API namespace/signature must not alter normalized decisions or final projection output.
   - Fixture and command: the observable fixture/outcome stated by this task; run `npm test -- --runInBand tests/browser/phase-04`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Run a Chrome fake-adapter and Edge fake-adapter through the same fixture corpus. Differences in API namespace/signature must not alter normalized decisions or final projection output.
   - Dependency gate: all index.md dependencies for IP-04 have merged to dev; phase work branch starts from latest origin/dev.

## 8. Kiểm chứng và nghiệm thu

- [ ] Once implementation exists, from repository root run the exact phase command `npm test -- --runInBand tests/browser/phase-04` (or the repository's equivalent command recorded by IP-01) with fixture root `fixtures/browser/phase-04/`; do not substitute a test that omits the fixtures.
- [ ] Run the Chrome and Edge fake-adapter matrix over `FX-TAB-EVENTS` and `FX-TAB-SNAPSHOT-CONVERGENCE`. The final projection contains one record per expected eligible `TabIdentity`, no duplicate IDs, and the same field values as the independent full snapshot.
- [ ] Run `FX-TAB-EVENT-COALESCE`, `FX-TAB-WINDOW-LIFECYCLE`, and `FX-TAB-ID-REUSE`. Repeated notifications are idempotent, effective changes survive coalescing, orphan records are removed, and an old numeric tab ID cannot mutate or activate a new identity.
- [ ] Run `FX-TAB-ELIGIBILITY` and `FX-PRIVATE-CONTEXT`. The result includes only eligible current-profile records; excluded records have stable safe reason codes; private records are isolated, in-memory only, and removed when private context ends.
- [ ] Run `FX-PERMISSION-DENIED` with required and optional capability denial. Required denial is fail-closed, retryable or resyncable, and visible through a bounded diagnostic status; optional denial preserves safe fields and identifies the missing capability.
- [ ] Observe the event-to-snapshot convergence signal required by FR-009 and NFR-005: after a successful full snapshot, indexed candidate IDs (as reported by the handoff) equal browser-visible eligible IDs. A dropped/uncertain stream sets `resync_required` instead of claiming convergence.
- [ ] Inspect diagnostics for NFR-007/NFR-009: no network request, page evaluation, unapproved data-source permission, private durable record, full URL, title, query string, or fragment appears in the event diagnostics or fixture output.
- [ ] Acceptance signal: `FX-TAB-EVENTS` converges without duplicates; excluded and private tabs have explicit outcomes; required and optional permission denial are distinguishable and diagnosable; every browser event class in FR-009 has a fixture and observable result.

## 9. Rủi ro và quyết định còn mở

- Rủi ro: Chrome and Edge can emit several partial or overlapping notifications for one browser action, and service-worker suspension can lose in-flight callbacks. Phương án xử lý đã chọn: normalize to field-level effective changes, suppress identical state, mark uncertain order, and obtain a full snapshot whenever continuity is not provable.
- Rủi ro: A browser reuses a numeric tab ID after removal. Phương án xử lý đã chọn: bind identity to IP-02 profile/context/epoch fields and reject events from an old epoch; never fall back to title, URL, or array order.
- Rủi ro: Required browser capability denial can look like an empty profile. Phương án xử lý đã chọn: distinguish successful empty snapshot from denied/failed snapshot, expose a bounded unavailable/deferred state, and require retry or resync before reporting an empty projection.
- Rủi ro: Private tabs can leak through a shared adapter cache or a normal-context snapshot. Phương án xử lý đã chọn: partition all records by `ContextKind`, keep private data transient, and test context teardown plus profile isolation as a state transition.
- Rủi ro: Optional window/group metadata may be unavailable across browser versions. Phương án xử lý đã chọn: preserve the eligible tab with explicit missing optional values; only a missing field required for identity or safe eligibility defers the record.
- Câu hỏi còn mở chỉ khi câu trả lời có thể thay đổi contract: which exact `tabGroups` event APIs are available in the minimum Chrome and Edge versions declared by packaging. The adapter must feature-detect them and retain the same normalized event contract; this cannot weaken privacy or convergence guarantees.

## 10. References ngoài `docs/`

- Skill: [`feature-delivery`](../../../.agent/skills/common/workflow/feature-delivery/SKILL.md)
- Skill: [`secure-development`](../../../.agent/skills/common/security/secure-development/SKILL.md)
- Skill: [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md)
- Skill: [`debugging`](../../../.agent/skills/common/engineering/debugging/SKILL.md)
- Skill: [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md)
- Skill: [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md)
- Skill: [`git-workflow`](../../../.agent/skills/common/engineering/git-workflow/SKILL.md)
- Project context: [`CONTEXT.md`](../../../CONTEXT.md)
- Source/config/test path ngoài `docs/`: `extension/src/browser/tab-observer.ts`, `extension/src/browser/eligibility.ts`, `extension/src/browser/event-normalizer.ts`, `fixtures/browser/phase-04/`, `tests/browser/phase-04/` (all `to-create`)
- Fixture/tool/artifact ngoài `docs/`: `FX-TAB-EVENTS`, `FX-TAB-EVENT-COALESCE`, `FX-TAB-WINDOW-LIFECYCLE`, `FX-TAB-ELIGIBILITY`, `FX-PRIVATE-CONTEXT`, `FX-PERMISSION-DENIED`, `FX-TAB-ID-REUSE`, and `FX-TAB-SNAPSHOT-CONVERGENCE` under `fixtures/browser/phase-04/` (to-create); repository-root browser fixture runner (to-create)
