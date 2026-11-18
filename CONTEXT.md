# Project context

This file is the shared language and operating context for agents working in this project. Keep it concise, current, and specific to decisions that change how work should be planned, implemented, reviewed, or reported.

## Project

- Name: InfoBoard.
- Purpose: A lightweight, high-performance Chromium desktop extension for keyboard-first search and activation of open browser tabs.
- Primary users or consumers: Desktop users who keep many tabs across windows and need fast, predictable tab switching.
- Architecture: Pure Manifest V3 Web Extension. Runs 100% in-browser on Chromium's V8 engine with zero native host binaries, zero daemons, and zero database dependencies.

## Domain vocabulary

| Term | Meaning in this project | Avoid or distinguish from |
| --- | --- | --- |
| Tab search | Search and activate currently open browser tabs. | Whole-browser-history search. |
| Hot path | Shortcut-to-focused-input, query-to-results, and selection-to-activation behavior. | Native IPC, disk indexing, or diagnostics daemon. |
| MRU order | Most Recently Used sequence of tabs tracked via `chrome.storage.session` and `tab.lastAccessed`. | Browsing history database. |
| Scorer | In-memory prefix, word-boundary, and acronym ranking engine with MRU boost. | Heavy tokenizers or external search daemons. |

## Architecture boundaries

- System boundary: Chrome and Edge desktop on Linux, macOS, and Windows.
- Major components: Manifest V3 extension, focused popup surface (`_execute_action`), background MRU session manager, in-memory multi-tier scorer, keyboard navigation controller.
- Ownership boundaries: All operations run locally inside the browser extension process. No external native process, SQLite, cloud service, or loopback listener.
- Important data flows: Shortcut -> popup open -> live query over in-memory tab metadata -> deterministic ranked results -> confirmed tab/window activation. Tab activation events keep the MRU session ring buffer updated.

## Testing and verification

- Unit tests: `tests/scorer.test.mjs` running on Node.js native test runner via `npm test`.
- Build verification: `npm run build:extension`.
- Manual verification: Load unpacked extension in Chrome/Edge, test `Alt+W` shortcut, typing prefixes, acronyms, and Enter tab switching.
