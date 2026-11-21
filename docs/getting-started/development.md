# Local Development Guide

This guide covers setting up your development environment, compiling TypeScript, running tests, and debugging the extension.

---

## 1. Toolchain & Prerequisites

- **Node.js**: `v20.x` or later (LTS recommended).
- **TypeScript**: `v5.9+` (configured under `extension/tsconfig.json`).
- **No External Daemon / No Python**: Pure Manifest V3 architecture runs entirely in Node.js (for build/test) and Chromium V8 (for runtime).

---

## 2. Common Development Commands

All tasks are driven through root `npm` scripts:

| Command | Action | Output / Target |
| :--- | :--- | :--- |
| `npm run clean` | Deletes `extension/dist` build output | Clean directory |
| `npm run build:extension` | Compiles TypeScript and copies HTML/CSS/icons/locales | `extension/dist/` |
| `npm test` | Rebuilds and runs the entire automated test suite | 34 unit tests in ~75ms |
| `npm run package` | Builds and creates a production-ready CWS zip archive | `release/infoboard-v0.1.0.zip` |

---

## 3. Directory Layout

```text
InfoBoard/
├── extension/
│   ├── manifest.json              # Extension metadata, permissions & shortcuts
│   ├── icons/                     # Monochrome icons (16, 32, 48, 128px)
│   ├── _locales/                  # Bilingual i18n dictionaries (en, vi)
│   ├── dist/                      # Compiled JS and static assets
│   └── src/
│       ├── background/            # Service worker & MRU ring buffer
│       │   ├── mru-manager.ts     # Ephemeral session storage tracker
│       │   └── service-worker.ts  # Chrome tab/window event listeners
│       ├── search/                # Focused search popup
│       │   ├── scorer.ts          # Lexical search & ranking algorithm
│       │   ├── surface-entrypoint.ts # DOM rendering & keyboard navigation
│       │   ├── surface-shell.html # Popup shell structure
│       │   ├── surface-shell.css  # Dark/light theme & waterfall keyframes
│       │   └── types.ts           # SearchTabItem models & normalizers
│       └── types/
│           └── chrome.d.ts        # Pure Chrome WebExtension type declarations
├── scripts/
│   ├── copy-extension-assets.mjs  # Asset distributor (HTML, CSS, icons, locales)
│   └── package-extension.mjs      # Production zip packager
└── tests/
    ├── manifest-and-bundle.test.mjs # Manifest compliance & < 50KB size check
    ├── mru-manager.test.mjs         # Ring buffer & MRU ordering tests
    ├── scorer.test.mjs              # Multi-tier ranking & recency tests
    ├── surface.test.mjs             # DOM render & keyboard interaction tests
    └── types.test.mjs               # Tab normalizers & domain extractors
```

---

## 4. Debugging & Hot Reloading

1. Make edits in `extension/src/`.
2. Run `npm run build:extension` (or compile via `tsc -w -p extension/tsconfig.json`).
3. In Chromium:
   - Navigate to `chrome://extensions`.
   - Click the 🔄 **Reload** icon on the InfoBoard card.
4. **Debugging the Popup**:
   - Press `Alt + W` to open InfoBoard.
   - Right-click anywhere in the search popup $\rightarrow$ click **Inspect**.
   - DevTools will dock to the popup window.
5. **Debugging the Background Service Worker**:
   - Go to `chrome://extensions`.
   - Click **Inspect views: service worker** on InfoBoard's card to view background console logs.
