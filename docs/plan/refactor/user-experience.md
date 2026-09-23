# User experience contract

## Search surface

The command opens a small extension-owned focused window for the active browser profile. The surface contains a query input, result list, runtime status, and a short privacy/status affordance. It must be usable without a pointer.

The surface does not navigate the current tab to search, does not inject UI into arbitrary pages, and does not alter browser history by opening a result until the user activates it.

## Core journey

1. User presses the configured shortcut.
2. InfoBoard opens or focuses the search surface.
3. Focus is in the query input; the previous session is not silently reused.
4. User types a fragment.
5. Results update on each accepted query revision.
6. The first result is highlighted only when results exist.
7. Arrow keys move selection; `Enter` activates the highlighted tab; `Esc` dismisses.
8. On activation, the search surface closes and the target tab/window becomes active.

## Query behavior

- Leading/trailing whitespace is ignored.
- Multiple whitespace-separated tokens are treated as an AND-oriented query for primary matches; partial matches may remain as lower-ranked candidates.
- Quoted text is treated as one phrase when the UI can represent it without ambiguity.
- An empty query shows a bounded recent/context list, not the entire unranked tab set.
- Search is case-insensitive and Unicode-normalized.
- No query or result is sent to a network service.

## Result row

Each row contains:

- title as the primary label;
- domain or compact URL as the secondary label;
- window and group context when they disambiguate results;
- pinned indicator when relevant;
- a clear selected/focused state;
- an optional keyboard hint only when the mapping is stable.

Rows MUST not expose full URL query strings by default when they contain likely secrets. The underlying search projection may use the URL, but display redaction is a UI privacy rule.

## UI states

| State | User-visible behavior | Allowed action |
| --- | --- | --- |
| Opening | Focus indicator and no stale result flash. | Type or dismiss. |
| Empty query | Recent/context results with an explanation. | Type, navigate, activate, dismiss. |
| Results | Ordered rows with visible selection. | Navigate or activate. |
| No results | Explain the searched fields and offer query editing. | Edit or dismiss. |
| Rebuilding | Keep the surface usable; show freshness status and previous valid results only if clearly marked. | Wait, retry, or dismiss. |
| Runtime unavailable | Explain that local search is unavailable and provide retry/repair guidance. | Retry, open diagnostics, or dismiss. |
| Stale selection | Refresh projection before activation; never silently activate an unrelated tab. | Retry or choose a refreshed result. |
| Activation failure | Report failure and keep the search surface available. | Retry, choose another result, or dismiss. |
| Private context unavailable | State that the current browser context is outside the configured permission boundary. | Switch context or dismiss. |

## Accessibility

- All actions MUST be reachable by keyboard.
- Focus order and selected-result state MUST be exposed to assistive technology.
- Color MUST not be the only indicator of selection, pinning, freshness, or failure.
- Reduced-motion preference MUST disable nonessential animation.
- Text scaling and narrow windows MUST preserve the query field and selected result.
- Screen-reader labels MUST describe title, domain, context, selected state, and action outcome.

## Configuration experience

The settings view exposes shortcut instructions, result count, display density, theme, context-label visibility, runtime status, privacy boundary, reset, and uninstall guidance. It does not offer switches for excluded data sources because those sources are outside the contract.

## Failure and trust rules

A failure message MUST state what failed, whether browser state changed, and what the user can do next. The UI must never imply that a tab was activated when the browser API or runtime did not confirm activation.
