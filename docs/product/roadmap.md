# Product Roadmap

This document outlines the delivered milestones and future exploration areas for InfoBoard.

---

## 1. Delivered Milestones (v1.0.0 - Pure Manifest V3)

- [x] **Milestone 1: Pure MV3 Architecture & Codebase Simplification**
  - Retired external Go native messaging host, SQLite database, and legacy reconcilers.
  - Reduced extension footprint from multi-process binary install to < 20KB pure browser bundle.
- [x] **Milestone 2: MRU Session Tracking**
  - Implemented `mru-manager.ts` tracking `tabs.onActivated`, `tabs.onRemoved`, and `windows.onFocusChanged`.
  - Persisted bounded ring buffer (max 100 entries) in `chrome.storage.session`.
- [x] **Milestone 3: In-Memory Multi-Tier Scorer**
  - Deterministic scoring: Exact domain prefix (1000) > Title prefix (950) > Word boundary (800-850) > Acronym (650) > Subsequence (400) + MRU boost.
  - Sub-millisecond execution (< 0.2ms for 1,000 tabs) on Chromium V8.
- [x] **Milestone 4: Quick-Toggle Ergonomics & Search Surface**
  - Automatic pre-selection of previous tab (Index 1) on empty search for instant `Alt+W` -> `Enter` toggle.
  - Keyboard navigation: `↑/↓/Ctrl+N/Ctrl+P`, `Enter` activation, `Esc` dismissal.
  - Accessible ARIA combobox and listbox implementation.
- [x] **Milestone 5: Comprehensive Automated Testing Suite**
  - 32 automated unit tests across 5 test suites running in ~75ms with zero external test framework dependencies.

---

## 2. Future Explorations (Post v1.0.0)

- [ ] **Tab Group Filtering**: Support filtering tabs by group name using prefixes (e.g., `#dev`, `#reading`).
- [ ] **In-Surface Tab Actions**: Support closing tabs (`Ctrl+W` or `Delete`) or muting audio (`M`) directly from the search list.
- [ ] **Multi-Window Tagging**: Display subtle badges indicating which window a tab belongs to when multiple windows are open.
- [ ] **Customizable Density & Themes**: Provide options in a settings panel for compact vs. spacious row layouts and custom color accents.
