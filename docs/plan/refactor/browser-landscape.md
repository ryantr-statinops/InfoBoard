# Browser landscape

## Initial platform boundary

| Stage | Platform | Reason |
| --- | --- | --- |
| 1 | Chromium desktop: Chrome and Edge | Shared extension model and strong API coverage |
| 2 | Other Chromium browsers | Validate API behavior individually after the core works |
| 3 | Firefox desktop | WebExtensions compatibility with browser-specific differences |
| 4 | Safari | Additional packaging, distribution, and platform constraints |
| Later | Mobile browsers | Separate UX and extension capability constraints |

## Useful browser data surfaces

| Data | Value | Initial policy |
| --- | --- | --- |
| Current tab URL/title | Core search fields | Use |
| Rendered page title/text | Optional richer matching | Defer until needed |
| Selected text | Useful command-palette action | Candidate feature |
| Open tabs/windows | Core dataset | Use |
| Tab groups/pinned state | Ranking and context | Use |
| Recent activation order | Recency ranking | Use |
| Browser history | Powerful but sensitive | Defer |
| Browser bookmarks/downloads | Separate product direction | Defer |
| Cookies/local storage/network data | High-risk credentials/session data | Exclude from initial scope |

## Extension model

The extension can receive browser events, maintain the current tab projection, and invoke a user-configured command. The Go runtime must not assume that the extension service worker remains loaded forever; reconnect and index refresh are required.

A user action such as a toolbar click, context-menu item, or keyboard command is preferred over persistent access to every page. `activeTab` and `scripting` are suitable for optional current-page actions without making full-site access the default.

## Product implications

The most promising first data boundary is:

```text
open tabs + title + URL + domain + window + group + pinned state + recent activation
```

This supports a useful product without reading full history, cookies, local storage, or network requests.

## Sources

- Chrome Tabs API: https://developer.chrome.com/docs/extensions/reference/api/tabs
- Chrome Scripting API: https://developer.chrome.com/docs/extensions/reference/api/scripting
- Chrome `activeTab`: https://developer.chrome.com/docs/extensions/develop/concepts/activeTab
- Chrome Sessions API: https://developer.chrome.com/docs/extensions/reference/api/sessions
- Chrome Tab Groups API: https://developer.chrome.com/docs/extensions/reference/api/tabGroups
- Firefox WebExtensions permissions: https://developer.mozilla.org/en-US/docs/Mozilla/Add-ons/WebExtensions/manifest.json/permissions
- Safari Web Extensions: https://developer.apple.com/documentation/safariservices/safari-web-extensions
- Porting Chrome extensions to Edge: https://learn.microsoft.com/en-us/microsoft-edge/extensions/developer-guide/port-chrome-extension
