export interface SearchTabItem {
  id: number;
  windowId: number;
  title: string;
  url: string;
  domain: string;
  favIconUrl?: string;
  mruRank: number;
  active: boolean;
  pinned: boolean;
  groupId?: number;
  audible?: boolean;
  muted?: boolean;
}

export function extractDomain(rawUrl: string): string {
  try {
    const parsed = new URL(rawUrl);
    return parsed.hostname.replace(/^www\./i, '');
  } catch {
    return '';
  }
}

export function normalizeTab(tab: chrome.tabs.Tab, mruMap: Map<number, number>): SearchTabItem | null {
  if (typeof tab.id !== 'number' || !tab.url) return null;
  if (tab.url.startsWith('chrome-extension://')) return null;

  const domain = extractDomain(tab.url);
  const title = (tab.title && tab.title.trim().length > 0) ? tab.title.trim() : domain || tab.url;
  const mruRank = mruMap.has(tab.id) ? (mruMap.get(tab.id) as number) : 9999;

  return {
    id: tab.id,
    windowId: tab.windowId,
    title,
    url: tab.url,
    domain,
    favIconUrl: tab.favIconUrl,
    mruRank,
    active: Boolean(tab.active),
    pinned: Boolean(tab.pinned),
    groupId: tab.groupId && tab.groupId > 0 ? tab.groupId : undefined,
    audible: Boolean(tab.audible),
    muted: Boolean(tab.mutedInfo?.muted)
  };
}
