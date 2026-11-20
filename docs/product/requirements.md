# Product Requirements

This document outlines the functional and quality requirements for the InfoBoard Pure Manifest V3 tab search extension.

---

## 1. Functional Requirements (FR)

### Ingestion & Tab Tracking
- **FR-001 (Tab Observation):** Automatically observe browser tab lifecycle events (`tabs.onActivated`, `tabs.onRemoved`, `windows.onFocusChanged`) to maintain an accurate MRU (Most Recently Used) ring buffer (max 100 entries).
- **FR-002 (In-Memory Session Storage):** Store the active MRU queue in `chrome.storage.session` for instant access across background service worker restarts without disk persistence.
- **FR-003 (Tab Hydration):** On extension launch or background worker wake-up, query all existing open tabs across all non-incognito browser windows via `chrome.tabs.query`.

### Search Surface & Ergonomics
- **FR-004 (Shortcut Invocation):** Support opening the search popup via configurable shortcut (`Alt+W` on Linux/Windows, `Command+Shift+K` on macOS) or clicking the action icon.
- **FR-005 (Autofocus & Instant Render):** Autofocus the search query input immediately upon opening with 0ms debounce and zero IPC overhead.
- **FR-006 (Quick Toggle Selection):** When opened without a query (empty input), pre-select index 1 (the previously active tab), allowing the user to hit `Enter` immediately to switch between their two most recent tabs (OS `Alt+Tab` experience).
- **FR-007 (Keyboard Navigation):** Provide complete keyboard control:
  - `ArrowDown` / `Ctrl+N`: Move selection downward (with bounds clamping).
  - `ArrowUp` / `Ctrl+P`: Move selection upward.
  - `Enter`: Activate selected tab and window, then dismiss popup.
  - `Escape`: Dismiss popup with zero side-effects.

### Search Scoring & Ranking
- **FR-008 (Multi-Tier Scoring):** Support typing short queries and prefixes with deterministic scoring:
  - Exact prefix match on clean domain (1,000 pts) and title (950 pts).
  - Word-boundary prefix match on domain (850 pts) and title (800 pts).
  - Acronym / initialism match on title words (e.g., `gh` -> "GitHub", `yt` -> "YouTube") (650 pts).
  - Subsequence match (400 pts).
  - Dynamic MRU Recency Boost: `+ 50 / (mruRank + 1)`.
- **FR-009 (Match Highlighting):** Highlight the matched characters in the title and URL within the search result rows.

### Activation & Safety
- **FR-010 (Tab & Window Activation):** Focus the target tab via `chrome.tabs.update(tabId, { active: true })` and bring its window to focus via `chrome.windows.update(windowId, { focused: true })`.
- **FR-011 (Stale Tab Guard):** Gracefully handle tabs closed while the popup was open without crashing or leaving phantom state.

---

## 2. Non-Functional Requirements (NFR)

### Performance & Latency
- **NFR-001 (Search Latency):** Query scoring across 1,000 tabs must execute in < 1ms on Chromium V8.
- **NFR-002 (Render Latency):** Input-to-DOM render latency must be p95 < 16ms (within a single animation frame).
- **NFR-003 (Bundle Footprint):** Total uncompressed bundle size must be < 50KB (current production build is ~18KB).
- **NFR-004 (Zero Native Overhead):** Zero background daemon binaries, zero native messaging processes, zero SQLite databases.

### Privacy & Permissions
- **NFR-005 (Least Privilege):** Request only minimal necessary permissions:
  - `tabs`: Read tab titles, URLs, and active status.
  - `storage`: Persist MRU ring buffer in `chrome.storage.session`.
  - `tabGroups`: Read tab group names and colors for disambiguation.
- **NFR-006 (Zero Network Egress):** Absolutely zero network requests, tracking, analytics, or external telemetry.
- **NFR-007 (Local Isolation):** All processing and tab scoring happen locally in-memory inside the browser tab context.

### Accessibility & Compatibility
- **NFR-008 (Accessibility):** Search results list formatted as an ARIA listbox (`role="combobox"`, `role="listbox"`, `aria-selected`, `aria-activedescendant`).
- **NFR-009 (Cross-Platform):** Compatible with Google Chrome and Microsoft Edge on Linux, macOS, and Windows.
