# Local Development Guide

This guide covers setting up your development environment, compiling TypeScript, running tests, and debugging the extension.

---

## 1. Toolchain & Prerequisites

### Required Environment
- **Node.js**: `v20.x` or later (LTS recommended) OR alternative JavaScript runtimes like **Bun** (`v1.0+`).
- **TypeScript**: `v5.9+` (configured under `extension/tsconfig.json`).
- **No External Companion Daemon / No Native Messaging Host**: InfoBoard runs 100% inside Chromium's V8 engine at runtime.

### Why Node.js in a Browser Extension?
A common question when developing browser extensions is why Node.js is required if the extension runs in the browser. 

> [!NOTE]
> **Build-Time Toolchain vs. Browser Runtime:**
> Node.js functions strictly as an **offline developer toolchain**; it **never** runs inside the browser or extension runtime.

Node.js fulfills three essential development responsibilities in InfoBoard:
1. **TypeScript Compilation (`tsc`)**: InfoBoard's source code is written in strongly-typed TypeScript (`extension/src/**/*.ts`). Node.js runs `tsc` to compile this into production-grade, optimized JavaScript inside `extension/dist/`.
2. **Native Zero-Dependency Test Suite (`node --test`)**: InfoBoard utilizes Node.js's built-in test runner (`node --test tests/*.test.mjs`) to validate scoring algorithms, regex boundaries, and tab normalizers across 34 test cases in under 100ms without requiring heavy third-party testing frameworks (no Jest/Vitest).
3. **Cross-Platform Asset Pipeline**: Node.js executes pure ESM scripts ([scripts/copy-extension-assets.mjs](file:///D:/01_Workspace/04_Personal_Showcase/InfoBoard/scripts/copy-extension-assets.mjs) and [scripts/package-extension.mjs](file:///D:/01_Workspace/04_Personal_Showcase/InfoBoard/scripts/package-extension.mjs)) to structure distribution bundles consistently on Windows, macOS, and Linux without shell incompatibilities.

---

## 2. Alternative Toolchains (Bun & Deno)

If you prefer a faster or lighter environment without installing Node.js, InfoBoard is fully compatible with modern alternative runtimes:

### Using Bun (`bun`)
[Bun](https://bun.sh) has built-in TypeScript transpilation and an ultra-fast package manager:
```bash
# Install dependencies
bun install

# Build extension
bun run build:extension

# Run automated tests
bun test tests/*.test.mjs
```

### Pure Zero-Toolchain Development (No Build Step)
If you wish to develop without any runtime or compiler:
- You can write native ES modules directly in `.js` within `extension/`.
- Point `manifest.json` directly to source HTML/JS files.
*(Note: You will forfeit TypeScript compile-time type verification and automated test runners).*

---

## 3. Common Development Commands

All tasks are driven through root `npm` scripts:

| Command | Action | Output / Target |
| :--- | :--- | :--- |
| `npm run clean` | Deletes `extension/dist` build output | Clean directory |
| `npm run build:extension` | Compiles TypeScript and copies HTML/CSS/icons/locales | `extension/dist/` |
| `npm test` | Rebuilds and runs the entire automated test suite | 34 unit tests in ~75ms |
| `npm run package` | Builds and creates a production-ready CWS zip archive | `release/infoboard-v0.1.0.zip` |

---

## 4. Directory Layout

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

## 5. Debugging & Hot Reloading

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
