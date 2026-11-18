import { normalizeTab, type SearchTabItem } from './types.js';
import { searchTabs } from './scorer.js';
import { MRU_STORAGE_KEY } from '../background/mru-manager.js';

export let allTabs: SearchTabItem[] = [];
export let currentResults: SearchTabItem[] = [];
export let selectedIndex = 0;

export async function loadTabsAndReconcileMRU(): Promise<SearchTabItem[]> {
  const [sessionData, rawTabs] = await Promise.all([
    chrome.storage.session.get(MRU_STORAGE_KEY).catch(() => ({})),
    chrome.tabs.query({}).catch(() => [] as chrome.tabs.Tab[])
  ]);

  const mruIds: number[] = Array.isArray(sessionData[MRU_STORAGE_KEY]) ? sessionData[MRU_STORAGE_KEY] : [];
  const mruMap = new Map<number, number>();
  mruIds.forEach((id, idx) => mruMap.set(id, idx));

  if (mruMap.size === 0) {
    const sorted = [...rawTabs].sort((a, b) => (b.lastAccessed ?? 0) - (a.lastAccessed ?? 0));
    sorted.forEach((tab, idx) => {
      if (typeof tab.id === 'number') mruMap.set(tab.id, idx);
    });
  }

  const items: SearchTabItem[] = [];
  for (const tab of rawTabs) {
    const item = normalizeTab(tab, mruMap);
    if (item) items.push(item);
  }

  items.sort((a, b) => a.mruRank - b.mruRank);
  return items;
}

export function renderTabList(doc: Document = document): void {
  const listEl = doc.getElementById('tab-results');
  const emptyEl = doc.getElementById('empty-state');
  if (!listEl) return;

  listEl.innerHTML = '';

  if (currentResults.length === 0) {
    if (emptyEl) emptyEl.hidden = false;
    return;
  }
  if (emptyEl) emptyEl.hidden = true;

  currentResults.forEach((tab, idx) => {
    const li = doc.createElement('li');
    li.className = 'tab-item';
    li.setAttribute('role', 'option');
    li.id = `tab-opt-${tab.id}`;
    li.setAttribute('aria-selected', idx === selectedIndex ? 'true' : 'false');

    // Favicon
    if (tab.favIconUrl && !tab.favIconUrl.startsWith('chrome://')) {
      const img = doc.createElement('img');
      img.className = 'tab-favicon';
      img.src = tab.favIconUrl;
      img.alt = '';
      img.onerror = () => {
        img.replaceWith(createFaviconPlaceholder(doc, tab.title || tab.domain));
      };
      li.appendChild(img);
    } else {
      li.appendChild(createFaviconPlaceholder(doc, tab.title || tab.domain));
    }

    // Tab Info
    const info = doc.createElement('div');
    info.className = 'tab-info';

    const titleEl = doc.createElement('span');
    titleEl.className = 'tab-title';
    titleEl.textContent = tab.title;
    info.appendChild(titleEl);

    const meta = doc.createElement('div');
    meta.className = 'tab-meta';

    if (tab.domain) {
      const domainEl = doc.createElement('span');
      domainEl.className = 'tab-domain';
      domainEl.textContent = tab.domain;
      meta.appendChild(domainEl);
    }

    if (tab.active) {
      const activeBadge = doc.createElement('span');
      activeBadge.className = 'tab-badge tab-badge-active';
      activeBadge.textContent = 'Current';
      meta.appendChild(activeBadge);
    }

    if (tab.pinned) {
      const pinBadge = doc.createElement('span');
      pinBadge.className = 'tab-badge';
      pinBadge.textContent = 'Pinned';
      meta.appendChild(pinBadge);
    }

    info.appendChild(meta);
    li.appendChild(info);
    listEl.appendChild(li);
  });

  // Ensure selected item is visible
  const selectedEl = listEl.children[selectedIndex] as HTMLElement | undefined;
  if (selectedEl) {
    selectedEl.scrollIntoView({ block: 'nearest' });
  }
}

function createFaviconPlaceholder(doc: Document, text: string): HTMLElement {
  const placeholder = doc.createElement('span');
  placeholder.className = 'tab-favicon-placeholder';
  placeholder.textContent = (text.trim()[0] || '?').toUpperCase();
  return placeholder;
}

export async function initializeSearchSurface(doc: Document = document): Promise<void> {
  const input = doc.querySelector<HTMLInputElement>('[data-search-input]');
  if (!input) return;

  allTabs = await loadTabsAndReconcileMRU();
  currentResults = [...allTabs];

  // Pre-select previous tab (index 1) for instant Alt+Tab toggle
  selectedIndex = currentResults.length > 1 ? 1 : 0;
  renderTabList(doc);

  input.addEventListener('input', () => {
    const query = input.value.trim();
    if (query.length === 0) {
      currentResults = [...allTabs];
      selectedIndex = currentResults.length > 1 ? 1 : 0;
    } else {
      currentResults = searchTabs(allTabs, query);
      selectedIndex = 0;
    }
    renderTabList(doc);
  });

  input.focus();
}

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', () => { void initializeSearchSurface(); }, { once: true });
} else {
  void initializeSearchSurface();
}
