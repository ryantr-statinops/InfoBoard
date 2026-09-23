# Search behavior

## Query lifecycle

1. Trim leading and trailing whitespace.
2. Apply case folding and Unicode normalization.
3. Split tokens on whitespace and punctuation while preserving useful URL/domain segments.
4. Bound query length and token count before processing.
5. Recompute results for each accepted query revision without a submit step.
6. Tie every result set to the known projection revision.

Multiple whitespace-separated tokens are AND-oriented for primary matches. Partial or lower-confidence matches may remain below complete matches when the result limit permits. An empty or whitespace-only query returns a bounded recent/context list, never an unranked dump of all tabs.

## Match explanation

The ranking model uses named components rather than an opaque learned score:

```text
lexical
+ field weight
+ context bonus
+ recency bonus
+ active-window bonus
+ pinned/group signal
- stale/missing penalty
```

Match precedence is approximately exact title/token, exact domain/token, URL token, title prefix/phrase, other fuzzy match, then context-only match. Lexical quality dominates recency; a recent unrelated tab cannot outrank an exact title match solely because it is recent.

## Deterministic ordering

Ties resolve by stronger match class, title over URL-only, recent activation, active window, pinned state, and stable browser tab/window identity. The same projection, query, and ranking timestamp must produce the same IDs, order, score components, and ranking-model version.

## Activation safety

A result is valid only for the profile and projection revision used to produce it. If the tab disappears or the revision changes before activation, refresh first. The UI must report stale or failed activation rather than silently selecting a different tab.

See the [canonical ranking contract](../plan/refactor/search-and-ranking.md) for the complete model and versioning rules.
