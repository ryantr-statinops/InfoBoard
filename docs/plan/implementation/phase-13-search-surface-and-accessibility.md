# Phase 13 — Search surface and accessibility

> Plan ID: IP-13
> Status: not_started
> Execution owner: extension UI owner
> Dependencies: IP-03, IP-12
> Parallel boundary: IP-14, IP-15, IP-16 after their listed dependencies
> Requirement IDs: FR-002, FR-006, FR-008, NFR-001, NFR-008; supporting FR-004, FR-012
> Owned paths: `extension/search/` (to-create), `extension/manifest.json` command handoff integration (to-create), `tests/browser/search-surface/` (to-create), `tests/accessibility/search-surface/` (to-create), `fixtures/ui/` (to-create)

## 1. Mục tiêu

- Xây dựng focused search surface do extension sở hữu, mở từ browser command configurable và đặt con trỏ trong query input trước khi người dùng gõ.
- Cung cấp hot path chỉ dùng bàn phím: shortcut → surface mở và focus → nhập query → kết quả cập nhật theo từng query revision → mũi tên đổi lựa chọn → `Enter` bàn giao activation → `Esc` đóng mà không làm đổi tab đã chọn.
- Hiển thị rõ title, domain hoặc URL dạng gọn, window/group context khi cần phân biệt, pinned state và trạng thái lựa chọn; không làm lộ query string có khả năng chứa bí mật theo mặc định.
- Biểu diễn loading/opening, empty query, results, no-result, runtime unavailable, rebuilding/stale và activation/error states theo cách có thể hành động, không khẳng định activation nếu browser chưa xác nhận.
- Đạt accessibility baseline cho keyboard, visible focus, tên semantic, screen reader, contrast/non-color cues, reduced motion, text scaling và cửa sổ hẹp.

## 2. Phạm vi

- Bao gồm:
  - Entry point của command từ Manifest V3 và handoff tới focused extension surface; surface phải được focus hoặc đưa lên trước nếu đã tồn tại, nhưng không tái sử dụng query/result của phiên trước một cách im lặng.
  - Semantic structure cho dialog/surface, query input, result list/listbox, result row/option, status/live region, retry/dismiss actions và privacy/status affordance.
  - Query controller gửi query bounded tới contract IP-12 trên mỗi accepted input revision, hủy hoặc bỏ qua response cũ, giữ revision/projection status cần cho render và activation handoff.
  - Keyboard model: focus input khi mở; `ArrowDown`/`ArrowUp` đổi lựa chọn; `Home`/`End` nếu supported phải giữ thứ tự deterministic; `Enter` yêu cầu extension-owned activation handoff; `Esc` đóng surface; tab order và focus trap chỉ áp dụng cho controls thực sự hiện diện.
  - Bounded result rendering với context và explanation ngắn từ query API; empty query dùng recent/context list bounded theo contract, không render toàn bộ projection.
  - UI state machine cho opening/loading, empty query, results, no results, rebuilding, runtime unavailable, stale result, persistence-degraded, activation failure/success và private-context unavailable; mỗi lỗi phải nêu điều thất bại, browser state có đổi hay chưa và hành động kế tiếp.
  - Responsive CSS/layout cho narrow window, text zoom lớn, high contrast/contrast baseline, visible focus và `prefers-reduced-motion`; không để title/query bị cắt tới mức không thể thao tác.
  - Browser fixture và accessibility/performance seams để chứng minh keyboard-only journey, names/roles/states, responsive behavior và p95 shortcut-to-focus ≤100 ms trên reference hardware.
- Ngoài phạm vi:
  - Không sở hữu ranking, normalization, projection reconciliation, protocol framing, host lifecycle, durable storage, browser event conversion hoặc tab activation API; các boundary này thuộc IP-03/IP-04/IP-07/IP-09/IP-10/IP-11/IP-12/IP-14.
  - Không index hoặc hiển thị page content, historical browsing records, cookies, local storage, network data hay dữ liệu ngoài open-tab metadata được phép.
  - Không thêm browser/OS ngoài Chrome và Edge desktop trên Linux, macOS, Windows; không tạo content-script overlay hay navigation thay thế trên current tab.
  - Không tự gửi request mạng, mở transport riêng, thực hiện SQL, tự suy đoán result stale là hợp lệ, hoặc dùng màu sắc duy nhất để biểu diễn state.
  - Không thay đổi product contract của result limit, profile isolation, protocol error literals hoặc activation validation; thay đổi contract phải đi qua tài liệu refactor và phase sở hữu tương ứng.

## 3. Điều kiện tiên quyết

- IP-03 cung cấp Manifest V3 skeleton, browser command configurable, focused-surface entrypoint, browser adapter seam và permission set tối thiểu. Command handoff phải có cách truyền profile/surface context mà không cấp thêm quyền ngoài contract.
- IP-12 cung cấp query request/result contract: bounded limit, query revision, result references, projection revision, runtime/index status, no-result, stale/rebuilding, host-down và persistence-degraded responses. UI chỉ render dữ liệu đã validate theo contract.
- Canonical product behavior là [user experience contract](../refactor/user-experience.md), [requirements](../refactor/requirements.md), [architecture](../refactor/architecture.md), [runtime protocol](../refactor/runtime-protocol.md), [search and ranking](../refactor/search-and-ranking.md) và [verification and acceptance](../refactor/verification-and-acceptance.md).
- Source tree hiện chưa có; các path dưới `extension/`, `tests/` và `fixtures/` đều là `to-create`. Khi implementation scaffold xuất hiện, phải map lại target path và symbol thật trước khi thực hiện phase.
- Browser test harness phải có Chrome/Edge adapters or equivalently observable fixtures, hỗ trợ mở command, kiểm tra cửa sổ surface, gửi keyboard events, đọc focus/ARIA tree và ghi timestamps. Accessibility runner phải kiểm tra role/name/state thay vì chỉ snapshot HTML.

## 4. Đầu ra cần bàn giao

- `extension/search/` (to-create): module surface controller, query input/controller, result list/row, status/error presentation, keyboard/focus policy và activation handoff adapter; module không chứa ranking hoặc browser API policy.
- `extension/search/index.html` hoặc tương đương (to-create): semantic focused surface shell với title, query field, list/status regions, dismiss/retry affordances và privacy/status copy.
- `extension/search/search.css` hoặc tương đương (to-create): bounded responsive layout, visible focus, selected-state non-color cues, reduced-motion rules, text scaling và narrow-window rules.
- `extension/manifest.json` command wiring (to-create): command name/description and surface entrypoint integration owned by IP-03; this phase supplies the UI handoff contract and verifies it without adding permissions.
- `fixtures/ui/` (to-create): deterministic fixtures `UI-SHORTCUT-FOCUS`, `UI-LIVE-QUERY`, `UI-ARROW-ENTER`, `UI-ESC-DISMISS`, `UI-EMPTY`, `UI-NO-RESULT`, `UI-OPENING`, `UI-REBUILDING`, `UI-HOST-DOWN`, `UI-PERSISTENCE-DEGRADED`, `UI-STALE`, `UI-ACTIVATION-FAILURE`, `UI-PRIVATE-UNAVAILABLE`, `UI-DUPLICATE-CONTEXT`, `UI-NARROW-TEXT-SCALE` with expected observable states.
- `tests/browser/search-surface/` and `tests/accessibility/search-surface/` (to-create): focused browser journeys, accessibility assertions, responsive evidence and latency measurement artifacts.
- A handoff record from the implementation commit identifying browser/host contracts consumed, exact fixture outcomes, p95 measurement environment, and any explicit compatibility exception.

## 5. Skill và tài liệu áp dụng

- Skill tags:
  - `common/foundation/task-planning` — sequence the UI state machine, public seams, fixture contracts and measurable acceptance before implementation.
  - `common/engineering/git-workflow` — keep the phase commit scoped to this plan and verify path/link/diff boundaries.
  - `personal/engineering/frontend/lightweight-web` — choose semantic HTML, minimal UI dependencies, fast feedback and a maintainable narrow responsive surface.
  - `common/workflow/feature-delivery` — deliver the keyboard-first flow vertically from command handoff through query/render/dismiss and review its observable outcome.
  - `common/engineering/testing` — cover successful keyboard journeys plus no-result, stale, runtime, responsive and accessibility boundaries with deterministic fixtures.
  - `common/engineering/documentation` — record the contract, setup assumptions, exact commands, observable evidence and limitations for a future implementer.
- Tài liệu trong `docs/`:
  - [Product contract and boundaries](../refactor/README.md)
  - [Requirements](../refactor/requirements.md) — primary FR/NFR acceptance signals and permission boundary.
  - [User experience](../refactor/user-experience.md) — surface journey, row content, UI states, configuration and accessibility rules.
  - [Architecture](../refactor/architecture.md) — extension ownership and local data flow.
  - [Runtime protocol](../refactor/runtime-protocol.md) — result revision/status, bounded responses and activation ownership.
  - [Search and ranking](../refactor/search-and-ranking.md) — normalized result fields, explanations and deterministic empty-query behavior.
  - [Verification and acceptance](../refactor/verification-and-acceptance.md) — keyboard, responsive, accessibility and performance evidence.
- Quy ước code, ADR, context ngoài `docs/`:
  - `CONTEXT.md` defines the Chromium desktop boundary, hot path, privacy limits, local-only behavior, test seams and `to-create` source layout.
  - Use semantic browser controls and native keyboard events before adding a UI library. Keep render state explicit and make stale response rejection observable through the query revision/projection revision.
  - Do not invent a framework, component library, persistence field or permission. If the browser API differs between Chrome and Edge, keep the adapter difference at IP-03/IP-04 boundary and preserve this surface contract.

## 6. Công việc triển khai

- [ ] Việc 1: Define the surface state machine and public seams in `extension/search/` (to-create): opening → ready/empty-query → querying → results/no-results, with explicit rebuilding, unavailable, stale, persistence-degraded and activation-failure branches. Each state exposes permitted actions and whether browser state is confirmed unchanged.
- [ ] Việc 2: Implement command handoff from IP-03: open/focus the extension-owned surface, clear prior query/result state, focus the query input, and record command-start/focus timestamps for NFR-001. The handoff must tolerate an already-open surface and service-worker restart without creating duplicate windows.
- [ ] Việc 3: Implement query input behavior: trim/accept input according to IP-12, issue a bounded request for every accepted revision, render only the matching response, and suppress late responses from an older revision or projection. Empty/whitespace input renders bounded recent/context results with explanatory status.
- [ ] Việc 4: Render result rows from IP-12's bounded display metadata: title as accessible primary label; redacted domain/compact URL; disambiguating window/group context; pinned state; compact match explanation when safe; selected state and stable result reference. Never expose raw URL query strings by default.
- [ ] Việc 5: Implement keyboard and focus semantics: input remains the predictable typing target; arrow keys update `aria-activedescendant` or an equivalent selected option; `Enter` invokes the IP-14 activation handoff with result ID and projection revision; `Esc` dismisses without activation; explicit close/dismiss controls are keyboard reachable and return focus according to the surface lifecycle.
- [ ] Việc 6: Implement visible and semantic accessibility: stable labels/descriptions for input, result list, each row, selection, pin/context, loading, no-result, runtime failure, stale state and activation outcome; live-region announcements must be concise and must not include sensitive URL/query data. Selection, pinning, freshness and errors need text/icon/structure in addition to color.
- [ ] Việc 7: Implement responsive and preference rules: surface remains operable at the narrow-window fixture dimensions and browser text scaling; content wraps or truncates with accessible full labels; focus rings remain visible; `prefers-reduced-motion: reduce` removes nonessential transitions; no animation is required for correctness.
- [ ] Việc 8: Add deterministic fixtures and tests for `UI-SHORTCUT-FOCUS` (command opens, focus is input, no stale flash), `UI-LIVE-QUERY` (each revision maps to rendered revision), `UI-ARROW-ENTER` (expected result handoff), `UI-ESC-DISMISS` (browser state unchanged), duplicate-title context, empty/no-result, opening/loading, rebuilding, host-down, persistence-degraded, stale, activation failure, private unavailable, narrow width, text scaling, reduced motion and screen-reader semantics.
- [ ] Việc 9: Measure shortcut-to-focus from command dispatch to `document.activeElement`/equivalent focused input on supported reference hardware. Capture p50/p95/p99, browser/OS/profile state and sample count; fail acceptance when p95 exceeds 100 ms in a healthy extension run. Measure query-to-render only as supporting evidence; NFR-002 remains owned by IP-12.
- [ ] Việc 10: Verify that this phase requests no extra permission, performs no network access, does not mutate tabs on typing/dismissal, and leaves activation confirmation and stale-result validation to IP-14.

## 7. Kế hoạch commit

1. `feat(extension-search): implement focused keyboard search surface`
   - Thay đổi: create the semantic surface/controller, command handoff seam, input/revision flow, bounded result rows, state rendering, keyboard/focus handling, responsive styles and accessibility labels under the phase-owned `to-create` paths.
   - Cách kiểm tra: from repository root, run the focused browser fixture command `go test ./tests/browser/search-surface -run 'TestShortcutFocus|TestLiveQuery|TestKeyboardSelection|TestEscDismiss'` (or the repository's equivalent once the chosen harness exists), then exercise `UI-SHORTCUT-FOCUS` → `UI-LIVE-QUERY` → `UI-ARROW-ENTER` → `UI-ESC-DISMISS` in Chrome and Edge fixture runs.
2. `test(extension-search): cover accessibility and responsive states`
   - Thay đổi: add accessibility, state-boundary, narrow-window/text-scaling, reduced-motion, stale/host-down/persistence-degraded and latency evidence tests under `tests/accessibility/search-surface/` and `fixtures/ui/`.
   - Cách kiểm tra: from repository root, run `go test ./tests/accessibility/search-surface -run 'TestAccessibleNames|TestVisibleFocus|TestReducedMotion|TestNarrowWindow'` (or the selected UI harness equivalent), run the accessibility audit for the focused surface, and record the shortcut-to-focus p95 artifact. These command paths are future implementation targets because the source tree is currently `to-create`.

## 8. Kiểm chứng và nghiệm thu

- [ ] Chạy lệnh exact với working directory, fixture và env rõ ràng: from repository root, run the implementation-specific browser fixture command against `fixtures/ui/UI-SHORTCUT-FOCUS` with a supported Chrome and Edge desktop profile; run the accessibility suite against the rendered surface. Until the harness exists, preserve the command shape and fixture IDs above rather than substituting an unbounded generic test command.
- [ ] Verify keyboard-only journey: dispatch the configured command with no pointer; observe focused query input before text entry; type a query; observe the matching result revision; use arrow keys; press `Enter`; observe only the explicitly selected tab/window activation confirmation; reopen and press `Esc`; observe surface dismissal with browser tab/window state unchanged.
- [ ] Verify live-query and stale protection: rapid revisions render only the newest accepted response; rebuilding/stale/unknown revision is visibly marked and cannot silently activate a different tab; host-down and persistence-degraded states remain actionable and do not claim unavailable results are current.
- [ ] Verify result comprehension: duplicate-title fixture exposes title plus enough domain/window/group context; pinned state and selection are conveyed without color-only dependence; display labels redact likely secret URL query strings.
- [ ] Verify accessibility: automated role/name/state audit passes; keyboard order and visible focus are present; selected option, loading, no-result, errors, stale status and activation outcome are announced with safe labels; contrast and reduced-motion checks pass; browser text scaling preserves input and selected result.
- [ ] Verify responsive evidence at the chosen narrow-window viewport and at the supported text scale. Record screenshots or structured accessibility snapshots plus browser/OS/browser-version metadata; no horizontal clipping may make query, result selection or dismiss unavailable.
- [ ] Verify performance: on declared reference hardware and healthy extension, collect enough repeated command launches to report p50/p95/p99; p95 shortcut dispatch → focused input is ≤100 ms (NFR-001). Report warm/cold or extension-start conditions separately rather than hiding a slow case.
- [ ] Verify permissions/privacy boundary: manifest still contains only the approved command/tab/window/group/native-messaging permissions; no request or resource fetch leaves the local product path; typing, rendering and dismissal do not change browser state.
- [ ] Đạt FR-002, FR-006, FR-008, NFR-001 and NFR-008 acceptance signals, with supporting FR-004 and FR-012 evidence; no unresolved in-scope UI, accessibility, state, or measurement defect remains.

## 9. Rủi ro và quyết định còn mở

- Rủi ro: Chrome and Edge may differ in focused-window behavior, command event timing, accessible tree details or extension window APIs. Phương án xử lý đã chọn: keep browser-specific behavior in the IP-03 adapter, run the same fixture contract in both browsers, and record an explicit compatibility exception rather than weakening keyboard/focus behavior.
- Rủi ro: service-worker suspension or a slow host can delay first results after the surface is focused. Phương án xử lý đã chọn: focus input independently of query readiness, show opening/loading or runtime status, keep stale results out of activation, and provide retry/dismiss actions. The 100 ms target measures shortcut-to-focus, not host query completion.
- Rủi ro: delayed query responses can overwrite newer results. Phương án xử lý đã chọn: bind each request and render to query revision plus projection revision; discard mismatches and show rebuilding/stale state when the contract requires resync.
- Rủi ro: small windows, large text or long localized labels can hide controls. Phương án xử lý đã chọn: use a responsive single-column layout, wrapping/truncation with accessible full labels, bounded scrolling and keyboard access independent of visual width.
- Rủi ro: live-region announcements may expose a sensitive URL or query. Phương án xử lý đã chọn: announce counts, safe titles only when allowed, context state and next action; redact URL query strings and never include raw diagnostics.
- Rủi ro: “focus” can be measured inconsistently across automation and OS window activation. Phương án xử lý đã chọn: define start/end timestamps in the fixture (command dispatch to confirmed focused input), record browser/OS/hardware and foreground-window preconditions, and report both instrumented and observed traces.
- Câu hỏi còn mở chỉ khi câu trả lời có thể thay đổi contract: none. Any proposed change to command behavior, result content, permissions, privacy, activation validation or supported browser matrix must update the owning refactor document before implementation.

## 10. References ngoài `docs/`

- Skill: [`lightweight-web`](../../../.agent/skills/personal/engineering/frontend/lightweight-web/SKILL.md)
- Skill: [`feature-delivery`](../../../.agent/skills/common/workflow/feature-delivery/SKILL.md)
- Skill: [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md)
- Skill: [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md)
- Skill: [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md)
- Skill: [`git-workflow`](../../../.agent/skills/common/engineering/git-workflow/SKILL.md)
- Project context: [`CONTEXT.md`](../../../CONTEXT.md)
- Source/config/test path ngoài `docs/`: `extension/search/`, `extension/manifest.json`, `tests/browser/search-surface/`, `tests/accessibility/search-surface/`, `fixtures/ui/` (all `to-create` logical paths).
- Fixture/tool/artifact ngoài `docs/`: `fixtures/ui/UI-SHORTCUT-FOCUS`, `UI-LIVE-QUERY`, `UI-ARROW-ENTER`, `UI-ESC-DISMISS`, `UI-EMPTY`, `UI-NO-RESULT`, `UI-OPENING`, `UI-REBUILDING`, `UI-HOST-DOWN`, `UI-PERSISTENCE-DEGRADED`, `UI-STALE`, `UI-ACTIVATION-FAILURE`, `UI-PRIVATE-UNAVAILABLE`, `UI-DUPLICATE-CONTEXT`, `UI-NARROW-TEXT-SCALE` (to-create); future browser accessibility snapshots and latency artifact directory under `tests/accessibility/search-surface/` (to-create).
- External target refs: [Chrome Commands API](https://developer.chrome.com/docs/extensions/reference/api/commands), [Chrome Tabs API](https://developer.chrome.com/docs/extensions/reference/api/tabs), [Chrome Native Messaging](https://developer.chrome.com/docs/extensions/develop/concepts/native-messaging), [Chrome service-worker lifecycle](https://developer.chrome.com/docs/extensions/develop/concepts/service-workers/lifecycle), [Edge native messaging](https://learn.microsoft.com/en-us/microsoft-edge/extensions/developer-guide/native-messaging), and [Porting Chrome extensions to Edge](https://learn.microsoft.com/en-us/microsoft-edge/extensions/developer-guide/port-chrome-extension).
