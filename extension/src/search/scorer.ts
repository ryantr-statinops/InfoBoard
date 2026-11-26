import type { SearchTabItem } from './types.js';

/**
 * Calculates a subsequence match score between target text and query string.
 * Employs fzf/VS Code style scoring with bonuses for:
 * - Consecutive character matches (+15 per consecutive match)
 * - Word boundary, delimiter, or CamelCase starts (+40)
 * - Penalty for wide spans between first and last match
 * Returns a score between 450 and 580 if matched, or 0 if query is not a subsequence.
 */
export function matchSubsequence(target: string, query: string): number {
  if (query.length === 0 || target.length === 0 || query.length > target.length) {
    return 0;
  }

  const tLower = target.toLowerCase();
  const qLower = query.toLowerCase();

  let tIdx = 0;
  let qIdx = 0;
  let consecutive = 0;
  let bonus = 0;
  let firstMatch = -1;
  let lastMatch = -1;

  while (tIdx < target.length && qIdx < query.length) {
    if (tLower[tIdx] === qLower[qIdx]) {
      if (firstMatch === -1) firstMatch = tIdx;
      lastMatch = tIdx;

      // Consecutive match bonus
      bonus += consecutive * 15;
      consecutive++;

      // Word boundary or delimiter bonus: start of word, preceded by space, dash, dot, slash, underscore
      if (tIdx === 0) {
        bonus += 40;
      } else {
        const prevChar = target[tIdx - 1];
        const currChar = target[tIdx];
        if (/[\s._/\-:]/.test(prevChar)) {
          bonus += 40;
        } else if (prevChar >= 'a' && prevChar <= 'z' && currChar >= 'A' && currChar <= 'Z') {
          // CamelCase boundary: e.g. "Docs" in "GoogleDocs"
          bonus += 40;
        }
      }

      qIdx++;
    } else {
      consecutive = 0;
    }
    tIdx++;
  }

  if (qIdx < query.length) {
    return 0; // Not all query characters were matched in order
  }

  // Span distance penalty: compact matches score higher than sprawling matches
  const span = lastMatch - firstMatch + 1;
  const excessSpan = Math.max(0, span - query.length);
  const spanPenalty = excessSpan * 4;

  const rawScore = 500 + bonus - spanPenalty;
  return Math.min(580, Math.max(450, rawScore));
}

export function scoreTab(item: SearchTabItem, rawQuery: string): number {
  const q = rawQuery.trim().toLowerCase();
  
  // Empty query: perfectly ordered by MRU
  if (q.length === 0) {
    return 1000 - Math.min(item.mruRank, 999);
  }

  const domain = item.domain.toLowerCase();
  const title = item.title.toLowerCase();

  let matchScore = 0;

  // 1. Exact Prefix on Domain or Title
  if (domain.startsWith(q)) {
    matchScore = 1000;
  } else if (title.startsWith(q)) {
    matchScore = 950;
  } else {
    // 2. Word-boundary Prefix Match (Title or Domain)
    const escaped = q.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    const boundaryRegex = new RegExp(`(^|[\\s._/\\-:])${escaped}`, 'i');

    if (boundaryRegex.test(domain)) {
      matchScore = 850;
    } else if (boundaryRegex.test(title)) {
      matchScore = 800;
    } else {
      // 3. Acronym Match (e.g., "yt" -> "YouTube", "gh" -> "GitHub")
      const titleWords = title.split(/[\s._/\-:]+/).filter(w => w.length > 0);
      const acronym = titleWords.map(w => w[0]).join('').toLowerCase();
      if (acronym.length >= q.length && acronym.startsWith(q)) {
        matchScore = 650;
      } else {
        // 4. Subsequence / FZF-style ordered character match
        let qIdx = 0;
        let gaps = 0;
        for (let i = 0; i < title.length && qIdx < q.length; i++) {
          if (title[i] === q[qIdx]) {
            qIdx++;
          } else if (qIdx > 0) {
            gaps++;
          }
        }
        if (qIdx === q.length) {
          matchScore = Math.max(100, 450 - gaps * 10);
        }
      }
    }
  }

  if (matchScore <= 0) {
    return -1;
  }

  // 5. MRU Recency Boost: tabs used more recently receive higher score for equal matches
  const mruBoost = 50 / (item.mruRank + 1);
  return matchScore + mruBoost;
}

export function searchTabs(items: SearchTabItem[], query: string): SearchTabItem[] {
  const scored = items
    .map(item => ({ item, score: scoreTab(item, query) }))
    .filter(res => res.score > 0);

  scored.sort((a, b) => b.score - a.score);
  return scored.map(res => res.item);
}
