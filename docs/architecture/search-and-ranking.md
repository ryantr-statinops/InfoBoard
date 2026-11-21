# Architecture: Search & Ranking Algorithm

InfoBoard features a specialized, deterministic lexical scoring engine implemented in [extension/src/search/scorer.ts](file:///home/ryantran/Projects/InfoBoard/extension/src/search/scorer.ts). It provides instant, predictable search results with zero fuzzy false-positives.

---

## 1. Multi-Tier Scoring Hierarchy

Every eligible open tab is evaluated against the user query. The algorithm assigns a base score from the highest tier matched:

```text
Score Hierarchy:
  Tier 1: Exact Domain Prefix Match (1000 pts)
    └─ Example: query "ti" matches "tiki.vn" or "tiktok.com"
  Tier 2: Word-Boundary Prefix Match (800 - 850 pts)
    └─ Example: query "th" matches "Instagram - Threads" (score 800)
    └─ Example: query "go" matches "Google Search" at word boundary (score 850)
  Tier 3: Acronym Match (650 pts)
    └─ Example: query "gh" matches "GitHub"
    └─ Example: query "yt" matches "YouTube"
  Tier 4: Substring Containment (400 - 450 pts)
    └─ Example: query "book" matches "facebook.com"
```

---

## 2. Most Recently Used (MRU) Recency Boost

When multiple tabs match a query at the same tier, InfoBoard breaks ties using an in-memory recency boost:

$$\text{Final Score} = \text{Base Tier Score} + \frac{50}{\text{mruRank} + 1}$$

- `mruRank = 0` (most recently active tab): receives $+50.0$ bonus points.
- `mruRank = 1` (previous tab): receives $+25.0$ bonus points.
- `mruRank = 2`: receives $+16.7$ bonus points.

### Strategic Rule: Dominance of Higher Tiers
A higher matching tier **always** dominates a lower tier, regardless of recency:
- An exact prefix match on an older tab (1000 pts) will rank above a substring match on the active tab ($400 + 50 = 450$ pts).

---

## 3. Empty Query & Quick Switch (Alt+Tab Pattern)

When the query is empty (`query === ''`):
1. All open tabs are returned sorted strictly by `mruRank` ascending (most recently used first).
2. The popup automatically highlights **index 1** (the previously active tab).
3. **Usage Pattern**: Press `Alt + W` $\rightarrow$ `Enter` to instantly toggle between your two most recent tabs, matching the OS `Alt + Tab` workflow across browser windows.

---

## 4. Normalization & Safety Guarantees

- **Whitespace & Case Insensitivity**: Queries and tab titles are trimmed and evaluated in lowercase.
- **Regex Safety**: InfoBoard does not rely on raw user input in `RegExp` constructors, preventing catastrophic backtracking and unhandled syntax errors.
- **Domain Extraction**: Hostnames are cleanly extracted via `new URL(tab.url).hostname.replace(/^www\./i, '')` with fallback for invalid or internal schemes.
