# Tab search idea brief

## Problem

Users lose time searching through many open tabs. Browser tab switching is optimized for spatial or recent navigation, not for quickly finding a tab by remembered words.

## Proposed interaction

1. User presses a configurable keyboard shortcut.
2. A focused search UI opens.
3. The user types an exact or approximate query.
4. Matching tabs appear immediately.
5. Results prioritize strong text matches and recently used tabs.
6. `Enter` switches to the selected tab; `Esc` closes the UI.

## Initial searchable fields

- Tab title
- URL
- Domain
- Window
- Tab group
- Pinned state
- Recent activation order

## Differentiator

The differentiator is not simply fuzzy matching. It is a fast ranking model that combines:

```text
exact match
+ prefix/fuzzy match
+ recency
+ active window
+ tab group
+ pinned state
```

Semantic similarity may later help when the user remembers the meaning of a page but not its title. It is not required for normal tab switching.

## MVP boundary

- Chromium desktop first.
- Open tabs only.
- Keyboard-first interaction.
- Fuzzy lexical search.
- Realtime updates for tab create/update/activate/remove.
- No browser history, cookies, local storage, or network interception.
- No requirement to index full page content.
