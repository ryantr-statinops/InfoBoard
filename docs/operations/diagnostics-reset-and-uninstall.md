# Diagnostics, Reset, and Uninstall

## Diagnostics

InfoBoard operates completely in-memory with near-zero latency (< 1ms). Diagnostic verification can be checked locally:

1. **Automated Unit & Integration Tests**:
   Run `npm test` to verify all 32 tests (Scorer, MRU Manager, Tab Normalizer, Surface, Manifest).
2. **Browser DevTools**:
   - Inspect background service worker at `chrome://extensions` -> `Inspect views: service worker`.
   - Inspect popup UI by right-clicking the extension icon -> `Inspect popup`.

---

## Reset State

Because InfoBoard stores MRU tab history in `chrome.storage.session`, resetting the extension state is effortless:

1. **Clear Session Memory**:
   Closing and reopening the browser automatically purges the in-memory MRU state.
2. **Manual Reload**:
   Clicking the **Reload** button on `chrome://extensions` immediately restarts the service worker and rehydrates MRU state from active tabs.
3. **Tab Safety**:
   Resetting or reloading the extension never modifies, closes, or rearranges open tabs.

---

## Uninstall

Uninstallation is handled entirely through standard browser controls:

- Right-click the extension icon on the toolbar -> **Remove from Chrome...**
- Or remove via `chrome://extensions`.

Because InfoBoard does not write to the host filesystem, registry, or local databases, removing the extension leaves **zero residual files or background processes** on the user's operating system.
