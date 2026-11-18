import type { SearchTabItem } from './types.js';

export function scoreTab(item: SearchTabItem, rawQuery: string): number {
  const q = rawQuery.trim().toLowerCase();
  if (q.length === 0) {
    return 1000 - item.mruRank;
  }

  const domain = item.domain.toLowerCase();
  const title = item.title.toLowerCase();

  // 1. Exact Prefix on Domain or Title
  if (domain.startsWith(q)) {
    return 1000;
  }
  if (title.startsWith(q)) {
    return 950;
  }

  // 2. Word-boundary Prefix Match (Title or Domain)
  // Escapes special regex characters in query
  const escaped = q.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const boundaryRegex = new RegExp(`(^|[\\s._/\\-:])${escaped}`, 'i');

  if (boundaryRegex.test(domain)) {
    return 850;
  }
  if (boundaryRegex.test(title)) {
    return 800;
  }

  // 3. Acronym Match (e.g., "yt" -> "YouTube", "gh" -> "GitHub")
  const titleWords = title.split(/[\s._/\-:]+/).filter(w => w.length > 0);
  const acronym = titleWords.map(w => w[0]).join('').toLowerCase();
  if (acronym.length >= q.length && acronym.startsWith(q)) {
    return 650;
  }

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
    return Math.max(100, 450 - gaps * 10);
  }

  return -1;
}
