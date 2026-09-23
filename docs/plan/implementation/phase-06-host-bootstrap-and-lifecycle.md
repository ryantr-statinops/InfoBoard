# Phase 06 — Host bootstrap and lifecycle

> Plan ID: IP-06
> Status: See README.md execution tracker
> Execution owner: Go host-runtime implementer
> Dependencies: IP-01, IP-02
> Parallel boundary: IP-04 and IP-08 may proceed after IP-02; IP-07 consumes this host/session boundary and must align its wire contract before implementation merge
> Requirement IDs: NFR-006 (primary); FR-012 and operational timeout/error rules (supporting)
> Owned paths: `host/cmd/infoboard-host/main.go` (to-create); `host/internal/runtime/bootstrap.go` (to-create); `host/internal/runtime/session.go` (to-create); `host/internal/runtime/lifecycle.go` (to-create); `host/internal/runtime/cancellation.go` (to-create); `host/internal/runtime/health.go` (to-create); `fixtures/host/phase-06/` (to-create); `tests/host/phase-06/` (to-create)

## 1. Mục tiêu

- Define the process and session boundary for a Go Native Messaging host that is launched by Chrome or Edge and communicates only through the browser-provided framed stdin/stdout channel.
- Make startup, handshake handoff, synchronization handoff, ready operation, normal shutdown, unexpected disconnect, cancellation, and fresh-session reconnect explicit and bounded.
- Ensure the host exposes enough lifecycle state for the extension search surface to distinguish `healthy`, `unavailable`, and `recovering`, with a retryable/error reason and safe diagnostics for each transition. This supplies FR-012 while IP-07 owns the exact serialized message envelope.
- Provide an implementation seam that keeps browser authority in the extension, local indexing and runtime state in the host, and SQLite/persistence decisions in IP-08. A host process may retain memory while its Native Messaging connection is useful, but it must not become a permanent service or listening daemon.
- Make NFR-006 measurable: after a normal host failure, a new Native Messaging session can return to a healthy state through reconnect and full snapshot rebuild without reinstalling the product.

## 2. Phạm vi

- Bao gồm:
  - A small `host/cmd/infoboard-host` entrypoint that owns process bootstrap, root cancellation, signal/EOF handling, and exit status. `stdout` is reserved for Native Messaging frames; all diagnostics use a bounded, redacted local sink on `stderr` or the diagnostics seam.
  - Explicit session ownership for one browser-provided stdin/stdout connection: one frame reader owns stdin, one serialized writer owns stdout, and application handlers never write directly to either stream.
  - Lifecycle states and transitions for `Starting`, `Handshaking`, `Synchronizing`, `Ready`, `Closing`, `Unavailable`, and `Recovering`, plus the public health mapping `healthy`, `unavailable`, and `recovering`.
  - Startup sequencing: validate runtime limits and registration assumptions, construct cancellable dependencies, hand off protocol negotiation to IP-07, accept only a compatible profile/session, and hand off full snapshot synchronization to IP-05/IP-09 before exposing `Ready`.
  - Normal shutdown on explicit close or stdin EOF, bounded cancellation of in-flight work, bounded persistence flush through an IP-08 interface, final safe status/error where the channel is still writable, and deterministic process exit.
  - Unexpected host exit, broken pipe, malformed/disconnected stream, protocol timeout, and dependency failure classification. The extension is responsible for opening a fresh Native Messaging connection and snapshot; the host does not spawn a replacement process or reconnect through another transport.
  - Centralized timeout and resource-budget configuration for startup, handshake, request cancellation, shutdown drain, frame/queue sizes, concurrent work, and diagnostic retention handoffs. Every budget has a fixture-visible value and an observable timeout/error class.
  - Test seams for fake framed transports, blocking reads/writes, cancellation, EOF, host crash, reconnect with a fresh process/session, protocol mismatch, and health transition observation.
- Ngoài phạm vi:
  - Native Messaging JSON envelope fields, message types, typed error literals, compatibility negotiation, and wire serialization; IP-07 owns those contracts, while this phase supplies the session hooks and bounded transport ownership it needs.
  - Browser tab observation, eligibility, profile projection records, full snapshot/delta semantics, or projection reconciliation; IP-04/IP-05/IP-09 own those behaviors. This phase does not read browser APIs or mutate browser tabs.
  - Lexical normalization, indexing, ranking, query orchestration, result rendering, or activation; IP-10 through IP-14 own those paths.
  - SQLite schema, migrations, retention implementation, reset, or uninstall deletion; IP-08/IP-15/IP-18 own them. This phase only defines a narrow lifecycle callback for bounded flush/readiness and reports persistence health supplied by that owner.
  - Installer, host registration manifests, package signing, origin allowlists, or platform-specific installation; IP-18 owns registration and packaging. The host must validate the supplied runtime context but must not silently repair installation.
  - Any network listener, loopback server, loopback transport, cloud endpoint, telemetry channel, browser-history source, page-content access, or other data source outside the current open-tab contract.

## 3. Điều kiện tiên quyết

- IP-01 is merged and provides the requirement/fixture ownership map, logical roots, global limits vocabulary, error/health naming rules, and future test command conventions.
- IP-02 is merged and provides `profile_id`, ownership boundaries, projection epoch/revision semantics, and the rule that the extension remains authoritative for browser state. A host session must reject or quarantine a profile mismatch rather than infer a replacement identity.
- Read the binding [`product boundary`](../refactor/README.md), [`target architecture`](../refactor/architecture.md), [`requirements`](../refactor/requirements.md), [`runtime protocol`](../refactor/runtime-protocol.md), [`persistence and lifecycle`](../refactor/persistence-and-lifecycle.md), [`packaging and operations`](../refactor/packaging-and-operations.md), [`verification and acceptance`](../refactor/verification-and-acceptance.md), [`roadmap`](../refactor/roadmap.md), and [`decision log`](../refactor/decisions.md).
- Read [`CONTEXT.md`](../../../CONTEXT.md). The implementation roots are currently absent, so every owned source, fixture, and test path in this plan is a logical `to-create` target and must not be described as an observed file.
- The Native Messaging registration and browser adapter contract exist as an implementation prerequisite or fixture seam. The host is started by the browser; it must not assume a shell, a current working directory, writable installation directories, or an available network interface.
- IP-07 and IP-09 must consume the lifecycle interfaces without duplicating process ownership. Before implementation, agree the exact wire representation for health/error transitions while retaining the state semantics below.

## 4. Đầu ra cần bàn giao

- [ ] `host/cmd/infoboard-host/main.go` (to-create): a thin entrypoint that creates the root context, bounded runtime configuration, redacted diagnostic sink, framed session, and shutdown path; it has no business logic and emits no non-frame bytes to stdout.
- [ ] `host/internal/runtime/bootstrap.go` (to-create): dependency construction and startup ordering with explicit failure classification. It validates limits and required registration/runtime assumptions, then returns a session that is either ready for protocol handoff or reports a safe `unavailable` reason.
- [ ] `host/internal/runtime/session.go` (to-create): one-connection ownership model with a single reader, serialized writer, session identity, state machine, request cancellation registry, and hooks for protocol, projection/index, persistence, and diagnostics owners. No shared mutable process-global session state.
- [ ] `host/internal/runtime/lifecycle.go` (to-create): legal transition table, idempotent close, EOF/broken-pipe classification, reconnect boundary, and bounded shutdown drain. A finished session cannot accept new work or report itself healthy.
- [ ] `host/internal/runtime/cancellation.go` (to-create): root/session/request contexts, cancellation propagation, deadline composition, cancellation cause mapping, and bounded handling of blocked fake/real I/O. A request cancellation must not leak a goroutine or cancel an unrelated session.
- [ ] `host/internal/runtime/health.go` (to-create): an internal health snapshot and observer seam containing public state (`healthy`, `unavailable`, or `recovering`), lifecycle substate, retryability, safe error class, protocol/host versions when known, projection revision/count when supplied, and transition timing/counts. It must never require raw titles, URLs, query strings, tokens, or page data.
- [ ] A documented handoff contract for the following sequence: `Starting -> Handshaking -> Synchronizing -> Ready`; `Ready -> Closing` for normal end; any broken session -> `Unavailable` or `Recovering` according to whether a new connection is pending; a fresh process/session starts at `Starting` and must receive a full snapshot before `Ready`.
- [ ] Fixtures under `fixtures/host/phase-06/` with bounded input, expected observable output, and owning requirement:
  - `FX-HOST-START-HEALTHY`: registered host receives a compatible handshake/snapshot handoff and exposes `healthy` only after synchronization readiness.
  - `FX-HOST-ABSENT`: no host process/registration or failed bootstrap exposes `unavailable`, a safe retryable reason, and no tab mutation.
  - `FX-HOST-RECOVERING`: unexpected EOF/crash transitions from healthy to `recovering` or `unavailable` with a bounded reconnect hint; a fresh session plus snapshot reaches healthy without reinstall.
  - `FX-HOST-STDIO-OWNERSHIP`: a protocol frame is the only stdout byte sequence, diagnostics are absent from stdout, and concurrent responses are serialized without interleaving.
  - `FX-HOST-TIMEOUT`: a blocked handshake/request/shutdown operation reaches its configured deadline, returns its typed timeout/error class, and leaves no unbounded wait.
  - `FX-HOST-CANCEL-DISCONNECT`: client cancellation and stdin EOF stop in-flight work, close the session idempotently, and leave no leaked worker in the fake transport harness.
  - `FX-HOST-RESOURCE-LIMIT`: oversized frames, excessive queued work, or too many concurrent operations are rejected before unbounded allocation and are observable as a bounded error.
  - `FX-HOST-NO-LOOPBACK`: process/network inspection of the host path finds no listening socket, network dial, or child daemon; communication remains browser-provided stdin/stdout only.
- [ ] Tests under `tests/host/phase-06/` (to-create) exercise state transitions and observable fixture output with deterministic clocks and fake transports. Include healthy, unavailable, recovering, cancellation, disconnect, timeout, serialization, bounded-resource, and no-loopback cases on Linux; platform packaging smoke tests consume the same lifecycle contract later.

## 5. Skill và tài liệu áp dụng

- Skill tags:
  - [`go`](../../../.agent/skills/personal/engineering/backend/go/SKILL.md): define explicit Go packages, contexts, interfaces, concurrency ownership, standard tooling, and measurable runtime behavior.
  - [`architecture-review`](../../../.agent/skills/personal/product/architecture-review/SKILL.md): map the process/session boundary, failure isolation, operational ownership, and the rejected complexity of a standalone service.
  - [`secure-development`](../../../.agent/skills/common/security/secure-development/SKILL.md): constrain untrusted frames, file/process boundaries, diagnostics, permissions, and fail-closed behavior on uncertain protocol or identity state.
  - [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md): specify deterministic success, invalid, timeout, cancellation, disconnect, and resource-bound behavior at observable seams.
  - [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md): record a standalone lifecycle contract, prerequisites, failure presentation, and exact future verification commands.
  - [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md): sequence bootstrap, session, cancellation, health, fixtures, and acceptance around the dependency boundary.
  - [`git-workflow`](../../../.agent/skills/common/engineering/git-workflow/SKILL.md): keep the phase plan and future implementation slices focused and reviewable.
- Tài liệu trong `docs/`: [`README`](../refactor/README.md), [`architecture`](../refactor/architecture.md), [`requirements`](../refactor/requirements.md), [`runtime protocol`](../refactor/runtime-protocol.md), [`persistence and lifecycle`](../refactor/persistence-and-lifecycle.md), [`packaging and operations`](../refactor/packaging-and-operations.md), [`verification and acceptance`](../refactor/verification-and-acceptance.md), [`roadmap`](../refactor/roadmap.md), and [`decision log`](../refactor/decisions.md).
- Quy ước code, ADR, context ngoài `docs/`: [`CONTEXT.md`](../../../CONTEXT.md); logical future targets are `host/`, `fixtures/host/phase-06/`, and `tests/host/phase-06/`. Keep browser authority in `extension/`, transport envelopes in the IP-07 boundary, projection/index state in their owning host packages, and durable storage in the IP-08 repository.

## 6. Công việc triển khai

- [ ] `IP-06-T01` **Bootstrap and process boundary:** create the thin `main` entrypoint and runtime configuration. Acquire stdin/stdout once, reject unsupported invocation/configuration before accepting tab data, route diagnostics to a bounded redacted sink, and make process exit codes distinguish clean close from startup/protocol/runtime failure without leaking sensitive input.
- [ ] `IP-06-T02` **Framed stream ownership:** implement a transport seam in which exactly one reader owns stdin and exactly one serialized writer owns stdout. Enforce the protocol frame size before allocation (with IP-07), flush complete frames, reject partial/malformed input fail-closed, and ensure no `fmt.Print`, default logger, panic text, or child-process output can contaminate stdout.
- [ ] `IP-06-T03` **Session state machine:** implement legal transitions and idempotent terminal behavior. Do not mark `healthy` at process start or after handshake alone; require protocol compatibility and successful projection/index readiness handoff. A session with unknown profile, invalid version, failed limits, or rebuilding projection remains unavailable/recovering and cannot accept query work.
- [ ] `IP-06-T04` **Startup handoff:** order runtime limit construction, protocol hello/ack, profile validation, persistence readiness callback, and snapshot/index readiness callback. Each handoff receives a context/deadline and returns a typed safe error. Do not send or persist tab data before protocol/profile checks succeed.
- [ ] `IP-06-T05` **Normal shutdown:** on explicit close or clean stdin EOF, stop accepting new work, cancel request contexts, wait only up to the configured shutdown budget, flush bounded local metadata through the IP-08 interface if available, emit a final safe status/error when writable, close stdout and dependencies exactly once, and exit without closing tabs or touching unrelated browser state.
- [ ] `IP-06-T06` **Failure and reconnect:** classify broken pipe, EOF, host crash, context cancellation, timeout, protocol mismatch, and dependency failure separately. A session failure must be visible to the extension as unavailable or recovering with retryability. Reconnect means the extension requests a new browser Native Messaging connection; the new host session starts clean and requires an authoritative snapshot, rather than trusting stale memory or trying to connect to a port.
- [ ] `IP-06-T07` **Cancellation and bounded concurrency:** derive per-request deadlines from one central budget, propagate cancellation to protocol/index/persistence callbacks, cap in-flight operations and queued frames, and make blocked fake transport operations interruptible in tests. On deadline, return the relevant safe class (for example `QUERY_TIMEOUT` or `HOST_SHUTDOWN` at the owning wire layer) and release all per-request resources.
- [ ] `IP-06-T08` **Health and diagnostics:** expose `healthy`, `unavailable`, and `recovering` transitions to an observer consumed by IP-07/IP-16. Record only state, cause/error class, retryability, duration, counts, versions, and projection revision/count when allowed. Redact raw tab fields and keep diagnostics bounded; persistence failure changes the supplied persistence status but does not make lexical runtime unavailable by itself.
- [ ] `IP-06-T09` **Resource and security limits:** enforce configured maximum frame/queue/request sizes, concurrent work, startup and shutdown durations, and diagnostic writes before large allocations or goroutine creation. Avoid network APIs, filesystem writes outside explicit lifecycle dependencies, shell execution, and process spawning in the hot path. Verify that no package opens a listening socket or dials a remote endpoint.
- [ ] `IP-06-T10` **Fixture-driven tests:** implement the eight host fixtures above with a deterministic clock and fake `io.Reader`/`io.Writer`/transport. Assert observable state/output, exact bounded completion, frame integrity, error class/retryability, and cleanup; do not assert private goroutine layout or incidental log text.

## 7. Kế hoạch commit

1. `feat(host): implement ip-06-t01`
   - Task IDs: `IP-06-T01`.
   - Owned target paths: `host/cmd/infoboard-host/main.go` (to-create); `host/internal/runtime/bootstrap.go` (to-create); `host/internal/runtime/session.go` (to-create); `host/internal/runtime/lifecycle.go` (to-create); `host/internal/runtime/cancellation.go` (to-create); `host/internal/runtime/health.go` (to-create); `fixtures/host/phase-06/` (to-create); `tests/host/phase-06/` (to-create).
   - Behavior: **Bootstrap and process boundary:** create the thin `main` entrypoint and runtime configuration. Acquire stdin/stdout once, reject unsupported invocation/configuration before accepting tab data, route diagnostics to a bounded redacted sink, and make process exit codes distinguish clean close from startup/protocol/runtime failure without leaking sensitive input.
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/... ./tests/host/phase-06 -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Bootstrap and process boundary:** create the thin `main` entrypoint and runtime configuration. Acquire stdin/stdout once, reject unsupported invocation/configuration before accepting tab data, route diagnostics to a bounded redacted sink, and make process exit codes distinguish clean close from startup/protocol/runtime failure without leaking sensitive input.
   - Dependency gate: all index.md dependencies for IP-06 have merged to dev; phase work branch starts from latest origin/dev.

2. `feat(host): implement ip-06-t02`
   - Task IDs: `IP-06-T02`.
   - Owned target paths: `host/cmd/infoboard-host/main.go` (to-create); `host/internal/runtime/bootstrap.go` (to-create); `host/internal/runtime/session.go` (to-create); `host/internal/runtime/lifecycle.go` (to-create); `host/internal/runtime/cancellation.go` (to-create); `host/internal/runtime/health.go` (to-create); `fixtures/host/phase-06/` (to-create); `tests/host/phase-06/` (to-create).
   - Behavior: **Framed stream ownership:** implement a transport seam in which exactly one reader owns stdin and exactly one serialized writer owns stdout. Enforce the protocol frame size before allocation (with IP-07), flush complete frames, reject partial/malformed input fail-closed, and ensure no `fmt.Print`, default logger, panic text, or child-process output can contaminate stdout.
   - Fixture and command: IP-07; run `go test ./host/... ./tests/host/phase-06 -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Framed stream ownership:** implement a transport seam in which exactly one reader owns stdin and exactly one serialized writer owns stdout. Enforce the protocol frame size before allocation (with IP-07), flush complete frames, reject partial/malformed input fail-closed, and ensure no `fmt.Print`, default logger, panic text, or child-process output can contaminate stdout.
   - Dependency gate: all index.md dependencies for IP-06 have merged to dev; phase work branch starts from latest origin/dev.

3. `feat(host): implement ip-06-t03`
   - Task IDs: `IP-06-T03`.
   - Owned target paths: `host/cmd/infoboard-host/main.go` (to-create); `host/internal/runtime/bootstrap.go` (to-create); `host/internal/runtime/session.go` (to-create); `host/internal/runtime/lifecycle.go` (to-create); `host/internal/runtime/cancellation.go` (to-create); `host/internal/runtime/health.go` (to-create); `fixtures/host/phase-06/` (to-create); `tests/host/phase-06/` (to-create).
   - Behavior: **Session state machine:** implement legal transitions and idempotent terminal behavior. Do not mark `healthy` at process start or after handshake alone; require protocol compatibility and successful projection/index readiness handoff. A session with unknown profile, invalid version, failed limits, or rebuilding projection remains unavailable/recovering and cannot accept query work.
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/... ./tests/host/phase-06 -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Session state machine:** implement legal transitions and idempotent terminal behavior. Do not mark `healthy` at process start or after handshake alone; require protocol compatibility and successful projection/index readiness handoff. A session with unknown profile, invalid version, failed limits, or rebuilding projection remains unavailable/recovering and cannot accept query work.
   - Dependency gate: all index.md dependencies for IP-06 have merged to dev; phase work branch starts from latest origin/dev.

4. `feat(host): implement ip-06-t04`
   - Task IDs: `IP-06-T04`.
   - Owned target paths: `host/cmd/infoboard-host/main.go` (to-create); `host/internal/runtime/bootstrap.go` (to-create); `host/internal/runtime/session.go` (to-create); `host/internal/runtime/lifecycle.go` (to-create); `host/internal/runtime/cancellation.go` (to-create); `host/internal/runtime/health.go` (to-create); `fixtures/host/phase-06/` (to-create); `tests/host/phase-06/` (to-create).
   - Behavior: **Startup handoff:** order runtime limit construction, protocol hello/ack, profile validation, persistence readiness callback, and snapshot/index readiness callback. Each handoff receives a context/deadline and returns a typed safe error. Do not send or persist tab data before protocol/profile checks succeed.
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/... ./tests/host/phase-06 -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Startup handoff:** order runtime limit construction, protocol hello/ack, profile validation, persistence readiness callback, and snapshot/index readiness callback. Each handoff receives a context/deadline and returns a typed safe error. Do not send or persist tab data before protocol/profile checks succeed.
   - Dependency gate: all index.md dependencies for IP-06 have merged to dev; phase work branch starts from latest origin/dev.

5. `feat(host): implement ip-06-t05`
   - Task IDs: `IP-06-T05`.
   - Owned target paths: `host/cmd/infoboard-host/main.go` (to-create); `host/internal/runtime/bootstrap.go` (to-create); `host/internal/runtime/session.go` (to-create); `host/internal/runtime/lifecycle.go` (to-create); `host/internal/runtime/cancellation.go` (to-create); `host/internal/runtime/health.go` (to-create); `fixtures/host/phase-06/` (to-create); `tests/host/phase-06/` (to-create).
   - Behavior: **Normal shutdown:** on explicit close or clean stdin EOF, stop accepting new work, cancel request contexts, wait only up to the configured shutdown budget, flush bounded local metadata through the IP-08 interface if available, emit a final safe status/error when writable, close stdout and dependencies exactly once, and exit without closing tabs or touching unrelated browser state.
   - Fixture and command: IP-08; run `go test ./host/... ./tests/host/phase-06 -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Normal shutdown:** on explicit close or clean stdin EOF, stop accepting new work, cancel request contexts, wait only up to the configured shutdown budget, flush bounded local metadata through the IP-08 interface if available, emit a final safe status/error when writable, close stdout and dependencies exactly once, and exit without closing tabs or touching unrelated browser state.
   - Dependency gate: all index.md dependencies for IP-06 have merged to dev; phase work branch starts from latest origin/dev.

6. `feat(host): implement ip-06-t06`
   - Task IDs: `IP-06-T06`.
   - Owned target paths: `host/cmd/infoboard-host/main.go` (to-create); `host/internal/runtime/bootstrap.go` (to-create); `host/internal/runtime/session.go` (to-create); `host/internal/runtime/lifecycle.go` (to-create); `host/internal/runtime/cancellation.go` (to-create); `host/internal/runtime/health.go` (to-create); `fixtures/host/phase-06/` (to-create); `tests/host/phase-06/` (to-create).
   - Behavior: **Failure and reconnect:** classify broken pipe, EOF, host crash, context cancellation, timeout, protocol mismatch, and dependency failure separately. A session failure must be visible to the extension as unavailable or recovering with retryability. Reconnect means the extension requests a new browser Native Messaging connection; the new host session starts clean and requires an authoritative snapshot, rather than trusting stale memory or trying to connect to a port.
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/... ./tests/host/phase-06 -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Failure and reconnect:** classify broken pipe, EOF, host crash, context cancellation, timeout, protocol mismatch, and dependency failure separately. A session failure must be visible to the extension as unavailable or recovering with retryability. Reconnect means the extension requests a new browser Native Messaging connection; the new host session starts clean and requires an authoritative snapshot, rather than trusting stale memory or trying to connect to a port.
   - Dependency gate: all index.md dependencies for IP-06 have merged to dev; phase work branch starts from latest origin/dev.

7. `test(host): implement ip-06-t07`
   - Task IDs: `IP-06-T07`.
   - Owned target paths: `host/cmd/infoboard-host/main.go` (to-create); `host/internal/runtime/bootstrap.go` (to-create); `host/internal/runtime/session.go` (to-create); `host/internal/runtime/lifecycle.go` (to-create); `host/internal/runtime/cancellation.go` (to-create); `host/internal/runtime/health.go` (to-create); `fixtures/host/phase-06/` (to-create); `tests/host/phase-06/` (to-create).
   - Behavior: **Cancellation and bounded concurrency:** derive per-request deadlines from one central budget, propagate cancellation to protocol/index/persistence callbacks, cap in-flight operations and queued frames, and make blocked fake transport operations interruptible in tests. On deadline, return the relevant safe class (for example `QUERY_TIMEOUT` or `HOST_SHUTDOWN` at the owning wire layer) and release all per-request resources.
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/... ./tests/host/phase-06 -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Cancellation and bounded concurrency:** derive per-request deadlines from one central budget, propagate cancellation to protocol/index/persistence callbacks, cap in-flight operations and queued frames, and make blocked fake transport operations interruptible in tests. On deadline, return the relevant safe class (for example `QUERY_TIMEOUT` or `HOST_SHUTDOWN` at the owning wire layer) and release all per-request resources.
   - Dependency gate: all index.md dependencies for IP-06 have merged to dev; phase work branch starts from latest origin/dev.

8. `feat(host): implement ip-06-t08`
   - Task IDs: `IP-06-T08`.
   - Owned target paths: `host/cmd/infoboard-host/main.go` (to-create); `host/internal/runtime/bootstrap.go` (to-create); `host/internal/runtime/session.go` (to-create); `host/internal/runtime/lifecycle.go` (to-create); `host/internal/runtime/cancellation.go` (to-create); `host/internal/runtime/health.go` (to-create); `fixtures/host/phase-06/` (to-create); `tests/host/phase-06/` (to-create).
   - Behavior: **Health and diagnostics:** expose `healthy`, `unavailable`, and `recovering` transitions to an observer consumed by IP-07/IP-16. Record only state, cause/error class, retryability, duration, counts, versions, and projection revision/count when allowed. Redact raw tab fields and keep diagnostics bounded; persistence failure changes the supplied persistence status but does not make lexical runtime unavailable by itself.
   - Fixture and command: IP-07, IP-16; run `go test ./host/... ./tests/host/phase-06 -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Health and diagnostics:** expose `healthy`, `unavailable`, and `recovering` transitions to an observer consumed by IP-07/IP-16. Record only state, cause/error class, retryability, duration, counts, versions, and projection revision/count when allowed. Redact raw tab fields and keep diagnostics bounded; persistence failure changes the supplied persistence status but does not make lexical runtime unavailable by itself.
   - Dependency gate: all index.md dependencies for IP-06 have merged to dev; phase work branch starts from latest origin/dev.

9. `feat(host): implement ip-06-t09`
   - Task IDs: `IP-06-T09`.
   - Owned target paths: `host/cmd/infoboard-host/main.go` (to-create); `host/internal/runtime/bootstrap.go` (to-create); `host/internal/runtime/session.go` (to-create); `host/internal/runtime/lifecycle.go` (to-create); `host/internal/runtime/cancellation.go` (to-create); `host/internal/runtime/health.go` (to-create); `fixtures/host/phase-06/` (to-create); `tests/host/phase-06/` (to-create).
   - Behavior: **Resource and security limits:** enforce configured maximum frame/queue/request sizes, concurrent work, startup and shutdown durations, and diagnostic writes before large allocations or goroutine creation. Avoid network APIs, filesystem writes outside explicit lifecycle dependencies, shell execution, and process spawning in the hot path. Verify that no package opens a listening socket or dials a remote endpoint.
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/... ./tests/host/phase-06 -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Resource and security limits:** enforce configured maximum frame/queue/request sizes, concurrent work, startup and shutdown durations, and diagnostic writes before large allocations or goroutine creation. Avoid network APIs, filesystem writes outside explicit lifecycle dependencies, shell execution, and process spawning in the hot path. Verify that no package opens a listening socket or dials a remote endpoint.
   - Dependency gate: all index.md dependencies for IP-06 have merged to dev; phase work branch starts from latest origin/dev.

10. `test(host): implement ip-06-t10`
   - Task IDs: `IP-06-T10`.
   - Owned target paths: `host/cmd/infoboard-host/main.go` (to-create); `host/internal/runtime/bootstrap.go` (to-create); `host/internal/runtime/session.go` (to-create); `host/internal/runtime/lifecycle.go` (to-create); `host/internal/runtime/cancellation.go` (to-create); `host/internal/runtime/health.go` (to-create); `fixtures/host/phase-06/` (to-create); `tests/host/phase-06/` (to-create).
   - Behavior: **Fixture-driven tests:** implement the eight host fixtures above with a deterministic clock and fake `io.Reader`/`io.Writer`/transport. Assert observable state/output, exact bounded completion, frame integrity, error class/retryability, and cleanup; do not assert private goroutine layout or incidental log text.
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/... ./tests/host/phase-06 -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Fixture-driven tests:** implement the eight host fixtures above with a deterministic clock and fake `io.Reader`/`io.Writer`/transport. Assert observable state/output, exact bounded completion, frame integrity, error class/retryability, and cleanup; do not assert private goroutine layout or incidental log text.
   - Dependency gate: all index.md dependencies for IP-06 have merged to dev; phase work branch starts from latest origin/dev.

## 8. Kiểm chứng và nghiệm thu

- [ ] From repository root, once the Go module exists, run `go test ./host/... ./tests/host/phase-06 -count=1` with fixtures loaded from `fixtures/host/phase-06/`; the run must report fixture IDs and pass healthy, unavailable, recovering, cancellation, disconnect, timeout, resource-limit, stdio-ownership, and no-loopback cases.
- [ ] Run `go test -race ./host/... ./tests/host/phase-06` and assert that concurrent frame production remains serialized, cancellation does not leak work, and close is idempotent.
- [ ] Launch the future host binary under the Native Messaging fixture harness with a compatible hello and full snapshot. Observe `healthy` only after the snapshot/index readiness callback; send EOF or terminate the host, observe `unavailable`/`recovering` with retryability, create a fresh connection, resend the authoritative snapshot, and observe `healthy` without reinstalling or changing browser tabs.
- [ ] Drive `FX-HOST-TIMEOUT` and `FX-HOST-CANCEL-DISCONNECT` with deliberately blocked fake reads/writes. Each case returns its declared typed error within the configured budget, closes the session, and reports no leaked worker or unbounded queue.
- [ ] Inspect captured stdout bytes for every fixture: only valid Native Messaging frames are present, no diagnostic or panic text is present, and concurrent writes are not interleaved. Inspect the redacted diagnostic sink to confirm it contains state/error metadata but no raw title, URL, query, token, or page data.
- [ ] Run the no-loopback harness (`ss -ltnp`/platform equivalent plus injected dial/listener guards) while starting, operating, disconnecting, and reconnecting the host. It must observe no host-owned listening socket, network dial, child daemon, or alternate transport.
- [ ] Confirm NFR-006 and FR-012 evidence: normal host failure reaches a new healthy session through bounded reconnect plus full snapshot; the search-surface health contract can distinguish healthy, unavailable, and recovering and can present a retryable action.
- [ ] Confirm operational rules: every request path has a bounded timeout and observable error class; every transition is diagnosable with local redacted data; incompatible protocol/session identity fails closed; shutdown and reconnect do not mutate browser state.

## 9. Rủi ro và quyết định còn mở

- Rủi ro: Native Messaging pipes do not expose portable write deadlines, so a naive blocking write can outlive cancellation. Phương án xử lý đã chọn: isolate all writes behind the serialized writer seam, use context-aware bounded queues and close-on-shutdown behavior, test blocked fake writers, and terminate the host session at the shutdown budget rather than waiting indefinitely. Platform-specific behavior must be measured in IP-19.
- Rủi ro: A process exit can happen before it sends a final health frame. Phương án xử lý đã chọn: the extension treats EOF/process absence as unavailable or recovering using the last safe session state; host diagnostics are best-effort and never required to declare failure.
- Rủi ro: SQLite initialization or migration failure could incorrectly make search unavailable. Phương án xử lý đã chọn: expose persistence as a degraded dependency status; keep host lifecycle/index readiness independent where the IP-08 contract allows, and never use SQLite as live-tab truth.
- Rủi ro: Reconnect could accidentally reuse an old in-memory projection or profile. Phương án xử lý đã chọn: each Native Messaging connection has a new session/epoch, requires profile validation and an authoritative snapshot before `healthy`, and rejects stale references.
- Rủi ro: A future implementation may add a convenient local port or shell helper that violates the product boundary. Phương án xử lý đã chọn: keep transport construction limited to stdin/stdout, add a no-loopback fixture and static review check, and require a new decision record before changing the boundary.
- Câu hỏi còn mở chỉ khi câu trả lời có thể thay đổi contract: the exact production numeric timeout/resource values and platform-specific interrupt strategy must be selected from IP-01 limits and measured reference environments; changing the state meanings, transport boundary, or recovery requirement requires an updated refactor decision before implementation.

## 10. References ngoài `docs/`

- Skill: [`go`](../../../.agent/skills/personal/engineering/backend/go/SKILL.md)
- Skill: [`architecture-review`](../../../.agent/skills/personal/product/architecture-review/SKILL.md)
- Skill: [`secure-development`](../../../.agent/skills/common/security/secure-development/SKILL.md)
- Skill: [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md)
- Skill: [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md)
- Skill: [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md)
- Skill: [`git-workflow`](../../../.agent/skills/common/engineering/git-workflow/SKILL.md)
- Project context: [`CONTEXT.md`](../../../CONTEXT.md)
- Source/config/test path ngoài `docs/`: `host/cmd/infoboard-host/main.go` (to-create); `host/internal/runtime/` (to-create); `tests/host/phase-06/` (to-create)
- Fixture/tool/artifact ngoài `docs/`: `fixtures/host/phase-06/FX-HOST-START-HEALTHY` through `FX-HOST-NO-LOOPBACK` (to-create); future Go fixture harness and Native Messaging framed-stdio runner (to-create)
