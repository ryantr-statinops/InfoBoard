# Phase 17 — Privacy, permissions, and profile isolation

> Plan ID: IP-17
> Status: See README.md execution tracker
> Execution owner: Privacy and security implementation owner
> Dependencies: IP-03, IP-04, IP-05, IP-07, IP-08, IP-16
> Parallel boundary: IP-18 may consume this policy after IP-17 is merged; IP-19 consumes the evidence; no shared owned paths
> Requirement IDs: FR-014, NFR-007, NFR-009 (primary); supporting FR-003, FR-015, and operational permission-minimization and retention requirements
> Owned paths: `extension/src/privacy/` (to-create), `host/privacy/` (to-create), `fixtures/privacy/phase-17/` (to-create), `tests/privacy/phase-17/` (to-create)

## 1. Mục tiêu

- Establish one enforceable privacy boundary for Chrome and Edge desktop on Linux, macOS, and Windows: only the current browser profile's eligible open-tab metadata is admitted to the local projection and search path.
- Define a permission matrix that maps every requested browser capability to a named requirement, owning component, denial behavior, and static/runtime evidence. The manifest MUST not request capabilities for data sources outside the product contract.
- Make profile and private-context separation explicit across the extension, Native Messaging session, in-memory projection/index, SQLite rows, diagnostics, reset, uninstall, and activation. A result, setting, or event from one scope MUST never be queryable or activatable from another scope.
- Define sensitive-field handling, deny-by-default redaction, secret policy, bounded input validation, and retention audit so privacy evidence is reproducible rather than inferred from implementation intent.
- Deliver concrete abuse-case fixtures proving that excluded data sources are not requested, collected, persisted, or transmitted; that permission denial fails closed; and that diagnostics do not leak raw metadata.

## 2. Phạm vi

- Bao gồm:
  - A data inventory for the allowed current open-tab projection: opaque `profile_id`, tab/window/group operational IDs, title, full URL and derived domain, window/group labels, pinned/active/eligible state, observation time, and projection revision. Full URL, title, labels, query input, and browser IDs are sensitive even when they are required transiently for local search or activation.
  - A durable-data inventory limited to profile-scoped configuration, installation state, bounded activation metadata, and redacted diagnostics. Activation metadata is at most 500 records and 30 days; diagnostics are at most 7 days or 10 MB, whichever is reached first. Live projection and query input remain memory-only and rebuildable.
  - A permission matrix covering command registration, tab/window/group metadata, tab activation and window focus, private-context capability, Native Messaging, and explicitly denied browser capabilities. The matrix is normative for the Manifest V3 package and the Chrome/Edge release review.
  - A trust-boundary contract for browser APIs to extension, extension to Native Messaging host, host to in-memory index, host to SQLite, diagnostics to UI/export, and installer/manifest to host registration. Every boundary validates identity, context, type, size, and authorization before use.
  - A scope key of `(browser_family, profile_id, context_kind)`, where `context_kind` is `normal` or `private`. The extension and host bind every session, snapshot, delta, query, result, activation acknowledgement, setting, diagnostic row, and reset operation to that key. A private session MAY use an additional ephemeral session nonce, but it MUST NOT be persisted or treated as a browser account identifier.
  - Normal/private separation. Private access is used only when the browser grants it; otherwise the extension reports an unavailable-context state and does not broaden permissions. Private projection records and query input are ephemeral, never written to SQLite, never copied into normal state, and are cleared when the private context/session ends.
  - A redaction policy shared by health responses, typed errors, logs, SQLite diagnostics, and local exports. The default output allowlist is enum states, counts, durations, sizes, versions, revisions, safe status/error classes, retryability, and bounded retention counters. Unknown fields are dropped rather than recursively serialized.
  - Secret handling and bounded input controls for message envelopes, profile/context identifiers, query input, tab fields, result limits, tab counts, log events, and file/path values. Input is validated before indexing, SQL construction, or diagnostic allocation; browser data is never interpreted as code, a command, a path, a URL to fetch, or SQL.
  - Static permission inspection and runtime spies/fixtures that prove there is no network request, page evaluation/read, browser-history access, cookie/session access, local-storage access, broad host access, or external telemetry in the normal path. The Go host uses only the registered framed Native Messaging transport.
  - Privacy retention audits through the IP-08 repository seam: profile-scoped age/byte eviction, private-scope non-persistence, reset/uninstall owned-data removal, and proof that failed cleanup is visible rather than silently broadened.
- Ngoài phạm vi:
  - Browser event conversion, eligibility rules, projection ordering, and full-snapshot authority remain IP-04/IP-05. This phase consumes their profile and context fields and verifies the privacy policy at the boundary.
  - Manifest V3 command/surface scaffolding remains IP-03; Native Messaging frame/envelope definitions and numeric transport limits remain IP-07; SQLite tables, migrations, quarantine, and deletion primitives remain IP-08; health-state presentation and diagnostic state transitions remain IP-16.
  - Ranking, query orchestration, result layout, and activation browser calls remain their owning phases. This phase only requires their scope and redaction contracts.
  - Whole-browser history, saved-item/download data, page DOM or rendered text, selected text, cookies, local storage, session data, network interception, cloud services, account sync, cross-profile search, unsupported browsers, mobile, and any remote or loopback transport.
  - A raw-metadata debug mode. If a future decision introduces a short-lived local debug capture, it requires a separate decision record, explicit consent, an allowlist, an automatic expiry, and updated privacy evidence; the default contract here remains redacted.

## 3. Điều kiện tiên quyết

- IP-03 has defined the Manifest V3 command, extension-owned focused surface, browser adapter seam, and minimal permission starting point. No privacy work may add a permission merely to simplify implementation.
- IP-04 has defined event eligibility, permission-denial behavior, and private-context event handling. Missing or denied fields must remain absent rather than being filled from another profile or data source.
- IP-05 has defined profile-scoped projection identity, authoritative snapshots, revisions, and atomic reconciliation. A privacy check must run before a record enters a projection or crosses the host boundary.
- IP-07 has frozen the versioned envelope, handshake, profile mismatch behavior, message types, typed errors, and pre-allocation payload/count limits. This phase uses `PROFILE_MISMATCH` and `PAYLOAD_LIMIT` rather than inventing parallel errors.
- IP-08 has defined profile-keyed SQLite ownership, transactional writes, activation and diagnostic retention, reset, corruption/degraded behavior, and private-scope persistence rules.
- IP-16 has defined the health/diagnostic event seam and redacted observability contract. This phase supplies privacy policy and audit fixtures; it does not create a second logging system.
- Re-read [requirements](../refactor/requirements.md), [browser landscape](../refactor/browser-landscape.md), [domain and privacy](../refactor/domain-and-privacy.md), [architecture](../refactor/architecture.md), [runtime protocol](../refactor/runtime-protocol.md), [persistence and lifecycle](../refactor/persistence-and-lifecycle.md), [packaging and operations](../refactor/packaging-and-operations.md), [verification and acceptance](../refactor/verification-and-acceptance.md), and [decision log](../refactor/decisions.md) before implementation.
- The implementation source tree is currently absent. All paths below are logical `to-create` targets under `extension/`, `host/`, `fixtures/`, and `tests/`; replace a target with an observed path only after rereading the scaffold and preserving the same ownership boundary.

## 4. Đầu ra cần bàn giao

- [ ] A permission-policy module under `extension/src/privacy/` (to-create) containing the Chrome/Edge matrix, manifest allowlist/denylist, capability-denial state, private-access capability check, and a machine-readable mapping from capability to FR/NFR/operational requirement.
- [ ] A scope-policy module under `extension/src/privacy/` (to-create) that derives the current opaque profile/context scope from IP-02/IP-05 contracts, rejects mismatched host responses, prevents normal/private mixing, and clears private state at context end.
- [ ] A host authorization/redaction/limits module under `host/privacy/` (to-create) that validates scope-bound messages, applies IP-07 limits before allocation, rejects secrets and disallowed fields, emits only allowlisted diagnostics, and passes parameterized values to IP-08.
- [ ] A documented field-class policy in code comments or contract fixtures: transient local search fields; bounded durable fields; diagnostic allowlist fields; and forbidden fields. The policy must state retention, owner, permitted boundary, and redaction behavior for each class.
- [ ] Permission and trust-boundary fixtures under `fixtures/privacy/phase-17/` (to-create), including:
  - `FX-PRIVACY-PERMISSION-MATRIX`: manifest and API calls match the approved matrix; denied capabilities are absent.
  - `FX-PRIVACY-PROFILE-CROSS-READ`: profile A cannot query profile B's projection, settings, diagnostics, or result references.
  - `FX-PRIVACY-PROFILE-CROSS-ACTIVATE`: a result from another profile or context is rejected before browser activation.
  - `FX-PRIVACY-PRIVATE-SEPARATION`: normal and private records never enter one another's projection/index/session.
  - `FX-PRIVACY-PRIVATE-TEARDOWN`: ending a private context removes its in-memory records and leaves no SQLite row or diagnostic payload containing its fields.
  - `FX-PRIVACY-PERMISSION-DENIED`: unavailable tab/group/private access yields a bounded, user-safe degraded state without a permission escalation.
  - `FX-PRIVACY-REDACTION`: sentinel title, URL, query string, fragment, label, cookie-like value, token-like value, and tab ID cannot appear in logs, errors, exports, or SQLite diagnostics.
  - `FX-PRIVACY-SECRET-REJECTION`: secret-shaped inputs are rejected or dropped before persistence/export and never echoed in an error.
  - `FX-PRIVACY-INPUT-LIMITS`: overlong, malformed, unknown, and over-count inputs return the IP-07 typed limit/error outcome without unbounded allocation.
  - `FX-PRIVACY-RETENTION-AUDIT`: exact age and aggregate-byte boundaries evict only the owning profile's oldest records, with private rows absent.
  - `FX-PRIVACY-EXCLUDED-SURFACE`: spies show no excluded browser API, network request, page read, or external transmission during command, sync, query, activation, reset, and uninstall journeys.
- [ ] Focused tests under `tests/privacy/phase-17/` (to-create): static manifest/API audit, extension integration, Go scope/redaction/limits tests, SQLite retention audit, private teardown, and cross-profile/cross-context abuse tests.
- [ ] A security-review handoff containing the permission matrix, trust-boundary diagram/table, abuse-case outcomes, redaction evidence, limits, retention evidence, residual risks, and explicit non-goals. No new product data source is approved by this phase.

## 5. Skill và tài liệu áp dụng

- Skill tags:
  - [`secure-development`](../../../.agent/skills/common/security/secure-development/SKILL.md): the phase handles identity, browser permissions, untrusted tab fields, secrets, file/storage boundaries, input limits, outputs, and logging.
  - [`security-review`](../../../.agent/skills/common/security/security-review/SKILL.md): profile confusion, private-context leakage, permission overreach, redaction bypass, and local-process misuse require concrete threat and abuse-case review.
  - [`architecture-review`](../../../.agent/skills/personal/product/architecture-review/SKILL.md): the policy crosses extension, Native Messaging, host, in-memory index, SQLite, diagnostics, and packaging ownership boundaries; each owner and failure path must remain explicit.
  - [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md): isolation, denial, redaction, limit, retention, and excluded-surface claims need deterministic success, boundary, invalid, and failure fixtures.
  - [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md): this standalone plan must make the permission/data contract and its evidence executable for implementers and reviewers.
  - [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md): six upstream contracts must be sequenced into independently verifiable policy, runtime, fixture, and audit work.
  - [`git-workflow`](../../../.agent/skills/common/engineering/git-workflow/SKILL.md): the phase is one focused file on one branch and requires a path-only diff, exact commit, and remote verification.
- Tài liệu trong `docs/`: [product refactor README](../refactor/README.md), [requirements](../refactor/requirements.md), [browser landscape](../refactor/browser-landscape.md), [domain and privacy](../refactor/domain-and-privacy.md), [architecture](../refactor/architecture.md), [runtime protocol](../refactor/runtime-protocol.md), [persistence and lifecycle](../refactor/persistence-and-lifecycle.md), [packaging and operations](../refactor/packaging-and-operations.md), [verification and acceptance](../refactor/verification-and-acceptance.md), and [decision log](../refactor/decisions.md).
- Quy ước code, ADR, context ngoài `docs/`: [`CONTEXT.md`](../../../CONTEXT.md); use opaque profile identity, current-profile vocabulary, bounded local data, typed protocol errors, fail-closed behavior, deterministic results, and explicit ownership.

## 6. Công việc triển khai

- [ ] `IP-17-T01` **Việc 1 — Freeze the field-class and ownership matrix.** In `extension/src/privacy/` and `host/privacy/` (to-create), record for every field its source, owner, allowed boundary, storage class, retention, display policy, and redaction rule:
  - Transient search/projection values: title, full URL, derived domain, window/group labels, tab/window/group IDs, pinned/active state, private marker, query input, and projection revision. They may cross the extension-to-host local connection only after scope and limit checks; they remain memory-only and are never diagnostics or SQLite values.
  - Bounded durable values: validated profile-scoped settings, installation/schema state, and activation metadata containing only the permitted tab identity/domain/timestamp/source fields. Never write query text, URL path/query/fragment, page text, cookies, tokens, or raw browser payloads.
  - Diagnostic values: status/error enums, counts, durations, versions, sizes, revisions, retryability, retention counters, and opaque local correlation IDs. Raw title, URL, domain, labels, query input, browser IDs, private identifiers, and arbitrary error text are denied.
  - Forbidden values: credentials, cookies, session tokens, auth headers, extension/host private keys, environment secrets, page contents, network payloads, and data from excluded browser surfaces. A field that cannot be classified is rejected rather than forwarded.
- [ ] `IP-17-T02` **Việc 2 — Implement the permission matrix and static audit.** Keep the manifest allowlist narrow and map each capability as follows:

  | Capability/API | Product use and owner | Manifest/permission decision | Denial or absence behavior | Evidence |
  | --- | --- | --- | --- | --- |
  | Browser command registration | Opens the focused surface; extension owns it (FR-001). | Command declaration only; no additional broad permission. | Command remains visible or reports an install/configuration error; it does not request a new capability at runtime. | Manifest inspection and shortcut fixture. |
  | Tab metadata and activation | Reads eligible open-tab title/URL/domain/state and activates the validated tab; extension owns it (FR-003, FR-007). | Use the narrow tab capability required by Chrome/Edge for title/URL and tab identity. Do not add page access or URL host patterns. | Missing/denied sensitive fields are absent; search degrades to available fields and activation fails closed for an unavailable target. | Chrome/Edge manifest audit, API spy, event fixture. |
  | Window metadata and focus | Reads window ID/label and focuses the selected window; extension owns it (FR-003, FR-007). | Use the browser window API through the adapter; add no standalone broad permission unless the target browser documents one as required. | Missing labels degrade to no-label context; focus/activation reports a bounded failure and never chooses another profile. | Browser adapter capability test and activation fixture. |
  | Tab-group metadata | Uses group ID/label as optional context; extension owns it (FR-003). | Request only the browser's tab-group capability if the package needs labels; it is optional and not a reason to widen host access. | Group fields are omitted and ranking/display continue without group context. | Permission matrix and denied-group fixture. |
  | Private/incognito context | Separates eligible private records when user/browser access is enabled; extension owns it. | Use split private-context behavior; no permission escalation. The user/browser grant is a capability check, not a bypass. | Report unavailable private context, collect nothing from it, and keep normal scope unchanged. | Private enable/deny/teardown fixtures on Chrome and Edge. |
  | Native Messaging | Sends bounded local protocol messages to the registered Go host; client/host share ownership. | Request only the Native Messaging capability needed for the registered host; exact origin allowlist is reviewed by IP-18. | Missing registration or origin mismatch yields unavailable/repairable state and sends no tab data. | Manifest/host-registration inspection and handshake fixture. |
  | Extension local storage API | Stores only the opaque REF-013 identity key; SQLite owns configuration and activation persistence. | Request `storage` only for `chrome.storage.local` key `profile_id`; prohibit any other key, `storage.sync`, and page storage. | Read/write failure fails closed before host/tab work; invalid stored IDs are not replaced. | Manifest permission check, exact-key storage spy, failure and malformed-value fixtures. |
  | History, saved-item, download, cookie, page, network, and broad-host capabilities | Outside the product boundary. | MUST NOT be requested, declared, or called; no wildcard host patterns, page scripting, interception, or data-source permission. | No data is collected and no permission prompt is shown. A future request requires a new decision record and updated privacy docs. | Manifest denylist, dependency/API scan, runtime spies, review sign-off. |

  - Build the static audit so it fails on an undeclared permission, wildcard/broad host pattern, page access, unapproved API import, or a permission-to-requirement mapping that is missing. Do not treat an API denial as permission to try a different excluded API.
- [ ] `IP-17-T03` **Việc 3 — Bind every message and browser action to one scope.**
  - Obtain IP-02's opaque `profile_id` from the active extension profile; never derive it from a signed-in account, URL, title, or arbitrary message field. Pair it with `context_kind` and an ephemeral private session nonce where available.
  - During `hello`, bind the host session to `(browser_family, profile_id, context_kind)`; require the same binding on `snapshot`, `delta`, `query`, health, reset, activation acknowledgement, and diagnostics requests. Reject absent, malformed, changed, or foreign scope with `PROFILE_MISMATCH` before projection/index/SQLite access.
  - In the extension, verify scope and projection revision on every result before rendering or activation. A result from another profile/context, or a result whose context ended, is stale and cannot be used as a fallback target.
  - Key SQLite reads/writes by the same scope. Normal and private scopes have separate in-memory sessions; private writes are rejected at the repository boundary even if a caller supplies a normal-looking profile ID. Reset and uninstall operate only on the requested owned scope and never inspect or delete another profile's rows.
  - On profile switch, browser restart, service-worker restart, permission revocation, host reconnect, or private-context teardown, discard the old session and request a fresh authoritative snapshot. Do not reuse an old in-memory index or a stale host connection across scopes.
- [ ] `IP-17-T04` **Việc 4 — Enforce private-context lifetime and data flow.**
  - Keep private tab projection/index/query/result data in an isolated memory arena keyed by its private scope; do not merge it with normal projection counts, ranking recency, settings, or diagnostic payloads.
  - Disable activation-recency writes for private records. If the shared IP-14 path reports an activation, record only a safe non-persistent outcome; never persist the private tab identity, domain, timestamp, or source.
  - Clear private memory and cancel pending private requests when the browser reports the context is gone. A late response is ignored by scope/nonce validation and cannot populate normal state.
  - Add a browser capability probe for Chrome and Edge. A denied probe has a deterministic unavailable state and no attempt to inspect private tabs through page APIs, host permissions, or a normal-context snapshot.
- [ ] `IP-17-T05` **Việc 5 — Apply redaction before every durable or observable output.**
  - Validate UTF-8, schema, and field types first; select the explicit diagnostic allowlist second; redact/drop URL path, query, fragment, title, labels, raw IDs, query text, private identifiers, and arbitrary error detail third; then serialize and enforce the record-size cap. Redaction MUST happen before SQLite insertion, export, crash/error formatting, and test snapshots.
  - Replace raw correlation material with a per-session opaque ID or one-way local digest only when the contract needs correlation. Never use a reversible encoding, account identifier, raw profile ID, tab ID, title, URL, or query as a diagnostic correlation value.
  - Keep UI search results local and bounded: the user may see the allowed title/domain/URL context for the current scope, but the query, raw metadata, or private fields must not enter diagnostics or persistence. Strip URL credentials, query strings, fragments, and other unneeded components from any displayed context according to the privacy/display contract.
  - Ensure typed errors contain only the safe error literal, retryability, scope-safe request identity, and recovery action. Panic/crash paths and command-line output use the same deny-by-default formatter; no raw browser field is interpolated.
  - Test sentinel values through every output sink and assert absence byte-for-byte, including nested/unknown objects and malformed error values. Redaction failure is a hard test failure, not a warning.
- [ ] `IP-17-T06` **Việc 6 — Define secrets and input limits at the boundary.**
  - Consume IP-07's exact constants for frame bytes, identifier bytes, field bytes, query characters, tab count, synchronization count, result limit, timeout, and diagnostic record bytes; do not duplicate or silently widen them. Enforce result limit 1–50 and the declared 1,000-tab operating envelope before allocation/indexing.
  - Reject invalid UTF-8, NUL/control injection where the API does not permit it, negative/overflowing numeric values, unknown scope/context enums, duplicate conflicting identities, malformed URLs, missing required scope, and fields exceeding their limit. Omit optional fields only when their absence is an allowed contract state.
  - Reject or drop secret-shaped fields and never read environment secrets, cookies, session tokens, auth headers, private keys, browser storage values, or network payloads. Do not place secrets in fixtures, source, test snapshots, logs, crash reports, command arguments, or artifacts.
  - Bound diagnostic serialization and total retention bytes before SQLite writes. SQL statements remain static/parameterized; no tab/query/profile field is interpolated into SQL, a path, a shell command, or a host invocation.
  - Exercise oversized and adversarial inputs in `FX-PRIVACY-INPUT-LIMITS` and `FX-PRIVACY-SECRET-REJECTION`; expected outcomes are typed rejection or safe omission with no partial scope mutation and no sensitive echo.
- [ ] `IP-17-T07` **Việc 7 — Audit logs, retention, reset, and uninstall.**
  - Verify normal diagnostic rows are profile-scoped and private rows are absent. On insert/startup, apply IP-08's seven-day age eviction, then oldest-first eviction until serialized diagnostics are at most 10,000,000 bytes; cap one row before insertion and record only a safe eviction outcome.
  - Verify activation metadata is profile-scoped, at most 500 rows/30 days, and disabled for private context. Ensure a profile's retention pass cannot evict or inspect another profile's rows.
  - Run reset and uninstall fixtures to prove only InfoBoard-owned configuration, installation state, activation metadata, diagnostics, caches, manifests, and binaries are removed; browser tabs, browser history, unrelated profile data, and unrelated paths remain unchanged.
  - Inspect diagnostics and exports after permission denial, profile mismatch, private teardown, host crash, protocol mismatch, storage degradation, reset, and uninstall. Each transition must be explainable by safe status/error fields without raw metadata.
  - Add an audit result that reports permission set, API-call set, storage classes, retention limits, and fixture hashes. The audit itself contains no raw titles, URLs, queries, tokens, or private identifiers.
- [ ] `IP-17-T08` **Việc 8 — Exercise concrete abuse cases and hand off evidence.**
  - Cross-profile confused deputy: inject profile B's request into profile A's session; expect `PROFILE_MISMATCH`, no result/settings/row access, and no activation.
  - Private-to-normal leakage: send a private delta followed by a normal query and inspect index, result, SQLite, and logs; expect zero private fields outside the private scope.
  - Stale private response: end the private context before a delayed result; expect cancellation/ignore and no normal fallback activation.
  - Permission creep: add an unapproved permission/API in a fixture manifest; expect static audit failure before packaging.
  - Diagnostic exfiltration: inject sentinel title/URL/query/token/error text into every error and log path; expect redacted output with only allowlisted fields.
  - Resource exhaustion: send maximum and over-maximum frames, fields, tab counts, nesting, and log records; expect bounded rejection without process growth or SQLite mutation.
  - Retention bypass: insert exactly-at-boundary and over-boundary rows for two profiles; expect only the active profile's oldest rows to be evicted and the final byte/age limits to hold.

## 7. Kế hoạch commit

1. `feat(privacy): implement ip-17-t01`
   - Task IDs: `IP-17-T01`.
   - Owned target paths: extension/src/privacy/, host/privacy/.
   - Behavior: **Việc 1 — Freeze the field-class and ownership matrix.** In `extension/src/privacy/` and `host/privacy/` (to-create), record for every field its source, owner, allowed boundary, storage class, retention, display policy, and redaction rule:
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/privacy/... -run 'Test(Profile|Private|Redaction|Secret|InputLimit|Retention)'`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Việc 1 — Freeze the field-class and ownership matrix.** In `extension/src/privacy/` and `host/privacy/` (to-create), record for every field its source, owner, allowed boundary, storage class, retention, display policy, and redaction rule:
   - Dependency gate: all index.md dependencies for IP-17 have merged to dev; phase work branch starts from latest origin/dev.

2. `test(privacy): implement ip-17-t02`
   - Task IDs: `IP-17-T02`.
   - Owned target paths: `extension/src/privacy/` (to-create), `host/privacy/` (to-create), `fixtures/privacy/phase-17/` (to-create), `tests/privacy/phase-17/` (to-create).
   - Behavior: **Việc 2 — Implement the permission matrix and static audit.** Keep the manifest allowlist narrow and map each capability as follows:
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/privacy/... -run 'Test(Profile|Private|Redaction|Secret|InputLimit|Retention)'`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Việc 2 — Implement the permission matrix and static audit.** Keep the manifest allowlist narrow and map each capability as follows:
   - Dependency gate: all index.md dependencies for IP-17 have merged to dev; phase work branch starts from latest origin/dev.

3. `feat(privacy): implement ip-17-t03`
   - Task IDs: `IP-17-T03`.
   - Owned target paths: `extension/src/privacy/` (to-create), `host/privacy/` (to-create), `fixtures/privacy/phase-17/` (to-create), `tests/privacy/phase-17/` (to-create).
   - Behavior: **Việc 3 — Bind every message and browser action to one scope.**
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/privacy/... -run 'Test(Profile|Private|Redaction|Secret|InputLimit|Retention)'`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Việc 3 — Bind every message and browser action to one scope.**
   - Dependency gate: all index.md dependencies for IP-17 have merged to dev; phase work branch starts from latest origin/dev.

4. `feat(privacy): implement ip-17-t04`
   - Task IDs: `IP-17-T04`.
   - Owned target paths: `extension/src/privacy/` (to-create), `host/privacy/` (to-create), `fixtures/privacy/phase-17/` (to-create), `tests/privacy/phase-17/` (to-create).
   - Behavior: **Việc 4 — Enforce private-context lifetime and data flow.**
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/privacy/... -run 'Test(Profile|Private|Redaction|Secret|InputLimit|Retention)'`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Việc 4 — Enforce private-context lifetime and data flow.**
   - Dependency gate: all index.md dependencies for IP-17 have merged to dev; phase work branch starts from latest origin/dev.

5. `feat(privacy): implement ip-17-t05`
   - Task IDs: `IP-17-T05`.
   - Owned target paths: `extension/src/privacy/` (to-create), `host/privacy/` (to-create), `fixtures/privacy/phase-17/` (to-create), `tests/privacy/phase-17/` (to-create).
   - Behavior: **Việc 5 — Apply redaction before every durable or observable output.**
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/privacy/... -run 'Test(Profile|Private|Redaction|Secret|InputLimit|Retention)'`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Việc 5 — Apply redaction before every durable or observable output.**
   - Dependency gate: all index.md dependencies for IP-17 have merged to dev; phase work branch starts from latest origin/dev.

6. `feat(privacy): implement ip-17-t06`
   - Task IDs: `IP-17-T06`.
   - Owned target paths: `extension/src/privacy/` (to-create), `host/privacy/` (to-create), `fixtures/privacy/phase-17/` (to-create), `tests/privacy/phase-17/` (to-create).
   - Behavior: **Việc 6 — Define secrets and input limits at the boundary.**
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/privacy/... -run 'Test(Profile|Private|Redaction|Secret|InputLimit|Retention)'`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Việc 6 — Define secrets and input limits at the boundary.**
   - Dependency gate: all index.md dependencies for IP-17 have merged to dev; phase work branch starts from latest origin/dev.

7. `test(privacy): implement ip-17-t07`
   - Task IDs: `IP-17-T07`.
   - Owned target paths: `extension/src/privacy/` (to-create), `host/privacy/` (to-create), `fixtures/privacy/phase-17/` (to-create), `tests/privacy/phase-17/` (to-create).
   - Behavior: **Việc 7 — Audit logs, retention, reset, and uninstall.**
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/privacy/... -run 'Test(Profile|Private|Redaction|Secret|InputLimit|Retention)'`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Việc 7 — Audit logs, retention, reset, and uninstall.**
   - Dependency gate: all index.md dependencies for IP-17 have merged to dev; phase work branch starts from latest origin/dev.

8. `feat(privacy): implement ip-17-t08`
   - Task IDs: `IP-17-T08`.
   - Owned target paths: `extension/src/privacy/` (to-create), `host/privacy/` (to-create), `fixtures/privacy/phase-17/` (to-create), `tests/privacy/phase-17/` (to-create).
   - Behavior: **Việc 8 — Exercise concrete abuse cases and hand off evidence.**
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/privacy/... -run 'Test(Profile|Private|Redaction|Secret|InputLimit|Retention)'`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Việc 8 — Exercise concrete abuse cases and hand off evidence.**
   - Dependency gate: all index.md dependencies for IP-17 have merged to dev; phase work branch starts from latest origin/dev.

## 8. Kiểm chứng và nghiệm thu

- [ ] From the implementation repository root, run `git diff --check` and a path audit asserting that this phase changes only `docs/plan/implementation/phase-17-privacy-permissions-and-profile-isolation.md`.
- [ ] After the to-create tests exist, run `go test ./host/privacy/... -run 'Test(Profile|Private|Redaction|Secret|InputLimit|Retention)'` from the repository root with `INFOBOARD_PRIVACY_FIXTURES=fixtures/privacy/phase-17`; all scope, redaction, secret, limit, and retention cases pass.
- [ ] Run `python3 tests/privacy/phase-17/validate_permissions.py --manifest extension/manifest.json --fixtures fixtures/privacy/phase-17` from the repository root; it must fail on any unapproved permission, API, host pattern, private fallback, or excluded-surface call and pass `FX-PRIVACY-PERMISSION-MATRIX`.
- [ ] Run `python3 tests/privacy/phase-17/validate_redaction.py --fixtures fixtures/privacy/phase-17 --diagnostics fixtures/privacy/phase-17/output` from the repository root; sentinel title, URL, query, token, private ID, browser ID, and error text are absent from every persisted/exported output.
- [ ] Run `python3 tests/privacy/phase-17/run_browser_matrix.py --browsers chrome,edge --contexts normal,private --fixtures fixtures/privacy/phase-17` in each supported OS environment; permission grant/denial, profile switching, private teardown, and activation scope outcomes match the fixtures. A blocked environment must record the exact browser/OS reason and cannot be counted as pass.
- [ ] Run `python3 tests/privacy/phase-17/audit_retention.py --db fixtures/privacy/phase-17/retention.sqlite --max-age-days 30 --max-diagnostics-bytes 10000000`; assert diagnostics <= 7 days/10 MB, activation metadata <= 500/30 days, profile ownership is preserved, and no private row exists.
- [ ] Manually trace input -> observable output for profile A query, profile B injection, normal/private simultaneous tabs, denied private capability, delayed private response, redacted error, oversized frame, reset, and uninstall. The observable result must be rejection/degradation or a current-scope result, never cross-scope data or an unconfirmed activation.
- [ ] Acceptance is met only when FR-014's two-profile fixture isolates settings and projections; NFR-007's static and runtime audit finds no network access, page-content read, or unbounded hot-path collection; NFR-009's permission, API, storage, transport, log, and retention audit finds no excluded data source requested, collected, persisted, or transmitted; supporting FR-003 returns only current-profile supported fields and FR-015 removes only owned data.
- [ ] Security review records each abuse case, evidence command, expected failure class, residual risk, and reviewer decision. No known in-scope privacy, permission, profile-isolation, redaction, or retention defect remains.

## 9. Rủi ro và quyết định còn mở

- Rủi ro: Chrome and Edge may expose different permission requirements or private-context capabilities for the same tab/window/group API.
  - Phương án xử lý đã chọn: keep one product policy and maintain a browser-specific capability matrix. Use the narrow documented permission for each package, degrade missing optional fields, and fail closed when required identity or activation capability is unavailable. Do not broaden access to compensate.
- Rủi ro: an opaque profile identifier may be reused or a stale host process may receive a new profile's message.
  - Phương án xử lý đã chọn: bind every handshake and request to browser family, opaque profile ID, context kind, session identity, and projection revision; discard the old session on restart/switch and reject mismatch before reading state.
- Rủi ro: private context may end while a request or host response is in flight.
  - Phương án xử lý đã chọn: cancel/ignore by private session nonce and scope, clear memory, reject late results, and never fall back to normal context for activation or display.
- Rủi ro: title and URL are required for user value but can contain credentials, query material, or adversarial control characters.
  - Phương án xử lý đã chọn: keep them transient and local, enforce field/encoding limits, strip unneeded URL components for display, never persist them in activation/diagnostic data, and test sentinel redaction through every sink.
- Rủi ro: diagnostics can leak through an unexpected serializer, panic, dependency, or future field.
  - Phương án xử lý đã chọn: allowlist structured fields at one shared boundary, drop unknown fields, cap records before insertion, scan output fixtures byte-for-byte, and require a security review for any new data class.
- Câu hỏi còn mở chỉ khi câu trả lời có thể thay đổi contract: if a supported browser requires a permission that grants broader access than the current policy, stop packaging and record a new decision with user value, privacy impact, retention, removal, and threat-model evidence before implementation.

## 10. References ngoài `docs/`

- Skill: [`secure-development`](../../../.agent/skills/common/security/secure-development/SKILL.md), [`security-review`](../../../.agent/skills/common/security/security-review/SKILL.md), [`architecture-review`](../../../.agent/skills/personal/product/architecture-review/SKILL.md), [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md), [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md), [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md), [`git-workflow`](../../../.agent/skills/common/engineering/git-workflow/SKILL.md)
- Project context: [`CONTEXT.md`](../../../CONTEXT.md)
- Source/config/test path ngoài `docs/`: `extension/src/privacy/` (to-create), `host/privacy/` (to-create), `extension/manifest.json` (to-create, consumed from IP-03), `fixtures/privacy/phase-17/` (to-create), `tests/privacy/phase-17/` (to-create)
- Fixture/tool/artifact ngoài `docs/`: `FX-PRIVACY-PERMISSION-MATRIX`, `FX-PRIVACY-PROFILE-CROSS-READ`, `FX-PRIVACY-PROFILE-CROSS-ACTIVATE`, `FX-PRIVACY-PRIVATE-SEPARATION`, `FX-PRIVACY-PRIVATE-TEARDOWN`, `FX-PRIVACY-PERMISSION-DENIED`, `FX-PRIVACY-REDACTION`, `FX-PRIVACY-SECRET-REJECTION`, `FX-PRIVACY-INPUT-LIMITS`, `FX-PRIVACY-RETENTION-AUDIT`, `FX-PRIVACY-EXCLUDED-SURFACE` under `fixtures/privacy/phase-17/`; future commands under `tests/privacy/phase-17/`
