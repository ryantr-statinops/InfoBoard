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

  return -1;
}
