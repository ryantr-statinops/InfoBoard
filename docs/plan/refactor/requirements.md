# Product requirements

## Requirement notation

- `MUST`: binding product behavior.
- `SHOULD`: expected behavior unless a decision record documents why it is not suitable.
- `MAY`: optional implementation detail that must not weaken a `MUST`.

## Functional requirements

| ID | Requirement | Acceptance signal |
| --- | --- | --- |
| FR-001 | The extension MUST expose one user-configurable browser command for opening search. | Chrome and Edge command settings show the command and invoke the search surface. |
| FR-002 | The search surface MUST focus its query input before accepting text. | Keyboard trace reaches a focused input without pointer interaction. |
| FR-003 | The system MUST search the current profile's open-tab projection by title, URL, domain, window, group, and state labels. | Fixture queries return expected field matches and exclude unavailable data. |
| FR-004 | The system MUST update results as the query changes without requiring submit. | Each accepted query revision renders a corresponding result revision. |
| FR-005 | The ranking engine MUST use deterministic lexical, context, and recency rules. | Repeating the same fixture and inputs produces the same ordered IDs. |
| FR-006 | The UI MUST show tab title, domain or URL context, window/group context when available, and pinned state when relevant. | Duplicate-title fixture remains distinguishable. |
| FR-007 | The user MUST activate a highlighted result with `Enter` and move selection with arrow keys. | Keyboard journey activates the expected tab and window. |
| FR-008 | `Esc` and explicit dismissal MUST close the surface without changing the selected tab. | Dismissal fixture leaves browser state unchanged. |
| FR-009 | The extension MUST reconcile tab create, update, move, group, pin, activate, window, and remove events. | Event fixture reaches the expected projection without duplicate records. |
| FR-010 | The system MUST rebuild from a full tab snapshot after missed events or runtime reconnect. | Dropped-event fixture converges to the browser snapshot. |
| FR-011 | A result removed before activation MUST be treated as stale and never activate a different tab silently. | Race fixture reports refresh or chooses only an explicitly valid next result. |
| FR-012 | The system MUST expose runtime availability and index freshness to the search surface. | Host-down and stale-index states are visible and actionable. |
| FR-013 | Normal lexical search MUST remain available when optional semantic or durable-storage components fail. | Optional-component failure fixture still returns and activates lexical results. |
| FR-014 | Configuration MUST remain scoped to the current browser profile. | Two profiles do not share settings or tab projections. |
| FR-015 | Reset and uninstall MUST remove InfoBoard-owned local data without affecting browser tabs or unrelated browser data. | Removal fixture verifies owned paths only. |

## Quality requirements

| ID | Requirement | Target |
| --- | --- | --- |
| NFR-001 | Shortcut-to-focused-input latency | p95 <= 100 ms on supported reference hardware when extension is healthy. |
| NFR-002 | Query-to-render latency | p95 <= 50 ms for 1,000 indexed tabs and a 64-character query. |
| NFR-003 | Selection-to-activation latency | p95 <= 100 ms excluding browser scheduling delays. |
| NFR-004 | Ranking determinism | Same projection, query, and ranking timestamp produce identical order and scores. |
| NFR-005 | Projection convergence | After a full snapshot, indexed IDs equal browser-visible eligible IDs. |
| NFR-006 | Runtime recovery | Host reconnect or rebuild reaches healthy state without user reinstall in the normal failure case. |
| NFR-007 | Resource bound | The hot path does not perform network access, page-content fetches, or semantic inference. |
| NFR-008 | Accessibility | Keyboard operation, visible focus, contrast, reduced motion, and screen-reader labels meet the supported browser accessibility baseline. |
| NFR-009 | Privacy | No excluded data source is requested, collected, persisted, or transmitted. |
| NFR-010 | Compatibility | Chrome and Edge supported on Linux, macOS, and Windows for the declared release matrix. |

## Permission requirements

The extension MUST request only permissions required by the product contract. Any permission for history, bookmarks, downloads, page content, cookies, storage beyond the declared local store, or broad host access requires a new decision record and updated privacy documentation.

## Operational requirements

- Every protocol request MUST have a bounded timeout and an observable error class.
- Every runtime state transition MUST be diagnosable without collecting tab content outside the local machine.
- Schema and protocol changes MUST be versioned and backward-incompatible changes MUST fail closed with a recovery path.
- Installer, updater, reset, and uninstaller MUST be idempotent.
- Logs MUST default to metadata such as counts, durations, versions, and error classes; URLs and titles are redacted or disabled by default.
