# InfoBoard Documentation Hub

Welcome to the official documentation for **InfoBoard**, an ultra-fast, keyboard-first, open-source tab search and switcher extension built on the Pure Chrome Manifest V3 standard.

---

## Architecture at a Glance

```text
┌────────────────────────────────────────────────────────┐
│               Chromium Browser (Chrome / Edge)         │
│                                                        │
│  ┌──────────────────────────────────────────────────┐  │
│  │ 1. Search Surface (Action Popup)                 │  │
│  │    • Triggered via Alt+W / Cmd+Shift+K           │  │
│  │    • Input autofocus, arrow cycling, Alt+D close │  │
│  │    • Bidirectional 500ms waterfall animations    │  │
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
│  │    • tabs.onActivated / onRemoved listeners      │  │
│  │    • storage.session ring buffer (max 100 tabs)  │  │
│  └──────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────┘
```

---

## Documentation Map

Explore the documentation through the 5 core modules:

### 🚀 1. Getting Started
- [**Installation Guide**](getting-started/installation.md): How to install from the Chrome Web Store or load unpacked from source in Developer Mode.
- [**Development Workflow**](getting-started/development.md): Setting up your Node.js/TypeScript environment, build targets, running tests, and debugging.

### 🏛️ 2. Architecture & Design
- [**System Overview**](architecture/system-overview.md): Pure MV3 topology, component boundaries, and ephemeral service worker lifecycle.
- [**Search & Ranking Algorithm**](architecture/search-and-ranking.md): Multi-tier lexical scoring, MRU recency boost formula, and the Alt+Tab quick-switch pattern.
- [**Security & Privacy Model**](architecture/security-and-privacy.md): Zero-network policy, Data Minimization audit, and permission justifications.

### 📖 3. User Guide
- [**Keyboard Shortcuts & Navigation**](user-guide/keyboard-shortcuts.md): Complete keyboard shortcut cheat-sheet (`Alt+W`, `Alt+D`, `Delete`, `Escape`).
- [**Features & UI Guide**](user-guide/features-and-ui.md): Tab badges (Audio, Pinned, Current), hover close button, and the 500ms motion engine.
- [**Internationalization (i18n)**](user-guide/internationalization.md): Automatic bilingual localization for English and Vietnamese.

### ⚙️ 4. Operations & DevOps
- [**Packaging & Distribution**](operations/packaging-and-distribution.md): Creating CWS release archives with `npm run package` and release checklists.
- [**CI/CD Pipeline**](operations/ci-cd-pipeline.md): GitHub Actions continuous integration, automated quality gates, and bundle size enforcement (< 50KB).

### 🤝 5. Contributing & Community
- [**Contributing Guidelines**](contributing/CONTRIBUTING.md): Git branching model (`dev` $\rightarrow$ `main`), atomic commit standards, and PR requirements.
- [**Code of Conduct**](contributing/CODE_OF_CONDUCT.md): Contributor Covenant v2.1 community pledge and enforcement standards.
