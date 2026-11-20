# Product Contract

## Problem

Users waste time when dozens or hundreds of browser tabs are distributed across windows and tab groups. When recalling a tab, their memory is usually a fragment of a title, a domain, or an acronym, while browser navigation favors spatial and mouse movement.

InfoBoard turns that fragment into an instant, predictable tab switch with zero lag and zero setup complexity.

---

## Primary User and Outcome

The primary user is a keyboard-first desktop user who regularly keeps many tabs open across multiple windows and needs a lightning-fast way to switch between tasks.

### Core User Journey:
1. **Quick Toggle**: Press `Alt+W` followed immediately by `Enter` to switch to the previously active tab (Alt+Tab pattern).
2. **Instant Search**: Press `Alt+W`, type a few letters (`t` for TikTok/Threads, `ti` for Tiki/TikTok, `gh` for GitHub), see matches update in < 1ms, and press `Enter` to activate.
3. **Keyboard Ergonomics**: Use `↑` / `↓` or `Ctrl+N` / `Ctrl+P` to cycle selections, `Esc` to cancel without side effects.

---

## Capability Boundary

- **Action Popup Surface**: Triggered by standard browser command (`_execute_action`), accessible anywhere in the browser.
- **In-Memory Tab Search**: Real-time evaluation of open tabs across all browser windows.
- **Multi-Tier Matching**:
  - Exact prefix on domain & title
  - Word-boundary matching on title words
  - Acronym matching for initials (`gh` -> GitHub, `yt` -> YouTube)
  - Subsequence fuzzy matching
  - MRU Recency Boost: recently used tabs rank higher on equal keyword matches.
- **MRU Tracking**: Real-time tab activation tracking backed by `chrome.storage.session` and `tab.lastAccessed`.
- **Tab Activation**: Confirmed tab and window focus with safe error handling if a tab was closed concurrently.

---

## Explicit Exclusions

- NO Go Native Messaging host or external OS binaries.
- NO SQLite or local disk databases.
- NO browsing history indexing or page content scraping.
- NO cookies, session credentials, or network interception.
- NO telemetry, remote servers, or cloud dependencies.

---

## Platform & Distribution

- **Browsers**: All Chromium-based desktop browsers (Google Chrome, Microsoft Edge, Brave, Opera, Vivaldi, Arc).
- **Operating Systems**: macOS, Windows, Linux, ChromeOS.
- **Distribution**: Chrome Web Store, Edge Add-ons, or unpacked developer mode. Zero OS installers required.
