import type { SearchTabItem } from './types.js';

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
