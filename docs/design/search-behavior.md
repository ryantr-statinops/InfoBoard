# Search Behavior and Scoring Algorithm

InfoBoard features a specialized, deterministic in-memory search algorithm designed specifically for high-speed tab switching in browser environments.

---

## 1. Query Normalization Lifecycle

Every keystroke in the search field triggers immediate re-scoring (0ms debounce):
1. **Trim & Length Clamp**: Whitespace is trimmed; input longer than 100 characters is clamped.
2. **Case Folding**: Query is lowercased.
3. **Empty Query Behavior**: When the query is empty (`""`), all open tabs are displayed ordered strictly by recency (`mruRank`), with index 1 (the previously active tab) pre-selected for instantaneous two-key switching.

---

## 2. Multi-Tier Scoring Hierarchy

When a search query is entered, each tab is scored deterministically based on match specificity. The scorer assigns base points according to the highest matched tier, supplemented by an MRU recency boost:

```text
Final Score = Base Score + Recency Boost
```

### Base Score Tiers (`extension/src/search/scorer.ts`)

| Tier | Match Condition | Points | Examples |
| :---: | :--- | :---: | :--- |
| **Tier 1** | **Exact Domain Prefix** | **1,000** | Query `ti` matches `tiki.vn`, `tik` matches `tiktok.com` |
| **Tier 1b** | **Exact Title Prefix** | **950** | Query `in` matches "Inbox (3) - Gmail" |
| **Tier 2** | **Domain Word Boundary** | **850** | Query `do` matches `docs.google.com` (boundary at `.`) |
| **Tier 2b** | **Title Word Boundary** | **800** | Query `sh` matches "Google Sheets - Q3 Plan" |
| **Tier 3** | **Title Acronym Match** | **650** | Query `gh` matches "GitHub", `yt` matches "YouTube", `fb` matches "Facebook" |
| **Tier 4** | **Subsequence Match** | **400** | Characters appear in order with intervening characters |

Tabs that do not match any tier are assigned a score of `0` and filtered out of the results.

---

## 3. Dynamic MRU Recency Boost

To ensure recently used tabs surface ahead of identical lexical matches across distant tabs, the scorer applies a decaying recency bonus:

$$\text{MRU Boost} = \frac{50}{\text{mruRank} + 1}$$

- Rank 0 (Currently active tab): $+50.0\text{ pts}$
- Rank 1 (Previously active tab): $+25.0\text{ pts}$
- Rank 2: $+16.7\text{ pts}$
- Rank 3: $+12.5\text{ pts}$
- Rank 4: $+10.0\text{ pts}$
- Rank 9+: $< 5.0\text{ pts}$

This design ensures an exact domain prefix match (1,000 pts) always beats a lower-tier match regardless of recency, while breaking ties between similar matches in favor of active work.

---

## 4. Visual Match Highlighting

When tabs are displayed in the popup list, matched characters in the title and URL are enclosed in `<mark>` tags to provide immediate visual feedback of why a tab matched the query.
