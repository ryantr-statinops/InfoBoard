# InfoBoard

InfoBoard is an ultra-fast, keyboard-first Chrome/Edge desktop extension for searching and switching open tabs across windows.

Built with a **Pure Manifest V3 architecture**, it runs 100% in-browser with zero external native hosts, zero daemons, and zero database overhead.

---

## Key Features

- **Blazing Fast (< 1ms)**: In-memory lexical search and ranking directly on Chromium's V8 engine.
- **MRU Tab Tracking**: Automatically orders open tabs by Most Recently Used recency using in-memory `chrome.storage.session` and `tab.lastAccessed`.
- **Quick Switch (Alt+Tab Pattern)**: Opening search automatically pre-selects the previous tab—press shortcut then `Enter` to toggle immediately between your last two tabs.
- **Direct Tab Close**: Close tabs instantly using `Alt + D`, `Delete`, or the hover `x` button without switching.
- **Tab Audio Indicator**: Live soundwave and muted indicator on tabs actively playing audio.
- **Bilingual i18n**: Automatic localization for English and Vietnamese based on browser language.
- **Cinematic 500ms Motion Engine**: Smooth waterfall entrance cascade and reverse exit dissolution.
- **Multi-tier Fuzzy & Prefix Matcher**:
  - Exact prefix on domain & title (`ti` -> `tiki.vn`, `tiktok.com`)
  - Word-boundary matching (`t` -> `Instagram - Threads`)
  - Acronym matching (`gh` -> `GitHub`, `yt` -> `YouTube`)
  - Recency Boost: tabs used more recently rank higher on equal keyword matches.
- **Privacy-first & Local**: Operates strictly on open-tab metadata (title, URL, domain). No history indexing, no page content scraping, no cloud connections.

---

## Project Structure

```text
InfoBoard/
├── .github/workflows/ci.yml    # GitHub Actions automated test & package pipeline
├── docs/                       # Comprehensive open-source documentation hub
├── extension/
│   ├── manifest.json           # Manifest V3 (tabs, storage, tabGroups)
│   ├── icons/                  # Monochrome icons (16, 32, 48, 128px)
│   ├── _locales/               # Bilingual i18n dictionaries (en, vi)
│   ├── src/
│   │   ├── background/
│   │   │   ├── mru-manager.ts  # Session-based MRU ring buffer
│   │   │   └── service-worker.ts # Tab/window activation listeners
│   │   ├── search/
│   │   │   ├── types.ts        # Tab item definitions & normalizers
│   │   │   ├── scorer.ts       # Prefix/boundary/acronym/boost scoring engine (< 1KB)
│   │   │   ├── surface-shell.html # Popup UI layout with i18n data bindings
│   │   │   ├── surface-shell.css  # Dark/light styling & 500ms waterfall animations
│   │   │   └── surface-entrypoint.ts # DOM rendering, keyboard navigation & tab close
│   │   └── types/
│   │       └── chrome.d.ts     # Minimal Chrome WebExtension type declarations
│   └── tsconfig.json
├── tests/
│   ├── manifest-and-bundle.test.mjs # Validates MV3 manifest and < 50KB bundle
│   ├── mru-manager.test.mjs         # Tests MRU ring buffer and session storage
│   ├── scorer.test.mjs              # Multi-tier prefix/boundary/acronym ranking
│   ├── surface.test.mjs             # Popup DOM rendering, keyboard actions & closeTab
│   └── types.test.mjs               # Tab normalization, domain & audio extraction
├── scripts/
│   ├── copy-extension-assets.mjs    # Static asset distribution
│   └── package-extension.mjs        # Production zip packager (< 100KB)
├── LICENSE                     # MIT License
├── PRIVACY.md                  # Chrome Web Store compliant privacy policy
└── package.json
```

---

## Documentation

Comprehensive guides are available in the [**Documentation Hub**](docs/README.md):

- [**Installation Guide**](docs/getting-started/installation.md) & [**Development Setup**](docs/getting-started/development.md)
- [**Architecture Overview**](docs/architecture/system-overview.md), [**Search Algorithm**](docs/architecture/search-and-ranking.md) & [**Security Model**](docs/architecture/security-and-privacy.md)
- [**Keyboard Shortcuts**](docs/user-guide/keyboard-shortcuts.md), [**UI Features**](docs/user-guide/features-and-ui.md) & [**Internationalization**](docs/user-guide/internationalization.md)
- [**Packaging & Distribution**](docs/operations/packaging-and-distribution.md) & [**CI/CD Pipeline**](docs/operations/ci-cd-pipeline.md)
- [**Contributing Guidelines**](docs/contributing/CONTRIBUTING.md) & [**Code of Conduct**](docs/contributing/CODE_OF_CONDUCT.md)

---

## Development & Build

### Prerequisites
- Node.js >= 18

### Build Extension
```bash
npm run build:extension
```
Build output is generated into `extension/dist/`.

### Run Tests
```bash
npm test
```

---

## Load in Browser

1. Open Chrome or Edge and navigate to `chrome://extensions` (or `edge://extensions`).
2. Enable **Developer mode** toggle in the top-right corner.
3. Click **Load unpacked** and select the `extension/` directory.
4. Press `Alt+W` (or `Command+Shift+K` on macOS) to open Tab Search.

---

## License

This project is licensed under the terms of the [MIT License](LICENSE). Copyright (c) 2026 Ryan Tran (InfoBoard Contributors).
