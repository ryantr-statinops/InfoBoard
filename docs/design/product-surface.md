# Product Surface and User Ergonomics

The InfoBoard search surface is designed as a distraction-free, zero-latency popup focused solely on finding and switching tabs.

---

## 1. Invocation & Focus

- **Default Shortcut**:
  - Linux / Windows: `Alt+W`
  - macOS: `Command+Shift+K`
- **Focus**: The popup window automatically focuses the `<input>` element immediately upon launch.
- **Debounce**: 0 ms. Queries are evaluated synchronously on the main thread in Chromium's V8 engine without network or IPC latency.

---

## 2. Quick-Toggle Tab Switching (`Alt+Tab` Style)

A primary use case is toggling back and forth between two active tabs (e.g., code editor tab and documentation tab).

1. User presses `Alt+W`.
2. The popup opens with an empty query.
3. InfoBoard automatically places selection on **Index 1** (the tab activated just prior to the current tab).
4. User presses `Enter`.
5. The browser instantly switches to the previous tab and closes the popup.

This enables instantaneous two-keystroke tab switching without taking hands off the keyboard.

---

## 3. Keyboard Navigation

The popup provides exhaustive keyboard controls:

| Key Binding | Action |
| :--- | :--- |
| **`ArrowDown`** / **`Ctrl+N`** | Move selection down to the next result (clamped at bottom). |
| **`ArrowUp`** / **`Ctrl+P`** | Move selection up to the previous result (clamped at index 0). |
| **`Enter`** | Activate the currently selected tab and window; dismiss popup. |
| **`Escape`** | Close the popup immediately without changing active tabs. |
| **Click / Mouse Hover** | Selects or activates the hovered tab item. |

---

## 4. Visual Layout & Accessibility

- **Input Area**: Clean search bar with an integrated search icon and real-time result count indicator.
- **Result Row Structure**:
  - **Favicon**: Cached tab favicon or fallback globe icon.
  - **Title**: Primary label with highlighted matching text (`<mark>`).
  - **Tab Group Badge**: If a tab is part of a Chromium Tab Group, its group title and color accent are displayed.
  - **Clean Domain / URL**: Secondary label with highlighted matching text.
  - **Pinned Icon**: Displays a pin icon for pinned tabs.
- **Accessibility**:
  - Implements WAI-ARIA 1.2 Combobox pattern (`role="combobox"`, `role="listbox"`, `role="option"`).
  - Selected tab maintains `aria-selected="true"` and updates `aria-activedescendant`.
  - Accessible contrast ratios for dark and light modes according to WCAG 2.1 AA standards.
