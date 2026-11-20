# Privacy and Security

InfoBoard is engineered with a strict privacy-first model. Because it operates 100% inside the browser's extension sandbox with zero native binaries, user tab data never leaves the browser process.

---

## 1. Minimal Permissions & Least Privilege

InfoBoard requests only the absolute minimum permissions required for tab search and activation:

```json
{
  "permissions": [
    "tabs",
    "storage",
    "tabGroups"
  ]
}
```

### Excluded Permissions
- **NO `history`**: InfoBoard cannot see closed tabs, past browsing history, or search queries from browser omnibox.
- **NO `cookies`**: No access to authentication tokens or session cookies.
- **NO `<all_urls>` / host permissions**: InfoBoard cannot read page contents, DOM nodes, passwords, or personal data.
- **NO `webRequest` / network proxy**: InfoBoard cannot intercept or monitor network traffic.
- **NO `nativeMessaging`**: No external OS processes or daemons are registered or spawned.

---

## 2. Zero Network Egress

- **100% Offline Execution**: All tab scoring, sorting, and filtering happen locally in-memory using JavaScript in the browser process.
- **No Analytics / Telemetry**: No tracking pings, Google Analytics, Sentry, or remote logging.
- **No Remote Code Execution**: All JavaScript is packaged locally in the extension bundle, complying strictly with Manifest V3 policies.

---

## 3. Data Storage & Privacy Boundaries

- **Session-Only Persistence**: Tab recency metadata (MRU ring buffer) is stored exclusively in `chrome.storage.session`. This data is stored in memory and completely purged when the browser process terminates.
- **No Disk Database**: Zero SQLite, IndexedDB, or local filesystem footprint.
- **Incognito Profile Isolation**: By default in Chromium, extensions are disabled in incognito windows. If the user explicitly enables InfoBoard in incognito mode, incognito tabs are searched only within the incognito window context and never mixed with normal browsing profiles.
