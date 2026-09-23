# Product surface

## Invocation

The user invokes one configurable browser command. InfoBoard opens or focuses a small extension-owned search surface for the active browser profile. The surface does not navigate the current tab, inject UI into arbitrary pages, or send a query to a network service.

The query input receives focus before text is accepted. A reopened surface does not silently reuse a previous query unless the user explicitly reopens the same search session.

## Layout responsibilities

| Area | Responsibility |
| --- | --- |
| Query field | Accept input, expose validation, preserve visible focus, and announce changes. |
| Result list | Show bounded ordered results and one visible selection. |
| Result row | Show title, domain or compact URL, disambiguating window/group context, and relevant pin state. |
| Status area | Show freshness, runtime availability, rebuilding, and activation outcomes. |
| Privacy affordance | Explain local-only search and the product's excluded data sources. |
| Settings surface | Explain command configuration, result count, density, theme, context labels, reset, and uninstall. |

## Result row rules

The title is the primary label. Domain or compact URL is secondary. Window/group context appears when it distinguishes otherwise similar tabs. Full URL query strings are redacted by default when they may contain secrets. Selection, pinning, freshness, and failure cannot be communicated by color alone.

## Keyboard journey

- Arrow keys move the highlighted result.
- `Enter` activates the highlighted result.
- `Esc` dismisses without changing the selected tab.
- Pointer selection remains available but is not required.
- The first result is highlighted only when results exist.

Activation closes the surface only after the browser confirms the target tab/window outcome. A failed activation leaves the surface available for retry or another selection.

See the [canonical UX contract](../plan/refactor/user-experience.md) for the complete interaction boundary.
