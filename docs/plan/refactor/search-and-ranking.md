# Search and ranking semantics

## Design goals

The ranking system must be fast, deterministic, explainable, and useful when the user remembers only part of a title, URL, domain, window, or group. It must not require embeddings, network access, or page-content indexing.

## Normalization

For indexed fields and queries:

1. Apply Unicode normalization and case folding.
2. Preserve a display form separately from the searchable form.
3. Split on whitespace and punctuation while retaining URL/domain segments.
4. Remove empty tokens; do not remove meaningful domain or project terms as generic stop words.
5. Bound field length and token count before indexing.
6. Store field provenance so a result can explain whether it matched title, domain, URL, window, or group.

## Match classes

For each query token, calculate the best field match:

1. exact full-field match;
2. exact token match;
3. token prefix match;
4. contiguous phrase match;
5. fuzzy edit-distance or character-subsequence match;
6. context-only match on window/group/state labels.

A result must satisfy all required query tokens through a primary field or an explicitly lower-confidence fallback. Partial-token results may appear below complete matches when the result limit permits.

## Score model

The implementation MUST expose named components rather than an opaque learned score:

```text
score = lexical
      + field_weight
      + context_bonus
      + recency_bonus
      + active_window_bonus
      + pinned_or_group_bonus
      - stale_or_missing_penalty
```

The exact integer weights are configuration constants versioned with the ranking model. Lexical match dominates context and recency. A recently used unrelated tab must not outrank an exact title match solely because it is recent.

Suggested precedence:

```text
exact title/token > exact domain/token > URL token
> title prefix/phrase > other fuzzy match > context-only match
```

## Context signals

- Active window is a small bonus, never a hard filter.
- Recent activation is a decaying bonus with a bounded time horizon.
- Same tab group is a small bonus when the query already matches the tab.
- Pinned state is a display and stability signal, not an unconditional ranking override.
- Window and group labels are useful fallback fields but must not overwhelm title/domain matches.

## Deterministic ordering

When scores tie, order by:

1. stronger lexical match class;
2. title match before URL-only match;
3. higher recent-activation timestamp;
4. active window;
5. pinned state;
6. stable browser tab/window identity.

The final identity tie-breaker ensures identical input produces identical output without depending on map iteration order.

## Empty and malformed queries

- Empty query returns a bounded recent/context list ordered by recency, active window, pinned state, and stable identity.
- Whitespace-only input is equivalent to empty query.
- Overlong or malformed input is truncated or rejected with a visible but non-sensitive validation state.
- Query tokens never trigger network access, shell execution, SQL interpolation, or page evaluation.

## Result explanation

The UI MAY show compact reasons such as `title match`, `domain match`, `recent`, `current window`, or `same group`. Diagnostics MUST retain component scores only when local debug mode is enabled. User-facing explanations must not expose hidden URLs or private metadata.

## Ranking model lifecycle

Each ranking model has a version. A changed model is benchmarked against fixed fixtures containing exact matches, duplicates, typos, multiple windows, groups, pinned tabs, stale records, empty queries, and adversarially long fields. A model change requires updated acceptance expectations and a decision-log entry when ordering changes materially.
