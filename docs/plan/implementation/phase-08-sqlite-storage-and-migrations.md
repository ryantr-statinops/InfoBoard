# Phase 08 — SQLite storage and migrations

> Plan ID: IP-08
> Status: not_started
> Execution owner: Go host persistence owner
> Dependencies: IP-02, IP-06
> Parallel boundary: IP-07
> Requirement IDs: FR-013, FR-014, FR-015, NFR-006, NFR-009; operational schema versioning, migration recovery, reset/uninstall idempotency
> Owned paths: `host/storage/` (to-create), `host/storage/migrations/` (to-create), `fixtures/sqlite/` (to-create), `tests/storage/` (to-create)

## 1. Mục tiêu

- Xây dựng kế hoạch cho repository SQLite cục bộ mà Go Native Messaging host dùng để lưu cấu hình theo profile, trạng thái cài đặt, metadata activation có giới hạn và diagnostics đã redact. SQLite là persistence boundary, không phải source of truth cho tab projection đang sống.
- Định nghĩa schema logic, khóa profile, access pattern, retention, migration transaction và lỗi có thể phục hồi để implementation không phải tự suy đoán.
- Giữ hot path lexical search và browser-owned activation hoạt động khi SQLite không mở được, migration thất bại, database bị hỏng hoặc database có schema tương lai chưa được host hỗ trợ.
- Làm cho reset và uninstall chỉ xóa dữ liệu/registration do InfoBoard sở hữu, có thể lặp lại an toàn và không đóng tab, không sửa browser history hoặc dữ liệu profile khác.
- Bàn giao các fixture và acceptance signal để chứng minh migration/quarantine có thể phục hồi, profile không dùng nhầm dữ liệu và failure của persistence không lan sang query/activation.

## 2. Phạm vi

- Bao gồm:
  - SQLite file cục bộ dưới runtime data directory do host sở hữu, với quyền file hạn chế, bounded busy/open timeout và không có network/loopback dependency.
  - Schema version đơn điệu, migration registry/checksum, `PRAGMA user_version` hoặc metadata tương đương, và nguyên tắc chỉ ghi version sau khi transaction commit thành công.
  - Bốn nhóm dữ liệu logic:
    - `configuration`: khóa `profile_id`, result limit, density, theme, context-label preference, protocol/ranking compatibility versions, cờ lưu activation recency và thời điểm cập nhật.
    - `installation_state`: một bản ghi host/install-owned cho extension/host/browser/protocol/ranking versions, host registration state, last successful migration và trạng thái repair/upgrade-required; không dùng bảng này làm tab projection.
    - `activation_metadata`: `profile_id`, opaque tab identity, domain đã chuẩn hóa, activated-at và source; không lưu query string, fragment, page body, cookie, token hoặc network payload.
    - `diagnostics`: thời điểm, error class, retryability, event/state, duration/count, protocol/ranking/schema versions và byte size của payload redacted; không ghi title/URL/query theo mặc định.
  - Retention bắt buộc: activation tối đa **500 bản ghi cho mỗi profile và 30 ngày**, xóa bản ghi cũ hơn trước rồi xóa oldest khi vượt count; diagnostics tối đa **7 ngày hoặc 10 MiB (`10 * 1024 * 1024` bytes), điều kiện nào đến trước**, prune sau mỗi append và lúc startup.
  - Profile isolation ở mọi read/write: `profile_id` phải là khóa/điều kiện bắt buộc cho configuration và activation; không có query hoặc activation metadata lookup xuyên profile.
  - Atomic migration, failed-migration rollback, unknown-future-version read-only/upgrade-required state, corruption detection/quarantine bằng rename an toàn và rebuild từ defaults.
  - Persistence-degraded mode: defaults và session-only recency trong memory; query index vẫn phục vụ lexical results; extension vẫn validate và activate tab qua browser API; lỗi chỉ trả status/diagnostics bounded.
  - Reset profile và reset/uninstall toàn cục theo ownership manifest: xóa bảng/rows/artifacts của InfoBoard trong transaction hoặc bước cleanup có ownership marker, không đụng browser state hay files ngoài manifest.
  - Test seams cho startup, migration, retention, profile boundaries, corruption, disk/lock failure, reset và recovery sau reconnect.
- Ngoài phạm vi:
  - Live tab projection, lexical index, ranking và query orchestration; các thành phần đó giữ state trong memory và thuộc các phase khác.
  - Browser API, tab activation call, Native Messaging framing/handshake và process lifecycle; phase này chỉ cung cấp persistence contract/status cho chúng.
  - Toàn bộ browsing history, page content, cookies, local storage, network interception, cloud service, telemetry bên ngoài máy và bất kỳ nguồn dữ liệu không có trong product contract.
  - Thiết kế UI settings/search, installer/manifest registration, hoặc thay đổi permission list; phase này chỉ nêu observable persistence states mà các phase đó phải hiển thị/xử lý.
  - Schema để cache projection sống: projection phải rebuild từ browser snapshot, không được biến SQLite thành nguồn sự thật thứ hai.

## 3. Điều kiện tiên quyết

- IP-02 đã chốt `profile_id`, tab identity, activation record, ownership extension/host/SQLite và privacy/retention boundary. Phase này không đổi các identity đó; nếu contract thay đổi phải cập nhật dependency trước khi code.
- IP-06 đã chốt Go host bootstrap, shutdown, cancellation, bounded resource policy và đường dẫn runtime data directory. Repository mở/đóng theo lifecycle đó, không tự tạo daemon, listener hoặc process mới.
- IP-07 cung cấp error/status mapping cho `PERSISTENCE_DEGRADED`, `PROTOCOL_MISMATCH`, `PROFILE_MISMATCH`, `HOST_SHUTDOWN` và request timeout. Storage errors phải map vào các status typed này mà không lộ đường dẫn, SQL, URL hoặc title.
- Các quyết định binding nằm trong [`persistence-and-lifecycle.md`](../refactor/persistence-and-lifecycle.md), [`architecture.md`](../refactor/architecture.md), [`domain-and-privacy.md`](../refactor/domain-and-privacy.md), [`requirements.md`](../refactor/requirements.md) và [`verification-and-acceptance.md`](../refactor/verification-and-acceptance.md); implementation chỉ thêm chi tiết không làm rộng data boundary.
- Go/SQLite driver, test harness temporary directory và fixture loader phải được chọn trong `host/` và `tests/` (hiện là `to-create`); input schema fixture không được lấy từ browser profile thật.

## 4. Đầu ra cần bàn giao

- `host/storage/` (to-create): package/repository boundary cho open, read/write configuration, installation state, append/prune activation, append/prune diagnostics, reset, close và health/status.
- `host/storage/schema.go` hoặc module tương đương (to-create): schema constants, table/index definitions, supported schema version, ownership manifest và bounded limits; tên file là target planning, không phải file hiện có.
- `host/storage/migrations/` (to-create): từng migration có version, precondition, transactional `up`, checksum và test rollback; không chạy migration unknown future version.
- `host/storage/recovery.go` (to-create): integrity check, corruption quarantine, fresh-default rebuild, upgrade-required state và recovery markers; quarantine artifact phải giữ lại để chẩn đoán và không bị xóa bởi reset ngoài ownership manifest.
- `fixtures/sqlite/` (to-create): SQL/JSON fixtures `SQL-001` đến `SQL-010` với expected status, row counts, retention boundary và ownership outcome.
- `tests/storage/` (to-create): focused unit/integration tests chạy trên temporary database, injected open/write/commit/integrity failures và profile pairs; không cần browser thật.
- Contract notes cho IP-09/IP-12/IP-14/IP-15/IP-16: query path nhận persistence health nhưng không phụ thuộc vào SQLite; activation ack không bị chặn bởi recency write failure; reset/uninstall có danh sách artifact rõ ràng.

## 5. Skill và tài liệu áp dụng

- Skill tags:
  - [`databases`](../../../.agent/skills/personal/engineering/data/databases/SKILL.md) — mô hình schema, access pattern, retention, migration, observability và validation cho workload local bounded.
  - [`go`](../../../.agent/skills/personal/engineering/backend/go/SKILL.md) — repository/interface boundary, cancellation, explicit errors, standard tooling và testable host implementation.
  - [`secure-development`](../../../.agent/skills/common/security/secure-development/SKILL.md) — trust boundary host–SQLite, ownership checks, input/size limits, redaction, fail-closed migration và safe deletion.
  - [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md) — test success, boundary, invalid, corruption, rollback, degraded behavior và reset ownership bằng observable outcomes.
  - [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md) — giữ schema/retention/recovery contract có thể đọc và kiểm chứng, link tới source of truth thay vì sao chép tùy tiện.
  - [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md) — chia work thành schema, migration, recovery, retention và acceptance seams theo dependency/risk.
  - [`git-workflow`](../../../.agent/skills/common/engineering/git-workflow/SKILL.md) — commit nhỏ, diff chỉ một phase file, kiểm tra path và đồng bộ branch an toàn.
- Tài liệu trong `docs/`:
  - [`persistence-and-lifecycle.md`](../refactor/persistence-and-lifecycle.md) — persistence boundary, bảng dữ liệu, retention, migration, startup, reset và uninstall.
  - [`architecture.md`](../refactor/architecture.md) — ownership của SQLite và failure isolation khỏi live projection/search.
  - [`domain-and-privacy.md`](../refactor/domain-and-privacy.md) — entity fields, profile isolation, trust boundary và dữ liệu không được lưu.
  - [`requirements.md`](../refactor/requirements.md) — FR-013/014/015, NFR-006/009 và operational requirements về version, timeout, idempotency, redacted logs.
  - [`verification-and-acceptance.md`](../refactor/verification-and-acceptance.md) — persistence failure, reset/uninstall ownership, privacy và recovery evidence.
- Quy ước code, ADR, context ngoài `docs/`:
  - [Project context](../../../CONTEXT.md) — Go Native Messaging host, in-memory hot path, bounded local data và không mở rộng data class.
  - Schema changes phải có version, checksum/identity ổn định, transaction boundary và compatibility note; không nhận arbitrary SQL/paths từ extension input.
  - `profile_id` là opaque local identity; không log nguyên giá trị khi không cần. Tất cả diagnostic payload phải redacted trước khi tính `payload_bytes` và insert.
  - Database path và quarantine path phải do path builder kiểm soát, không ghép với tab fields hoặc user query; thao tác xóa yêu cầu ownership marker.

## 6. Công việc triển khai

- [ ] Việc 1 — Chốt logical schema tại `host/storage/schema.go` (to-create):
  - `schema_meta`/`schema_migrations`: supported schema version, migration ID/checksum và committed-at; `user_version` chỉ tăng sau commit.
  - `configuration(profile_id PRIMARY KEY, result_limit, density, theme, show_context_labels, persist_activation_recency, protocol_version, ranking_model_version, updated_at)` với CHECK bounds và default an toàn.
  - `installation_state(installation_key PRIMARY KEY, extension_version, host_version, browser_family, protocol_version, ranking_model_version, host_registration_state, last_successful_migration, repair_state, updated_at)`; không chứa tab rows.
  - `activation_metadata(activation_id PRIMARY KEY, profile_id, tab_identity, domain, activated_at, source)` với index `(profile_id, activated_at DESC)` và bounded field lengths.
  - `diagnostics(diagnostic_id PRIMARY KEY, occurred_at, profile_id NULLABLE, event_kind, error_class, retryable, duration_ms, projection_revision, protocol_version, ranking_model_version, schema_version, payload_bytes, redacted_payload)` với index `(occurred_at)`; `profile_id` chỉ là opaque scope, không phải raw browser account identifier.
  - Không tạo bảng live projection, query history hoặc raw URL storage. Kiểm tra `foreign_keys`, journaling/atomicity và file permission policy theo platform mà không giả định một driver cụ thể.
- [ ] Việc 2 — Định nghĩa repository API và failure isolation tại `host/storage/repository.go` (to-create): mọi method nhận context/cancellation, profile scope và bounded input; trả typed result (`Ready`, `Degraded`, `UpgradeRequired`, `Quarantined/Rebuilt`) cùng retryability. Read config fallback defaults khi open/read fail; write failure không làm query/index hoặc browser activation fail.
- [ ] Việc 3 — Implement migration runner tại `host/storage/migrations/` (to-create): acquire bounded lock, validate current version, execute ordered migrations inside one transaction, verify postconditions, insert migration record và commit; rollback giữ database version trước đó khi bất kỳ step/checksum/postcondition nào fail. Migration retry phải idempotent và không ghi `last_successful_migration` trước commit.
- [ ] Việc 4 — Xử lý compatibility: nếu on-disk version lớn hơn supported version, không rewrite/drop/attempt downgrade; mở read-only hoặc đóng repository, trả upgrade-required và giữ file nguyên vẹn. Nếu protocol/ranking compatibility mismatch, giữ status riêng và không dùng dữ liệu version không hiểu cho ranking; lexical query vẫn dùng projection memory.
- [ ] Việc 5 — Xử lý corruption tại `host/storage/recovery.go` (to-create): bounded open và `PRAGMA integrity_check`/schema validation; khi corrupt, flush/close handle, atomic rename sang owned quarantine name có timestamp/nonce, tạo database mới từ schema/defaults, ghi recovery marker nếu có thể và trả trạng thái degraded/recovered. Không xóa quarantine tự động; nếu rename/create thất bại, giữ browser activation và lexical path hoạt động với session-only memory state.
- [ ] Việc 6 — Enforce retention bằng transaction nhỏ và deterministic order: activation prune theo `activated_at ASC, activation_id ASC` đến giới hạn 30 ngày và 500 row/profile; diagnostics prune theo thời gian rồi cumulative UTF-8 `payload_bytes` đến 7 ngày/10 MiB. Startup, append, graceful shutdown đều chạy bounded prune; prune error chỉ làm persistence degraded, không reject query/activation.
- [ ] Việc 7 — Implement reset/uninstall ownership tại `host/storage/reset.go` (to-create): profile reset xóa configuration/activation của đúng `profile_id` và các diagnostic rows được phép gắn profile; full reset/uninstall transactionally xóa mọi InfoBoard-owned table/state rồi cleanup generated host files/registration theo ownership manifest. Lệnh lặp lại là no-op thành công; lỗi giữa chừng không xóa browser tabs/history/unrelated files và phải báo recovery state.
- [ ] Việc 8 — Wire host lifecycle: open/migrate/load config trước `Ready`, close/flush bounded metadata khi shutdown, mark unavailable/recovering khi failure, và expose schema/migration/retention health qua IP-07/IP-16. Không block snapshot indexing, lexical query hoặc extension activation on durable write.
- [ ] Việc 9 — Tạo fixtures/tests cho baseline, migration, unknown future version, failed commit, corrupt file, both retention limits, profile isolation, storage outage, reset và uninstall. Assert observable statuses, row contents/counts, preserved quarantined file, unchanged browser-state fake and lexical/activation success; không assert driver-specific implementation details ngoài contract.
- [ ] Việc 10 — Ghi evidence/diagnostic redaction: verify title, full URL, query, cookie/token và SQL/path input không xuất hiện trong stored diagnostics/export; health chỉ nêu counts, durations, versions, revisions và error classes. Liên kết result với FR-013/014/015 và NFR-006/009 trong acceptance output.

## 7. Kế hoạch commit

1. `feat(storage): add bounded SQLite schema and repository boundary`
   - Thay đổi: tạo schema/version constants, ownership manifest, typed repository API, defaults, profile-scoped configuration/activation access và diagnostics byte accounting dưới `host/storage/`.
   - Cách kiểm tra: `cd /workspace/InfoBoard && go test ./host/storage/... -run 'TestSchema|TestProfileIsolation|TestRetention' -count=1` với `INFOBOARD_FIXTURE_ROOT=fixtures/sqlite`.
2. `feat(storage): add transactional migrations and recovery`
   - Thay đổi: migration runner, unknown-future guard, failed-transaction rollback, integrity check, quarantine/rebuild và degraded status.
   - Cách kiểm tra: `cd /workspace/InfoBoard && go test ./host/storage/... -run 'TestMigration|TestUnknownFuture|TestCorrupt|TestDegraded' -count=1` với temporary `INFOBOARD_DATA_DIR`.
3. `feat(storage): add reset and retention lifecycle`
   - Thay đổi: activation/diagnostics pruning, shutdown flush, profile/full reset và owned-artifact cleanup hooks.
   - Cách kiểm tra: `cd /workspace/InfoBoard && go test ./host/storage/... -run 'TestReset|TestUninstall|TestActivationRetention|TestDiagnosticsRetention' -count=1` với `INFOBOARD_FIXTURE_ROOT=fixtures/sqlite`.
4. `test(storage): cover persistence failure isolation`
   - Thay đổi: failure-injection integration fixtures chứng minh SQLite outage/degraded state không chặn lexical query hoặc browser-owned activation và migration/quarantine có đường phục hồi.
   - Cách kiểm tra: `cd /workspace/InfoBoard && go test ./tests/storage/... -run 'TestPersistenceFailureDoesNotBlockLexicalOrActivation' -count=1` với `INFOBOARD_FIXTURE_ROOT=fixtures/sqlite`.

## 8. Kiểm chứng và nghiệm thu

- [ ] Chạy từ repository root (sau khi các logical targets được tạo): `go test ./host/storage/... ./tests/storage/... -count=1` với `INFOBOARD_FIXTURE_ROOT="$PWD/fixtures/sqlite"` và temporary `INFOBOARD_DATA_DIR`; test không dùng database/profile thật.
- [ ] `SQL-001 baseline-v1`: mở database mới tạo đúng tables/indexes/defaults; `configuration` và activation của profile A không đọc được bởi profile B.
- [ ] `SQL-002 migration-v1-v2`: migration commit atomic, `schema_version`/last-successful marker chỉ đổi sau commit, retry không nhân đôi rows; `SQL-003 migration-failure` giữ file/version trước migration và trả degraded/retryable.
- [ ] `SQL-004 unknown-future`: database có version lớn hơn supported không bị rewrite/quarantine; status upgrade-required và file byte-for-byte còn nguyên.
- [ ] `SQL-005 corrupt-db`: integrity failure renames database thành owned quarantine artifact, rebuild defaults thành công và ghi trạng thái recoverable; quarantine vẫn tồn tại để inspect.
- [ ] `SQL-006 activation-retention`: sau append, rows cũ hơn 30 ngày bị loại và không quá 500 row/profile; thứ tự cắt deterministic tại timestamp/id boundary.
- [ ] `SQL-007 diagnostics-retention`: rows cũ hơn 7 ngày hoặc tổng redacted UTF-8 payload vượt 10 MiB bị prune, không xóa row trong cửa sổ nếu vẫn dưới byte cap; prune lặp lại an toàn.
- [ ] `SQL-008 profile-isolation`: hai profile có config/activation riêng, reset A không xóa B; không có query/activation operation trả row khác profile.
- [ ] `SQL-009 storage-outage`: injected open/read/write/commit failure trả persistence-degraded/defaults; cùng snapshot vẫn cho lexical results bình thường và result reference vẫn gửi được cho extension.
- [ ] `SQL-010 reset-owned`: profile reset/full reset/uninstall xóa đúng InfoBoard-owned rows/files, lặp lại không lỗi, không đóng tab, đổi history hoặc xóa unrelated sentinel file.
- [ ] Smoke flow input → observable output: start host với database bình thường, synchronize snapshot/index, query lexical, force SQLite unavailable, query lại và activate expected tab, restore storage, reconnect/rebuild; UI/health hiển thị degraded rồi recovered mà không silent stale activation.
- [ ] Đạt FR-013 (lexical search/activation survives durable-storage failure), FR-014 (configuration and access profile scoped), FR-015 (owned-only reset/uninstall), NFR-006 (recoverable migration/reconnect state) và NFR-009 (no excluded data persisted/transmitted); diagnostics không chứa raw title/URL/query/token.
- [ ] Không còn lỗi đã biết thuộc phase: unknown schema không bị downgrade, failed migration không partial-commit, corruption không phá hot path, retention không vượt bound, reset không vượt ownership manifest.

## 9. Rủi ro và quyết định còn mở

- Rủi ro: SQLite driver/platform journaling semantics, file locks hoặc abrupt process termination có thể làm open/commit behavior khác nhau giữa Linux, macOS và Windows.
  - Phương án xử lý đã chọn: dùng transaction + bounded timeout + reopen/integrity verification, test trên từng OS ở IP-19; không coi `Ready` là đạt trước khi schema/health đọc lại được.
- Rủi ro: database nhiều profile có thể làm retention hoặc reset hiểu nhầm là global.
  - Phương án xử lý đã chọn: configuration/activation retention và reset profile luôn yêu cầu `profile_id`; diagnostics/install state là host-owned và full reset/uninstall có scope explicit trong ownership manifest.
- Rủi ro: corruption quarantine có thể thất bại do permission/disk-full.
  - Phương án xử lý đã chọn: không overwrite file hỏng; chuyển sang in-memory defaults/session-only recency, expose degraded/repairable status và giữ activation/query path độc lập.
- Rủi ro: diagnostics 10 MiB có thể bị tính khác nhau nếu dựa trên SQLite page size.
  - Phương án xử lý đã chọn: tính `payload_bytes` trên redacted UTF-8 payload trước insert và prune theo tổng logical bytes; database file size là operational metric riêng, không dùng để mở rộng retention.
- Rủi ro: migration mới cần rewrite dữ liệu và rollback lâu hơn host timeout.
  - Phương án xử lý đã chọn: migration step bounded, không đọc arbitrary input, checkpoint/backup strategy do implementation ghi rõ; nếu vượt timeout thì rollback/degraded thay vì chạy không giới hạn.
- Câu hỏi còn mở chỉ khi câu trả lời có thể thay đổi contract: driver SQLite cụ thể và native filesystem atomic-rename primitive cho từng OS cần được chốt trong implementation/IP-19; không được thay đổi retention, ownership hoặc degraded-path semantics.

## 10. References ngoài `docs/`

- Skill: [`databases`](../../../.agent/skills/personal/engineering/data/databases/SKILL.md)
- Skill: [`go`](../../../.agent/skills/personal/engineering/backend/go/SKILL.md)
- Skill: [`secure-development`](../../../.agent/skills/common/security/secure-development/SKILL.md)
- Skill: [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md)
- Skill: [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md)
- Skill: [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md)
- Skill: [`git-workflow`](../../../.agent/skills/common/engineering/git-workflow/SKILL.md)
- Project context: [`CONTEXT.md`](../../../CONTEXT.md)
- Source/config/test path ngoài `docs/`: `host/storage/` (to-create), `host/storage/migrations/` (to-create), `host/storage/recovery.go` (to-create), `tests/storage/` (to-create)
- Fixture/tool/artifact ngoài `docs/`: `fixtures/sqlite/SQL-001` … `fixtures/sqlite/SQL-010` (to-create), `INFOBOARD_DATA_DIR` temporary database directory, Go `testing` package with injected SQLite failure seams (to-create)
