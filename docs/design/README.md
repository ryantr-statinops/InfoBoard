# Design

This folder explains how the product behaves from the user's perspective. The binding contracts are [`user-experience.md`](../plan/refactor/user-experience.md) and [`search-and-ranking.md`](../plan/refactor/search-and-ranking.md); these documents make the journey, states, and visual responsibilities easier to review without creating a second contract.

## Reading order

1. [Product surface](product-surface.md) — invocation, layout, result rows, and configuration.
2. [Interaction states](interaction-states.md) — opening, searching, failure, recovery, and dismissal.
3. [Search behavior](search-behavior.md) — query handling, ranking explanation, and activation safety.
4. [Mockups](mockups/README.md) — optional visual references only.

The surface must remain keyboard-first, local-only, deterministic, accessible, and explicit about runtime or activation failures.
