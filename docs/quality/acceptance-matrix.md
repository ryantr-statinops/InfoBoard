# Acceptance Matrix

The InfoBoard Pure Manifest V3 extension is accepted for production release based on verification against the following gates:

| Category | Gate / Criterion | Verification Method | Status |
| :--- | :--- | :--- | :--- |
| **Invocation & Ergonomics** | Shortcut (`Alt+W` / `Command+Shift+K`) opens popup; search input is instantly autofocused with 0ms delay. | Manual testing in Chromium & Edge. | Pass |
| **Quick-Toggle Switch** | Opening popup with empty query pre-selects index 1 (the previously active tab). Hitting `Enter` immediately switches tabs. | Unit test (`tests/surface.test.mjs`) & manual verification. | Pass |
| **Search Scoring** | Accurate ranking for prefixes, word boundaries, and acronyms (e.g., `gh` -> GitHub, `yt` -> YouTube, `ti` -> Tiki/TikTok). | Automated unit tests (`tests/scorer.test.mjs`, 10 tests). | Pass |
| **Recency MRU Boost** | Tabs recently activated receive ranking bonus `+ 50 / (mruRank + 1)`. | Automated unit tests (`tests/scorer.test.mjs`). | Pass |
| **Keyboard Navigation** | `ArrowDown`/`Ctrl+N` and `ArrowUp`/`Ctrl+P` navigate with boundary clamping; `Escape` dismisses popup. | Unit test (`tests/surface.test.mjs`, 4 tests). | Pass |
| **Activation & Safety** | Activating a tab switches window focus; closing a tab removes it from MRU ring buffer without leaving dangling entries. | Unit tests (`tests/mru-manager.test.mjs`, 9 tests). | Pass |
| **Bundle Footprint** | Uncompressed bundle size must be under 50 KB. | Automated test (`tests/manifest-and-bundle.test.mjs`) verifies ~18.5 KB size. | Pass |
| **Clean Build** | TypeScript compiles cleanly with zero type errors (`tsc --noEmit`). | Verified during `npm test` and `npm run build:extension`. | Pass |
| **Privacy & Security** | Zero host permissions, zero network calls, zero external background processes or native binaries. | Verified in `manifest.json` review and network log audits. | Pass |
