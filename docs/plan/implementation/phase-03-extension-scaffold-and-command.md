# Phase 03 — Extension scaffold and configurable command

> Plan ID: IP-03
> Status: See README.md execution tracker
> Execution owner: extension-platform implementer
> Dependencies: IP-01, IP-02
> Parallel boundary: IP-06, IP-08 after IP-02 contracts are stable
> Requirement IDs: FR-001 (primary); FR-002, FR-006, FR-008, NFR-007, NFR-008, NFR-010 (supporting)
> Owned paths: `extension/manifest.json` (to-create); `extension/src/background/service-worker.ts` (to-create); `extension/src/browser/browser-adapter.ts` (to-create); `extension/src/search/surface-entrypoint.ts` (to-create); `extension/src/search/surface-shell.html` (to-create); `fixtures/extension/command-open-surface.json` (to-create); `tests/extension/manifest-permissions.test.ts` (to-create); `tests/extension/command-open-surface.test.ts` (to-create); `tests/extension/service-worker-lifecycle.test.ts` (to-create)

## 1. Mục tiêu

- Tạo nền tảng Manifest V3 tối thiểu cho Chrome và Edge desktop, có đúng một command người dùng có thể cấu hình để mở search surface của InfoBoard.
- Khi command được phát ra, extension phải mở hoặc đưa search surface extension-owned lên trước trong profile hiện tại, chuyển focus vào ô query trước khi nhận phím, và không điều hướng hay thay đổi tab đang được chọn.
- Tách quyền browser khỏi logic surface bằng một browser-adapter seam có thể chạy với Chrome hoặc Edge và có fake adapter cho fixture. Service worker phải hoạt động đúng sau startup, suspension/resume và browser restart mà không dựa vào state toàn cục sống lâu.
- Chốt contract bàn giao cho các phase sau: IP-04 sẽ sở hữu quan sát/reconcile tab events, IP-12/IP-13 sẽ sở hữu query và UI states đầy đủ, còn phase này chỉ sở hữu bootstrap, entrypoint, focus handoff và lifecycle boundary.
- Đáp ứng FR-001 làm requirement owner; cung cấp các seam cần thiết cho FR-002, FR-006, FR-008 và các baseline NFR-007, NFR-008, NFR-010.

## 2. Phạm vi

- Bao gồm:
  - Manifest V3 với extension identity, version, extension-owned search surface URL, service-worker background entrypoint, một command định danh ổn định (ví dụ `open-search`) và nhãn command hiển thị trong Chrome/Edge command settings.
  - Command handler idempotent: tạo popup/focused extension window nếu chưa có, hoặc focus cửa sổ/search tab hiện có nếu command được gọi lần nữa; không tạo các cửa sổ trùng lặp trong một command burst.
  - `surface-entrypoint` khởi tạo một search session mới, đặt focus vào query input bằng code (không yêu cầu pointer), không tái sử dụng query cũ âm thầm, và phát tín hiệu sẵn sàng để UI/query phases tiếp tục.
  - Browser adapter interface và các adapter calls cần cho `runtime.getURL`, tìm/tạo/cập nhật window, cập nhật active tab, đọc command event và đóng surface. Adapter phải cho phép inject fake API trong tests; không để logic sản phẩm gọi trực tiếp global `chrome`/`browser` ngoài boundary.
  - Service-worker lifecycle: đăng ký listeners ở top level để survive worker suspension, khởi tạo/recover adapter state cho `onStartup`/`onInstalled`, xử lý command sau worker restart, hủy promise/operation dang dở khi browser context chết, và không giữ timer/network daemon để kéo dài worker.
  - Permissions tối thiểu cho các API thuộc boundary này. Manifest chỉ được khai báo các quyền thực sự cần cho current open-tab projection/activation và Native Messaging integration: `tabs`, `windows`, `tabGroups`, `nativeMessaging` (hoặc một tập con được chứng minh bằng API probe). Không khai báo host pattern; `host_permissions` phải được bỏ qua hoặc là mảng rỗng.
  - Accessibility handoff tối thiểu: query input có accessible name/label, là control đầu tiên nhận focus khi surface sẵn sàng, có visible focus style do shell cung cấp và không phụ thuộc animation; detailed result labels, live regions, reduced-motion styling và keyboard result navigation bàn giao cho IP-13.
  - Fixture và test seams cho command invocation, existing-surface focus, focus-before-typing, Esc dismissal, manifest permissions, worker suspension/resume và Chrome/Edge adapter parity.
- Ngoài phạm vi:
  - Không implement tab create/update/move/group/pin/activate/remove observation, eligibility filtering, private-context policy hoặc projection reconciliation; các boundary đó thuộc IP-04/IP-05.
  - Không implement Native Messaging frame/envelope/handshake, Go host lifecycle, SQLite, lexical index, ranking, query orchestration, health hoặc diagnostics; các boundary đó thuộc IP-06 đến IP-12.
  - Không implement đầy đủ result list, ranking explanations, live query states, activation race guard, configuration/reset/uninstall hoặc packaging/host registration; phase này chỉ phát ra shell/adapter contracts cho các phase sở hữu chúng.
  - Không xin hoặc sử dụng quyền history, downloads, cookies, page-content access, scripting/injection, broad host access, network access hay bất kỳ browser-data-library nào. Không đọc DOM của page, không fetch URL, không tạo loopback server và không thêm semantic inference.

## 3. Điều kiện tiên quyết

- IP-01 đã chốt requirement traceability, fixture naming, logical roots (`extension/`, `host/`, `packaging/`, `fixtures/`, `tests/`) và global test command conventions. Vì source tree hiện chưa tồn tại, mọi path trong phase này phải được tạo mới và được ghi nhận là `to-create` cho đến khi implementation scaffold xuất hiện.
- IP-02 đã chốt `profile_id`, ownership của extension đối với browser authority, identity/revision/activation metadata và quy tắc không vượt profile boundary. Command handler không tự tạo profile identity và không gửi query/activation cho profile khác.
- Có một browser fixture harness có thể mô phỏng Chrome/Edge command event, window/tab creation và `document.activeElement`; harness phải cho phép worker bị suspend rồi phát lại command để chứng minh lifecycle.
- Browser support target là Chrome và Edge desktop trên Linux, macOS và Windows. Chrome và Edge adapter dùng cùng contract; khác biệt namespace/API phải được cô lập trong adapter, không rò rỉ vào surface.
- Định dạng Manifest V3, extension build output và test runner của repository được IP-01 công bố. Native host có thể chưa chạy trong test command; chỉ cần manifest declaration và một injectable Native Messaging capability seam, vì protocol ownership thuộc IP-07.

## 4. Đầu ra cần bàn giao

- `extension/manifest.json` (to-create): Manifest V3 tối thiểu, command `open-search`, service-worker entry, search surface resource, browser-compatible metadata, exact permissions và không có broad host permissions.
- `extension/src/background/service-worker.ts` (to-create): top-level listener registration, command dispatch, startup/install hooks, operation cancellation/recovery boundary và adapter construction. Không giữ projection/index state trong worker globals.
- `extension/src/browser/browser-adapter.ts` (to-create): typed browser capability seam cho command, window/tab focus, extension URL resolution, surface close và lifecycle hooks; Chrome/Edge implementation selection và fake implementation seam.
- `extension/src/search/surface-entrypoint.ts` (to-create): extension-page bootstrap, fresh search-session token, semantic query input lookup, focus handoff, close/dismiss event boundary và ready signal.
- `extension/src/search/surface-shell.html` (to-create): tối thiểu semantic shell có labeled query input, result/status mount points, keyboard-dismiss boundary và visible focus affordance; không nhúng page content hay remote assets.
- `fixtures/extension/command-open-surface.json` (to-create): command event, browser profile/window state, expected created-or-focused surface, focus target, no-selected-tab-change và dismissal observations.
- `tests/extension/manifest-permissions.test.ts` (to-create): parse built manifest và fail nếu permission/host permission vượt allowlist hoặc chứa quyền excluded.
- `tests/extension/command-open-surface.test.ts` (to-create): command-to-window-to-focus journey, duplicate invocation, fresh session, Esc dismissal và adapter injection.
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

- [ ] `IP-03-T01` **Manifest và package boundary:** tạo `extension/manifest.json` với `manifest_version: 3`, một `background.service_worker`, surface resource trong extension package và command name/description ổn định. Chọn `action`/icon metadata chỉ khi build target yêu cầu; không thêm product entrypoint thứ hai. Xác định permission allowlist bằng API probe, ưu tiên `tabs`, `windows`, `tabGroups`, `nativeMessaging`; `commands` là manifest key chứ không phải permission.
- [ ] `IP-03-T02` **Permission fail-closed:** viết manifest parser test đọc artifact build, assert command tồn tại đúng một lần, `permissions` là subset allowlist được duyệt, `host_permissions` absent/empty, và reject `history`, `downloads`, `cookies`, `scripting`, `activeTab`, `webNavigation`, browser-data-library capability, arbitrary URL patterns hoặc broad host access. Nếu API probe chứng minh một quyền không cần, loại nó khỏi manifest thay vì giữ cho tiện.
- [ ] `IP-03-T03` **Browser adapter contract:** định nghĩa typed methods cho `extensionUrl(path)`, `listSearchSurfaces()`, `createSearchSurface(url, bounds)`, `focusSearchSurface(windowId, tabId)`, `closeSearchSurface(windowId)`, `currentProfileContext()` và lifecycle signal registration. Mỗi method phải trả về bounded result/error, không expose raw `chrome` object; fake adapter phải ghi observable calls để test không cần browser thật.
- [ ] `IP-03-T04` **Command dispatch:** đăng ký một top-level `commands.onCommand` listener cho `open-search`; kiểm tra command name, lấy profile context từ adapter, tìm surface thuộc đúng extension URL/profile, rồi focus existing surface hoặc tạo focused popup surface. Dedupe concurrent invocation bằng operation token và always settle/release it on success, API error, cancellation hoặc worker suspension.
- [ ] `IP-03-T05` **Focused surface creation:** tạo surface bằng extension-owned URL, kích thước nhỏ hợp lý và `focused: true`; không navigate current tab. Surface route phải có `noopener`-equivalent browser boundary tự nhiên (extension page), không load remote script/style/font và không nhận URL tùy ý từ command input.
- [ ] `IP-03-T06` **Keyboard focus handoff:** trong `surface-entrypoint.ts`, tạo fresh session, reset query value, render accessible shell, chờ DOM ready rồi gọi focus query input với `preventScroll`; assert `document.activeElement === queryInput` trong fixture trước khi dispatch text. Focus failure, missing input hoặc closed document phải phát signal lỗi recoverable, không giả vờ đã sẵn sàng.
- [ ] `IP-03-T07` **Dismissal boundary:** wire `Esc` và explicit close action tới adapter close surface; dismissal không gọi tab activation, không đổi selected tab/window và không lưu query/title/URL vào log. IP-13 có thể mở rộng keyboard navigation nhưng không được thay đổi close invariant.
- [ ] `IP-03-T08` **Service-worker lifecycle:** đăng ký listeners synchronously at module evaluation, giữ worker stateless ngoài bounded in-flight operation map, xử lý `runtime.onStartup`/`onInstalled` để validate package/adapter readiness, và recreate adapter after suspension. Không dùng polling, websocket, loopback server, long-lived timer hay page-content listener để giữ worker sống.
- [ ] `IP-03-T09` **Browser parity:** implement namespace resolution/configuration cho Chrome và Edge cùng contract; fixture phải chạy cùng assertions với `BROWSER=chrome` và `BROWSER=edge`, chỉ thay binary/namespace. API divergence phải được normalize tại adapter, không rẽ nhánh trong command handler/surface.
- [ ] `IP-03-T10` **Handoff to later phases:** expose surface-ready/session/profile context without owning query protocol; expose adapter events for IP-04/IP-14 without implementing projection or stale-result activation. Ghi rõ invariant rằng extension browser authority quyết định window/tab activation và host không được mở surface.
- [ ] `IP-03-T11` **Targeted tests:** fixture phải cover command success, existing surface, duplicate invocation, missing/closed surface, permission denial/API rejection, worker restart, focus-before-text, missing query input, Esc dismissal, and no network/page read. Tests assert observable browser state and emitted status, not implementation details.

## 7. Kế hoạch commit

1. `feat(extension): implement ip-03-t01`
   - Task IDs: `IP-03-T01`.
   - Owned target paths: extension/manifest.json.
   - Behavior: **Manifest và package boundary:** tạo `extension/manifest.json` với `manifest_version: 3`, một `background.service_worker`, surface resource trong extension package và command name/description ổn định. Chọn `action`/icon metadata chỉ khi build target yêu cầu; không thêm product entrypoint thứ hai. Xác định permission allowlist bằng API probe, ưu tiên `tabs`, `windows`, `tabGroups`, `nativeMessaging`; `commands` là manifest key chứ không phải permission.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-03 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Manifest và package boundary:** tạo `extension/manifest.json` với `manifest_version: 3`, một `background.service_worker`, surface resource trong extension package và command name/description ổn định. Chọn `action`/icon metadata chỉ khi build target yêu cầu; không thêm product entrypoint thứ hai. Xác định permission allowlist bằng API probe, ưu tiên `tabs`, `windows`, `tabGroups`, `nativeMessaging`; `commands` là manifest key chứ không phải permission.
   - Dependency gate: all index.md dependencies for IP-03 have merged to dev; phase work branch starts from latest origin/dev.

2. `test(extension): implement ip-03-t02`
   - Task IDs: `IP-03-T02`.
   - Owned target paths: `extension/manifest.json` (to-create); `extension/src/background/service-worker.ts` (to-create); `extension/src/browser/browser-adapter.ts` (to-create); `extension/src/search/surface-entrypoint.ts` (to-create); `extension/src/search/surface-shell.html` (to-create); `fixtures/extension/command-open-surface.json` (to-create); `tests/extension/manifest-permissions.test.ts` (to-create); `tests/extension/command-open-surface.test.ts` (to-create); `tests/extension/service-worker-lifecycle.test.ts` (to-create).
   - Behavior: **Permission fail-closed:** viết manifest parser test đọc artifact build, assert command tồn tại đúng một lần, `permissions` là subset allowlist được duyệt, `host_permissions` absent/empty, và reject `history`, `downloads`, `cookies`, `scripting`, `activeTab`, `webNavigation`, browser-data-library capability, arbitrary URL patterns hoặc broad host access. Nếu API probe chứng minh một quyền không cần, loại nó khỏi manifest thay vì giữ cho tiện.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-03 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Permission fail-closed:** viết manifest parser test đọc artifact build, assert command tồn tại đúng một lần, `permissions` là subset allowlist được duyệt, `host_permissions` absent/empty, và reject `history`, `downloads`, `cookies`, `scripting`, `activeTab`, `webNavigation`, browser-data-library capability, arbitrary URL patterns hoặc broad host access. Nếu API probe chứng minh một quyền không cần, loại nó khỏi manifest thay vì giữ cho tiện.
   - Dependency gate: all index.md dependencies for IP-03 have merged to dev; phase work branch starts from latest origin/dev.

3. `test(extension): implement ip-03-t03`
   - Task IDs: `IP-03-T03`.
   - Owned target paths: `extension/manifest.json` (to-create); `extension/src/background/service-worker.ts` (to-create); `extension/src/browser/browser-adapter.ts` (to-create); `extension/src/search/surface-entrypoint.ts` (to-create); `extension/src/search/surface-shell.html` (to-create); `fixtures/extension/command-open-surface.json` (to-create); `tests/extension/manifest-permissions.test.ts` (to-create); `tests/extension/command-open-surface.test.ts` (to-create); `tests/extension/service-worker-lifecycle.test.ts` (to-create).
   - Behavior: **Browser adapter contract:** định nghĩa typed methods cho `extensionUrl(path)`, `listSearchSurfaces()`, `createSearchSurface(url, bounds)`, `focusSearchSurface(windowId, tabId)`, `closeSearchSurface(windowId)`, `currentProfileContext()` và lifecycle signal registration. Mỗi method phải trả về bounded result/error, không expose raw `chrome` object; fake adapter phải ghi observable calls để test không cần browser thật.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-03 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Browser adapter contract:** định nghĩa typed methods cho `extensionUrl(path)`, `listSearchSurfaces()`, `createSearchSurface(url, bounds)`, `focusSearchSurface(windowId, tabId)`, `closeSearchSurface(windowId)`, `currentProfileContext()` và lifecycle signal registration. Mỗi method phải trả về bounded result/error, không expose raw `chrome` object; fake adapter phải ghi observable calls để test không cần browser thật.
   - Dependency gate: all index.md dependencies for IP-03 have merged to dev; phase work branch starts from latest origin/dev.

4. `feat(extension): implement ip-03-t04`
   - Task IDs: `IP-03-T04`.
   - Owned target paths: `extension/manifest.json` (to-create); `extension/src/background/service-worker.ts` (to-create); `extension/src/browser/browser-adapter.ts` (to-create); `extension/src/search/surface-entrypoint.ts` (to-create); `extension/src/search/surface-shell.html` (to-create); `fixtures/extension/command-open-surface.json` (to-create); `tests/extension/manifest-permissions.test.ts` (to-create); `tests/extension/command-open-surface.test.ts` (to-create); `tests/extension/service-worker-lifecycle.test.ts` (to-create).
   - Behavior: **Command dispatch:** đăng ký một top-level `commands.onCommand` listener cho `open-search`; kiểm tra command name, lấy profile context từ adapter, tìm surface thuộc đúng extension URL/profile, rồi focus existing surface hoặc tạo focused popup surface. Dedupe concurrent invocation bằng operation token và always settle/release it on success, API error, cancellation hoặc worker suspension.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-03 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Command dispatch:** đăng ký một top-level `commands.onCommand` listener cho `open-search`; kiểm tra command name, lấy profile context từ adapter, tìm surface thuộc đúng extension URL/profile, rồi focus existing surface hoặc tạo focused popup surface. Dedupe concurrent invocation bằng operation token và always settle/release it on success, API error, cancellation hoặc worker suspension.
   - Dependency gate: all index.md dependencies for IP-03 have merged to dev; phase work branch starts from latest origin/dev.

5. `feat(extension): implement ip-03-t05`
   - Task IDs: `IP-03-T05`.
   - Owned target paths: `extension/manifest.json` (to-create); `extension/src/background/service-worker.ts` (to-create); `extension/src/browser/browser-adapter.ts` (to-create); `extension/src/search/surface-entrypoint.ts` (to-create); `extension/src/search/surface-shell.html` (to-create); `fixtures/extension/command-open-surface.json` (to-create); `tests/extension/manifest-permissions.test.ts` (to-create); `tests/extension/command-open-surface.test.ts` (to-create); `tests/extension/service-worker-lifecycle.test.ts` (to-create).
   - Behavior: **Focused surface creation:** tạo surface bằng extension-owned URL, kích thước nhỏ hợp lý và `focused: true`; không navigate current tab. Surface route phải có `noopener`-equivalent browser boundary tự nhiên (extension page), không load remote script/style/font và không nhận URL tùy ý từ command input.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-03 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Focused surface creation:** tạo surface bằng extension-owned URL, kích thước nhỏ hợp lý và `focused: true`; không navigate current tab. Surface route phải có `noopener`-equivalent browser boundary tự nhiên (extension page), không load remote script/style/font và không nhận URL tùy ý từ command input.
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
   - Behavior: **Dismissal boundary:** wire `Esc` và explicit close action tới adapter close surface; dismissal không gọi tab activation, không đổi selected tab/window và không lưu query/title/URL vào log. IP-13 có thể mở rộng keyboard navigation nhưng không được thay đổi close invariant.
   - Fixture and command: IP-13; run `the exact phase-03 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Dismissal boundary:** wire `Esc` và explicit close action tới adapter close surface; dismissal không gọi tab activation, không đổi selected tab/window và không lưu query/title/URL vào log. IP-13 có thể mở rộng keyboard navigation nhưng không được thay đổi close invariant.
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
   - Behavior: **Browser parity:** implement namespace resolution/configuration cho Chrome và Edge cùng contract; fixture phải chạy cùng assertions với `BROWSER=chrome` và `BROWSER=edge`, chỉ thay binary/namespace. API divergence phải được normalize tại adapter, không rẽ nhánh trong command handler/surface.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-03 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Browser parity:** implement namespace resolution/configuration cho Chrome và Edge cùng contract; fixture phải chạy cùng assertions với `BROWSER=chrome` và `BROWSER=edge`, chỉ thay binary/namespace. API divergence phải được normalize tại adapter, không rẽ nhánh trong command handler/surface.
   - Dependency gate: all index.md dependencies for IP-03 have merged to dev; phase work branch starts from latest origin/dev.

10. `feat(extension): implement ip-03-t10`
   - Task IDs: `IP-03-T10`.
   - Owned target paths: `extension/manifest.json` (to-create); `extension/src/background/service-worker.ts` (to-create); `extension/src/browser/browser-adapter.ts` (to-create); `extension/src/search/surface-entrypoint.ts` (to-create); `extension/src/search/surface-shell.html` (to-create); `fixtures/extension/command-open-surface.json` (to-create); `tests/extension/manifest-permissions.test.ts` (to-create); `tests/extension/command-open-surface.test.ts` (to-create); `tests/extension/service-worker-lifecycle.test.ts` (to-create).
   - Behavior: **Handoff to later phases:** expose surface-ready/session/profile context without owning query protocol; expose adapter events for IP-04/IP-14 without implementing projection or stale-result activation. Ghi rõ invariant rằng extension browser authority quyết định window/tab activation và host không được mở surface.
   - Fixture and command: IP-04, IP-14; run `the exact phase-03 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Handoff to later phases:** expose surface-ready/session/profile context without owning query protocol; expose adapter events for IP-04/IP-14 without implementing projection or stale-result activation. Ghi rõ invariant rằng extension browser authority quyết định window/tab activation và host không được mở surface.
   - Dependency gate: all index.md dependencies for IP-03 have merged to dev; phase work branch starts from latest origin/dev.

11. `test(extension): implement ip-03-t11`
   - Task IDs: `IP-03-T11`.
   - Owned target paths: `extension/manifest.json` (to-create); `extension/src/background/service-worker.ts` (to-create); `extension/src/browser/browser-adapter.ts` (to-create); `extension/src/search/surface-entrypoint.ts` (to-create); `extension/src/search/surface-shell.html` (to-create); `fixtures/extension/command-open-surface.json` (to-create); `tests/extension/manifest-permissions.test.ts` (to-create); `tests/extension/command-open-surface.test.ts` (to-create); `tests/extension/service-worker-lifecycle.test.ts` (to-create).
   - Behavior: **Targeted tests:** fixture phải cover command success, existing surface, duplicate invocation, missing/closed surface, permission denial/API rejection, worker restart, focus-before-text, missing query input, Esc dismissal, and no network/page read. Tests assert observable browser state and emitted status, not implementation details.
   - Fixture and command: the observable fixture/outcome stated by this task; run `the exact phase-03 fixture/check command in Section 8 after its source prerequisite exists`. This is a future check until its declared source and fixture prerequisites exist.
   - Observable result before commit: **Targeted tests:** fixture phải cover command success, existing surface, duplicate invocation, missing/closed surface, permission denial/API rejection, worker restart, focus-before-text, missing query input, Esc dismissal, and no network/page read. Tests assert observable browser state and emitted status, not implementation details.
   - Dependency gate: all index.md dependencies for IP-03 have merged to dev; phase work branch starts from latest origin/dev.

## 8. Kiểm chứng và nghiệm thu

- [ ] From repository root, after implementation creates the declared test script, run `NODE_ENV=test BROWSER=chrome npm run test:extension -- tests/extension/manifest-permissions.test.ts tests/extension/command-open-surface.test.ts tests/extension/service-worker-lifecycle.test.ts --fixture fixtures/extension/command-open-surface.json` with a clean extension build. The command must report the manifest allowlist, command journey, lifecycle, focus and dismissal assertions as passing.
- [ ] Repeat the same fixture without changing assertions for Edge: `NODE_ENV=test BROWSER=edge npm run test:extension -- tests/extension/manifest-permissions.test.ts tests/extension/command-open-surface.test.ts tests/extension/service-worker-lifecycle.test.ts --fixture fixtures/extension/command-open-surface.json`. The runner must use the declared Edge binary and fail if the adapter depends on Chrome-only behavior.
- [ ] In each browser fixture, invoke the configured `open-search` command without pointer input. Observe an extension-owned focused surface for the active profile, exactly one surface after two rapid invocations, and `document.activeElement.id === "query-input"` before dispatching a sample query. The first typed character must be accepted by that input.
- [ ] Repeat after service-worker suspension/restart. The command must still open/focus the surface, must not duplicate it, must not change the selected tab/window, and must emit a recoverable status if the browser API rejects creation or focus.
- [ ] Trigger `Esc` and explicit dismissal from the focused surface. Observe that the surface closes and the previously selected tab/window remains selected; no activation call is made and no raw query/title/URL is written to diagnostics.
- [ ] Parse the built Manifest V3 artifact in the permission fixture. Acceptance requires exactly one command, only the reviewed minimum permission subset, no `host_permissions` patterns, and explicit absence of `history`, `downloads`, cookies, browser-data-library/page-content capability, and broad host access. A newly added permission must fail the fixture until its decision/contract is updated.
- [ ] Run the accessibility smoke in the same harness: query input has an accessible name, visible focus remains present at default and keyboard navigation settings, focus does not depend on animation, and the shell remains usable at a narrow popup viewport. Full result-list accessibility and reduced-motion evidence remain IP-13 acceptance, but this phase must not regress the handoff.
- [ ] Confirm resource-bound evidence by intercepting extension requests during the fixture: zero network requests, zero page navigations/injections, zero page-content reads, and no semantic/remote service invocation. The command path must use only extension-owned resources and browser APIs.
- [ ] Acceptance mapping: FR-001 is proven by Chrome/Edge command settings plus command journey; FR-002 by focus-before-text observation; FR-006 by stable labeled result/status mount points for later context rendering; FR-008 by Esc/explicit dismissal preserving browser selection; NFR-007 by permission and request interception checks; NFR-008 by keyboard focus/name/focus-visible checks; NFR-010 by the Chrome/Edge matrix on Linux/macOS/Windows release runners.

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
