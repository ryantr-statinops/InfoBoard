# Browser and platform boundary

## Supported product matrix

| Browser | OS | Support target | Notes |
| --- | --- | --- | --- |
| Chrome desktop | Linux, macOS, Windows | Supported | Manifest V3 extension and registered Native Messaging host. |
| Edge desktop | Linux, macOS, Windows | Supported | Chrome-compatible extension surface with Edge packaging and host registration checks. |
| Other Chromium browsers | Any | Not supported by the product contract | May be tested separately; no compatibility promise. |
| Firefox, Safari, mobile browsers | Any | Excluded | Separate extension, UX, packaging, and permission contracts would be required. |

## Browser data policy

| Data surface | Product use | Policy |
| --- | --- | --- |
| Open tabs and windows | Search dataset and activation target | Required. |
| Title, URL, domain | Lexical fields and display context | Required; URL display may be redacted in privacy mode. |
| Window ID and label | Context and activation | Required where the API exposes it. |
| Tab group ID and label | Context and ranking | Use when supported; degrade to no-group state. |
| Pinned state | Context and ranking | Use. |
| Recent activation order | Recency ranking | Local bounded metadata only. |
| Browser history | Broader recall | Excluded. |
| Bookmarks and downloads | Different search product | Excluded. |
| Page DOM, rendered text, selected text | Richer semantic search | Excluded from the product contract. |
| Cookies, page local/session storage | Credentials and private state | Excluded; the extension never reads page storage. |
| Extension-owned `chrome.storage.local` | Stable opaque profile identity only | Allowed only for one locally generated `profile_id` at key `profile_id`, per REF-013. No other values, sync, page storage, or content. |

## Extension model

The extension owns the command, search surface, browser event listeners, profile boundary, tab activation, and user-visible state. The service worker may be suspended or restarted, so it MUST persist only what is necessary and MUST reconcile from browser APIs after reconnect.

The search surface is the extension-owned action popup anchored in the browser toolbar. It stays inside the browser UI, does not navigate or activate the current page, and does not require page DOM access, scripts, or host permissions. The browser action icon is the native container for the configured command's popup.

## Profile and private-window rules

- Each browser profile has an isolated InfoBoard configuration, runtime session, and tab projection.
- Normal and private/incognito contexts MUST remain separate.
- Private-window support is enabled only when the browser grants the required extension access; otherwise the UI explains that the context is unavailable.
- No tab projection is copied between profiles or uploaded to a service.

## Permissions principle

Permission requests MUST map to a named requirement. The implementation should prefer tab/window/group APIs and command registration over broad host permissions. Permission review is a release gate, not a post-release cleanup task.

## Sources

- Chrome Tabs API: https://developer.chrome.com/docs/extensions/reference/api/tabs
- Chrome Commands API: https://developer.chrome.com/docs/extensions/reference/api/commands
- Chrome Windows API: https://developer.chrome.com/docs/extensions/reference/api/windows
- Chrome Tab Groups API: https://developer.chrome.com/docs/extensions/reference/api/tabGroups
- Chrome Native Messaging: https://developer.chrome.com/docs/extensions/develop/concepts/native-messaging
- Chrome extension service-worker lifecycle: https://developer.chrome.com/docs/extensions/develop/concepts/service-workers/lifecycle
- Edge native messaging: https://learn.microsoft.com/en-us/microsoft-edge/extensions/developer-guide/native-messaging
- Porting Chrome extensions to Edge: https://learn.microsoft.com/en-us/microsoft-edge/extensions/developer-guide/port-chrome-extension
