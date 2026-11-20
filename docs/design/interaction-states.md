# Interaction States

Every state in InfoBoard provides immediate, unambiguous feedback to the user while maintaining keyboard focus.

---

## State Transition Matrix

| State | Visual Presentation | Allowed Actions |
| :--- | :--- | :--- |
| **Opening (Empty Query)** | Input focused; tabs listed in MRU order; **Index 1 pre-selected**. | Type query, hit `Enter` (quick-toggle), navigate with `↑/↓`, or hit `Esc`. |
| **Active Query Matches** | Bounded list of ranked matching tabs with text highlighting; **Index 0 pre-selected**. | Continue typing, navigate with `ArrowDown` / `Ctrl+N`, hit `Enter` to switch, or hit `Esc`. |
| **No Matches Found** | Friendly "No tabs found matching query" message with zero result rows. | Backspace / edit query, or press `Escape` to dismiss. |
| **Tab Activation** | Target tab and window focused via `chrome.tabs.update`; popup closes cleanly. | Normal browsing resumes. |
| **Stale Tab Closed** | If a tab was closed right before activation, the browser ignores gracefully. | Popup dismisses; remaining tabs reflect accurate state on next open. |
| **Dismissal** | Popup dismissed via `Escape` or clicking outside. | Zero side-effects; browser active tab remains unchanged. |

---

## Accessibility & Focus Invariants

1. **Focus Trapping**: Keyboard focus is locked within the search popup while open, preventing background window interactions from intercepting keys.
2. **Deterministic Selection**: The selection never becomes null or out-of-bounds while results exist.
3. **Smooth Scroll**: When navigating down large tab lists, the active selection smoothly scrolls into view.
