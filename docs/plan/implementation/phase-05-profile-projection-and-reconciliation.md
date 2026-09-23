# Phase 05 — Profile projection and reconciliation

> Plan ID: IP-05
> Status: not_started
> Execution owner: profile-projection and reconciliation owner
> Dependencies: IP-02, IP-04, IP-06, IP-07
> Parallel boundary: IP-09 consumes this phase's reconciliation contract; IP-10 consumes its committed projection handoff. No shared implementation files with IP-04, IP-06, IP-07, or IP-08.
> Requirement IDs: FR-003, FR-009, FR-010, FR-014, NFR-004, NFR-005 (supporting evidence; primary ownership remains IP-12, IP-04, IP-09, IP-17, IP-11 respectively)
> Owned paths: `extension/src/projection/reconciler.ts` (to-create), `host/internal/projection/state.go` (to-create), `host/internal/projection/reconciler.go` (to-create), `fixtures/projection/phase-05/` (to-create), `tests/projection/phase-05/` (to-create)

## 1. Mục tiêu

- Xây dựng projection sống cho **browser profile hiện tại** từ full eligible-tab snapshot do extension thu thập và các event delta đã được IP-04 chuẩn hóa. Browser snapshot là authority; host chỉ mirror, reconcile và bàn giao một projection đã commit cho index.
- Định nghĩa revision/epoch và state machine có thể kiểm chứng: revision `0` là uninitialized, snapshot authority đầu tiên trong một epoch là revision `1`, mỗi effective event được chấp nhận tăng revision đúng một lần, còn duplicate/stale/conflict không được làm thay đổi state ngoài acknowledgement hoặc resync signal đã quy định.
- Làm cho delta create/update/move/group/pin/activate/remove có thứ tự, idempotent và profile-scoped. Event bị trễ, đảo thứ tự, mất event, duplicate khác nội dung hoặc vượt qua epoch hiện tại phải bị từ chối an toàn thay vì tạo phantom hoặc duplicate record.
- Khôi phục được sau service-worker/host restart và reconnect: projection cũ không được coi là live truth, full snapshot mới phải được nhận và thay thế atomically trước khi session trở lại `Ready`.
- Bảo đảm sau mỗi full snapshot thành công, tập identity được commit và tập eligible browser IDs của profile/context tương ứng bằng nhau; output canonical không phụ thuộc vào thứ tự mảng đầu vào, phục vụ kết quả deterministic của các phase index/ranking sau.
- Giữ profile isolation và private-context partition ở mọi bước; không để projection, event, result reference hoặc callback của profile/session này mutates projection của profile/session khác.

## 2. Phạm vi

- Bao gồm:
  - Extension-side projection producer/reducer tại `extension/src/projection/`: nhận full snapshot và normalized events từ IP-04, cấp phát `projection_epoch`/`projection_revision` theo IP-02, dựng delta có thứ tự, và phát handoff bounded cho Native Messaging client của IP-07/IP-09.
  - Host-side in-memory reconciler tại `host/internal/projection/`: validate profile/context/epoch/revision, apply ordered deltas, maintain one map keyed by `TabIdentity`, publish committed state, and expose `SnapshotReady`, `Ready`, `ResyncRequired`, `Stale`, and profile-mismatch outcomes to the session/index seams.
  - Full snapshot contract containing one known profile, context partition, epoch, revision, bounded eligible records, and a deterministic digest/canonical order for duplicate detection and evidence. Missing required identity, duplicate identity, invalid profile, or an incomplete authoritative read is not an empty snapshot.
  - Delta contract using the protocol fields supplied by IP-07 plus the normalized event sequence/base-revision data from IP-04. A valid delta applies only to the current profile/context/epoch and expected predecessor; a batch is staged and committed as a unit so a failed member cannot expose partial state.
  - Monotonic revision rules: within an epoch, accepted effective transitions advance one bounded unsigned revision at a time; a matching duplicate is acknowledged without another advance; an older unknown event, gap, conflict, or out-of-order event leaves state unchanged and requests a full snapshot. A new restart/reconnect/resync lineage receives a fresh epoch and starts its authoritative snapshot at revision `1`.
  - Atomic replacement: validate and stage every snapshot record, enforce uniqueness and bounds, build the new map in isolation, then swap one committed state pointer/version. Readers and the index handoff observe either the prior complete map or the new complete map, never a partially loaded snapshot.
  - Profile and context boundaries: `profile_id` is mandatory on every state, record, event and handoff; normal and private contexts remain separate partitions; a private partition is memory-only and is not included in a normal-profile snapshot or durable handoff.
  - Recovery signals and bounded metadata for IP-06/IP-07/IP-09: `resync_required` reason, current epoch/revision/count, expected predecessor, accepted/stale/duplicate counters, and timing. Diagnostics never include raw titles, URLs, queries, tokens, or page data.
  - Fixture-driven extension/host tests proving snapshot authority, ordered deltas, stale and out-of-order fencing, event-loss recovery, restart convergence, profile isolation, atomic replacement, duplicate suppression, and deterministic canonical output.
- Ngoài phạm vi:
  - Chrome/Edge API listener signatures, eligibility decisions, private-window discovery, and browser event normalization; IP-04 owns those inputs. This phase consumes its normalized records and independent full-snapshot result rather than adding another browser adapter.
  - Native Messaging framing, hello negotiation, exact envelope serialization, typed wire literals, process bootstrap, and reconnect transport; IP-06/IP-07 own those boundaries. This phase defines the state handoff they call, not a second transport.
  - End-to-end extension/host reconnect orchestration, index rebuild scheduling, query availability, and user-facing degraded states; IP-09/IP-10/IP-12 consume the statuses and committed-state callback defined here.
  - Unicode/search normalization, score calculation, explanations, UI rendering, browser activation, activation recency, configuration, reset, uninstall, schema/migration, or durable writes.
  - Any new browser data source, network path, page read, or cross-profile lookup. Projection data is limited to the current eligible open-tab metadata contract.

## 3. Điều kiện tiên quyết

- IP-02 is merged and fixes `ProfileID`, `ContextKind`, `TabIdentity`, `ProjectionEpoch`, `ProjectionRevision`, `EligibleTabRecord`, `ProjectionState`, lifecycle states, and identity-based tie-break rules. This phase must consume those types rather than redefine profile or tab identity.
- IP-04 is merged and provides normalized create/update/move/group/pin/activate/window/remove events, eligibility outcomes, event sequence/recovery markers, private-context disposal, and an independent full eligible-tab snapshot oracle. Required browser capability denial or an incomplete snapshot must be distinguishable from a successful empty result.
- IP-06 is merged and provides one Native Messaging session lifecycle with `Starting`/`Synchronizing`/`Ready`/`Recovering` boundaries, cancellation, bounded resource budgets, and fresh-session ownership. A new session must not inherit a previous host map without an authoritative snapshot.
- IP-07 is merged and provides the `snapshot`, `delta`, `sync_ack`, and `resync_required` message contract, `profile_id`/`projection_revision` envelope fields, payload limits, request correlation, and `PROFILE_MISMATCH`, `SNAPSHOT_REQUIRED`, `REVISION_MISMATCH`, and `PAYLOAD_LIMIT` error mapping.
- Read the binding [`product boundary`](../refactor/README.md), [`target architecture`](../refactor/architecture.md), [`domain and privacy contract`](../refactor/domain-and-privacy.md), [`runtime protocol`](../refactor/runtime-protocol.md), [`persistence and lifecycle`](../refactor/persistence-and-lifecycle.md), [`requirements`](../refactor/requirements.md), and [`verification and acceptance`](../refactor/verification-and-acceptance.md).
- Read [`CONTEXT.md`](../../../CONTEXT.md) and the IP-01 fixture/limit conventions. Implementation roots are absent at authoring time; every path in this phase is a logical `to-create` target and must be replaced with an exact observed path if the source tree exists before implementation begins.
- The fixture harness must control profile IDs, contexts, epochs, revisions, event sequence, dropped/repeated/reordered events, interrupted snapshot staging, restart/session boundaries, and independent browser-visible eligible-ID oracles without reading a real browser profile.

## 4. Đầu ra cần bàn giao

- `extension/src/projection/reconciler.ts` (to-create):
  - A profile/context-partitioned projection owner that accepts the IP-04 snapshot/event handoff and exposes a bounded snapshot/delta stream to the IP-07 client.
  - Epoch/revision allocation and lifecycle transitions for startup, worker restart, host reconnect, resync, private-context disposal, and shutdown. A new lineage invalidates pending deltas and old references.
  - Canonical snapshot construction: validate required fields, reject incomplete authority, sort by immutable identity for digest/evidence, and emit one atomic snapshot payload without leaking raw diagnostic fields.
  - Ordered event reduction: coalesce only exact no-op duplicates, retain every effective field change, and mark uncertainty instead of guessing when sequence/revision continuity is not provable.
- `host/internal/projection/state.go` (to-create):
  - Profile/context/epoch/revision state value and committed map keyed by `TabIdentity`, with explicit lifecycle/status and bounded counters.
  - Read-only committed-state snapshot and index-handoff view containing the exact accepted identity set, projection revision, epoch, and profile; no API to mutate browser state or durable storage.
- `host/internal/projection/reconciler.go` (to-create):
  - Full snapshot validation and atomic replacement, including profile/session binding, record bounds, duplicate detection, canonical digest, private partition rules, and commit notification.
  - Ordered delta application with predecessor checks, matching-duplicate idempotency, stale/out-of-order/conflict outcomes, remove semantics, and an all-or-nothing staged batch path.
  - Fresh-epoch recovery boundary for reconnect/restart/resync; old epoch events and old profile events cannot mutate the new state. The reconciler must refuse query/index-ready handoff while `ResyncRequired` or rebuilding input is unresolved.
- `fixtures/projection/phase-05/` (to-create): stable inputs, expected state/status/revision outputs, exact fixture IDs, and bounded diagnostic records:
  - `FX-PROJECTION-SNAPSHOT-AUTHORITY` — accept snapshot `A`, deliver deltas that would produce `B`, then accept a newer authoritative snapshot containing `B` and `C`; the committed identity set and fields equal the snapshot exactly, including removal of records absent from it.
  - `FX-PROJECTION-ORDERED-DELTAS` — apply create, title/URL, move/group, pin/active, and remove events in sequence; each effective event advances one revision and the final identity set has no duplicate.
  - `FX-PROJECTION-DUPLICATE-DELTA` — replay identical bytes/content and then replay the same event key with changed content; the first replay is a no-op acknowledgement, the conflict leaves state unchanged and emits a bounded resync/revision outcome.
  - `FX-PROJECTION-STALE-REVISION` — submit an old epoch, lower revision, already-consumed sequence, and wrong predecessor; none mutates state or produces an index-ready handoff, and each has the expected stale or resync status.
  - `FX-PROJECTION-MISSED-EVENT` — drop one event from an ordered stream, observe `ResyncRequired`, then deliver an authoritative full snapshot; the result converges to the independent browser-visible eligible-ID oracle with no stale record.
  - `FX-PROJECTION-RESTART` — discard the old session/worker state, create a new epoch, and submit a full snapshot; old records, old deltas, and old references cannot reappear, and the new session reaches snapshot-ready only after the replacement commits.
  - `FX-PROJECTION-PROFILE-BOUNDARY` — run two profiles with equal numeric tab/window/group IDs and different records; each profile has an independent map/revision/epoch and cross-profile input is rejected without mutation.
  - `FX-PROJECTION-ATOMIC-REPLACE` — interrupt or fail validation while staging a snapshot; readers continue seeing the previous complete committed state, then a valid retry swaps the complete new set once with no intermediate duplicate or empty state.
  - `FX-PROJECTION-CANONICAL-OUTPUT` — permute the same valid snapshot/event input and repeat it at the same timestamp; canonical identity order, digest, revisions, and accepted-state output are byte-equivalent.
- `tests/projection/phase-05/` (to-create): fixture runner, extension reducer tests, host reconciler tests, deterministic clock/sequence helpers, atomic-read observer, and profile/session isolation assertions. Tests must assert observable state, IDs, status, revision, epoch, and handoff output rather than map implementation or private method names.
- A handoff note consumed by IP-09/IP-10/IP-12 defining the commit notification (`profile_id`, `context_kind`, `projection_epoch`, `projection_revision`, eligible identity set/count, readiness/status), the conditions for `resync_required`, and the rule that a stale or uncertain state is not queryable or activatable.

## 5. Skill và tài liệu áp dụng

- Skill tags:
  - [`pipelines`](../../../.agent/skills/personal/engineering/data/pipelines/SKILL.md) — define snapshot/delta inputs, transformation stages, lineage, checkpoints, idempotency, and restart-safe convergence rather than treating event delivery as implicitly reliable.
  - [`databases`](../../../.agent/skills/personal/engineering/data/databases/SKILL.md) — apply consistency, bounded access patterns, atomic replacement, lifecycle, and inspectable state principles to the in-memory projection without turning durable storage into live-tab authority.
  - [`go`](../../../.agent/skills/personal/engineering/backend/go/SKILL.md) — specify explicit host packages, value/state boundaries, cancellation-safe handoffs, bounded allocation, and observable Go test seams for the reconciler.
  - [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md) — cover success, duplicate, stale, gap, conflict, restart, profile, atomicity, and deterministic-output behavior through fixture-observable results.
  - [`debugging`](../../../.agent/skills/common/engineering/debugging/SKILL.md) — make divergence, revision gaps, stale epochs, interrupted staging, and cross-profile contamination reproducible and diagnosable with safe counters rather than raw tab content.
  - [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md) — publish the standalone projection contract, handoff boundaries, fixture inputs, exact commands, risks, and acceptance evidence.
  - [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md) — sequence snapshot authority, delta reducer, fencing, atomic commit, recovery, fixtures, and handoffs so implementation dependencies are explicit.
  - [`git-workflow`](../../../.agent/skills/common/engineering/git-workflow/SKILL.md) — keep the phase plan and future implementation slices reviewable, limited to owned paths, and committed with the required message.
- Tài liệu trong `docs/`:
  - [`refactor/README.md`](../refactor/README.md) — binding product boundary and source-of-truth rule.
  - [`refactor/architecture.md`](../refactor/architecture.md) — extension browser authority, host mirror/index ownership, snapshot/delta data flow, and lifecycle isolation.
  - [`refactor/domain-and-privacy.md`](../refactor/domain-and-privacy.md) — profile/context identity, eligible record fields, projection lifecycle, privacy, and trust boundaries.
  - [`refactor/runtime-protocol.md`](../refactor/runtime-protocol.md) — snapshot authority, ordered deltas, revision fencing, resync messages, limits, and fail-closed error behavior.
  - [`refactor/persistence-and-lifecycle.md`](../refactor/persistence-and-lifecycle.md) — memory-only live projection, restart/rebuild boundary, and persistence failure isolation.
  - [`refactor/requirements.md`](../refactor/requirements.md) — FR-003, FR-009, FR-010, FR-014, NFR-004, NFR-005 and operational timeout/profile rules.
  - [`refactor/verification-and-acceptance.md`](../refactor/verification-and-acceptance.md) — event, restart, profile-separation, snapshot-replacement, convergence, and deterministic-output evidence.
- Quy ước code, ADR, context ngoài `docs/`: [`CONTEXT.md`](../../../CONTEXT.md); reuse IP-02 domain records, IP-04 normalized event fields, IP-06 lifecycle hooks, and IP-07 wire/status contracts instead of introducing parallel identities, transport, or error vocabularies.

## 6. Công việc triển khai

- [ ] **Define the projection boundary:** map IP-02 `ProjectionState`/`EligibleTabRecord` and IP-04 normalized event fields into the extension producer and host reconciler packages. Make `profile_id`, `context_kind`, `projection_epoch`, and `projection_revision` mandatory partition/fence fields; reject synthetic identity fallbacks.
- [ ] **Acquire and validate authoritative snapshots:** accept a snapshot as authority only when the browser call completed for the known current profile/context and every record passes IP-02 bounds/identity checks. Distinguish an intentional successful empty snapshot from permission failure, partial read, or unavailable profile state.
- [ ] **Canonicalize snapshot input:** sort by immutable `TabIdentity`, compute a bounded digest for duplicate/conflict comparison, reject duplicate identities and cross-profile records, preserve explicit absent optional fields, and ensure equivalent input order produces one canonical output.
- [ ] **Allocate epochs and revisions:** initialize at revision `0`; assign revision `1` to the first authoritative snapshot in a new epoch; increment one time per accepted effective event; never compare or reuse revisions across a new epoch. Start a fresh epoch for worker restart, host reconnect, or a full-resync lineage so stale in-flight work is fenced.
- [ ] **Implement ordered delta reduction:** require current profile/context/epoch and the expected predecessor revision/sequence; stage field updates by `TabIdentity`; retain effective title, URL, window, group, pinned, active, eligibility, and remove transitions; commit each valid effective event exactly once. An all-or-nothing batch must not expose earlier operations if a later operation is invalid.
- [ ] **Handle duplicates, stale events, and gaps:** acknowledge an exact already-consumed event without advancing revision; classify lower/old epoch, wrong predecessor, missing sequence, changed duplicate, unknown tab update, and invalid remove as stale/conflict/gap; preserve the last committed map and emit `resync_required` where continuity is uncertain.
- [ ] **Replace atomically:** build and validate a candidate map off to the side, publish one immutable committed state only after every record passes, and notify the index handoff once. During staging failure or cancellation, readers keep the previous complete state; no partial map, duplicate identity, or transient empty map is observable.
- [ ] **Recover after restart and reconnect:** clear or quarantine old-session state, bind the new session to the expected profile, require the new-epoch full snapshot before `Ready`, discard old private records and pending deltas, and expose `SnapshotRequired`/`Recovering` rather than serving silently stale state.
- [ ] **Enforce profile/context isolation:** keep separate state partitions and counters for profiles and normal/private contexts, reject mismatched envelope/record/event profile IDs before mutation, and prove equal browser IDs in different profiles cannot collide in the host map or handoff.
- [ ] **Publish safe handoff metadata:** send only committed epoch/revision/count/identity references, status, digest/version, durations, and bounded error classes to IP-07/IP-09/IP-10/IP-16. Never put raw title, URL, query, token, or page values in divergence diagnostics.
- [ ] **Add fixture-driven tests:** implement all phase-05 fixtures and assert exact identity sets, record fields, lifecycle/status, revision/epoch transitions, profile boundaries, duplicate behavior, atomic visibility, and canonical output. Run both extension-side and host-side consumers against the same fixture schemas.
- [ ] **Exercise the recovery sequence:** from the repository root, drive snapshot → ordered delta → dropped event → `resync_required` → new authoritative snapshot → index handoff, then repeat through a fresh session/epoch. The final committed IDs must equal the eligible browser oracle and contain no duplicate record.

## 7. Kế hoạch commit

1. `feat(projection): reconcile authoritative profile snapshots`
   - Thay đổi: add the extension snapshot producer and host projection state/reconciler seams, profile/context validation, canonical snapshot digest, and atomic full-snapshot replacement under the owned paths.
   - Cách kiểm tra: run `python3 tests/projection/phase-05/run_fixtures.py --suite phase-05 --fixtures fixtures/projection/phase-05 --only FX-PROJECTION-SNAPSHOT-AUTHORITY,FX-PROJECTION-ATOMIC-REPLACE,FX-PROJECTION-PROFILE-BOUNDARY --strict`; assert one complete committed state and no cross-profile mutation.
2. `feat(projection): apply ordered deltas and revision fences`
   - Thay đổi: add effective-event revision allocation, sequence/predecessor checks, exact duplicate acknowledgement, stale/conflict/gap handling, remove semantics, and resync outcomes.
   - Cách kiểm tra: run `go test ./host/... ./tests/projection/phase-05 -run 'TestProjection_(Delta|Revision|Duplicate|Stale|Gap)' -count=1` with `FX-PROJECTION-ORDERED-DELTAS`, `FX-PROJECTION-DUPLICATE-DELTA`, and `FX-PROJECTION-STALE-REVISION`.
3. `feat(projection): recover profile state across restart`
   - Thay đổi: add fresh-epoch/session fencing, missed-event snapshot recovery, private/context partition handoff, and committed-state/index readiness callbacks for IP-09/IP-10.
   - Cách kiểm tra: run `FX-PROJECTION-MISSED-EVENT` and `FX-PROJECTION-RESTART`; verify old epoch work is rejected, new snapshot is required before ready, and final IDs equal the browser oracle.
4. `test(projection): prove deterministic convergence and isolation`
   - Thay đổi: add the full phase fixture corpus, deterministic permutation/replay tests, atomic-read observer, and extension/host parity checks.
   - Cách kiểm tra: run `node --test tests/projection/phase-05/*.test.mjs`, `go test ./host/... ./tests/projection/phase-05 -count=1`, and the strict Python fixture runner; require zero duplicate identities and byte-equivalent canonical outputs for repeated inputs.
5. `docs(implementation): add phase 05 profile projection and reconciliation`
   - Thay đổi: publish this standalone phase plan only; implementation commits above are future work and must not be mixed into the documentation branch.
   - Cách kiểm tra: run `git diff --check`; validate exactly one metadata block, exactly ten numbered headings, resolving canonical/skill/context links, and a path-only diff containing this phase file.

## 8. Kiểm chứng và nghiệm thu

- [ ] From repository root, once the implementation targets exist, run `python3 tests/projection/phase-05/run_fixtures.py --suite phase-05 --fixtures fixtures/projection/phase-05 --strict`. The runner must load every phase-05 fixture and print bounded pass/fail records with fixture ID, final status, epoch/revision, and identity-count evidence.
- [ ] Run `go test ./host/... ./tests/projection/phase-05 -run 'TestProjection_(Snapshot|Delta|Revision|Recovery|Profile|Atomic|Canonical)' -count=1` and `node --test tests/projection/phase-05/*.test.mjs` from the repository root. The commands must use only the declared local fixture root and no external service.
- [ ] `FX-PROJECTION-SNAPSHOT-AUTHORITY` must prove that a newer complete snapshot removes every absent old record, adds every eligible new record, replaces fields as a unit, and leaves the committed IDs exactly equal to the snapshot's eligible browser IDs.
- [ ] `FX-PROJECTION-ORDERED-DELTAS` must prove create/update/move/group/pin/active/remove ordering, one revision advance per accepted effective event, no duplicate `TabIdentity`, and a committed output independent of reducer map iteration order.
- [ ] `FX-PROJECTION-DUPLICATE-DELTA` and `FX-PROJECTION-STALE-REVISION` must prove exact replay is idempotent, changed replay is rejected, old epoch/lower revision/gap/out-of-order input never mutates state, and the handoff exposes the expected stale/resync outcome.
- [ ] `FX-PROJECTION-MISSED-EVENT` must prove a dropped event does not produce a silently incomplete result: the state becomes `ResyncRequired`, no index-ready handoff is published for the uncertain revision, a full snapshot is accepted atomically, and final IDs/fields equal the independent eligible browser oracle.
- [ ] `FX-PROJECTION-RESTART` must prove that service-worker/host restart creates a fresh epoch, rejects old deltas and references, requires an authoritative snapshot before `Ready`, and converges without duplicate or orphan records.
- [ ] `FX-PROJECTION-PROFILE-BOUNDARY` must prove equal numeric browser IDs in two profiles remain separate in state, revision, epoch, digest, and handoff; a profile mismatch yields `PROFILE_MISMATCH` or the IP-07 equivalent with no mutation. This supports FR-014 without claiming its primary ownership.
- [ ] `FX-PROJECTION-ATOMIC-REPLACE` must prove that staging interruption/validation failure keeps readers on the previous complete map and a valid retry produces exactly one new committed map; no observer may see partial, duplicate, or transient empty state.
- [ ] `FX-PROJECTION-CANONICAL-OUTPUT` must prove the same projection, event stream, and timestamp produce byte-equivalent canonical identity output, revision, digest, and handoff metadata. This supplies deterministic projection input evidence for NFR-004; score/ranking ownership remains IP-11.
- [ ] Inspect the committed-state-to-index handoff for NFR-005: after every successful full snapshot, indexed candidate IDs equal browser-visible eligible IDs exactly, including removals and private-context disposal. A failed/uncertain snapshot must report recovery rather than claim convergence.
- [ ] Inspect diagnostics and runtime activity for the phase boundary: no raw title, URL, query, token, or page value is emitted; no network or unrelated browser API path is introduced; all limits and timeouts come from IP-01/IP-07 contracts.
- [ ] At authoring time the implementation roots are absent, so the commands above are future acceptance contracts rather than executed runtime tests. Current documentation proof is `git diff --check`, the exact ten-heading/metadata check, and resolution of every link in this file.

## 9. Rủi ro và quyết định còn mở

- Rủi ro: revision-only ordering cannot distinguish a new runtime from an old one after service-worker or host restart. Phương án xử lý đã chọn: fence every lineage with an opaque `projection_epoch`; restart/reconnect/full-resync creates a new epoch whose authoritative snapshot starts at revision `1`, and all old work is rejected before map mutation.
- Rủi ro: a full snapshot may be empty because the browser read failed or was incomplete, which could erase valid state. Phương án xử lý đã chọn: only a successfully completed, profile-bound, validated browser snapshot is authoritative; denial, timeout, partial fields, or unknown profile produce `ResyncRequired`/degraded status and preserve the prior committed map until a valid replacement.
- Rủi ro: a duplicate event with changed content or a missing event can make an apparently plausible map silently wrong. Phương án xử lý đã chọn: compare event identity/sequence/content, require the expected predecessor, make conflict/gap state-preserving, and require an authoritative snapshot rather than guessing.
- Rủi ro: readers or the index could observe a candidate map while it is still being validated. Phương án xử lý đã chọn: construct and validate off to the side, publish one immutable committed state, and emit one handoff only after the atomic swap; interrupted staging leaves the previous map visible.
- Rủi ro: equal browser IDs across profiles or contexts could collide in a shared host process. Phương án xử lý đã chọn: make `profile_id` and `context_kind` mandatory key partitions, bind them at session handshake, and reject mismatches before any state or counter mutation.
- Rủi ro: extension and host could disagree about whether a batch advances one or multiple revisions. Phương án xử lý đã chọn: IP-02's event-level rule is binding; a batch is staged atomically, each accepted effective event advances one revision, and the final acknowledgement reports the resulting revision/sequence. Any ambiguity yields `REVISION_MISMATCH`/resync rather than partial application.
- Rủi ro: a large snapshot could create unbounded staging memory or block readiness. Phương án xử lý đã chọn: enforce IP-01/IP-07 record, field, payload, and timeout limits before allocation/indexing; reject over-limit input with the protocol-owned bounded error and keep the previous committed state.
- Câu hỏi còn mở chỉ khi câu trả lời có thể thay đổi contract: none. Future package names may follow the observed implementation scaffold, but changing snapshot authority, epoch/revision fencing, profile isolation, or atomic visibility requires an update to the canonical refactor decision before implementation.

## 10. References ngoài `docs/`

- Skill: [`pipelines`](../../../.agent/skills/personal/engineering/data/pipelines/SKILL.md), [`databases`](../../../.agent/skills/personal/engineering/data/databases/SKILL.md), [`go`](../../../.agent/skills/personal/engineering/backend/go/SKILL.md), [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md), [`debugging`](../../../.agent/skills/common/engineering/debugging/SKILL.md), [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md), [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md), [`git-workflow`](../../../.agent/skills/common/engineering/git-workflow/SKILL.md)
- Project context: [`CONTEXT.md`](../../../CONTEXT.md)
- Canonical refactor contracts: [`README.md`](../refactor/README.md), [`architecture.md`](../refactor/architecture.md), [`domain-and-privacy.md`](../refactor/domain-and-privacy.md), [`runtime-protocol.md`](../refactor/runtime-protocol.md), [`requirements.md`](../refactor/requirements.md), [`verification-and-acceptance.md`](../refactor/verification-and-acceptance.md)
- Source/config/test path ngoài `docs/`: `extension/src/projection/reconciler.ts` (to-create), `host/internal/projection/state.go` (to-create), `host/internal/projection/reconciler.go` (to-create), `tests/projection/phase-05/` (to-create)
- Fixture/tool/artifact ngoài `docs/`: `fixtures/projection/phase-05/FX-PROJECTION-SNAPSHOT-AUTHORITY`, `FX-PROJECTION-ORDERED-DELTAS`, `FX-PROJECTION-DUPLICATE-DELTA`, `FX-PROJECTION-STALE-REVISION`, `FX-PROJECTION-MISSED-EVENT`, `FX-PROJECTION-RESTART`, `FX-PROJECTION-PROFILE-BOUNDARY`, `FX-PROJECTION-ATOMIC-REPLACE`, and `FX-PROJECTION-CANONICAL-OUTPUT` (all to-create); `tests/projection/phase-05/run_fixtures.py` (to-create); future Go/Node fixture runners (to-create)
