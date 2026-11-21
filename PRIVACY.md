# InfoBoard Privacy Policy

*Last updated: November 22, 2026*

InfoBoard ("the extension") is a lightweight, high-performance tab search and switcher built strictly on the Chrome Manifest V3 standard. Your privacy is paramount: **InfoBoard operates 100% locally on your machine and never collects, transmits, or shares any personal data.**

---

## 1. Zero Data Collection & Zero Telemetry

- **No Remote Servers**: InfoBoard does not own, run, or communicate with any external backend servers, APIs, or databases.
- **No Analytics / Tracking**: The extension contains zero tracking scripts, telemetry tools, or advertising trackers (no Google Analytics, no Mixpanel, etc.).
- **No Cookies or Identifiers**: InfoBoard creates no tracking cookies, persistent identifiers, or device fingerprints.

---

## 2. Chrome Permissions & How They Are Used

InfoBoard requests only the absolute minimum permissions strictly necessary for its single-purpose function:

| Permission | Technical Purpose | Data Storage & Transmission |
| :--- | :--- | :--- |
| `tabs` | Reads the titles, URLs, favicons, and audible status of open tabs to populate search results in memory. | Handled entirely in-memory (RAM) inside your browser. Never sent over any network. |
| `tabGroups` | Reads tab group titles and colors to help you organize search results by group context. | Processed entirely in-memory. Never stored or exported. |
| `storage` | Uses `chrome.storage.session` to track the recency order of active tab IDs (MRU ring buffer). | Ephemeral in-memory session storage. Automatically purged when browser restarts. |

---

## 3. Data Retention & Sharing

- InfoBoard does not sell, rent, monetize, or transmit user information to any third party under any circumstance.
- Because all tab queries and MRU states live in your local browser session, closing your browser clears the session state completely.

---

## 4. Open Source & Transparency

InfoBoard is completely open-source. You can inspect every line of source code, verify build dependencies, and audit network activity in our public repository:
[https://github.com/ryantr-statinops/InfoBoard](https://github.com/ryantr-statinops/InfoBoard)

---

## 5. Contact & Questions

If you have any questions or feedback regarding this Privacy Policy, please open an issue on our GitHub repository:
[https://github.com/ryantr-statinops/InfoBoard/issues](https://github.com/ryantr-statinops/InfoBoard/issues)
