# Binding decision log

A decision is binding when it records user value, scope, privacy impact, architecture impact, failure behavior, acceptance signal, and revisit trigger. Open proposals do not override the product contract.

| ID | Status | Decision | Rationale | Revisit trigger |
| --- | --- | --- | --- | --- |
| `REF-001` | accepted | InfoBoard is a keyboard-first open-tab search and activation product. | Solves tab overload without broad browsing-data access. | Evidence shows a different primary problem dominates. |
| `REF-002` | accepted | Chrome and Edge desktop are supported; Linux, macOS, and Windows are release targets. | Shared extension model with complete desktop distribution. | Browser/platform support cost prevents acceptance gates. |
| `REF-003` | accepted | Search uses current-profile open tabs and lightweight tab metadata only. | Sufficient user value with bounded privacy surface. | Users cannot find required targets without a named new data source. |
| `REF-004` | accepted | The local runtime is a Go Native Messaging host, not a loopback daemon. | Uses a browser-supported local boundary and avoids an application-owned port. | Installation or lifecycle measurements fail the release criteria. |
| `REF-005` | accepted | The hot path is deterministic lexical ranking with context and recency signals. | Fast, explainable, testable, and sufficient for remembered tab fragments. | Fixed fixtures show a concrete repeated failure that lexical signals cannot solve. |
| `REF-006` | accepted | SQLite stores configuration and bounded local activation metadata; live tabs remain rebuildable memory state. | Persistence supports continuity without making a database the source of browser truth. | Measured behavior requires another durable data class. |
| `REF-007` | accepted | No history, page contents, cookies, page local/session storage, network interception, cloud indexing, or telemetry-by-default. | Protects privacy and keeps permissions narrow; REF-013 defines the single extension-owned identity-only storage exception. | A new product proposal documents value, consent, retention, and threat model. |
| `REF-008` | accepted | Search opens as an extension-owned browser action popup, not a detached OS window, side panel, or page-injected overlay. | Matches the in-browser browser-chrome surface while keeping the active tab's DOM untouched and the user's tab selected. | Chrome/Edge action-popup behavior cannot meet focus, size, or keyboard acceptance. |
| `REF-009` | accepted | Protocol and ranking versions are explicit and incompatible pairs fail closed. | Prevents silent stale results and ambiguous behavior during update. | A compatibility model with stronger safety is adopted. |
| `REF-010` | accepted | Release requires install, update, repair, reset, uninstall, recovery, accessibility, privacy, and performance acceptance. | A complete product includes lifecycle behavior, not only the search path. | Release model or distribution channel changes. |
| `REF-011` | deferred | Semantic search and page-content indexing are not part of the product contract. | No proven need justifies extra collection, latency, or operational complexity. | A measured user problem and privacy-approved design exist. |
| `REF-012` | deferred | Other browsers, mobile, bookmarks, downloads, and cross-profile search are separate products. | They require different UX, permissions, and packaging contracts. | A new product boundary is approved. |
| `REF-013` | accepted | The extension MAY persist only its locally generated opaque `profile_id` in `chrome.storage.local`, under the single key `profile_id`, so profile identity survives browser/service-worker restarts. | The extension must preserve a stable profile partition without relying on page storage or an account/path identifier; the value is never synced, uploaded, derived from account/path data, or used for any other setting, control, or content. It is deleted only after sessions disconnect during explicit reset or uninstall. Page `window.localStorage`, page/session storage, cookies, `storage.sync`, history, page content, network, and cross-profile data remain prohibited. | A browser API or lifecycle change prevents identity-only local persistence without expanding the stored data class. |
| `REF-014` | accepted | The `open-search` command opens InfoBoard as the browser's extension action popup, anchored in the browser UI, not as a detached OS window, side panel, or injected page overlay. | The toolbar popup matches the approved preview while preserving the active tab and requiring no page injection or page-data access; `action` is a manifest key, not a permission. | Chrome/Edge action-popup API cannot meet the 480×600, focus, or keyboard acceptance criteria.
## Rejected shortcuts

- Treating the current refactor idea brief as an MVP specification.
- Starting with a storage or database technology before user behavior is defined.
- Spawning a Go process for every query.
- Exposing a loopback listener without a separate authentication and threat-model decision.
- Indexing browser history or page content to compensate for weak lexical ranking.
