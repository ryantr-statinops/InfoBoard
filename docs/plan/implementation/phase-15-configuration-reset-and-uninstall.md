# Phase 15 — Configuration, reset, and uninstall

> Plan ID: IP-15
> Status: See README.md execution tracker
> Execution owner: extension settings and host lifecycle owner
> Dependencies: IP-07, IP-08, IP-13, IP-14
> Parallel boundary: IP-16 may proceed in parallel using the lifecycle status contract; IP-18 consumes this phase's owned-path and uninstall contract.
> Requirement IDs: FR-015 (primary); FR-014, NFR-009 (supporting); operational idempotent reset/uninstall lifecycle
> Owned paths: `extension/src/settings/` (to-create), `host/lifecycle/` (to-create), `packaging/lifecycle/` (to-create), `fixtures/lifecycle/phase-15/` (to-create), `tests/lifecycle/phase-15/` (to-create)

## 1. Mục tiêu

- Define one profile-scoped configuration contract for result limit, display density, theme, context-label visibility, and the recency signal option. The contract must have typed bounds, safe defaults, and no implicit global profile.
- Define the extension-to-Go-host lifecycle for reading, validating, updating, resetting, and uninstalling InfoBoard-owned local state. SQLite remains the bounded persistence implementation owned by IP-08; this phase owns the lifecycle orchestration and ownership checks around it.
- Make persistence failure visible and recoverable: the extension and Go host continue lexical search over the current in-memory tab projection with safe defaults, while a settings write or cleanup operation never claims durable success when SQLite or an owned path is unavailable.
- Make reset and uninstall repeatable and narrowly scoped. They remove only InfoBoard-owned configuration, installation state, activation metadata, diagnostics, caches, manifests, and binaries as applicable; they never close, move, or modify browser tabs and never alter unrelated browser data, including history.

## 2. Phạm vi

- Bao gồm:
  - **Profile-scoped settings contract.** Store a row keyed by the validated current `profile_id`, with these keys and initial defaults: `result_limit` (integer, default 10, accepted range 1–50), `density` (enum `comfortable` by default, `compact` or `comfortable`), `theme` (enum `system` by default, `light` or `dark`), `context_labels` (boolean, default true), and `recency_enabled` (boolean, default true). Invalid, missing, or future values are rejected or replaced with the field default without widening the range.
  - **Ownership and isolation.** The extension supplies the active profile identity; the host rejects a request whose envelope `profile_id` does not match the session. Reads, writes, reset, activation-recency records, diagnostics, and effective settings are all scoped by that identity. There is no fallback global settings row.
  - **Extension boundary.** `extension/src/settings/` owns settings controls, validation before sending a request, effective/default rendering, and a no-browser-mutation reset path. The extension continues to own browser APIs and activation, but lifecycle actions MUST NOT call tab-close, tab-update, navigation, or history APIs.
  - **Go host boundary.** `host/lifecycle/` owns lifecycle commands, operation state, cancellation, bounded timeouts, allowlisted owned paths, and rebuilding the in-memory lexical index from the current projection after reset. It must preserve search over the current projection when optional SQLite persistence is degraded.
  - **SQLite boundary.** IP-08 owns tables, migrations, transaction primitives, corruption quarantine, and the `PERSISTENCE_DEGRADED` status. This phase consumes those interfaces and specifies which profile rows a lifecycle operation may delete; it must not create a second storage implementation.
  - **Reset.** Reset the active profile's configuration, activation metadata, and profile-scoped diagnostics in one transaction where possible; clear the host's in-memory settings/index state and request a fresh browser snapshot so the live projection is authoritative again. Recreate only safe defaults and required schema/version state. A reset never deletes a browser tab or any unrelated local path.
  - **Uninstall.** Use a fixed, reviewed manifest of InfoBoard-owned database/cache paths, registered Native Messaging manifests, host binaries, and extension artifacts. Stop the host session, remove only those exact paths, and verify absence. Do not recursively remove an untrusted parent, follow symlinks outside the allowlist, or touch browser profile databases.
  - **Failure and idempotency.** Missing rows and already-absent owned paths count as already complete. Permission errors, a locked database, a failed transaction, or an unavailable host produce a retryable, user-safe status and never a false success. Repeating a successful or partially completed operation converges to the same owned-state result.
  - **Privacy.** Configuration and lifecycle diagnostics contain no page content, query text, raw title, or raw URL. Redacted operation counts, versions, durations, and error classes are sufficient for repair and audit.
- Ngoài phạm vi:
  - Browser-tab observation, eligibility, projection ordering, and activation validation remain with IP-04, IP-05, and IP-14. Reset may request a new snapshot but does not redefine their contracts.
  - SQLite schema/migration implementation remains with IP-08; search-surface layout and accessibility remain with IP-13; Native Messaging envelope and typed error literals remain with IP-07.
  - No network transport, page evaluation, external service, browser-history database, or data source outside the current eligible open-tab projection.
  - No feature that closes tabs, changes tab contents, changes browser history, or cleans paths not explicitly owned by InfoBoard.

## 3. Điều kiện tiên quyết

- IP-07 has frozen request/response envelopes, profile mismatch handling, bounded payloads, and the `PERSISTENCE_DEGRADED`/shutdown status behavior.
- IP-08 has supplied the versioned SQLite repository API for configuration, installation state, activation metadata, and diagnostics, including transaction, reset, corruption, and degraded-persistence behavior.
- IP-13 has a settings entry point that can display effective values, a persistence-degraded state, reset confirmation, and uninstall guidance without claiming an unacknowledged operation.
- IP-14 has defined profile-bound activation metadata and the stale-result guard; reset/uninstall must invalidate or remove that metadata without activating anything.
- The implementation starts with the logical roots `extension/`, `host/`, `packaging/`, `fixtures/`, and `tests/` because no source path is assumed to exist yet. Exact module names are to-create targets until implementation begins.

## 4. Đầu ra cần bàn giao

- A profile-scoped settings contract and validator under `extension/src/settings/` (to-create), including the five keys, defaults, bounds/enums, serialization, and effective-versus-persisted status.
- A Go lifecycle coordinator under `host/lifecycle/` (to-create) that implements load/save, reset, and uninstall orchestration against the IP-08 repository and IP-07 protocol, with cancellation and bounded operation results.
- An allowlisted ownership manifest and platform-aware cleanup adapter under `packaging/lifecycle/` (to-create). It must describe only InfoBoard artifacts and support missing-path success, explicit permission failure, and post-operation absence checks.
- Deterministic fixtures under `fixtures/lifecycle/phase-15/` (to-create) and focused tests under `tests/lifecycle/phase-15/` (to-create) covering profiles, defaults, persistence failure, reset, uninstall, idempotency, path safety, and browser non-mutation.
- A lifecycle status/result contract for UI and diagnostics: operation, profile scope, completed steps, retryable failure class, persistence state, and whether the owned-state postcondition was verified. It must not include raw titles, URLs, query text, or arbitrary filesystem paths supplied by a caller.

## 5. Skill và tài liệu áp dụng

- Skill tags:
  - `common/engineering/documentation` — document the ownership boundary, defaults, lifecycle semantics, and observable recovery behavior.
  - `common/foundation/task-planning` — sequence settings, repository calls, memory rebuild, cleanup, and verification as independently checkable slices.
  - `common/engineering/testing` — cover profile isolation, invalid/default values, idempotency, and non-mutation invariants rather than implementation details.
  - `common/engineering/git-workflow` — keep the phase commit limited to this file and provide reproducible future implementation commits.
  - `personal/engineering/data/databases` — model profile-keyed configuration, transactional deletion, retention, and persistence degradation on SQLite.
  - `common/security/secure-development` — enforce profile trust boundaries, input/path allowlists, redacted diagnostics, symlink safety, and fail-closed cleanup.
  - `common/delivery/release` — specify install/upgrade/uninstall artifact ownership, repair, rollback, and repeatable lifecycle evidence.
- Tài liệu trong `docs/`:
  - [Product requirements](../refactor/requirements.md) — FR-014, FR-015, NFR-009, and idempotent reset/uninstall rules.
  - [Persistence and lifecycle](../refactor/persistence-and-lifecycle.md) — SQLite boundary, data retention, defaults on failure, and reset/uninstall ownership.
  - [Domain, data, privacy, and security](../refactor/domain-and-privacy.md) — profile isolation, activation records, and local-data boundary.
  - [User experience contract](../refactor/user-experience.md) — settings controls, runtime status, reset, and uninstall guidance.
  - [Packaging and operations](../refactor/packaging-and-operations.md) — host registration, exact artifacts, and uninstall behavior.
  - [Runtime protocol](../refactor/runtime-protocol.md) — profile-bound requests, bounded errors, and lifecycle status exchange.
  - [Verification and acceptance](../refactor/verification-and-acceptance.md) — reset/uninstall ownership, privacy, and recovery evidence.
- Quy ước code, ADR, context ngoài `docs/`: use `CONTEXT.md` for repository conventions; IP-08 is the storage contract owner, IP-07 is the protocol contract owner, and IP-14 is the activation-metadata contract owner. Keep a single allowlist for cleanup paths and never derive deletion targets from request input.

## 6. Công việc triển khai

- [ ] `IP-15-T01` Việc 1: Define `ConfigKey` and `ProfileConfig` in `extension/src/settings/` (to-create) and mirror the wire shape in `host/lifecycle/` (to-create): `profile_id`, `result_limit`, `density`, `theme`, `context_labels`, `recency_enabled`, protocol/ranking versions, and effective persistence status. Reject empty/mismatched profile IDs, unknown keys, out-of-range limits, invalid enums, and malformed booleans before a write.
- [ ] `IP-15-T02` Việc 2: Implement a profile-keyed load path that returns persisted values only for the current profile and fills missing fields with the safe defaults (limit 10, comfortable, system, labels enabled, recency enabled). Return an explicit degraded status when SQLite cannot be opened/read; keep the effective defaults in memory and keep lexical queries available over the current projection.
- [ ] `IP-15-T03` Việc 3: Implement validated updates with a transactional repository call. On a write failure, retain the last known/effective value for the session, surface a retryable persistence status, and do not report the setting as durable. Verify one profile's write cannot affect another profile's read, reset, or activation recency.
- [ ] `IP-15-T04` Việc 4: Implement reset orchestration in `host/lifecycle/`: authenticate the current `profile_id`, stop new writes for that profile, delete only its configuration/activation/diagnostic rows through the IP-08 transaction API, clear its in-memory settings and lexical index, request a full browser snapshot, and publish defaults plus the resulting projection revision. Handle cancellation and transaction failure without deleting another profile's rows.
- [ ] `IP-15-T05` Việc 5: Implement uninstall orchestration in `packaging/lifecycle/` and `host/lifecycle/`: close the Native Messaging session, enumerate only the checked-in ownership manifest, remove files/directories without following links outside the manifest, and verify the owned postcondition. Treat absent artifacts as success; preserve browser tabs, browser history, unrelated files, and other profiles.
- [ ] `IP-15-T06` Việc 6: Define repair behavior for locked/corrupt/unavailable SQLite, missing host registration, interrupted reset, and interrupted uninstall. Retry only allowlisted operations, preserve a redacted diagnostic event, and expose a status that distinguishes complete, degraded, and retryable/incomplete outcomes.
- [ ] `IP-15-T07` Việc 7: Add fixture IDs and assertions in `fixtures/lifecycle/phase-15/` (to-create): `CFG-DEFAULTS-001` (new profile gets exact defaults); `CFG-PROFILE-ISOLATION-001` (profiles A/B retain distinct values); `CFG-INVALID-001` (invalid values are rejected/defaulted); `CFG-RECENCY-OPTION-001` (recency can be disabled without changing tab identity); `CFG-PERSIST-DEGRADED-001` (SQLite failure keeps lexical search and marks degraded); `RESET-OWNED-001` (only active-profile-owned records disappear); `RESET-NON-MUTATION-001` (tabs/history/external sentinel remain byte-for-byte unchanged); `RESET-IDEMPOTENT-001` (second reset has the same postcondition); `UNINSTALL-OWNED-001` (manifested artifacts are absent); `UNINSTALL-IDEMPOTENT-001` (repeat uninstall succeeds); `UNINSTALL-PARTIAL-001` (missing/locked artifact yields precise retryable status); and `UNINSTALL-PATH-SAFETY-001` (untrusted path input cannot escape the allowlist).
- [ ] `IP-15-T08` Việc 8: Add focused tests that assert observable contracts: profile A/B config and activation metadata never cross; defaults render after reset; SQLite failure still returns lexical results; reset/uninstall issue zero browser mutation calls; repeated operations converge; and the post-cleanup path set equals the owned manifest minus unrelated sentinels.

## 7. Kế hoạch commit

1. `feat(lifecycle): implement ip-15-t01`
   - Task IDs: `IP-15-T01`.
   - Owned target paths: extension/src/settings/, host/lifecycle/.
   - Behavior: Việc 1: Define `ConfigKey` and `ProfileConfig` in `extension/src/settings/` (to-create) and mirror the wire shape in `host/lifecycle/` (to-create): `profile_id`, `result_limit`, `density`, `theme`, `context_labels`, `recency_enabled`, protocol/ranking versions, and effective persistence status. Reject empty/mismatched profile IDs, unknown keys, out-of-range limits, invalid enums, and malformed booleans before a write.
   - Fixture and command: the observable fixture/outcome stated by this task; run `INFOBOARD_FIXTURES=fixtures/lifecycle/phase-15 go test ./host/lifecycle/... -run 'Test(Config|Profile|Reset|Uninstall)' -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Việc 1: Define `ConfigKey` and `ProfileConfig` in `extension/src/settings/` (to-create) and mirror the wire shape in `host/lifecycle/` (to-create): `profile_id`, `result_limit`, `density`, `theme`, `context_labels`, `recency_enabled`, protocol/ranking versions, and effective persistence status. Reject empty/mismatched profile IDs, unknown keys, out-of-range limits, invalid enums, and malformed booleans before a write.
   - Dependency gate: all index.md dependencies for IP-15 have merged to dev; phase work branch starts from latest origin/dev.

2. `feat(lifecycle): implement ip-15-t02`
   - Task IDs: `IP-15-T02`.
   - Owned target paths: `extension/src/settings/` (to-create), `host/lifecycle/` (to-create), `packaging/lifecycle/` (to-create), `fixtures/lifecycle/phase-15/` (to-create), `tests/lifecycle/phase-15/` (to-create).
   - Behavior: Việc 2: Implement a profile-keyed load path that returns persisted values only for the current profile and fills missing fields with the safe defaults (limit 10, comfortable, system, labels enabled, recency enabled). Return an explicit degraded status when SQLite cannot be opened/read; keep the effective defaults in memory and keep lexical queries available over the current projection.
   - Fixture and command: the observable fixture/outcome stated by this task; run `INFOBOARD_FIXTURES=fixtures/lifecycle/phase-15 go test ./host/lifecycle/... -run 'Test(Config|Profile|Reset|Uninstall)' -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Việc 2: Implement a profile-keyed load path that returns persisted values only for the current profile and fills missing fields with the safe defaults (limit 10, comfortable, system, labels enabled, recency enabled). Return an explicit degraded status when SQLite cannot be opened/read; keep the effective defaults in memory and keep lexical queries available over the current projection.
   - Dependency gate: all index.md dependencies for IP-15 have merged to dev; phase work branch starts from latest origin/dev.

3. `feat(lifecycle): implement ip-15-t03`
   - Task IDs: `IP-15-T03`.
   - Owned target paths: `extension/src/settings/` (to-create), `host/lifecycle/` (to-create), `packaging/lifecycle/` (to-create), `fixtures/lifecycle/phase-15/` (to-create), `tests/lifecycle/phase-15/` (to-create).
   - Behavior: Việc 3: Implement validated updates with a transactional repository call. On a write failure, retain the last known/effective value for the session, surface a retryable persistence status, and do not report the setting as durable. Verify one profile's write cannot affect another profile's read, reset, or activation recency.
   - Fixture and command: the observable fixture/outcome stated by this task; run `INFOBOARD_FIXTURES=fixtures/lifecycle/phase-15 go test ./host/lifecycle/... -run 'Test(Config|Profile|Reset|Uninstall)' -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Việc 3: Implement validated updates with a transactional repository call. On a write failure, retain the last known/effective value for the session, surface a retryable persistence status, and do not report the setting as durable. Verify one profile's write cannot affect another profile's read, reset, or activation recency.
   - Dependency gate: all index.md dependencies for IP-15 have merged to dev; phase work branch starts from latest origin/dev.

4. `feat(lifecycle): implement ip-15-t04`
   - Task IDs: `IP-15-T04`.
   - Owned target paths: host/lifecycle/.
   - Behavior: Việc 4: Implement reset orchestration in `host/lifecycle/`: authenticate the current `profile_id`, stop new writes for that profile, delete only its configuration/activation/diagnostic rows through the IP-08 transaction API, clear its in-memory settings and lexical index, request a full browser snapshot, and publish defaults plus the resulting projection revision. Handle cancellation and transaction failure without deleting another profile's rows.
   - Fixture and command: IP-08; run `INFOBOARD_FIXTURES=fixtures/lifecycle/phase-15 go test ./host/lifecycle/... -run 'Test(Config|Profile|Reset|Uninstall)' -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Việc 4: Implement reset orchestration in `host/lifecycle/`: authenticate the current `profile_id`, stop new writes for that profile, delete only its configuration/activation/diagnostic rows through the IP-08 transaction API, clear its in-memory settings and lexical index, request a full browser snapshot, and publish defaults plus the resulting projection revision. Handle cancellation and transaction failure without deleting another profile's rows.
   - Dependency gate: all index.md dependencies for IP-15 have merged to dev; phase work branch starts from latest origin/dev.

5. `feat(lifecycle): implement ip-15-t05`
   - Task IDs: `IP-15-T05`.
   - Owned target paths: packaging/lifecycle/, host/lifecycle/.
   - Behavior: Việc 5: Implement uninstall orchestration in `packaging/lifecycle/` and `host/lifecycle/`: close the Native Messaging session, enumerate only the checked-in ownership manifest, remove files/directories without following links outside the manifest, and verify the owned postcondition. Treat absent artifacts as success; preserve browser tabs, browser history, unrelated files, and other profiles.
   - Fixture and command: the observable fixture/outcome stated by this task; run `INFOBOARD_FIXTURES=fixtures/lifecycle/phase-15 go test ./host/lifecycle/... -run 'Test(Config|Profile|Reset|Uninstall)' -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Việc 5: Implement uninstall orchestration in `packaging/lifecycle/` and `host/lifecycle/`: close the Native Messaging session, enumerate only the checked-in ownership manifest, remove files/directories without following links outside the manifest, and verify the owned postcondition. Treat absent artifacts as success; preserve browser tabs, browser history, unrelated files, and other profiles.
   - Dependency gate: all index.md dependencies for IP-15 have merged to dev; phase work branch starts from latest origin/dev.

6. `feat(lifecycle): implement ip-15-t06`
   - Task IDs: `IP-15-T06`.
   - Owned target paths: `extension/src/settings/` (to-create), `host/lifecycle/` (to-create), `packaging/lifecycle/` (to-create), `fixtures/lifecycle/phase-15/` (to-create), `tests/lifecycle/phase-15/` (to-create).
   - Behavior: Việc 6: Define repair behavior for locked/corrupt/unavailable SQLite, missing host registration, interrupted reset, and interrupted uninstall. Retry only allowlisted operations, preserve a redacted diagnostic event, and expose a status that distinguishes complete, degraded, and retryable/incomplete outcomes.
   - Fixture and command: the observable fixture/outcome stated by this task; run `INFOBOARD_FIXTURES=fixtures/lifecycle/phase-15 go test ./host/lifecycle/... -run 'Test(Config|Profile|Reset|Uninstall)' -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Việc 6: Define repair behavior for locked/corrupt/unavailable SQLite, missing host registration, interrupted reset, and interrupted uninstall. Retry only allowlisted operations, preserve a redacted diagnostic event, and expose a status that distinguishes complete, degraded, and retryable/incomplete outcomes.
   - Dependency gate: all index.md dependencies for IP-15 have merged to dev; phase work branch starts from latest origin/dev.

7. `test(lifecycle): implement ip-15-t07`
   - Task IDs: `IP-15-T07`.
   - Owned target paths: fixtures/lifecycle/phase-15/.
   - Behavior: Việc 7: Add fixture IDs and assertions in `fixtures/lifecycle/phase-15/` (to-create): `CFG-DEFAULTS-001` (new profile gets exact defaults); `CFG-PROFILE-ISOLATION-001` (profiles A/B retain distinct values); `CFG-INVALID-001` (invalid values are rejected/defaulted); `CFG-RECENCY-OPTION-001` (recency can be disabled without changing tab identity); `CFG-PERSIST-DEGRADED-001` (SQLite failure keeps lexical search and marks degraded); `RESET-OWNED-001` (only active-profile-owned records disappear); `RESET-NON-MUTATION-001` (tabs/history/external sentinel remain byte-for-byte unchanged); `RESET-IDEMPOTENT-001` (second reset has the same postcondition); `UNINSTALL-OWNED-001` (manifested artifacts are absent); `UNINSTALL-IDEMPOTENT-001` (repeat uninstall succeeds); `UNINSTALL-PARTIAL-001` (missing/locked artifact yields precise retryable status); and `UNINSTALL-PATH-SAFETY-001` (untrusted path input cannot escape the allowlist).
   - Fixture and command: CFG-DEFAULTS-001, CFG-PROFILE-ISOLATION-001, CFG-INVALID-001, CFG-RECENCY-OPTION-001, CFG-PERSIST-DEGRADED-001, RESET-OWNED-001, RESET-NON-MUTATION-001, RESET-IDEMPOTENT-001, STALL-OWNED-001, STALL-IDEMPOTENT-001, STALL-PARTIAL-001, STALL-PATH-SAFETY-001; run `INFOBOARD_FIXTURES=fixtures/lifecycle/phase-15 go test ./host/lifecycle/... -run 'Test(Config|Profile|Reset|Uninstall)' -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Việc 7: Add fixture IDs and assertions in `fixtures/lifecycle/phase-15/` (to-create): `CFG-DEFAULTS-001` (new profile gets exact defaults); `CFG-PROFILE-ISOLATION-001` (profiles A/B retain distinct values); `CFG-INVALID-001` (invalid values are rejected/defaulted); `CFG-RECENCY-OPTION-001` (recency can be disabled without changing tab identity); `CFG-PERSIST-DEGRADED-001` (SQLite failure keeps lexical search and marks degraded); `RESET-OWNED-001` (only active-profile-owned records disappear); `RESET-NON-MUTATION-001` (tabs/history/external sentinel remain byte-for-byte unchanged); `RESET-IDEMPOTENT-001` (second reset has the same postcondition); `UNINSTALL-OWNED-001` (manifested artifacts are absent); `UNINSTALL-IDEMPOTENT-001` (repeat uninstall succeeds); `UNINSTALL-PARTIAL-001` (missing/locked artifact yields precise retryable status); and `UNINSTALL-PATH-SAFETY-001` (untrusted path input cannot escape the allowlist).
   - Dependency gate: all index.md dependencies for IP-15 have merged to dev; phase work branch starts from latest origin/dev.

8. `test(lifecycle): implement ip-15-t08`
   - Task IDs: `IP-15-T08`.
   - Owned target paths: `extension/src/settings/` (to-create), `host/lifecycle/` (to-create), `packaging/lifecycle/` (to-create), `fixtures/lifecycle/phase-15/` (to-create), `tests/lifecycle/phase-15/` (to-create).
   - Behavior: Việc 8: Add focused tests that assert observable contracts: profile A/B config and activation metadata never cross; defaults render after reset; SQLite failure still returns lexical results; reset/uninstall issue zero browser mutation calls; repeated operations converge; and the post-cleanup path set equals the owned manifest minus unrelated sentinels.
   - Fixture and command: the observable fixture/outcome stated by this task; run `INFOBOARD_FIXTURES=fixtures/lifecycle/phase-15 go test ./host/lifecycle/... -run 'Test(Config|Profile|Reset|Uninstall)' -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Việc 8: Add focused tests that assert observable contracts: profile A/B config and activation metadata never cross; defaults render after reset; SQLite failure still returns lexical results; reset/uninstall issue zero browser mutation calls; repeated operations converge; and the post-cleanup path set equals the owned manifest minus unrelated sentinels.
   - Dependency gate: all index.md dependencies for IP-15 have merged to dev; phase work branch starts from latest origin/dev.

## 8. Kiểm chứng và nghiệm thu

- [ ] Exact unit command from repository root after the to-create modules exist: `INFOBOARD_FIXTURES=fixtures/lifecycle/phase-15 go test ./host/lifecycle/... -run 'Test(Config|Profile|Reset|Uninstall)' -count=1`. The fixture directory is read-only input; no network or live browser profile is allowed.
- [ ] Exact integration command from `tests/`: `INFOBOARD_FIXTURES=../fixtures/lifecycle/phase-15 go test ./lifecycle/phase-15 -count=1 -v`; the runner must seed profiles A and B, distinct values, two eligible tabs, one unrelated browser-data sentinel, and one external filesystem sentinel.
- [ ] Manual/E2E input → observable output: set profile A to compact/dark/limit 25/labels off/recency off, set profile B to comfortable/system/limit 10/labels on/recency on, reload each profile, and observe the exact independent values and result ordering. Reset A; A returns to defaults, A-owned activation/config rows are absent, B remains unchanged, the projection is rebuilt, and no tab/history/browser-data sentinel changes. Run reset a second time and observe the same result. Uninstall; every allowlisted InfoBoard artifact is absent, no tab is closed, history and unrelated sentinels are unchanged, and a second uninstall reports complete without error.
- [ ] Persistence-failure signal: make the fixture database unreadable/locked, open settings, and observe exact safe defaults plus `PERSISTENCE_DEGRADED`; lexical query still returns the current projection, and a failed update/reset is marked retryable rather than durable success.
- [ ] Security and privacy signal: send a mismatched profile ID, invalid key/value, path traversal, symlink, and arbitrary path fixture; observe rejection with no cross-profile read and no deletion outside the ownership manifest. Diagnostics contain operation/status/error class only, never raw titles, URLs, queries, tokens, or untrusted paths.
- [ ] Acceptance is met when FR-015 removal is limited to InfoBoard-owned local data, FR-014 profile settings never cross profiles, NFR-009 lifecycle operations are repeatable, and external tabs/history remain unchanged across reset and uninstall.

## 9. Rủi ro và quyết định còn mở

- Rủi ro: a host crash or SQLite lock can interrupt reset after memory has been cleared but before the transaction commits. Phương án xử lý đã chọn: use one repository transaction, retain a retryable status, rebuild from browser authority, and make the next reset reconcile by owned profile key rather than assuming a first attempt completed.
- Rủi ro: uninstall may run when the extension or host is already absent. Phương án xử lý đã chọn: treat absent allowlisted artifacts as complete, verify every remaining owned path, and never broaden deletion to a parent directory.
- Rủi ro: a profile identity supplied by an untrusted caller could target another profile. Phương án xử lý đã chọn: derive the active identity at the extension boundary, bind it to the Native Messaging session, reject envelope mismatch, and test A/B cross-reads and cross-deletes.
- Rủi ro: a symlink or junction could redirect cleanup outside the product directory. Phương án xử lý đã chọn: compare canonical paths to a checked-in allowlist, refuse links/targets outside it, and report a repairable failure instead of deleting.
- Rủi ro: settings UI and the host may disagree while persistence is degraded. Phương án xử lý đã chọn: display effective in-memory values with degraded status, mark updates non-durable, and require a successful read-back before clearing the status.
- Câu hỏi còn mở chỉ khi câu trả lời có thể thay đổi contract: the packaging owner must confirm the platform-specific native-manifest and binary locations before IP-18 freezes the final allowlist; until then each target is a named to-create manifest entry, not an inferred wildcard.

## 10. References ngoài `docs/`

- Skill: [documentation](../../../.agent/skills/common/engineering/documentation/SKILL.md)
- Skill: [task-planning](../../../.agent/skills/common/foundation/task-planning/SKILL.md)
- Skill: [testing](../../../.agent/skills/common/engineering/testing/SKILL.md)
- Skill: [git-workflow](../../../.agent/skills/common/engineering/git-workflow/SKILL.md)
- Skill: [databases](../../../.agent/skills/personal/engineering/data/databases/SKILL.md)
- Skill: [secure-development](../../../.agent/skills/common/security/secure-development/SKILL.md)
- Skill: [release](../../../.agent/skills/common/delivery/release/SKILL.md)
- Project context: [CONTEXT.md](../../../CONTEXT.md)
- Source/config/test path ngoài `docs/`: `extension/src/settings/`, `host/lifecycle/`, `packaging/lifecycle/`, and `tests/lifecycle/phase-15/` (all to-create; no implementation source is assumed).
- Fixture/tool/artifact ngoài `docs/`: `fixtures/lifecycle/phase-15/` (to-create), including SQLite seed state, profile A/B sentinels, browser API mutation spy, owned-path manifest, locked-database simulation, and post-cleanup path report.
