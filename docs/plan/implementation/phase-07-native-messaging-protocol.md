# Phase 07 — Native messaging protocol

> Plan ID: IP-07
> Status: See README.md execution tracker
> Execution owner: Go protocol/session implementer
> Dependencies: IP-02, IP-06
> Parallel boundary: IP-04 and IP-08 after the IP-02/IP-06 ownership contracts are stable; IP-05 and IP-09 consume this contract
> Requirement IDs: FR-010, FR-012, FR-013, NFR-005, NFR-006; operational protocol versioning, payload limits, bounded timeouts, and fail-closed errors
> Owned paths: `host/internal/protocol/envelope.go` (to-create); `host/internal/protocol/messages.go` (to-create); `host/internal/protocol/codec.go` (to-create); `host/internal/protocol/session.go` (to-create); `extension/src/runtime/native-messaging-client.ts` (to-create); `fixtures/protocol/` (to-create); `tests/protocol/` (to-create)

## 1. Mục tiêu
- Chốt một contract framed Native Messaging giữa Manifest V3 extension và Go host, để cả hai phía có cùng schema, thứ tự, giới hạn, timeout và cách chuyển trạng thái lỗi.
- Định nghĩa chính xác envelope gồm `protocol`, `type`, `request_id`, `profile_id`, `projection_revision`, `payload`; mọi message hợp lệ và mọi phản hồi lỗi đều phải đi qua contract này.
- Chốt handshake, đồng bộ snapshot/delta, query, activation status và health sao cho full snapshot là authority, delta không thể làm index lùi revision, query không trả dữ liệu âm thầm stale, và reconnect có đường resync quan sát được.
- Sở hữu operational contract về protocol version, giới hạn frame/payload/field, timeout hữu hạn và fail-closed behavior; cung cấp fixture có observable outcome cho malformed, oversized, out-of-order, duplicate và disconnect.

## 2. Phạm vi
- Bao gồm:
  - Framing stdin/stdout Native Messaging: 4-byte unsigned little-endian length prefix, UTF-8 JSON object, bounded read trước khi decode hoặc cấp phát payload.
  - Schema strict cho envelope và payload của `hello`, `hello_ack`, `snapshot`, `delta`, `sync_ack`, `resync_required`, `query`, `query_result`, activation status (`activation_observed`, `activation_failed`, `activation_ack`), health (`health`, `health_result`) và `error`.
  - Protocol version negotiation, profile binding, request identity, response correlation, idempotency trong một connection, revision/sequence ordering và session shutdown/reconnect.
  - Exact error literals: `INVALID_FRAME`, `PROTOCOL_MISMATCH`, `PROFILE_MISMATCH`, `PAYLOAD_LIMIT`, `SNAPSHOT_REQUIRED`, `REVISION_MISMATCH`, `INDEX_REBUILDING`, `QUERY_TIMEOUT`, `PERSISTENCE_DEGRADED`, `HOST_SHUTDOWN`, `INTERNAL_FAILURE`.
  - Redacted error payload, health/degraded states, bounded request timeouts và test seams độc lập với browser UI.
- Ngoài phạm vi:
  - Không mở loopback/listening server, không thêm network transport và không đọc nguồn dữ liệu ngoài profile projection do extension gửi.
  - Không sở hữu browser API, tab eligibility/identity, activation thực tế, lexical tokenization/ranking, SQLite schema hoặc installer; các boundary đó thuộc IP-02, IP-04, IP-08, IP-10, IP-14 và IP-18.
  - Không thay đổi focused search UI, command, permissions hay packaging. Protocol chỉ mang dữ liệu contract đã được phép và không tự mở rộng quyền.

## 3. Điều kiện tiên quyết
- IP-02 phải chốt `profile_id`, tab identity, projection revision, eligible-tab record và ownership extension/host; phase này không được tự định nghĩa lại các field domain đó.
- IP-06 phải chốt process/session lifetime, stdin/stdout ownership, startup/shutdown, cancellation và reconnect seam của Go host. Protocol runner phải dùng boundary đó, không tạo daemon hoặc transport thứ hai.
- Canonical contract phải được đọc trước khi code: `docs/plan/refactor/runtime-protocol.md`, `architecture.md`, `requirements.md` và `verification-and-acceptance.md`. Các field mới hoặc thay đổi breaking phải tăng major protocol và có recovery path.
- Cần có fixture runner deterministic nhận raw framed bytes, clock/deadline injection, profile ID, current revision/sequence và fake index/persistence health; không cần browser thật để kiểm tra codec/session.

## 4. Đầu ra cần bàn giao
- Go protocol package tại các target path trong metadata:
  - Strict envelope/message types, JSON codec và bounded native frame reader/writer.
  - Session state machine cho handshake, profile binding, snapshot authority, delta ordering, request ledger, cancellation, timeout và shutdown.
  - Typed error constructors/serialization với safe code, retryability và request identity; không serialize raw title, URL, secret hoặc stack trace.
- Extension client seam tại `extension/src/runtime/native-messaging-client.ts`: encode/decode cùng schema, correlate request/response, expose Ready/Degraded/Reconnecting/Incompatible/Shutdown states và phát tín hiệu resync.
- Fixture catalog dưới `fixtures/protocol/` gồm raw frames và expected events/errors cho handshake, every message type, limits, ordering, duplicate, interrupted frame và reconnect.
- Targeted protocol tests dưới `tests/protocol/` chứng minh exact envelope, byte limits, state transitions, typed errors, no stale query result và no cross-profile mutation.
- Protocol compatibility note trong code comments/test data: additive fields chỉ khi decoder vẫn bounded và unknown version/message fail closed; breaking change increments major protocol.

## 5. Skill và tài liệu áp dụng
- Skill tags:
  - `common/engineering/documentation` — ghi contract, fixture intent và acceptance observable để implementation/review không suy diễn schema.
  - `common/foundation/task-planning` — chia boundary codec/session/state/fixture thành các commit triển khai được.
  - `common/engineering/testing` — thiết kế fixture deterministic cho byte stream, ordering, timeout, reconnect và typed errors.
  - `common/engineering/git-workflow` — giữ thay đổi phase ở một file và lập commit cut nhỏ, kiểm chứng được.
  - `personal/engineering/backend/api-design` — thiết kế envelope, message direction, correlation, compatibility và error contract.
  - `personal/engineering/backend/go` — framed I/O, context cancellation, deadline, bounded allocation và session ownership trong Go.
  - `common/security/secure-development` — fail closed, input limits, profile isolation, redaction và chống resource exhaustion.
- Tài liệu trong `docs/`:
  - [Runtime protocol](../refactor/runtime-protocol.md) — envelope, message families, compatibility và exact error literals.
  - [Target architecture](../refactor/architecture.md) — extension/host ownership và không dùng transport ngoài Native Messaging.
  - [Product requirements](../refactor/requirements.md) — FR-010, FR-012, FR-013, NFR-005, NFR-006 và operational requirements.
  - [Verification and acceptance](../refactor/verification-and-acceptance.md) — handshake, mismatch, malformed/oversized, duplicate, ordering, reconnect và persistence failure journeys.
- Quy ước code, ADR, context ngoài `docs/`: `CONTEXT.md` là vocabulary và source-of-truth rule; target implementation hiện là logical `to-create`, không giả định symbol đã tồn tại.

## 6. Công việc triển khai
- [ ] `IP-07-T01` Việc 1 — Framing và envelope: tại `host/internal/protocol/codec.go`, đọc đúng 4-byte little-endian length; từ chối length bằng 0 hoặc lớn hơn 1 MiB trước khi allocate, giới hạn JSON payload encoded ở 256 KiB, parse một JSON object duy nhất rồi reject trailing bytes. Envelope phải có đúng sáu field bắt buộc: `protocol` integer major (hiện tại 1), `type` enum, `request_id` opaque non-empty string tối đa 128 bytes, `profile_id` opaque non-empty string tối đa 128 bytes, `projection_revision` non-negative integer, `payload` object. Reject missing, null, sai type, duplicate JSON keys và field ngoài allowlist bằng `INVALID_FRAME`.
- [ ] `IP-07-T02` Việc 2 — Limits trước cấp phát/index: giới hạn `query` 512 Unicode scalars, 10,000 tab records/snapshot, 1,000 delta operations/message, 50 results/query, title 512 scalars, domain 255 bytes, URL 2,048 bytes, 32 capabilities và 64 bytes cho message type. Mỗi giới hạn phải có fixture vừa dưới ngưỡng và vượt đúng một byte/record; vượt ngưỡng trả `PAYLOAD_LIMIT`, không partial-apply và không index dữ liệu.
- [ ] `IP-07-T03` Việc 3 — Handshake/session: tại `host/internal/protocol/session.go`, chỉ nhận `hello` khi connection chưa bound; payload gồm extension version, browser family/version, profile identity, capabilities và supported protocol versions. `hello_ack` trả accepted protocol, host version, limits, ranking model version và persistence health. Chưa có ack thì không nhận tab data/query; mismatch version trả `PROTOCOL_MISMATCH`, không gửi/ghi projection, state là Incompatible và hướng dẫn repair. Bind profile từ hello đầu tiên; mọi frame khác profile trả `PROFILE_MISMATCH` và không mutate state.
- [ ] `IP-07-T04` Việc 4 — Message contract: tại `host/internal/protocol/messages.go`, lập schema direction và required payload sau đây:
  - `snapshot` extension → host: payload `{tabs, sequence}`; envelope revision là revision đầy đủ. Apply atomically; snapshot là authority và có thể thay projection cũ bằng revision mới hơn. Host trả `sync_ack` payload `{accepted_revision, accepted_sequence}`.
  - `delta` extension → host: payload `{base_revision, sequence_start, sequence_end, operations}`; operations ordered create/update/move/group/pin/activate/remove. Envelope revision là target revision. Chỉ nhận khi base revision là revision hiện tại, sequence bắt đầu đúng expected và sequence contiguous; không nhận delta trước snapshot.
  - `resync_required` host → extension: payload `{reason, expected_revision, expected_sequence}`; reason chỉ là safe enum. Extension phải gửi full snapshot, không retry delta mù.
  - `query` extension → host: payload `{query, result_limit, current_window_id}`; revision trong envelope là revision client biết. `query_result` trả bounded result references/display metadata, ranking model version, revision và timing counters. Host reject unknown/rebuilding revision thay vì trả silently stale data.
  - Activation status extension → host: `activation_observed` hoặc `activation_failed` payload gồm result reference, tab ID, revision và safe outcome/error class; host trả `activation_ack`. Extension vẫn sở hữu browser activation; host không nhận lệnh để chọn tab.
  - `health` extension → host và `health_result` host → extension: payload gồm host/protocol/session/storage/index state, current revision/sequence, freshness age, retryable flag và bounded counters; không gồm raw tab fields.
  - `error` response: payload bắt buộc `{code, retryable, message_key}`, tùy chọn bounded `retry_after_ms`/`expected_revision`/`expected_sequence`; code chỉ thuộc exact error enum và response echo `request_id`.
- [ ] `IP-07-T05` Việc 5 — Ordering và idempotency: snapshot thay atomically và luôn thắng delta cũ; delta có base revision/sequence range monotonic. Gap, wrong base, out-of-order hoặc target revision không hợp lệ trả `resync_required`/`REVISION_MISMATCH` theo matrix, không apply một phần. Request ledger key là (connection, request_id): duplicate cùng bytes replay cùng response không chạy side effect lần hai; cùng ID khác payload trả `INVALID_FRAME` và đóng session. Exact duplicate snapshot/delta đã commit có thể trả lại cùng `sync_ack` khi digest khớp; duplicate khác digest trả `REVISION_MISMATCH` và yêu cầu snapshot.
- [ ] `IP-07-T06` Việc 6 — Timeout, cancellation và disconnect: deadline constants phải nằm trong một config typed, không rải magic number: frame read 2 s, hello/hello_ack 2 s, snapshot/delta apply + ack 1 s, query 40 ms host budget và 100 ms request budget, health 100 ms, activation status 500 ms, graceful shutdown drain 1 s. On deadline query trả `QUERY_TIMEOUT` không partial result; on cancellation drop response safely; EOF/short frame/partial write marks session Disconnected, fails in-flight requests with `HOST_SHUTDOWN` hoặc transport-unavailable mapping, clears request ledger and requires a new hello/snapshot after reconnect.
- [ ] `IP-07-T07` Việc 7 — Exact error/state matrix: `INVALID_FRAME` cho malformed JSON/schema/trailing bytes; `PROTOCOL_MISMATCH` cho unsupported version/handshake; `PROFILE_MISMATCH` cho wrong profile; `PAYLOAD_LIMIT` cho any bound; `SNAPSHOT_REQUIRED` cho delta/query before accepted snapshot; `REVISION_MISMATCH` cho stale base/sequence; `INDEX_REBUILDING` cho query while rebuild; `QUERY_TIMEOUT` for query deadline; `PERSISTENCE_DEGRADED` as visible degraded health/query annotation while lexical result remains available; `HOST_SHUTDOWN` for intentional close/in-flight drain; `INTERNAL_FAILURE` for unexpected safe-contained failure. Every error records retryable and redacted diagnostic metadata only.
- [ ] `IP-07-T08` Việc 8 — Extension client seam: tại `extension/src/runtime/native-messaging-client.ts`, reject uncorrelated responses, enforce one pending request deadline, surface typed state transitions, stop sending after protocol/profile failure, and request snapshot on `resync_required`. The client must not render query results whose revision is unknown and must preserve host-down/rebuilding/degraded status for the UI owner.
- [ ] `IP-07-T09` Việc 9 — Fixture/test seam: create deterministic framed byte fixtures với IDs `NM-PROTO-001` malformed envelope, `NM-PROTO-002` unsupported protocol, `NM-PROTO-003` profile mismatch, `NM-PROTO-004` frame/payload boundary, `NM-PROTO-005` hello-before-data, `NM-PROTO-006` snapshot authority, `NM-PROTO-007` delta gap/order, `NM-PROTO-008` exact duplicate replay, `NM-PROTO-009` duplicate request ID with changed payload, `NM-PROTO-010` query while rebuilding/timeout, `NM-PROTO-011` persistence degraded, `NM-PROTO-012` disconnect during frame, and `NM-PROTO-013` reconnect requiring hello then snapshot. Each fixture declares input bytes/events, pre-state, expected emitted frames, expected state, no-side-effect assertion and redaction assertion.

## 7. Kế hoạch commit

1. `feat(protocol): implement ip-07-t01`
   - Task IDs: `IP-07-T01`.
   - Owned target paths: host/internal/protocol/codec.go.
   - Behavior: Việc 1 — Framing và envelope: tại `host/internal/protocol/codec.go`, đọc đúng 4-byte little-endian length; từ chối length bằng 0 hoặc lớn hơn 1 MiB trước khi allocate, giới hạn JSON payload encoded ở 256 KiB, parse một JSON object duy nhất rồi reject trailing bytes. Envelope phải có đúng sáu field bắt buộc: `protocol` integer major (hiện tại 1), `type` enum, `request_id` opaque non-empty string tối đa 128 bytes, `profile_id` opaque non-empty string tối đa 128 bytes, `projection_revision` non-negative integer, `payload` object. Reject missing, null, sai type, duplicate JSON keys và field ngoài allowlist bằng `INVALID_FRAME`.
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/... -run 'TestProtocol_(Envelope|Handshake|Ordering|Limits|Timeout|Disconnect)' -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Việc 1 — Framing và envelope: tại `host/internal/protocol/codec.go`, đọc đúng 4-byte little-endian length; từ chối length bằng 0 hoặc lớn hơn 1 MiB trước khi allocate, giới hạn JSON payload encoded ở 256 KiB, parse một JSON object duy nhất rồi reject trailing bytes. Envelope phải có đúng sáu field bắt buộc: `protocol` integer major (hiện tại 1), `type` enum, `request_id` opaque non-empty string tối đa 128 bytes, `profile_id` opaque non-empty string tối đa 128 bytes, `projection_revision` non-negative integer, `payload` object. Reject missing, null, sai type, duplicate JSON keys và field ngoài allowlist bằng `INVALID_FRAME`.
   - Dependency gate: all index.md dependencies for IP-07 have merged to dev; phase work branch starts from latest origin/dev.

2. `test(protocol): implement ip-07-t02`
   - Task IDs: `IP-07-T02`.
   - Owned target paths: `host/internal/protocol/envelope.go` (to-create); `host/internal/protocol/messages.go` (to-create); `host/internal/protocol/codec.go` (to-create); `host/internal/protocol/session.go` (to-create); `extension/src/runtime/native-messaging-client.ts` (to-create); `fixtures/protocol/` (to-create); `tests/protocol/` (to-create).
   - Behavior: Việc 2 — Limits trước cấp phát/index: giới hạn `query` 512 Unicode scalars, 10,000 tab records/snapshot, 1,000 delta operations/message, 50 results/query, title 512 scalars, domain 255 bytes, URL 2,048 bytes, 32 capabilities và 64 bytes cho message type. Mỗi giới hạn phải có fixture vừa dưới ngưỡng và vượt đúng một byte/record; vượt ngưỡng trả `PAYLOAD_LIMIT`, không partial-apply và không index dữ liệu.
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/... -run 'TestProtocol_(Envelope|Handshake|Ordering|Limits|Timeout|Disconnect)' -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Việc 2 — Limits trước cấp phát/index: giới hạn `query` 512 Unicode scalars, 10,000 tab records/snapshot, 1,000 delta operations/message, 50 results/query, title 512 scalars, domain 255 bytes, URL 2,048 bytes, 32 capabilities và 64 bytes cho message type. Mỗi giới hạn phải có fixture vừa dưới ngưỡng và vượt đúng một byte/record; vượt ngưỡng trả `PAYLOAD_LIMIT`, không partial-apply và không index dữ liệu.
   - Dependency gate: all index.md dependencies for IP-07 have merged to dev; phase work branch starts from latest origin/dev.

3. `feat(protocol): implement ip-07-t03`
   - Task IDs: `IP-07-T03`.
   - Owned target paths: host/internal/protocol/session.go.
   - Behavior: Việc 3 — Handshake/session: tại `host/internal/protocol/session.go`, chỉ nhận `hello` khi connection chưa bound; payload gồm extension version, browser family/version, profile identity, capabilities và supported protocol versions. `hello_ack` trả accepted protocol, host version, limits, ranking model version và persistence health. Chưa có ack thì không nhận tab data/query; mismatch version trả `PROTOCOL_MISMATCH`, không gửi/ghi projection, state là Incompatible và hướng dẫn repair. Bind profile từ hello đầu tiên; mọi frame khác profile trả `PROFILE_MISMATCH` và không mutate state.
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/... -run 'TestProtocol_(Envelope|Handshake|Ordering|Limits|Timeout|Disconnect)' -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Việc 3 — Handshake/session: tại `host/internal/protocol/session.go`, chỉ nhận `hello` khi connection chưa bound; payload gồm extension version, browser family/version, profile identity, capabilities và supported protocol versions. `hello_ack` trả accepted protocol, host version, limits, ranking model version và persistence health. Chưa có ack thì không nhận tab data/query; mismatch version trả `PROTOCOL_MISMATCH`, không gửi/ghi projection, state là Incompatible và hướng dẫn repair. Bind profile từ hello đầu tiên; mọi frame khác profile trả `PROFILE_MISMATCH` và không mutate state.
   - Dependency gate: all index.md dependencies for IP-07 have merged to dev; phase work branch starts from latest origin/dev.

4. `feat(protocol): implement ip-07-t04`
   - Task IDs: `IP-07-T04`.
   - Owned target paths: host/internal/protocol/messages.go.
   - Behavior: Việc 4 — Message contract: tại `host/internal/protocol/messages.go`, lập schema direction và required payload sau đây:
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/... -run 'TestProtocol_(Envelope|Handshake|Ordering|Limits|Timeout|Disconnect)' -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Việc 4 — Message contract: tại `host/internal/protocol/messages.go`, lập schema direction và required payload sau đây:
   - Dependency gate: all index.md dependencies for IP-07 have merged to dev; phase work branch starts from latest origin/dev.

5. `feat(protocol): implement ip-07-t05`
   - Task IDs: `IP-07-T05`.
   - Owned target paths: `host/internal/protocol/envelope.go` (to-create); `host/internal/protocol/messages.go` (to-create); `host/internal/protocol/codec.go` (to-create); `host/internal/protocol/session.go` (to-create); `extension/src/runtime/native-messaging-client.ts` (to-create); `fixtures/protocol/` (to-create); `tests/protocol/` (to-create).
   - Behavior: Việc 5 — Ordering và idempotency: snapshot thay atomically và luôn thắng delta cũ; delta có base revision/sequence range monotonic. Gap, wrong base, out-of-order hoặc target revision không hợp lệ trả `resync_required`/`REVISION_MISMATCH` theo matrix, không apply một phần. Request ledger key là (connection, request_id): duplicate cùng bytes replay cùng response không chạy side effect lần hai; cùng ID khác payload trả `INVALID_FRAME` và đóng session. Exact duplicate snapshot/delta đã commit có thể trả lại cùng `sync_ack` khi digest khớp; duplicate khác digest trả `REVISION_MISMATCH` và yêu cầu snapshot.
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/... -run 'TestProtocol_(Envelope|Handshake|Ordering|Limits|Timeout|Disconnect)' -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Việc 5 — Ordering và idempotency: snapshot thay atomically và luôn thắng delta cũ; delta có base revision/sequence range monotonic. Gap, wrong base, out-of-order hoặc target revision không hợp lệ trả `resync_required`/`REVISION_MISMATCH` theo matrix, không apply một phần. Request ledger key là (connection, request_id): duplicate cùng bytes replay cùng response không chạy side effect lần hai; cùng ID khác payload trả `INVALID_FRAME` và đóng session. Exact duplicate snapshot/delta đã commit có thể trả lại cùng `sync_ack` khi digest khớp; duplicate khác digest trả `REVISION_MISMATCH` và yêu cầu snapshot.
   - Dependency gate: all index.md dependencies for IP-07 have merged to dev; phase work branch starts from latest origin/dev.

6. `feat(protocol): implement ip-07-t06`
   - Task IDs: `IP-07-T06`.
   - Owned target paths: `host/internal/protocol/envelope.go` (to-create); `host/internal/protocol/messages.go` (to-create); `host/internal/protocol/codec.go` (to-create); `host/internal/protocol/session.go` (to-create); `extension/src/runtime/native-messaging-client.ts` (to-create); `fixtures/protocol/` (to-create); `tests/protocol/` (to-create).
   - Behavior: Việc 6 — Timeout, cancellation và disconnect: deadline constants phải nằm trong một config typed, không rải magic number: frame read 2 s, hello/hello_ack 2 s, snapshot/delta apply + ack 1 s, query 40 ms host budget và 100 ms request budget, health 100 ms, activation status 500 ms, graceful shutdown drain 1 s. On deadline query trả `QUERY_TIMEOUT` không partial result; on cancellation drop response safely; EOF/short frame/partial write marks session Disconnected, fails in-flight requests with `HOST_SHUTDOWN` hoặc transport-unavailable mapping, clears request ledger and requires a new hello/snapshot after reconnect.
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/... -run 'TestProtocol_(Envelope|Handshake|Ordering|Limits|Timeout|Disconnect)' -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Việc 6 — Timeout, cancellation và disconnect: deadline constants phải nằm trong một config typed, không rải magic number: frame read 2 s, hello/hello_ack 2 s, snapshot/delta apply + ack 1 s, query 40 ms host budget và 100 ms request budget, health 100 ms, activation status 500 ms, graceful shutdown drain 1 s. On deadline query trả `QUERY_TIMEOUT` không partial result; on cancellation drop response safely; EOF/short frame/partial write marks session Disconnected, fails in-flight requests with `HOST_SHUTDOWN` hoặc transport-unavailable mapping, clears request ledger and requires a new hello/snapshot after reconnect.
   - Dependency gate: all index.md dependencies for IP-07 have merged to dev; phase work branch starts from latest origin/dev.

7. `feat(protocol): implement ip-07-t07`
   - Task IDs: `IP-07-T07`.
   - Owned target paths: `host/internal/protocol/envelope.go` (to-create); `host/internal/protocol/messages.go` (to-create); `host/internal/protocol/codec.go` (to-create); `host/internal/protocol/session.go` (to-create); `extension/src/runtime/native-messaging-client.ts` (to-create); `fixtures/protocol/` (to-create); `tests/protocol/` (to-create).
   - Behavior: Việc 7 — Exact error/state matrix: `INVALID_FRAME` cho malformed JSON/schema/trailing bytes; `PROTOCOL_MISMATCH` cho unsupported version/handshake; `PROFILE_MISMATCH` cho wrong profile; `PAYLOAD_LIMIT` cho any bound; `SNAPSHOT_REQUIRED` cho delta/query before accepted snapshot; `REVISION_MISMATCH` cho stale base/sequence; `INDEX_REBUILDING` cho query while rebuild; `QUERY_TIMEOUT` for query deadline; `PERSISTENCE_DEGRADED` as visible degraded health/query annotation while lexical result remains available; `HOST_SHUTDOWN` for intentional close/in-flight drain; `INTERNAL_FAILURE` for unexpected safe-contained failure. Every error records retryable and redacted diagnostic metadata only.
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/... -run 'TestProtocol_(Envelope|Handshake|Ordering|Limits|Timeout|Disconnect)' -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Việc 7 — Exact error/state matrix: `INVALID_FRAME` cho malformed JSON/schema/trailing bytes; `PROTOCOL_MISMATCH` cho unsupported version/handshake; `PROFILE_MISMATCH` cho wrong profile; `PAYLOAD_LIMIT` cho any bound; `SNAPSHOT_REQUIRED` cho delta/query before accepted snapshot; `REVISION_MISMATCH` cho stale base/sequence; `INDEX_REBUILDING` cho query while rebuild; `QUERY_TIMEOUT` for query deadline; `PERSISTENCE_DEGRADED` as visible degraded health/query annotation while lexical result remains available; `HOST_SHUTDOWN` for intentional close/in-flight drain; `INTERNAL_FAILURE` for unexpected safe-contained failure. Every error records retryable and redacted diagnostic metadata only.
   - Dependency gate: all index.md dependencies for IP-07 have merged to dev; phase work branch starts from latest origin/dev.

8. `feat(protocol): implement ip-07-t08`
   - Task IDs: `IP-07-T08`.
   - Owned target paths: extension/src/runtime/native-messaging-client.ts.
   - Behavior: Việc 8 — Extension client seam: tại `extension/src/runtime/native-messaging-client.ts`, reject uncorrelated responses, enforce one pending request deadline, surface typed state transitions, stop sending after protocol/profile failure, and request snapshot on `resync_required`. The client must not render query results whose revision is unknown and must preserve host-down/rebuilding/degraded status for the UI owner.
   - Fixture and command: the observable fixture/outcome stated by this task; run `go test ./host/... -run 'TestProtocol_(Envelope|Handshake|Ordering|Limits|Timeout|Disconnect)' -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Việc 8 — Extension client seam: tại `extension/src/runtime/native-messaging-client.ts`, reject uncorrelated responses, enforce one pending request deadline, surface typed state transitions, stop sending after protocol/profile failure, and request snapshot on `resync_required`. The client must not render query results whose revision is unknown and must preserve host-down/rebuilding/degraded status for the UI owner.
   - Dependency gate: all index.md dependencies for IP-07 have merged to dev; phase work branch starts from latest origin/dev.

9. `test(protocol): implement ip-07-t09`
   - Task IDs: `IP-07-T09`.
   - Owned target paths: `host/internal/protocol/envelope.go` (to-create); `host/internal/protocol/messages.go` (to-create); `host/internal/protocol/codec.go` (to-create); `host/internal/protocol/session.go` (to-create); `extension/src/runtime/native-messaging-client.ts` (to-create); `fixtures/protocol/` (to-create); `tests/protocol/` (to-create).
   - Behavior: Việc 9 — Fixture/test seam: create deterministic framed byte fixtures với IDs `NM-PROTO-001` malformed envelope, `NM-PROTO-002` unsupported protocol, `NM-PROTO-003` profile mismatch, `NM-PROTO-004` frame/payload boundary, `NM-PROTO-005` hello-before-data, `NM-PROTO-006` snapshot authority, `NM-PROTO-007` delta gap/order, `NM-PROTO-008` exact duplicate replay, `NM-PROTO-009` duplicate request ID with changed payload, `NM-PROTO-010` query while rebuilding/timeout, `NM-PROTO-011` persistence degraded, `NM-PROTO-012` disconnect during frame, and `NM-PROTO-013` reconnect requiring hello then snapshot. Each fixture declares input bytes/events, pre-state, expected emitted frames, expected state, no-side-effect assertion and redaction assertion.
   - Fixture and command: NM-PROTO-001, NM-PROTO-002, NM-PROTO-003, NM-PROTO-004, NM-PROTO-005, NM-PROTO-006, NM-PROTO-007, NM-PROTO-008, NM-PROTO-009, NM-PROTO-010, NM-PROTO-011, NM-PROTO-012, NM-PROTO-013; run `go test ./host/... -run 'TestProtocol_(Envelope|Handshake|Ordering|Limits|Timeout|Disconnect)' -count=1`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: Việc 9 — Fixture/test seam: create deterministic framed byte fixtures với IDs `NM-PROTO-001` malformed envelope, `NM-PROTO-002` unsupported protocol, `NM-PROTO-003` profile mismatch, `NM-PROTO-004` frame/payload boundary, `NM-PROTO-005` hello-before-data, `NM-PROTO-006` snapshot authority, `NM-PROTO-007` delta gap/order, `NM-PROTO-008` exact duplicate replay, `NM-PROTO-009` duplicate request ID with changed payload, `NM-PROTO-010` query while rebuilding/timeout, `NM-PROTO-011` persistence degraded, `NM-PROTO-012` disconnect during frame, and `NM-PROTO-013` reconnect requiring hello then snapshot. Each fixture declares input bytes/events, pre-state, expected emitted frames, expected state, no-side-effect assertion and redaction assertion.
   - Dependency gate: all index.md dependencies for IP-07 have merged to dev; phase work branch starts from latest origin/dev.

## 8. Kiểm chứng và nghiệm thu
- [ ] Chạy từ repository root sau khi implementation tạo các target path: `go test ./host/... -run 'TestProtocol_(Envelope|Handshake|Ordering|Limits|Timeout|Disconnect)' -count=1`.
- [ ] Chạy `node --test tests/protocol/native-messaging-client.test.mjs`; fixture runner phải load chính xác `fixtures/protocol/NM-PROTO-001..013` và không dùng network/browser history/page content.
- [ ] `NM-PROTO-001` malformed/trailing/unknown field → exactly `INVALID_FRAME`, no state mutation; unsupported version → exactly `PROTOCOL_MISMATCH`, no tab data accepted; wrong profile → exactly `PROFILE_MISMATCH`.
- [ ] `NM-PROTO-004` at-limit frame/payload succeeds and one-byte/one-record over limit → exactly `PAYLOAD_LIMIT`, no partial allocation/indexing; query/frame deadlines emit exactly `QUERY_TIMEOUT` and no partial results.
- [ ] `NM-PROTO-005` data before successful hello/snapshot → exactly `SNAPSHOT_REQUIRED`; `NM-PROTO-006` newer full snapshot atomically replaces older projection and indexed IDs equal snapshot eligible IDs (FR-010, NFR-005).
- [ ] `NM-PROTO-007` gap/out-of-order/stale delta → exactly `resync_required` plus `REVISION_MISMATCH` where applicable; no out-of-order operation remains visible; duplicate exact bytes replay one response and one side effect, changed bytes under same request ID fail closed.
- [ ] `NM-PROTO-010` rebuilding/host timeout exposes `INDEX_REBUILDING`/`QUERY_TIMEOUT`; `NM-PROTO-011` exposes `PERSISTENCE_DEGRADED` while lexical query remains available (FR-013); health exposes availability and freshness (FR-012).
- [ ] `NM-PROTO-012` EOF, short length, interrupted write and process shutdown fail in-flight work with `HOST_SHUTDOWN` or explicit disconnect state, clear session identity, and never accept post-disconnect frames; `NM-PROTO-013` reconnect reaches hello → snapshot → ready without reinstall (NFR-006).
- [ ] Verify every error response has the original request ID when safely parseable, bounded safe fields, retryability, and no raw title/URL/secret; verify unknown protocol/message and malformed frames fail closed.
- [ ] Verify timing counters and deadline tests are deterministic under injected clock; no command in this phase opens a loopback port or performs network access.

## 9. Rủi ro và quyết định còn mở
- Rủi ro: Native Messaging framing and JSON parsing can become an allocation/resource-exhaustion boundary. Phương án xử lý đã chọn: enforce 1 MiB frame/256 KiB payload and all nested limits while reading, reject before decode/index, and keep limits returned by `hello_ack`.
- Rủi ro: reconnect or duplicate delivery can regress the lexical index. Phương án xử lý đã chọn: snapshot authority, monotonic revision/sequence, digest-checked idempotent replay, and mandatory resync on ambiguity.
- Rủi ro: protocol errors can leak tab content. Phương án xử lý đã chọn: stable safe error codes/message keys and metadata-only diagnostics; never include raw fields by default.
- Rủi ro: persistence failure could incorrectly make search unavailable. Phương án xử lý đã chọn: expose `PERSISTENCE_DEGRADED` in health/query metadata while keeping in-memory lexical path available; IP-08 owns storage repair semantics.
- Rủi ro: a stale response could cross a profile or revision boundary. Phương án xử lý đã chọn: bind profile at hello, echo request IDs, require known revision for query/result use, and clear all in-flight state on disconnect.
- Câu hỏi còn mở chỉ khi câu trả lời có thể thay đổi contract: whether a future additive protocol version needs a capability bit for streaming deltas; default remains one bounded message and no streaming until separately versioned.

## 10. References ngoài `docs/`
- Skill: [`api-design`](../../../.agent/skills/personal/engineering/backend/api-design/SKILL.md)
- Skill: [`go`](../../../.agent/skills/personal/engineering/backend/go/SKILL.md)
- Skill: [`secure-development`](../../../.agent/skills/common/security/secure-development/SKILL.md)
- Skill: [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md)
- Skill: [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md)
- Skill: [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md)
- Skill: [`git-workflow`](../../../.agent/skills/common/engineering/git-workflow/SKILL.md)
- Project context: [`CONTEXT.md`](../../../CONTEXT.md)
- Source/config/test path ngoài `docs/`: `host/internal/protocol/`, `extension/src/runtime/native-messaging-client.ts`, `fixtures/protocol/`, `tests/protocol/` (all to-create logical paths).
- Fixture/tool/artifact ngoài `docs/`: `fixtures/protocol/NM-PROTO-001..013` and framed stdin/stdout byte-stream harness (to-create).
