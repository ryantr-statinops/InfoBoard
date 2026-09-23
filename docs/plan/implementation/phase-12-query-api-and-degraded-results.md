# Phase 12 — Query API and degraded results

> Plan ID: IP-12
> Status: See README.md execution tracker
> Execution owner: Go query-orchestration and extension query-state owner
> Dependencies: IP-07, IP-09, IP-10, IP-11
> Parallel boundary: IP-13 consumes the query/result and status contract; IP-14 consumes revision-bound result references. No shared implementation paths with those phases.
> Requirement IDs: FR-003, FR-004, FR-013, NFR-002 (primary); FR-005, FR-012 (supporting evidence)
> Owned paths: `host/query/` (to-create), `extension/src/runtime/query-client.ts` (to-create), `fixtures/query/phase-12/` (to-create), `tests/query/phase-12/` (to-create)

## 1. Mục tiêu

- Define the bounded query request, response, error, and status contract between the focused extension search surface and the Go Native Messaging host. A request is bound to one opaque `profile_id` and the client-known `projection_revision`; a successful result is never detached from those identities.
- Make FR-003 and FR-004 executable: every accepted input revision searches the current profile's eligible open-tab projection by title, URL, domain, window, group, and state labels, and returns an ordered result revision that the extension can render without an explicit submit action.
- Make FR-013 and FR-012 observable: lexical results continue when optional SQLite persistence is degraded, while host-down, rebuilding, unknown-revision, timeout, and validation states expose bounded status and a safe next action to the search surface.
- Preserve IP-11's deterministic ranking output and IP-10's normalized/indexed representation. This phase orchestrates those contracts; it does not introduce a second normalizer, ranker, projection authority, or persistence source of truth.
- Define a reproducible query-to-render benchmark for NFR-002 at 1,000 indexed tabs and a 64-character query, including a p95 budget of 50 ms and an observable pass/fail record rather than an unmeasured performance claim.

## 2. Phạm vi

- Bao gồm:
  - A host-side query boundary under `host/query/` (to-create) that validates the Native Messaging envelope, profile/revision binding, query bounds, result limit, current-context fields, cancellation/deadline, and index state before invoking IP-10/IP-11.
  - The `query` request contract: required `protocol`, `type`, `request_id`, `profile_id`, and `projection_revision` envelope fields plus a bounded payload containing user query text, requested result limit, and optional current-window/current-group context. Raw user text is UTF-8 input to the IP-10 contract; it is normalized once, locally, with no second query interpretation in this phase.
  - The `query_result` success contract with request identity, profile/revision binding, query sequence/revision, bounded result references and display metadata, ranking model version, accepted limit, result status, index freshness, persistence health, and timing counters. Result rows are sourced only from IP-11 and contain no unbounded raw metadata.
  - Bounded result policy: an omitted limit uses the profile's configured safe default; a non-positive, non-integer, or over-negotiated limit is rejected before indexing/allocation with a safe `PAYLOAD_LIMIT` outcome. The effective limit is never greater than the negotiated host maximum, configured maximum, or caller request. `no_results` is a successful bounded response with an empty list, not an exception.
  - Query states: `empty_query` (bounded recent/context list, or zero results for an empty projection), `results`, `no_results`, and `invalid_query`. Invalid UTF-8/control characters, malformed phrase delimiters, and overlong input produce a visible bounded validation state or protocol validation error according to IP-07/IP-10; they do not mutate the index or execute external work.
  - Projection/index gates: an exact current published revision and `Ready` index may be queried; an unknown/mismatched revision returns `REVISION_MISMATCH` with no result rows; a known revision whose index is staging/rebuilding returns `INDEX_REBUILDING` with no result rows. The host MUST NOT silently query an older revision.
  - Runtime and persistence degradation: host transport loss, shutdown, or unavailable/recovering session maps to a bounded unavailable status with retry/repair guidance and no fabricated results. SQLite open/read/write/migration failure maps to `persistence_state: degraded` while the in-memory lexical index remains queryable; session defaults and session-only recency are used when durable metadata is unavailable.
  - Extension query client behavior: assign a monotonic local query sequence, send the latest current projection revision, cancel or ignore superseded requests, expose host/index/persistence status, and render only a response matching the current profile, projection revision, and input sequence. The client must keep stale responses from replacing newer visible results.
  - Fixed behavioral fixtures for successful field matches, empty/no-result, overlong/malformed input, result-limit boundaries, unknown/rebuilding revisions, host down/recovery, persistence degradation, deterministic result handoff, and the 1,000-tab/64-character query-to-render benchmark.
- Ngoài phạm vi:
  - Browser API calls, tab activation, stale-tab validation, window focus, and activation acknowledgement; IP-14 owns those browser-owned actions. This phase returns revision-bound result references only.
  - Snapshot/delta ordering, projection reconciliation, reconnect orchestration, index build/rebuild mechanics, Unicode/tokenization policy, and ranking weights; IP-09, IP-10, and IP-11 own those contracts and this phase consumes their published seams.
  - Focused UI layout, keyboard navigation, accessibility presentation, reduced motion, text scaling, and final loading/no-result/error rendering; IP-13 consumes this API and owns the surface.
  - SQLite schema, migrations, retention, reset, uninstall, or durable recency writes; IP-08 owns storage. Query code MUST NOT make SQLite a prerequisite for live lexical search.
  - Network access, page evaluation/content reads, browser-history sources, cross-profile lookup, cloud services, or a second transport. Query text and result metadata remain local to the current profile.
  - Changing primary requirement ownership: FR-003, FR-004, FR-013, and NFR-002 are owned here; FR-005 and FR-012 are supported with evidence while their ranking/health policy remains owned by IP-11/IP-16 respectively.

## 3. Điều kiện tiên quyết

- IP-07 has defined the versioned Native Messaging envelope, request identity, negotiated limits, query message type, typed errors, bounded timeouts, and fail-closed behavior. This phase uses `INVALID_FRAME`, `PROFILE_MISMATCH`, `PAYLOAD_LIMIT`, `REVISION_MISMATCH`, `INDEX_REBUILDING`, `QUERY_TIMEOUT`, `PERSISTENCE_DEGRADED`, `HOST_SHUTDOWN`, and `INTERNAL_FAILURE` without renaming their wire literals.
- IP-09 has published one authoritative profile/revision/index state, the `snapshot -> index -> ready` barrier, recovery statuses, and the rule that unknown or rebuilding revisions yield no rows. Query must not invent a fallback revision or read a partially rebuilt index.
- IP-10 has defined query normalization, field/token bounds, empty-query representation, malformed-input outcomes, and the immutable index lookup seam. IP-11 has defined deterministic ordering, result references, score/provenance bounds, explanations, and ranking model version.
- IP-02/IP-05 provide opaque profile and tab/window/group identities and monotonic projection semantics. IP-08 provides the persistence health and session-default fallback contract; SQLite is not live-tab authority.
- Read the canonical [`runtime-protocol.md`](../refactor/runtime-protocol.md), [`requirements.md`](../refactor/requirements.md), [`architecture.md`](../refactor/architecture.md), [`search-and-ranking.md`](../refactor/search-and-ranking.md), [`user-experience.md`](../refactor/user-experience.md), [`persistence-and-lifecycle.md`](../refactor/persistence-and-lifecycle.md), and [`verification-and-acceptance.md`](../refactor/verification-and-acceptance.md) contracts before implementation.
- No implementation source tree is observed for these boundaries. Keep every path in this plan marked `to-create`; if a scaffold exists before implementation, reread it and replace only the affected target with its exact path.
- The benchmark harness MUST have a stable clock/measurement boundary: capture input dispatch, host query completion, response decode, and result-list render completion under one fixture run. Warm-up, iteration count, environment, and percentile calculation must be recorded with the result, not inferred from a single sample.

## 4. Đầu ra cần bàn giao

- [ ] `host/query/` (to-create) with an explicit query orchestrator boundary:
  - `QueryRequest` validation for envelope identity, profile, revision, UTF-8/query bounds, context, limit, deadline, and cancellation before index work.
  - `QueryResponse`/`QueryError` serialization for bounded success, no-result, empty, invalid, unavailable, revision, rebuilding, timeout, and persistence-degraded outcomes.
  - A state gate that reads only the currently published index snapshot; it rejects unknown/rebuilding revisions before invoking ranking and returns no result rows on rejection.
  - A local execution path that invokes IP-10 query preparation and IP-11 ranking, caps rows to the accepted limit, propagates ranking model/provenance/explanation fields, and reports bounded query timing without raw title/URL/query logging.
  - A persistence-health adapter that annotates successful lexical results as `healthy` or `degraded` and never blocks the hot path on optional storage.
- [ ] `extension/src/runtime/query-client.ts` (to-create) with a profile-scoped client that sends query revisions, tracks latest input/revision, ignores superseded responses, maps typed errors to observable statuses, and exposes a render-safe result model to IP-13. It must distinguish `no_results` from unavailable, rebuilding, invalid, and stale states.
- [ ] `fixtures/query/phase-12/` (to-create) with machine-readable fixtures and exact expected observable outcomes:
  - `QUERY-001-field-match.json`: title, URL, domain, window, group, and state-label queries return expected profile-bound result references and IP-11 ordering.
  - `QUERY-002-live-input-revisions.json`: successive input revisions return matching query sequence/revision and a superseded response cannot replace the latest result.
  - `QUERY-003-empty-and-no-results.json`: empty/whitespace input returns a bounded context list; an empty projection and unmatched query return bounded zero-row `no_results` responses.
  - `QUERY-004-invalid-and-overlong.json`: invalid control/phrase input and query above the IP-10 scalar/token bound return visible validation outcomes without index mutation; overlong payload returns `PAYLOAD_LIMIT`.
  - `QUERY-005-result-limit.json`: omitted/default, minimum, configured, negotiated-maximum, zero, negative, and above-maximum limits prove deterministic cap/rejection behavior.
  - `QUERY-006-unknown-revision.json`: a request for an unknown profile revision returns `REVISION_MISMATCH`, zero rows, an observable refresh/resync action, and no older-revision fallback.
  - `QUERY-007-rebuilding-revision.json`: a known revision while index rebuild is in progress returns `INDEX_REBUILDING`, zero rows, freshness status, and retry guidance.
  - `QUERY-008-host-down-recovery.json`: transport close/host shutdown exposes unavailable/recovering state, never claims results, then a fresh ready revision becomes queryable after reconnect.
  - `QUERY-009-persistence-degraded.json`: forced SQLite failure still returns deterministic lexical rows and activation references from memory, marks persistence degraded, and uses session/default recency without blocking the query.
  - `QUERY-010-deterministic-handoff.json`: repeated identical request/projection/ranking timestamp returns byte-equivalent bounded result IDs, scores, explanations, model version, status, and revision metadata.
  - `QUERY-011-adversarial-bounds.json`: oversized result metadata, many tokens, malformed envelope fields, duplicate request IDs, deadline expiry, and cancellation fail closed with bounded allocations and safe errors.
  - `QUERY-012-query-to-render-1000-tabs-64-chars.json`: exactly 1,000 eligible records, a 64-character query, configured result limit, fixed ranking timestamp, and expected rendered result/status shape for NFR-002.
- [ ] `tests/query/phase-12/` (to-create) covering contract validation, profile/revision binding, input transitions, result-limit boundaries, empty/no-result behavior, unknown/rebuilding rejection, host recovery, persistence degradation, deterministic output, cancellation/deadline, redacted diagnostics, and response/status bounds.
- [ ] A benchmark command under `tests/query/phase-12/cmd/query-to-render-benchmark/` (to-create) that consumes `QUERY-012`, performs 20 warm-up and 200 measured iterations, reports p50/p95/p99 for query-to-render plus allocation/result-count summaries, and exits non-zero when p95 exceeds 50 ms or result/status expectations differ.

## 5. Skill và tài liệu áp dụng

- Skill tags:
  - [`api-design`](../../../.agent/skills/personal/engineering/backend/api-design/SKILL.md) — define versioned request/response envelopes, error semantics, idempotency, bounded limits, and compatibility between extension and host.
  - [`go`](../../../.agent/skills/personal/engineering/backend/go/SKILL.md) — implement typed host contracts, cancellation/deadlines, bounded allocations, deterministic serialization, and local hot-path behavior.
  - [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md) — build fixture-driven behavioral coverage for normal, boundary, malformed, stale, degraded, and performance cases.
  - [`secure-development`](../../../.agent/skills/common/security/secure-development/SKILL.md) — enforce fail-closed input validation, profile isolation, privacy-safe errors/logs, payload limits, and no external execution from query text.
  - [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md) — record standalone contracts, ownership boundaries, commands, fixtures, and acceptance evidence.
  - [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md) — order contract, host, client, fixture, integration, and benchmark slices with explicit handoffs.
  - [`git-workflow`](../../../.agent/skills/common/engineering/git-workflow/SKILL.md) — keep future implementation commits narrow, reviewable, and independently verifiable.
- Tài liệu trong `docs/`:
  - [`docs/plan/refactor/README.md`](../refactor/README.md) — binding product boundary and source-of-truth rule.
  - [`docs/plan/refactor/runtime-protocol.md`](../refactor/runtime-protocol.md) — envelope, query message, limits, revision guard, typed errors, and status safety.
  - [`docs/plan/refactor/requirements.md`](../refactor/requirements.md) — FR-003, FR-004, FR-005, FR-012, FR-013, and NFR-002 acceptance signals.
  - [`docs/plan/refactor/architecture.md`](../refactor/architecture.md) — extension/host ownership, local query flow, lifecycle, and failure isolation.
  - [`docs/plan/refactor/search-and-ranking.md`](../refactor/search-and-ranking.md) — normalized fields, match classes, deterministic score/order, empty/malformed behavior, and explanations.
  - [`docs/plan/refactor/user-experience.md`](../refactor/user-experience.md) — input revisions, no-result/rebuilding/unavailable states, and trust rules for rendering.
  - [`docs/plan/refactor/persistence-and-lifecycle.md`](../refactor/persistence-and-lifecycle.md) — SQLite degradation, session defaults, and bounded local data.
  - [`docs/plan/refactor/verification-and-acceptance.md`](../refactor/verification-and-acceptance.md) — fixed fixtures, 1,000-tab p50/p95/p99 evidence, redaction, and end-to-end journeys.

## 6. Công việc triển khai

- [ ] `IP-12-T01` **Contract and limits:** write the versioned `query`/`query_result`/typed-error schemas, enumerate required/optional fields, enforce `profile_id` and `projection_revision` binding, define negotiated/configured result-limit precedence, and reject malformed/oversized input before index allocation.
- [ ] `IP-12-T02` **Host orchestration:** implement the readiness gate (`Ready` plus exact published revision only), call IP-10 preparation and IP-11 ranking, cap and serialize bounded rows, include model/revision/status/timing metadata, and preserve deterministic ordering and explanations.
- [ ] `IP-12-T03` **Validation and empty paths:** distinguish valid empty query, empty projection, no result, invalid query, overlong payload, cancellation, and deadline expiration. Every path must leave the index unchanged and return a bounded observable status.
- [ ] `IP-12-T04` **Revision safety:** prove unknown revision and rebuilding revision return no result rows, no older snapshot fallback, typed error code, current known revision/freshness metadata where safe, and a retry/resync action for the client.
- [ ] `IP-12-T05` **Degraded runtime:** map transport unavailable/shutdown/recovering to no-result unavailable state; map optional SQLite failure to successful lexical result with `persistence_state=degraded`, safe defaults/session-only recency, and a non-blocking diagnostic status.
- [ ] `IP-12-T06` **Extension query client:** increment the local input sequence on each accepted change, send the latest projection revision, discard out-of-order/superseded responses, and expose a render model whose status cannot imply confirmed results or activation when the host did not return them.
- [ ] `IP-12-T07` **Fixture and test seams:** implement `QUERY-001` through `QUERY-012` and focused tests against observable wire/client behavior, including profile mismatch, duplicate request identity, bounded error payloads, deterministic repeated runs, and redacted diagnostics.
- [ ] `IP-12-T08` **Performance proof:** implement the exact benchmark command and fixture, capture environment and warm-up/iteration settings, report p50/p95/p99 and allocation counts, and fail the command when the NFR-002 p95 budget or expected status/result shape is violated.

## 7. Kế hoạch commit

1. `feat(query): implement ip-12-t01`
   - Task IDs: `IP-12-T01`.
   - Owned target paths: `host/query/` (to-create), `extension/src/runtime/query-client.ts` (to-create), `fixtures/query/phase-12/` (to-create), `tests/query/phase-12/` (to-create).
   - Behavior: **Contract and limits:** write the versioned `query`/`query_result`/typed-error schemas, enumerate required/optional fields, enforce `profile_id` and `projection_revision` binding, define negotiated/configured result-limit precedence, and reject malformed/oversized input before index allocation.
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/query/... ./tests/query/phase-12/... -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Contract and limits:** write the versioned `query`/`query_result`/typed-error schemas, enumerate required/optional fields, enforce `profile_id` and `projection_revision` binding, define negotiated/configured result-limit precedence, and reject malformed/oversized input before index allocation.
   - Dependency gate: all index.md dependencies for IP-12 have merged to dev; phase work branch starts from latest origin/dev.

2. `feat(query): implement ip-12-t02`
   - Task IDs: `IP-12-T02`.
   - Owned target paths: `host/query/` (to-create), `extension/src/runtime/query-client.ts` (to-create), `fixtures/query/phase-12/` (to-create), `tests/query/phase-12/` (to-create).
   - Behavior: **Host orchestration:** implement the readiness gate (`Ready` plus exact published revision only), call IP-10 preparation and IP-11 ranking, cap and serialize bounded rows, include model/revision/status/timing metadata, and preserve deterministic ordering and explanations.
   - Fixture and command: IP-10, IP-11; run `go test ./host/query/... ./tests/query/phase-12/... -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Host orchestration:** implement the readiness gate (`Ready` plus exact published revision only), call IP-10 preparation and IP-11 ranking, cap and serialize bounded rows, include model/revision/status/timing metadata, and preserve deterministic ordering and explanations.
   - Dependency gate: all index.md dependencies for IP-12 have merged to dev; phase work branch starts from latest origin/dev.

3. `feat(query): implement ip-12-t03`
   - Task IDs: `IP-12-T03`.
   - Owned target paths: `host/query/` (to-create), `extension/src/runtime/query-client.ts` (to-create), `fixtures/query/phase-12/` (to-create), `tests/query/phase-12/` (to-create).
   - Behavior: **Validation and empty paths:** distinguish valid empty query, empty projection, no result, invalid query, overlong payload, cancellation, and deadline expiration. Every path must leave the index unchanged and return a bounded observable status.
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/query/... ./tests/query/phase-12/... -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Validation and empty paths:** distinguish valid empty query, empty projection, no result, invalid query, overlong payload, cancellation, and deadline expiration. Every path must leave the index unchanged and return a bounded observable status.
   - Dependency gate: all index.md dependencies for IP-12 have merged to dev; phase work branch starts from latest origin/dev.

4. `feat(query): implement ip-12-t04`
   - Task IDs: `IP-12-T04`.
   - Owned target paths: `host/query/` (to-create), `extension/src/runtime/query-client.ts` (to-create), `fixtures/query/phase-12/` (to-create), `tests/query/phase-12/` (to-create).
   - Behavior: **Revision safety:** prove unknown revision and rebuilding revision return no result rows, no older snapshot fallback, typed error code, current known revision/freshness metadata where safe, and a retry/resync action for the client.
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/query/... ./tests/query/phase-12/... -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Revision safety:** prove unknown revision and rebuilding revision return no result rows, no older snapshot fallback, typed error code, current known revision/freshness metadata where safe, and a retry/resync action for the client.
   - Dependency gate: all index.md dependencies for IP-12 have merged to dev; phase work branch starts from latest origin/dev.

5. `feat(query): implement ip-12-t05`
   - Task IDs: `IP-12-T05`.
   - Owned target paths: `host/query/` (to-create), `extension/src/runtime/query-client.ts` (to-create), `fixtures/query/phase-12/` (to-create), `tests/query/phase-12/` (to-create).
   - Behavior: **Degraded runtime:** map transport unavailable/shutdown/recovering to no-result unavailable state; map optional SQLite failure to successful lexical result with `persistence_state=degraded`, safe defaults/session-only recency, and a non-blocking diagnostic status.
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/query/... ./tests/query/phase-12/... -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Degraded runtime:** map transport unavailable/shutdown/recovering to no-result unavailable state; map optional SQLite failure to successful lexical result with `persistence_state=degraded`, safe defaults/session-only recency, and a non-blocking diagnostic status.
   - Dependency gate: all index.md dependencies for IP-12 have merged to dev; phase work branch starts from latest origin/dev.

6. `test(query): implement ip-12-t06`
   - Task IDs: `IP-12-T06`.
   - Owned target paths: `host/query/` (to-create), `extension/src/runtime/query-client.ts` (to-create), `fixtures/query/phase-12/` (to-create), `tests/query/phase-12/` (to-create).
   - Behavior: **Extension query client:** increment the local input sequence on each accepted change, send the latest projection revision, discard out-of-order/superseded responses, and expose a render model whose status cannot imply confirmed results or activation when the host did not return them.
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/query/... ./tests/query/phase-12/... -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Extension query client:** increment the local input sequence on each accepted change, send the latest projection revision, discard out-of-order/superseded responses, and expose a render model whose status cannot imply confirmed results or activation when the host did not return them.
   - Dependency gate: all index.md dependencies for IP-12 have merged to dev; phase work branch starts from latest origin/dev.

7. `test(query): implement ip-12-t07`
   - Task IDs: `IP-12-T07`.
   - Owned target paths: `host/query/` (to-create), `extension/src/runtime/query-client.ts` (to-create), `fixtures/query/phase-12/` (to-create), `tests/query/phase-12/` (to-create).
   - Behavior: **Fixture and test seams:** implement `QUERY-001` through `QUERY-012` and focused tests against observable wire/client behavior, including profile mismatch, duplicate request identity, bounded error payloads, deterministic repeated runs, and redacted diagnostics.
   - Fixture and command: QUERY-001, QUERY-012; run `go test ./host/query/... ./tests/query/phase-12/... -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Fixture and test seams:** implement `QUERY-001` through `QUERY-012` and focused tests against observable wire/client behavior, including profile mismatch, duplicate request identity, bounded error payloads, deterministic repeated runs, and redacted diagnostics.
   - Dependency gate: all index.md dependencies for IP-12 have merged to dev; phase work branch starts from latest origin/dev.

8. `test(query): implement ip-12-t08`
   - Task IDs: `IP-12-T08`.
   - Owned target paths: `host/query/` (to-create), `extension/src/runtime/query-client.ts` (to-create), `fixtures/query/phase-12/` (to-create), `tests/query/phase-12/` (to-create).
   - Behavior: **Performance proof:** implement the exact benchmark command and fixture, capture environment and warm-up/iteration settings, report p50/p95/p99 and allocation counts, and fail the command when the NFR-002 p95 budget or expected status/result shape is violated.
   - Fixture and command: NFR-002; run `go test ./host/query/... ./tests/query/phase-12/... -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Performance proof:** implement the exact benchmark command and fixture, capture environment and warm-up/iteration settings, report p50/p95/p99 and allocation counts, and fail the command when the NFR-002 p95 budget or expected status/result shape is violated.
   - Dependency gate: all index.md dependencies for IP-12 have merged to dev; phase work branch starts from latest origin/dev.

## 8. Kiểm chứng và nghiệm thu

- [ ] From repository root, run `go test ./host/query/... ./tests/query/phase-12/... -count=1` after the planned Go targets exist; fixture tests MUST cover every `QUERY-001` through `QUERY-011` outcome and compare observable status/result payloads rather than implementation details.
- [ ] From repository root, run `GOMAXPROCS=1 go run ./tests/query/phase-12/cmd/query-to-render-benchmark --fixture fixtures/query/phase-12/QUERY-012-query-to-render-1000-tabs-64-chars.json --warmup 20 --iterations 200 --p95-budget-ms 50`; record p50/p95/p99, host/query/render timings, result count, environment, and exit status in the acceptance artifact.
- [ ] `QUERY-001` proves FR-003 field coverage and profile-scoped results; `QUERY-002` proves FR-004 live update and stale-response suppression; `QUERY-010` supports FR-005 deterministic result ordering and model metadata.
- [ ] `QUERY-006` and `QUERY-007` prove that unknown or rebuilding revisions never return misleading older rows; the client exposes refresh/retry/freshness status and no activation reference from a rejected response.
- [ ] `QUERY-008` proves host-down/unavailable/recovering status is observable, bounded, actionable, and never claims a result; a fresh synchronized revision becomes queryable after recovery.
- [ ] `QUERY-009` proves FR-013: forced optional persistence failure leaves lexical rows and browser-owned activation references available, marks persistence degraded, and does not block on SQLite.
- [ ] All response payloads enforce negotiated field/result limits, redact raw query/title/URL values from errors and diagnostics by default, and fail closed on malformed envelope, profile mismatch, duplicate request, cancellation, timeout, and oversized input.
- [ ] Acceptance is complete only when NFR-002's 1,000-tab/64-character benchmark reports p95 <= 50 ms for query-to-render and every status/result shape remains within the bounded contract; no known in-scope query, revision, host, or persistence failure remains unobservable.

## 9. Rủi ro và quyết định còn mở

- **Rủi ro:** A result response may arrive after a newer input or projection revision and overwrite the visible list. **Phương án xử lý đã chọn:** bind every request/response to local query sequence, opaque profile, and projection revision; the client discards superseded responses and the host rejects unknown/rebuilding revisions.
- **Rủi ro:** Treating SQLite as required for query execution could turn a durable-storage fault into total search failure. **Phương án xử lý đã chọn:** query reads the published in-memory index only; persistence health is an annotation/status, with session defaults and session-only recency when degraded.
- **Rủi ro:** Unbounded query/result fields or diagnostics can cause allocation, latency, or privacy regressions. **Phương án xử lý đã chọn:** validate envelope and scalar/token/result bounds before index work, serialize only bounded metadata, and redact raw fields from errors/diagnostics.
- **Rủi ro:** A benchmark that measures only host ranking can miss extension decode/render latency. **Phương án xử lý đã chọn:** `QUERY-012` and its harness measure dispatch through rendered result completion, with separate host/query/decode/render counters.
- **Câu hỏi còn mở chỉ khi câu trả lời có thể thay đổi contract:** None. Negotiated numeric limits and profile-configured defaults are supplied by IP-07/IP-08; this phase must consume rather than redefine them.

## 10. References ngoài `docs/`

- Skill: [`api-design`](../../../.agent/skills/personal/engineering/backend/api-design/SKILL.md)
- Skill: [`go`](../../../.agent/skills/personal/engineering/backend/go/SKILL.md)
- Skill: [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md)
- Skill: [`secure-development`](../../../.agent/skills/common/security/secure-development/SKILL.md)
- Skill: [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md)
- Skill: [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md)
- Skill: [`git-workflow`](../../../.agent/skills/common/engineering/git-workflow/SKILL.md)
- Project context: [`CONTEXT.md`](../../../CONTEXT.md)
- Source/config/test path ngoài `docs/`: `host/query/` (to-create), `extension/src/runtime/query-client.ts` (to-create), `tests/query/phase-12/` (to-create)
- Fixture/tool/artifact ngoài `docs/`: `fixtures/query/phase-12/QUERY-001` through `QUERY-012` (to-create); benchmark command `tests/query/phase-12/cmd/query-to-render-benchmark/` (to-create)
