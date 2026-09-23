# Phase 09 — Projection sync and recovery

> Plan ID: IP-09
> Status: See README.md execution tracker
> Execution owner: extension-host synchronization and recovery owner
> Dependencies: IP-04, IP-05, IP-07, IP-08
> Parallel boundary: IP-10 consumes the published projection/index handoff and rebuild status; IP-12 consumes revision availability and degraded-status outputs. No shared implementation paths with IP-04, IP-05, IP-07, or IP-08.
> Requirement IDs: FR-010, NFR-005 (primary ownership); FR-012, FR-013, NFR-006 (supporting evidence; primary ownership remains IP-16, IP-12, and IP-16 respectively)
> Owned paths: `extension/src/runtime/projection-sync.ts` (to-create), `host/internal/sync/orchestrator.go` (to-create), `host/internal/recovery/rebuild.go` (to-create), `fixtures/recovery/phase-09/` (to-create), `tests/recovery/phase-09/` (to-create)

## 1. Mục tiêu

- Xây dựng một synchronization orchestrator có thể kiểm chứng cho đường đi `hello -> snapshot -> index -> ready` giữa extension và Go Native Messaging host. `Ready` chỉ được công bố khi handshake, projection, index và revision đều nhất quán.
- Làm cho full snapshot là authority sau service-worker restart, host reconnect, dropped event, sequence gap hoặc yêu cầu resync. Snapshot mới phải thay thế state cũ atomically và tạo indexed ID set đúng bằng eligible browser ID set của profile/context hiện tại.
- Chốt delta path có thứ tự, bounded và idempotent: chỉ acknowledge delta sau khi projection và index đã publish cùng revision; duplicate/stale/gap/conflict không được làm lùi revision hoặc lộ state một phần.
- Cung cấp recovery state quan sát được (`Disconnected`, `Connecting`, `Handshaking`, `Synchronizing`, `Rebuilding`, `Ready`, `Recovering`) để search surface biết host availability và index freshness, đồng thời từ chối kết quả khi revision unknown hoặc index đang rebuilding.
- Giữ lexical search và browser-owned activation hoạt động khi SQLite unavailable, migration/corruption recovery hoặc write failure làm persistence chuyển sang degraded. Persistence lỗi không được biến live projection hay in-memory index thành unavailable.

## 2. Phạm vi

- Bao gồm:
  - Extension synchronization controller tại `extension/src/runtime/projection-sync.ts`: mở session cho profile hiện tại, chờ `hello_ack`, gửi authoritative snapshot, phát ordered deltas, xử lý `sync_ack`/`resync_required`, nhận health/status và điều phối reconnect với bounded backoff do IP-06 cung cấp.
  - Host synchronization orchestrator tại `host/internal/sync/orchestrator.go`: bind session vào `profile_id`/context/epoch, enforce lifecycle gates, stage projection-to-index handoff, publish one ready revision, and map typed protocol errors/statuses.
  - Initial sequence: extension obtains a completed full eligible-tab snapshot from IP-04, starts Native Messaging, sends `hello`; host validates protocol/profile/capabilities and returns `hello_ack`; extension sends `snapshot`; host validates and atomically installs it, requests a detached index build from IP-10, waits for the matching index publication, returns `sync_ack`, then exposes `Ready`.
  - Atomic snapshot handoff: validate the whole snapshot before mutation, build projection/index state off to the side, associate both with one synchronization generation and `projection_revision`, publish only after every stage succeeds, and discard incomplete staging state on cancellation, disconnect, invalid input or build failure.
  - Ordered delta handoff: require the current profile/context/epoch and expected predecessor revision/sequence, apply a bounded batch in staging, publish projection and index together, and acknowledge the resulting revision/sequence only after the commit barrier. A matching duplicate is idempotently acknowledged; stale, out-of-order or conflicting input leaves committed state untouched and emits `resync_required`.
  - Resync and dropped-event recovery: stop or discard pending deltas after a gap, report the current revision and reason without raw tab fields, request a complete browser snapshot, and repeat the authoritative snapshot-to-ready sequence. A full snapshot never merges with an untrusted delta stream or overrides a newer committed generation.
  - Reconnect and full rebuild: a new Native Messaging connection starts at `Handshaking`/`Synchronizing`; it does not reuse an old session's projection as live truth. The extension obtains a fresh snapshot after handshake, the host rebuilds the projection/index, and only then returns to `Ready`. Host absence/crash, service-worker suspension and cancellation are bounded and visible as `Recovering`/`Disconnected`.
  - Persistence-degraded synchronization: accept `hello_ack` with `PERSISTENCE_DEGRADED`, use host defaults and session-only metadata, and continue snapshot, index, query and browser activation. Storage recovery may resume later without forcing a second projection rebuild; persistence health remains visible in `health_result` and diagnostics.
  - Revision guard: a query or result handoff is valid only for the currently published profile/context/generation/revision. An unknown revision returns `REVISION_MISMATCH` with no result; a known revision whose index is staging returns `INDEX_REBUILDING` with no result; the host never falls back to an older index silently.
  - Fixture-driven extension/host integration tests for initial sync, dropped events, duplicate/out-of-order deltas, interrupted snapshots, reconnect, profile mismatch, unknown/rebuilding revision refusal and SQLite degraded operation.
- Ngoài phạm vi:
  - Browser API listener signatures, event normalization, eligibility/private-context decisions and browser snapshot collection; IP-04 owns those inputs. This phase consumes its completed snapshot/event contract.
  - Tab identity, profile/context domain types, epoch allocation and canonical projection semantics; IP-02/IP-05 own those contracts. This phase fences and transports them rather than redefining them.
  - Native Messaging frame encoding, envelope schema, version negotiation literals and process stdin/stdout ownership; IP-06/IP-07 own those boundaries. This phase calls their session seam.
  - SQLite schema, migrations, corruption quarantine and reset/uninstall ownership; IP-08 owns storage behavior. This phase consumes persistence health and does not cache live projection in SQLite.
  - Unicode normalization, postings, lexical scoring, ranking explanations, query API/UI rendering or browser activation; IP-10 through IP-14 own those behaviors. The synchronization contract only exposes a committed index revision and safe status.
  - Network access, page reads, browser history, cross-profile lookup, new data sources or any second transport. Recovery is local and Native Messaging only.

## 3. Điều kiện tiên quyết

- IP-04 đã cung cấp profile-bound full eligible-tab snapshots, normalized create/update/move/group/pin/activate/window/remove events, event sequence metadata, missed-event detection and a distinction between successful empty projection and incomplete/denied browser read.
- IP-05 đã chốt `ProjectionEpoch`, `ProjectionRevision`, `ProjectionState`, canonical identity sets, atomic projection replacement, delta predecessor rules, profile/context isolation and `ResyncRequired` outcomes. This phase must use those outputs instead of inventing a second revision authority.
- IP-07 đã chốt the Native Messaging envelope and `hello`, `hello_ack`, `snapshot`, `delta`, `sync_ack`, `resync_required`, `health`, `query` and typed `error` contracts, including `SNAPSHOT_REQUIRED`, `REVISION_MISMATCH`, `INDEX_REBUILDING`, `PERSISTENCE_DEGRADED`, `PROFILE_MISMATCH`, `HOST_SHUTDOWN` and bounded payload/request limits.
- IP-08 đã chốt persistence health, defaults, session-only fallback, corruption/migration status and the rule that SQLite is not source of truth for live tabs. The orchestrator must continue lexical availability when the repository is degraded.
- IP-06's host lifecycle supplies connection ownership, cancellation, bounded reconnect/backoff and shutdown signals; this phase does not create a daemon, listener or replacement transport.
- IP-10 exposes a detached build/publish seam for a validated projection, a `Rebuilding` status, a published `profile_id`/revision and an explicit failure result. Exact package names remain `to-create` until implementation scaffolding exists.
- The canonical contracts to implement are [`runtime-protocol.md`](../refactor/runtime-protocol.md), [`architecture.md`](../refactor/architecture.md), [`requirements.md`](../refactor/requirements.md), [`persistence-and-lifecycle.md`](../refactor/persistence-and-lifecycle.md), [`domain-and-privacy.md`](../refactor/domain-and-privacy.md) and [`verification-and-acceptance.md`](../refactor/verification-and-acceptance.md).

## 4. Đầu ra cần bàn giao

- `extension/src/runtime/projection-sync.ts` (to-create): a profile-scoped synchronization state machine that gates snapshot/delta sends on `hello_ack`, tracks acknowledged epoch/revision/sequence, stops delivery on `resync_required`, and exposes bounded `Ready`, `Recovering`, `Disconnected`, `Rebuilding` and degraded-persistence status to the query/UI seam.
- `host/internal/sync/orchestrator.go` (to-create): the host-side sync coordinator that validates profile/context/generation, stages snapshot/delta commits, invokes the index build/publish seam, emits `sync_ack` only at the commit barrier, and maps gaps, cancellation, shutdown and mismatch into protocol outcomes.
- `host/internal/recovery/rebuild.go` (to-create): reconnect/resync/full-rebuild coordination, bounded retry transitions, stale-generation invalidation, and the no-stale-result gate while projection or index state is not published.
- `fixtures/recovery/phase-09/` (to-create): deterministic fixtures with IDs `SYNC-001` (hello-to-ready), `SYNC-002` (dropped-event convergence), `SYNC-003` (duplicate/stale/out-of-order delta), `SYNC-004` (host reconnect), `SYNC-005` (interrupted/atomic snapshot), `SYNC-006` (unknown or rebuilding revision), `SYNC-007` (SQLite persistence degraded), and `SYNC-008` (profile mismatch). Each fixture records input frames/events, browser-authoritative IDs, expected state transitions, final revision, typed error/status and whether results are allowed.
- `tests/recovery/phase-09/` (to-create): Go integration tests for orchestration and recovery plus extension tests for client state transitions. Test seams must observe committed IDs, epoch/revision/sequence, lifecycle status, `sync_ack`/`resync_required`, query result presence/absence and persistence health without asserting private implementation details.
- A handoff contract for IP-10/IP-12: published index metadata contains the matching profile/context and projection revision; status distinguishes `Ready`, `Rebuilding`, `Recovering`, `Disconnected` and `PERSISTENCE_DEGRADED`; no query result is emitted for an unknown or rebuilding revision.

## 5. Skill và tài liệu áp dụng

- Skill tags:
  - [`pipelines`](../../../.agent/skills/personal/engineering/data/pipelines/SKILL.md) — model snapshot, delta, staging, publication, lineage and replay as a restartable local pipeline with explicit idempotency and convergence checks.
  - [`go`](../../../.agent/skills/personal/engineering/backend/go/SKILL.md) — keep the host coordinator explicit, cancellation-aware, bounded and observable while handing state to the Go index and protocol packages.
  - [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md) — define deterministic event/frame fixtures, failure boundaries, result-absence assertions and an executable dropped-event convergence proof.
  - [`debugging`](../../../.agent/skills/common/engineering/debugging/SKILL.md) — design state-transition evidence, redacted diagnostics and recovery probes for host loss, gaps, interrupted builds and persistence degradation.
  - [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md) — make the synchronization sequence, ownership boundaries, observable statuses and exact acceptance signals executable by a later implementer.
  - [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md) — order transport handoff, snapshot publication, delta processing, recovery and test seams so each implementation slice is independently verifiable.
  - [`git-workflow`](../../../.agent/skills/common/engineering/git-workflow/SKILL.md) — keep this phase isolated to its assigned plan file and define small behavioral implementation commits.
- Tài liệu trong `docs/`:
  - [`Target architecture`](../refactor/architecture.md) — extension/host ownership, lifecycle states, snapshot-first data flow and failure isolation.
  - [`Runtime protocol`](../refactor/runtime-protocol.md) — envelope fields, handshake, synchronization message semantics, revision refusal and typed errors.
  - [`Product requirements`](../refactor/requirements.md) — FR-010, FR-012, FR-013, NFR-005 and NFR-006 acceptance outcomes.
  - [`Persistence and lifecycle`](../refactor/persistence-and-lifecycle.md) — memory-only live projection, bounded SQLite fallback, startup and reconnect behavior.
  - [`Domain and privacy`](../refactor/domain-and-privacy.md) — profile isolation, trust boundaries, bounded fields and redacted diagnostics.
  - [`Verification and acceptance`](../refactor/verification-and-acceptance.md) — dropped-event, reconnect, snapshot replacement and persistence-failure evidence.
- Quy ước code, ADR, context ngoài `docs/`: [`CONTEXT.md`](../../../CONTEXT.md) supplies the product vocabulary, local-only boundary, ownership rules and required measurable acceptance signals. No implementation source tree is present; every target in this plan is explicitly `to-create`.

## 6. Công việc triển khai

- [ ] `IP-09-T01` **Implement the sync state machine:** in `extension/src/runtime/projection-sync.ts`, represent `Disconnected -> Connecting -> Handshaking -> Synchronizing -> Rebuilding -> Ready` and `Recovering` transitions. Reject tab data before a successful `hello_ack`; preserve profile/context/epoch/revision/sequence as mandatory fences; expose bounded status without leaking title, URL or query data.
- [ ] `IP-09-T02` **Implement hello-to-ready gating:** send one `hello` with the current profile and supported capabilities, validate `hello_ack` protocol/limits/profile and persistence health, request a completed IP-04 snapshot, send `snapshot`, and wait for the host to validate and stage the matching index. Emit `sync_ack` and enter `Ready` only when projection and index publish the same revision/generation.
- [ ] `IP-09-T03` **Implement atomic snapshot publication:** in `host/internal/sync/orchestrator.go`, validate all records and counts before allocation, hand a detached snapshot to IP-05/IP-10, and publish one complete generation. On invalid, partial, canceled or failed build input, discard staging state, keep no new queryable revision, and return `SNAPSHOT_REQUIRED`, `REVISION_MISMATCH` or a bounded internal/recovery status as appropriate.
- [ ] `IP-09-T04` **Implement ordered delta processing:** require the expected predecessor and matching profile/context/epoch; stage a whole bounded batch; call the index publication seam; then acknowledge accepted revision/sequence. Treat exact duplicates as idempotent acknowledgement, and treat gaps, conflicting duplicates, stale events or out-of-order events as no-op plus `resync_required` with current revision and reason.
- [ ] `IP-09-T05` **Implement dropped-event resync:** on `resync_required`, stop sending further deltas, discard obsolete queued deltas, acquire a new full snapshot from the browser projection seam and repeat snapshot -> rebuild -> ready. Record fixture-visible counts/revisions and prove that final indexed IDs equal the browser-authoritative eligible IDs, with no duplicate records.
- [ ] `IP-09-T06` **Implement reconnect and full rebuild:** in `host/internal/recovery/rebuild.go`, invalidate session-scoped work on disconnect, prevent old callbacks/results from publishing, use IP-06 bounded reconnect/cancellation, and force a new hello plus authoritative snapshot. Host crash, service-worker suspension, interrupted snapshot and index-build failure must end in `Recovering`/`Rebuilding` or `Disconnected`, never silently reuse stale state.
- [ ] `IP-09-T07` **Implement persistence-degraded continuation:** consume IP-08 health at handshake and during runtime. If SQLite open/migration/write fails, keep defaults and session-only metadata in memory, keep a successfully published lexical index queryable and let extension activation proceed; surface `PERSISTENCE_DEGRADED` and retryable health/diagnostic data without blocking synchronization.
- [ ] `IP-09-T08` **Implement revision-safe query handoff:** before forwarding or answering a query, compare requested profile/context/generation/revision with the published ready state. Return `REVISION_MISMATCH` for unknown/old revisions and `INDEX_REBUILDING` for a revision still staging; return no result payload in either case, and never serve the last ready revision as an implicit fallback.
- [ ] `IP-09-T09` **Add fixtures and observability:** create the `SYNC-001`–`SYNC-008` fixtures and tests under the owned roots. Assert lifecycle transitions, exact typed statuses, `sync_ack`/`resync_required`, final identity sets, revision equality, persistence-degraded lexical result/activation, bounded retries and redacted diagnostic fields. Include a no-network/no-page-read test seam.

## 7. Kế hoạch commit

1. `feat(sync): implement ip-09-t01`
   - Task IDs: `IP-09-T01`.
   - Owned target paths: extension/src/runtime/projection-sync.ts.
   - Behavior: **Implement the sync state machine:** in `extension/src/runtime/projection-sync.ts`, represent `Disconnected -> Connecting -> Handshaking -> Synchronizing -> Rebuilding -> Ready` and `Recovering` transitions. Reject tab data before a successful `hello_ack`; preserve profile/context/epoch/revision/sequence as mandatory fences; expose bounded status without leaking title, URL or query data.
   - Fixture and command: the observable fixture/outcome stated by this task; run `python3 tests/recovery/phase-09/run_fixtures.py --suite phase-09 --fixtures fixtures/recovery/phase-09 --strict`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Implement the sync state machine:** in `extension/src/runtime/projection-sync.ts`, represent `Disconnected -> Connecting -> Handshaking -> Synchronizing -> Rebuilding -> Ready` and `Recovering` transitions. Reject tab data before a successful `hello_ack`; preserve profile/context/epoch/revision/sequence as mandatory fences; expose bounded status without leaking title, URL or query data.
   - Dependency gate: all index.md dependencies for IP-09 have merged to dev; phase work branch starts from latest origin/dev.

2. `feat(sync): implement ip-09-t02`
   - Task IDs: `IP-09-T02`.
   - Owned target paths: `extension/src/runtime/projection-sync.ts` (to-create), `host/internal/sync/orchestrator.go` (to-create), `host/internal/recovery/rebuild.go` (to-create), `fixtures/recovery/phase-09/` (to-create), `tests/recovery/phase-09/` (to-create).
   - Behavior: **Implement hello-to-ready gating:** send one `hello` with the current profile and supported capabilities, validate `hello_ack` protocol/limits/profile and persistence health, request a completed IP-04 snapshot, send `snapshot`, and wait for the host to validate and stage the matching index. Emit `sync_ack` and enter `Ready` only when projection and index publish the same revision/generation.
   - Fixture and command: IP-04; run `python3 tests/recovery/phase-09/run_fixtures.py --suite phase-09 --fixtures fixtures/recovery/phase-09 --strict`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Implement hello-to-ready gating:** send one `hello` with the current profile and supported capabilities, validate `hello_ack` protocol/limits/profile and persistence health, request a completed IP-04 snapshot, send `snapshot`, and wait for the host to validate and stage the matching index. Emit `sync_ack` and enter `Ready` only when projection and index publish the same revision/generation.
   - Dependency gate: all index.md dependencies for IP-09 have merged to dev; phase work branch starts from latest origin/dev.

3. `feat(sync): implement ip-09-t03`
   - Task IDs: `IP-09-T03`.
   - Owned target paths: host/internal/sync/orchestrator.go.
   - Behavior: **Implement atomic snapshot publication:** in `host/internal/sync/orchestrator.go`, validate all records and counts before allocation, hand a detached snapshot to IP-05/IP-10, and publish one complete generation. On invalid, partial, canceled or failed build input, discard staging state, keep no new queryable revision, and return `SNAPSHOT_REQUIRED`, `REVISION_MISMATCH` or a bounded internal/recovery status as appropriate.
   - Fixture and command: IP-05, IP-10; run `python3 tests/recovery/phase-09/run_fixtures.py --suite phase-09 --fixtures fixtures/recovery/phase-09 --strict`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Implement atomic snapshot publication:** in `host/internal/sync/orchestrator.go`, validate all records and counts before allocation, hand a detached snapshot to IP-05/IP-10, and publish one complete generation. On invalid, partial, canceled or failed build input, discard staging state, keep no new queryable revision, and return `SNAPSHOT_REQUIRED`, `REVISION_MISMATCH` or a bounded internal/recovery status as appropriate.
   - Dependency gate: all index.md dependencies for IP-09 have merged to dev; phase work branch starts from latest origin/dev.

4. `feat(sync): implement ip-09-t04`
   - Task IDs: `IP-09-T04`.
   - Owned target paths: `extension/src/runtime/projection-sync.ts` (to-create), `host/internal/sync/orchestrator.go` (to-create), `host/internal/recovery/rebuild.go` (to-create), `fixtures/recovery/phase-09/` (to-create), `tests/recovery/phase-09/` (to-create).
   - Behavior: **Implement ordered delta processing:** require the expected predecessor and matching profile/context/epoch; stage a whole bounded batch; call the index publication seam; then acknowledge accepted revision/sequence. Treat exact duplicates as idempotent acknowledgement, and treat gaps, conflicting duplicates, stale events or out-of-order events as no-op plus `resync_required` with current revision and reason.
   - Fixture and command: the observable fixture/outcome stated by this task; run `python3 tests/recovery/phase-09/run_fixtures.py --suite phase-09 --fixtures fixtures/recovery/phase-09 --strict`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Implement ordered delta processing:** require the expected predecessor and matching profile/context/epoch; stage a whole bounded batch; call the index publication seam; then acknowledge accepted revision/sequence. Treat exact duplicates as idempotent acknowledgement, and treat gaps, conflicting duplicates, stale events or out-of-order events as no-op plus `resync_required` with current revision and reason.
   - Dependency gate: all index.md dependencies for IP-09 have merged to dev; phase work branch starts from latest origin/dev.

5. `test(sync): implement ip-09-t05`
   - Task IDs: `IP-09-T05`.
   - Owned target paths: `extension/src/runtime/projection-sync.ts` (to-create), `host/internal/sync/orchestrator.go` (to-create), `host/internal/recovery/rebuild.go` (to-create), `fixtures/recovery/phase-09/` (to-create), `tests/recovery/phase-09/` (to-create).
   - Behavior: **Implement dropped-event resync:** on `resync_required`, stop sending further deltas, discard obsolete queued deltas, acquire a new full snapshot from the browser projection seam and repeat snapshot -> rebuild -> ready. Record fixture-visible counts/revisions and prove that final indexed IDs equal the browser-authoritative eligible IDs, with no duplicate records.
   - Fixture and command: the observable fixture/outcome stated by this task; run `python3 tests/recovery/phase-09/run_fixtures.py --suite phase-09 --fixtures fixtures/recovery/phase-09 --strict`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Implement dropped-event resync:** on `resync_required`, stop sending further deltas, discard obsolete queued deltas, acquire a new full snapshot from the browser projection seam and repeat snapshot -> rebuild -> ready. Record fixture-visible counts/revisions and prove that final indexed IDs equal the browser-authoritative eligible IDs, with no duplicate records.
   - Dependency gate: all index.md dependencies for IP-09 have merged to dev; phase work branch starts from latest origin/dev.

6. `feat(sync): implement ip-09-t06`
   - Task IDs: `IP-09-T06`.
   - Owned target paths: host/internal/recovery/rebuild.go.
   - Behavior: **Implement reconnect and full rebuild:** in `host/internal/recovery/rebuild.go`, invalidate session-scoped work on disconnect, prevent old callbacks/results from publishing, use IP-06 bounded reconnect/cancellation, and force a new hello plus authoritative snapshot. Host crash, service-worker suspension, interrupted snapshot and index-build failure must end in `Recovering`/`Rebuilding` or `Disconnected`, never silently reuse stale state.
   - Fixture and command: IP-06; run `python3 tests/recovery/phase-09/run_fixtures.py --suite phase-09 --fixtures fixtures/recovery/phase-09 --strict`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Implement reconnect and full rebuild:** in `host/internal/recovery/rebuild.go`, invalidate session-scoped work on disconnect, prevent old callbacks/results from publishing, use IP-06 bounded reconnect/cancellation, and force a new hello plus authoritative snapshot. Host crash, service-worker suspension, interrupted snapshot and index-build failure must end in `Recovering`/`Rebuilding` or `Disconnected`, never silently reuse stale state.
   - Dependency gate: all index.md dependencies for IP-09 have merged to dev; phase work branch starts from latest origin/dev.

7. `feat(sync): implement ip-09-t07`
   - Task IDs: `IP-09-T07`.
   - Owned target paths: `extension/src/runtime/projection-sync.ts` (to-create), `host/internal/sync/orchestrator.go` (to-create), `host/internal/recovery/rebuild.go` (to-create), `fixtures/recovery/phase-09/` (to-create), `tests/recovery/phase-09/` (to-create).
   - Behavior: **Implement persistence-degraded continuation:** consume IP-08 health at handshake and during runtime. If SQLite open/migration/write fails, keep defaults and session-only metadata in memory, keep a successfully published lexical index queryable and let extension activation proceed; surface `PERSISTENCE_DEGRADED` and retryable health/diagnostic data without blocking synchronization.
   - Fixture and command: IP-08; run `python3 tests/recovery/phase-09/run_fixtures.py --suite phase-09 --fixtures fixtures/recovery/phase-09 --strict`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Implement persistence-degraded continuation:** consume IP-08 health at handshake and during runtime. If SQLite open/migration/write fails, keep defaults and session-only metadata in memory, keep a successfully published lexical index queryable and let extension activation proceed; surface `PERSISTENCE_DEGRADED` and retryable health/diagnostic data without blocking synchronization.
   - Dependency gate: all index.md dependencies for IP-09 have merged to dev; phase work branch starts from latest origin/dev.

8. `feat(sync): implement ip-09-t08`
   - Task IDs: `IP-09-T08`.
   - Owned target paths: `extension/src/runtime/projection-sync.ts` (to-create), `host/internal/sync/orchestrator.go` (to-create), `host/internal/recovery/rebuild.go` (to-create), `fixtures/recovery/phase-09/` (to-create), `tests/recovery/phase-09/` (to-create).
   - Behavior: **Implement revision-safe query handoff:** before forwarding or answering a query, compare requested profile/context/generation/revision with the published ready state. Return `REVISION_MISMATCH` for unknown/old revisions and `INDEX_REBUILDING` for a revision still staging; return no result payload in either case, and never serve the last ready revision as an implicit fallback.
   - Fixture and command: the observable fixture/outcome stated by this task; run `python3 tests/recovery/phase-09/run_fixtures.py --suite phase-09 --fixtures fixtures/recovery/phase-09 --strict`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Implement revision-safe query handoff:** before forwarding or answering a query, compare requested profile/context/generation/revision with the published ready state. Return `REVISION_MISMATCH` for unknown/old revisions and `INDEX_REBUILDING` for a revision still staging; return no result payload in either case, and never serve the last ready revision as an implicit fallback.
   - Dependency gate: all index.md dependencies for IP-09 have merged to dev; phase work branch starts from latest origin/dev.

9. `test(sync): implement ip-09-t09`
   - Task IDs: `IP-09-T09`.
   - Owned target paths: `extension/src/runtime/projection-sync.ts` (to-create), `host/internal/sync/orchestrator.go` (to-create), `host/internal/recovery/rebuild.go` (to-create), `fixtures/recovery/phase-09/` (to-create), `tests/recovery/phase-09/` (to-create).
   - Behavior: **Add fixtures and observability:** create the `SYNC-001`–`SYNC-008` fixtures and tests under the owned roots. Assert lifecycle transitions, exact typed statuses, `sync_ack`/`resync_required`, final identity sets, revision equality, persistence-degraded lexical result/activation, bounded retries and redacted diagnostic fields. Include a no-network/no-page-read test seam.
   - Fixture and command: SYNC-001, SYNC-008; run `python3 tests/recovery/phase-09/run_fixtures.py --suite phase-09 --fixtures fixtures/recovery/phase-09 --strict`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Add fixtures and observability:** create the `SYNC-001`–`SYNC-008` fixtures and tests under the owned roots. Assert lifecycle transitions, exact typed statuses, `sync_ack`/`resync_required`, final identity sets, revision equality, persistence-degraded lexical result/activation, bounded retries and redacted diagnostic fields. Include a no-network/no-page-read test seam.
   - Dependency gate: all index.md dependencies for IP-09 have merged to dev; phase work branch starts from latest origin/dev.

## 8. Kiểm chứng và nghiệm thu

- [ ] From repository root, once the implementation targets exist, run `python3 tests/recovery/phase-09/run_fixtures.py --suite phase-09 --fixtures fixtures/recovery/phase-09 --strict`. The runner must print one bounded pass/fail record per `SYNC-001`–`SYNC-008`, including final state, profile, epoch/revision/sequence, indexed ID set and typed status.
- [ ] Run `go test ./host/... ./tests/recovery/phase-09 -run 'TestSync_(Hello|Snapshot|Delta|Resync|Reconnect|Rebuild|Revision|Persistence|Profile)' -count=1` from the repository root. The test must use only deterministic local fixtures and injected clocks/transport failures.
- [ ] Run `node --test tests/recovery/phase-09/*.test.mjs` from the repository root. Verify the extension sends no delta before `hello_ack`, stops after `resync_required`, reconnects with a fresh snapshot, and reports `Ready` only after the matching `sync_ack`.
- [ ] For `SYNC-002`, drop one delta and deliver a later sequence: assert `resync_required`, a new authoritative snapshot, no partial/duplicate commit, and final indexed IDs exactly equal the browser fixture's eligible IDs. This is the FR-010/NFR-005 acceptance signal.
- [ ] For `SYNC-006`, query an unknown revision and a revision whose index is rebuilding: assert `REVISION_MISMATCH` and `INDEX_REBUILDING` respectively, with no result references or display records in either response and no fallback to an older revision.
- [ ] For `SYNC-007`, force SQLite open/migration/write failure after handshake: assert `PERSISTENCE_DEGRADED` is visible while snapshot, lexical query and browser activation still succeed from memory. Verify recovery of storage does not change the live projection or require reinstall.
- [ ] Exercise host crash, service-worker suspension, interrupted snapshot and reconnect: assert bounded `Recovering`/`Disconnected` states, a fresh hello/snapshot, eventual `Ready` without manual reinstall, and no callback from the old generation mutating the new state (NFR-006).
- [ ] Confirm profile/context mismatch and malformed/out-of-order input fail closed; confirm diagnostics contain only bounded counts, durations, versions, revisions and error classes, with no raw title, URL, query, token or page data.
- [ ] Confirm the synchronization path performs no network request, page evaluation, history read or SQLite query for live projection data, and that `git diff --check` is clean for the implementation commits.

## 9. Rủi ro và quyết định còn mở

- Rủi ro: a delta gap can be detected after the host has accepted earlier events. Phương án xử lý đã chọn: commit each batch only at the projection/index barrier; keep the last complete generation, emit `resync_required`, discard queued deltas and use a new full snapshot as authority.
- Rủi ro: an index rebuild can finish after disconnect or after a newer snapshot starts. Phương án xử lý đã chọn: tag every staged build with session generation and projection revision; publish callbacks only when both still match, otherwise discard them and report recovery.
- Rủi ro: a valid empty snapshot could be confused with a failed browser read. Phương án xử lý đã chọn: consume IP-04's explicit completion/authority marker; only a completed profile-bound empty snapshot may replace the prior projection.
- Rủi ro: persistence failure could make the host incorrectly report unavailable search. Phương án xử lý đã chọn: treat SQLite as optional for the live path, use in-memory defaults/session metadata, expose `PERSISTENCE_DEGRADED`, and keep lexical results/activation independent.
- Rủi ro: a result request can race with a revision transition. Phương án xử lý đã chọn: require exact profile/context/generation/revision matching at query and activation handoff; unknown or rebuilding state returns no result rather than stale data.
- Câu hỏi còn mở: none. The synchronization, recovery, revision refusal and persistence-degraded contracts are fixed here; implementation may optimize staging without changing their observable outcomes.

## 10. References ngoài `docs/`

- Skill: [`pipelines`](../../../.agent/skills/personal/engineering/data/pipelines/SKILL.md), [`go`](../../../.agent/skills/personal/engineering/backend/go/SKILL.md), [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md), [`debugging`](../../../.agent/skills/common/engineering/debugging/SKILL.md), [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md), [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md), [`git-workflow`](../../../.agent/skills/common/engineering/git-workflow/SKILL.md)
- Project context: [`CONTEXT.md`](../../../CONTEXT.md)
- Source/config/test path ngoài `docs/`: `extension/src/runtime/projection-sync.ts` (to-create), `host/internal/sync/orchestrator.go` (to-create), `host/internal/recovery/rebuild.go` (to-create), `tests/recovery/phase-09/` (to-create)
- Fixture/tool/artifact ngoài `docs/`: `fixtures/recovery/phase-09/SYNC-001`–`SYNC-008` (to-create); `python3 tests/recovery/phase-09/run_fixtures.py --suite phase-09 --fixtures fixtures/recovery/phase-09 --strict` (future executable acceptance runner)
