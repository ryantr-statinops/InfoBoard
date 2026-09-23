# Phase 19 — Cross-platform browser verification

> Plan ID: IP-19
> Status: See README.md execution tracker
> Execution owner: Cross-platform verification and release owner
> Dependencies: IP-13, IP-14, IP-15, IP-16, IP-17, IP-18
> Parallel boundary: IP-20 only, after this matrix and its evidence are complete or have accepted blocked reasons
> Requirement IDs: NFR-010 (primary); NFR-001, NFR-002, NFR-003, NFR-006, NFR-008, NFR-009 (supporting); FR-001 through FR-015 evidence (supporting)
> Owned paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create)

## 1. Mục tiêu

- Prove the declared compatibility contract for InfoBoard on Chrome and Edge desktop on Linux, macOS, and Windows. The six browser/OS cells are independent acceptance units; a green aggregate cannot hide a missing cell.
- Exercise the complete local product boundary: the configurable browser command opens the focused search surface, the extension searches the current profile's eligible open-tab projection, the Go Native Messaging host performs bounded local work, and confirmed activation reaches the intended tab and window.
- Record reproducible evidence for clean install, startup, upgrade, reconnect, reset, uninstall, failure recovery, permission/privacy boundaries, accessibility, and the three hot-path p95 targets.
- Produce a release-facing evidence ledger that IP-20 can consume without re-running an undocumented scenario. Every cell is PASS, FAIL, or BLOCKED with a precise reason and artifact manifest.

## 2. Phạm vi

### Bao gồm

- Chrome Stable and Edge Stable on Linux, macOS, and Windows, using the exact six-cell matrix in Section 6. Each cell uses a declared reference machine, a clean browser profile, the same synthetic fixture seed, and the same extension/host artifact pair.
- Lifecycle journeys: clean install and host registration, first startup, browser/service-worker restart, upgrade from the previous release, host reconnect after absence or crash, full projection rebuild, reset, and uninstall. The checks must prove idempotence and must not alter unrelated browser state.
- Hot-path measurements: shortcut-to-focused-input (NFR-001, p95 <= 100 ms), query-to-render for 1,000 indexed tabs and a 64-character query (NFR-002, p95 <= 50 ms), and selection-to-activation excluding browser scheduling delay (NFR-003, p95 <= 100 ms). Record p50, p95, p99, every sample, and memory observations.
- Accessibility evidence for keyboard-only operation, visible focus, semantic labels and announcements, contrast, reduced motion, text scaling, and narrow windows on every browser/OS cell.
- Permission and privacy evidence: manifest and Native Messaging origin inspection, zero normal-path network requests, local-only data flow, profile/private-context separation, redacted diagnostics, and reset/uninstall ownership checks.
- Failure journeys for host absence/crash, malformed or oversized frames, permission denial, service-worker suspension, missed events, stale activation results, storage degradation, browser restart, and projection rebuild. Each journey must have an observable user-safe status and a recovery or explicit terminal outcome.
- Evidence capture from synthetic tab metadata only. Evidence may include logs, timings, accessibility reports, screenshots, and videos, but must not expose real URLs, titles, query strings, tokens, or machine secrets.

### Ngoài phạm vi

- Implementing extension, host, protocol, storage, ranking, packaging, or UI behavior; those contracts are supplied by IP-13 through IP-18. This phase owns verification runners, fixtures, reports, and gate decisions only.
- Supporting browsers other than Chrome and Edge desktop, mobile browsers, or operating systems outside Linux, macOS, and Windows.
- Collecting page content, cookies, local storage, network-interception data, whole browsing history, or cloud copies. No network service or loopback listener may be introduced to make a test pass.
- Changing product requirements, permissions, protocol versions, ranking rules, retention, reset ownership, or installer behavior when a check fails. A failure produces evidence and routes back to the owning phase.
- Treating a virtual machine, emulation layer, averaged cross-platform score, or an unrecorded developer workstation as reference performance evidence. Such runs may be exploratory only and must be labeled non-gating.

## 3. Điều kiện tiên quyết

- IP-13, IP-14, IP-15, IP-16, IP-17, and IP-18 are merged and provide immutable contracts for the search surface, activation guard, configuration/reset/uninstall, health states, privacy policy, package, host binary, Native Messaging manifest, allowed origins, and version compatibility.
- The release candidate supplies an extension package, Go Native Messaging host binaries for all three operating systems, native-host registration instructions, extension ID, host name, protocol version, schema version, ranking model version, and checksums. Missing or mismatched artifacts block the affected cell rather than being repaired inside this phase.
- A reference-hardware inventory exists at `fixtures/compatibility/phase-19/reference-hardware.json` (to-create). It records immutable machine ID, CPU model and topology, RAM, storage type, display scale, OS build, browser build, power mode, locale, and background-load policy. Timing runs are on physical reference machines with AC power and no VM or remote desktop.
- The canonical synthetic fixture bundle exists at `fixtures/compatibility/phase-19/fixtures.json` (to-create), with stable IDs, expected projection IDs, expected result IDs, expected health transitions, and no real user data. IP-01 fixture IDs are authoritative; this phase must not create aliases that change their meaning.
- Each test machine can create a fresh browser profile and remove only InfoBoard-owned artifacts. A snapshot or disposable OS account is required for install and uninstall journeys. The runner must record the profile ID without recording account names or secrets.
- The matrix runner, performance runner, accessibility runner, privacy scanner, and recovery runner are planned under `tests/compatibility/phase-19/`, `tests/performance/phase-19/`, `tests/accessibility/phase-19/`, `tests/privacy/phase-19/`, and `tests/recovery/phase-19/`. They are to-create paths; the commands in Sections 6 and 8 are their required interfaces.
- The browser channels, host registration tool, screen-reader/accessibility tooling, process/network instrumentation, and artifact storage are available on each declared machine. If one is unavailable, the cell receives a BLOCKED record with the exact missing capability and owner.

## 4. Đầu ra cần bàn giao

- `fixtures/compatibility/phase-19/reference-hardware.json` (to-create): six-cell hardware/browser inventory and the rules for replacing a machine without invalidating prior evidence.
- `fixtures/compatibility/phase-19/fixtures.json` (to-create): deterministic lifecycle, projection, query, activation-race, failure, privacy, and accessibility fixtures, including the 1,000-tab/64-character benchmark seed.
- `tests/compatibility/phase-19/run-cell` (to-create): one command that executes the complete lifecycle and failure journey set for one browser/OS/reference tuple and writes PASS, FAIL, or BLOCKED evidence.
- `tests/performance/phase-19/measure-hot-path` (to-create): a fixed-sample runner that emits raw monotonic timestamps, p50/p95/p99, memory observations, threshold comparisons, and the exact fixture and artifact hashes.
- `tests/accessibility/phase-19/run` (to-create): automated and manual-check manifest for keyboard, semantics, focus, contrast, reduced motion, text scaling, and narrow-window evidence.
- `tests/privacy/phase-19/audit` (to-create): static permission/origin inspection plus runtime network, storage, log-redaction, profile-isolation, and private-context checks.
- `tests/recovery/phase-19/run` (to-create): host, protocol, projection, activation, storage, service-worker, and browser-restart failure journeys with expected state transitions.
- `artifacts/verification/phase-19/matrix-report.json` (to-create): one row for each Chrome/Edge × Linux/macOS/Windows cell, with environment identity, command lines, commit and package hashes, result status, evidence paths, gate results, and blocked reason when applicable.
- `artifacts/verification/phase-19/README.txt` (to-create): a short replay guide generated from the evidence manifest; it must not contain raw tab data.

## 5. Skill và tài liệu áp dụng

- Skill tags:
  - [testing](../../../.agent/skills/common/engineering/testing/SKILL.md) — defines boundary, invalid, regression, and observable-result coverage for every matrix journey and performance sample.
  - [change-review](../../../.agent/skills/common/delivery/change-review/SKILL.md) — applies the final scope, regression, evidence, and handoff review before a cell or the matrix is accepted.
  - [release](../../../.agent/skills/common/delivery/release/SKILL.md) — checks release artifacts, version compatibility, install/upgrade/uninstall readiness, and rollback evidence.
  - [lightweight-web](../../../.agent/skills/personal/engineering/frontend/lightweight-web/SKILL.md) — guides focused-surface usability checks with semantic structure, minimal dependencies, responsive behavior, and fast feedback.
  - [documentation](../../../.agent/skills/common/engineering/documentation/SKILL.md) — keeps the matrix contract, replay commands, evidence schema, limitations, and gate decisions understandable and verifiable.
  - [task-planning](../../../.agent/skills/common/foundation/task-planning/SKILL.md) — sequences fixtures, runners, environment setup, measurements, and release gates into independently verifiable units.
  - [git-workflow](../../../.agent/skills/common/engineering/git-workflow/SKILL.md) — keeps phase and implementation commits focused, reviewable, and reproducible from a clean checkout.
- Canonical documents in docs/: [product requirements](../refactor/requirements.md), [target architecture](../refactor/architecture.md), [runtime protocol](../refactor/runtime-protocol.md), [user experience](../refactor/user-experience.md), [domain and privacy](../refactor/domain-and-privacy.md), and [verification and acceptance](../refactor/verification-and-acceptance.md). The refactor documents are the contract; this phase records evidence against them and does not redefine them.
- Cross-phase contracts: IP-13 supplies UI states and accessibility seams; IP-14 supplies activation acknowledgement and stale-result guards; IP-15 supplies profile-scoped configuration/reset/uninstall semantics; IP-16 supplies health and redacted diagnostics; IP-17 supplies permission, privacy, and retention assertions; IP-18 supplies packages, registration, checksums, and compatibility metadata.
- Global conventions: use local synthetic data, monotonic clocks, bounded samples, explicit state transitions, deterministic fixture seeds, redacted logs, and fail-closed outcomes. A green result requires the observed behavior and its artifact manifest, not merely a command exit code.

## 6. Công việc triển khai

### 6.1 Freeze the matrix and evidence schema

- [ ] `IP-19-T01` Create `fixtures/compatibility/phase-19/reference-hardware.json` with these required reference classes. The file must be populated with exact immutable device IDs before any cell is marked PASS: R-LINUX (physical x86-64, at least 4 physical cores, 16 GiB RAM, NVMe/SSD, Linux release declared), R-MACOS (physical Apple Silicon or declared supported Mac class, at least 16 GiB RAM, internal SSD, macOS release declared), and R-WINDOWS (physical x86-64, at least 4 physical cores, 16 GiB RAM, NVMe/SSD, Windows release declared). The class is a minimum; the report records the exact CPU, RAM, storage, OS build, browser build, power mode, and display scale.
- [ ] `IP-19-T02` Create a six-row matrix manifest. The required rows and command substitutions are:

| Cell ID | Browser | OS | Reference | Evidence directory | Required command arguments |
| --- | --- | --- | --- | --- | --- |
| C01 | Chrome Stable | Linux | R-LINUX | `artifacts/verification/phase-19/chrome-linux/` | `--browser chrome --os linux --reference R-LINUX` |
| C02 | Edge Stable | Linux | R-LINUX | `artifacts/verification/phase-19/edge-linux/` | `--browser edge --os linux --reference R-LINUX` |
| C03 | Chrome Stable | macOS | R-MACOS | `artifacts/verification/phase-19/chrome-macos/` | `--browser chrome --os macos --reference R-MACOS` |
| C04 | Edge Stable | macOS | R-MACOS | `artifacts/verification/phase-19/edge-macos/` | `--browser edge --os macos --reference R-MACOS` |
| C05 | Chrome Stable | Windows | R-WINDOWS | `artifacts/verification/phase-19/chrome-windows/` | `--browser chrome --os windows --reference R-WINDOWS` |
| C06 | Edge Stable | Windows | R-WINDOWS | `artifacts/verification/phase-19/edge-windows/` | `--browser edge --os windows --reference R-WINDOWS` |

- [ ] `IP-19-T03` Require every cell report to contain cell ID, status, run timestamp, repository commit, extension package SHA-256, host binary SHA-256, Native Messaging manifest SHA-256, protocol/schema/ranking versions, fixture SHA-256, hardware ID, OS/browser build, locale, power mode, exact commands, raw artifact paths, gate statuses, and `blocked_reason` when status is BLOCKED. Empty status or an unqualified “not run” is invalid.
- [ ] `IP-19-T04` Adopt these stable fixture IDs from the canonical fixture inventory: `FX-SHORTCUT-FOCUS`, `FX-QUERY-1000-64`, `FX-ACTIVATION-RACE`, `FX-EVENT-RECONCILE`, `FX-MISSED-EVENT`, `FX-HOST-DOWN`, `FX-HOST-CRASH`, `FX-MALFORMED-FRAME`, `FX-OVERSIZED-FRAME`, `FX-STORAGE-DEGRADED`, `FX-PERMISSION-DENIED`, `FX-PROFILE-ISOLATION`, `FX-PRIVATE-CONTEXT`, `FX-RESET-OWNERSHIP`, `FX-UNINSTALL-OWNERSHIP`, and `FX-ACCESSIBILITY`. Each fixture includes setup, input, expected observable output, cleanup, and synthetic-data assertions.

### 6.2 Execute the lifecycle and compatibility journeys

- [ ] `IP-19-T05` For each C01–C06, run from repository root with the exact future runner interface:

      ./tests/compatibility/phase-19/run-cell --browser <chrome|edge> --os <linux|macos|windows> --reference <R-LINUX|R-MACOS|R-WINDOWS> --fixture-manifest fixtures/compatibility/phase-19/fixtures.json --journeys clean-install,startup,upgrade,reconnect,reset,uninstall --evidence-dir artifacts/verification/phase-19/<browser>-<os>

  Substitute the six explicit argument rows in the matrix; do not run a different browser, OS, reference, or fixture seed under the same cell ID.
- [ ] `IP-19-T06` Clean install: start from a disposable browser profile and clean host registration, install the exact package pair from IP-18, verify the configured browser command appears in Chrome/Edge command settings, launch the focused surface, and verify the query input owns focus before text input. Capture install output, command trace, focused-input trace, package hashes, and initial health state.
- [ ] `IP-19-T07` Startup and suspension: restart the browser, allow the extension service worker to suspend, invoke the command again, and verify hello/ready, profile identity, projection revision, and focused surface recovery. Record host availability, index freshness, and no duplicate projection IDs.
- [ ] `IP-19-T08` Upgrade: install the previous release artifacts, seed configuration and synthetic tabs, upgrade extension and host independently and together, restart the browser, and verify migration, protocol compatibility, retained allowed configuration, projection convergence, and no unrelated browser-state change. Run the upgrade twice to prove idempotence.
- [ ] `IP-19-T09` Reconnect and rebuild: stop or isolate the host, observe a visible unavailable/recovering state, restart the registered host, complete handshake and full snapshot, and verify healthy index readiness, one result per eligible tab, and no stale activation. Run once after a clean browser restart and once after a service-worker suspension.
- [ ] `IP-19-T10` Reset: set non-default profile-scoped configuration, seed diagnostics and activation metadata, invoke reset, and verify only InfoBoard-owned records are removed, safe defaults are restored, open tabs remain open, and unrelated browser data is unchanged. Repeat reset to prove idempotence.
- [ ] `IP-19-T11` Uninstall: remove the extension and native registration using the package's supported uninstaller, verify owned files and profile data are removed, verify open tabs and unrelated browser state remain unchanged, and repeat the uninstaller to prove idempotence. Capture the before/after ownership manifest and process list.
- [ ] `IP-19-T12` Run clean-install, startup, upgrade, reconnect, reset, and uninstall in each matrix cell even when the same host class is reused. A Linux result cannot stand in for macOS or Windows, and Chrome cannot stand in for Edge.

### 6.3 Measure NFR-001, NFR-002, and NFR-003

- [ ] `IP-19-T13` Implement `tests/performance/phase-19/measure-hot-path` with the following exact interface, from repository root:

      ./tests/performance/phase-19/measure-hot-path --browser <chrome|edge> --os <linux|macos|windows> --reference <R-LINUX|R-MACOS|R-WINDOWS> --fixture fixtures/compatibility/phase-19/fixtures.json --tab-count 1000 --query p19-cross-platform-tab-activation-latency-benchmark-query-000000 --samples 101 --warmup 10 --seed p19 --output artifacts/verification/phase-19/<browser>-<os>/performance.json

- [ ] `IP-19-T14` Use the same 101 measured repetitions after 10 discarded warmups per cell, a monotonic clock, no DevTools throttling, AC power, a quiet machine, and the exact deterministic seed. Sort each metric independently; define p95 as rank ceil(0.95 × 101) = 96, and retain p50, p95, p99, minimum, maximum, and all raw samples. Never average the six cells into one pass/fail value.
- [ ] `IP-19-T15` Measure the boundaries exactly: NFR-001 starts at the browser command dispatch and stops only when the focused search surface is visible and `document.activeElement` is the query input; NFR-002 starts when a query revision is accepted and stops when the matching result revision is rendered and observable; NFR-003 starts at highlighted-result Enter keydown and stops at the browser activation acknowledgement, while separately recording browser scheduling delay and excluding it from the NFR-003 duration.
- [ ] `IP-19-T16` Seed a real indexed projection of exactly 1,000 eligible synthetic tab records for `FX-QUERY-1000-64`. The query is exactly 64 characters: `p19-cross-platform-tab-activation-latency-benchmark-query-000000`. The report must include projection revision, query revision, result count, and checksum of the expected ordered IDs so a fast but incomplete result cannot pass.
- [ ] `IP-19-T17` Compare each cell to the gates below and record a machine-readable result. A cell passes only when all three measured p95 values meet their limits and no measurement setup exception is open.

| Requirement | Measurement | Gate per matrix cell | Evidence |
| --- | --- | --- | --- |
| NFR-001 | Shortcut dispatch → focused visible input | p95 <= 100 ms | Raw samples, timing summary, focus trace, screen capture hash |
| NFR-002 | Query accepted → rendered result revision, 1,000 tabs and 64-character query | p95 <= 50 ms | Raw samples, result-revision trace, projection/query fixture hash |
| NFR-003 | Enter keydown → activation acknowledgement, scheduling delay excluded | p95 <= 100 ms | Raw samples, activation acknowledgement trace, browser scheduling trace |

- [ ] `IP-19-T18` Also record p50/p99 and process/browser memory at setup, after warmup, after the measured run, and after 10 repeated runs. A memory increase without a bounded explanation is a release risk even when latency passes. Do not silently discard outliers; annotate GC, OS scheduling, browser restart, or fixture failures in the raw record.

### 6.4 Verify accessibility on each browser/OS cell

- [ ] `IP-19-T19` Run from repository root with the exact future interface:

      ./tests/accessibility/phase-19/run --browser <chrome|edge> --os <linux|macos|windows> --reference <R-LINUX|R-MACOS|R-WINDOWS> --fixture FX-ACCESSIBILITY --viewport 320x480 --text-scale 200 --reduced-motion --keyboard-only --output artifacts/verification/phase-19/<browser>-<os>/accessibility.json

- [ ] `IP-19-T20` Trace the pointer-free journey: invoke the configured command, confirm the query input is focused, type a synthetic query, move with ArrowDown and ArrowUp, activate with Enter, dismiss with Escape, reopen, and verify focus is restored. Capture key events, active element, selected result, activation acknowledgement, dismissal, and browser state.
- [ ] `IP-19-T21` Inspect semantic labels, roles, descriptions, live status announcements, result count changes, visible focus indicator, contrast, and error/degraded/recovering states. Use the supported browser accessibility tree and an automated accessibility audit; record rule IDs, versions, findings, and any accepted manual-review reason.
- [ ] `IP-19-T22` Repeat at 200% text scale and a 320 × 480 CSS-pixel window. Enable reduced motion and verify no required state depends on animation. For each OS, include the supported screen reader evidence (Orca on Linux, VoiceOver on macOS, and Narrator or the declared Windows screen reader) with synthetic content only.
- [ ] `IP-19-T23` Mark NFR-008 PASS only when keyboard operation, visible focus, semantics, contrast, reduced motion, text scaling, and narrow-window behavior all pass in the cell; otherwise retain the failing trace and route the defect to IP-13.

### 6.5 Verify permissions, privacy, and profile boundaries

- [ ] `IP-19-T24` Run from repository root with the exact future interface:

      ./tests/privacy/phase-19/audit --browser <chrome|edge> --os <linux|macos|windows> --reference <R-LINUX|R-MACOS|R-WINDOWS> --extension-package <path-to-package> --host-manifest <path-to-native-manifest> --fixture fixtures/compatibility/phase-19/fixtures.json --output artifacts/verification/phase-19/<browser>-<os>/privacy.json

- [ ] `IP-19-T25` Compare manifest permissions, optional permissions, host origins, and host registration paths with the IP-17 matrix and IP-18 package manifest. Reject any unapproved permission for history, downloads, cookies, broad host access, page content, or unrelated storage. Record the complete inspected lists and hashes, not just a boolean.
- [ ] `IP-19-T26` Instrument the normal path from install through query and activation and assert zero network requests, zero loopback listeners, no page evaluation or content reads, no command execution outside the registered host boundary, and no data sent off-machine. Save request/listener/API summaries with URL, title, query, and token fields redacted or omitted.
- [ ] `IP-19-T27` Use two disposable browser profiles and one private-context run. Prove that profile A cannot query or activate profile B, private-context records are excluded or cleaned according to IP-17, configuration remains profile-scoped, and reset/uninstall do not alter unrelated profile data. Capture synthetic IDs and counts only.
- [ ] `IP-19-T28` Export diagnostics after host-down, reconnect, stale-result, and storage-degraded journeys. Assert that logs contain metadata such as counts, durations, versions, and error classes but no raw URLs, titles, query strings, tokens, or secrets. Record redaction findings and retention cleanup results.
- [ ] `IP-19-T29` Mark NFR-009 PASS only when static permissions, runtime data flow, profile/private isolation, diagnostics redaction, and ownership cleanup all pass. Any high-severity privacy finding blocks the cell and the release gate.

### 6.6 Exercise failure and recovery journeys

- [ ] `IP-19-T30` Run from repository root with the exact future interface:

      ./tests/recovery/phase-19/run --browser <chrome|edge> --os <linux|macos|windows> --reference <R-LINUX|R-MACOS|R-WINDOWS> --fixture-manifest fixtures/compatibility/phase-19/fixtures.json --output artifacts/verification/phase-19/<browser>-<os>/recovery.json

- [ ] `IP-19-T31` Use the following fixture-to-observable contract in every cell:

| Fixture | Injected condition | Required observable result and evidence |
| --- | --- | --- |
| FX-HOST-DOWN | Host absent before query | Surface reports unavailable/retryable state; no false results or activation; health/error class and recovery action captured |
| FX-HOST-CRASH | Host exits during session | Recovering state, reconnect, full snapshot, healthy index, no duplicate IDs; NFR-006 trace captured |
| FX-MISSED-EVENT | Event sequence has a gap | Snapshot/resync is requested; indexed eligible IDs equal browser snapshot IDs; FR-010 trace captured |
| FX-EVENT-RECONCILE | Create/update/move/group/pin/activate/window/remove sequence | Projection converges with no duplicate records and expected context fields; FR-009 trace captured |
| FX-ACTIVATION-RACE | Selected tab disappears or changes before Enter | Stale result is rejected or refreshed; a different tab is never activated silently; FR-011 trace captured |
| FX-MALFORMED-FRAME | Invalid JSON/envelope/request identity | Fail-closed typed error and visible recoverable state; no process or browser-state corruption |
| FX-OVERSIZED-FRAME | Payload exceeds declared limit | PAYLOAD_LIMIT-style failure, bounded resource use, repair/retry guidance; no partial projection commit |
| FX-STORAGE-DEGRADED | SQLite unavailable or corrupt | Diagnostic state is visible; lexical search and valid activation remain available; FR-013 trace captured |
| FX-PERMISSION-DENIED | Required browser capability denied | Clear permission/recovery guidance; no broadened permission request or silent data loss |
| FX-PRIVATE-CONTEXT | Private window/profile is active | Contract-defined exclusion and cleanup hold; no cross-context result or persisted sensitive record |
| FX-RESET-OWNERSHIP | Reset during healthy and recovering states | Only owned records/configuration are removed, safe defaults restored, tabs and unrelated data unchanged |
| FX-UNINSTALL-OWNERSHIP | Uninstall and repeated uninstall | Owned registration/files/data removed; tabs and unrelated browser data unchanged; operation is idempotent |

- [ ] `IP-19-T32` Include service-worker suspension and browser restart in both FX-HOST-CRASH and FX-MISSED-EVENT runs. Record state sequence, projection revision before/after, result revision, host health, index freshness, duplicate-ID count, and recovery duration.
- [ ] `IP-19-T33` Verify every error is diagnosable and user-safe: no unconfirmed activation, no hidden recoverable failure, no unbounded retry loop, and no raw sensitive data in the evidence. Route failures to IP-14, IP-16, or IP-17 according to the owning contract.

### 6.7 Map evidence to requirements and release gates

- [ ] `IP-19-T34` Generate a requirement traceability table in `matrix-report.json`. The primary owner remains the phase named by the implementation index; IP-19 contributes cross-platform evidence only.

| Requirement | Evidence captured in this phase | Observable acceptance signal |
| --- | --- | --- |
| FR-001 | FX-SHORTCUT-FOCUS in C01–C06 | Chrome and Edge command settings expose one command and open the surface |
| FR-002 | FX-SHORTCUT-FOCUS and accessibility keyboard trace | Query input is focused before text is accepted without pointer input |
| FR-003 | FX-QUERY-1000-64, field-query fixture | Current-profile open-tab fields return expected matches and exclude unavailable data |
| FR-004 | Live-query trace in FX-QUERY-1000-64 | Each accepted query revision renders its corresponding result revision |
| FR-005 | Repeated ranking checksum in performance report | Same projection, query, and ranking timestamp produce identical ordered IDs and scores |
| FR-006 | Duplicate-title/context accessibility capture | Title, domain/URL context, window/group context, and pinned state remain distinguishable |
| FR-007 | Keyboard activation trace | Arrow selection and Enter activate the confirmed intended tab/window |
| FR-008 | Escape/dismissal trace | Surface closes and selected browser tab remains unchanged |
| FR-009 | FX-EVENT-RECONCILE | Tab and window event sequence converges without duplicate records |
| FR-010 | FX-MISSED-EVENT and reconnect report | Full snapshot after a missed event reaches browser-visible eligible IDs |
| FR-011 | FX-ACTIVATION-RACE | Removed result never activates a different tab silently |
| FR-012 | FX-HOST-DOWN, FX-HOST-CRASH, health capture | Availability and index freshness are visible and actionable |
| FR-013 | FX-STORAGE-DEGRADED | Lexical search and valid activation remain usable when optional persistence fails |
| FR-014 | FX-PROFILE-ISOLATION and FX-PRIVATE-CONTEXT | Profiles and contexts do not share configuration or projections |
| FR-015 | FX-RESET-OWNERSHIP and FX-UNINSTALL-OWNERSHIP | Owned local data is removed without changing tabs or unrelated browser data |
| NFR-001 | Per-cell performance.json | Shortcut-to-focused-input p95 <= 100 ms |
| NFR-002 | Per-cell performance.json | 1,000-tab/64-character query-to-render p95 <= 50 ms |
| NFR-003 | Per-cell performance.json | Selection-to-activation p95 <= 100 ms excluding browser scheduling |
| NFR-006 | Per-cell recovery.json | Host reconnect/rebuild reaches healthy state without reinstall |
| NFR-008 | Per-cell accessibility.json | Keyboard, focus, contrast, reduced-motion, semantic, and text-scale checks pass |
| NFR-009 | Per-cell privacy.json | No excluded source is requested, collected, persisted, or transmitted |
| NFR-010 | Six-row matrix-report.json | Chrome and Edge each pass or have an explicitly accepted exception on Linux, macOS, and Windows |

- [ ] `IP-19-T35` Include supporting evidence for projection convergence, ranking determinism, resource bounds, and operational timeout/error-class rules without claiming their primary ownership. Forward the same artifact hashes to IP-05, IP-11, IP-16, and IP-18 as applicable.
- [ ] `IP-19-T36` Require the final gate ledger to contain performance, accessibility, privacy, recovery, compatibility, install/upgrade/uninstall, and documentation/protocol consistency statuses. A gate is PASS, FAIL, or BLOCKED with an owner and evidence path; no blank or implicit status is accepted.

## 7. Kế hoạch commit

1. `test(compatibility): implement ip-19-t01`
   - Task IDs: `IP-19-T01`.
   - Owned target paths: fixtures/compatibility/phase-19/reference-hardware.json.
   - Behavior: Create `fixtures/compatibility/phase-19/reference-hardware.json` with these required reference classes. The file must be populated with exact immutable device IDs before any cell is marked PASS: R-LINUX (physical x86-64, at least 4 physical cores, 16 GiB RAM, NVMe/SSD, Linux release declared), R-MACOS (physical Apple Silicon or declared supported Mac class, at least 16 GiB RAM, internal SSD, macOS release declared), and R-WINDOWS (physical x86-64, at least 4 physical cores, 16 GiB RAM, NVMe/SSD, Windows release declared). The class is a minimum; the report records the exact CPU, RAM, storage, OS build, browser build, power mode, and display scale.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Create `fixtures/compatibility/phase-19/reference-hardware.json` with these required reference classes. The file must be populated with exact immutable device IDs before any cell is marked PASS: R-LINUX (physical x86-64, at least 4 physical cores, 16 GiB RAM, NVMe/SSD, Linux release declared), R-MACOS (physical Apple Silicon or declared supported Mac class, at least 16 GiB RAM, internal SSD, macOS release declared), and R-WINDOWS (physical x86-64, at least 4 physical cores, 16 GiB RAM, NVMe/SSD, Windows release declared). The class is a minimum; the report records the exact CPU, RAM, storage, OS build, browser build, power mode, and display scale.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

2. `feat(compatibility): implement ip-19-t02`
   - Task IDs: `IP-19-T02`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Create a six-row matrix manifest. The required rows and command substitutions are:
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Create a six-row matrix manifest. The required rows and command substitutions are:
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

3. `test(compatibility): implement ip-19-t03`
   - Task IDs: `IP-19-T03`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Require every cell report to contain cell ID, status, run timestamp, repository commit, extension package SHA-256, host binary SHA-256, Native Messaging manifest SHA-256, protocol/schema/ranking versions, fixture SHA-256, hardware ID, OS/browser build, locale, power mode, exact commands, raw artifact paths, gate statuses, and `blocked_reason` when status is BLOCKED. Empty status or an unqualified “not run” is invalid.
   - Fixture and command: SHA-256; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Require every cell report to contain cell ID, status, run timestamp, repository commit, extension package SHA-256, host binary SHA-256, Native Messaging manifest SHA-256, protocol/schema/ranking versions, fixture SHA-256, hardware ID, OS/browser build, locale, power mode, exact commands, raw artifact paths, gate statuses, and `blocked_reason` when status is BLOCKED. Empty status or an unqualified “not run” is invalid.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

4. `test(compatibility): implement ip-19-t04`
   - Task IDs: `IP-19-T04`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Adopt these stable fixture IDs from the canonical fixture inventory: `FX-SHORTCUT-FOCUS`, `FX-QUERY-1000-64`, `FX-ACTIVATION-RACE`, `FX-EVENT-RECONCILE`, `FX-MISSED-EVENT`, `FX-HOST-DOWN`, `FX-HOST-CRASH`, `FX-MALFORMED-FRAME`, `FX-OVERSIZED-FRAME`, `FX-STORAGE-DEGRADED`, `FX-PERMISSION-DENIED`, `FX-PROFILE-ISOLATION`, `FX-PRIVATE-CONTEXT`, `FX-RESET-OWNERSHIP`, `FX-UNINSTALL-OWNERSHIP`, and `FX-ACCESSIBILITY`. Each fixture includes setup, input, expected observable output, cleanup, and synthetic-data assertions.
   - Fixture and command: FX-SHORTCUT-FOCUS, FX-QUERY-1000-64, FX-ACTIVATION-RACE, FX-EVENT-RECONCILE, FX-MISSED-EVENT, FX-HOST-DOWN, FX-HOST-CRASH, FX-MALFORMED-FRAME, FX-OVERSIZED-FRAME, FX-STORAGE-DEGRADED, FX-PERMISSION-DENIED, FX-PROFILE-ISOLATION, FX-PRIVATE-CONTEXT, FX-RESET-OWNERSHIP, FX-UNINSTALL-OWNERSHIP, FX-ACCESSIBILITY; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Adopt these stable fixture IDs from the canonical fixture inventory: `FX-SHORTCUT-FOCUS`, `FX-QUERY-1000-64`, `FX-ACTIVATION-RACE`, `FX-EVENT-RECONCILE`, `FX-MISSED-EVENT`, `FX-HOST-DOWN`, `FX-HOST-CRASH`, `FX-MALFORMED-FRAME`, `FX-OVERSIZED-FRAME`, `FX-STORAGE-DEGRADED`, `FX-PERMISSION-DENIED`, `FX-PROFILE-ISOLATION`, `FX-PRIVATE-CONTEXT`, `FX-RESET-OWNERSHIP`, `FX-UNINSTALL-OWNERSHIP`, and `FX-ACCESSIBILITY`. Each fixture includes setup, input, expected observable output, cleanup, and synthetic-data assertions.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

5. `feat(compatibility): implement ip-19-t05`
   - Task IDs: `IP-19-T05`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: For each C01–C06, run from repository root with the exact future runner interface:
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: For each C01–C06, run from repository root with the exact future runner interface:
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

6. `feat(compatibility): implement ip-19-t06`
   - Task IDs: `IP-19-T06`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Clean install: start from a disposable browser profile and clean host registration, install the exact package pair from IP-18, verify the configured browser command appears in Chrome/Edge command settings, launch the focused surface, and verify the query input owns focus before text input. Capture install output, command trace, focused-input trace, package hashes, and initial health state.
   - Fixture and command: IP-18; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Clean install: start from a disposable browser profile and clean host registration, install the exact package pair from IP-18, verify the configured browser command appears in Chrome/Edge command settings, launch the focused surface, and verify the query input owns focus before text input. Capture install output, command trace, focused-input trace, package hashes, and initial health state.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

7. `feat(compatibility): implement ip-19-t07`
   - Task IDs: `IP-19-T07`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Startup and suspension: restart the browser, allow the extension service worker to suspend, invoke the command again, and verify hello/ready, profile identity, projection revision, and focused surface recovery. Record host availability, index freshness, and no duplicate projection IDs.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Startup and suspension: restart the browser, allow the extension service worker to suspend, invoke the command again, and verify hello/ready, profile identity, projection revision, and focused surface recovery. Record host availability, index freshness, and no duplicate projection IDs.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

8. `feat(compatibility): implement ip-19-t08`
   - Task IDs: `IP-19-T08`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Upgrade: install the previous release artifacts, seed configuration and synthetic tabs, upgrade extension and host independently and together, restart the browser, and verify migration, protocol compatibility, retained allowed configuration, projection convergence, and no unrelated browser-state change. Run the upgrade twice to prove idempotence.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Upgrade: install the previous release artifacts, seed configuration and synthetic tabs, upgrade extension and host independently and together, restart the browser, and verify migration, protocol compatibility, retained allowed configuration, projection convergence, and no unrelated browser-state change. Run the upgrade twice to prove idempotence.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

9. `feat(compatibility): implement ip-19-t09`
   - Task IDs: `IP-19-T09`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Reconnect and rebuild: stop or isolate the host, observe a visible unavailable/recovering state, restart the registered host, complete handshake and full snapshot, and verify healthy index readiness, one result per eligible tab, and no stale activation. Run once after a clean browser restart and once after a service-worker suspension.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Reconnect and rebuild: stop or isolate the host, observe a visible unavailable/recovering state, restart the registered host, complete handshake and full snapshot, and verify healthy index readiness, one result per eligible tab, and no stale activation. Run once after a clean browser restart and once after a service-worker suspension.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

10. `feat(compatibility): implement ip-19-t10`
   - Task IDs: `IP-19-T10`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Reset: set non-default profile-scoped configuration, seed diagnostics and activation metadata, invoke reset, and verify only InfoBoard-owned records are removed, safe defaults are restored, open tabs remain open, and unrelated browser data is unchanged. Repeat reset to prove idempotence.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Reset: set non-default profile-scoped configuration, seed diagnostics and activation metadata, invoke reset, and verify only InfoBoard-owned records are removed, safe defaults are restored, open tabs remain open, and unrelated browser data is unchanged. Repeat reset to prove idempotence.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

11. `feat(compatibility): implement ip-19-t11`
   - Task IDs: `IP-19-T11`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Uninstall: remove the extension and native registration using the package's supported uninstaller, verify owned files and profile data are removed, verify open tabs and unrelated browser state remain unchanged, and repeat the uninstaller to prove idempotence. Capture the before/after ownership manifest and process list.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Uninstall: remove the extension and native registration using the package's supported uninstaller, verify owned files and profile data are removed, verify open tabs and unrelated browser state remain unchanged, and repeat the uninstaller to prove idempotence. Capture the before/after ownership manifest and process list.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

12. `feat(compatibility): implement ip-19-t12`
   - Task IDs: `IP-19-T12`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Run clean-install, startup, upgrade, reconnect, reset, and uninstall in each matrix cell even when the same host class is reused. A Linux result cannot stand in for macOS or Windows, and Chrome cannot stand in for Edge.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Run clean-install, startup, upgrade, reconnect, reset, and uninstall in each matrix cell even when the same host class is reused. A Linux result cannot stand in for macOS or Windows, and Chrome cannot stand in for Edge.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

13. `test(compatibility): implement ip-19-t13`
   - Task IDs: `IP-19-T13`.
   - Owned target paths: tests/performance/phase-19/measure-hot-path.
   - Behavior: Implement `tests/performance/phase-19/measure-hot-path` with the following exact interface, from repository root:
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Implement `tests/performance/phase-19/measure-hot-path` with the following exact interface, from repository root:
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

14. `test(compatibility): implement ip-19-t14`
   - Task IDs: `IP-19-T14`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Use the same 101 measured repetitions after 10 discarded warmups per cell, a monotonic clock, no DevTools throttling, AC power, a quiet machine, and the exact deterministic seed. Sort each metric independently; define p95 as rank ceil(0.95 × 101) = 96, and retain p50, p95, p99, minimum, maximum, and all raw samples. Never average the six cells into one pass/fail value.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Use the same 101 measured repetitions after 10 discarded warmups per cell, a monotonic clock, no DevTools throttling, AC power, a quiet machine, and the exact deterministic seed. Sort each metric independently; define p95 as rank ceil(0.95 × 101) = 96, and retain p50, p95, p99, minimum, maximum, and all raw samples. Never average the six cells into one pass/fail value.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

15. `test(compatibility): implement ip-19-t15`
   - Task IDs: `IP-19-T15`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Measure the boundaries exactly: NFR-001 starts at the browser command dispatch and stops only when the focused search surface is visible and `document.activeElement` is the query input; NFR-002 starts when a query revision is accepted and stops when the matching result revision is rendered and observable; NFR-003 starts at highlighted-result Enter keydown and stops at the browser activation acknowledgement, while separately recording browser scheduling delay and excluding it from the NFR-003 duration.
   - Fixture and command: NFR-001, NFR-002, NFR-003; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Measure the boundaries exactly: NFR-001 starts at the browser command dispatch and stops only when the focused search surface is visible and `document.activeElement` is the query input; NFR-002 starts when a query revision is accepted and stops when the matching result revision is rendered and observable; NFR-003 starts at highlighted-result Enter keydown and stops at the browser activation acknowledgement, while separately recording browser scheduling delay and excluding it from the NFR-003 duration.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

16. `test(compatibility): implement ip-19-t16`
   - Task IDs: `IP-19-T16`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Seed a real indexed projection of exactly 1,000 eligible synthetic tab records for `FX-QUERY-1000-64`. The query is exactly 64 characters: `p19-cross-platform-tab-activation-latency-benchmark-query-000000`. The report must include projection revision, query revision, result count, and checksum of the expected ordered IDs so a fast but incomplete result cannot pass.
   - Fixture and command: FX-QUERY-1000-64; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Seed a real indexed projection of exactly 1,000 eligible synthetic tab records for `FX-QUERY-1000-64`. The query is exactly 64 characters: `p19-cross-platform-tab-activation-latency-benchmark-query-000000`. The report must include projection revision, query revision, result count, and checksum of the expected ordered IDs so a fast but incomplete result cannot pass.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

17. `test(compatibility): implement ip-19-t17`
   - Task IDs: `IP-19-T17`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Compare each cell to the gates below and record a machine-readable result. A cell passes only when all three measured p95 values meet their limits and no measurement setup exception is open.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Compare each cell to the gates below and record a machine-readable result. A cell passes only when all three measured p95 values meet their limits and no measurement setup exception is open.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

18. `test(compatibility): implement ip-19-t18`
   - Task IDs: `IP-19-T18`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Also record p50/p99 and process/browser memory at setup, after warmup, after the measured run, and after 10 repeated runs. A memory increase without a bounded explanation is a release risk even when latency passes. Do not silently discard outliers; annotate GC, OS scheduling, browser restart, or fixture failures in the raw record.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Also record p50/p99 and process/browser memory at setup, after warmup, after the measured run, and after 10 repeated runs. A memory increase without a bounded explanation is a release risk even when latency passes. Do not silently discard outliers; annotate GC, OS scheduling, browser restart, or fixture failures in the raw record.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

19. `feat(compatibility): implement ip-19-t19`
   - Task IDs: `IP-19-T19`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Run from repository root with the exact future interface:
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Run from repository root with the exact future interface:
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

20. `feat(compatibility): implement ip-19-t20`
   - Task IDs: `IP-19-T20`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Trace the pointer-free journey: invoke the configured command, confirm the query input is focused, type a synthetic query, move with ArrowDown and ArrowUp, activate with Enter, dismiss with Escape, reopen, and verify focus is restored. Capture key events, active element, selected result, activation acknowledgement, dismissal, and browser state.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Trace the pointer-free journey: invoke the configured command, confirm the query input is focused, type a synthetic query, move with ArrowDown and ArrowUp, activate with Enter, dismiss with Escape, reopen, and verify focus is restored. Capture key events, active element, selected result, activation acknowledgement, dismissal, and browser state.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

21. `test(compatibility): implement ip-19-t21`
   - Task IDs: `IP-19-T21`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Inspect semantic labels, roles, descriptions, live status announcements, result count changes, visible focus indicator, contrast, and error/degraded/recovering states. Use the supported browser accessibility tree and an automated accessibility audit; record rule IDs, versions, findings, and any accepted manual-review reason.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Inspect semantic labels, roles, descriptions, live status announcements, result count changes, visible focus indicator, contrast, and error/degraded/recovering states. Use the supported browser accessibility tree and an automated accessibility audit; record rule IDs, versions, findings, and any accepted manual-review reason.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

22. `feat(compatibility): implement ip-19-t22`
   - Task IDs: `IP-19-T22`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Repeat at 200% text scale and a 320 × 480 CSS-pixel window. Enable reduced motion and verify no required state depends on animation. For each OS, include the supported screen reader evidence (Orca on Linux, VoiceOver on macOS, and Narrator or the declared Windows screen reader) with synthetic content only.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Repeat at 200% text scale and a 320 × 480 CSS-pixel window. Enable reduced motion and verify no required state depends on animation. For each OS, include the supported screen reader evidence (Orca on Linux, VoiceOver on macOS, and Narrator or the declared Windows screen reader) with synthetic content only.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

23. `feat(compatibility): implement ip-19-t23`
   - Task IDs: `IP-19-T23`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Mark NFR-008 PASS only when keyboard operation, visible focus, semantics, contrast, reduced motion, text scaling, and narrow-window behavior all pass in the cell; otherwise retain the failing trace and route the defect to IP-13.
   - Fixture and command: NFR-008, IP-13; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Mark NFR-008 PASS only when keyboard operation, visible focus, semantics, contrast, reduced motion, text scaling, and narrow-window behavior all pass in the cell; otherwise retain the failing trace and route the defect to IP-13.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

24. `feat(compatibility): implement ip-19-t24`
   - Task IDs: `IP-19-T24`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Run from repository root with the exact future interface:
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Run from repository root with the exact future interface:
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

25. `feat(compatibility): implement ip-19-t25`
   - Task IDs: `IP-19-T25`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Compare manifest permissions, optional permissions, host origins, and host registration paths with the IP-17 matrix and IP-18 package manifest. Reject any unapproved permission for history, downloads, cookies, broad host access, page content, or unrelated storage. Record the complete inspected lists and hashes, not just a boolean.
   - Fixture and command: IP-17, IP-18; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Compare manifest permissions, optional permissions, host origins, and host registration paths with the IP-17 matrix and IP-18 package manifest. Reject any unapproved permission for history, downloads, cookies, broad host access, page content, or unrelated storage. Record the complete inspected lists and hashes, not just a boolean.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

26. `feat(compatibility): implement ip-19-t26`
   - Task IDs: `IP-19-T26`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Instrument the normal path from install through query and activation and assert zero network requests, zero loopback listeners, no page evaluation or content reads, no command execution outside the registered host boundary, and no data sent off-machine. Save request/listener/API summaries with URL, title, query, and token fields redacted or omitted.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Instrument the normal path from install through query and activation and assert zero network requests, zero loopback listeners, no page evaluation or content reads, no command execution outside the registered host boundary, and no data sent off-machine. Save request/listener/API summaries with URL, title, query, and token fields redacted or omitted.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

27. `feat(compatibility): implement ip-19-t27`
   - Task IDs: `IP-19-T27`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Use two disposable browser profiles and one private-context run. Prove that profile A cannot query or activate profile B, private-context records are excluded or cleaned according to IP-17, configuration remains profile-scoped, and reset/uninstall do not alter unrelated profile data. Capture synthetic IDs and counts only.
   - Fixture and command: IP-17; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Use two disposable browser profiles and one private-context run. Prove that profile A cannot query or activate profile B, private-context records are excluded or cleaned according to IP-17, configuration remains profile-scoped, and reset/uninstall do not alter unrelated profile data. Capture synthetic IDs and counts only.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

28. `feat(compatibility): implement ip-19-t28`
   - Task IDs: `IP-19-T28`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Export diagnostics after host-down, reconnect, stale-result, and storage-degraded journeys. Assert that logs contain metadata such as counts, durations, versions, and error classes but no raw URLs, titles, query strings, tokens, or secrets. Record redaction findings and retention cleanup results.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Export diagnostics after host-down, reconnect, stale-result, and storage-degraded journeys. Assert that logs contain metadata such as counts, durations, versions, and error classes but no raw URLs, titles, query strings, tokens, or secrets. Record redaction findings and retention cleanup results.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

29. `feat(compatibility): implement ip-19-t29`
   - Task IDs: `IP-19-T29`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Mark NFR-009 PASS only when static permissions, runtime data flow, profile/private isolation, diagnostics redaction, and ownership cleanup all pass. Any high-severity privacy finding blocks the cell and the release gate.
   - Fixture and command: NFR-009; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Mark NFR-009 PASS only when static permissions, runtime data flow, profile/private isolation, diagnostics redaction, and ownership cleanup all pass. Any high-severity privacy finding blocks the cell and the release gate.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

30. `feat(compatibility): implement ip-19-t30`
   - Task IDs: `IP-19-T30`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Run from repository root with the exact future interface:
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Run from repository root with the exact future interface:
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

31. `test(compatibility): implement ip-19-t31`
   - Task IDs: `IP-19-T31`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Use the following fixture-to-observable contract in every cell:
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Use the following fixture-to-observable contract in every cell:
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

32. `feat(compatibility): implement ip-19-t32`
   - Task IDs: `IP-19-T32`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Include service-worker suspension and browser restart in both FX-HOST-CRASH and FX-MISSED-EVENT runs. Record state sequence, projection revision before/after, result revision, host health, index freshness, duplicate-ID count, and recovery duration.
   - Fixture and command: FX-HOST-CRASH, FX-MISSED-EVENT; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Include service-worker suspension and browser restart in both FX-HOST-CRASH and FX-MISSED-EVENT runs. Record state sequence, projection revision before/after, result revision, host health, index freshness, duplicate-ID count, and recovery duration.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

33. `feat(compatibility): implement ip-19-t33`
   - Task IDs: `IP-19-T33`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Verify every error is diagnosable and user-safe: no unconfirmed activation, no hidden recoverable failure, no unbounded retry loop, and no raw sensitive data in the evidence. Route failures to IP-14, IP-16, or IP-17 according to the owning contract.
   - Fixture and command: IP-14, IP-16, IP-17; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Verify every error is diagnosable and user-safe: no unconfirmed activation, no hidden recoverable failure, no unbounded retry loop, and no raw sensitive data in the evidence. Route failures to IP-14, IP-16, or IP-17 according to the owning contract.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

34. `feat(compatibility): implement ip-19-t34`
   - Task IDs: `IP-19-T34`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Generate a requirement traceability table in `matrix-report.json`. The primary owner remains the phase named by the implementation index; IP-19 contributes cross-platform evidence only.
   - Fixture and command: IP-19; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Generate a requirement traceability table in `matrix-report.json`. The primary owner remains the phase named by the implementation index; IP-19 contributes cross-platform evidence only.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

35. `feat(compatibility): implement ip-19-t35`
   - Task IDs: `IP-19-T35`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Include supporting evidence for projection convergence, ranking determinism, resource bounds, and operational timeout/error-class rules without claiming their primary ownership. Forward the same artifact hashes to IP-05, IP-11, IP-16, and IP-18 as applicable.
   - Fixture and command: IP-05, IP-11, IP-16, IP-18; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Include supporting evidence for projection convergence, ranking determinism, resource bounds, and operational timeout/error-class rules without claiming their primary ownership. Forward the same artifact hashes to IP-05, IP-11, IP-16, and IP-18 as applicable.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

36. `feat(compatibility): implement ip-19-t36`
   - Task IDs: `IP-19-T36`.
   - Owned target paths: `fixtures/compatibility/phase-19/` (to-create); `tests/compatibility/phase-19/` (to-create); `tests/performance/phase-19/` (to-create); `tests/accessibility/phase-19/` (to-create); `tests/privacy/phase-19/` (to-create); `tests/recovery/phase-19/` (to-create); `artifacts/verification/phase-19/` (to-create).
   - Behavior: Require the final gate ledger to contain performance, accessibility, privacy, recovery, compatibility, install/upgrade/uninstall, and documentation/protocol consistency statuses. A gate is PASS, FAIL, or BLOCKED with an owner and evidence path; no blank or implicit status is accepted.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-19 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Require the final gate ledger to contain performance, accessibility, privacy, recovery, compatibility, install/upgrade/uninstall, and documentation/protocol consistency statuses. A gate is PASS, FAIL, or BLOCKED with an owner and evidence path; no blank or implicit status is accepted.
   - Dependency gate: all index.md dependencies for IP-19 have merged to dev; phase work branch starts from latest origin/dev.

## 8. Kiểm chứng và nghiệm thu

- [ ] From the repository root, run `git diff --check` and verify the implementation branch contains only the phase-19 plan during documentation authoring.
- [ ] Populate exact reference-hardware IDs and package/host/protocol/schema/ranking hashes before running a gate. A missing reference machine, missing package, or missing runner is an explicit BLOCKED reason, never a PASS.
- [ ] Run the Section 6.2 lifecycle command for C01, C02, C03, C04, C05, and C06. Each row has a reproducible evidence directory or a BLOCKED record containing blocker, owner, attempted command, environment identity, and next decision date.
- [ ] Run the Section 6.3 performance command for all six cells. Confirm p95 NFR-001 <= 100 ms, NFR-002 <= 50 ms, and NFR-003 <= 100 ms, with raw 101-sample traces and scheduling-delay annotation.
- [ ] Run the Section 6.4 accessibility command for all six cells. Confirm keyboard-only open/query/navigate/activate/dismiss, focus visibility, semantic labels and announcements, contrast, reduced motion, 200% text scaling, and 320 × 480 behavior.
- [ ] Run the Section 6.5 privacy command for all six cells. Confirm permission/origin allowlists, zero normal-path network requests and listeners, local-only data flow, profile/private separation, redacted diagnostics, and owned-data cleanup.
- [ ] Run the Section 6.6 recovery command for all six cells. Confirm host absence/crash, malformed/oversized frames, missed events, stale activation, storage degradation, permission denial, service-worker suspension, and browser restart have visible and recoverable or explicitly terminal outcomes.
- [ ] Inspect `matrix-report.json` and ensure all six rows have PASS, FAIL, or BLOCKED; no row is missing evidence, an artifact hash, or a `blocked_reason` when blocked. Every FR-001 through FR-015 and the required NFR IDs map to an observable artifact.
- [ ] Apply the release gates from the verification contract: compatibility is green or has an accepted exception, performance is within target without unexplained memory growth, accessibility and privacy have no unresolved high-severity findings, recovery/install/update/reset/uninstall pass, and artifact/protocol/schema/ranking/permission metadata agree.
- [ ] Re-run the winning cell commands from a clean checkout using the evidence manifest. The replay must produce the same fixture checksums, requirement outcomes, and compatibility decision without access to real user data.

## 9. Rủi ro và quyết định còn mở

- Risk: Reference machines drift in CPU load, power mode, browser channel, or OS patch level and invalidate p95 comparisons. Mitigation: immutable machine IDs, AC power, declared load policy, exact builds, 10 warmups plus 101 samples, raw traces, and per-cell decisions; replace a machine only with a new reference ID and a rerun.
- Risk: Browser scheduling and service-worker startup produce timing noise. Mitigation: define timestamp boundaries, record scheduling separately, run on physical machines, repeat failures, and do not loosen NFR thresholds.
- Risk: Native host registration differs by OS or browser. Mitigation: use IP-18 installers/manifests and capture registration paths, origin allowlists, checksums, and repair output in each cell; fail closed on mismatch.
- Risk: Accessibility tooling or a supported screen reader is unavailable on a reference machine. Mitigation: mark only that cell's accessibility gate BLOCKED with an exact reason and owner; do not infer accessibility from another OS/browser cell.
- Risk: A test runner could accidentally read real profile data or emit sensitive evidence. Mitigation: disposable profiles, synthetic fixtures, allowlisted APIs, runtime network instrumentation, redaction assertions, and a hard failure when real data is detected.
- Risk: A passing performance fixture could bypass the real projection/index/UI boundary. Mitigation: seed the actual profile-scoped projection and observe the rendered result revision and activation acknowledgement; include projection and result checksums in the report.
- Risk: A host crash or storage error can leave partial records. Mitigation: verify atomic snapshot/revision behavior, duplicate-ID count, health transitions, bounded retry, and lexical availability against the failure fixture in every cell.
- Open decision: exact release versions and physical device IDs must be frozen by the release owner before execution. This does not change the browser/OS matrix or any NFR threshold; absence is a recorded block.
- Open decision: if a platform cannot supply one required accessibility technology, the release owner must accept a documented exception or provide a replacement reference machine before IP-20. “Not tested” is never an implicit acceptance.

## 10. References ngoài `docs/`

- Skill: [testing](../../../.agent/skills/common/engineering/testing/SKILL.md)
- Skill: [change-review](../../../.agent/skills/common/delivery/change-review/SKILL.md)
- Skill: [release](../../../.agent/skills/common/delivery/release/SKILL.md)
- Skill: [lightweight-web](../../../.agent/skills/personal/engineering/frontend/lightweight-web/SKILL.md)
- Skill: [documentation](../../../.agent/skills/common/engineering/documentation/SKILL.md)
- Skill: [task-planning](../../../.agent/skills/common/foundation/task-planning/SKILL.md)
- Skill: [git-workflow](../../../.agent/skills/common/engineering/git-workflow/SKILL.md)
- Project context: [CONTEXT.md](../../../CONTEXT.md)
- Source/config/test paths outside docs (to-create): `extension/`, `host/`, `packaging/`, `fixtures/compatibility/phase-19/`, `tests/compatibility/phase-19/`, `tests/performance/phase-19/`, `tests/accessibility/phase-19/`, `tests/privacy/phase-19/`, `tests/recovery/phase-19/`, `artifacts/verification/phase-19/`.
- Fixture/tool/artifact references outside docs (to-create): `fixtures/compatibility/phase-19/fixtures.json`, `fixtures/compatibility/phase-19/reference-hardware.json`, `tests/compatibility/phase-19/run-cell`, `tests/performance/phase-19/measure-hot-path`, `tests/accessibility/phase-19/run`, `tests/privacy/phase-19/audit`, `tests/recovery/phase-19/run`, and `artifacts/verification/phase-19/matrix-report.json`.
