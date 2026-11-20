# Component Ownership

Component ownership is strictly defined to ensure maintainability, testability, and zero unnecessary dependencies.

---

## 1. Popup Search Surface (`extension/src/search/`)

### `surface-shell.html` & `surface-shell.css`
- **Responsibilities**:
  - Semantic HTML layout for search input, tab list (`role="listbox"`), empty state, and keyboard shortcuts footer.
  - Sleek, dark/light aware responsive CSS with high-contrast active item states (`aria-selected="true"`) and matched character highlights (`<mark>`).
  - Fixed dimensions (480px width, 540px height) designed for rapid keyboard invocation.

### `surface-entrypoint.ts`
- **Responsibilities**:
  - Automatically focuses the search input on popup opening.
  - Queries open tabs via `chrome.tabs.query({})` and reads MRU ranks from `chrome.storage.session`.
  - Pre-selects the previously active tab (`selectedIndex = 1` if tabs > 1) to enable instant `Alt+Tab` style switching.
  - Dispatches keyboard navigation (`ArrowDown`, `ArrowUp`, `Ctrl+N`, `Ctrl+P`, `Home`, `End`) without re-rendering the DOM list.
  - Handles `Enter` to activate the tab via `chrome.tabs.update()` and focus the window via `chrome.windows.update()`, followed by `window.close()`.
  - Handles `Escape` to close the search surface immediately without side effects.

---

## 2. In-Memory Search & Scorer Engine (`extension/src/search/scorer.ts`, `types.ts`)

### `types.ts`
- **Responsibilities**:
  - Defines `SearchTabItem` data structures.
  - Provides `extractDomain(url)` to cleanly strip protocols, paths, query params, ports, and `www.` prefixes.
  - Provides `normalizeTab(tab, mruMap)` to filter out invalid/extension tabs and assign MRU rank.

### `scorer.ts`
- **Responsibilities**:
  - Scores a `SearchTabItem` against a query string using a deterministic 4-tier model:
    1. **Exact Prefix Match** (1000 pts for domain, 950 pts for title).
    2. **Word-boundary Match** (850 pts for domain, 800 pts for title).
    3. **Acronym Match** (650 pts for title initials: e.g. `gh` -> `GitHub`, `yt` -> `YouTube`).
    4. **Subsequence / FZF-style Match** (400 pts minus gap penalties).
  - Calculates **MRU Recency Boost**: `+ 50 / (mruRank + 1)`.
  - Provides `searchTabs(items, query)` to filter and sort tabs in descending score order.
  - Returns raw MRU order when query is empty.

---

## 3. Background MRU Session Manager (`extension/src/background/`)

### `mru-manager.ts`
- **Responsibilities**:
  - Manages a bounded ring buffer of tab IDs (`MAX_MRU_SIZE = 100`) stored under key `'mru_tab_ids'` in `chrome.storage.session`.
  - `touchTab(tabId)`: Moves the specified `tabId` to index 0, deduplicating any prior occurrences.
  - `removeTab(tabId)`: Evicts the closed `tabId` from the MRU stack.
  - `hydrateFromTabs(tabs)`: Populates the initial ring buffer from `tab.lastAccessed` descending if session storage is empty.

### `service-worker.ts`
- **Responsibilities**:
  - Registers listeners for `chrome.tabs.onActivated`, `chrome.tabs.onRemoved`, and `chrome.windows.onFocusChanged`.
  - Registers `runtime.onStartup` and `runtime.onInstalled` hooks to trigger MRU hydration.
  - Remains stateless and sleep-friendly (does not keep long-running interval timers).
