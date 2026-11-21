# Architecture: Security & Privacy Model

InfoBoard is designed under strict **Data Minimization** and **Zero-Trust Client Isolation** principles. It requires no user account, transmits no network traffic, and respects browser security boundaries.

---

## 1. Zero-Network Guarantee

- **No Remote Telemetry**: Zero analytics, zero logging SDKs (no Google Analytics, Sentry, or Mixpanel).
- **No External CDN Dependencies**: All assets (icons, styles, scripts) are packaged locally in the extension bundle.
- **Content Security Policy (CSP)**: Extension operates under default MV3 CSP, forbidding `eval()` and remote script execution.

---

## 2. Permission Audit & Rationale

InfoBoard requests only the absolute minimum permissions required for its single purpose:

| Permission | Scope | Technical Justification |
| :--- | :--- | :--- |
| `tabs` | Open tab metadata | Required to query open tabs, read tab titles, URLs, favicons, and audible status. Handled strictly in-memory. |
| `tabGroups` | Tab group metadata | Required to display group names and color badges on grouped tabs. |
| `storage` | Ephemeral session | Uses `chrome.storage.session` to retain the order of active tab IDs across service worker idle periods. |

### Permissions Explicitly Omitted
- ❌ `nativeMessaging`: No companion binary or OS communication.
- ❌ `<all_urls>` / Content Scripts: No DOM injection or page content scraping.
- ❌ `history`: Only currently open tabs are searched; browsing history is never indexed.
- ❌ `webRequest`: No network request inspection.

---

## 3. Data Storage & Lifecycle

1. **In-Memory RAM Only**: Tab metadata is queried on-demand when the popup opens and discarded when it closes.
2. **Session Storage Scoping**: `chrome.storage.session` holds only an array of integer tab IDs (`[102, 101, 105]`), capped at 100 entries. No titles or URLs are written to persistent storage.
3. **Automatic Cleanup**: Session storage is destroyed when the browser process exits.
