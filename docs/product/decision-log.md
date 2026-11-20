# Decision Log

This document records the architectural and product decisions made for InfoBoard.

---

## ADR-001: Pure Manifest V3 Architecture (Retiring Go Native Host & SQLite)

### Context
InfoBoard was originally conceived with a dual-tier architecture:
- An extension frontend in TypeScript.
- An external native Go host communicating over stdio via Chromium Native Messaging.
- An SQLite database on disk for tab persistence, migrations, and query history.

During testing and performance benchmarking, this design demonstrated substantial unnecessary complexity:
1. Native Messaging IPC introduced 10–50ms round-trip latency on cold start and query execution.
2. Cross-platform OS packaging required native installer scripts, registry key setup (Windows), and manifest directory symlinks (macOS/Linux).
3. Open tabs are volatile browser state; storing them in an SQLite database duplicated Chromium's internal tab store and required complex synchronization and reconciliation logic (~40KB reconciler, 20-phase DAG).
4. Chromium's V8 engine executes lexical scoring across 1,000 tabs in < 0.2ms natively in JavaScript.

### Decision
**Completely remove the external Go host, SQLite database, and Native Messaging layer.** Re-implement InfoBoard as a 100% Pure Manifest V3 browser extension running directly in Chromium.

### Consequences
- **Positive:**
  - Zero-latency search: tab filtering occurs in < 1ms directly in V8.
  - Zero external installation steps: users install directly from Chrome Web Store with 1 click.
  - Minimal bundle size: ~18KB total extension footprint (< 50KB requirement).
  - Codebase reduction: eliminated ~25,000 lines of Go code, fixtures, and redundant reconcilers.
  - Maintenance: single TypeScript toolchain with zero native runtime dependencies.
- **Negative / Trade-offs:**
  - Cannot read system-level OS windows outside of the browser (out of product scope).

---

## ADR-002: In-Memory Multi-Tier Scoring & Ranking Algorithm

### Context
Users search for tabs using short tokens, domain fragments, and acronyms (e.g., `gh` for GitHub, `yt` for YouTube, `t` for TikTok/Threads, `ti` for Tiki/TikTok). A naive full-text or substring search produces unpredictable rankings and misses common abbreviation patterns.

### Decision
Implement a deterministic, multi-tier ranking engine in `extension/src/search/scorer.ts`:
1. **Tier 1 (Exact Prefix):** Clean domain prefix (`1,000 pts`) > Title prefix (`950 pts`).
2. **Tier 2 (Word Boundary Prefix):** Clean domain word prefix (`850 pts`) > Title word prefix (`800 pts`).
3. **Tier 3 (Acronym Match):** Initial letters of words in title (`650 pts`).
4. **Tier 4 (Subsequence Match):** Fuzzy character sequence (`400 pts`).
5. **MRU Boost:** Recency weight calculated as `+ 50 / (mruRank + 1)`.

### Consequences
- Short queries (`t`, `gh`, `yt`) reliably surface the expected tabs at the top.
- Recent tabs get a noticeable lift without overriding exact text matches.
- Completely deterministic with zero ML/vector dependencies.

---

## ADR-003: Quick-Toggle Ergonomics (Pre-selecting Index 1 on Empty Query)

### Context
Switching back and forth between two tabs is the most frequent browser multitasking action (analogous to `Alt+Tab` in operating systems).

### Decision
When the search surface is opened and the query is empty:
- Index 0 represents the currently active tab (which the user is already on).
- Pre-select Index 1 (the previously active tab in the MRU ring buffer).
- When the user presses `Alt+W` then `Enter`, InfoBoard instantly switches to the previous tab.

### Consequences
- Delivers instantaneous two-key tab switching without requiring mouse clicks or arrow key presses.
- If the user begins typing, the selection automatically updates to Index 0 of the matching results.

---

## ADR-004: MRU Ring Buffer in `chrome.storage.session`

### Context
Service workers in Manifest V3 are ephemeral and can terminate after 30 seconds of inactivity. In-memory variables inside the service worker can be lost between tab activations.

### Decision
Store the MRU tab ID ring buffer (bounded to `MAX_MRU_SIZE = 100`) in `chrome.storage.session`. When the service worker awakens, it loads the ring buffer from session storage; on `tabs.onActivated`, it updates the buffer and writes back.

### Consequences
- State survives service worker suspensions without writing to physical disk.
- Session storage is automatically cleared when the browser profile closes, ensuring no privacy leaks or stale IDs survive browser restarts.
