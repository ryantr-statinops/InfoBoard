# Phase 16 — Diagnostics, health, and observability

> Plan ID: IP-16
> Status: not_started
> Execution owner: Host reliability and diagnostics implementer
> Dependencies: IP-06, IP-07, IP-08, IP-09, IP-12
> Parallel boundary: IP-13, IP-14, IP-15 (after their listed dependencies; no shared owned paths)
> Requirement IDs: FR-012, NFR-006; supporting FR-013, NFR-009; operational diagnosable-transition and log-policy requirements
> Owned paths: `host/diagnostics/` (to-create), `fixtures/diagnostics/` (to-create), `tests/diagnostics/` (to-create)

## 1. Mục tiêu

- Tạo một health model có thể quan sát từ extension search surface và từ local diagnostics mà không làm lộ raw title, URL, query, token, page content, hay dữ liệu ngoài product boundary.
- Phân biệt rõ host availability, Native Messaging protocol/session state, tab-projection freshness, in-memory lexical-index freshness, and SQLite persistence health. A healthy lexical path MUST remain usable when optional durable persistence is degraded.
- Emit one bounded, structured diagnostic event for every runtime state transition, protocol failure, synchronization/rebuild milestone, storage transition, query degradation, and repair outcome. The event must explain what changed, why, whether a retry is safe, and what action is allowed.
- Give the UI a stable, user-safe status and explicit retryability so host-down, stale-index, rebuilding, protocol incompatibility, and persistence-degraded states are visible and actionable (FR-012, NFR-006, supporting FR-013).
- Enforce local diagnostic retention of at most 7 days and 10 MB, whichever is reached first, with oldest-first eviction and a bounded per-record size. Retention itself must be observable without creating an unbounded failure loop.

## 2. Phạm vi

- Bao gồm:
  - A planned Go diagnostics package under `host/diagnostics/` for health snapshots, component freshness, transition events, typed error mapping, redaction, bounded repair orchestration, and retention accounting. These are `to-create` modules; no source tree is assumed to exist yet.
  - An aggregate health state machine aligned with the host lifecycle: `disconnected`, `connecting`, `handshaking`, `synchronizing`, `ready`, `degraded`, `recovering`, `incompatible`, and `stopping`. Component statuses distinguish `host`, `protocol`, `projection`, `index`, and `storage`, so `degraded` means lexical search may continue while persistence is unhealthy rather than hiding the failure.
  - Explicit freshness fields: current profile scope, last successful handshake, last accepted projection revision, indexed projection revision, last snapshot/delta sequence, index build status, storage migration status, and age/duration counters. Overall `ready` requires a valid session, compatible protocol, an index built from the current projection revision, and no unresolved sync gap. Storage degradation does not invalidate lexical search.
  - Transition records with bounded identifiers and metadata: diagnostic schema version, event ID, event time, monotonic duration, session correlation ID, component, `from_state`, `to_state`, reason/error class, projection/index revisions, protocol/ranking/schema versions, record counts, latency buckets, `retryable`, and `recommended_action`. Profile and request correlation values are opaque local identifiers or one-way local digests; browser tab IDs are not exported as diagnostic content.
  - A health payload consumed through IP-07's `health`/`health_result` contract. The extension owns presentation and browser authority; the Go Native Messaging host owns health calculation, diagnostics emission, local redaction, and the bounded persistence adapter. IP-13 owns the visual status surface and accessibility wording.
  - A complete mapping from protocol error classes and lifecycle failures to user-safe status codes, retryability, automatic-retry policy, and repair guidance. The mapping must preserve request identity without echoing sensitive request fields.
  - Bounded repair actions: retry the Native Messaging connection, request a full projection snapshot, rebuild the in-memory lexical index, and retry/reopen the SQLite repository through IP-08's ownership boundary. Repair never closes or changes browser tabs, never starts a loopback listener, and never accepts an unbounded retry loop.
  - Structured redaction and local diagnostics export. The default allowlist is counts, durations, enum states, versions, revisions, sizes, status codes, and error classes. Redaction must cover raw URLs, query strings, URL fragments, titles, page content, cookies, session tokens, secrets, and arbitrary browser-field values before persistence or export.
  - Retention enforcement in the SQLite diagnostics repository owned by IP-08: delete records older than 7 days first, then evict oldest records until serialized diagnostic payload bytes are at most `10_000_000` bytes. A record is bounded before insertion; a rejected or summarized oversized record must itself be counted as a safe diagnostic outcome.
  - Fixture-driven coverage for startup, host crash/reconnect, protocol mismatch, profile mismatch, snapshot/delta gaps, index rebuild, stale revisions, query timeout, storage migration/corruption, privacy redaction, repair bounds, and both retention limits.
- Ngoài phạm vi:
  - Browser tab observation, event conversion, projection reconciliation, or browser activation; those are owned by IP-04/IP-05/IP-14 and are consumed here only through their revision and outcome signals.
  - Native Messaging framing, envelope parsing, message-type definitions, protocol version negotiation, and transport limits; IP-07 owns those contracts. This phase defines only the health/diagnostic fields and mappings that implement their observable outcomes.
  - SQLite schema/migration mechanics, corruption quarantine, reset, and uninstall deletion; IP-08 owns storage mechanics. This phase supplies the diagnostics repository contract, retention assertions, and repair outcome reporting.
  - Lexical normalization, ranking weights, query orchestration, and result rendering; IP-10/IP-11/IP-12/IP-13 own those behaviors. This phase reports their freshness, timeout, and degraded outcomes without collecting query text.
  - Any network service, cloud telemetry, page-content access, browser-history data source, or permanent daemon. Diagnostics remain local and bounded.

## 3. Điều kiện tiên quyết

- IP-06 has defined the Go Native Messaging host lifecycle, process cancellation, startup/shutdown, and reconnect boundary, including bounded host absence behavior.
- IP-07 has defined the versioned message envelope, `health`/`health_result` messages, request IDs, limits, and typed error literals. This phase MUST use those contracts rather than inventing a second transport.
- IP-08 has defined the transactional SQLite repository, migration state, corruption/degraded behavior, reset ownership, and the diagnostics store's 7-day/10 MB policy boundary.
- IP-09 has defined snapshot/delta reconciliation, resync, reconnect, projection revision, index rebuild, and the host behavior for unknown or rebuilding revisions.
- IP-12 has defined query status, timeout, stale/rebuilding rejection, host-down behavior, and persistence-degraded result behavior.
- Canonical contracts to reread during implementation are [requirements](../refactor/requirements.md), [architecture](../refactor/architecture.md), [runtime protocol](../refactor/runtime-protocol.md), [persistence and lifecycle](../refactor/persistence-and-lifecycle.md), [packaging and operations](../refactor/packaging-and-operations.md), [domain and privacy](../refactor/domain-and-privacy.md), and [verification and acceptance](../refactor/verification-and-acceptance.md).
- The implementation branch must create the logical targets listed in Section 4; until then, paths are intentionally marked `to-create` and no existing function or package signature may be assumed.

## 4. Đầu ra cần bàn giao

- `host/diagnostics/health.go` (to-create): aggregate health state, component freshness snapshot, legal transitions, invariant checks, and publishable health payload projection.
- `host/diagnostics/events.go` (to-create): versioned structured transition/event record, bounded field validation, event IDs, monotonic timing, and safe correlation metadata.
- `host/diagnostics/errors.go` (to-create): typed error-class to user-safe status, retryability, retry policy, and recommended-action mapping for all IP-07 error literals plus lifecycle/storage/index conditions.
- `host/diagnostics/redaction.go` (to-create): allowlist serializer and deny-by-default handling for arbitrary context; tests prove sensitive values cannot reach persisted records or exports.
- `host/diagnostics/repair.go` (to-create): bounded retry, resync, index-rebuild, and persistence-reopen orchestration with cancellation, one-in-flight guards, deadlines, and explicit terminal outcomes.
- `host/diagnostics/retention.go` (to-create): age/byte accounting, oldest-first eviction, startup compaction, insertion-size bound, and retention metrics. SQLite table/migration changes remain in IP-08's storage-owned paths.
- `fixtures/diagnostics/` (to-create): canonical JSON/JSONL fixtures named in Section 6, each with input sequence and expected observable health/events/statuses.
- `tests/diagnostics/` (to-create): focused Go package tests and extension/contract integration tests for transitions, error mapping, redaction, repair bounds, freshness, and retention. The extension UI test remains a consumer test owned by IP-13.
- A handoff note in the implementation change description containing the error/status table, state transition table, retention constants, and evidence commands. No new documentation file is required by this phase.

## 5. Skill và tài liệu áp dụng

- Skill tags:
  - `common/engineering/documentation` — the health contract, status wording, redaction rules, and exact verification path must be readable and executable by the next implementation agent.
  - `common/foundation/task-planning` — this phase crosses host lifecycle, protocol, SQLite, projection, and query dependencies, so each boundary, failure path, and fixture must be ordered before implementation.
  - `common/engineering/testing` — health transitions, retryability, privacy redaction, and 7-day/10 MB boundaries are observable contracts requiring success, invalid, and failure fixtures.
  - `common/engineering/git-workflow` — the phase is delivered as one focused documentation commit and must preserve the assigned path boundary.
  - `common/engineering/debugging` — runtime failures need reproducible transition evidence, root-cause classification, and confirmation that repair reaches a known state rather than hiding errors.
  - `common/security/security-review` — diagnostics cross browser, extension, Native Messaging, host, and SQLite trust boundaries; redaction, correlation IDs, and repair inputs need concrete abuse-case review.
  - `common/delivery/change-review` — final review must check contract alignment, privacy leakage, retry loops, retention enforcement, and the evidence needed for a release handoff.
- Tài liệu trong `docs/`:
  - [Product refactor README](../refactor/README.md) — binding boundary and source-of-truth rule.
  - [Requirements](../refactor/requirements.md) — FR-012, FR-013, NFR-006, NFR-009, and operational diagnosability/log policy.
  - [Target architecture](../refactor/architecture.md) — ownership, lifecycle states, and failure isolation.
  - [Native Messaging runtime protocol](../refactor/runtime-protocol.md) — envelope, health messages, typed errors, limits, and fail-closed behavior.
  - [Persistence and lifecycle](../refactor/persistence-and-lifecycle.md) — diagnostics retention, degraded persistence, recovery, reset, and uninstall boundary.
  - [Packaging and operations](../refactor/packaging-and-operations.md) — repair statuses, diagnostics fields, redaction, and local bounded logs.
  - [Domain, data, privacy, and security](../refactor/domain-and-privacy.md) — trust boundaries, redaction invariants, and profile/private separation.
  - [Verification and acceptance](../refactor/verification-and-acceptance.md) — runtime failure, privacy, recovery, and release evidence.
- Quy ước code, ADR, context ngoài `docs/`:
  - Keep the extension as the browser authority and the Go Native Messaging host as the local health/diagnostics authority; do not add an application-owned listener.
  - Treat projection revision and indexed revision as opaque bounded integers/identifiers for freshness comparison; never use diagnostic state as a source of truth for live tabs.
  - Use the project vocabulary and constraints in [`CONTEXT.md`](../../../CONTEXT.md), especially bounded local data, deterministic behavior, recoverable failure, and profile isolation.

## 6. Công việc triển khai

- [ ] Việc 1 — Define the health state machine in `host/diagnostics/health.go` (to-create):
  - Record legal transitions and reason codes for `disconnected → connecting → handshaking → synchronizing → ready`.
  - Record recoverable paths `connecting/handshaking/synchronizing/ready → recovering`, `ready → degraded` for persistence-only failure, and `recovering → synchronizing` after reconnect or `recovering → disconnected` after bounded attempts.
  - Record terminal/repairable paths to `incompatible` for protocol or host-version mismatch, `stopping` for intentional shutdown, and back to `ready` only after handshake, current snapshot, and index readiness are all confirmed.
  - Reject impossible transitions without changing the public state; emit a bounded internal diagnostic with `INTERNAL_FAILURE` and a repairable outcome rather than silently accepting contradictory state.
  - Keep component status independent: host `available/unavailable`, protocol `compatible/mismatch`, projection `current/unknown/gapped`, index `empty/building/ready/stale/failed`, and storage `healthy/degraded/quarantined/upgrade_required`. Overall health must expose the worst actionable condition while preserving lexical availability when storage is degraded.

- [ ] Việc 2 — Define freshness and health payload semantics at the IP-07 boundary:
  - Include protocol/ranking/schema versions, profile scope marker, session status, last successful handshake time, current projection revision, indexed projection revision, last sequence, index build status, storage migration status, diagnostic age, and bounded counts/durations.
  - Represent freshness with explicit `current`, `stale`, `unknown`, or `rebuilding` states and a numeric age/duration; do not infer freshness from a missing field.
  - Permit query/activation only when IP-12/IP-14's revision guards pass. A rebuilding or unknown revision returns a user-safe refresh/retry status, never a silently stale result.
  - Ensure `PERSISTENCE_DEGRADED` is visible in health while lexical query and confirmed activation continue with safe defaults where their contracts permit.
  - Emit a transition before exposing the new health snapshot and atomically update the in-memory snapshot plus event sequence so a reader cannot observe a new state with old freshness metadata.

- [ ] Việc 3 — Implement the structured diagnostic event contract in `host/diagnostics/events.go` (to-create):
  - Use a versioned event schema with `event_id`, `event_time`, monotonic `duration_ms`, `component`, `from_state`, `to_state`, `reason_code`, `error_class`, `retryable`, `recommended_action`, protocol/ranking/schema versions, projection/index revisions, counts, and bounded timing counters.
  - Use a per-session opaque correlation ID and request ID only where the protocol already supplies one; do not persist raw request payloads, browser IDs, or field values.
  - Define reason codes for host start/exit, handshake success/failure, protocol rejection, snapshot accepted, delta gap, index build start/finish/failure, query timeout, storage migration/degradation/recovery, repair start/finish/exhaustion, reset, and retention eviction.
  - Ensure every state transition has exactly one transition event even when persistence is unavailable; in-memory counters and a safe `diagnostics_persistence_degraded` marker cover the path until storage recovers.

- [ ] Việc 4 — Map error classes to user-safe status and retryability in `host/diagnostics/errors.go` (to-create):
  - Preserve these protocol classes exactly: `INVALID_FRAME`, `PROTOCOL_MISMATCH`, `PROFILE_MISMATCH`, `PAYLOAD_LIMIT`, `SNAPSHOT_REQUIRED`, `REVISION_MISMATCH`, `INDEX_REBUILDING`, `QUERY_TIMEOUT`, `PERSISTENCE_DEGRADED`, `HOST_SHUTDOWN`, and `INTERNAL_FAILURE`.
  - Use the following contract table; implementation may add internal reason detail only if it remains redacted and does not change the user-safe code:

    | Error class | User-safe status | Retryability and next action |
    | --- | --- | --- |
    | `INVALID_FRAME` | `runtime_protocol_error` | Not retryable with the same frame; fail closed, record the bounded class, reconnect only through a fresh handshake. |
    | `PROTOCOL_MISMATCH` | `runtime_incompatible` | Not automatically retryable; repair/update host or extension, then rerun compatibility check. |
    | `PROFILE_MISMATCH` | `profile_refresh_required` | Retryable only after discarding the session and sending the current profile snapshot; never query or activate another profile. |
    | `PAYLOAD_LIMIT` | `input_too_large` | Not retryable with the same input; user may shorten query or the producer may send a bounded snapshot. |
    | `SNAPSHOT_REQUIRED` | `projection_refresh_required` | Retryable once after a full snapshot; stale deltas/results are not applied. |
    | `REVISION_MISMATCH` | `results_refresh_required` | Retryable after obtaining the current projection revision; do not activate the old reference. |
    | `INDEX_REBUILDING` | `index_rebuilding` | Retryable after the bounded rebuild completes; show freshness and keep the surface usable for wait/retry/dismiss. |
    | `QUERY_TIMEOUT` | `query_timed_out` | Retryable once with the same bounded query after cancellation; then surface retry/diagnostics rather than looping. |
    | `PERSISTENCE_DEGRADED` | `persistence_degraded` | Retryable in the background with bounded reopen/repair; lexical search remains available and the UI explains the reduced recency continuity. |
    | `HOST_SHUTDOWN` | `runtime_restarting` | Retryable through bounded reconnect and fresh snapshot; no browser state is changed. |
    | `INTERNAL_FAILURE` | `runtime_recoverable_error` | Retryable once through the owning recovery path; if it repeats, stop automatic retries and offer bounded repair/diagnostics. |

  - Map local lifecycle conditions such as host missing, permission-limited context, database quarantine, and unknown future schema to statuses that state what failed, whether browser state changed, and the next safe action.
  - Make retryability a first-class boolean plus an explicit retry budget/cooldown, not a UI inference from error text.

- [ ] Việc 5 — Implement redaction and privacy-safe structured logging in `host/diagnostics/redaction.go` (to-create):
  - Serialize only an allowlist of enums, counts, durations, versions, revisions, sizes, bounded status codes, error classes, and retention counters. Unknown fields are dropped rather than recursively serialized.
  - Reject or replace values that look like full URLs, URL query/fragment material, titles, query text, page content, cookie/session material, secrets, arbitrary browser labels, or raw tab identifiers. Use a stable non-reversible local digest only when correlation is required and document its scope.
  - Ensure typed errors, health payloads, transition events, SQLite diagnostic rows, and user exports all pass the same redaction boundary. Error messages must never interpolate raw field values.
  - Add a negative test that injects representative sensitive strings into every available context slot and asserts none occurs in serialized diagnostics, exports, or UI-safe status payloads.
  - Include private-context and profile-isolation checks: diagnostics identify only the active opaque profile scope and never retain private records after the private context ends.

- [ ] Việc 6 — Add bounded repair orchestration in `host/diagnostics/repair.go` (to-create):
  - `retry_connection`: at most three attempts for one incident with bounded backoff (100 ms, 500 ms, 1,000 ms), a five-second total reconnect budget, and one in-flight repair per session.
  - `resync_projection`: request one full snapshot when a sequence/revision gap is detected; cancel an older request when a newer profile revision is known; respect IP-07 payload/count limits and report `SNAPSHOT_REQUIRED`/`REVISION_MISMATCH` without applying partial state.
  - `rebuild_index`: permit one cancellable rebuild from the authoritative in-memory snapshot; publish `index_rebuilding`, then `ready` only for the same projection revision, or `recovering`/`INTERNAL_FAILURE` on deadline or cancellation.
  - `repair_persistence`: invoke IP-08's bounded reopen/migration/quarantine path with a ten-second operation deadline. Do not delete data, alter browser state, or retry a confirmed unknown-future schema automatically.
  - Stop automatic retries after budget exhaustion, emit one `repair_exhausted` event, expose a user-triggered repair action, and apply a cooldown so repeated clicks cannot create concurrent work.
  - Return an observable result with operation, attempt count, elapsed duration, final health, retryability, and recommended action; never report repair success before the owning component confirms it.

- [ ] Việc 7 — Enforce diagnostics retention in `host/diagnostics/retention.go` (to-create) through the IP-08 repository seam:
  - Validate event schema and serialized UTF-8 payload size before insertion; cap one record at a fixed implementation limit (for example 64 KiB) and summarize/drop oversized records without storing their source content.
  - On each insert and startup, delete records whose event time is older than seven UTC days, then evict oldest records until aggregate serialized diagnostic payload bytes are at most `10_000_000` (10 MB). Both age and byte eviction must be idempotent and transactionally observable.
  - Maintain bounded counters for inserted, rejected, age-evicted, byte-evicted, and persistence-failed records; counters themselves are subject to the same retention policy and redaction.
  - If the database is unavailable, keep an in-memory bounded diagnostic ring for the current session, expose `PERSISTENCE_DEGRADED`, and flush only records that still satisfy age/byte policy after recovery.
  - Prove that a single oversized or malformed event cannot make the store exceed the byte budget and that a failed cleanup cannot trigger an unbounded retry loop.

- [ ] Việc 8 — Create the fixture and test seams:
  - `fixtures/diagnostics/health-transitions.jsonl` (to-create): startup, host absence, handshake, snapshot/index readiness, delta gap, reconnect, and clean shutdown; expected event order and final component freshness are explicit.
  - `fixtures/diagnostics/error-status-matrix.json` (to-create): each IP-07 class plus host missing, permission-limited, storage quarantine, and schema upgrade-required; expected safe status, retryable value, and action are exact.
  - `fixtures/diagnostics/redaction-sensitive-fields.json` (to-create): title, full URL with query/fragment, query text, token-like value, page content, browser ID, and private-context data; expected serialized output contains none of them.
  - `fixtures/diagnostics/repair-bounds.json` (to-create): connection failures, dropped-event resync, rebuild cancellation, persistence reopen timeout, repeated user repair, and expected attempt/deadline/cooldown counters.
  - `fixtures/diagnostics/retention-boundaries.json` (to-create): exactly seven-day-old, one-second-inside-window, byte-budget-minus-one, byte-budget-exact, and byte-budget-plus-one records; expected oldest-first eviction and final byte total `<= 10_000_000`.
  - `tests/diagnostics/` (to-create): assert state-transition legality, freshness/revision invariants, complete error mapping, redaction, repair budgets, storage-degraded lexical availability, age/byte retention, and private/profile isolation through observable outputs rather than internal field copies.

## 7. Kế hoạch commit

1. `feat(diagnostics): define health and freshness contract`
   - Thay đổi: Add the planned `host/diagnostics/` state model, component freshness fields, transition reason codes, and the IP-07 health payload adapter. Keep protocol framing and SQLite schema ownership in IP-07/IP-08.
   - Cách kiểm tra: From repository root, run `go test ./host/diagnostics/... -run 'TestHealth(Transitions|Freshness|Invariants)$'` with `fixtures/diagnostics/health-transitions.jsonl`; assert the expected final state and projection/index revisions.
2. `feat(diagnostics): add error mapping and bounded repair`
   - Thay đổi: Add the complete error-class/user-safe-status table, explicit retry budgets, reconnect/resync/rebuild/persistence repair orchestration, cancellation, cooldown, and terminal outcomes.
   - Cách kiểm tra: Run `go test ./host/diagnostics/... -run 'Test(ErrorStatusMatrix|RepairBounds|RepairCancellation)$'` with `fixtures/diagnostics/error-status-matrix.json` and `fixtures/diagnostics/repair-bounds.json`; assert no attempt exceeds its count or deadline and no browser mutation is requested.
3. `feat(diagnostics): enforce redaction and retention`
   - Thay đổi: Add allowlist serialization, sensitive-value rejection, bounded event insertion, seven-day/10 MB eviction, startup compaction, in-memory degraded ring, and IP-08 repository integration.
   - Cách kiểm tra: Run `go test ./host/diagnostics/... -run 'Test(Redaction|RetentionBoundaries|PersistenceDegraded)$'` with `fixtures/diagnostics/redaction-sensitive-fields.json` and `fixtures/diagnostics/retention-boundaries.json`; assert sensitive values are absent and final retained bytes are `<= 10_000_000`.
4. `test(diagnostics): cover runtime observability acceptance`
   - Thay đổi: Add focused regression and contract tests for every transition/error/repair/retention boundary and the extension health payload consumer seam.
   - Cách kiểm tra: Run `go test ./host/... -run 'TestDiagnostics'` and `node --test tests/diagnostics/*.test.mjs` from repository root with `INFOBOARD_DIAGNOSTIC_FIXTURES=fixtures/diagnostics`; inspect the emitted status, retryability, and redaction assertions.

## 8. Kiểm chứng và nghiệm thu

- [ ] Chạy lệnh exact với working directory, fixture và env rõ ràng:
  - `cd <repository-root> && go test ./host/diagnostics/... -run 'TestHealth(Transitions|Freshness|Invariants)$'` using `fixtures/diagnostics/health-transitions.jsonl`.
  - `cd <repository-root> && go test ./host/diagnostics/... -run 'Test(ErrorStatusMatrix|RepairBounds|RepairCancellation)$'` using `fixtures/diagnostics/error-status-matrix.json` and `fixtures/diagnostics/repair-bounds.json`.
  - `cd <repository-root> && go test ./host/diagnostics/... -run 'Test(Redaction|RetentionBoundaries|PersistenceDegraded)$'` using `fixtures/diagnostics/redaction-sensitive-fields.json` and `fixtures/diagnostics/retention-boundaries.json`.
  - `cd <repository-root> && INFOBOARD_DIAGNOSTIC_FIXTURES=fixtures/diagnostics node --test tests/diagnostics/*.test.mjs` for the extension/health payload seam when that planned test target exists.
- [ ] Kiểm tra thủ công hoặc end-to-end input → observable output:
  - Start without a registered/available host: the surface receives `runtime_unavailable`, `retryable=true`, and a bounded retry/repair action; no title or URL appears in diagnostics.
  - Complete hello → snapshot → index build: health transitions reach `ready` only when indexed revision equals the accepted projection revision, and the health payload reports host/protocol/projection/index freshness.
  - Drop a delta or force a stale query revision: the host emits `SNAPSHOT_REQUIRED` or `REVISION_MISMATCH`, marks freshness stale/rebuilding, and refuses silently stale results.
  - Force SQLite failure while the projection is current: status is `persistence_degraded`, lexical query and confirmed activation remain available, and repair attempts are bounded.
  - Inject sensitive field values and export diagnostics: output contains status codes, counts, durations, versions, and error classes only; raw title, URL, query, token, page content, and browser-field data are absent.
  - Insert records at the age and byte boundaries: records older than seven days and oldest records beyond `10_000_000` serialized bytes are removed; the final retained payload is within both limits.
- [ ] Đạt FR-012: runtime availability and index freshness are visible and actionable, including host-down and stale-index states.
- [ ] Đạt NFR-006: normal host reconnect/rebuild reaches a healthy state without reinstall, with a diagnosable transition sequence and no silent data loss.
- [ ] Đạt supporting FR-013: optional SQLite failure does not remove normal lexical search or confirmed activation; degraded persistence is visible.
- [ ] Đạt supporting NFR-009 and operational privacy/log requirements: no excluded data source is requested, collected, persisted, or transmitted; every runtime transition is diagnosable without raw title/URL; logs default to redacted metadata.
- [ ] Retryability is explicit in every health/error response, repair attempts are bounded by count/deadline/cooldown, and repeated failure reaches a user-actionable terminal state.
- [ ] Retention is enforced on insertion and startup at 7 days or `10_000_000` bytes, whichever is stricter; cleanup and cleanup failure are observable.
- [ ] Không còn lỗi đã biết thuộc phạm vi phase: no transition loses its reason/error class, no health payload reports `ready` for an unknown or rebuilding revision, no diagnostic record bypasses redaction, and no repair path loops indefinitely.

## 9. Rủi ro và quyết định còn mở

- Rủi ro: Clock changes can make persisted age checks appear newer or older than reality.
  - Phương án xử lý đã chọn: store validated UTC event time for retention and monotonic durations for in-session latency; clamp invalid/future timestamps to a safe diagnostic outcome and never extend retention because of a clock jump.
- Rủi ro: Host, projection, index, and storage can report contradictory states during reconnect or cancellation.
  - Phương án xử lý đã chọn: update component snapshot and transition event atomically; require revision equality and protocol readiness before aggregate `ready`; treat disagreement as `recovering` or `degraded`, not healthy.
- Rủi ro: A diagnostics failure can hide the failure it is meant to explain.
  - Phương án xử lý đã chọn: keep a bounded in-memory session ring and counters, emit a persistence-degraded marker, and avoid recursive logging of diagnostics failures.
- Rủi ro: Raw browser metadata can leak through free-form error strings, nested context, or exports.
  - Phương án xử lý đã chọn: allowlist serialization at the last boundary, bounded enum/error fields, negative sensitive-input fixtures, and no raw values in default or repair output.
- Rủi ro: Automatic reconnect/resync can become a CPU or process storm.
  - Phương án xử lý đã chọn: three reconnect attempts within five seconds, one in-flight operation per kind, ten-second persistence repair deadline, cancellation on newer revision, cooldown after exhaustion, and explicit user repair.
- Rủi ro: Byte accounting can diverge from SQLite page overhead.
  - Phương án xử lý đã chọn: define the product retention budget over serialized diagnostic payload bytes, enforce per-record limits before insertion, measure and expose the accounted total, and keep physical database compaction under IP-08 without claiming it changes the logical policy.
- Câu hỏi còn mở chỉ khi câu trả lời có thể thay đổi contract: none. IP-08 may choose the SQLite row/index layout, and IP-13 may choose presentation wording, but neither may change health states, retryability, redaction, or retention acceptance without updating this phase and the canonical refactor contract.

## 10. References ngoài `docs/`

- Skill: [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md)
- Skill: [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md)
- Skill: [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md)
- Skill: [`git-workflow`](../../../.agent/skills/common/engineering/git-workflow/SKILL.md)
- Skill: [`debugging`](../../../.agent/skills/common/engineering/debugging/SKILL.md)
- Skill: [`security-review`](../../../.agent/skills/common/security/security-review/SKILL.md)
- Skill: [`change-review`](../../../.agent/skills/common/delivery/change-review/SKILL.md)
- Project context: [`CONTEXT.md`](../../../CONTEXT.md)
- Source/config/test path ngoài `docs/`: `host/diagnostics/` (to-create), `tests/diagnostics/` (to-create), `fixtures/diagnostics/` (to-create); extension health payload consumer remains the IP-13-owned `extension/` (to-create) boundary.
- Fixture/tool/artifact ngoài `docs/`: `fixtures/diagnostics/health-transitions.jsonl`, `fixtures/diagnostics/error-status-matrix.json`, `fixtures/diagnostics/redaction-sensitive-fields.json`, `fixtures/diagnostics/repair-bounds.json`, and `fixtures/diagnostics/retention-boundaries.json` (all to-create).
