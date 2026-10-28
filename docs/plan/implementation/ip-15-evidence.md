# IP-15 evidence - Configuration, reset, and uninstall

Status: implementation and phase acceptance checks passed on `dev`.

## Delivered contract

- Profile-scoped configuration defaults are `result_limit=10`, `density=comfortable`, `theme=system`, `context_labels=true`, and `recency_enabled=true`.
- Lifecycle operations are `load`, `update`, `reset`, and `uninstall`, carried by `lifecycle` / `lifecycle_result` within the frozen IP-07 envelope.
- The settings surface is exposed through `options_ui`; the IP-13 search footer has a Settings control that calls `chrome.runtime.openOptionsPage()`.
- Reset publishes defaults, requests a fresh snapshot, and marks the session for resync.
- Uninstall removes only reviewed storage and manifest paths, treats absent paths as success, and shuts the session down after the response.
- The full `host/cmd` protocol/projection/query composition remains deferred to IP-18/IP-20; IP-15 verifies the lifecycle seam with a fake native host.

## Commands and results

```sh
npm run build:extension
node --test tests/settings/phase-15/*.test.mjs tests/protocol/*.test.mjs
go test ./host/internal/protocol ./host/lifecycle ./tests/lifecycle/phase-15 -count=1
go vet ./host/internal/protocol ./host/lifecycle ./tests/lifecycle/phase-15
BROWSER=chromium npx playwright test --config tests/extension/playwright.config.ts phase-15/settings-entry.test.ts
python3 tests/implementation/validate_corpus.py
git diff --check
```

Observed:

- Extension build passed.
- Settings/protocol Node suite: 28/28 passed.
- Go protocol, lifecycle, and phase-15 integration packages passed.
- `go vet` passed and changed Go files are gofmt-clean.
- Playwright phase-15 settings/entry suite: 3/3 passed.
- Corpus validator: PASS (20 phases, 236 tasks, 25 requirements, 77 fixtures, 7 invariants, 19 phase seams).
- `git diff --check` passed.

## Coverage

| Acceptance area | Evidence |
|---|---|
| FR-014 profile isolation | `TestProtocolFakeHostLifecycleIntegration` updates A and B independently; A reset leaves B unchanged. |
| Exact defaults | `TestProtocolFakeHostLifecycleIntegration`, `TestProtocolFakeHostDegradedPersistenceKeepsDefaults`, and settings contract/options tests. |
| Reset and resync | Protocol integration asserts `resync_required`, defaults after reset, and a convergent second reset. |
| Uninstall ownership and idempotency | Protocol integration removes the database plus one manifest entry, preserves an unrelated sentinel, shuts the session down, and completes a second uninstall. |
| Path safety | `TestProtocolFakeHostRejectsUnsafeUninstallPath` rejects a relative database path before cleanup. |
| Degraded persistence | Protocol integration returns safe defaults with `degraded` persistence and no durable claim after the store is closed. |
| Settings entry | Playwright asserts the options page defaults/save/reset/uninstall flow and the search-footer Settings control. |
| Zero browser mutation | Settings source scan rejects tab/history/window mutation APIs; Playwright asserts lifecycle actions do not change the page set or navigate the settings page. |
| Wire shape | Lifecycle results always encode `completed_steps` as an array, including load/update with no completed steps. |

## Known environment limits

- Pre-existing `host/storage` Windows permission/temp-file cleanup tests remain outside IP-15.
- `host/cmd/infoboard-host` tests remain blocked by the local Application Control policy.
- Branded Chrome/Edge verification remains IP-19 work.
