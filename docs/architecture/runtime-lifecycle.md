# Runtime Lifecycle

Manifest V3 extensions operate under an ephemeral execution model where background service workers terminate after short periods of inactivity (~30 seconds). InfoBoard is designed specifically to thrive under this model without data loss or startup delays.

---

## Service Worker Lifecycle

```text
Browser Startup / Extension Load
  │
  ▼
[Initialize MRU State] ────► Query open tabs ────► Hydrate storage.session from lastAccessed
  │
  ▼
[Idle / Sleep State]
  │
  ├─► chrome.tabs.onActivated ──────► Wake worker ──► touchTab(id) ──► Save to session ──► Sleep
  ├─► chrome.windows.onFocusChanged ─► Wake worker ──► touchTab(id) ──► Save to session ──► Sleep
  └─► chrome.tabs.onRemoved ────────► Wake worker ──► removeTab(id) ─► Save to session ──► Sleep
```

### Why `chrome.storage.session`?
- **In-Memory RAM**: Stored directly in the browser's memory space, bypassing disk I/O.
- **Survives Worker Sleep**: Data remains intact when the background service worker terminates and wakes up again.
- **Session Bounded**: Automatically cleared when the browser closes, preventing stale state accumulation.

---

## Popup Surface Lifecycle

```text
User presses Alt+W (_execute_action)
  │
  ▼
[Popup HTML & Script Loaded] (< 15ms)
  │
  ├─► Promise.all([
  │     chrome.storage.session.get('mru_tab_ids'),
  │     chrome.tabs.query({})
  │   ])
  │
  ▼
[Normalize & Map MRU Rank] (< 1ms)
  │
  ▼
[Initial Render]
  │  • Query is empty: tabs ordered purely by MRU
  │  • Pre-select index 1 (previous tab)
  │  • Input autofocus
  │
  ▼
[User Keystrokes]
  │
  ├─► G keypress: scoreTab() runs in < 0.2ms ──► update DOM list (< 1ms)
  ├─► ↑ / ↓ arrows: updateSelection() ───────► update aria-selected (< 0.1ms)
  ├─► Escape: window.close() ────────────────► popup dismissed
  └─► Enter: activateTab() ──────────────────► chrome.tabs.update() + window.close()
```
