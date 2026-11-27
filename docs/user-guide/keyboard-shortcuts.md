# User Guide: Keyboard Shortcuts & Ergonomics

InfoBoard is a **keyboard-first** productivity tool designed to keep your hands on the home row while managing tabs.

---

## 1. Complete Shortcut Reference

| Action | Shortcut (Win / Linux) | Shortcut (macOS) | Behavior & Lifecycle |
| :--- | :--- | :--- | :--- |
| **Open Search** | `Alt + W` | `Command + Shift + K` | Opens the search popup with autofocus and plays the 500ms entrance waterfall. |
| **Force Close (Toggle)** | `Alt + W` | `Command + Shift + K` | Instantly destroys popup at OS bubble level (0ms delay). |
| **Dismiss Popup** | `Escape` | `Escape` | Rolls up with smooth reverse waterfall animation (500ms). |
| **Next Tab** | `ArrowDown` or `Ctrl + N` | `ArrowDown` or `Ctrl + N` | Moves active selection down; auto-scrolls into view. |
| **Previous Tab** | `ArrowUp` or `Ctrl + P` | `ArrowUp` or `Ctrl + P` | Moves active selection up; auto-scrolls into view. |
| **Jump to First** | `Home` | `Home` | Jumps selection to the first result. |
| **Jump to Last** | `End` | `End` | Jumps selection to the last result. |
| **Switch Tab** | `Enter` | `Return` | Activates target tab, focuses window, and closes smoothly (500ms). |
| **Close Tab Directly** | `Alt + D` or `Delete`* | `Alt + D` or `Delete`* | Closes target tab on browser immediately without switching. |

*\*Note: `Delete` key closes the selected tab when the query input is empty to avoid interfering with forward-deletion while editing search text.*

---

## 2. Power Workflows

### A. Instant Quick-Switch (`Alt + Tab` Pattern)
1. Press `Alt + W` (or `Cmd + Shift + K`).
2. The popup opens with the **previous tab** pre-selected (index 1).
3. Press `Enter`.
4. Result: You toggle back and forth between your two active tabs in less than 200ms.

### B. Bulk Tab Cleanup
1. Press `Alt + W`.
2. Type a domain or query (e.g. `youtube` or `meeting`).
3. Press `Alt + D` repeatedly or click the hover `x` button on matching items.
4. Result: Close unnecessary tabs in seconds without loading their web pages.

### C. Audio Triage & Context Filters
1. Press `Alt + W`.
2. Type `@` to instantly reveal the **Interactive Filter Chip Bar** (`@audio`, `@pinned`, `@group`, `@win`).
3. Press `Tab` or click any chip to autocomplete the filter immediately.
4. Prefix matching works out of the box: typing `@au` automatically activates the `@audio` filter without needing to type the full keyword.
5. Combine with a search term: `@audio lofi` (or `@au lofi`) to isolate your music stream.
6. Additional filters:
   - `@pinned` or `@p`: Restrict to pinned tabs.
   - `@group` or `@g`: Restrict to Chrome tab groups.
   - `@win` or `@w`: Restrict to the current focused window.

### D. Typo Tolerance & Smart Abbreviation
- Type `gdoc` to jump straight to Google Docs.
- Type `ytm` to jump to YouTube Music.
- Made a typo? Queries like `githbu` or `youtbe` automatically fall back to match GitHub or YouTube without retyping.
