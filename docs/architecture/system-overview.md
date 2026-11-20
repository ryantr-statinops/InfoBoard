# System Overview

## System Topology

InfoBoard runs entirely inside the Chromium extension environment. There are no native companion applications, background OS daemons, local network servers, or SQLite databases.

```text
┌────────────────────────────────────────────────────────┐
│               Chromium Browser (Chrome / Edge)         │
│                                                        │
│  ┌──────────────────────────────────────────────────┐  │
│  │ 1. Search Surface (Action Popup)                 │  │
│  │    • Triggered via _execute_action (Alt+W)       │  │
│  │    • Input autofocus, arrow cycling, Escape/Enter│  │
│  │    • Renders tabs, favicons, badges, highlights  │  │
│  └────────────────────────▲─────────────────────────┘  │
│                           │ in-memory (< 1ms)          │
│  ┌────────────────────────▼─────────────────────────┐  │
│  │ 2. Scorer & Tab Indexer                          │  │
│  │    • Multi-tier prefix/boundary/acronym matching │  │
│  │    • MRU Recency Boost (+ 50 / (rank + 1))       │  │
│  │    • Deterministic tie-breaking                  │  │
│  └────────────────────────▲─────────────────────────┘  │
│                           │ tab snapshot & MRU ranks   │
│  ┌────────────────────────▼─────────────────────────┐  │
│  │ 3. Background Service Worker (MRU Tracker)       │  │
│  │    • chrome.tabs.onActivated / onRemoved         │  │
│  │    • chrome.windows.onFocusChanged               │  │
│  │    • Bounded ring buffer in chrome.storage.session│ │
│  │    • Fallback: tab.lastAccessed                  │  │
│  └──────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────┘
```

## Component Boundaries

| Component | Owns | Does Not Own |
| :--- | :--- | :--- |
| **Search Surface (Popup)** | Focus management, query input, keyboard navigation, tab rendering, and tab activation via `chrome.tabs.update` | Tab observation events or long-term history storage |
| **Scorer Engine** | String normalization, exact prefix scoring, word-boundary regex matching, acronym matching, and MRU boost calculations | Browser permissions, DOM manipulation, or disk storage |
| **Background Service Worker** | Tab activation and window focus listeners, MRU ring-buffer state management | Persistent history, page content inspection, or UI layout |
| **Session Storage** | Holding up to 100 recent tab IDs in browser RAM across service worker sleep cycles | Disk persistence or sync across devices |

## Design Invariants

1. **Zero External Processes**: The extension never spawns OS processes, opens network sockets, or relies on native messaging.
2. **Zero Disk I/O on Search**: Querying and ranking occur entirely in-memory using JavaScript V8 string/regex operations.
3. **Sub-millisecond Hot Path**: Searching through 100+ open tabs executes in under 1 millisecond.
4. **Quick Toggle Ergonomics**: Opening the popup automatically pre-selects the previously active tab (`index 1`), allowing instant toggle back and forth using shortcut + `Enter`.
5. **Fail-safe Degradation**: If session storage is empty or fails, the system immediately falls back to Chromium's native `tab.lastAccessed` timestamp without interrupting search.
