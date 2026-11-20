# Packaging and Installation

## Release Artifacts

Because InfoBoard is a **Pure Manifest V3 Web Extension**, the release artifact is a single, self-contained extension zip archive containing:

- `manifest.json` (declarations, permissions, commands)
- `dist/search/surface-shell.html` & `surface-shell.css` (popup UI)
- `dist/src/search/` (in-memory scorer and popup controller)
- `dist/src/background/` (MRU manager and service worker)

Total uncompressed bundle size is **under 20 KB**. No operating system installers (`.exe`, `.msi`, `.pkg`, `.deb`), code-signing certificates, or registry configurations are required.

---

## Distribution Channels

### 1. Chrome Web Store & Edge Add-ons
- Users install directly with a single click from the official store.
- Automatic background updates are handled securely by Chromium.
- Zero administrative permissions required.

### 2. Developer / Unpacked Mode
- Clone repository or extract release zip.
- Open `chrome://extensions` or `edge://extensions`.
- Enable **Developer mode** in the upper right corner.
- Click **Load unpacked** and select the `extension/` folder.

---

## Permissions & Security

InfoBoard requests only 3 narrow browser permissions:
- `tabs`: Read open tab titles, URLs, and IDs to display search results and activate tabs.
- `storage`: Utilize `chrome.storage.session` for ephemeral in-memory MRU tracking.
- `tabGroups`: Group visibility and organization.

InfoBoard requests **NO host permissions** (`<all_urls>`), NO history permissions, and NO native messaging permissions.
