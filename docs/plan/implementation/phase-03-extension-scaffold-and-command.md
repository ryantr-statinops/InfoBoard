# Phase 03 — Extension scaffold and configurable command

> Plan ID: IP-03
> Status: See README.md execution tracker
> Execution owner: extension-platform implementer
> Dependencies: IP-01, IP-02
> Parallel boundary: IP-06, IP-08 after IP-02 contracts are stable
> Requirement IDs: FR-001 (primary); FR-002, FR-006, FR-008, NFR-007, NFR-008, NFR-010 (supporting)
> Owned paths: `extension/manifest.json` (to-create); `extension/src/background/service-worker.ts` (to-create); `extension/src/browser/browser-adapter.ts` (to-create); `extension/src/search/surface-entrypoint.ts` (to-create); `extension/src/search/surface-shell.html` (to-create); `fixtures/extension/command-open-surface.json` (to-create); `tests/extension/manifest-permissions.test.ts` (to-create); `tests/extension/command-open-surface.test.ts` (to-create); `tests/extension/service-worker-lifecycle.test.ts` (to-create)

## 1. Mục tiêu

- Tạo nền tảng Manifest V3 tối thiểu cho Chrome và Edge desktop với hai shortcut cấu hình được: `open-search` để mở browser action popup và `close-search` để đóng popup.
- `open-search` gọi action popup 480×600 bên trong browser chrome; `close-search` đóng popup đang mở. Ô query được focus trước khi nhận phím; cả hai shortcut giữ nguyên tab đang chọn và không tạo OS window/tab mới.
- Tách quyền browser khỏi logic popup bằng browser-adapter seam dùng chung Chrome/Edge. Worker phải hoạt động sau startup/suspension/restart mà không dựa vào global state sống lâu.
- Chốt contract cho các phase sau: IP-04 sở hữu tab observation/reconcile, IP-12/IP-13 sở hữu query và UI states đầy đủ; phase này sở hữu bootstrap, action-popup focus handoff và lifecycle boundary.
- Đáp ứng FR-001 làm requirement owner; cung cấp các seam cần thiết cho FR-002, FR-006, FR-008 và các baseline NFR-007, NFR-008, NFR-010.

## 2. Phạm vi

- Bao gồm:
  - Manifest V3 với `action.default_popup`, module service worker và hai command định danh `open-search`/`close-search`; toolbar action và `open-search` cùng mở một extension-owned popup.
  - Command handler single-flight: mở bằng action popup API trực tiếp trong user gesture hoặc gửi close signal tới popup; không tạo OS window/tab và không điều hướng tab đang chọn.
  - `surface-entrypoint` tạo session mới, reset query, focus input với `preventScroll` sau DOM ready và phát ready/error signal mà không tái sử dụng query cũ.
  - Browser adapter chỉ expose profile-context/storage contract, action-popup opening, command registration và lifecycle signals; không expose raw browser objects hoặc gọi tabs/page APIs ngoài boundary.
  - Service-worker lifecycle: listeners top-level, profile ID khởi tạo chỉ trong install event, command hoạt động sau restart, không có timer/network daemon.
  - Phase-local permissions are exactly `storage` and `windows`; `storage.local` may use only key `profile_id`. `action` and `commands` are manifest keys, not permissions. No `tabs`, host permissions, `sidePanel`, `scripting`, or page access in IP-03; defer `tabs`/`tabGroups` to IP-04 and `nativeMessaging` to IP-07.
  - Accessibility handoff tối thiểu: query input có accessible name/label, là control đầu tiên nhận focus khi surface sẵn sàng, có visible focus style do shell cung cấp và không phụ thuộc animation; detailed result labels, live regions, reduced-motion styling và keyboard result navigation bàn giao cho IP-13.
  - Fixture/test seams cover both shortcuts, toolbar popup invocation, focus-before-typing, single-flight open/close, Escape, explicit close, manifest permissions, worker suspension/restart and Chrome/Edge parity.
- Ngoài phạm vi:
  - Không implement tab create/update/move/group/pin/activate/remove observation, eligibility filtering, private-context policy hoặc projection reconciliation; các boundary đó thuộc IP-04/IP-05.
  - Không implement Native Messaging frame/envelope/handshake, Go host lifecycle, SQLite, lexical index, ranking, query orchestration, health hoặc diagnostics; các boundary đó thuộc IP-06 đến IP-12.
  - Không implement đầy đủ result list, ranking explanations, live query states, activation race guard, configuration/reset/uninstall hoặc packaging/host registration; phase này chỉ phát ra shell/adapter contracts cho các phase sở hữu chúng.
  - Không xin hoặc sử dụng quyền history, downloads, cookies, page-content access, scripting/injection, broad host access, network access hay bất kỳ browser-data-library nào. Không đọc DOM của page, không fetch URL, không tạo loopback server và không thêm semantic inference.

## 3. Điều kiện tiên quyết

- IP-01 đã chốt requirement traceability, fixture naming, logical roots (`extension/`, `host/`, `packaging/`, `fixtures/`, `tests/`) và global test command conventions. Vì source tree hiện chưa tồn tại, mọi path trong phase này phải được tạo mới và được ghi nhận là `to-create` cho đến khi implementation scaffold xuất hiện.
- IP-02 đã chốt `profile_id`, ownership của extension đối với browser authority, identity/revision/activation metadata và quy tắc không vượt profile boundary. Command handler không tự tạo profile identity và không gửi query/activation cho profile khác.
- Có một browser fixture harness cho Chrome/Edge command event, action-popup window context và `document.activeElement`; harness phải cho phép suspend/restart worker rồi phát lại command.
- Browser support target là Chrome và Edge desktop trên Linux, macOS và Windows. Chrome và Edge adapter dùng cùng contract; khác biệt namespace/API phải được cô lập trong adapter, không rò rỉ vào surface.
- Định dạng Manifest V3, extension build output và test runner của repository được IP-01 công bố. Native host có thể chưa chạy trong test command; chỉ cần manifest declaration và một injectable Native Messaging capability seam, vì protocol ownership thuộc IP-07.

## 4. Đầu ra cần bàn giao

- `extension/manifest.json`: Manifest V3 with `open-search`/`close-search` keyboard commands, `action.default_popup` pointing to the packaged search shell, service worker, only phase-local `storage`/`windows` permissions, and no host permissions.
- `extension/src/background/service-worker.ts` (to-create): top-level listener registration, command dispatch, startup/install hooks, operation cancellation/recovery boundary và adapter construction. Không giữ projection/index state trong worker globals.
- `extension/src/browser/browser-adapter.ts`: typed profile-context, identity-store, action-popup-open, command-listener, and lifecycle contracts; no tab creation/focus or page APIs.
- `extension/src/search/surface-entrypoint.ts` (to-create): extension-page bootstrap, fresh search-session token, semantic query input lookup, focus handoff, close/dismiss event boundary và ready signal.
- `extension/src/search/surface-shell.html` (to-create): tối thiểu semantic shell có labeled query input, result/status mount points, keyboard-dismiss boundary và visible focus affordance; không nhúng page content hay remote assets.
- `fixtures/extension/command-open-surface.json`: command/profile/context input and observable in-browser action popup, focus, selected-tab preservation, bounds, and dismissal expectations.
- `tests/extension/manifest-permissions.test.ts` (to-create): parse built manifest và fail nếu permission/host permission vượt allowlist hoặc chứa quyền excluded.
- `tests/extension/command-open-surface.test.ts`: command-to-action-popup focus journey, single-flight dispatch, fresh query session, Escape/close dismissal, and adapter seam.
- `tests/extension/service-worker-lifecycle.test.ts` (to-create): top-level listener registration, suspend/resume, startup/restart and cancellation/recovery observations.
- Implementation handoff note in code comments or test contract (not a new documentation tree): exact events/inputs/outputs that IP-04, IP-07, IP-13 and IP-14 consume without sharing global browser calls.

## 5. Skill và tài liệu áp dụng

- Skill tags:
  - [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md) — giữ phase plan standalone, links và acceptance evidence rõ ràng cho người implement/review.
  - [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md) — chia scaffold thành các boundary có dependency, error path, test seam và commit slices độc lập.
  - [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md) — thiết kế fixture observable cho command, focus, permission allowlist và worker lifecycle thay vì chỉ kiểm tra source text.
  - [`git-workflow`](../../../.agent/skills/common/engineering/git-workflow/SKILL.md) — giới hạn thay đổi phase file, kiểm tra diff/path và bàn giao commit reproducible.
  - [`feature-delivery`](../../../.agent/skills/common/workflow/feature-delivery/SKILL.md) — lập vertical slice từ manifest đến browser-visible command journey và review handoff.
  - [`lightweight-web`](../../../.agent/skills/personal/engineering/frontend/lightweight-web/SKILL.md) — chọn semantic HTML, ít dependency, nhanh và accessible cho search shell nhỏ, focused.
  - [`secure-development`](../../../.agent/skills/common/security/secure-development/SKILL.md) — áp dụng least privilege, validate command/window identifiers, không log URL/title và cô lập browser trust boundary.
- Tài liệu trong `docs/`:
  - [`requirements.md`](../refactor/requirements.md) — nguồn chuẩn của FR-001, FR-002, FR-006, FR-008, NFR-007, NFR-008, NFR-010 và permission rule.
  - [`architecture.md`](../refactor/architecture.md) — ownership của extension, command/search surface, browser adapter và Native Messaging boundary.
  - [`user-experience.md`](../refactor/user-experience.md) — command mở focused surface, fresh focus, keyboard-only flow và không inject UI vào page.
  - [`verification-and-acceptance.md`](../refactor/verification-and-acceptance.md) — extension integration, service-worker suspension/restart và keyboard/accessibility evidence.
  - [`domain-and-privacy.md`](../refactor/domain-and-privacy.md) — profile isolation, browser authority và dữ liệu được phép/không được phép.
- Quy ước code, ADR, context ngoài `docs/`:
  - [`CONTEXT.md`](../../../CONTEXT.md) — vocabulary, product boundary và quy tắc ưu tiên canonical refactor docs.
  - Manifest và TypeScript paths là logical `to-create` targets; không giả định source file hiện có hoặc ký hiệu implementation chưa được scaffold.
  - Browser API access chỉ qua `browser-adapter.ts`; surface code không được import `chrome`/`browser` trực tiếp. API error phải được phân loại thành recoverable focus/open failure hoặc unavailable browser context để UI báo đúng trạng thái.

## 6. Công việc triển khai

- [ ] `IP-03-T01` **Manifest và browser-popup boundary:** khai báo Manifest V3 với `action.default_popup: dist/search/surface-shell.html`, module service worker và `open-search`/`close-search` commands. Phase-local permissions đúng là `storage` và `windows`; storage chỉ dùng REF-013 key `profile_id`. `action` và `commands` là manifest keys, không phải permissions. Không khai báo `tabs`, host permissions, `sidePanel` hay scripting.

- [ ] `IP-03-T02` **Permission fail-closed:** test built manifest có đúng `storage`, `windows`, hai command open/close, một action popup, không `host_permissions`, và không history/download/cookie/scripting/page access.
- [ ] `IP-03-T03` **Browser adapter contract:** expose `currentProfileContext()`, `openSearchSurface()` using `chrome.action.openPopup()`, the identity-only profile store, and top-level command/lifecycle registration. Return bounded results; never expose raw Chrome/Edge objects.
- [ ] `IP-03-T04` **Command dispatch:** `open-search` calls `chrome.action.openPopup()` synchronously from the keyboard user gesture and single-flights repeated opens; `close-search` sends a bounded close message to the open popup. The popup validates profile identity before focus/ready. Neither command creates, activates, or navigates tabs.
- [ ] `IP-03-T05` **Popup bounds:** action popup là extension-owned resource `dist/search/surface-shell.html`, kích thước CSS 480×600; không tạo detached OS window hoặc load URL/asset ngoài extension package.
- [ ] `IP-03-T06` **Keyboard focus handoff:** tạo session mới, reset query, chờ DOM ready rồi focus input với `preventScroll`; ready chỉ khi `document.activeElement` đúng input. Missing input/focus failure hiển thị status recoverable.
- [ ] `IP-03-T07` **Dismissal boundary:** `Esc`, the close button, and the configured `close-search` shortcut close the action popup. Dismissal never activates/navigates the selected tab or persists/logs query, title, or URL.
- [ ] `IP-03-T08` **Service-worker lifecycle:** đăng ký listeners ở module evaluation, giữ chỉ bounded single-flight state, khởi tạo profile ID chỉ trong first-install, phục hồi từ extension storage sau restart; không polling/network/long-lived timer.
- [ ] `IP-03-T09` **Browser parity:** Chrome và Edge dùng cùng action-popup contract và fixture, chỉ khác browser channel; API errors được normalize trong adapter.
- [ ] `IP-03-T10` **Handoff:** expose profile/context/session-ready seams; browser authority remains in extension; no host, projection or page-content responsibility enters this phase.
- [ ] `IP-03-T11` **Targeted tests:** cover `open-search` and `close-search` shortcuts, popup focus, single-flight opening, missing/corrupt profile ID, storage failures, worker restart, Escape, explicit close, and no network/page read.

## 7. Kế hoạch commit

1. `feat(extension): implement ip-03-t01`
   - Task IDs: `IP-03-T01`.
   - Owned target paths: extension/manifest.json.
   - Behavior: configure the browser action popup resource, one `open-search` command and one `close-search` command; only `storage` and `windows` are requested, with storage restricted to REF-013 key `profile_id`.
   - Fixture and command: read `fixtures/extension/command-open-surface.json` directly, then run `npm run build:extension`, `BROWSER=chrome npm run test:extension -- tests/extension/manifest-permissions.test.ts tests/extension/command-open-surface.test.ts tests/extension/service-worker-lifecycle.test.ts`, and the identical command with `BROWSER=edge`.
   - Observable result before commit: the built manifest includes the toolbar popup, both keyboard commands, exactly the phase-local `storage` and `windows` permissions, and no host permissions.
   - Dependency gate: all index.md dependencies for IP-03 have merged to dev; phase work branch starts from latest origin/dev.

2. `test(extension): implement ip-03-t02`
   - Task IDs: `IP-03-T02`.
   - Owned target paths: `extension/manifest.json` (to-create); `extension/src/background/service-worker.ts` (to-create); `extension/src/browser/browser-adapter.ts` (to-create); `extension/src/search/surface-entrypoint.ts` (to-create); `extension/src/search/surface-shell.html` (to-create); `fixtures/extension/command-open-surface.json` (to-create); `tests/extension/manifest-permissions.test.ts` (to-create); `tests/extension/command-open-surface.test.ts` (to-create); `tests/extension/service-worker-lifecycle.test.ts` (to-create).
   - Behavior: manifest parser test requires exactly the `storage`/`windows` permissions, one action popup, `open-search` and `close-search` command declarations, and no host permissions or excluded browser-data permissions.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-03 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: undeclared permissions, host patterns, an absent popup, or missing open/close command declarations fail the manifest audit.
   - Dependency gate: all index.md dependencies for IP-03 have merged to dev; phase work branch starts from latest origin/dev.

3. `test(extension): implement ip-03-t03`
   - Task IDs: `IP-03-T03`.
   - Owned target paths: `extension/manifest.json` (to-create); `extension/src/background/service-worker.ts` (to-create); `extension/src/browser/browser-adapter.ts` (to-create); `extension/src/search/surface-entrypoint.ts` (to-create); `extension/src/search/surface-shell.html` (to-create); `fixtures/extension/command-open-surface.json` (to-create); `tests/extension/manifest-permissions.test.ts` (to-create); `tests/extension/command-open-surface.test.ts` (to-create); `tests/extension/service-worker-lifecycle.test.ts` (to-create).
   - Behavior: define bounded adapter methods for `currentProfileContext()` and `openSearchSurface()`, plus profile-ID storage and top-level command/lifecycle listener registration. Normalize browser API failures without exposing Chrome/Edge objects or raw exceptions.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-03 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: the same adapter opens the action popup in the active Chrome/Edge window and reads/writes/removes only `chrome.storage.local.profile_id`.
   - Dependency gate: all index.md dependencies for IP-03 have merged to dev; phase work branch starts from latest origin/dev.

4. `feat(extension): implement ip-03-t04`
   - Task IDs: `IP-03-T04`.
   - Owned target paths: `extension/manifest.json` (to-create); `extension/src/background/service-worker.ts` (to-create); `extension/src/browser/browser-adapter.ts` (to-create); `extension/src/search/surface-entrypoint.ts` (to-create); `extension/src/search/surface-shell.html` (to-create); `fixtures/extension/command-open-surface.json` (to-create); `tests/extension/manifest-permissions.test.ts` (to-create); `tests/extension/command-open-surface.test.ts` (to-create); `tests/extension/service-worker-lifecycle.test.ts` (to-create).
   - Behavior: `open-search` calls `chrome.action.openPopup()` synchronously from the user gesture; `close-search` sends a close message to the popup. Single-flight repeated opens. Profile-ID validation happens in the popup before it becomes ready.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-03 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: opening stays inside browser chrome; the close shortcut dismisses only the popup; missing/corrupt identity leaves the popup unready with a bounded status.
   - Dependency gate: all index.md dependencies for IP-03 have merged to dev; phase work branch starts from latest origin/dev.

5. `feat(extension): implement ip-03-t05`
   - Task IDs: `IP-03-T05`.
   - Owned target paths: `extension/manifest.json` (to-create); `extension/src/background/service-worker.ts` (to-create); `extension/src/browser/browser-adapter.ts` (to-create); `extension/src/search/surface-entrypoint.ts` (to-create); `extension/src/search/surface-shell.html` (to-create); `fixtures/extension/command-open-surface.json` (to-create); `tests/extension/manifest-permissions.test.ts` (to-create); `tests/extension/command-open-surface.test.ts` (to-create); `tests/extension/service-worker-lifecycle.test.ts` (to-create).
   - Behavior: declare the extension-owned toolbar action popup with a 480×600 CSS viewport; preserve the active browser tab and window, load only packaged extension assets, and add no page or host access.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-03 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: command invocation displays the popup inside Chrome/Edge browser chrome rather than a detached OS window; the currently selected tab remains unchanged.
   - Dependency gate: all index.md dependencies for IP-03 have merged to dev; phase work branch starts from latest origin/dev.

6. `test(extension): implement ip-03-t06`
   - Task IDs: `IP-03-T06`.
   - Owned target paths: `extension/manifest.json` (to-create); `extension/src/background/service-worker.ts` (to-create); `extension/src/browser/browser-adapter.ts` (to-create); `extension/src/search/surface-entrypoint.ts` (to-create); `extension/src/search/surface-shell.html` (to-create); `fixtures/extension/command-open-surface.json` (to-create); `tests/extension/manifest-permissions.test.ts` (to-create); `tests/extension/command-open-surface.test.ts` (to-create); `tests/extension/service-worker-lifecycle.test.ts` (to-create).
   - Behavior: **Keyboard focus handoff:** trong `surface-entrypoint.ts`, tạo fresh session, reset query value, render accessible shell, chờ DOM ready rồi gọi focus query input với `preventScroll`; assert `document.activeElement === queryInput` trong fixture trước khi dispatch text. Focus failure, missing input hoặc closed document phải phát signal lỗi recoverable, không giả vờ đã sẵn sàng.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-03 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Keyboard focus handoff:** trong `surface-entrypoint.ts`, tạo fresh session, reset query value, render accessible shell, chờ DOM ready rồi gọi focus query input với `preventScroll`; assert `document.activeElement === queryInput` trong fixture trước khi dispatch text. Focus failure, missing input hoặc closed document phải phát signal lỗi recoverable, không giả vờ đã sẵn sàng.
   - Dependency gate: all index.md dependencies for IP-03 have merged to dev; phase work branch starts from latest origin/dev.

7. `feat(extension): implement ip-03-t07`
   - Task IDs: `IP-03-T07`.
   - Owned target paths: `extension/manifest.json` (to-create); `extension/src/background/service-worker.ts` (to-create); `extension/src/browser/browser-adapter.ts` (to-create); `extension/src/search/surface-entrypoint.ts` (to-create); `extension/src/search/surface-shell.html` (to-create); `fixtures/extension/command-open-surface.json` (to-create); `tests/extension/manifest-permissions.test.ts` (to-create); `tests/extension/command-open-surface.test.ts` (to-create); `tests/extension/service-worker-lifecycle.test.ts` (to-create).
   - Behavior: Escape, explicit close, and `close-search` use the action-popup lifecycle; no tab activation/navigation or sensitive-value logging.
   - Fixture and command: IP-13; run `the exact phase-03 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: all three dismissal paths close the popup and preserve the previously selected tab/window.
   - Dependency gate: all index.md dependencies for IP-03 have merged to dev; phase work branch starts from latest origin/dev.

8. `feat(extension): implement ip-03-t08`
   - Task IDs: `IP-03-T08`.
   - Owned target paths: `extension/manifest.json` (to-create); `extension/src/background/service-worker.ts` (to-create); `extension/src/browser/browser-adapter.ts` (to-create); `extension/src/search/surface-entrypoint.ts` (to-create); `extension/src/search/surface-shell.html` (to-create); `fixtures/extension/command-open-surface.json` (to-create); `tests/extension/manifest-permissions.test.ts` (to-create); `tests/extension/command-open-surface.test.ts` (to-create); `tests/extension/service-worker-lifecycle.test.ts` (to-create).
   - Behavior: **Service-worker lifecycle:** đăng ký listeners synchronously at module evaluation, giữ worker stateless ngoài bounded in-flight operation map, xử lý `runtime.onStartup`/`onInstalled` để validate package/adapter readiness, và recreate adapter after suspension. Không dùng polling, websocket, loopback server, long-lived timer hay page-content listener để giữ worker sống.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-03 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Service-worker lifecycle:** đăng ký listeners synchronously at module evaluation, giữ worker stateless ngoài bounded in-flight operation map, xử lý `runtime.onStartup`/`onInstalled` để validate package/adapter readiness, và recreate adapter after suspension. Không dùng polling, websocket, loopback server, long-lived timer hay page-content listener để giữ worker sống.
   - Dependency gate: all index.md dependencies for IP-03 have merged to dev; phase work branch starts from latest origin/dev.

9. `test(extension): implement ip-03-t09`
   - Task IDs: `IP-03-T09`.
   - Owned target paths: `extension/manifest.json` (to-create); `extension/src/background/service-worker.ts` (to-create); `extension/src/browser/browser-adapter.ts` (to-create); `extension/src/search/surface-entrypoint.ts` (to-create); `extension/src/search/surface-shell.html` (to-create); `fixtures/extension/command-open-surface.json` (to-create); `tests/extension/manifest-permissions.test.ts` (to-create); `tests/extension/command-open-surface.test.ts` (to-create); `tests/extension/service-worker-lifecycle.test.ts` (to-create).
   - Behavior: run identical action-popup assertions in Chrome and Edge; browser-family detection and bounded API error mapping remain in the adapter. Do not branch behavior by browser in the popup.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-03 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: both browsers display the same toolbar popup and focus/dismiss behavior with no permission or host-access drift.
   - Dependency gate: all index.md dependencies for IP-03 have merged to dev; phase work branch starts from latest origin/dev.

10. `feat(extension): implement ip-03-t10`
   - Task IDs: `IP-03-T10`.
   - Owned target paths: `extension/manifest.json` (to-create); `extension/src/background/service-worker.ts` (to-create); `extension/src/browser/browser-adapter.ts` (to-create); `extension/src/search/surface-entrypoint.ts` (to-create); `extension/src/search/surface-shell.html` (to-create); `fixtures/extension/command-open-surface.json` (to-create); `tests/extension/manifest-permissions.test.ts` (to-create); `tests/extension/command-open-surface.test.ts` (to-create); `tests/extension/service-worker-lifecycle.test.ts` (to-create).
   - Behavior: expose the profile/context/session-ready seams and action-popup surface-ready signal to later phases. The extension remains the sole browser authority; no host or page API opens the popup.
   - Fixture and command: IP-04, IP-14; run `the exact phase-03 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: downstream phases can read the bounded profile/session context and popup readiness without page access or browser-state mutation.
   - Dependency gate: all index.md dependencies for IP-03 have merged to dev; phase work branch starts from latest origin/dev.

11. `test(extension): implement ip-03-t11`
   - Task IDs: `IP-03-T11`.
   - Owned target paths: `extension/manifest.json` (to-create); `extension/src/background/service-worker.ts` (to-create); `extension/src/browser/browser-adapter.ts` (to-create); `extension/src/search/surface-entrypoint.ts` (to-create); `extension/src/search/surface-shell.html` (to-create); `fixtures/extension/command-open-surface.json` (to-create); `tests/extension/manifest-permissions.test.ts` (to-create); `tests/extension/command-open-surface.test.ts` (to-create); `tests/extension/service-worker-lifecycle.test.ts` (to-create).
   - Behavior: test both configurable shortcuts, action popup focus and bounds, single-flight opening, close hotkey, Escape, explicit close, storage/profile fail-closed paths, restart, and zero page/network access.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-03 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: browser tests assert the popup and active-tab state; fakes cover deterministic API failures but never stand in for Chrome/Edge parity.
   - Dependency gate: all index.md dependencies for IP-03 have merged to dev; phase work branch starts from latest origin/dev.

## 8. Kiểm chứng và nghiệm thu

- [ ] From repository root, run `npm ci`, `npm run build:extension`, then `BROWSER=chrome npm run test:extension -- tests/extension/manifest-permissions.test.ts tests/extension/command-open-surface.test.ts tests/extension/service-worker-lifecycle.test.ts`. The fixture is read directly from `fixtures/extension/command-open-surface.json`.
- [ ] Repeat with `BROWSER=edge npm run test:extension -- tests/extension/manifest-permissions.test.ts tests/extension/command-open-surface.test.ts tests/extension/service-worker-lifecycle.test.ts` using the Edge browser binary.
- [ ] In each browser, invoke configured `open-search` without pointer input. Observe a browser-toolbar action popup (not a detached OS window), 480×600 CSS viewport, focused `#query`, and an unchanged active product tab/window.
- [ ] Repeat after service-worker suspension/restart. `open-search` must still open the popup and single-flight duplicate opens; unavailable profile identity must leave the popup unready with a bounded recovery status.
- [ ] Trigger the configured `close-search` shortcut (default Ctrl+Shift+X; Command+Shift+X on macOS), Escape, and the close control. Each closes only the popup and preserves the previously selected tab/window without activation or sensitive-value logging.
- [ ] Parse the built MV3 manifest. Acceptance requires exactly the `storage` and `windows` permissions, action popup path `dist/search/surface-shell.html`, exactly `open-search` and `close-search` commands, no `host_permissions`, and no history/download/cookie/page/scripting access.
- [ ] Run the accessibility smoke: accessible query label, visible focus, no animation-dependent focus, and usability at the declared popup viewport. Full result-list accessibility and reduced-motion evidence remain IP-13.
- [ ] Intercept extension requests: zero network requests, page navigation/injection, page-content reads, or remote service calls.
- [ ] Acceptance mapping: FR-001 by Chrome/Edge command settings plus open/close shortcut journeys; FR-002 by focus-before-text; FR-006 by stable result/status mount points; FR-008 by close/Escape preserving the active tab; NFR-007 by permission/request audit; NFR-008 by keyboard focus/accessibility; NFR-010 by Chrome/Edge release matrix.

## 9. Rủi ro và quyết định còn mở

- Rủi ro: Chrome and Edge may differ in window discovery, command dispatch timing, or permission semantics. Phương án xử lý đã chọn: adapter contract plus same fixture assertions on both browsers; browser-specific code is confined to adapter and any divergence is recorded as a bounded capability error.
- Rủi ro: a Manifest V3 worker can suspend between command receipt and window/focus operations. Phương án xử lý đã chọn: top-level listener registration, short bounded operations, cancellation cleanup and reconstruction from browser state; never rely on worker-global durable state.
- Rủi ro: opening a second popup for every shortcut can create duplicate surfaces or steal focus from a user action. Phương án xử lý đã chọn: identify only extension-owned surface URLs in the active profile, serialize create/focus operations, focus existing surface first and test two rapid invocations.
- Rủi ro: `tabs`, `windows` or `tabGroups` may expose more metadata than the exact build needs. Phương án xử lý đã chọn: run an API/permission probe, remove any unnecessary permission, and make the permission fixture fail closed. No broad host pattern is an acceptable fallback.
- Rủi ro: focus can be attempted before the extension document commits its input. Phương án xử lý đã chọn: surface ready lifecycle after DOM construction, direct input focus with an observable active-element assertion, bounded retry only within the page lifecycle, and explicit recoverable failure when the input is absent.
- Rủi ro: a browser API rejection could leave the UI claiming that search is ready. Phương án xử lý đã chọn: return typed open/focus failure to the surface, keep the current tab unchanged, and expose a retryable status for IP-13 rather than swallowing the exception.
- Rủi ro: an implementation may accidentally add page-content access while implementing tab metadata. Phương án xử lý đã chọn: no content scripts, no `activeTab`, no host permissions, no page evaluation/fetch; IP-04 owns tab APIs and must consume the adapter seam only.
- Câu hỏi còn mở chỉ khi câu trả lời có thể thay đổi contract: whether a target browser release requires `tabGroups` or `windows` as explicit manifest permission after API probe, and the exact popup bounds accepted by the release accessibility matrix. Neither question may relax one-command, focus-before-text, least-privilege or no-page-content invariants.

## 10. References ngoài `docs/`

- Skill: [`documentation`](../../../.agent/skills/common/engineering/documentation/SKILL.md)
- Skill: [`task-planning`](../../../.agent/skills/common/foundation/task-planning/SKILL.md)
- Skill: [`testing`](../../../.agent/skills/common/engineering/testing/SKILL.md)
- Skill: [`git-workflow`](../../../.agent/skills/common/engineering/git-workflow/SKILL.md)
- Skill: [`feature-delivery`](../../../.agent/skills/common/workflow/feature-delivery/SKILL.md)
- Skill: [`lightweight-web`](../../../.agent/skills/personal/engineering/frontend/lightweight-web/SKILL.md)
- Skill: [`secure-development`](../../../.agent/skills/common/security/secure-development/SKILL.md)
- Project context: [`CONTEXT.md`](../../../CONTEXT.md)
- Source/config/test path ngoài `docs/`: `extension/manifest.json` (to-create); `extension/src/background/service-worker.ts` (to-create); `extension/src/browser/browser-adapter.ts` (to-create); `extension/src/search/surface-entrypoint.ts` (to-create); `extension/src/search/surface-shell.html` (to-create); `fixtures/extension/command-open-surface.json` (to-create); `tests/extension/manifest-permissions.test.ts` (to-create); `tests/extension/command-open-surface.test.ts` (to-create); `tests/extension/service-worker-lifecycle.test.ts` (to-create).
- Fixture/tool/artifact ngoài `docs/`: `fixtures/extension/command-open-surface.json` and the built Manifest V3 artifact consumed by `tests/extension/manifest-permissions.test.ts`; browser binaries are selected by `BROWSER=chrome|edge` in the targeted command.
