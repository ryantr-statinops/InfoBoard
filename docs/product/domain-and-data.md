# Domain and Data

This document defines the tab domain model, data ownership, retention policies, and privacy boundaries in InfoBoard.

---

## 1. Tab Search Domain Model

The primary searchable entity in InfoBoard is the `SearchTabItem` (`extension/src/search/types.ts`), representing an open browser tab enriched with metadata:

```typescript
export interface SearchTabItem {
  id: number;
  windowId: number;
  title: string;
  url: string;
  domain: string; // e.g. "github.com" or "tiktok.com"
  favIconUrl?: string;
  mruRank: number; // 0 = currently active, 1 = previous, ...
  active: boolean;
  pinned: boolean;
  groupId?: number;
}
```

### Domain Field Derivation
- **domain**: Derived by parsing `url` via `new URL(url).hostname` and stripping leading `www.`. Non-standard URLs (e.g., `chrome://`, `edge://`) use their protocol or host prefix.
- **mruRank**: Position in the MRU ring buffer. Lower numbers indicate more recent activation. Tabs not in the buffer are assigned `mruRank = buffer.length`.

---

## 2. Data Ownership and Lifecycle

InfoBoard owns **zero persistent disk storage**. All state is ephemeral and bounded by the browser lifecycle.

| Data | Owner | Storage Mechanism | Retention Policy |
| :--- | :--- | :--- | :--- |
| **Open Tabs** | Chromium Browser | In-memory (`chrome.tabs.query`) | Ephemeral (matches live browser tabs) |
| **MRU Order** | InfoBoard Service Worker | `chrome.storage.session` | Session-scoped ring buffer (max 100 entries); cleared on browser exit |
| **Search Index** | InfoBoard Popup Surface | In-memory JS array | Discarded when popup closes |
| **User Settings** | InfoBoard Extension | `chrome.storage.local` (optional) | Until extension uninstalled |

---

## 3. Privacy and Trust Invariants

InfoBoard adheres to strict privacy-by-design principles:

1. **Zero Network Egress**: The extension defines zero host permissions and issues zero network calls (`fetch`, `XMLHttpRequest`, or WebSockets).
2. **Zero Page-Content Injection**: InfoBoard never injects scripts into web pages and cannot read DOM contents, cookies, form fields, or local storage.
3. **No History Access**: InfoBoard only interacts with open tabs; it does not request the `history` permission or access past browsing logs.
4. **Isolated Incognito Handling**: By default in Chromium, extensions do not run in incognito mode unless explicitly granted by the user in `chrome://extensions`. Even when enabled, incognito tabs are isolated to their private window context.
