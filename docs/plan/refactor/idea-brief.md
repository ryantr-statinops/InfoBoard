# Product contract

## Product

InfoBoard helps a person find and activate an open browser tab without scanning tab strips, windows, or browser history. The product is a keyboard-first Chromium desktop extension backed by a local Go runtime.

## User problem

People lose time when many tabs are distributed across windows and groups. Browser navigation favors spatial and recent movement, while the user's memory is often a fragment of a title, domain, URL, project name, or topic. The product must turn that fragment into a predictable tab switch without adding a cloud service or broad browsing-data access.

## Primary user

A desktop user who regularly keeps many tabs open across several windows and wants a fast, keyboard-driven way to return to a known tab. The user values speed, predictable ordering, privacy, and recovery when the local runtime is unavailable.

## Product outcome

For any currently open tab that is visible to the extension, the user can:

1. invoke the configured shortcut;
2. type a remembered fragment;
3. understand why the top results are ordered as shown;
4. activate the intended tab; and
5. return to the previous browser task without losing context.

## Complete product capabilities

### 1. Invocation and search surface

- Register a configurable browser command.
- Open a focused extension-owned search surface in the current browser profile.
- Focus the query field automatically and preserve the previous query only when the same search session is reopened by an explicit user action.
- Support keyboard-only operation and mouse/pointer selection.
- Close without side effects on `Esc`, outside dismissal, or browser navigation away from the search surface.

### 2. Live tab projection

- Track open tabs across all windows in the current browser profile.
- Process tab creation, title/URL change, move, group change, pin change, activation, window change, and removal.
- Reconcile after extension service-worker restart, browser restart, runtime reconnect, and missed events.
- Keep incognito/private profiles isolated from normal profiles and follow browser permission rules.

### 3. Search and ranking

- Search title, URL, domain, window label, tab-group label, and bounded state labels.
- Support empty query, exact tokens, prefixes, multi-token queries, and typo-tolerant fuzzy matching.
- Rank exact and field-appropriate lexical matches before context and recency signals.
- Show enough metadata to distinguish duplicate titles and domains.
- Keep ranking deterministic and bounded for the supported open-tab dataset.

### 4. Activation

- Activate the selected tab and its window.
- Preserve pinned, grouped, and window state.
- Handle a result disappearing between render and activation by refreshing the projection and selecting a valid next result.
- Report activation failure without claiming success.

### 5. Configuration

- Configure the browser command through the browser's supported command settings.
- Configure result count, compact/comfortable density, theme preference, and whether context labels are shown.
- Provide runtime status, index freshness, privacy boundary, and reset/uninstall controls.
- Keep configuration local to the browser profile unless a future explicit sync contract is accepted.

### 6. Local runtime and recovery

- Use a registered Go Native Messaging host for extension-to-runtime communication.
- Keep the hot lexical index in memory and rebuild it from a full tab snapshot when required.
- Reconnect after host exit, service-worker restart, browser restart, protocol mismatch, or missed events.
- Keep normal lexical search usable when optional semantic or persistence components fail.

### 7. Distribution and removal

- Ship a Chrome extension package and an Edge-compatible package from the same product source.
- Ship signed or integrity-checked native-host installers for Linux, macOS, and Windows.
- Support update, rollback-safe migration, disable, uninstall, and complete local-data removal.
- Explain permissions and local runtime behavior during installation.

## Product boundaries

Included:

- Chrome and Edge desktop.
- Linux, macOS, and Windows packaging.
- Open-tab metadata in the current browser profile.
- Keyboard-first search, ranking, and tab activation.
- Local Go runtime, Native Messaging, bounded SQLite persistence, diagnostics, recovery, and uninstall.

Excluded from this contract:

- Whole browsing-history indexing.
- Cookies, local storage, session tokens, network interception, or page-source collection.
- Full page text or DOM indexing.
- Cloud indexing, telemetry by default, or account-based synchronization.
- Search across bookmarks, downloads, or other browser profiles.
- Financial, personal, or external-service integrations unrelated to tab switching.
- Semantic search as a dependency of the normal query path.

## Success definition

The product is successful when a user can repeatedly find and activate an intended open tab with a short query, understands the ordering when results are close, experiences no data-boundary surprise, and can recover from runtime failure without losing tabs or browser state.

Detailed measurable criteria are defined in [requirements](requirements.md) and [verification and acceptance](verification-and-acceptance.md).
