# ADR 0001: Search Engine Enhancements

- **Status**: Implemented (v0.2.0)
- **Date**: 2026-11-26
- **Scope**: `extension/src/search/scorer.ts`, `extension/src/search/surface-entrypoint.ts`

## Context

Tab switching needs fast, predictable local matching over small ephemeral datasets (open tabs only). Users need abbreviation matching (`gdoc`, `ytm`), tolerance for small typos (`githbu`), and state-based filtering (`@audio`, `@pinned`).

## Decision

Keep a pure in-memory, deterministic scorer with no external search dependency:

1. **Context filters**: parse `@audio/@a`, `@pinned/@p`, `@group/@g`, `@win/@w` before lexical scoring; filters can combine with text query.
2. **Multi-tier lexical scoring**: exact/prefix and word-boundary matches rank highest, then acronym, then subsequence/fuzzy ordered-character matching with boundary/consecutive bonuses and span penalty. MRU provides recency boost.
3. **Two-phase typo fallback**: only when phase 1 returns no results and clean query length is >= 4, evaluate Damerau-Levenshtein distance <= 1 against domain/title tokens and return closest matches with a UI notice.

## Constraints

- Pure Manifest V3 extension runtime; no native host, daemon, or remote search service.
- No runtime npm dependencies.
- Bundle/behavior guarded by automated tests (`npm test`) and CI build/package workflow.

## Consequences

- Predictable keyboard-first behavior for common tab-switch patterns.
- Typo fallback cost is isolated to no-match queries, preserving the fast path.
- New search behavior must be covered by `tests/scorer.test.mjs` and related surface tests.
