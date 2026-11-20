# Test Strategy

InfoBoard employs a lightweight, automated testing suite powered by Node.js native test runner (`node:test`). The entire suite runs without external daemons, browser harnesses, or database instances, completing in **under 100 milliseconds**.

---

## Test Execution

```bash
# Build TypeScript and run all automated test suites
npm test
```

---

## Test Suites Overview

| Suite | File | Tests | Responsibility |
| :--- | :--- | :---: | :--- |
| **Scorer Engine** | `tests/scorer.test.mjs` | 10 | Exact prefix, word boundaries, acronyms, subsequence matching, recency boost, empty query MRU ordering, and regex character safety (`[`, `*`, `?`, `\`). |
| **MRU Session Manager** | `tests/mru-manager.test.mjs` | 9 | Ring buffer operations, deduplication on activation, eviction at 100 items, invalid input handling (`NaN`, negatives), removal on tab close, and hydration. |
| **Tab Types & Normalizer** | `tests/types.test.mjs` | 6 | Domain extraction (`extractDomain`), `www.` prefix stripping, invalid URL fallbacks, title fallbacks, internal URL rejection (`chrome-extension://`). |
| **Popup UI & Controller** | `tests/surface.test.mjs` | 4 | DOM tab rendering, selection indicator cycling (`updateSelection`), keyboard event handling, and tab/window activation calls (`activateTab`). |
| **Manifest & Bundle Compliance**| `tests/manifest-and-bundle.test.mjs` | 3 | Manifest V3 compliance, strict permission trimming (`tabs`, `storage`, `tabGroups`), shortcut configuration (`_execute_action`), and bundle size verification (< 50KB). |

---

## Total Test Metrics

- **Total Test Cases**: 32 tests
- **Execution Duration**: ~75 ms
- **Pass Rate**: 100%
- **Coverage Areas**: Core ranking logic, session storage persistence, URL parsing, DOM accessibility, and extension manifest validity.
