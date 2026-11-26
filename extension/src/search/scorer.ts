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

  const rawScore = 480 + bonus - spanPenalty;
  return Math.min(600, Math.max(200, rawScore));
}

/**
 * Computes Damerau-Levenshtein distance between two strings with support for
 * adjacent transpositions, insertions, deletions, and substitutions.
 * Memory complexity: O(min(N, M)) using 3 rolling rows.
 * Early exit: if current row minimum exceeds maxDistance, terminates early.
 */
export function damerauLevenshtein(a: string, b: string, maxDistance: number = 1): number {
  if (a === b) return 0;
  const lenA = a.length;
  const lenB = b.length;
  if (Math.abs(lenA - lenB) > maxDistance) return maxDistance + 1;

  // Ensure 'a' is shorter or equal to 'b' to minimize column array size
  if (lenA > lenB) {
    return damerauLevenshtein(b, a, maxDistance);
  }

  // Row buffers: dMinus2 (i-2), dMinus1 (i-1), dCurrent (i)
  let dMinus2 = new Int32Array(lenA + 1);
  let dMinus1 = new Int32Array(lenA + 1);
  let dCurrent = new Int32Array(lenA + 1);

  for (let j = 0; j <= lenA; j++) {
    dMinus1[j] = j;
  }

  for (let i = 1; i <= lenB; i++) {
    dCurrent[0] = i;
    let minInRow = dCurrent[0];

    for (let j = 1; j <= lenA; j++) {
      const cost = a[j - 1] === b[i - 1] ? 0 : 1;
      
      let val = Math.min(
        dCurrent[j - 1] + 1,     // insertion
        dMinus1[j] + 1,          // deletion
        dMinus1[j - 1] + cost    // substitution
      );

      // Transposition check
      if (
        i > 1 &&
        j > 1 &&
        b[i - 1] === a[j - 2] &&
        b[i - 2] === a[j - 1]
      ) {
        val = Math.min(val, dMinus2[j - 2] + 1);
      }

      dCurrent[j] = val;
      if (val < minInRow) {
        minInRow = val;
      }
    }

    // Early termination if all values in this row exceed maxDistance
    if (minInRow > maxDistance) {
      return maxDistance + 1;
    }

    // Shift row buffers
    const temp = dMinus2;
    dMinus2 = dMinus1;
    dMinus1 = dCurrent;
    dCurrent = temp;
  }

  return dMinus1[lenA];
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
        // 4. Subsequence / FZF-style ordered character match (Domain & Title)
        const subDomainScore = matchSubsequence(item.domain, q);
        const subTitleScore = matchSubsequence(item.title, q);
        matchScore = Math.max(subDomainScore, subTitleScore);
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
