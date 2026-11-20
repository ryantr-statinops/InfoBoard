# InfoBoard Documentation

InfoBoard is a keyboard-first, ultra-lightweight Chrome/Edge extension for searching and activating open browser tabs across windows.

This documentation describes the **Pure Manifest V3 Architecture**, product contracts, search scoring algorithms, UX design, operations, and quality assurance.

---

## Architecture at a Glance

InfoBoard operates 100% inside the browser sandbox:
- **No Native Messaging Host**: Zero external background binaries or daemons.
- **No SQLite / Disk Database**: State is maintained in-memory and in `chrome.storage.session`.
- **Instant Response (< 1ms)**: Multi-tier lexical search running directly on Chromium's V8 engine.

```text
Chrome / Edge Extension (Manifest V3)
  ├── Popup Search Surface (HTML / CSS / TS)
  │     ├── Input autofocus & instant live filtering (0ms debounce)
  │     ├── ArrowUp / ArrowDown / Ctrl+N / Ctrl+P navigation
  │     └── Quick Toggle: Pre-selects previous tab for instant Enter switch
  ├── In-Memory Scorer & Tab Indexer
  │     ├── Exact Prefix on Domain/Title (1000 pts)
  │     ├── Word-boundary Prefix Match (800-850 pts)
  │     ├── Acronym Match (650 pts)
  │     └── MRU Recency Boost (+ 50 / (rank + 1))
  └── Background Service Worker (MRU Tracker)
        ├── chrome.tabs.onActivated / onRemoved
        ├── chrome.windows.onFocusChanged
        └── chrome.storage.session ring buffer (max 100 tabs)
```

---

## Documentation Map

| Directory | Scope & Purpose |
| :--- | :--- |
| [`architecture/`](architecture/README.md) | System shape, component ownership, data flow, and lifecycle boundaries |
| [`product/`](product/README.md) | User problem, requirements, product contract, and architectural decisions |
| [`design/`](design/README.md) | Popup UX, interaction states, keyboard ergonomics, and visual design |
| [`operations/`](operations/README.md) | Packaging, Chrome Web Store distribution, updates, and uninstallation |
| [`quality/`](quality/README.md) | Test strategy (32 automated tests, ~75ms), privacy model, and performance |
| [`plan/`](plan/README.md) | Implementation roadmap and refactor records |
