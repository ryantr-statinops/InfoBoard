# Data and Failure Boundaries

## Data Boundary & Privacy

InfoBoard adheres to strict data minimalism:

- **Collected Data**:
  - `tab.id`, `tab.windowId` (ephemeral identifiers)
  - `tab.title`, `tab.url` (used strictly for in-memory string matching)
  - `tab.favIconUrl` (visual icon display)
  - `tab.lastAccessed` (fallback recency timestamp)
  - `tab.active`, `tab.pinned`, `tab.groupId` (context badges)
- **Forbidden Data**:
  - NO browsing history indexing
  - NO DOM or page content scraping
  - NO cookies or session credentials
  - NO network request interception
  - NO telemetry or remote tracking
  - NO disk database persistence

All metadata stays strictly within the local browser memory and is discarded when tabs or the browser are closed.

---

## Failure Isolation

| Failure Scenario | Consequence | Recovery Mechanism |
| :--- | :--- | :--- |
| **Service Worker Suspended** | Normal operation in Manifest V3 | Browser wakes worker on next `onActivated` event; session storage state is fully preserved. |
| **Session Storage Unavailable** | MRU ring buffer cannot be read | System transparently falls back to sorting open tabs by `tab.lastAccessed`. Search remains fully functional. |
| **Selected Tab Closed Concurrently** | User presses Enter on a tab closed just before activation | `chrome.tabs.update()` throws an error caught by `try/catch`. Popup closes cleanly without crashing the browser. |
| **Special Regex Characters in Query** | User types `[`, `*`, `+`, `?`, `\`, `(` | Query string is sanitized with `replace(/[.*+?^${}()|[\]\\]/g, '\\$&')` before building boundary regex. Never throws `SyntaxError`. |
| **Extension Updated While Browser Runs** | Existing popup instance might hold stale context | Browser unloads previous popup; next shortcut invocation loads fresh assets from `dist/`. |
