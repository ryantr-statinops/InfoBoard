# Phase 10 — Lexical normalization and index

> Plan ID: IP-10
> Status: not_started
> Execution owner: Go host lexical-index owner
> Dependencies: IP-02, IP-05, IP-09
> Parallel boundary: IP-11 and IP-12 consume this phase's index and normalized-query contracts; no shared implementation paths
> Requirement IDs: FR-003, FR-004, FR-005, FR-013, NFR-002, NFR-004, NFR-007 (supporting evidence; primary ownership remains with the requirement matrix)
> Owned paths: `host/index/` (to-create), `fixtures/lexical/` (to-create), `tests/index/` (to-create)

## 1. Mục tiêu

- Định nghĩa một pipeline thuần local từ eligible-tab projection hoặc query text tới searchable representation bounded, có provenance field và có kết quả lặp lại được. Cùng một projection, query, ranking timestamp và model/config version phải tạo cùng representation, không phụ thuộc thứ tự iteration của map.
- Chốt normalization Unicode, case folding, tokenization, field boundaries và display/search forms cho `title`, `domain`, `url`, `window_label`, `group_label`, `pinned`, `active` và các state labels. Stable browser IDs vẫn là metadata/tie-break, không biến thành text search tùy tiện.
- Lập kế hoạch cho immutable in-memory lexical index: build trên staging state, quan sát được trong trạng thái `rebuilding`, rồi atomic swap sang revision mới; không query một index nửa xây dựng và không trả dữ liệu âm thầm thuộc revision cũ.
- Cung cấp các fixture và test seams chứng minh empty projection/query, malformed hoặc overlong query, adversarial fields, deterministic rebuild, giới hạn tài nguyên và p95 query envelope 1,000 tabs/64-character query.
- Giữ lexical path hoạt động khi durable persistence không khả dụng. Phase này không biến persistence thành source of truth và không để optional failure làm thay đổi contract của index hiện tại.

## 2. Phạm vi

- Bao gồm:
  - Input contract cho một eligible tab thuộc đúng `profile_id` và `projection_revision`: tab/window/group identity, display fields, URL/domain, pinned/active/eligibility state, observed metadata và revision.
  - Normalization theo thứ tự: validate UTF-8; Unicode NFKC; full Unicode case folding; normalize whitespace/control handling; tokenize theo chữ/số, whitespace và punctuation; sau đó tạo field tokens và token positions/provenance. Không dùng ASCII-only lowercasing hoặc generic stop-word list.
  - URL/domain segmentation thuần string: giữ host/domain labels và các path/project segments hữu ích; scheme, punctuation và separators tạo boundary nhưng không làm mất toàn bộ domain term. Dùng `domain` từ projection khi có; URL parsing chỉ là parse cục bộ, không resolve, fetch hoặc inspect page.
  - Hai dạng dữ liệu tách biệt: bounded display form bảo toàn casing/ngôn ngữ cần hiển thị (đã sanitize control characters), và search form đã normalize/tokenize để matching. Full URL chỉ tồn tại transiently trong local index theo contract; result display không mặc nhiên lộ query string hoặc fragment.
  - Field map/provenance cho title, domain, URL, window label, group label và state labels (`pinned`/`unpinned`, `active`/`inactive`, `eligible` khi dùng cho invariant). `eligible=false` không được vào searchable postings; window/group IDs chỉ dùng identity/tie-break nếu thiếu label.
  - Bounded limits được kiểm tra trước khi tạo postings: đề xuất cố định trong contract implementation là title 512 Unicode scalars, URL 4096, domain 253, window/group label mỗi loại 128, tối đa 128 tokens/field, 512 tokens/record, query 64 scalars và 16 tokens. Tab count/result limit vẫn phải tuân protocol limits của IP-07/IP-09.
  - In-memory inverted postings theo field/token, record table theo stable tab identity, optional token positions cho phrase matching, và immutable snapshot metadata gồm `profile_id`, `projection_revision`, build status, model/normalizer version và counts. Postings phải có stable ordering trước khi publish.
  - Full build/rebuild từ snapshot authoritative và replacement atomic. Build detached state trước, validate mọi record, publish một lần khi thành công; empty snapshot cũng tạo index `ready` hợp lệ ở revision đó. Failure discard staging state, expose typed status/diagnostic và không phục vụ stale results cho revision mới.
  - Query preparation: whitespace-only/punctuation-only sau normalization là empty query; invalid UTF-8, disallowed control characters, unbalanced phrase delimiters hoặc overlong input trả lỗi validation bounded, không mutate index. Empty query chuyển candidate selection bounded cho ranking layer; empty projection luôn trả zero results.
  - Test seams cho Unicode equivalence/case folding, URL/domain segments, field/token truncation, duplicate IDs, missing optional labels, private/profile-scoped records đã được eligibility layer lọc, rebuild visibility, deterministic output, malformed input và no-external-I/O boundary.
- Ngoài phạm vi:
  - Browser API/event conversion, profile eligibility policy, snapshot/delta ordering và revision reconciliation; các contract đó thuộc IP-02, IP-04, IP-05 và IP-09. Phase này chỉ nhận snapshot contract đã được xác nhận.
  - Match scoring, lexical/context/recency weights, final ranking precedence, explanations và result API; IP-11/IP-12 sở hữu các policy đó, còn phase này chỉ cung cấp field provenance, tokens và deterministic candidates.
  - SQLite schema/repository, activation recency writes, protocol framing, UI rendering hoặc browser activation. Persistence failure được mô phỏng qua test seam, không được kéo storage dependency vào `host/index/`.
  - Network access, page-content reads/evaluation, shell/process execution, arbitrary SQL construction, remote services hoặc bất kỳ nguồn dữ liệu nào ngoài open-tab projection và contract metadata.

## 3. Điều kiện tiên quyết

- IP-02 đã chốt `profile_id`, stable tab/window/group identity, eligible-tab record, ownership boundaries và display/search privacy rules. Nếu field hoặc identity contract thay đổi, phải cập nhật input fixture trước khi implementation.
- IP-05 cung cấp full snapshot, field availability, profile isolation và monotonic `projection_revision`; một snapshot mới là authority, không được merge tùy tiện với record của profile khác.
- IP-09 cung cấp sequence `snapshot -> build -> ready`, delta/resync/rebuild behavior và rejection khi revision unknown hoặc rebuilding. Index phải expose đủ status để protocol/query layer map được `INDEX_REBUILDING`, `REVISION_MISMATCH` hoặc safe internal failure.
- Các quyết định binding cần đọc: [`search-and-ranking.md`](../refactor/search-and-ranking.md), [`domain-and-privacy.md`](../refactor/domain-and-privacy.md), [`architecture.md`](../refactor/architecture.md), [`runtime-protocol.md`](../refactor/runtime-protocol.md), [`requirements.md`](../refactor/requirements.md) và [`verification-and-acceptance.md`](../refactor/verification-and-acceptance.md).
- Repository hiện chưa có source roots lexical tương ứng; `host/index/`, `fixtures/lexical/` và `tests/index/` vẫn là `to-create`. Khi scaffold thực tế xuất hiện trước implementation, reread exact package paths và cập nhật plan trước commit code.
- Primary ownership không bị thay đổi bởi phase này: query-to-render và bounded query contract thuộc IP-12; ranking determinism thuộc IP-11; no-excluded-data/privacy enforcement thuộc IP-17. Phase 10 chỉ bàn giao observable transformation/index evidence.

## 4. Đầu ra cần bàn giao

- `host/index/` (to-create): package boundary cho validated field values, Unicode normalizer/case folder, tokenizer, bounded query parser, field provenance và immutable in-memory lexical index; package không import browser, storage, network hoặc process-control code.
- `host/index/normalization.go` hoặc module tương đương (to-create): versioned normalization policy, display/search forms, scalar/token limits, malformed-input outcome và safe field truncation. Tên file là target planning, không phải file hiện có.
- `host/index/index.go` hoặc module tương đương (to-create): record table, postings, build/rebuild transaction, revision/status publication, empty-index behavior và deterministic iteration/tie metadata.
- `host/index/query.go` hoặc module tương đương (to-create): bounded query preparation, empty-query representation, phrase-boundary validation và typed non-sensitive validation outcome; không thực hiện scoring/ranking policy.
- `fixtures/lexical/` (to-create): stable JSON fixtures cho Unicode/case, URL-domain segmentation, every bounded field, state/provenance, empty projection/query, malformed/overlong query, duplicate/rebuild and 1,000-tab benchmark input. Fixture expected output phải gồm normalized digest/tokens, index counts/status hoặc explicit error class.
- `tests/index/` (to-create): focused unit/property-style tests for normalization and token bounds, integration tests for atomic rebuild and revision visibility, deterministic repeatability tests, and a benchmark/threshold harness for NFR-002.
- A short implementation handoff in package comments/test names documenting normalizer version and the invariant that index output is local, bounded, deterministic and rebuildable from the authoritative projection.

## 5. Skill và tài liệu áp dụng

- Skill tags:
  - [`databases`](../../../.agent/skills/personal/engineering/data/databases/SKILL.md): model bounded field/posting access patterns and keep live projection separate from durable storage.
  - [`pipelines`](../../../.agent/skills/personal/engineering/data/pipelines/SKILL.md): define snapshot input, normalization stages, lineage/provenance, idempotent rebuild and quarantine of invalid records.
  - [`go`](../../../.agent/skills/personal/engineering/backend/go/SKILL.md): implement explicit Go packages, Unicode handling, bounded allocations, cancellation and observable build status.
  - [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md): cover normal, boundary, malformed, deterministic and failure behavior with focused fixtures and benchmarks.
  - [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md): keep the standalone contract, commands, links and observable acceptance signals executable for a later implementer.
  - [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md): sequence normalization, index publication, tests and performance checks around the IP-02/IP-05/IP-09 interfaces.
  - [`git-workflow`](../../../.agent/skills/common/engineering/git-workflow/SKILL.md): preserve the one-phase-file plan commit and the future small implementation commits.
- Tài liệu trong `docs/`: [`refactor/README.md`](../refactor/README.md), [`refactor/search-and-ranking.md`](../refactor/search-and-ranking.md), [`refactor/domain-and-privacy.md`](../refactor/domain-and-privacy.md), [`refactor/architecture.md`](../refactor/architecture.md), [`refactor/runtime-protocol.md`](../refactor/runtime-protocol.md), [`refactor/requirements.md`](../refactor/requirements.md), [`refactor/verification-and-acceptance.md`](../refactor/verification-and-acceptance.md), [`refactor/roadmap.md`](../refactor/roadmap.md) và [`implementation/index.md`](index.md).
- Quy ước code, ADR, context ngoài `docs/`: [`CONTEXT.md`](../../../CONTEXT.md); target roots `host/`, `fixtures/` và `tests/` là logical `to-create` paths, còn exact package names must follow IP-01's module map.

## 6. Công việc triển khai

- [ ] Chốt normalizer version và field limits trong one shared contract. Implement valid UTF-8 checking, NFKC, full Unicode case folding, control/whitespace policy and scalar-safe truncation before token allocation; return a bounded typed outcome for invalid input.
- [ ] Implement separate display/search forms. Preserve sanitized display text and normalized search text independently; retain field name, token positions and source provenance without retaining unbounded raw input.
- [ ] Implement field-aware tokenization. Split title/labels on Unicode whitespace/punctuation; emit domain labels and URL host/path segments; preserve meaningful separators as boundaries; do not apply a generic stop-word list; keep state labels explicit and stable.
- [ ] Define and enforce the field table: title, URL, domain, window label, group label, pinned, active, eligibility. Ensure missing optional fields produce no fabricated token, while `pinned`/`active` states remain queryable and `eligible=false` cannot produce postings.
- [ ] Implement staged index build from one authoritative snapshot. Validate profile and revision at the boundary, reject duplicate stable IDs deterministically, build sorted postings/record metadata off to the side, publish one immutable ready state, and expose counts plus normalizer/model versions.
- [ ] Implement rebuild transitions and failure behavior. Emit `rebuilding` before work, reject queries for the incoming unknown/rebuilding revision, atomically publish `ready` or a safe failure state, and make an empty snapshot observable as `ready` with zero records rather than as host failure.
- [ ] Implement bounded query preparation. Treat empty and whitespace-only queries equivalently; reject malformed UTF-8/control/phrase-delimiter input and overlong queries without changing the current index; return normalized tokens and phrase boundaries for IP-11/IP-12.
- [ ] Add fixture IDs and expected outputs: `FX-LEX-UNICODE-CASE`, `FX-LEX-URL-SEGMENTS`, `FX-LEX-FIELD-BOUNDS`, `FX-LEX-STATE-PROVENANCE`, `FX-LEX-EMPTY`, `FX-LEX-MALFORMED`, `FX-LEX-REBUILD`, `FX-LEX-DETERMINISTIC`, `FX-LEX-1000-TABS`. Register these IDs in the shared fixture catalog owned by IP-01 rather than creating a second fixture naming system.
- [ ] Add tests that repeat the same projection/query several times and compare normalized representation, postings digest, candidate IDs and rebuild status; assert map insertion order and input record order cannot change output.
- [ ] Add a package-boundary check that the lexical index has no network client, SQL/storage, browser API, page-evaluation or process-execution dependency. Exercise the optional persistence failure seam and prove the in-memory lexical result remains available.
- [ ] Add benchmark data for 1,000 eligible tabs and a 64-scalar query. Measure p50/p95/p99 and allocations for normalization, build and query separately; fail the query-to-render harness when p95 exceeds 50 ms on the declared reference environment.

## 7. Kế hoạch commit

1. `feat(host-index): add bounded lexical normalization`
   - Thay đổi: create the normalization policy, field limits, display/search forms, tokenizer, query validation and Unicode/URL fixtures under the planned `host/index/` and `fixtures/lexical/` boundaries.
   - Cách kiểm tra: from repository root, run `go test ./host/index -run 'TestNormalize|TestTokenize|TestPrepareQuery|TestFieldBounds' -count=1` and compare fixture digests for `FX-LEX-UNICODE-CASE`, `FX-LEX-URL-SEGMENTS`, `FX-LEX-FIELD-BOUNDS` and `FX-LEX-MALFORMED`.
2. `feat(host-index): publish atomic lexical rebuilds`
   - Thay đổi: create immutable record/posting state, snapshot build/rebuild transitions, empty projection handling, revision/status checks and deterministic publication.
   - Cách kiểm tra: run `go test ./host/index -run 'Test(IndexBuild|Rebuild|EmptyProjection|RevisionVisibility|Deterministic)' -count=1 -v` with `fixtures/lexical/FX-LEX-REBUILD.json` and assert `rebuilding -> ready`, zero-record empty readiness and identical digests.
3. `test(host-index): cover bounded query performance and isolation`
   - Thay đổi: add `tests/index/` integration/benchmark coverage, no-external-I/O boundary checks and persistence-degraded lexical-path fixture.
   - Cách kiểm tra: run `go test ./tests/index -run 'Test(Deterministic|NoExternalIO|PersistenceDegraded|Adversarial)' -count=1`; run the 1,000-tab benchmark command in section 8 and require the documented p95 threshold.

## 8. Kiểm chứng và nghiệm thu

- [ ] From repository root, with `fixtures/lexical/` loaded and no network services or external credentials, run `go test ./host/index -run 'TestNormalize|TestTokenize|TestPrepareQuery|TestFieldBounds|TestIndexBuild|TestRebuild|TestEmptyProjection|TestRevisionVisibility|TestDeterministic' -count=1 -v`. Expected: all fixture expectations pass; no input produces an unbounded field/token/posting count.
- [ ] Run `go test ./tests/index -run 'TestLexicalFixtures|TestNoExternalIO|TestPersistenceDegradedPath' -count=1 -v` from repository root. Expected: `FX-LEX-UNICODE-CASE` and equivalent canonically composed/decomposed inputs produce the same search representation; `FX-LEX-DETERMINISTIC` produces the same normalized digest, postings digest and ordered candidate IDs on repeated runs; optional persistence failure still returns lexical candidates.
- [ ] Run `go test ./tests/index -run TestLexicalQueryP95 -bench '^BenchmarkQuery1000Tabs$' -benchmem -count=5 -args -fixture fixtures/lexical/FX-LEX-1000-TABS.json -query-runes 64 -p95-ms 50`. Expected: the harness reports p50/p95/p99 and exits non-zero if query-to-render p95 exceeds 50 ms on the declared reference machine; record allocations and index size in the evidence artifact.
- [ ] Run the malformed-input cases with `go test ./host/index -run 'TestPrepareQuery/(invalid_utf8|control|unbalanced_phrase|overlong|punctuation_only)' -count=1`. Expected: typed, non-sensitive validation for invalid/overlong input; punctuation-only and whitespace-only input is the empty-query representation; the existing index and revision remain unchanged.
- [ ] Run the rebuild scenario using `fixtures/lexical/FX-LEX-REBUILD.json`: submit revision R1, start R2 rebuild, query R2 while rebuilding, then complete R2. Expected observable sequence is `ready(R1) -> rebuilding(R2) -> INDEX_REBUILDING/REVISION_MISMATCH for R2 query -> ready(R2)`; no partial R2 postings or silent R1 result is returned.
- [ ] Verify field bounds with a fixture containing overlong Unicode scalars, long URL/domain segments, repeated tokens and missing labels. Expected: scalar-safe bounded representation, deterministic truncation/error marker, no panic, no fabricated fields, and stable provenance for every emitted token.
- [ ] Verify empty projection with `FX-LEX-EMPTY`: authoritative empty snapshot builds `ready` at its supplied revision; empty and non-empty queries return zero result candidates; the host remains available for the next snapshot.
- [ ] Verify NFR-007 by package import/dependency review and `TestNoExternalIO`: lexical normalization/index code performs no network access, SQL interpolation/storage calls, page evaluation, browser API call or process execution. A URL string is parsed only as local text.
- [ ] Acceptance signal: for every repeated projection/query pair, normalized representation and index digest are byte-for-byte equal; all fields/tokens/results stay within declared bounds; rebuild status is externally observable; and lexical search remains available when optional durable persistence fails. This phase supplies evidence for FR-003/FR-004/FR-013 and NFR-002/NFR-007; IP-11/IP-12/IP-17 retain final primary acceptance ownership for ranking, query API and privacy.

## 9. Rủi ro và quyết định còn mở

- Rủi ro: Unicode case folding or NFKC can change token boundaries and may produce different output across library versions. Phương án xử lý đã chọn: pin the normalizer policy/version, record it in the index metadata, include composed/decomposed, case-variant and non-ASCII fixtures, and reject a model/version mismatch rather than silently mixing representations.
- Rủi ro: truncating a URL/title in the middle of a multi-byte sequence or token can create invalid display/search data. Phương án xử lý đã chọn: bound by Unicode scalars before tokenization, sanitize display controls, and make truncation deterministic and visible to diagnostics without logging raw content.
- Rủi ro: a full rebuild for a large projection can expose stale results or block the query path. Phương án xử lý đã chọn: build detached immutable state, publish atomically, reject the rebuilding revision, and measure 1,000-tab build/query allocations before release.
- Rủi ro: URL punctuation and labels can either destroy useful domain terms or create noisy postings. Phương án xử lý đã chọn: field-aware segmentation with explicit domain labels and URL path segments, no generic stop words, provenance per emitted token, and fixed fixture expectations.
- Rủi ro: malformed or adversarial input can cause excess allocations or diagnostic leakage. Phương án xử lý đã chọn: validate and bound before allocation, return safe error classes, avoid raw field/query logging, and quarantine only the invalid record/query without weakening profile or revision checks.
- Câu hỏi còn mở chỉ khi câu trả lời có thể thay đổi contract: the exact Go Unicode package/version and the final numeric limits must be frozen with IP-07 protocol limits and IP-09 rebuild limits before implementation; changing either requires fixture digest/version updates, not a silent compatibility change.

## 10. References ngoài `docs/`

- Skill: [`databases`](../../../.agent/skills/personal/engineering/data/databases/SKILL.md), [`pipelines`](../../../.agent/skills/personal/engineering/data/pipelines/SKILL.md), [`go`](../../../.agent/skills/personal/engineering/backend/go/SKILL.md), [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md), [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md), [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md), [`git-workflow`](../../../.agent/skills/common/engineering/git-workflow/SKILL.md)
- Project context: [`CONTEXT.md`](../../../CONTEXT.md)
- Source/config/test path ngoài `docs/`: `host/index/` (to-create), `tests/index/` (to-create), `fixtures/lexical/` (to-create)
- Fixture/tool/artifact ngoài `docs/`: `fixtures/lexical/FX-LEX-*.json` (to-create), `go test ./host/index`, `go test ./tests/index`, and the 1,000-tab benchmark command in section 8 (all future targets until the source scaffold exists)