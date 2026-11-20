# Updates, Rollback, and Repair

This document details the update lifecycle, rollback mechanisms, and diagnostic recovery procedures for the InfoBoard extension.

---

## 1. Extension Update Lifecycle

Because InfoBoard has zero native binaries or external daemons, all updates follow standard Chromium extension lifecycle rules:

### Automated Web Store Updates
- When a new version is published to the Chrome Web Store or Edge Add-ons Store, the browser automatically downloads and stages the update in the background.
- Upon browser restart or service worker update cycle, Chromium smoothly transitions to the new version without requiring system restarts or installer scripts.

### Developer / Unpacked Updates
- To update local development builds:
  ```bash
  npm run build:extension
  ```
- Navigate to `chrome://extensions` and click the **Reload** icon on the InfoBoard card.
- The background service worker reloads and automatically re-hydrates active tabs via `hydrateTabs()`.

---

## 2. Rollback Procedures

### Web Store Rollback
- In the event of a regression, developers can revert the git commit on `main`, rebuild, and publish a new version bump to the Web Store.

### Local Unpacked Rollback
- To roll back a local installation:
  ```bash
  git checkout <previous-tag-or-commit>
  npm run build:extension
  ```
- Reload the extension in `chrome://extensions`.

---

## 3. Repair & Self-Healing

InfoBoard incorporates built-in self-healing mechanisms:

| Symptom | Cause | Automatic Resolution |
| :--- | :--- | :--- |
| **Empty or Stale Tab List** | Service worker was asleep during mass tab creation/closing | The popup executes `hydrateTabs()` on launch, performing a fresh `chrome.tabs.query` to rebuild the tab set. |
| **Missing Recency Order** | `chrome.storage.session` cleared or corrupted | `mru-manager.ts` handles missing storage gracefully by falling back to `tab.lastAccessed` or tab index order. |
| **Shortcut Conflicts** | Another extension has bound `Alt+W` | Visit `chrome://extensions/shortcuts` to reassign the invocation shortcut. |
